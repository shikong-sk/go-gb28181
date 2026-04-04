package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// CatalogSubscription 目录订阅状态
type CatalogSubscription struct {
	DeviceID     string             // 设备ID
	SN           string             // 序列号
	EventID      string             // 事件ID
	Expires      int                // 过期时间（秒）
	SubscribedAt time.Time          // 订阅时间
	Cancel       context.CancelFunc // 取消函数
}

// CatalogSubscriptionService 目录订阅服务
// 管理目录订阅的生命周期，支持自动续订
type CatalogSubscriptionService struct {
	client      *sipgo.Client
	deviceRepo  *repository.DeviceRepository
	channelRepo *repository.ChannelRepository
	localID     string // 本地设备 ID
	localIP     string // 本地 IP
	localPort   int    // 本地端口

	subscriptions sync.Map // key: "Catalog:DeviceID:SN", value: *CatalogSubscription
	mu            sync.RWMutex
}

// NewCatalogSubscriptionService 创建目录订阅服务
func NewCatalogSubscriptionService(
	client *sipgo.Client,
	deviceRepo *repository.DeviceRepository,
	channelRepo *repository.ChannelRepository,
	localID, localIP string,
	localPort int,
) *CatalogSubscriptionService {
	return &CatalogSubscriptionService{
		client:      client,
		deviceRepo:  deviceRepo,
		channelRepo: channelRepo,
		localID:     localID,
		localIP:     localIP,
		localPort:   localPort,
	}
}

// Subscribe 发起目录订阅
// expires: 订阅有效期（秒）
func (s *CatalogSubscriptionService) Subscribe(deviceID string, expires int) error {
	if s.client == nil {
		return fmt.Errorf("SIP 客户端未初始化")
	}

	// 从数据库获取设备信息
	device, err := s.deviceRepo.GetByDeviceID(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}

	// 生成订阅参数
	sn := utils.GenerateSN()
	eventID := utils.GenerateSN() // 使用 SN 生成器生成 EventID

	// 创建订阅请求
	subscribeReq := manscdp.NewSubscribeReq(sn, deviceID, strconv.Itoa(expires), eventID)

	// 编码为 XML (GBK)
	body, err := utils.XMLMarshal(subscribeReq, "gbk")
	if err != nil {
		return fmt.Errorf("XML 编码失败: %w", err)
	}

	// 构建目标 URI
	target := sip.Uri{
		User: deviceID,
		Host: device.IP,
		Port: device.Port,
	}

	// 创建 SUBSCRIBE 请求
	req := sip.NewRequest(sip.SUBSCRIBE, target)

	// 设置 SIP 头
	from := sip.NewHeader("From", fmt.Sprintf("<sip:%s@%s:%d>;tag=%s", s.localID, s.localIP, s.localPort, utils.GenerateFromTag()))
	to := sip.NewHeader("To", fmt.Sprintf("<sip:%s@%s:%d>", deviceID, device.IP, device.Port))
	callID := sip.NewHeader("Call-ID", fmt.Sprintf("%d@%s", time.Now().UnixNano(), s.localIP))
	cseq := sip.NewHeader("CSeq", "1 SUBSCRIBE")
	contentType := sip.NewHeader("Content-Type", "Application/MANSCDP+xml")
	contact := sip.NewHeader("Contact", fmt.Sprintf("<sip:%s@%s:%d>", s.localID, s.localIP, s.localPort))
	expiresHeader := sip.NewHeader("Expires", strconv.Itoa(expires))
	event := sip.NewHeader("Event", "Catalog")

	req.AppendHeader(from)
	req.AppendHeader(to)
	req.AppendHeader(callID)
	req.AppendHeader(cseq)
	req.AppendHeader(contentType)
	req.AppendHeader(contact)
	req.AppendHeader(expiresHeader)
	req.AppendHeader(event)
	req.SetBody(body)

	// 创建订阅上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 保存订阅状态
	sub := &CatalogSubscription{
		DeviceID:     deviceID,
		SN:           sn,
		EventID:      eventID,
		Expires:      expires,
		SubscribedAt: time.Now(),
		Cancel:       cancel,
	}

	key := s.buildKey(deviceID, sn)
	s.subscriptions.Store(key, sub)

	// 发送 SUBSCRIBE 请求
	tx, err := s.client.TransactionRequest(ctx, req)
	if err != nil {
		s.subscriptions.Delete(key)
		cancel()
		log.Error().Err(err).Str("device_id", deviceID).Msg("发送目录订阅请求失败")
		return fmt.Errorf("发送目录订阅请求失败: %w", err)
	}
	defer tx.Terminate()

	log.Info().
		Str("device_id", deviceID).
		Str("sn", sn).
		Str("event_id", eventID).
		Int("expires", expires).
		Msg("目录订阅请求已发送")

	// 启动自动续订定时器
	go s.autoRenewal(deviceID, sn, expires)

	return nil
}

