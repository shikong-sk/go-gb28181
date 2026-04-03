package sip

import (
	"context"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/config"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/icholy/digest"
)

// SIPServer SIP 服务管理器
type SIPServer struct {
	config        *config.Config
	deviceService *service.DeviceService
	alarmService  *service.AlarmService
	recordService *service.RecordService

	// 新增服务
	subscriptionService *service.SubscriptionService
	positionService     *service.PositionService
	playService         *service.PlayService

	ua         *sipgo.UserAgent
	client     *sipgo.Client
	server     *sipgo.Server
	externalIP string // 对外 IP (用于 SIP 消息)

	ctx    context.Context
	cancel context.CancelFunc
}

// NewSIPServer 创建 SIP 服务器
func NewSIPServer(cfg *config.Config, deviceService *service.DeviceService, alarmService *service.AlarmService) *SIPServer {
	return &SIPServer{
		config:        cfg,
		deviceService: deviceService,
		alarmService:  alarmService,
	}
}

// SetRecordService 设置录像查询服务
func (s *SIPServer) SetRecordService(recordService *service.RecordService) {
	s.recordService = recordService
}

// SetSubscriptionService 设置订阅服务
func (s *SIPServer) SetSubscriptionService(subscriptionService *service.SubscriptionService) {
	s.subscriptionService = subscriptionService
}

// SetPositionService 设置定位服务
func (s *SIPServer) SetPositionService(positionService *service.PositionService) {
	s.positionService = positionService
}

// SetPlayService 设置播放服务
func (s *SIPServer) SetPlayService(playService *service.PlayService) {
	s.playService = playService
}

// Start 启动 SIP 服务
func (s *SIPServer) Start() error {
	// 解决 UDP 包大于 MTU 报错
	sip.UDPMTUSize = math.MaxInt

	// 获取全局 logger (已在 pkg/log 中配置 ConsoleWriter)
	logger := *log.GetLogger()

	if s.config.Debug {
		sip.SIPDebug = s.config.Debug
		// 设置自定义 SIP 日志跟踪器，使用 ConsoleWriter 格式输出
		sip.SIPDebugTracer(log.NewSIPTracer())
	}

	// 创建上下文
	s.ctx, s.cancel = context.WithCancel(context.Background())
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
	// 传入 logger 以输出可视化格式的日志
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

	// 转换为 UTF-8 用于日志输出
	bodyUTF8, _ := utils.DetectAndConvertToUTF8(body)
	log.Debug().Str("body", string(bodyUTF8)).Msg("收到 SIP MESSAGE")

	// 解析 XML 消息头（通用解析，支持 Query/Response/Notify）
	header := new(manscdp.MessageHeader)
	if err := utils.XMLUnmarshal(body, header); err != nil {
		log.Error().Err(err).Msg("解析 MANSCDP 消息头失败")
		resp := sip.NewResponseFromRequest(req, 400, "Bad Request", nil)
		tx.Respond(resp)
		return
	}

	log.Info().Str("cmd_type", header.CmdType).Str("xml_name", header.XMLName.Local).Msg("收到 MANSCDP 消息")

	// 根据消息类型处理
	switch header.CmdType {
	case "Catalog":
		s.handleCatalogMessage(req, bodyUTF8)
	case "Keepalive":
		s.handleKeepaliveMessage(req, bodyUTF8)
	case "Alarm":
		s.handleAlarmMessage(req, bodyUTF8)
	case "RecordInfo":
		s.handleRecordInfoMessage(req, bodyUTF8)
	case "DeviceStatus":
		s.handleDeviceStatusMessage(req, bodyUTF8)
	case "MobilePosition":
		s.handleMobilePositionMessage(req, bodyUTF8)
	case "MediaStatus":
		s.handleMediaStatusMessage(req, bodyUTF8)
	default:
		log.Warn().Str("cmd_type", header.CmdType).Str("xml_name", header.XMLName.Local).Msg("未处理的 MANSCDP 消息类型")
	}

	// 响应 200 OK
	// 响应 200 OK
	resp := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(resp); err != nil {
		log.Error().Err(err).Msg("响应 MESSAGE 失败")
	}
}

// handleCatalogMessage 处理 Catalog 消息
func (s *SIPServer) handleCatalogMessage(req *sip.Request, body []byte) {
	// 尝试解析为 Response
	resp := new(manscdp.CatalogResp)
	if err := utils.XMLUnmarshal(body, resp); err == nil && resp.SumNum != "" {
		// 这是 Catalog 响应
		log.Info().Str("device_id", resp.DeviceID).Str("sum_num", resp.SumNum).Msg("收到目录响应")
		s.processCatalogResponse(resp)
		return
	}

	// 尝试解析为 Query
	query := new(manscdp.CatalogReq)
	if err := utils.XMLUnmarshal(body, query); err == nil {
		log.Info().Str("device_id", query.DeviceID).Msg("收到目录查询请求")
		// TODO: 实现目录查询响应（作为设备端）
	}
}

