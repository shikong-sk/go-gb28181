package service

import (
	"context"
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
	client        *sipgo.Client
	zlm           *zlmediakit.ZLMediaKit
	config        PlayConfig
	deviceService *DeviceService // 设备服务，用于获取设备真实 IP
	sessions      map[string]*PlaySession
	mu            sync.RWMutex
}

// PlayConfig 播放配置
type PlayConfig struct {
	ZLMHost string // ZLMediaKit 主机 IP
	ZLMPort int    // ZLMediaKit HTTP 端口
	AppName string // 应用名, 默认 "rtp"
}

// PlayMode 播放模式
type PlayMode string

const (
	PlayModeLive     PlayMode = "live"
	PlayModePlayback PlayMode = "playback"
)

// PlaySession 播放会话
type PlaySession struct {
	StreamId   string     // 流 ID
	DeviceId   string     // 设备 ID
	ChannelId  string     // 通道 ID
	RTPPort    int        // RTP 端口
	StartTime  time.Time  // 会话创建时间
	Status     string     // 状态: pending, playing, stopped
	SSRC       string     // SSRC
	Mode       PlayMode   // 播放模式
	RangeStart *time.Time // 回放开始时间
	RangeEnd   *time.Time // 回放结束时间
	Via        string     // Via header value
	CallID     string     // Call-ID
	From       string     // From header
	To         string     // To header (from 200 OK response)
	CSeq       int        // CSeq number
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
func NewPlayService(client *sipgo.Client, zlm *zlmediakit.ZLMediaKit, config PlayConfig, deviceService *DeviceService) *PlayService {
	if config.AppName == "" {
		config.AppName = "rtp"
	}
	return &PlayService{
		client:        client,
		zlm:           zlm,
		config:        config,
		deviceService: deviceService,
		sessions:      make(map[string]*PlaySession),
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

func (s *PlayService) startPlay(deviceId, channelId string, mode PlayMode, rangeStart, rangeEnd *time.Time) (*PlayResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("SIP 客户端未初始化")
	}
	if s.zlm == nil {
		return nil, fmt.Errorf("ZLMediaKit 未初始化")
	}

	streamId := s.generateStreamId(deviceId, channelId, mode, rangeStart, rangeEnd)

	s.mu.RLock()
	if session, exists := s.sessions[streamId]; exists && session.Status == PlayStatusPlaying {
		s.mu.RUnlock()
		return s.buildPlayResult(session), nil
	}
	s.mu.RUnlock()

	rtpResp, err := s.zlm.OpenRtpServer(streamId, 0)
	if err != nil {
		log.Error().Err(err).Str("stream_id", streamId).Msg("打开 RTP 服务器失败")
		return nil, fmt.Errorf("打开 RTP 服务器失败: %w", err)
	}

	session := &PlaySession{
		StreamId:   streamId,
		DeviceId:   deviceId,
		ChannelId:  channelId,
		RTPPort:    rtpResp.Port,
		StartTime:  time.Now(),
		Status:     PlayStatusPending,
		SSRC:       s.generateSSRC(streamId, mode),
		Mode:       mode,
		RangeStart: rangeStart,
		RangeEnd:   rangeEnd,
	}

	s.mu.Lock()
	s.sessions[streamId] = session
	s.mu.Unlock()

	sdp := s.buildSDP(session)
	if err := s.sendInvite(session, sdp); err != nil {
		_, _ = s.zlm.CloseRtpServer(streamId)
		s.mu.Lock()
		delete(s.sessions, streamId)
		s.mu.Unlock()
		return nil, err
	}

	session.Status = PlayStatusPlaying
	log.Info().Str("device_id", deviceId).Str("channel_id", channelId).Str("stream_id", streamId).Str("mode", string(mode)).Msg("开始播放")
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

	if err := s.sendBye(session); err != nil {
		log.Warn().Err(err).Str("stream_id", streamId).Msg("发送 BYE 失败")
	}

	if _, err := s.zlm.CloseRtpServer(streamId); err != nil {
		log.Warn().Err(err).Str("stream_id", streamId).Msg("关闭 RTP 服务器失败")
	}

	session.Status = PlayStatusStopped
	delete(s.sessions, streamId)

	log.Info().Str("stream_id", streamId).Str("mode", string(session.Mode)).Msg("停止播放")
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
	}

	return fmt.Sprintf(`v=0
o=- 0 0 IN IP4 %s
s=%s
u=%s:0
c=IN IP4 %s
t=%d %d
m=video %d RTP/AVP 96
a=recvonly
a=rtpmap:96 PS/90000
y=%s
`, s.config.ZLMHost, title, session.ChannelId, s.config.ZLMHost, start, end, session.RTPPort, session.SSRC)
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
	deviceIP := s.config.ZLMHost
	devicePort := 5060

	if s.deviceService != nil {
		device, err := s.deviceService.GetDevice(session.DeviceId)
		if err == nil && device.IP != "" {
			deviceIP = device.IP
			devicePort = device.Port
			log.Debug().Str("device_id", session.DeviceId).Str("ip", deviceIP).Int("port", devicePort).Msg("使用设备真实地址发送 INVITE")
		} else {
			log.Warn().Err(err).Str("device_id", session.DeviceId).Msg("无法获取设备信息，使用默认地址")
		}
	}

	target := sip.Uri{User: session.ChannelId, Host: deviceIP, Port: devicePort}
	req := sip.NewRequest(sip.INVITE, target)
	req.SetTransport("UDP")
	req.AppendHeader(sip.NewHeader("Content-Type", "application/sdp"))
	req.AppendHeader(sip.NewHeader("Subject", fmt.Sprintf("%s:%s,%s:0", session.ChannelId, session.SSRC, session.DeviceId)))
	req.SetBody([]byte(sdpContent))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := s.client.TransactionRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("发送 INVITE 请求失败: %w", err)
	}
	defer tx.Terminate()

	resp, err := s.getResponse(tx)
	if err != nil {
		return fmt.Errorf("等待 INVITE 响应失败: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("INVITE 失败: %d", resp.StatusCode)
	}

	s.mu.Lock()
	storedSession, exists := s.sessions[session.StreamId]
	if exists {
		storedSession.Status = PlayStatusPlaying
		if via := resp.Via(); via != nil {
			storedSession.Via = via.String()
		}
		if callID := resp.CallID(); callID != nil {
			storedSession.CallID = callID.String()
		}
		if from := resp.From(); from != nil {
			storedSession.From = from.String()
		}
		if to := resp.To(); to != nil {
			storedSession.To = to.String()
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

	log.Info().Str("stream_id", session.StreamId).Msg("播放已建立")
	return nil
}

// getResponse 从事务获取响应
func (s *PlayService) getResponse(tx sip.ClientTransaction) (*sip.Response, error) {
	select {
	case <-tx.Done():
		return nil, fmt.Errorf("transaction timeout")
	case res := <-tx.Responses():
		return res, nil
	}
}

// sendAck 发送 ACK
func (s *PlayService) sendAck(session *PlaySession, inviteReq *sip.Request, resp *sip.Response) error {
	deviceIP := s.config.ZLMHost
	devicePort := 5060

	if s.deviceService != nil {
		device, err := s.deviceService.GetDevice(session.DeviceId)
		if err == nil && device.IP != "" {
			deviceIP = device.IP
			devicePort = device.Port
		}
	}

	target := sip.Uri{User: session.ChannelId, Host: deviceIP, Port: devicePort}
	ack := sip.NewRequest(sip.ACK, target)
	ack.SetTransport("UDP")

	if via := inviteReq.Via(); via != nil {
		ack.AppendHeader(sip.NewHeader("Via", via.String()))
	}
	if callID := inviteReq.CallID(); callID != nil {
		ack.AppendHeader(sip.NewHeader("Call-ID", callID.String()))
	}
	if from := inviteReq.From(); from != nil {
		ack.AppendHeader(sip.NewHeader("From", from.String()))
	}
	if to := resp.To(); to != nil {
		ack.AppendHeader(sip.NewHeader("To", to.String()))
	}
	if cseq := inviteReq.CSeq(); cseq != nil {
		ack.AppendHeader(sip.NewHeader("CSeq", fmt.Sprintf("%d ACK", cseq.SeqNo)))
	}

	if err := s.client.WriteRequest(ack); err != nil {
		return fmt.Errorf("发送 ACK 失败: %w", err)
	}

	log.Debug().Str("stream_id", session.StreamId).Msg("ACK 已发送")
	return nil
}

// sendBye 发送 BYE
func (s *PlayService) sendBye(session *PlaySession) error {
	deviceIP := s.config.ZLMHost
	devicePort := 5060

	if s.deviceService != nil {
		device, err := s.deviceService.GetDevice(session.DeviceId)
		if err == nil && device.IP != "" {
			deviceIP = device.IP
			devicePort = device.Port
		}
	}

	target := sip.Uri{User: session.ChannelId, Host: deviceIP, Port: devicePort}
	req := sip.NewRequest(sip.BYE, target)
	req.SetTransport("UDP")

	if session.Via != "" {
		req.AppendHeader(sip.NewHeader("Via", session.Via))
	}
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

	if err := s.client.WriteRequest(req); err != nil {
		return fmt.Errorf("发送 BYE 请求失败: %w", err)
	}

	log.Debug().Str("device_id", session.DeviceId).Str("channel_id", session.ChannelId).Str("stream_id", session.StreamId).Msg("BYE 请求已发送")
	return nil
}

// parseSSRCFromSDP 从 SDP 解析 SSRC
func parseSSRCFromSDP(sdp string) string {
	lines := strings.Split(sdp, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "y=") {
			return strings.TrimPrefix(line, "y=")
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

// CleanupStaleSessions 清理过期会话
func (s *PlayService) CleanupStaleSessions(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for streamId, session := range s.sessions {
		if now.Sub(session.StartTime) > timeout {
			_, _ = s.zlm.CloseRtpServer(streamId)
			delete(s.sessions, streamId)
			log.Info().Str("stream_id", streamId).Msg("清理过期会话")
		}
	}
}

// GetMediaInfo 获取媒体信息
func (s *PlayService) GetMediaInfo(streamId string) (*types.GetMediaListResp, error) {
	return s.zlm.GetMediaList(s.config.AppName, streamId)
}
