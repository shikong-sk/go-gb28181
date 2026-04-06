package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit/types"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// PlayService 视频点播服务
type PlayService struct {
	client        *sipgo.Client
	zlm           *zlmediakit.ZLMediaKit
	config        PlayConfig
	deviceService *DeviceService // 设备服务，用于获取设备真实 IP
	ssrcService   *SsrcService   // SSRC 管理服务
	sessions      map[string]*PlaySession
	mu            sync.RWMutex
	inviteTimeout int // INVITE 超时时间（秒）
}

// PlayConfig 播放配置
type PlayConfig struct {
	ZLMHost     string // ZLMediaKit 主机 IP
	ZLMPort     int    // ZLMediaKit HTTP 端口
	AppName     string // 应用名, 默认 "rtp"
	LocalId     string // COMPAT_JAVA: 本地平台 ID (用于 Subject header)
	SIPListenIP string // SIP 监听 IP (用于 Contact header)
	SIPPort     int    // SIP 端口 (用于 Contact header)
	RtpPort     int    // RTP 收流端口 (0=自动分配, 推荐61200-61250范围)
}

// PlayMode 播放模式
type PlayMode string

const (
	PlayModeLive     PlayMode = "live"
	PlayModePlayback PlayMode = "playback"
	PlayModeDownload PlayMode = "download"
)

// PlaySession 播放会话
type PlaySession struct {
	StreamId       string     // 流 ID
	DeviceId       string     // 设备 ID
	ChannelId      string     // 通道 ID
	RTPPort        int        // RTP 端口
	StartTime      time.Time  // 会话创建时间
	LastActiveTime time.Time  // 流最后活跃时间（用于超时检测）
	Status         string     // 状态: pending, playing, stopped
	SSRC           string     // SSRC
	Mode           PlayMode   // 播放模式
	RangeStart     *time.Time // 回放开始时间
	RangeEnd       *time.Time // 回放结束时间
	TargetHost     string     // 目标设备 IP
	TargetPort     int        // 目标设备端口
	Via            string     // Via header value
	CallID         string     // Call-ID value
	From           string     // From header value
	To             string     // To header value (from 200 OK response)
	CSeq           int        // CSeq number
	ByeSent        bool       // BYE 是否已发送（用于 INVITE 前 BYE 检查）
	ReaderCount    int        // 当前观看人数（来自 ZLM webhook）

	// 重连相关字段
	RetryCount          int       // 当前重连尝试次数（最多3次）
	LastRetryTime       time.Time // 上次重连尝试时间
	IsReconnecting      bool      // 是否正在重连中
	UserStopped         bool      // 用户是否主动停止（区分用户停止和自动断流）
	DisconnectDetecting bool      // 是否正在等待断流检测结果（防止重复启动检测）

	streamRegistered chan bool // 流注册通知 channel（内部使用）
}

// PlayResult 播放结果
type PlayResult struct {
	StreamId string   `json:"stream_id"`
	Urls     []string `json:"urls"`
	RTPPort  int      `json:"rtp_port"`
	FLVURL   string   `json:"flv_url"`
	HLSURL   string   `json:"hls_url"`
	RTSPURL  string   `json:"rtsp_url"`
	RTMPURL  string   `json:"rtmp_url"`
	Mode     string   `json:"mode"`
}

// 播放状态
const (
	PlayStatusPending = "pending"
	PlayStatusPlaying = "playing"
	PlayStatusStopped = "stopped"
)

// NewPlayService 创建播放服务
func NewPlayService(client *sipgo.Client, zlm *zlmediakit.ZLMediaKit, config PlayConfig, deviceService *DeviceService, ssrcService *SsrcService, inviteTimeout int) *PlayService {
	if config.AppName == "" {
		config.AppName = "rtp"
	}
	// 确保超时时间有效
	if inviteTimeout <= 0 {
		inviteTimeout = 15
	}
	return &PlayService{
		client:        client,
		zlm:           zlm,
		config:        config,
		deviceService: deviceService,
		ssrcService:   ssrcService,
		sessions:      make(map[string]*PlaySession),
		inviteTimeout: inviteTimeout,
	}
}

// Play 开始实时播放
func (s *PlayService) Play(deviceId, channelId string) (*PlayResult, error) {
	return s.startPlay(deviceId, channelId, PlayModeLive, nil, nil)
}

// PlayBack 开始历史录像回放
func (s *PlayService) PlayBack(deviceId, channelId string, startTime, endTime time.Time) (*PlayResult, error) {
	if endTime.Before(startTime) {
		return nil, fmt.Errorf("结束时间不能早于开始时间")
	}
	return s.startPlay(deviceId, channelId, PlayModePlayback, &startTime, &endTime)
}

// Download 开始录像下载（支持倍速）
func (s *PlayService) Download(deviceId, channelId string, startTime, endTime time.Time, speed int) (*PlayResult, error) {
	if endTime.Before(startTime) {
		return nil, fmt.Errorf("结束时间不能早于开始时间")
	}
	// 倍速限制：仅支持 1/2/4
	if speed != 1 && speed != 2 && speed != 4 {
		return nil, fmt.Errorf("下载倍速仅支持 1/2/4")
	}
	return s.startPlay(deviceId, channelId, PlayModeDownload, &startTime, &endTime)
}