// processCatalogResponse 处理目录响应
func (s *SIPServer) processCatalogResponse(resp *manscdp.CatalogResp) {
	if s.deviceService == nil {
		log.Warn().Msg("设备服务未初始化，无法保存目录")
		return
	}

	// 解析设备列表
	if resp.DeviceList == nil || len(resp.DeviceList.Item) == 0 {
		log.Info().Str("device_id", resp.DeviceID).Msg("目录响应无设备列表")
		return
	}

	log.Info().Str("device_id", resp.DeviceID).Int("count", len(resp.DeviceList.Item)).Msg("处理目录响应")

	// 转换为 Channel 模型列表
	channels := make([]model.Channel, 0, len(resp.DeviceList.Item))
	for _, item := range resp.DeviceList.Item {
		port, _ := strconv.Atoi(item.Port)
		channel := model.Channel{
			ChannelID:    item.DeviceID,
			DeviceID:     resp.DeviceID,
			Name:         item.Name,
			Manufacturer: item.Manufacturer,
			Model:        item.Model,
			Owner:        item.Owner,
			CivilCode:    item.CivilCode,
			Address:      item.Address,
			Status:       item.Status,
			IP:           item.IPAddress,
			Port:         port,
		}
		channels = append(channels, channel)
	}

	// 调用服务层保存
	if err := s.deviceService.OnCatalogReceived(resp.DeviceID, channels); err != nil {
		log.Error().Err(err).Str("device_id", resp.DeviceID).Msg("保存目录失败")
	} else {
		log.Info().Str("device_id", resp.DeviceID).Int("count", len(channels)).Msg("目录保存成功")
	}
}

// handleKeepaliveMessage 处理 Keepalive 消息
func (s *SIPServer) handleKeepaliveMessage(req *sip.Request, body []byte) {
	keepalive := new(manscdp.KeepAliveReq)
	if err := utils.XMLUnmarshal(body, keepalive); err != nil {
		log.Error().Err(err).Msg("解析 Keepalive 消息失败")
		return
	}

	log.Debug().Str("device_id", keepalive.DeviceID).Msg("收到心跳消息")

	// 更新设备心跳时间
	if s.deviceService != nil {
		if err := s.deviceService.OnDeviceKeepalive(keepalive.DeviceID); err != nil {
			log.Error().Err(err).Str("device_id", keepalive.DeviceID).Msg("更新心跳时间失败")
		}
	}
}

// handleAlarmMessage 处理 Alarm 消息
func (s *SIPServer) handleAlarmMessage(req *sip.Request, body []byte) {
	alarm := new(manscdp.AlarmNotify)
	if err := utils.XMLUnmarshal(body, alarm); err != nil {
		log.Error().Err(err).Msg("解析 Alarm 消息失败")
		return
	}

	log.Info().
		Str("device_id", alarm.DeviceID).
		Str("alarm_priority", alarm.AlarmPriority).
		Str("alarm_method", alarm.AlarmMethod).
		Str("alarm_time", alarm.AlarmTime).
		Str("alarm_description", alarm.AlarmDescription).
		Msg("收到报警通知")

	// 保存报警信息到数据库
	if s.alarmService != nil {
		alarmRecord := &model.Alarm{
			DeviceID:         alarm.DeviceID,
			AlarmPriority:    alarm.AlarmPriority,
			AlarmMethod:      alarm.AlarmMethod,
			AlarmTime:        alarm.AlarmTime,
			AlarmDescription: alarm.AlarmDescription,
			AlarmInfo:        alarm.AlarmInfo,
		}
		if err := s.alarmService.SaveAlarm(alarmRecord); err != nil {
			log.Error().Err(err).Str("device_id", alarm.DeviceID).Msg("保存报警失败")
		}
	}
}

// handleRecordInfoMessage 处理 RecordInfo 消息
func (s *SIPServer) handleRecordInfoMessage(req *sip.Request, body []byte) {
	resp := new(manscdp.RecordInfoResp)
	if err := utils.XMLUnmarshal(body, resp); err != nil {
		log.Error().Err(err).Msg("解析 RecordInfo 消息失败")
		return
	}

	log.Info().
		Str("device_id", resp.DeviceID).
		Str("sn", resp.SN).
		Str("sum_num", resp.SumNum).
		Msg("收到历史录像查询响应")

	// 转发给 RecordService 处理
	if s.recordService != nil {
		s.recordService.HandleRecordInfoResponse(resp)
	}
}

// handleDeviceStatusMessage 处理 DeviceStatus 消息
func (s *SIPServer) handleDeviceStatusMessage(req *sip.Request, body []byte) {
	// 解析响应
	var resp manscdp.DeviceStatusResp
	err := utils.XMLUnmarshal([]byte(body), &resp)
	if err != nil {
		log.Error().Err(err).Msg("解析 DeviceStatus 响应失败")
		return
	}

	log.Info().
		Str("device_id", resp.DeviceID).
		Str("sn", resp.SN).
		Str("result", resp.Result).
		Str("online", resp.Online).
		Str("status", resp.Status).
		Msg("收到设备状态响应")

	// 通知订阅服务
	if s.subscriptionService != nil {
		s.subscriptionService.NotifyResponse(resp.DeviceID, resp.SN, &resp)
	}

	// 更新设备状态到数据库（可选）
	if s.deviceService != nil {
		// TODO: 实现设备状态更新逻辑
		log.Debug().Str("device_id", resp.DeviceID).Msg("设备状态响应已处理")
	}
}

