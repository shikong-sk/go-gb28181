package sip

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/config"
	"git.skcks.cn/Shikong/go-gb28181/internal/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// RegisterHandler 注册处理器
type RegisterHandler struct {
	config        *config.Config
	deviceService *service.DeviceService
	client        *sipgo.Client
	stopChan      chan struct{}
}

// NewRegisterHandler 创建注册处理器
func NewRegisterHandler(cfg *config.Config, deviceService *service.DeviceService, client *sipgo.Client) *RegisterHandler {
	return &RegisterHandler{
		config:        cfg,
		deviceService: deviceService,
		client:        client,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动注册
func (h *RegisterHandler) Start() {
	go h.registerLoop()
	log.Info().Msg("设备注册处理器启动")
}

// Stop 停止注册
func (h *RegisterHandler) Stop() {
	close(h.stopChan)
	log.Info().Msg("设备注册处理器停止")
}

// registerLoop 注册循环
func (h *RegisterHandler) registerLoop() {
	// 首次立即注册
	h.doRegister()

	// 定时注册
	ticker := time.NewTicker(time.Duration(h.config.SIP.RegisterCycle) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopChan:
			return
		case <-ticker.C:
			h.doRegister()
		}
	}
}

// doRegister 执行注册
func (h *RegisterHandler) doRegister() {
	target := sip.Uri{
		User: h.config.SIP.ServerID,
		Host: h.config.SIP.ServerIP,
		Port: h.config.SIP.ServerPort,
	}

	req := sip.NewRequest(sip.REGISTER, target)
	req.SetTransport("UDP")

	// 添加 Contact 头
	contact := sip.NewHeader("Contact", fmt.Sprintf("<sip:%s@%s:%d>",
		h.config.SIP.DeviceID, h.config.SIP.ListenIP, h.config.SIP.ListenPort))
	req.AppendHeader(contact)

	// 添加 Expires 头
	expires := sip.NewHeader("Expires", fmt.Sprintf("%d", h.config.SIP.RegisterCycle))
	req.AppendHeader(expires)

	// 发送请求
	tx, err := h.client.TransactionRequest(context.Background(), req)
	if err != nil {
		log.Error().Err(err).Msg("发送注册请求失败")
		return
	}
	defer tx.Terminate()

	// 等待响应
	res, err := getResponse(tx)
	if err != nil {
		log.Error().Err(err).Msg("注册请求执行失败")
		return
	}

	// 处理响应
	switch res.StatusCode {
	case 200:
		log.Info().Msg("设备注册成功")
		// 更新数据库
		_ = h.deviceService.OnDeviceRegister(h.config.SIP.DeviceID, h.config.SIP.ListenIP, h.config.SIP.ListenPort)

	case 401:
		// 需要 Digest 认证
		h.handleAuthChallenge(req, res)

	default:
		log.Warn().Int("status", int(res.StatusCode)).Msg("注册失败")
	}
}

// handleAuthChallenge 处理认证挑战
func (h *RegisterHandler) handleAuthChallenge(originalReq *sip.Request, resp *sip.Response) {
	wwwAuth := resp.GetHeader("WWW-Authenticate")
	if wwwAuth == nil {
		log.Error().Msg("401 响应缺少 WWW-Authenticate 头")
		return
	}

	// 构建带认证的请求
	newReq := originalReq.Clone()
	newReq.RemoveHeader("Via")
	newReq.RemoveHeader("Call-ID")
	newReq.RemoveHeader("CSeq")

	// 添加 Authorization 头 (简化实现，完整实现需要解析 WWW-Authenticate 并计算 digest)
	// TODO: 实现完整的 Digest 认证
	auth := sip.NewHeader("Authorization", fmt.Sprintf(`Digest username="%s", realm="...", nonce="...", uri="sip:%s@%s:%d", response="..."`,
		h.config.SIP.DeviceID, h.config.SIP.DeviceID, h.config.SIP.ServerIP, h.config.SIP.ServerPort))
	newReq.AppendHeader(auth)

	// 重新发送
	tx, err := h.client.TransactionRequest(context.Background(), newReq)
	if err != nil {
		log.Error().Err(err).Msg("发送认证注册请求失败")
		return
	}
	defer tx.Terminate()

	res, err := getResponse(tx)
	if err != nil {
		log.Error().Err(err).Msg("认证注册请求执行失败")
		return
	}

	if res.StatusCode == 200 {
		log.Info().Msg("设备认证注册成功")
		_ = h.deviceService.OnDeviceRegister(h.config.SIP.DeviceID, h.config.SIP.ListenIP, h.config.SIP.ListenPort)
	} else {
		log.Warn().Int("status", int(res.StatusCode)).Msg("认证注册失败")
	}
}

// KeepaliveHandler 心跳处理器
type KeepaliveHandler struct {
	config        *config.Config
	deviceService *service.DeviceService
	client        *sipgo.Client
	stopChan      chan struct{}
}

// NewKeepaliveHandler 创建心跳处理器
func NewKeepaliveHandler(cfg *config.Config, deviceService *service.DeviceService, client *sipgo.Client) *KeepaliveHandler {
	return &KeepaliveHandler{
		config:        cfg,
		deviceService: deviceService,
		client:        client,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动心跳
func (h *KeepaliveHandler) Start() {
	go h.keepaliveLoop()
	log.Info().Msg("设备心跳处理器启动")
}

// Stop 停止心跳
func (h *KeepaliveHandler) Stop() {
	close(h.stopChan)
	log.Info().Msg("设备心跳处理器停止")
}

// keepaliveLoop 心跳循环
func (h *KeepaliveHandler) keepaliveLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopChan:
			return
		case <-ticker.C:
			h.sendKeepalive()
		}
	}
}