func (s *PlayService) startPlay(deviceId, channelId string, mode PlayMode, rangeStart, rangeEnd *time.Time) (*PlayResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("SIP 客户端未初始化")
	}
	if s.zlm == nil {
		return nil, fmt.Errorf("ZLMediaKit 未初始化")
	}

	// 设备单流限制：仅对实时流生效，回放和下载不受限制
	// INVITE 前 BYE 检查：如果旧会话存在且 BYE 未发送，需要先发送 BYE
	if mode == PlayModeLive {
		existingSession := s.GetSessionByDeviceId(deviceId, PlayModeLive)
		if existingSession != nil {
			// 检查 ZLM 流是否真的存在
			if s.isStreamActive(existingSession.StreamId) {
				log.Info().
					Str("device_id", deviceId).
					Str("existing_stream_id", existingSession.StreamId).
					Str("new_channel_id", channelId).
					Bool("bye_sent", existingSession.ByeSent).
					Msg("设备已有活跃实时流，复用现有流地址（不发送BYE）")
				return s.buildPlayResult(existingSession), nil
			}
			// 流已断开，清理旧会话后再创建新会话
			// cleanupSession 会检查 ByeSent 并发送 BYE（如果 BYE 未发送）
			log.Warn().
				Str("device_id", deviceId).
				Str("existing_stream_id", existingSession.StreamId).
				Bool("bye_sent", existingSession.ByeSent).
				Msg("设备已有会话但流已断开，先发送BYE清理旧会话再重新建立")
			s.cleanupSession(existingSession.StreamId)
		}
	}

	streamId := s.generateStreamId(deviceId, channelId, mode, rangeStart, rangeEnd)

	s.mu.RLock()
	if session, exists := s.sessions[streamId]; exists && session.Status == PlayStatusPlaying {
		s.mu.RUnlock()
		// 检查 ZLM 流是否真的存在（避免返回已断开的会话）
		if s.isStreamActive(streamId) {
			log.Info().
				Str("stream_id", streamId).
				Bool("bye_sent", session.ByeSent).
				Msg("会话已存在且流活跃，复用会话（不发送BYE）")
			return s.buildPlayResult(session), nil
		}
		// 流已断开，清理旧会话
		// cleanupSession 会检查 ByeSent 并发送 BYE（如果 BYE 未发送）
		log.Warn().
			Str("stream_id", streamId).
			Bool("bye_sent", session.ByeSent).
			Msg("会话存在但流已断开，先发送BYE清理旧会话再重新建立")
		s.cleanupSession(streamId)
	} else {
		s.mu.RUnlock()
	}

	// 重要：点播场景必须优先让 ZLM 自动分配 RTP 端口。
	// 用户已经明确说明：应以 openRtpServer API 返回的端口为准，
	// 不能把“配置里写了某个固定端口”当成最终收流端口。
	// 这里显式传 0 给 ZLM，避免固定端口占用、过期端口或多会话冲突导致设备推流失败。
	rtpPort := 0
	log.Info().Int("config_rtp_port", s.config.RtpPort).Int("request_rtp_port", rtpPort).Msg("使用自动分配端口调用 OpenRtpServer")
	rtpResp, err := s.zlm.OpenRtpServer(streamId, rtpPort, 0)
	if err != nil {
		log.Error().Err(err).Str("stream_id", streamId).Msg("打开 RTP 服务器失败")
		return nil, fmt.Errorf("打开 RTP 服务器失败: %w", err)
	}
	log.Info().Str("stream_id", streamId).Int("rtp_port", rtpResp.Port).Int("code", rtpResp.Code).
		Msg("RTP 服务器已打开")

	// 验证 RTP Server 是否真的创建成功
	time.Sleep(100 * time.Millisecond) // 等待 100ms 让 ZLMediaKit 完成 RTP Server 创建
	rtpInfo, err := s.zlm.GetRtpInfo(streamId)
	if err != nil {
		log.Warn().Err(err).Str("stream_id", streamId).Msg("查询 RTP 信息失败")
	} else {
		log.Info().Str("stream_id", streamId).Bool("exist", rtpInfo.Exist).Str("ip", rtpInfo.IP).Int("port", rtpInfo.Port).
			Msg("RTP Server 状态验证")
	}

	// 从 Redis 池分配 SSRC
	var ssrc string
	if s.ssrcService != nil {
		var ssrcErr error
		if mode == PlayModePlayback {
			ssrc, ssrcErr = s.ssrcService.GetPlaybackSsrc()
		} else if mode == PlayModeDownload {
			ssrc, ssrcErr = s.ssrcService.GetDownloadSsrc()
		} else {
			ssrc, ssrcErr = s.ssrcService.GetPlaySsrc()
		}
		if ssrcErr != nil {
			_, _ = s.zlm.CloseRtpServer(streamId)
			log.Error().Err(ssrcErr).Str("device_id", deviceId).Str("channel_id", channelId).Str("mode", string(mode)).
				Msg("SSRC 分配失败（池耗尽）")
			return nil, fmt.Errorf("SSRC 分配失败: %w", ssrcErr)
		}
	} else {
		// 降级：Redis 未配置时使用本地生成
		ssrc = s.generateSSRC(streamId, mode)
		log.Warn().Str("device_id", deviceId).Str("channel_id", channelId).
			Msg("SSRC 服务未初始化，使用本地生成（不推荐）")
	}

	// 获取设备地址
	deviceIP := ""
	devicePort := 0
	if s.deviceService != nil {
		device, err := s.deviceService.GetDevice(deviceId)
		if err == nil && device.IP != "" && device.Port > 0 {
			deviceIP = device.IP
			devicePort = device.Port
		}
	}
	if deviceIP == "" || devicePort == 0 {
		_, _ = s.zlm.CloseRtpServer(streamId)
		if s.ssrcService != nil && ssrc != "" {
			s.ssrcService.ReleaseSsrc(ssrc)
		}
		return nil, fmt.Errorf("设备 %s 缺少有效的 SIP 地址，无法发起点播", deviceId)
	}

	session := &PlaySession{
		StreamId:         streamId,
		DeviceId:         deviceId,
		ChannelId:        channelId,
		RTPPort:          rtpResp.Port,
		StartTime:        time.Now(),
		Status:           PlayStatusPending,
		SSRC:             ssrc,
		Mode:             mode,
		RangeStart:       rangeStart,
		RangeEnd:         rangeEnd,
		TargetHost:       deviceIP,
		TargetPort:       devicePort,
		streamRegistered: make(chan bool, 1), // 创建流注册通知 channel
	}

	s.mu.Lock()
	s.sessions[streamId] = session
	s.mu.Unlock()

	sdp := s.buildSDP(session)
	if err := s.sendInvite(session, sdp); err != nil {
		// 释放资源：RTP Server、SSRC、会话
		_, _ = s.zlm.CloseRtpServer(streamId)
		if s.ssrcService != nil && ssrc != "" {
			s.ssrcService.ReleaseSsrc(ssrc)
		}
		s.mu.Lock()
		delete(s.sessions, streamId)
		s.mu.Unlock()
		return nil, err
	}

	// 启动流注册超时检测
	go s.monitorStreamRegistration(streamId)

	// 统一记录会话已经进入待确认阶段。
	// 这里不能把“返回播放地址”误判为 RTP 已经成功，
	// 真正成功仍以后续的 on_publish / on_stream_changed / getMediaList 为准。
	session.Status = PlayStatusPlaying
	log.Info().Str("device_id", deviceId).Str("channel_id", channelId).Str("stream_id", streamId).Str("mode", string(mode)).Msg("开始播放")
	return s.buildPlayResult(session), nil
}

// isStreamActive 检查 ZLM 流是否真实存在
func (s *PlayService) isStreamActive(streamId string) bool {
	if s.zlm == nil {
		return false
	}

	// 使用 GetRtpInfo 检查 GB28181 RTP 流是否存在
	// 这是检测 GB28181 流的正确方式
	rtpInfo, err := s.zlm.GetRtpInfo(streamId)
	if err == nil && rtpInfo.Code == 0 && rtpInfo.Exist {
		return true
	}

	return false
}

// cleanupSession 清理旧会话资源
// 如果 ByeSent=false，先发送 BYE 再清理
func (s *PlayService) cleanupSession(streamId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return
	}

	// 如果 BYE 未发送，先发送 BYE 通知设备停止推流
	if !session.ByeSent {
		log.Info().Str("stream_id", streamId).Msg("旧会话 BYE 未发送，先发送 BYE 再清理")
		if err := s.sendBye(session); err != nil {
			log.Warn().Err(err).Str("stream_id", streamId).Msg("清理旧会话时发送 BYE 失败")
		}
		session.ByeSent = true
	}

	// 释放 SSRC
	if s.ssrcService != nil && session.SSRC != "" {
		s.ssrcService.ReleaseSsrc(session.SSRC)
	}

	// 关闭 RTP Server
	if s.zlm != nil {
		_, _ = s.zlm.CloseRtpServer(streamId)
	}

	// 删除会话
	delete(s.sessions, streamId)

	log.Info().Str("stream_id", streamId).Msg("清理旧会话完成")
}

// Stop 停止播放（用户主动停止）
func (s *PlayService) Stop(streamId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return fmt.Errorf("会话不存在: %s", streamId)
	}

	// 标记为用户主动停止，防止自动重连
	session.UserStopped = true
	session.RetryCount = 3 // 设置为最大重连次数，防止 CheckStreamHealth 触发重连

	// 如果 BYE 未发送，先发送 BYE
	if !session.ByeSent {
		if err := s.sendBye(session); err != nil {
			log.Warn().Err(err).Str("stream_id", streamId).Msg("发送 BYE 失败")
		}
		session.ByeSent = true
	}

	if _, err := s.zlm.CloseRtpServer(streamId); err != nil {
		log.Warn().Err(err).Str("stream_id", streamId).Msg("关闭 RTP 服务器失败")
	}

	// 释放 SSRC
	if s.ssrcService != nil && session.SSRC != "" {
		s.ssrcService.ReleaseSsrc(session.SSRC)
	}

	session.Status = PlayStatusStopped
	delete(s.sessions, streamId)

	log.Info().Str("stream_id", streamId).Str("mode", string(session.Mode)).Str("ssrc", session.SSRC).Bool("user_stopped", true).Msg("用户主动停止播放")
	return nil
}

