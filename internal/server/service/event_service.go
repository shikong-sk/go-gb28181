package service

import (
	"encoding/json"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/websocket"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
)

// EventType 事件类型
type EventType string

const (
	EventDeviceOnline  EventType = "device_online"
	EventDeviceOffline EventType = "device_offline"
	EventAlarm         EventType = "alarm"
	EventChannelStatus EventType = "channel_status"
)

// Event 事件结构
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp int64       `json:"timestamp"` // 毫秒级时间戳
	Data      interface{} `json:"data"`
}

// DeviceOnlineEvent 设备上线事件数据
type DeviceOnlineEvent struct {
	DeviceID string `json:"deviceId"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
}

// DeviceOfflineEvent 设备离线事件数据
type DeviceOfflineEvent struct {
	DeviceID string `json:"deviceId"`
	Reason   string `json:"reason"` // "timeout" | "unregister"
}

// AlarmEvent 报警事件数据
type AlarmEvent struct {
	DeviceID         string `json:"deviceId"`
	AlarmPriority    string `json:"alarmPriority"`
	AlarmMethod      string `json:"alarmMethod"`
	AlarmTime        string `json:"alarmTime"`
	AlarmDescription string `json:"alarmDescription"`
}

// EventService 事件发布服务
type EventService struct {
	wsManager *websocket.WebSocketManager
}

// NewEventService 创建事件服务
func NewEventService() *EventService {
	return &EventService{}
}

// SetWebSocketManager 设置 WebSocket 管理器
func (s *EventService) SetWebSocketManager(wsManager *websocket.WebSocketManager) {
	s.wsManager = wsManager
}

// Publish 发布事件
func (s *EventService) Publish(eventType EventType, data interface{}) {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now().UnixMilli(),
		Data:      data,
	}

	// JSON 序列化
	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Error().Err(err).Str("type", string(eventType)).Msg("事件序列化失败")
		return
	}

	// 广播到 WebSocket
	if s.wsManager != nil {
		s.wsManager.Broadcast(jsonData)
		log.Debug().Str("type", string(eventType)).Msg("事件已发布")
	} else {
		log.Warn().Str("type", string(eventType)).Msg("WebSocket 管理器未初始化，事件未推送")
	}
}

// PublishDeviceOnline 发布设备上线事件
func (s *EventService) PublishDeviceOnline(deviceID, ip string, port int) {
	data := DeviceOnlineEvent{
		DeviceID: deviceID,
		IP:       ip,
		Port:     port,
	}
	s.Publish(EventDeviceOnline, data)
}

// PublishDeviceOffline 发布设备离线事件
func (s *EventService) PublishDeviceOffline(deviceID, reason string) {
	data := DeviceOfflineEvent{
		DeviceID: deviceID,
		Reason:   reason,
	}
	s.Publish(EventDeviceOffline, data)
}

// PublishAlarm 发布报警事件
func (s *EventService) PublishAlarm(deviceID, priority, method, alarmTime, description string) {
	data := AlarmEvent{
		DeviceID:         deviceID,
		AlarmPriority:    priority,
		AlarmMethod:      method,
		AlarmTime:        alarmTime,
		AlarmDescription: description,
	}
	s.Publish(EventAlarm, data)
}
