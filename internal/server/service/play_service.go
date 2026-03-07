package service

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit/types"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// PlayService 视频点播服务
type PlayService struct {
	client   *sipgo.Client
	zlm      *zlmediakit.ZLMediaKit
	config   PlayConfig
	sessions map[string]*PlaySession
	mu       sync.RWMutex
}

// PlayConfig 播放配置
type PlayConfig struct {
	ZLMHost string // ZLMediaKit 主机 IP
	ZLMPort int    // ZLMediaKit HTTP 端口
	AppName string // 应用名, 默认 "rtp"
}

// PlaySession 播放会话
type PlaySession struct {
	StreamId  string    // 流 ID
	DeviceId  string    // 设备 ID
	ChannelId string    // 通道 ID
	RTPPort   int       // RTP 端口
	StartTime time.Time // 开始时间
	Status    string    // 状态: pending, playing, stopped
	SSRC      string    // SSRC
}

// PlayResult 播放结果
type PlayResult struct {
	StreamId string   `json:"stream_id"`
	Urls     []string `json:"urls"` // 播放地址列表
	RTPPort  int      `json:"rtp_port"`
}

// 播放状态
const (
	PlayStatusPending = "pending"
	PlayStatusPlaying = "playing"
	PlayStatusStopped = "stopped"
)

// NewPlayService 创建播放服务
func NewPlayService(client *sipgo.Client, zlm *zlmediakit.ZLMediaKit, config PlayConfig) *PlayService {
	if config.AppName == "" {
		config.AppName = "rtp"
	}
	return &PlayService{
		client:   client,
		zlm:      zlm,
		config:   config,
		sessions: make(map[string]*PlaySession),
	}
}

// Play 开始播放
func (s *PlayService) Play(deviceId, channelId string) (*PlayResult, error) {
	// 生成流 ID
	streamId := s.generateStreamId(deviceId, channelId)

	// 检查是否已有会话
	s.mu.RLock()
	if session, exists := s.sessions[streamId]; exists && session.Status == PlayStatusPlaying {
		s.mu.RUnlock()
		// 返回已存在的播放地址
		return s.buildPlayResult(session), nil
	}
	s.mu.RUnlock()

	// 打开 RTP 服务器
	rtpResp, err := s.zlm.OpenRtpServer(streamId, 0)
	if err != nil {
		log.Error().Err(err).Str("stream_id", streamId).Msg("打开 RTP 服务器失败")
		return nil, fmt.Errorf("打开 RTP 服务器失败: %w", err)
	}

	// 创建会话
	session := &PlaySession{
		StreamId:  streamId,
		DeviceId:  deviceId,
		ChannelId: channelId,
		RTPPort:   rtpResp.Port,
		StartTime: time.Now(),
		Status:    PlayStatusPending,
	}

	s.mu.Lock()
	s.sessions[streamId] = session
	s.mu.Unlock()

	// 构建 SDP
	sdp := s.buildSDP(streamId, rtpResp.Port)

	// 发送 INVITE
	if err := s.sendInvite(deviceId, channelId, streamId, sdp); err != nil {
		// 清理会话
		s.zlm.CloseRtpServer(streamId)
		s.mu.Lock()
		delete(s.sessions, streamId)
		s.mu.Unlock()
		return nil, err
	}

	// 更新状态
	session.Status = PlayStatusPlaying

	log.Info().Str("device_id", deviceId).Str("channel_id", channelId).Str("stream_id", streamId).Msg("开始播放")

	return s.buildPlayResult(session), nil
}

// Stop 停止播放
func (s *PlayService) Stop(streamId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return fmt.Errorf("会话不存在: %s", streamId)
	}

	// 发送 BYE
	if err := s.sendBye(session.DeviceId, session.ChannelId, streamId); err != nil {
		log.Warn().Err(err).Str("stream_id", streamId).Msg("发送 BYE 失败")
	}

	// 关闭 RTP 服务器
	if _, err := s.zlm.CloseRtpServer(streamId); err != nil {
		log.Warn().Err(err).Str("stream_id", streamId).Msg("关闭 RTP 服务器失败")
	}

	// 更新状态
	session.Status = PlayStatusStopped
	delete(s.sessions, streamId)

	log.Info().Str("stream_id", streamId).Msg("停止播放")
	return nil
}

// GetSession 获取会话
func (s *PlayService) GetSession(streamId string) (*PlaySession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[streamId]
	return session, exists
}

// ListSessions 列出所有会话
func (s *PlayService) ListSessions() []*PlaySession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*PlaySession, 0, len(s.sessions))
	for _, session := range s.sessions {
		result = append(result, session)
	}
	return result
}

// generateStreamId 生成流 ID
func (s *PlayService) generateStreamId(deviceId, channelId string) string {
	return fmt.Sprintf("%s_%s", deviceId, channelId)
}

// buildSDP 构建 SDP
func (s *PlayService) buildSDP(streamId string, rtpPort int) string {
	// GB28181 视频点播 SDP
	sdp := fmt.Sprintf(`v=0
o=- 0 0 IN IP4 %s
s=Play
c=IN IP4 %s
t=0 0
m=video %d RTP/AVP 96
a=rtpmap:96 PS/90000
a=recvonly
y=%s
`, s.config.ZLMHost, s.config.ZLMHost, rtpPort, s.generateSSRC(streamId))

	return sdp
}