// monitorStreamRegistration 监控流注册超时
// 如果 10 秒内未收到流注册通知，自动发送 BYE 清理会话
func (s *PlayService) monitorStreamRegistration(streamId string) {
	s.mu.RLock()
	session, exists := s.sessions[streamId]
	s.mu.RUnlock()

	if !exists || session.streamRegistered == nil {
		return
	}

	select {
	case <-session.streamRegistered:
		log.Info().Str("stream_id", streamId).Msg("流注册成功")
	case <-time.After(10 * time.Second):
		log.Warn().Str("stream_id", streamId).Msg("流注册超时，发送 BYE 清理会话")
		// 调用 Stop 清理会话，会发送 BYE、关闭 RTP Server、释放 SSRC
		if err := s.Stop(streamId); err != nil {
			log.Error().Err(err).Str("stream_id", streamId).Msg("清理超时会话失败")
		}
	}
}

// NotifyStreamRegistered 通知流注册成功
// 由 on_stream_changed hook 调用
func (s *PlayService) NotifyStreamRegistered(streamId string) {
	s.mu.Lock()
	session, exists := s.sessions[streamId]
	if exists {
		// 更新流最后活跃时间
		session.LastActiveTime = time.Now()
		if session.streamRegistered != nil {
			select {
			case session.streamRegistered <- true:
				log.Debug().Str("stream_id", streamId).Msg("已发送流注册通知")
			default:
				// channel 已满，说明通知已发送或超时检测已退出
				log.Debug().Str("stream_id", streamId).Msg("流注册通知 channel 已满")
			}
		}
	}
	s.mu.Unlock()
}

// UpdateStreamActiveTime 更新流最后活跃时间
// 由 on_stream_changed hook 在流持续活跃时调用
func (s *PlayService) UpdateStreamActiveTime(streamId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if exists {
		session.LastActiveTime = time.Now()
		log.Debug().Str("stream_id", streamId).Msg("更新流活跃时间")
	}
}

// ClearStreamActiveTime 清除流活跃时间
// 由 on_stream_changed hook 在流注销时调用，用于触发断流检测
func (s *PlayService) ClearStreamActiveTime(streamId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if exists {
		// 清除活跃时间，让 CheckStreamHealth 能检测到断流
		session.LastActiveTime = time.Time{}
		log.Info().Str("stream_id", streamId).Msg("清除流活跃时间，等待健康检查")
	}
}

// UpdateReaderCount 更新流观看人数
// 由 on_stream_changed hook 在流状态变化时调用
func (s *PlayService) UpdateReaderCount(streamId string, readerCount int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if exists {
		session.ReaderCount = readerCount
		log.Debug().Str("stream_id", streamId).Int("reader_count", readerCount).Msg("更新流观看人数")
	}
}

// OnStreamDisconnected 流断开通知
// 由 on_stream_changed hook 在收到 regist=false 且 alive_second > 10 时调用
// 等待3秒后检查流是否恢复，未恢复则触发重连
func (s *PlayService) OnStreamDisconnected(streamId string) {
	// 使用写锁来原子性地检查和设置标记
	s.mu.Lock()
	session, exists := s.sessions[streamId]
	if !exists {
		s.mu.Unlock()
		return
	}

	// 如果用户主动停止，不触发重连
	if session.UserStopped {
		s.mu.Unlock()
		log.Debug().Str("stream_id", streamId).Msg("用户主动停止，不触发断流检测")
		return
	}

	// 如果已经在重连中，跳过
	if session.IsReconnecting {
		s.mu.Unlock()
		log.Debug().Str("stream_id", streamId).Msg("已在重连中，跳过")
		return
	}

	// 如果已经在等待断流检测结果，跳过（防止重复启动检测）
	if session.DisconnectDetecting {
		s.mu.Unlock()
		log.Debug().Str("stream_id", streamId).Msg("已在等待断流检测结果，跳过重复触发")
		return
	}

	// 记录当前会话的 CallID（用于后续验证是否是同一个会话）
	originalCallID := session.CallID

	// 设置断流检测标记
	session.DisconnectDetecting = true
	s.mu.Unlock()

	log.Info().Str("stream_id", streamId).Str("call_id", originalCallID).Msg("收到断流通知，启动断流检测")

	// 异步启动断流检测流程
	go func() {
		// 等待3秒，看流是否恢复
		time.Sleep(3 * time.Second)

		// 再次检查流是否活跃
		if s.isStreamActive(streamId) {
			log.Info().Str("stream_id", streamId).Msg("流已恢复，取消重连")
			s.UpdateStreamActiveTime(streamId)
			// 清除断流检测标记（只有同一个会话才清除）
			s.mu.Lock()
			if session, exists := s.sessions[streamId]; exists && session.CallID == originalCallID {
				session.DisconnectDetecting = false
			}
			s.mu.Unlock()
			return
		}

		// 流未恢复，再次检查会话状态（可能已被其他流程处理）
		s.mu.Lock()
		session, exists = s.sessions[streamId]
		var deviceId, channelId string
		var mode PlayMode
		var retryCount int
		var isReconnecting bool
		var userStopped bool
		var currentCallID string
		if exists {
			currentCallID = session.CallID
			deviceId = session.DeviceId
			channelId = session.ChannelId
			mode = session.Mode
			retryCount = session.RetryCount
			isReconnecting = session.IsReconnecting
			userStopped = session.UserStopped
			// 清除断流检测标记（只有同一个会话才清除）
			if session.CallID == originalCallID {
				session.DisconnectDetecting = false
			}
		}
		s.mu.Unlock()

		if !exists {
			return
		}

		// 关键检查：验证是否是同一个会话
		// 如果会话已被重建（重连成功），CallID 会变化，此时不应触发重连
		if currentCallID != originalCallID {
			log.Debug().
				Str("stream_id", streamId).
				Str("original_call_id", originalCallID).
				Str("current_call_id", currentCallID).
				Msg("会话已重建，跳过旧会话的重连触发")
			return
		}

		// 如果用户主动停止或已在重连中，跳过
		if userStopped {
			log.Debug().Str("stream_id", streamId).Msg("用户主动停止，取消重连")
			return
		}
		if isReconnecting {
			log.Debug().Str("stream_id", streamId).Msg("已在重连中，取消重复触发")
			return
		}

		log.Warn().
			Str("stream_id", streamId).
			Int("retry_count", retryCount).
			Msg("流断开3秒未恢复，触发重连")

		// 启动重连流程
		s.startReconnectProcess(streamId, deviceId, channelId, mode, retryCount)
	}()
}

// CheckStreamHealth 检查所有会话的流健康状态
// 只更新活跃时间，断流检测由 on_stream_changed webhook 驱动
// 如果流长时间不存在（超过30秒），清理会话资源
func (s *PlayService) CheckStreamHealth(timeout time.Duration) {
	s.mu.RLock()
	// 复制会话列表，避免在检查期间持有锁
	sessionsCopy := make([]struct {
		streamId       string
		lastActiveTime time.Time
		startTime      time.Time
		status         string
		userStopped    bool
		retryCount     int
		isReconnecting bool
		deviceId       string
		channelId      string
		mode           PlayMode
	}, 0, len(s.sessions))

	for streamId, session := range s.sessions {
		sessionsCopy = append(sessionsCopy, struct {
			streamId       string
			lastActiveTime time.Time
			startTime      time.Time
			status         string
			userStopped    bool
			retryCount     int
			isReconnecting bool
			deviceId       string
			channelId      string
			mode           PlayMode
		}{
			streamId:       streamId,
			lastActiveTime: session.LastActiveTime,
			startTime:      session.StartTime,
			status:         session.Status,
			userStopped:    session.UserStopped,
			retryCount:     session.RetryCount,
			isReconnecting: session.IsReconnecting,
			deviceId:       session.DeviceId,
			channelId:      session.ChannelId,
			mode:           session.Mode,
		})
	}
	s.mu.RUnlock()

	now := time.Now()
	for _, item := range sessionsCopy {
		// 只检查 playing 状态的会话
		if item.status != PlayStatusPlaying {
			continue
		}

		// 如果用户主动停止，跳过健康检查
		if item.userStopped {
			continue
		}

		// 如果正在重连中，跳过（由重连逻辑处理）
		if item.isReconnecting {
			continue
		}

		// 通过 ZLM API 检查流是否真的存在
		if s.isStreamActive(item.streamId) {
			// 流存在，更新活跃时间
			s.UpdateStreamActiveTime(item.streamId)
			continue
		}

		// 流不存在，检查是否需要触发断流检测
		// 如果会话刚开始（10秒内），可能是正常的推流流程，跳过
		sessionAge := now.Sub(item.startTime)
		if sessionAge < 10*time.Second {
			continue
		}

		// 如果 LastActiveTime 为零值，说明流还未注册，跳过
		if item.lastActiveTime.IsZero() {
			continue
		}

		// 流不存在超过3秒，触发断流检测流程（作为 webhook 的兜底机制）
		inactiveDuration := now.Sub(item.lastActiveTime)
		reconnectTimeout := 3 * time.Second
		if inactiveDuration > reconnectTimeout {
			log.Warn().
				Str("stream_id", item.streamId).
				Dur("inactive_duration", inactiveDuration).
				Dur("reconnect_timeout", reconnectTimeout).
				Int("retry_count", item.retryCount).
				Msg("流不存在超过3秒，触发断流检测流程")

			// 触发断流检测流程（webhook 可能没有触发，这里作为兜底）
			go s.OnStreamDisconnected(item.streamId)
		}
	}
}

