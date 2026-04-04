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
	TargetHost string     // 目标设备 IP
	TargetPort int        // 目标设备端口
	Via        string     // Via header value
	CallID     string     // Call-ID value
	From       string     // From header value
	To         string     // To header value (from 200 OK response)
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

	streamId := s.generateStreamId(deviceId, channelId, mode, rangeStart, rangeEnd)

	s.mu.RLock()
	if session, exists := s.sessions[streamId]; exists && session.Status == PlayStatusPlaying {
		s.mu.RUnlock()
		// 检查 ZLM 流是否真的存在（避免返回已断开的会话）
		if s.isStreamActive(streamId) {
			log.Info().Str("stream_id", streamId).Msg("会话已存在且流活跃，复用会话")
			return s.buildPlayResult(session), nil
		}
		// 流已断开，清理旧会话
		log.Warn().Str("stream_id", streamId).Msg("会话存在但流已断开，重新建立")
		s.cleanupSession(streamId)
	} else {
		s.mu.RUnlock()
	}

	rtpResp, err := s.zlm.OpenRtpServer(streamId, 0, 0) // port=0 自动分配, tcpMode=0 UDP 模式
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

	// 获取设备地址
	deviceIP := s.config.ZLMHost
	devicePort := 5060
	if s.deviceService != nil {
		device, err := s.deviceService.GetDevice(deviceId)
		if err == nil && device.IP != "" {
			deviceIP = device.IP
			devicePort = device.Port
		}
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

	session := &PlaySession{
		StreamId:   streamId,
		DeviceId:   deviceId,
		ChannelId:  channelId,
		RTPPort:    rtpResp.Port,
		StartTime:  time.Now(),
		Status:     PlayStatusPending,
		SSRC:       ssrc,
		Mode:       mode,
		RangeStart: rangeStart,
		RangeEnd:   rangeEnd,
		TargetHost: deviceIP,
		TargetPort: devicePort,
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

	session.Status = PlayStatusPlaying
	log.Info().Str("device_id", deviceId).Str("channel_id", channelId).Str("stream_id", streamId).Str("mode", string(mode)).Msg("开始播放")
	return s.buildPlayResult(session), nil
}

// isStreamActive 检查 ZLM 流是否真实存在
func (s *PlayService) isStreamActive(streamId string) bool {
	if s.zlm == nil {
		return false
	}

	// 方法1：查询 RTP 信息
	rtpInfo, err := s.zlm.GetRtpInfo(streamId)
	if err == nil && rtpInfo.Code == 0 {
		return true
	}

	// 方法2：查询媒体列表
	mediaList, err := s.zlm.GetMediaList("rtp", streamId)
	if err == nil && mediaList.Code == 0 && len(mediaList.Data) > 0 {
		return true
	}

	return false
}

// cleanupSession 清理旧会话资源
func (s *PlayService) cleanupSession(streamId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return
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

	log.Info().Str("stream_id", streamId).Msg("清理旧会话")
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

	// 释放 SSRC
	if s.ssrcService != nil && session.SSRC != "" {
		s.ssrcService.ReleaseSsrc(session.SSRC)
	}

	session.Status = PlayStatusStopped
	delete(s.sessions, streamId)

	log.Info().Str("stream_id", streamId).Str("mode", string(session.Mode)).Str("ssrc", session.SSRC).Msg("停止播放")
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
		return fmt.Sprintf(`v=0
o=%s 0 0 IN IP4 %s
s=%s
u=%s:0
c=IN IP4 %s
t=%d %d
m=video %d RTP/AVP 96 97 98 99
a=recvonly
a=rtpmap:96 PS/90000
a=rtpmap:97 MPEG4/90000
a=rtpmap:98 H264/90000
a=rtpmap:99 H265/90000
y=%s
f=
`, originUser, s.config.ZLMHost, title, session.ChannelId, s.config.ZLMHost, start, end, session.RTPPort, session.SSRC)
	}

	// 实时播放模式（无 u= 行）
	return fmt.Sprintf(`v=0
o=%s 0 0 IN IP4 %s
s=%s
c=IN IP4 %s
t=%d %d
m=video %d RTP/AVP 96 97 98 99
a=recvonly
a=rtpmap:96 PS/90000
a=rtpmap:97 MPEG4/90000
a=rtpmap:98 H264/90000
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
	// RFC 3261: INVITE 必须包含 Max-Forwards header
	req.AppendHeader(sip.NewHeader("Max-Forwards", "70"))
	req.AppendHeader(sip.NewHeader("Content-Type", "application/sdp"))
	// COMPAT_JAVA: Subject 格式为 {通道ID}:{SSRC},{平台ID}:0（与 Java InviteRequestBuilder 一致）
	localId := s.config.LocalId
	if localId == "" {
		localId = session.DeviceId // 回退使用设备ID
	}
	req.AppendHeader(sip.NewHeader("Subject", fmt.Sprintf("%s:%s,%s:0", session.ChannelId, session.SSRC, localId)))

	// RFC 3261: INVITE 应包含 Contact header
	// Contact: <sip:平台ID@服务器IP:端口>
	sipIP := s.config.SIPListenIP
	if sipIP == "" || sipIP == "0.0.0.0" {
		sipIP = s.config.ZLMHost // 回退使用 ZLMediaKit IP
	}
	sipPort := s.config.SIPPort
	if sipPort == 0 {
		sipPort = 5060 // 默认 SIP 端口
	}
	req.AppendHeader(sip.NewHeader("Contact", fmt.Sprintf("<sip:%s@%s:%d>", localId, sipIP, sipPort)))

	// User-Agent: 自定义标识
	req.AppendHeader(sip.NewHeader("User-Agent", "GB28181-Go-Server"))

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
	deviceIP := s.config.ZLMHost
	devicePort := 5060

	if s.deviceService != nil {
		device, err := s.deviceService.GetDevice(session.DeviceId)
		if err == nil && device.IP != "" {
			deviceIP = device.IP
			devicePort = device.Port
		}
	}

	// RFC 3261 Section 13.2.2.4: ACK Request-URI MUST be the same as INVITE
	// 标准实现：使用通道ID（与INVITE一致）
	target := sip.Uri{User: session.ChannelId, Host: deviceIP, Port: devicePort}
	ack := sip.NewRequest(sip.ACK, target)
	ack.SetTransport("UDP")

	// RFC 3261: Via header由sipgo库自动生成，包含唯一branch参数
	// 直接添加头部对象，避免 .String() 导致重复头部名称
	if callID := inviteReq.CallID(); callID != nil {
		ack.AppendHeader(callID)
	}
	if from := inviteReq.From(); from != nil {
		ack.AppendHeader(from)
	}
	if to := resp.To(); to != nil {
		ack.AppendHeader(to)
	}
	if cseq := inviteReq.CSeq(); cseq != nil {
		ack.AppendHeader(sip.NewHeader("CSeq", fmt.Sprintf("%d ACK", cseq.SeqNo)))
	}
	// RFC 3261: ACK 必须包含 Max-Forwards header
	ack.AppendHeader(sip.NewHeader("Max-Forwards", "70"))

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