// Unsubscribe 取消目录订阅
func (s *CatalogSubscriptionService) Unsubscribe(deviceID, sn string) error {
	key := s.buildKey(deviceID, sn)

	value, ok := s.subscriptions.Load(key)
	if !ok {
		return fmt.Errorf("订阅不存在: %s", key)
	}

	sub := value.(*CatalogSubscription)

	// 取消订阅上下文
	if sub.Cancel != nil {
		sub.Cancel()
	}

	// 从存储中删除
	s.subscriptions.Delete(key)

	log.Info().
		Str("device_id", deviceID).
		Str("sn", sn).
		Msg("目录订阅已取消")

	return nil
}

// HandleNotify 处理目录变化通知
func (s *CatalogSubscriptionService) HandleNotify(notify *manscdp.CatalogNotify) error {
	deviceID := notify.DeviceID
	sn := notify.SN

	log.Info().
		Str("device_id", deviceID).
		Str("sn", sn).
		Str("sum_num", notify.SumNum).
		Msg("收到目录变化通知")

	// 检查订阅是否存在
	key := s.buildKey(deviceID, sn)
	if _, ok := s.subscriptions.Load(key); !ok {
		log.Warn().
			Str("device_id", deviceID).
			Str("sn", sn).
			Msg("收到未知订阅的通知")
		return nil
	}

	// 处理设备列表
	if notify.DeviceList != nil && len(notify.DeviceList.Item) > 0 {
		for _, item := range notify.DeviceList.Item {
			// 保存或更新通道信息
			channel := &model.Channel{
				DeviceID:     deviceID,
				ChannelID:    item.DeviceID,
				Name:         item.Name,
				Manufacturer: item.Manufacturer,
				Model:        item.Model,
				Owner:        item.Owner,
				Address:      item.Address,
				Port:         parsePort(item.Port),
				Status:       item.Status,
				Longitude:    parseFloat(item.Longitude),
				Latitude:     parseFloat(item.Latitude),
			}

			if err := s.channelRepo.Upsert(channel); err != nil {
				log.Error().Err(err).
					Str("device_id", deviceID).
					Str("channel_id", item.DeviceID).
					Msg("保存通道失败")
				continue
			}

			log.Debug().
				Str("device_id", deviceID).
				Str("channel_id", item.DeviceID).
				Str("name", item.Name).
				Msg("通道信息已更新")
		}
	}

	return nil
}

// autoRenewal 自动续订
// 在订阅过期前 60 秒自动续订（COMPAT_JAVA）
func (s *CatalogSubscriptionService) autoRenewal(deviceID, sn string, expires int) {
	// 提前 60 秒续订
	renewalTime := time.Duration(expires-60) * time.Second
	if renewalTime < 0 {
		renewalTime = time.Duration(expires) * time.Second
	}

	timer := time.NewTimer(renewalTime)
	defer timer.Stop()

	<-timer.C

	// 检查订阅是否还存在
	key := s.buildKey(deviceID, sn)
	if _, ok := s.subscriptions.Load(key); !ok {
		return
	}

	log.Info().
		Str("device_id", deviceID).
		Str("sn", sn).
		Msg("自动续订目录订阅")

	// 重新订阅
	if err := s.Subscribe(deviceID, expires); err != nil {
		log.Error().Err(err).
			Str("device_id", deviceID).
			Str("sn", sn).
			Msg("自动续订失败")
	}
}

// GetSubscription 获取订阅信息
func (s *CatalogSubscriptionService) GetSubscription(deviceID, sn string) (*CatalogSubscription, bool) {
	key := s.buildKey(deviceID, sn)

	if value, ok := s.subscriptions.Load(key); ok {
		return value.(*CatalogSubscription), true
	}
	return nil, false
}

// ListSubscriptions 列出所有订阅
func (s *CatalogSubscriptionService) ListSubscriptions() []*CatalogSubscription {
	var subs []*CatalogSubscription
	s.subscriptions.Range(func(key, value interface{}) bool {
		sub := value.(*CatalogSubscription)
		subs = append(subs, sub)
		return true
	})
	return subs
}

// buildKey 构建订阅唯一标识 key
// 格式：Catalog:DeviceID:SN（COMPAT_JAVA）
func (s *CatalogSubscriptionService) buildKey(deviceID, sn string) string {
	return "Catalog:" + deviceID + ":" + sn
}

// parsePort 解析端口字符串
func parsePort(portStr string) int {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}
	return port
}

// parseFloat 解析浮点数字符串
func parseFloat(floatStr string) float64 {
	val, err := strconv.ParseFloat(floatStr, 64)
	if err != nil {
		return 0
	}
	return val
}