// CheckReconnectStatus 检查重连恢复状态
// 每秒调用一次，检查正在重连的会话是否已恢复
// 如果10秒内恢复，重置 RetryCount；否则继续下一次重连或清理
func (s *PlayService) CheckReconnectStatus() {
	s.mu.RLock()
	// 复制正在重连的会话列表
	reconnectingSessions := make([]struct {
		streamId       string
		deviceId       string
		channelId      string
		mode           PlayMode
		retryCount     int
		isReconnecting bool
		lastRetryTime  time.Time
		lastActiveTime time.Time
		status         string
		rangeStart     *time.Time
		rangeEnd       *time.Time
	}, 0, len(s.sessions))

	for streamId, session := range s.sessions {
		if session.IsReconnecting {
			reconnectingSessions = append(reconnectingSessions, struct {
				streamId       string
				deviceId       string
				channelId      string
				mode           PlayMode
				retryCount     int
				isReconnecting bool
				lastRetryTime  time.Time
				lastActiveTime time.Time
				status         string
				rangeStart     *time.Time
				rangeEnd       *time.Time
			}{
				streamId:       streamId,
				deviceId:       session.DeviceId,
				channelId:      session.ChannelId,
				mode:           session.Mode,
				retryCount:     session.RetryCount,
				isReconnecting: session.IsReconnecting,
				lastRetryTime:  session.LastRetryTime,
				lastActiveTime: session.LastActiveTime,
				status:         session.Status,
				rangeStart:     session.RangeStart,
				rangeEnd:       session.RangeEnd,
			})
		}
	}
	s.mu.RUnlock()

	now := time.Now()
	for _, item := range reconnectingSessions {
		// 检查流是否已恢复（LastActiveTime 在重连后有更新）
		s.mu.RLock()
		session, exists := s.sessions[item.streamId]
		s.mu.RUnlock()

		if !exists {
			// 会话已被删除（可能已成功重连并创建了新会话）
			log.Debug().Str("stream_id", item.streamId).Msg("重连会话已不存在，可能已成功重连")
			continue
		}

		// 检查是否恢复：流状态为 playing 且 LastActiveTime 在重连后有更新
		reconnectDuration := now.Sub(item.lastRetryTime)
		if session.Status == PlayStatusPlaying && !session.LastActiveTime.IsZero() && session.LastActiveTime.After(item.lastRetryTime) {
			// 流已恢复，重置重连状态
			s.mu.Lock()
			session.IsReconnecting = false
			session.RetryCount = 0              // 重置重连计数
			session.DisconnectDetecting = false // 清除断流检测标记
			s.mu.Unlock()
			log.Info().
				Str("stream_id", item.streamId).
				Dur("reconnect_duration", reconnectDuration).
				Msg("重连成功，流已恢复，重置重连计数")
			continue
		}

		// 检查是否超过10秒等待时间
		if reconnectDuration > 10*time.Second {
			log.Warn().
				Str("stream_id", item.streamId).
				Dur("reconnect_duration", reconnectDuration).
				Int("retry_count", item.retryCount).
				Msg("重连等待超时(10秒)，流未恢复")

			// 清理重连状态
			s.mu.Lock()
			session.IsReconnecting = false
			s.mu.Unlock()

			// 如果未达到最大重连次数，触发下一次重连
			if item.retryCount < 3 {
				log.Info().
					Str("stream_id", item.streamId).
					Int("retry_count", item.retryCount).
					Msg("触发下一次重连尝试")
				go s.startReconnectProcess(item.streamId, item.deviceId, item.channelId, item.mode, item.retryCount)
			} else {
				// 已达最大重连次数，清理会话
				log.Warn().
					Str("stream_id", item.streamId).
					Int("retry_count", item.retryCount).
					Msg("达到最大重连次数(3次)，清理会话")
				if err := s.StopWithBye(item.streamId); err != nil {
					log.Error().Err(err).Str("stream_id", item.streamId).Msg("清理重连失败会话出错")
				}
			}
		} else {
			// 还在等待中
			log.Debug().
				Str("stream_id", item.streamId).
				Dur("reconnect_duration", reconnectDuration).
				Dur("remaining", 10*time.Second-reconnectDuration).
				Msg("等待重连恢复中")
		}
	}
}

// StopWithBye 停止播放并发送 BYE（用于自动断流场景，如健康检查）
// 与 Stop 方法的区别：不设置 UserStopped，允许 CheckStreamHealth 判断是否需要重连
func (s *PlayService) StopWithBye(streamId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		log.Debug().Str("stream_id", streamId).Msg("会话不存在，跳过 BYE 发送")
		return nil
	}

	// 不设置 UserStopped（保持 false），不修改 RetryCount
	// 这样 CheckStreamHealth 可以判断是否需要重连

	// 发送 BYE 给设备
	if err := s.sendBye(session); err != nil {
		log.Warn().Err(err).Str("stream_id", streamId).Msg("发送 BYE 失败")
		// 即使 BYE 发送失败，也继续清理本地资源
	}

	// 关闭 RTP Server
	if s.zlm != nil {
		if _, err := s.zlm.CloseRtpServer(streamId); err != nil {
			log.Warn().Err(err).Str("stream_id", streamId).Msg("关闭 RTP 服务器失败")
		}
	}

	// 释放 SSRC
	if s.ssrcService != nil && session.SSRC != "" {
		s.ssrcService.ReleaseSsrc(session.SSRC)
	}

	session.Status = PlayStatusStopped
	delete(s.sessions, streamId)

	log.Info().Str("stream_id", streamId).Str("mode", string(session.Mode)).Str("ssrc", session.SSRC).Bool("user_stopped", session.UserStopped).Msg("自动断流，已发送 BYE 并清理会话")
	return nil
}

// ForceStop 强制停止播放（不触发重连，用于服务端关闭等场景）
func (s *PlayService) ForceStop(streamId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return fmt.Errorf("会话不存在: %s", streamId)
	}

	// 标记为用户停止，防止重连
	session.UserStopped = true
	session.RetryCount = 3

	// 发送 BYE
	if session.Status == PlayStatusPlaying && !session.ByeSent {
		if err := s.sendBye(session); err != nil {
			log.Warn().Err(err).Str("stream_id", streamId).Msg("强制停止时发送 BYE 失败")
		}
		session.ByeSent = true
	}

	// 关闭 RTP Server
	if s.zlm != nil {
		s.zlm.CloseRtpServer(streamId)
	}

	// 释放 SSRC
	if s.ssrcService != nil && session.SSRC != "" {
		s.ssrcService.ReleaseSsrc(session.SSRC)
	}

	session.Status = PlayStatusStopped
	delete(s.sessions, streamId)

	log.Info().Str("stream_id", streamId).Msg("强制停止播放")
	return nil
}

// GetSession 获取会话
func (s *PlayService) GetSession(streamId string) (*PlaySession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[streamId]
	return session, exists
}

