package sip

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/config"
	"git.skcks.cn/Shikong/go-gb28181/internal/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/rs/zerolog"
)

// SIPServer SIP 服务管理器
type SIPServer struct {
	config        *config.Config
	deviceService *service.DeviceService

	ua         *sipgo.UserAgent
	client     *sipgo.Client
	server     *sipgo.Server
	externalIP string // 对外 IP (用于 SIP 消息)

	ctx    context.Context
	cancel context.CancelFunc
}

// NewSIPServer 创建 SIP 服务器
func NewSIPServer(cfg *config.Config, deviceService *service.DeviceService) *SIPServer {
	return &SIPServer{
		config:        cfg,
		deviceService: deviceService,
	}
}

// Start 启动 SIP 服务
func (s *SIPServer) Start() error {
	// 解决 UDP 包大于 MTU 报错
	sip.UDPMTUSize = math.MaxInt

	// 初始化日志
	output := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stdout
		w.TimeFormat = time.RFC3339
	})
	logger := zerolog.New(output).With().Timestamp().Logger()

	if s.config.Debug {
		sip.SIPDebug = s.config.Debug
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	log.SetLogger(&logger)

	// 创建上下文
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// 创建 UserAgent
	addr := fmt.Sprintf("%s:%d", s.config.SIP.ListenIP, s.config.SIP.ListenPort)
	ua, err := sipgo.NewUA(
		sipgo.WithUserAgent(s.config.SIP.DeviceID),
		sipgo.WithUserAgentHostname(addr),
	)
	if err != nil {
		return fmt.Errorf("创建 UserAgent 失败: %w", err)
	}
	s.ua = ua

	// 获取对外 IP (用于 Via 头)
	externalIP := s.getExternalIP()
	s.externalIP = externalIP
	log.Info().Str("external_ip", externalIP).Msg("使用对外 IP")

	// 创建客户端 (发送/回复 SIP 指令) - 使用对外 IP
	client, err := sipgo.NewClient(ua,
		sipgo.WithClientHostname(externalIP),
		sipgo.WithClientPort(s.config.SIP.ListenPort),
	)
	if err != nil {
		return fmt.Errorf("创建 SIP 客户端失败: %w", err)
	}
	s.client = client

	// 创建服务端 (接收 SIP 指令)
	server, err := sipgo.NewServer(ua, sipgo.WithServerLogger(logger))
	if err != nil {
		return fmt.Errorf("创建 SIP 服务端失败: %w", err)
	}
	s.server = server

	// 注册消息处理器
	s.setupHandlers()

	// 启动 SIP 服务
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Error().Msgf("SIP 服务 panic: %v", err)
			}
		}()

		if err := server.ListenAndServe(s.ctx, "udp", addr); err != nil {
			log.Error().Err(err).Msg("SIP 服务停止")
		}
	}()

	log.Info().Str("addr", addr).Msg("SIP 服务启动成功")
	return nil
}

// Stop 停止 SIP 服务
func (s *SIPServer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	log.Info().Msg("SIP 服务已停止")
}

// setupHandlers 设置消息处理器
func (s *SIPServer) setupHandlers() {
	// 注册 MESSAGE 处理
	s.server.OnMessage(func(req *sip.Request, tx sip.ServerTransaction) {
		s.handleMessage(req, tx)
	})

	// 注册 REGISTER 处理
	s.server.OnRegister(func(req *sip.Request, tx sip.ServerTransaction) {
		s.handleRegister(req, tx)
	})
}

// handleMessage 处理 SIP MESSAGE
func (s *SIPServer) handleMessage(req *sip.Request, tx sip.ServerTransaction) {
	defer tx.Terminate()

	// 解析消息
	contentType := req.GetHeader("Content-Type")
	if contentType == nil || contentType.Value() != "Application/MANSCDP+xml" {
		log.Warn().Msg("收到非 MANSCDP 消息")
		return
	}

	body := req.Body()
	log.Debug().Str("body", string(body)).Msg("收到 SIP MESSAGE")

	// 响应 200 OK
	resp := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(resp); err != nil {
		log.Error().Err(err).Msg("响应 MESSAGE 失败")
	}
}

