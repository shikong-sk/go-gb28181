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

// AlarmSubscription 报警订阅状态
type AlarmSubscription struct {
	DeviceID     string             // 设备ID
	SN           string             // 序列号
	SubscribeID  string             // 订阅ID
	Expires      int                // 过期时间（秒）
	SubscribedAt time.Time          // 订阅时间
	Cancel       context.CancelFunc // 取消函数
}

// AlarmSubscriptionService 报警订阅服务
// 管理报警订阅的生命周期，支持自动续订和报警去重
type AlarmSubscriptionService struct {
	client       *sipgo.Client
	deviceRepo   *repository.DeviceRepository
	alarmRepo    *repository.AlarmRepository
	alarmService *AlarmService
	eventService *EventService // WebSocket 推送服务
	localID      string        // 本地设备 ID
	localIP      string        // 本地 IP
	localPort    int           // 本地端口

	subscriptions sync.Map // key: "Alarm:DeviceID:SN", value: *AlarmSubscription
	alarmCache    sync.Map // key: "DeviceID:AlarmTime:AlarmMethod", value: time.Time (报警去重缓存)
	mu            sync.RWMutex
}

// NewAlarmSubscriptionService 创建报警订阅服务
func NewAlarmSubscriptionService(
	client *sipgo.Client,
	deviceRepo *repository.DeviceRepository,
	alarmRepo *repository.AlarmRepository,
	alarmService *AlarmService,
	eventService *EventService,
	localID, localIP string,
	localPort int,
) *AlarmSubscriptionService {
	return &AlarmSubscriptionService{
		client:       client,
		deviceRepo:   deviceRepo,
		alarmRepo:    alarmRepo,
		alarmService: alarmService,
		eventService: eventService,
		localID:      localID,
		localIP:      localIP,
		localPort:    localPort,
	}
}

// Subscribe 发起报警订阅
// expires: 订阅有效期（秒）
func (s *AlarmSubscriptionService) Subscribe(deviceID string, expires int) error {
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
	subscribeID := utils.GenerateSN()

	// 创建报警订阅请求
	subscribeReq := manscdp.NewAlarmSubscribeReq(sn, deviceID, strconv.Itoa(expires), subscribeID)

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
	event := sip.NewHeader("Event", "Alarm")

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
	sub := &AlarmSubscription{
		DeviceID:     deviceID,
		SN:           sn,
		SubscribeID:  subscribeID,
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
		log.Error().Err(err).Str("device_id", deviceID).Msg("发送报警订阅请求失败")
		return fmt.Errorf("发送报警订阅请求失败: %w", err)
	}
	defer tx.Terminate()

	log.Info().
		Str("device_id", deviceID).
		Str("sn", sn).
		Str("subscribe_id", subscribeID).
		Int("expires", expires).
		Msg("报警订阅请求已发送")

	// 启动自动续订定时器
	go s.autoRenewal(deviceID, sn, expires)

	return nil
}

// Unsubscribe 取消报警订阅
func (s *AlarmSubscriptionService) Unsubscribe(deviceID, sn string) error {
	key := s.buildKey(deviceID, sn)

	value, ok := s.subscriptions.Load(key)
	if !ok {
		return fmt.Errorf("订阅不存在: %s", key)
	}

	sub := value.(*AlarmSubscription)

	// 取消订阅上下文
	if sub.Cancel != nil {
		sub.Cancel()
	}

	// 从存储中删除
	s.subscriptions.Delete(key)

	log.Info().
		Str("device_id", deviceID).
		Str("sn", sn).
		Msg("报警订阅已取消")

	return nil
}