// ListSessions 列出所有播放会话
func (s *PlayService) ListSessions() []*PlaySession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*PlaySession, 0, len(s.sessions))
	for _, session := range s.sessions {
		result = append(result, session)
	}
	return result
}

func (s *PlayService) generateStreamId(deviceId, channelId string, mode PlayMode, rangeStart, rangeEnd *time.Time) string {
	if mode == PlayModePlayback && rangeStart != nil && rangeEnd != nil {
		return fmt.Sprintf("%s_%s_%d_%d", deviceId, channelId, rangeStart.Unix(), rangeEnd.Unix())
	}
	return fmt.Sprintf("%s_%s", deviceId, channelId)
}

// buildSDP 构建 SDP
func (s *PlayService) buildSDP(session *PlaySession) string {
	title := "Play"
	start := int64(0)
	end := int64(0)
	if session.Mode == PlayModePlayback && session.RangeStart != nil && session.RangeEnd != nil {
		title = "Playback"
		start = session.RangeStart.Unix()
		end = session.RangeEnd.Unix()
	} else if session.Mode == PlayModeDownload && session.RangeStart != nil && session.RangeEnd != nil {
		title = "Download"
		start = session.RangeStart.Unix()
		end = session.RangeEnd.Unix()
	}

	// COMPAT_JAVA: o= 行使用 channelId 作为用户名（与 Java GB28181SDPBuilder 一致）
	originUser := session.ChannelId

	// 回放/下载模式需要 u= 行，实时播放不需要
	if session.Mode == PlayModePlayback || session.Mode == PlayModeDownload {
		// COMPAT_JAVA: rtpmap 顺序与 WVP 一致（96, 98, 97, 99），避免设备解析错误
		return fmt.Sprintf(`v=0
o=%s 0 0 IN IP4 %s
s=%s
u=%s:0
c=IN IP4 %s
t=%d %d
m=video %d RTP/AVP 96 97 98 99
a=recvonly
a=rtpmap:96 PS/90000
a=rtpmap:98 H264/90000
a=rtpmap:97 MPEG4/90000
a=rtpmap:99 H265/90000
y=%s
`, originUser, s.config.ZLMHost, title, session.ChannelId, s.config.ZLMHost, start, end, session.RTPPort, session.SSRC)
	}

	// 实时播放模式（无 u= 行）
	// COMPAT_JAVA: rtpmap 顺序与 WVP 一致（96, 98, 97, 99），避免设备解析错误
	// GB28181-2016: f= 行必须存在（可为空）
	return fmt.Sprintf(`v=0
o=%s 0 0 IN IP4 %s
s=%s
c=IN IP4 %s
t=%d %d
m=video %d RTP/AVP 96 97 98 99
a=recvonly
a=rtpmap:96 PS/90000
a=rtpmap:98 H264/90000
a=rtpmap:97 MPEG4/90000
a=rtpmap:99 H265/90000
y=%s
f=
`, originUser, s.config.ZLMHost, title, s.config.ZLMHost, start, end, session.RTPPort, session.SSRC)
}

// generateSSRC 生成 SSRC
func (s *PlayService) generateSSRC(streamId string, mode PlayMode) string {
	hash := 0
	for _, c := range streamId {
		hash = hash*31 + int(c)
	}
	prefix := 0
	if mode == PlayModePlayback {
		prefix = 1
	} else if mode == PlayModeDownload {
		prefix = 2
	}
	return fmt.Sprintf("%d%09d", prefix, (hash%1000000000+1000000000)%1000000000)
}

// buildPlayResult 构建播放结果
func (s *PlayService) buildPlayResult(session *PlaySession) *PlayResult {
	app := s.config.AppName
	host := s.config.ZLMHost
	rtspURL := fmt.Sprintf("rtsp://%s/%s/%s", host, app, session.StreamId)
	rtmpURL := fmt.Sprintf("rtmp://%s/%s/%s", host, app, session.StreamId)
	flvURL := fmt.Sprintf("http://%s:%d/%s/%s.live.flv", host, s.config.ZLMPort, app, session.StreamId)
	hlsURL := fmt.Sprintf("http://%s:%d/%s/%s/hls.m3u8", host, s.config.ZLMPort, app, session.StreamId)

	return &PlayResult{
		StreamId: session.StreamId,
		Urls:     []string{rtspURL, rtmpURL, flvURL, hlsURL},
		RTPPort:  session.RTPPort,
		FLVURL:   flvURL,
		HLSURL:   hlsURL,
		RTSPURL:  rtspURL,
		RTMPURL:  rtmpURL,
		Mode:     string(session.Mode),
	}
}