// handleRegister 处理 SIP REGISTER
func (s *SIPServer) handleRegister(req *sip.Request, tx sip.ServerTransaction) {
	defer tx.Terminate()

	// 获取设备 ID
	from := req.From()
	if from == nil {
		log.Warn().Msg("REGISTER 请求缺少 From 头")
		resp := sip.NewResponseFromRequest(req, 400, "Bad Request", nil)
		tx.Respond(resp)
		return
	}

	// 解析设备 ID
	fromURI := from.Address
	deviceID := fromURI.User // User 是属性
	if deviceID == "" {
		log.Warn().Msg("无法解析设备 ID")
		resp := sip.NewResponseFromRequest(req, 400, "Bad Request", nil)
		tx.Respond(resp)
		return
	}

	// 获取设备 IP 和端口
	via := req.Via()
	var deviceIP string
	var devicePort int
	if via != nil {
		deviceIP = via.Host   // Host 是属性
		devicePort = via.Port // Port 是属性
	}

	// 获取 Contact 头的地址（更准确的设备地址）
	contact := req.Contact()
	if contact != nil {
		if contact.Address.Host != "" {
			deviceIP = contact.Address.Host
		}
		if contact.Address.Port > 0 {
			devicePort = contact.Address.Port
		}
	}

	// 检查是否需要认证 (401)
	authorization := req.GetHeader("Authorization")
	if authorization == nil {
		// 首次注册，发送 401 要求认证
		log.Debug().Str("device_id", deviceID).Msg("设备首次注册，发送 401")
		resp := sip.NewResponseFromRequest(req, 401, "Unauthorized", nil)
		resp.AppendHeader(sip.NewHeader("WWW-Authenticate", fmt.Sprintf("Digest realm=\"%s\", nonce=\"%d\"", deviceID, time.Now().Unix())))
		tx.Respond(resp)
		return
	}

	// TODO: 验证 Digest 认证
	// 简化处理：直接接受注册

	// 保存设备信息到数据库
	if s.deviceService != nil {
		if err := s.deviceService.OnDeviceRegister(deviceID, deviceIP, devicePort); err != nil {
			log.Error().Err(err).Str("device_id", deviceID).Msg("保存设备注册信息失败")
		}
	}

	log.Info().Str("device_id", deviceID).Str("ip", deviceIP).Int("port", devicePort).Msg("设备注册成功")

	// 响应 200 OK
	resp := sip.NewResponseFromRequest(req, 200, "OK", nil)
	resp.AppendHeader(sip.NewHeader("Contact", fmt.Sprintf("<sip:%s@%s:%d>", deviceID, s.externalIP, s.config.SIP.ListenPort)))
	resp.AppendHeader(sip.NewHeader("Expires", "3600"))
	if err := tx.Respond(resp); err != nil {
		log.Error().Err(err).Msg("响应 REGISTER 失败")
	}
}

// GetClient 获取 SIP 客户端
func (s *SIPServer) GetClient() *sipgo.Client {
	return s.client
}

// GetServer 获取 SIP 服务端
func (s *SIPServer) GetServer() *sipgo.Server {
	return s.server
}

// getExternalIP 获取对外 IP
// 优先使用配置的 ExternalIP，否则自动检测本机 IP
func (s *SIPServer) getExternalIP() string {
	// 如果配置了 ExternalIP，直接使用
	if s.config.SIP.ExternalIP != "" {
		return s.config.SIP.ExternalIP
	}

	// 自动检测本机 IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Warn().Err(err).Msg("无法检测本机 IP，使用 ListenIP")
		return s.config.SIP.ListenIP
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
