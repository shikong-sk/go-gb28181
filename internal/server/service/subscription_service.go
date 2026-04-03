package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
)

// Subscription 表示一个异步响应订阅
type Subscription struct {
	CmdType   string             // 命令类型，如 "RecordInfo", "DeviceStatus", "Position"
	DeviceID  string             // 设备ID
	SN        string             // 序列号
	ResultCh  chan interface{}   // 响应结果通道
	Context   context.Context    // 上下文
	Cancel    context.CancelFunc // 取消函数
	CreatedAt time.Time          // 创建时间
}

// SubscriptionService 通用的异步响应订阅服务
// 用于管理 GB28181 协议中各类查询请求的响应订阅
// 支持 RecordInfo、DeviceStatus、Position 等多种命令类型
type SubscriptionService struct {
	subscriptions sync.Map // key: "deviceID:sn", value: *Subscription
	timeout       time.Duration
	mu            sync.RWMutex
}

// NewSubscriptionService 创建订阅服务
// 默认超时时间可在后续通过配置注入
func NewSubscriptionService(defaultTimeout time.Duration) *SubscriptionService {
	if defaultTimeout <= 0 {
		defaultTimeout = 30 * time.Second
	}
	return &SubscriptionService{
		timeout: defaultTimeout,
	}
}

// Subscribe 创建一个新的订阅
// cmdType: 命令类型，用于日志和调试
// deviceID: 设备ID
// sn: 序列号，与 deviceID 组成唯一标识
// timeout: 订阅超时时间，如果为0则使用默认超时
func (s *SubscriptionService) Subscribe(cmdType, deviceID, sn string, timeout time.Duration) (*Subscription, error) {
	if deviceID == "" || sn == "" {
		return nil, fmt.Errorf("deviceID 和 sn 不能为空")
	}

	if timeout <= 0 {
		timeout = s.timeout
	}

	key := s.buildKey(deviceID, sn)

	// 检查是否已存在相同订阅
	if _, ok := s.subscriptions.Load(key); ok {
		return nil, fmt.Errorf("订阅已存在: %s", key)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	sub := &Subscription{
		CmdType:   cmdType,
		DeviceID:  deviceID,
		SN:        sn,
		ResultCh:  make(chan interface{}, 1),
		Context:   ctx,
		Cancel:    cancel,
		CreatedAt: time.Now(),
	}

	s.subscriptions.Store(key, sub)

	log.Debug().
		Str("cmd_type", cmdType).
		Str("device_id", deviceID).
		Str("sn", sn).
		Dur("timeout", timeout).
		Msg("创建订阅")

	return sub, nil
}

// Unsubscribe 取消订阅并清理资源
func (s *SubscriptionService) Unsubscribe(deviceID, sn string) {
	key := s.buildKey(deviceID, sn)

	if value, ok := s.subscriptions.LoadAndDelete(key); ok {
		sub := value.(*Subscription)
		sub.Cancel()
		close(sub.ResultCh)
		log.Debug().
			Str("cmd_type", sub.CmdType).
			Str("device_id", deviceID).
			Str("sn", sn).
			Msg("取消订阅")
	}
}

// NotifyResponse 通知响应到达
// 将响应数据写入订阅的结果通道
// 如果订阅不存在或已关闭，不会阻塞
func (s *SubscriptionService) NotifyResponse(deviceID, sn string, response interface{}) bool {
	key := s.buildKey(deviceID, sn)

	value, ok := s.subscriptions.Load(key)
	if !ok {
		log.Debug().
			Str("device_id", deviceID).
			Str("sn", sn).
			Msg("订阅不存在，忽略响应")
		return false
	}

	sub := value.(*Subscription)

	// 检查上下文是否已过期
	if sub.Context.Err() != nil {
		log.Debug().
			Str("cmd_type", sub.CmdType).
			Str("device_id", deviceID).
			Str("sn", sn).
			Err(sub.Context.Err()).
			Msg("订阅已过期，忽略响应")
		return false
	}

	// 非阻塞写入
	select {
	case sub.ResultCh <- response:
		log.Debug().
			Str("cmd_type", sub.CmdType).
			Str("device_id", deviceID).
			Str("sn", sn).
			Msg("响应已写入订阅通道")
		return true
	default:
		log.Warn().
			Str("cmd_type", sub.CmdType).
			Str("device_id", deviceID).
			Str("sn", sn).
			Msg("订阅通道已满，响应被丢弃")
		return false
	}
}

// GetSubscription 获取订阅信息
// 用于检查订阅是否存在或获取订阅详情
func (s *SubscriptionService) GetSubscription(deviceID, sn string) (*Subscription, bool) {
	key := s.buildKey(deviceID, sn)

	if value, ok := s.subscriptions.Load(key); ok {
		return value.(*Subscription), true
	}
	return nil, false
}

// CleanupExpired 清理过期的订阅
// 应定期调用（如每分钟），防止内存泄漏
func (s *SubscriptionService) CleanupExpired() int {
	count := 0
	now := time.Now()

	s.subscriptions.Range(func(key, value interface{}) bool {
		sub := value.(*Subscription)

		// 检查上下文是否已过期
		if sub.Context.Err() != nil {
			s.subscriptions.Delete(key)
			sub.Cancel()
			close(sub.ResultCh)
			count++
			log.Debug().
				Str("cmd_type", sub.CmdType).
				Str("device_id", sub.DeviceID).
				Str("sn", sub.SN).
				Msg("清理过期订阅")
			return true
		}

		// 额外检查：如果创建时间超过3倍超时时间，也清理
		// 这是安全网，防止 context 未正确取消的情况
		maxAge := s.timeout * 3
		if now.Sub(sub.CreatedAt) > maxAge {
			s.subscriptions.Delete(key)
			sub.Cancel()
			close(sub.ResultCh)
			count++
			log.Warn().
				Str("cmd_type", sub.CmdType).
				Str("device_id", sub.DeviceID).
				Str("sn", sub.SN).
				Dur("age", now.Sub(sub.CreatedAt)).
				Msg("清理超龄订阅")
		}

		return true
	})

	if count > 0 {
		log.Info().Int("count", count).Msg("清理过期订阅完成")
	}

	return count
}

// Count 返回当前订阅数量
func (s *SubscriptionService) Count() int {
	count := 0
	s.subscriptions.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// buildKey 构建订阅的唯一标识 key
func (s *SubscriptionService) buildKey(deviceID, sn string) string {
	return deviceID + ":" + sn
}