// sendInvite 发送 INVITE 并等待响应
func (s *PlayService) sendInvite(session *PlaySession, sdpContent string) error {
	deviceIP := session.TargetHost
	devicePort := session.TargetPort
	if deviceIP == "" || devicePort == 0 {
		return fmt.Errorf("设备 %s 缺少有效的 SIP 地址，无法发送 INVITE", session.DeviceId)
	}

	target := sip.Uri{User: session.ChannelId, Host: deviceIP, Port: devicePort}
	req := sip.NewRequest(sip.INVITE, target)
	req.SetTransport("UDP")

	// COMPAT_WVP: 手动构建所有必需的 headers 以完全匹配 WVP 格式
	// From header: 必须使用域编码，不能有显示名
	localId := s.config.LocalId
	if localId == "" {
		localId = session.DeviceId // 回退使用设备ID
	}
	domain := ""
	if len(localId) >= 10 {
		domain = localId[:10]
	} else {
		domain = "4405010000" // 默认域
	}
	fromUri := sip.Uri{User: localId, Host: domain}
	fromHeader := &sip.FromHeader{
		Address: fromUri,
		Params:  sip.NewParams(),
	}
	fromHeader.Params.Add("tag", utils.GenerateFromTag())
	req.AppendHeader(fromHeader)

	// To header: 必须包含设备端口
	toUri := sip.Uri{User: session.ChannelId, Host: deviceIP, Port: devicePort}
	toHeader := &sip.ToHeader{
		Address: toUri,
	}
	req.AppendHeader(toHeader)

	// 其他必需的 headers
	req.AppendHeader(sip.NewHeader("Max-Forwards", "70"))
	req.AppendHeader(sip.NewHeader("Content-Type", "APPLICATION/SDP"))
	req.AppendHeader(sip.NewHeader("Subject", fmt.Sprintf("%s:%s,%s:0", session.ChannelId, session.SSRC, localId)))

	// Contact header
	sipIP := s.config.SIPListenIP
	if sipIP == "" || sipIP == "0.0.0.0" {
		sipIP = s.config.ZLMHost
	}
	sipPort := s.config.SIPPort
	if sipPort == 0 {
		sipPort = 5060
	}
	req.AppendHeader(sip.NewHeader("Contact", fmt.Sprintf("<sip:%s@%s:%d>", localId, sipIP, sipPort)))
	req.AppendHeader(sip.NewHeader("User-Agent", "GB28181-Go-Server"))

	// COMPAT_WVP: Call-ID 格式必须是 xxx@IP，而不是 UUID 格式
	// WVP 格式: 952bf4e3f07baad92a73e273a81cc4bd@10.10.10.95
	callIDValue := fmt.Sprintf("%s@%s", utils.GenerateCallID(), sipIP)
	req.AppendHeader(sip.NewHeader("Call-ID", callIDValue))

	// COMPAT_WVP: Via header 必须包含 ;rport 参数（RFC 3581）
	// 不使用 ClientRequestAddVia（它会添加重复的 Via）
	// 让 sipgo 自动生成 Via，然后手动修改添加 ;rport
	req.SetBody([]byte(sdpContent))

	log.Info().
		Str("channel_id", session.ChannelId).
		Str("device_id", session.DeviceId).
		Str("device_ip", deviceIP).
		Int("device_port", devicePort).
		Int("rtp_port", session.RTPPort).
		Str("zlm_host", s.config.ZLMHost).
		Str("ssrc", session.SSRC).
		Str("sdp", sdpContent).
		Msg("发送 INVITE 请求（含完整 SDP）")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 使用 ClientRequestBuild 让 sipgo 自动生成 Via，但我们手动修改它
	tx, err := s.client.TransactionRequest(ctx, req, sipgo.ClientRequestBuild)
	if err == nil {
		// 成功发送后，检查并修改 Via header 添加 ;rport
		// 注意：此时请求已发送，这个修改不会影响已发送的请求
		// 但可以在日志中看到正确的格式
		if via := req.Via(); via != nil {
			viaStr := via.String()
			if !strings.Contains(viaStr, ";rport") {
				// 对于后续的请求，我们需要确保 Via 包含 ;rport
				log.Debug().Str("via", viaStr).Msg("Via header 缺少 ;rport 参数")
			}
		}
	}
	if err != nil {
		return fmt.Errorf("发送 INVITE 请求失败: %w", err)
	}
	// 关键修复：不要在这里 defer tx.Terminate()
	// INVITE 事务需要保持活跃以处理 200 OK 重传（RFC 3261 要求持续 64*T1 ≈ 32秒）
	// 事务将在发送 ACK 后延迟终止

	resp, err := s.getResponse(tx)
	if err != nil {
		tx.Terminate() // 仅在错误时终止事务
		return fmt.Errorf("等待 INVITE 响应失败: %w", err)
	}

	if resp.StatusCode != 200 {
		tx.Terminate() // 仅在错误时终止事务
		return fmt.Errorf("INVITE 失败: %d", resp.StatusCode)
	}

	// 获取响应的真实源地址（重要：某些设备 SDP 中的 IP 可能错误，应使用实际源地址）
	// 这是兼容性关键：resp.Source() 返回实际的 SIP 响应源地址
	respSource := resp.Source()
	if respSource != "" {
		// 解析源地址（格式：ip:port）
		sourceParts := strings.Split(respSource, ":")
		if len(sourceParts) >= 1 {
			realDeviceIP := sourceParts[0]
			log.Info().
				Str("real_ip", realDeviceIP).
				Str("stream_id", session.StreamId).
				Msg("使用 SIP 响应的真实源地址")
			// 更新会话的目标地址为真实地址
			session.TargetHost = realDeviceIP
			if len(sourceParts) >= 2 {
				if port, err := strconv.Atoi(sourceParts[1]); err == nil {
					session.TargetPort = port
				}
			}
		}
	}

	// COMPAT_WVP: 解析 Contact header 获取 ACK 目标地址（优先级高于响应源地址）
	// RFC 3261 Section 13.2.2.4: 如果 200 OK 包含 Contact header，ACK 应发送到 Contact 地址
	if contact := resp.Contact(); contact != nil {
		contactAddr := contact.Address
		if contactAddr.Host != "" {
			log.Info().
				Str("contact_host", contactAddr.Host).
				Int("contact_port", contactAddr.Port).
				Str("stream_id", session.StreamId).
				Msg("从 Contact header 获取设备地址")
			// 更新会话的 ACK 目标地址
			session.TargetHost = contactAddr.Host
			if contactAddr.Port != 0 {
				session.TargetPort = contactAddr.Port
			}
		}
	}

	s.mu.Lock()
	storedSession, exists := s.sessions[session.StreamId]
	if exists {
		storedSession.Status = PlayStatusPlaying
		storedSession.TargetHost = session.TargetHost
		storedSession.TargetPort = session.TargetPort
		if via := resp.Via(); via != nil {
			storedSession.Via = via.String()
		}
		if callID := resp.CallID(); callID != nil {
			storedSession.CallID = callID.Value()
		}
		if from := resp.From(); from != nil {
			storedSession.From = strings.TrimPrefix(from.String(), "From: ")
		}
		if to := resp.To(); to != nil {
			storedSession.To = strings.TrimPrefix(to.String(), "To: ")
		}
		if body := resp.Body(); len(body) > 0 {
			if ssrc := parseSSRCFromSDP(string(body)); ssrc != "" {
				storedSession.SSRC = ssrc
			}
		}
		if cseq := req.CSeq(); cseq != nil {
			storedSession.CSeq = int(cseq.SeqNo)
		}
		*session = *storedSession
	}
	s.mu.Unlock()

	if err := s.sendAck(session, req, resp); err != nil {
		log.Warn().Err(err).Str("stream_id", session.StreamId).Msg("发送 ACK 失败")
	}

	// RFC 3261: INVITE 事务需保持活跃以处理 200 OK 重传（64*T1 ≈ 32秒）
	// 延迟终止事务，确保重传的 200 OK 能被正确处理
	time.AfterFunc(32*time.Second, func() {
		tx.Terminate()
		log.Debug().Str("stream_id", session.StreamId).Msg("INVITE 事务已终止")
	})

	log.Info().Str("stream_id", session.StreamId).Msg("播放已建立")
	return nil
}

// getResponse 从事务获取最终响应（跳过 1xx 临时响应）
func (s *PlayService) getResponse(tx sip.ClientTransaction) (*sip.Response, error) {
	for {
		select {
		case <-tx.Done():
			return nil, fmt.Errorf("transaction timeout")
		case res := <-tx.Responses():
			// SIP 协议：1xx 为临时响应（100 Trying, 180 Ringing 等），需继续等待最终响应
			// 最终响应：2xx（成功）、3xx-6xx（失败）
			log.Info().
				Int("status_code", int(res.StatusCode)).
				Str("reason", res.Reason).
				Bool("is_provisional", res.StatusCode >= 100 && res.StatusCode < 200).
				Msg("收到 SIP 响应")
			if res.StatusCode >= 100 && res.StatusCode < 200 {
				log.Debug().
					Int("status_code", int(res.StatusCode)).
					Str("reason", res.Reason).
					Msg("收到 SIP 临时响应，继续等待最终响应")
				continue // 跳过临时响应，继续等待
			}
			// 返回最终响应（2xx-6xx）
			log.Info().Int("status_code", int(res.StatusCode)).Msg("收到最终响应，返回")
			return res, nil
		}
	}
}

// sendAck 发送 ACK
func (s *PlayService) sendAck(session *PlaySession, inviteReq *sip.Request, resp *sip.Response) error {
	deviceIP := session.TargetHost
	devicePort := session.TargetPort
	if deviceIP == "" || devicePort == 0 {
		if contact := resp.Contact(); contact != nil {
			if contact.Address.Host != "" {
				deviceIP = contact.Address.Host
			}
			if contact.Address.Port != 0 {
				devicePort = contact.Address.Port
			}
		}
	}
	if deviceIP == "" || devicePort == 0 {
		return fmt.Errorf("设备 %s 缺少有效的 ACK 目标地址", session.DeviceId)
	}

	ackUser := session.DeviceId
	if ackUser == "" {
		ackUser = session.ChannelId
	}
	ackTarget := sip.Uri{User: ackUser, Host: deviceIP, Port: devicePort}
	ack := sip.NewRequest(sip.ACK, ackTarget)
	ack.SetTransport("UDP")

	// 关键修复：删除 sipgo 自动添加的 headers，避免重复
	// RFC 3261 Section 17.1.1.3: ACK 必须使用与 INVITE 相同的 Call-ID、From（含 tag）、To（含设备返回的 tag）
	// 我们将完全手动构建所有必需的 headers
	ack.RemoveHeader("Via")
	ack.RemoveHeader("From")
	ack.RemoveHeader("To")
	ack.RemoveHeader("Call-ID")
	ack.RemoveHeader("CSeq")
	ack.RemoveHeader("Max-Forwards")
	ack.RemoveHeader("Content-Length")

	// 手动添加 Via header（使用新的 branch，ACK 可以使用新的 Via）
	// 关键修复：检查 SIPListenIP 是否为无效地址 0.0.0.0
	sipIP := s.config.SIPListenIP
	if sipIP == "" || sipIP == "0.0.0.0" {
		sipIP = s.config.ZLMHost // 回退使用 ZLM IP（与 INVITE Contact header 一致）
	}
	sipPort := s.config.SIPPort
	if sipPort == 0 {
		sipPort = 5060
	}
	newVia := fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=%s;rport",
		sipIP, sipPort, utils.GenerateViaTag())
	ack.AppendHeader(sip.NewHeader("Via", newVia))

	// 复制 INVITE 的 Call-ID（必须完全匹配）
	if callID := inviteReq.CallID(); callID != nil {
		ack.AppendHeader(callID)
	}

	// 复制 INVITE 的 From header（必须包含 tag）
	if from := inviteReq.From(); from != nil {
		ack.AppendHeader(from)
	}

	// 从 200 OK 响应的 To header 复制（必须包含设备添加的 tag）
	if to := resp.To(); to != nil {
		ack.AppendHeader(to)
	}

	// 设置 CSeq，seq number 与 INVITE 相同，但 method 改为 ACK
	if cseq := inviteReq.CSeq(); cseq != nil {
		ack.AppendHeader(sip.NewHeader("CSeq", fmt.Sprintf("%d ACK", cseq.SeqNo)))
	}

	// Max-Forwards 限制跳数
	ack.AppendHeader(sip.NewHeader("Max-Forwards", "70"))

	if err := s.client.WriteRequest(ack); err != nil {
		return fmt.Errorf("发送 ACK 失败: %w", err)
	}

	log.Info().
		Str("stream_id", session.StreamId).
		Str("target_user", ackUser).
		Str("target_host", deviceIP).
		Int("target_port", devicePort).
		Msg("ACK 已发送")
	return nil
}