// generateSSRC 生成 SSRC
func (s *PlayService) generateSSRC(streamId string) string {
	// 使用 streamId 的哈希生成 10 位 SSRC
	hash := 0
	for _, c := range streamId {
		hash = hash*31 + int(c)
	}
	return fmt.Sprintf("%010d", (hash%10000000000+10000000000)%10000000000)
}

// buildPlayResult 构建播放结果
func (s *PlayService) buildPlayResult(session *PlaySession) *PlayResult {
	app := s.config.AppName
	host := s.config.ZLMHost

	urls := []string{
		fmt.Sprintf("rtsp://%s/%s/%s", host, app, session.StreamId),
		fmt.Sprintf("rtmp://%s/%s/%s", host, app, session.StreamId),
		fmt.Sprintf("http://%s:%d/%s/%s.live.flv", host, s.config.ZLMPort, app, session.StreamId),
		fmt.Sprintf("http://%s:%d/%s/%s/hls.m3u8", host, s.config.ZLMPort, app, session.StreamId),
	}

	return &PlayResult{
		StreamId: session.StreamId,
		Urls:     urls,
		RTPPort:  session.RTPPort,
	}
}

// sendInvite 发送 INVITE
func (s *PlayService) sendInvite(deviceId, channelId, streamId, sdpContent string) error {
	// 构建目标 URI
	target := sip.Uri{
		User: deviceId,
		Host: s.config.ZLMHost, // 实际应为设备 IP
	}

	// 创建 INVITE 请求
	req := sip.NewRequest(sip.INVITE, target)
	req.SetTransport("UDP")
	req.AppendHeader(sip.NewHeader("Content-Type", "application/sdp"))
	req.AppendHeader(sip.NewHeader("Subject", fmt.Sprintf("%s:%s,%s:0", channelId, streamId, deviceId)))
	req.SetBody([]byte(sdpContent))

	// 构建请求头
	if err := sipgo.ClientRequestBuild(s.client, req); err != nil {
		return fmt.Errorf("构建 INVITE 请求失败: %w", err)
	}

	// 发送请求
	if err := s.client.WriteRequest(req); err != nil {
		return fmt.Errorf("发送 INVITE 请求失败: %w", err)
	}

	log.Debug().Str("device_id", deviceId).Str("channel_id", channelId).Msg("INVITE 请求已发送")
	return nil
}

// sendBye 发送 BYE
func (s *PlayService) sendBye(deviceId, channelId, streamId string) error {
	// 构建目标 URI
	target := sip.Uri{
		User: deviceId,
		Host: s.config.ZLMHost,
	}

	// 创建 BYE 请求
	req := sip.NewRequest(sip.BYE, target)
	req.SetTransport("UDP")

	// 构建请求头
	if err := sipgo.ClientRequestBuild(s.client, req); err != nil {
		return fmt.Errorf("构建 BYE 请求失败: %w", err)
	}

	// 发送请求
	if err := s.client.WriteRequest(req); err != nil {
		return fmt.Errorf("发送 BYE 请求失败: %w", err)
	}

	log.Debug().Str("device_id", deviceId).Str("stream_id", streamId).Msg("BYE 请求已发送")
	return nil
}

// parseSSRCFromSDP 从 SDP 解析 SSRC
func parseSSRCFromSDP(sdp string) string {
	lines := strings.Split(sdp, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "y=") {
			return strings.TrimPrefix(line, "y=")
		}
	}
	return ""
}

// OnInviteResponse 处理 INVITE 响应
func (s *PlayService) OnInviteResponse(streamId string, resp *sip.Response) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return
	}

	if int(resp.StatusCode) == 200 {
		// 成功, 更新状态
		session.Status = PlayStatusPlaying

		// 从响应 SDP 中解析 SSRC
		if body := resp.Body(); len(body) > 0 {
			session.SSRC = parseSSRCFromSDP(string(body))
		}

		// 发送 ACK
		// TODO: 实现 ACK 发送

		log.Info().Str("stream_id", streamId).Msg("播放已建立")
	} else {
		// 失败, 清理会话
		session.Status = PlayStatusStopped
		s.zlm.CloseRtpServer(streamId)
		delete(s.sessions, streamId)

		log.Warn().Str("stream_id", streamId).Int("status", int(resp.StatusCode)).Msg("播放失败")
	}
}

// CleanupStaleSessions 清理过期会话
func (s *PlayService) CleanupStaleSessions(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for streamId, session := range s.sessions {
		if now.Sub(session.StartTime) > timeout {
			s.zlm.CloseRtpServer(streamId)
			delete(s.sessions, streamId)
			log.Info().Str("stream_id", streamId).Msg("清理过期会话")
		}
	}
}

// GetMediaInfo 获取媒体信息
func (s *PlayService) GetMediaInfo(streamId string) (*types.GetMediaListResp, error) {
	return s.zlm.GetMediaList(s.config.AppName)
}
