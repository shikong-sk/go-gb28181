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
	EventDeviceOnline     EventType = "device_online"
	EventDeviceOffline    EventType = "device_offline"
	EventAlarm            EventType = "alarm"
	EventChannelStatus    EventType = "channel_status"
	EventPlaySessionStart EventType = "play_session_start"
	EventPlaySessionStop  EventType = "play_session_stop"
	EventDownloadProgress EventType = "download_progress"
	EventChannelUpdate    EventType = "channel_update"
)

// Event 事件结构
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp int64       `json:"timestamp"` // 毫秒级时间戳
	Data      interface{} `json:"data"`
}

// DeviceOnlineEvent 设备上线事件数据
type DeviceOnlineEvent struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name,omitempty"`
	IP         string `json:"ip,omitempty"`
	Port       int    `json:"port,omitempty"`
}

// DeviceOfflineEvent 设备离线事件数据
type DeviceOfflineEvent struct {
	DeviceID string `json:"device_id"`
	Reason   string `json:"reason"` // "timeout" | "unregister"
}

// AlarmEvent 报警事件数据
type AlarmEvent struct {
	ID               int    `json:"id,omitempty"`
	DeviceID         string `json:"device_id"`
	AlarmPriority    string `json:"alarm_priority"`
	AlarmMethod      string `json:"alarm_method"`
	AlarmTime        string `json:"alarm_time"`
	AlarmDescription string `json:"alarm_description"`
}

// PlaySessionStartEvent 播放会话开始事件数据
type PlaySessionStartEvent struct {
	StreamID  string `json:"stream_id"`
	DeviceID  string `json:"device_id"`
	ChannelID string `json:"channel_id"`
	Mode      string `json:"mode"` // "live" | "playback" | "download"
}

// PlaySessionStopEvent 播放会话停止事件数据
type PlaySessionStopEvent struct {
	StreamID  string `json:"stream_id"`
	DeviceID  string `json:"device_id"`
	ChannelID string `json:"channel_id"`
	Mode      string `json:"mode"`
	Reason    string `json:"reason"` // "user_stop" | "timeout" | "error"
}

// DownloadProgressEvent 下载进度事件数据
type DownloadProgressEvent struct {
	StreamID    string `json:"stream_id"`
	DeviceID    string `json:"device_id"`
	ChannelID   string `json:"channel_id"`
	Progress    int    `json:"progress"`     // 下载进度百分比 (0-100)
	CurrentTime string `json:"current_time"` // 当前下载时间点
	EndTime     string `json:"end_time"`     // 结束时间
}

// ChannelUpdateEvent 通道状态更新事件数据
type ChannelUpdateEvent struct {
	DeviceID  string `json:"device_id"`
	ChannelID string `json:"channel_id"`
	Status    string `json:"status"` // "online" | "offline"
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

// PublishAlarmWithID 发布带ID的报警事件
func (s *EventService) PublishAlarmWithID(id int, deviceID, priority, method, alarmTime, description string) {
	data := AlarmEvent{
		ID:               id,
		DeviceID:         deviceID,
		AlarmPriority:    priority,
		AlarmMethod:      method,
		AlarmTime:        alarmTime,
		AlarmDescription: description,
	}
	s.Publish(EventAlarm, data)
}

// PublishPlaySessionStart 发布播放会话开始事件
func (s *EventService) PublishPlaySessionStart(streamID, deviceID, channelID, mode string) {
	data := PlaySessionStartEvent{
		StreamID:  streamID,
		DeviceID:  deviceID,
		ChannelID: channelID,
		Mode:      mode,
	}
	s.Publish(EventPlaySessionStart, data)
}

// PublishPlaySessionStop 发布播放会话停止事件
func (s *EventService) PublishPlaySessionStop(streamID, deviceID, channelID, mode, reason string) {
	data := PlaySessionStopEvent{
		StreamID:  streamID,
		DeviceID:  deviceID,
		ChannelID: channelID,
		Mode:      mode,
		Reason:    reason,
	}
	s.Publish(EventPlaySessionStop, data)
}

// PublishDownloadProgress 发布下载进度事件
func (s *EventService) PublishDownloadProgress(streamID, deviceID, channelID string, progress int, currentTime, endTime string) {
	data := DownloadProgressEvent{
		StreamID:    streamID,
		DeviceID:    deviceID,
		ChannelID:   channelID,
		Progress:    progress,
		CurrentTime: currentTime,
		EndTime:     endTime,
	}
	s.Publish(EventDownloadProgress, data)
}

// PublishChannelUpdate 发布通道状态更新事件
func (s *EventService) PublishChannelUpdate(deviceID, channelID, status string) {
	data := ChannelUpdateEvent{
		DeviceID:  deviceID,
		ChannelID: channelID,
		Status:    status,
	}
	s.Publish(EventChannelUpdate, data)
}
