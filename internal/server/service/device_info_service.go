package service

import (
	"context"
	"fmt"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// DeviceInfoService 设备信息查询服务
type DeviceInfoService struct {
	client              *sipgo.Client
	deviceService       *DeviceService
	subscriptionService *SubscriptionService
	localID             string
	localIP             string
	localPort           int
}

// NewDeviceInfoService 创建设备信息查询服务
func NewDeviceInfoService(
	client *sipgo.Client,
	deviceService *DeviceService,
	subscriptionService *SubscriptionService,
	localID, localIP string,
	localPort int,
) *DeviceInfoService {
	return &DeviceInfoService{
		client:              client,
		deviceService:       deviceService,
		subscriptionService: subscriptionService,
		localID:             localID,
		localIP:             localIP,
		localPort:           localPort,
	}
}

// QueryDeviceInfo 查询设备信息
// deviceID: 设备ID
// timeout: 超时时间，如果为0则使用默认超时
func (s *DeviceInfoService) QueryDeviceInfo(deviceID string, timeout time.Duration) (*manscdp.DeviceInfoResp, error) {
	if s.client == nil {
		return nil, fmt.Errorf("SIP 客户端未初始化")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 获取设备信息
	device, err := s.deviceService.GetDevice(deviceID)
	if err != nil {
		return nil, fmt.Errorf("获取设备失败: %w", err)
	}

	// 生成安全序列号
	sn := utils.GenerateSN()

	// 创建订阅
	subscription, err := s.subscriptionService.Subscribe(
		cmdtype.DeviceInfo,
		deviceID,
		sn,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("创建订阅失败: %w", err)
	}
	defer s.subscriptionService.Unsubscribe(deviceID, sn)

	// 构建设备信息查询请求
	query := manscdp.NewDeviceInfoReq(sn, deviceID)
	body, err := utils.XMLMarshal(query, "gbk")
	if err != nil {
		return nil, fmt.Errorf("序列化设备信息查询失败: %w", err)
	}

	// 构建 SIP MESSAGE 请求
	target := sip.Uri{
		User: deviceID,
		Host: device.IP,
		Port: device.Port,
	}
	req := sip.NewRequest(sip.MESSAGE, target)
	from := sip.NewHeader("From", fmt.Sprintf("<sip:%s@%s:%d>;tag=%s", s.localID, s.localIP, s.localPort, utils.GenerateFromTag()))
	to := sip.NewHeader("To", fmt.Sprintf("<sip:%s@%s:%d>", deviceID, device.IP, device.Port))
	callID := sip.NewHeader("Call-ID", fmt.Sprintf("%d@%s", time.Now().UnixNano(), s.localIP))
	cseq := sip.NewHeader("CSeq", "1 MESSAGE")
	contentType := sip.NewHeader("Content-Type", "Application/MANSCDP+xml")
	req.AppendHeader(from)
	req.AppendHeader(to)
	req.AppendHeader(callID)
	req.AppendHeader(cseq)
	req.AppendHeader(contentType)
	req.SetBody(body)

	// 发送 SIP MESSAGE
	requestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := s.client.TransactionRequest(requestCtx, req)
	if err != nil {
		return nil, fmt.Errorf("发送设备信息查询失败: %w", err)
	}
	defer tx.Terminate()

	log.Info().Str("device_id", deviceID).Str("sn", sn).Msg("设备信息查询请求已发送")

	// 等待响应
	select {
	case response := <-subscription.ResultCh:
		resp, ok := response.(*manscdp.DeviceInfoResp)
		if !ok {
			return nil, fmt.Errorf("无效的响应类型")
		}
		return resp, nil
	case <-subscription.Context.Done():
		return nil, fmt.Errorf("等待设备信息响应超时")
	}
}