// sendKeepalive 发送心跳
func (h *KeepaliveHandler) sendKeepalive() {
	// 构建心跳消息
	sn := fmt.Sprintf("%06d", rand.Intn(1000000))
	keepaliveReq := manscdp.NewKeepAliveReqWithOK(sn, h.config.SIP.DeviceID)

	body, err := utils.XMLMarshal(keepaliveReq, "gbk")
	if err != nil {
		log.Error().Err(err).Msg("序列化心跳消息失败")
		return
	}

	// 发送 MESSAGE
	target := sip.Uri{
		User: h.config.SIP.ServerID,
		Host: h.config.SIP.ServerIP,
		Port: h.config.SIP.ServerPort,
	}

	req := sip.NewRequest(sip.MESSAGE, target)
	req.SetTransport("UDP")
	req.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))
	req.SetBody(body)

	if err := sipgo.ClientRequestBuild(h.client, req); err != nil {
		log.Error().Err(err).Msg("构建心跳请求失败")
		return
	}

	if err := h.client.WriteRequest(req); err != nil {
		log.Error().Err(err).Msg("发送心跳失败")
		return
	}

	log.Debug().Msg("心跳发送成功")

	// 更新心跳时间
	_ = h.deviceService.OnDeviceKeepalive(h.config.SIP.DeviceID)
}

// CatalogHandler 目录处理器
type CatalogHandler struct {
	config        *config.Config
	deviceService *service.DeviceService
	client        *sipgo.Client
}

// NewCatalogHandler 创建目录处理器
func NewCatalogHandler(cfg *config.Config, deviceService *service.DeviceService, client *sipgo.Client) *CatalogHandler {
	return &CatalogHandler{
		config:        cfg,
		deviceService: deviceService,
		client:        client,
	}
}

// HandleCatalogQuery 处理目录查询请求
func (h *CatalogHandler) HandleCatalogQuery(req *sip.Request, tx sip.ServerTransaction) {
	// 解析请求
	query := new(manscdp.CatalogReq)
	if err := utils.XMLUnmarshal(req.Body(), query); err != nil {
		log.Error().Err(err).Msg("解析目录查询请求失败")
		return
	}

	log.Info().Str("device_id", query.DeviceID).Str("sn", query.SN).Msg("收到目录查询请求")
	tx.Done()

	// 构建响应
	device := manscdp.NewCateLogDevice(func(d *manscdp.CateLogDevice) {
		d.DeviceID = h.config.SIP.DeviceID
		d.Name = "设备名称"
		d.Manufacturer = "设备厂商"
		d.ErrCode = "0"
		d.Port = fmt.Sprintf("%d", h.config.SIP.ListenPort)
	})

	list := manscdp.NewCateLogDeviceList([]manscdp.CateLogDevice{*device})
	resp := manscdp.NewCatalogResp(1, list, query.SN, h.config.SIP.DeviceID)

	body, _ := utils.XMLMarshal(resp, "gbk")

	// 发送响应
	target := sip.Uri{
		User: h.config.SIP.ServerID,
		Host: h.config.SIP.ServerIP,
		Port: h.config.SIP.ServerPort,
	}

	nReq := sip.NewRequest(sip.MESSAGE, target)
	nReq.SetTransport("UDP")
	nReq.AppendHeader(sip.NewHeader("To", req.GetHeader("From").Value()))
	nReq.AppendHeader(sip.NewHeader("From", req.GetHeader("To").Value()))
	nReq.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))
	nReq.SetBody(body)

	if err := sipgo.ClientRequestBuild(h.client, nReq); err != nil {
		log.Error().Err(err).Msg("构建目录响应失败")
		return
	}

	if err := h.client.WriteRequest(nReq); err != nil {
		log.Error().Err(err).Msg("发送目录响应失败")
		return
	}

	log.Info().Msg("目录查询响应发送成功")
}