// handleMobilePositionMessage 处理 MobilePosition 消息
func (s *SIPServer) handleMobilePositionMessage(req *sip.Request, body []byte) {
	// 解析 Notify
	var notify manscdp.MobilePositionNotify
	err := utils.XMLUnmarshal([]byte(body), &notify)
	if err != nil {
		log.Error().Err(err).Msg("解析 MobilePosition 上报失败")
		return
	}

	log.Info().
		Str("device_id", notify.DeviceID).
		Str("sn", notify.SN).
		Float64("longitude", notify.Longitude).
		Float64("latitude", notify.Latitude).
		Float64("speed", notify.Speed).
		Int("direction", notify.Direction).
		Float64("altitude", notify.Altitude).
		Str("gps_time", notify.GPSTime).
		Msg("收到移动设备定位上报")

	// 调用 PositionService 处理
	if s.positionService != nil {
		if err := s.positionService.OnMobilePositionReceived(&notify); err != nil {
			log.Error().Err(err).Str("device_id", notify.DeviceID).Msg("处理 MobilePosition 失败")
		}
	} else {
		log.Warn().Str("device_id", notify.DeviceID).Msg("定位服务未初始化，无法处理定位上报")
	}
}

// handleMediaStatusMessage 处理 MediaStatus 消息
func (s *SIPServer) handleMediaStatusMessage(req *sip.Request, body []byte) {
	// 解析 Notify
	var notify manscdp.MediaStatusNotify
	err := utils.XMLUnmarshal([]byte(body), &notify)
	if err != nil {
		log.Error().Err(err).Msg("解析 MediaStatus 通知失败")
		return
	}

	log.Info().
		Str("device_id", notify.DeviceID).
		Str("sn", notify.SN).
		Int("notify_type", int(notify.NotifyType)).
		Str("stream_id", notify.StreamId).
		Msg("收到媒体状态通知")

	// 处理录像结束通知 (NotifyType=121)
	if notify.NotifyType == manscdp.MediaStatusNotifyTypeRecordEnd {
		log.Info().Str("stream_id", notify.StreamId).Msg("录像结束通知")

		// 清理播放会话
		if s.playService != nil {
			err := s.playService.OnMediaStatusReceived(notify.StreamId)
			if err != nil {
				log.Error().Err(err).Str("stream_id", notify.StreamId).Msg("处理 MediaStatus 失败")
			} else {
				log.Debug().Str("stream_id", notify.StreamId).Msg("录像结束通知已处理")
			}
		} else {
			log.Warn().Str("stream_id", notify.StreamId).Msg("播放服务未初始化，无法处理录像结束通知")
		}
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

	// 验证 Digest 认证
	chal, err := digest.ParseChallenge(authorization.Value())
	if err != nil {
		log.Warn().Err(err).Str("device_id", deviceID).Msg("解析 Authorization 失败")
		resp := sip.NewResponseFromRequest(req, 400, "Bad Request", nil)
		tx.Respond(resp)
		return
	}

	// 获取设备密码
	password := s.config.SIP.Password // 默认使用平台密码
	if s.deviceService != nil {
		device, err := s.deviceService.GetDevice(deviceID)
		if err == nil && device.Password != "" {
			password = device.Password // 使用设备独立密码
		}
	}

	// 从 Authorization 头提取 URI (客户端使用的 URI)
	authResp := authorization.Value()
	authURI := extractDigestURI(authResp)

	// 计算期望的 Digest 响应
	cred, err := digest.Digest(chal, digest.Options{
		Method:   "REGISTER",
		Username: deviceID,
		URI:      authURI,
		Password: password,
	})
	if err != nil {
		log.Warn().Err(err).Str("device_id", deviceID).Msg("计算 Digest 失败")
		resp := sip.NewResponseFromRequest(req, 500, "Internal Server Error", nil)
		tx.Respond(resp)
		return
	}

	// 验证响应值
	if !strings.Contains(authResp, "response=\""+cred.Response+"\"") {
		log.Warn().Str("device_id", deviceID).Msg("Digest 认证失败: 密码错误")
		resp := sip.NewResponseFromRequest(req, 403, "Forbidden", nil)
		tx.Respond(resp)
		return
	}

	log.Info().Str("device_id", deviceID).Msg("Digest 认证成功")

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

// extractDigestURI 从 Authorization 头提取 URI 值
func extractDigestURI(authHeader string) string {
	// 格式: Digest username="xxx", uri="sip:xxx", ...
	parts := strings.Split(authHeader, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "uri=") {
			// 提取 uri="xxx" 中的 xxx
			uri := strings.TrimPrefix(part, "uri=")
			uri = strings.Trim(uri, "\"")
			return uri
		}
	}
	return ""
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