// HandleAlarmNotify 处理报警通知
// COMPAT_JAVA: 报警去重窗口 5 分钟
// 注意：设备可以主动推送报警，不依赖于订阅
func (s *AlarmSubscriptionService) HandleAlarmNotify(notify *manscdp.AlarmNotify) error {
	deviceID := notify.DeviceID
	sn := notify.SN

	log.Info().
		Str("device_id", deviceID).
		Str("sn", sn).
		Str("alarm_priority", notify.AlarmPriority).
		Str("alarm_method", notify.AlarmMethod).
		Str("alarm_time", notify.AlarmTime).
		Msg("收到报警通知")

	// 检查是否为订阅报警（用于日志记录）
	key := s.buildKey(deviceID, sn)
	isSubscribed := false
	if _, ok := s.subscriptions.Load(key); ok {
		isSubscribed = true
		log.Debug().Str("device_id", deviceID).Str("sn", sn).Msg("订阅报警通知")
	} else {
		log.Debug().Str("device_id", deviceID).Str("sn", sn).Msg("设备主动推送报警")
	}

	// 报警去重检查 (5分钟窗口) - 无论是否订阅都要去重
	if s.isDuplicateAlarm(notify) {
		log.Debug().
			Str("device_id", deviceID).
			Str("alarm_time", notify.AlarmTime).
			Str("alarm_method", notify.AlarmMethod).
			Msg("报警重复，已忽略")
		return nil
	}

	// 保存报警信息到数据库 - 无论是否订阅都要保存
	alarmRecord := &model.Alarm{
		DeviceID:         notify.DeviceID,
		AlarmPriority:    notify.AlarmPriority,
		AlarmMethod:      notify.AlarmMethod,
		AlarmTime:        notify.AlarmTime,
		AlarmDescription: notify.AlarmDescription,
		AlarmInfo:        notify.AlarmInfo,
	}

	if s.alarmService != nil {
		if err := s.alarmService.SaveAlarm(alarmRecord); err != nil {
			log.Error().Err(err).Str("device_id", deviceID).Msg("保存报警失败")
			return err
		}
	} else if s.alarmRepo != nil {
		// 如果没有 alarmService，直接保存
		if err := s.alarmRepo.Create(alarmRecord); err != nil {
			log.Error().Err(err).Str("device_id", deviceID).Msg("保存报警失败")
			return err
		}
	}

	// WebSocket 推送报警事件
	if s.eventService != nil {
		s.eventService.PublishAlarm(
			notify.DeviceID,
			notify.AlarmPriority,
			notify.AlarmMethod,
			notify.AlarmTime,
			notify.AlarmDescription,
		)
	}

	log.Info().
		Str("device_id", deviceID).
		Str("alarm_priority", notify.AlarmPriority).
		Bool("subscribed", isSubscribed).
		Msg("报警已保存并推送")

	return nil
}

// isDuplicateAlarm 检查报警是否重复
// COMPAT_JAVA: 使用 5 分钟去重窗口
func (s *AlarmSubscriptionService) isDuplicateAlarm(notify *manscdp.AlarmNotify) bool {
	// 构建报警唯一标识：DeviceID + AlarmTime + AlarmMethod
	key := fmt.Sprintf("%s:%s:%s", notify.DeviceID, notify.AlarmTime, notify.AlarmMethod)

	now := time.Now()

	// 检查缓存中是否存在该报警
	if value, ok := s.alarmCache.Load(key); ok {
		lastTime := value.(time.Time)
		// 5 分钟去重窗口
		if now.Sub(lastTime) < 5*time.Minute {
			return true
		}
	}

	// 更新缓存
	s.alarmCache.Store(key, now)

	// 定期清理过期缓存（超过 10 分钟的记录）
	go s.cleanupAlarmCache()

	return false
}

// cleanupAlarmCache 清理过期的报警缓存
func (s *AlarmSubscriptionService) cleanupAlarmCache() {
	now := time.Now()
	count := 0

	s.alarmCache.Range(func(key, value interface{}) bool {
		lastTime := value.(time.Time)
		// 清理超过 10 分钟的缓存记录
		if now.Sub(lastTime) > 10*time.Minute {
			s.alarmCache.Delete(key)
			count++
		}
		return true
	})

	if count > 0 {
		log.Debug().Int("cleaned", count).Msg("清理过期报警缓存")
	}
}

// autoRenewal 自动续订
// 在订阅过期前 60 秒自动续订（COMPAT_JAVA）
func (s *AlarmSubscriptionService) autoRenewal(deviceID, sn string, expires int) {
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
		Msg("自动续订报警订阅")

	// 重新订阅
	if err := s.Subscribe(deviceID, expires); err != nil {
		log.Error().Err(err).
			Str("device_id", deviceID).
			Str("sn", sn).
			Msg("自动续订报警订阅失败")
	}
}

// GetSubscription 获取订阅信息
func (s *AlarmSubscriptionService) GetSubscription(deviceID, sn string) (*AlarmSubscription, bool) {
	key := s.buildKey(deviceID, sn)

	if value, ok := s.subscriptions.Load(key); ok {
		return value.(*AlarmSubscription), true
	}
	return nil, false
}

// ListSubscriptions 列出所有订阅
func (s *AlarmSubscriptionService) ListSubscriptions() []*AlarmSubscription {
	var subs []*AlarmSubscription
	s.subscriptions.Range(func(key, value interface{}) bool {
		sub := value.(*AlarmSubscription)
		subs = append(subs, sub)
		return true
	})
	return subs
}

// buildKey 构建订阅唯一标识 key
// 格式：Alarm:DeviceID:SN（COMPAT_JAVA）
func (s *AlarmSubscriptionService) buildKey(deviceID, sn string) string {
	return "Alarm:" + deviceID + ":" + sn
}