// SyncCatalog 同步设备目录
func (h *CatalogHandler) SyncCatalog(deviceID string) error {
	// 构建目录查询请求
	sn := fmt.Sprintf("%06d", rand.Intn(1000000))
	catalogReq := manscdp.NewCatalogReq(cmdtype.Catalog, sn, deviceID)

	body, err := utils.XMLMarshal(catalogReq, "gbk")
	if err != nil {
		return fmt.Errorf("序列化目录查询请求失败: %w", err)
	}

	// 发送 MESSAGE
	target := sip.Uri{
		User: deviceID,
		Host: h.config.SIP.ServerIP,
		Port: h.config.SIP.ServerPort,
	}

	req := sip.NewRequest(sip.MESSAGE, target)
	req.SetTransport("UDP")
	req.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))
	req.SetBody(body)

	if err := sipgo.ClientRequestBuild(h.client, req); err != nil {
		return fmt.Errorf("构建目录查询请求失败: %w", err)
	}

	if err := h.client.WriteRequest(req); err != nil {
		return fmt.Errorf("发送目录查询请求失败: %w", err)
	}

	log.Info().Str("device_id", deviceID).Msg("目录同步请求发送成功")
	return nil
}

// HandleCatalogNotify 处理目录通知 (收到设备上报的目录)
func (h *CatalogHandler) HandleCatalogNotify(body []byte, deviceID string) error {
	// 解析目录响应
	resp := new(manscdp.CatalogResp)
	if err := utils.XMLUnmarshal(body, resp); err != nil {
		return fmt.Errorf("解析目录响应失败: %w", err)
	}

	sumNum, _ := strconv.Atoi(resp.SumNum)
	log.Info().Str("device_id", deviceID).Int("sum_num", sumNum).Msg("收到目录通知")

	// 转换为 Channel 模型
	channels := make([]model.Channel, 0)
	if resp.DeviceList != nil {
		channels = make([]model.Channel, 0, len(resp.DeviceList.Item))
		for _, d := range resp.DeviceList.Item {
			channel := model.Channel{
				ChannelID:    d.DeviceID,
				DeviceID:     deviceID,
				Name:         d.Name,
				Manufacturer: d.Manufacturer,
				Model:        d.Model,
				Owner:        d.Owner,
				CivilCode:    d.CivilCode,
				Block:        d.Block,
				Address:      d.Address,
				ParentID:     d.ParentID,
				SafetyWay:    d.SafetyWay,
				RegisterWay:  d.RegisterWay,
				Secrecy:      d.Secrecy,
				IP:           d.IPAddress,
				Status:       d.Status,
			}

			// 解析端口
			if d.Port != "" {
				fmt.Sscanf(d.Port, "%d", &channel.Port)
			}

			// 解析经纬度
			if d.Longitude != "" {
				fmt.Sscanf(d.Longitude, "%f", &channel.Longitude)
			}
			if d.Latitude != "" {
				fmt.Sscanf(d.Latitude, "%f", &channel.Latitude)
			}

			channels = append(channels, channel)
		}
	}

	// 保存到数据库
	return h.deviceService.OnCatalogReceived(deviceID, channels)
}

// getResponse 从事务获取响应
func getResponse(tx sip.ClientTransaction) (*sip.Response, error) {
	select {
	case <-tx.Done():
		return nil, fmt.Errorf("事务超时")
	case res := <-tx.Responses():
		return res, nil
	}
}