// sendBye 发送 BYE
func (s *PlayService) sendBye(session *PlaySession) error {
	deviceIP := session.TargetHost
	devicePort := session.TargetPort
	if (deviceIP == "" || devicePort == 0) && s.deviceService != nil {
		device, err := s.deviceService.GetDevice(session.DeviceId)
		if err == nil && device.IP != "" && device.Port > 0 {
			deviceIP = device.IP
			devicePort = device.Port
		}
	}
	if deviceIP == "" || devicePort == 0 {
		return fmt.Errorf("设备 %s 缺少有效的 SIP 地址，无法发送 BYE", session.DeviceId)
	}

	target := sip.Uri{User: session.ChannelId, Host: deviceIP, Port: devicePort}
	req := sip.NewRequest(sip.BYE, target)
	req.SetTransport("UDP")

	// RFC 3261: Via header由sipgo库自动生成，包含唯一branch参数
	if session.CallID != "" {
		req.AppendHeader(sip.NewHeader("Call-ID", session.CallID))
	}
	if session.From != "" {
		req.AppendHeader(sip.NewHeader("From", session.From))
	}
	if session.To != "" {
		req.AppendHeader(sip.NewHeader("To", session.To))
	}
	req.AppendHeader(sip.NewHeader("CSeq", fmt.Sprintf("%d BYE", session.CSeq+1)))
	// RFC 3261: BYE 必须包含 Max-Forwards header
	req.AppendHeader(sip.NewHeader("Max-Forwards", "70"))

	if err := s.client.WriteRequest(req); err != nil {
		return fmt.Errorf("发送 BYE 请求失败: %w", err)
	}

	log.Debug().Str("device_id", session.DeviceId).Str("channel_id", session.ChannelId).Str("stream_id", session.StreamId).Msg("BYE 请求已发送")
	return nil
}

// parseSSRCFromSDP 从 SDP 解析 SSRC
// 兼容 LF 和 CRLF 换行符
func parseSSRCFromSDP(sdp string) string {
	// 统一换行符：将所有 CRLF 替换为 LF，然后分割
	normalizedSDP := strings.ReplaceAll(sdp, "\r\n", "\n")
	lines := strings.Split(normalizedSDP, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "y=") {
			// 提取 y= 后的内容并去除前后空格
			ssrc := strings.TrimPrefix(line, "y=")
			return strings.TrimSpace(ssrc)
		}
	}
	return ""
}

// OnInviteResponse 处理 INVITE 响应 (用于异步响应处理，现已改为同步)
func (s *PlayService) OnInviteResponse(streamId string, resp *sip.Response) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return
	}

	if resp.StatusCode == 200 {
		log.Debug().Str("stream_id", streamId).Msg("收到 INVITE 200 OK")
		return
	}

	session.Status = PlayStatusStopped
	_, _ = s.zlm.CloseRtpServer(streamId)
	delete(s.sessions, streamId)
	log.Warn().Str("stream_id", streamId).Int("status", int(resp.StatusCode)).Msg("播放失败")
}

// StopAllSessions 停止所有活跃会话（服务端优雅关闭）
// 遍历所有活跃会话，发送 BYE 请求，关闭 RTP Server，释放 SSRC
func (s *PlayService) StopAllSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.sessions) == 0 {
		log.Info().Msg("没有活跃的播放会话需要关闭")
		return
	}

	log.Info().Int("session_count", len(s.sessions)).Msg("正在关闭所有播放会话")

	// 遍历所有活跃会话
	for streamId, session := range s.sessions {
		// 发送 BYE 请求
		if session.Status == PlayStatusPlaying {
			if err := s.sendBye(session); err != nil {
				log.Warn().Err(err).Str("stream_id", streamId).Str("device_id", session.DeviceId).Msg("发送 BYE 失败")
			} else {
				log.Debug().Str("stream_id", streamId).Str("device_id", session.DeviceId).Msg("BYE 已发送")
			}
		}

		// 关闭 RTP Server
		if s.zlm != nil {
			if _, err := s.zlm.CloseRtpServer(streamId); err != nil {
				log.Warn().Err(err).Str("stream_id", streamId).Msg("关闭 RTP Server 失败")
			}
		}

		// 释放 SSRC
		if s.ssrcService != nil && session.SSRC != "" {
			s.ssrcService.ReleaseSsrc(session.SSRC)
		}

		// 标记会话状态为已停止
		session.Status = PlayStatusStopped

		log.Info().Str("stream_id", streamId).Str("mode", string(session.Mode)).Msg("会话已关闭")
	}

	// 清空 sessions map
	s.sessions = make(map[string]*PlaySession)

	log.Info().Msg("所有播放会话已关闭")
}

// CleanupStaleSessions 清理过期会话
func (s *PlayService) CleanupStaleSessions(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for streamId, session := range s.sessions {
		if now.Sub(session.StartTime) > timeout {
			// 释放 SSRC
			if s.ssrcService != nil && session.SSRC != "" {
				s.ssrcService.ReleaseSsrc(session.SSRC)
			}
			// 关闭 RTP Server
			_, _ = s.zlm.CloseRtpServer(streamId)
			// 删除会话
			delete(s.sessions, streamId)
			log.Info().Str("stream_id", streamId).Msg("清理过期会话")
		}
	}
}

// GetMediaInfo 获取媒体信息
func (s *PlayService) GetMediaInfo(streamId string) (*types.GetMediaListResp, error) {
	return s.zlm.GetMediaList(s.config.AppName, streamId)
}

// GetSessionByStreamId 根据 StreamId 查找会话
func (s *PlayService) GetSessionByStreamId(streamId string) *PlaySession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[streamId]
	if !exists {
		return nil
	}
	return session
}

// GetSessionByDeviceId 根据设备ID和播放模式查找活跃会话
// 用于设备单流限制与流复用：同一设备只能有一路实时流
func (s *PlayService) GetSessionByDeviceId(deviceId string, mode PlayMode) *PlaySession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, session := range s.sessions {
		// 匹配设备ID和播放模式，且会话状态为 playing
		if session.DeviceId == deviceId && session.Mode == mode && session.Status == PlayStatusPlaying {
			return session
		}
	}
	return nil
}

// RemoveSessionByStreamID 按 StreamID 移除会话
// ZLM 的 on_stream_changed / on_rtp_server_timeout 回调天然以 stream 作为主键，
// 因此这里必须支持按 stream 直接清理，避免用 Call-ID 删除失败后留下脏会话。
func (s *PlayService) RemoveSessionByStreamID(streamID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[streamID]; exists {
		delete(s.sessions, streamID)
		log.Info().Str("stream_id", streamID).Msg("会话已按 StreamID 移除")
		return
	}

	log.Debug().Str("stream_id", streamID).Msg("按 StreamID 移除会话时未找到对应会话")
}

// RemoveSession 移除会话
func (s *PlayService) RemoveSession(callId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 根据 CallID 查找会话
	for streamId, session := range s.sessions {
		if session.CallID == callId {
			delete(s.sessions, streamId)
			log.Info().Str("stream_id", streamId).Str("call_id", callId).Msg("会话已移除")
			return
		}
	}

	log.Debug().Str("call_id", callId).Msg("按 CallID 移除会话时未找到对应会话")
}

// OnMediaStatusReceived 处理媒体状态通知（录像结束等）
func (s *PlayService) OnMediaStatusReceived(streamId string) error {
	// 根据 StreamId 找到会话
	session := s.GetSessionByStreamId(streamId)
	if session == nil {
		log.Warn().Str("stream_id", streamId).Msg("未找到对应的播放会话")
		return nil
	}

	// 关闭 RTP Server
	_, err := s.zlm.CloseRtpServer(streamId)
	if err != nil {
		log.Error().Err(err).Str("stream_id", streamId).Msg("关闭 RTP Server 失败")
	}

	// 清理会话
	s.mu.Lock()
	delete(s.sessions, streamId)
	s.mu.Unlock()

	log.Info().Str("device_id", session.DeviceId).Str("stream_id", streamId).Msg("录像结束，已清理会话")
	return nil
}

// ReconnectSession 手动触发会话重连（供 HTTP API 调用）
// 返回新的流 ID 和播放结果
func (s *PlayService) ReconnectSession(streamId string) (*PlayResult, error) {
	s.mu.RLock()
	session, exists := s.sessions[streamId]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("会话不存在: %s", streamId)
	}

	// 检查是否可以重连
	if session.UserStopped {
		return nil, fmt.Errorf("用户已主动停止，无法重连")
	}
	if session.IsReconnecting {
		return nil, fmt.Errorf("正在重连中，请等待")
	}
	if session.RetryCount >= 3 {
		return nil, fmt.Errorf("已达到最大重连次数(3次)，无法重连")
	}

	log.Info().
		Str("stream_id", streamId).
		Str("device_id", session.DeviceId).
		Str("channel_id", session.ChannelId).
		Int("retry_count", session.RetryCount).
		Msg("手动触发重连")

	// 直接调用内部重连流程
	s.startReconnectProcess(streamId, session.DeviceId, session.ChannelId, session.Mode, session.RetryCount)

	// 查找新会话
	s.mu.RLock()
	var newSession *PlaySession
	for _, ns := range s.sessions {
		if ns.DeviceId == session.DeviceId && ns.ChannelId == session.ChannelId && ns.Mode == session.Mode {
			newSession = ns
			break
		}
	}
	s.mu.RUnlock()

	if newSession == nil {
		return nil, fmt.Errorf("重连失败：未找到新会话")
	}

	return s.buildPlayResult(newSession), nil
}

// startReconnectProcess 启动重连流程
// 检查重连条件，清理旧会话，重新建立流
func (s *PlayService) startReconnectProcess(streamId, deviceId, channelId string, mode PlayMode, currentRetryCount int) {
	// 检查重连次数限制
	if currentRetryCount >= 3 {
		log.Warn().
			Str("stream_id", streamId).
			Int("retry_count", currentRetryCount).
			Msg("超过最大重连次数(3次)，清理会话")
		// 超过重连次数，清理会话
		if err := s.StopWithBye(streamId); err != nil {
			log.Error().Err(err).Str("stream_id", streamId).Msg("清理超时会话失败")
		}
		return
	}

	s.mu.Lock()
	session, exists := s.sessions[streamId]
	if !exists {
		s.mu.Unlock()
		log.Warn().Str("stream_id", streamId).Msg("重连时会话已不存在")
		return
	}

	// 再次检查条件（防止并发问题）
	if session.UserStopped || session.RetryCount >= 3 || session.IsReconnecting {
		s.mu.Unlock()
		log.Debug().
			Str("stream_id", streamId).
			Bool("user_stopped", session.UserStopped).
			Int("retry_count", session.RetryCount).
			Bool("is_reconnecting", session.IsReconnecting).
			Msg("重连条件不满足，放弃重连")
		return
	}

	// 获取回放时间范围（用于重连回放流）
	rangeStart := session.RangeStart
	rangeEnd := session.RangeEnd

	// 更新重连状态
	session.IsReconnecting = true
	session.RetryCount++
	session.LastRetryTime = time.Now()
	s.mu.Unlock()

	log.Info().
		Str("stream_id", streamId).
		Str("device_id", deviceId).
		Str("channel_id", channelId).
		Int("retry_count", session.RetryCount).
		Str("mode", string(mode)).
		Msg("开始自动重连")

	// 清理旧会话的资源（发送 BYE、关闭 RTP、释放 SSRC）
	s.cleanupForReconnect(streamId, session)

	// 删除旧会话记录
	s.mu.Lock()
	delete(s.sessions, streamId)
	s.mu.Unlock()

	// 等待2秒后重新INVITE（给设备准备时间）
	log.Debug().Str("stream_id", streamId).Msg("等待2秒后重新INVITE")
	time.Sleep(2 * time.Second)

	// 重新建立流
	var err error
	switch mode {
	case PlayModeLive:
		_, err = s.Play(deviceId, channelId)
	case PlayModePlayback:
		if rangeStart != nil && rangeEnd != nil {
			_, err = s.PlayBack(deviceId, channelId, *rangeStart, *rangeEnd)
		} else {
			err = fmt.Errorf("回放模式缺少时间范围参数")
		}
	case PlayModeDownload:
		if rangeStart != nil && rangeEnd != nil {
			_, err = s.Download(deviceId, channelId, *rangeStart, *rangeEnd, 1)
		} else {
			err = fmt.Errorf("下载模式缺少时间范围参数")
		}
	default:
		err = fmt.Errorf("未知的播放模式: %s", mode)
	}

	// 更新新会话的重连状态
	s.mu.Lock()
	for newStreamId, newSession := range s.sessions {
		if newSession.DeviceId == deviceId && newSession.ChannelId == channelId && newSession.Mode == mode {
			// 继承重连计数
			newSession.RetryCount = session.RetryCount
			newSession.IsReconnecting = false
			newSession.DisconnectDetecting = false // 确保新会话的断流检测标记清除
			log.Info().
				Str("old_stream_id", streamId).
				Str("new_stream_id", newStreamId).
				Int("retry_count", newSession.RetryCount).
				Bool("success", err == nil).
				Msg("自动重连完成")
			break
		}
	}
	s.mu.Unlock()

	if err != nil {
		log.Error().Err(err).
			Str("stream_id", streamId).
			Str("device_id", deviceId).
			Str("channel_id", channelId).
			Int("retry_count", session.RetryCount).
			Msg("自动重连失败")
	}
}

// cleanupForReconnect 为重连清理旧会话资源
// 只清理资源，不删除会话记录
func (s *PlayService) cleanupForReconnect(streamId string, session *PlaySession) {
	// 发送 BYE（如果尚未发送）
	if !session.ByeSent {
		if err := s.sendBye(session); err != nil {
			log.Warn().Err(err).Str("stream_id", streamId).Msg("重连前发送 BYE 失败")
		}
		session.ByeSent = true
	}

	// 关闭 RTP Server
	if s.zlm != nil {
		if _, err := s.zlm.CloseRtpServer(streamId); err != nil {
			log.Warn().Err(err).Str("stream_id", streamId).Msg("重连前关闭 RTP Server 失败")
		}
	}

	// 释放 SSRC
	if s.ssrcService != nil && session.SSRC != "" {
		s.ssrcService.ReleaseSsrc(session.SSRC)
	}

	log.Debug().Str("stream_id", streamId).Msg("重连资源清理完成")
}
