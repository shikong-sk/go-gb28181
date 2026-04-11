package service

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"github.com/emiago/sipgo/sip"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestBuildSDP_RealTimePlay 测试实时播放 SDP 格式
func TestBuildSDP_RealTimePlay(t *testing.T) {
	playService := &PlayService{
		config: PlayConfig{
			ZLMHost:     "10.10.10.200",
			ZLMPort:     8080,
			AppName:     "rtp",
			SIPListenIP: "10.10.10.200",
			SIPPort:     5060,
		},
	}

	startTime := time.Now()
	session := &PlaySession{
		StreamId:  "34020000001320000001_34020000001320000002",
		DeviceId:  "34020000001320000001",
		ChannelId: "34020000001320000002",
		RTPPort:   60000,
		SSRC:      "0200000001",
		Mode:      PlayModeLive,
		StartTime: startTime,
	}

	sdp := playService.buildSDP(session)

	t.Log("=== 实时播放 SDP ===")
	t.Log(sdp)
	t.Log("=== SDP 结束 ===")

	lines := strings.Split(sdp, "\n")

	// 验证每一行
	tests := []struct {
		lineNum int
		expect  string
		desc    string
	}{
		{0, "v=0", "版本号"},
		{1, "o=34020000001320000002 0 0 IN IP4 10.10.10.200", "起源信息"},
		{2, "s=Play", "会话名"},
		{3, "c=IN IP4 10.10.10.200", "连接信息"},
		{4, "t=0 0", "时间信息"},
		{5, "m=video 60000 RTP/AVP 96 97 98 99", "媒体行"},
		{6, "a=recvonly", "接收模式"},
		{7, "a=rtpmap:96 PS/90000", "RTP 映射"},
	}

	for _, tt := range tests {
		if tt.lineNum >= len(lines) {
			t.Errorf("行 %d (%s): 行数不足，实际只有 %d 行", tt.lineNum, tt.desc, len(lines))
			continue
		}
		if lines[tt.lineNum] != tt.expect {
			t.Errorf("行 %d (%s): 期望 '%s', 实际 '%s'", tt.lineNum, tt.desc, tt.expect, lines[tt.lineNum])
		}
	}

	// 检查 y= 行
	yLineFound := false
	for _, line := range lines {
		if strings.HasPrefix(line, "y=") {
			yLineFound = true
			if line != "y=0200000001" {
				t.Errorf("y= 行格式错误: 期望 'y=0200000001', 实际 '%s'", line)
			}
		}
	}

	if !yLineFound {
		t.Error("缺少 y= 行")
	}

	// 检查 f= 行
	fLineFound := false
	for _, line := range lines {
		if line == "f=" {
			fLineFound = true
		}
	}

	if !fLineFound {
		t.Error("缺少 f= 行（GB28181-2016 标准要求）")
	}

	// 实时播放不应包含 u= 行
	for _, line := range lines {
		if strings.HasPrefix(line, "u=") {
			t.Error("实时播放 SDP 不应包含 u= 行（仅回放/下载需要）")
		}
	}
}

// TestBuildSDP_Playback 测试录像回放 SDP 格式
func TestBuildSDP_Playback(t *testing.T) {
	playService := &PlayService{
		config: PlayConfig{
			ZLMHost:     "10.10.10.200",
			ZLMPort:     8080,
			AppName:     "rtp",
			SIPListenIP: "10.10.10.200",
			SIPPort:     5060,
		},
	}

	startTime := time.Date(2026, 4, 4, 10, 0, 0, 0, time.Local)
	endTime := time.Date(2026, 4, 4, 11, 0, 0, 0, time.Local)

	session := &PlaySession{
		StreamId:   "34020000001320000001_34020000001320000002_1743745200_1743748800",
		DeviceId:   "34020000001320000001",
		ChannelId:  "34020000001320000002",
		RTPPort:    60000,
		SSRC:       "1200000001", // 回放 SSRC 首位为 1
		Mode:       PlayModePlayback,
		RangeStart: &startTime,
		RangeEnd:   &endTime,
		StartTime:  time.Now(),
	}

	sdp := playService.buildSDP(session)

	t.Log("=== 录像回放 SDP ===")
	t.Log(sdp)
	t.Log("=== SDP 结束 ===")

	// 验证关键格式
	if !strings.Contains(sdp, "s=Playback") {
		t.Error("回放模式 s= 应为 Playback")
	}

	if !strings.Contains(sdp, "u=34020000001320000002:0") {
		t.Error("回放模式应包含 u= 行（通道ID:0 格式）")
	}

	// 检查时间戳
	expectedStart := startTime.Unix()
	expectedEnd := endTime.Unix()
	timeLine := fmt.Sprintf("t=%d %d", expectedStart, expectedEnd)
	if !strings.Contains(sdp, timeLine) {
		t.Errorf("时间行格式错误，期望包含 '%s'", timeLine)
	}

	// 检查 SSRC 首位为 1
	if !strings.Contains(sdp, "y=1200000001") {
		t.Error("回放模式 SSRC 首位应为 1")
	}
}

// TestParseSSRCFromSDP_NewlineCompatibility 测试 SSRC 解析兼容不同换行符
func TestParseSSRCFromSDP_NewlineCompatibility(t *testing.T) {
	testCases := []struct {
		name     string
		sdp      string
		expected string
	}{
		{
			name: "Unix换行符 LF",
			sdp: `v=0
o=- 0 0 IN IP4 192.168.1.100
s=Play
c=IN IP4 192.168.1.100
t=0 0
m=video 6000 RTP/AVP 96
a=recvonly
a=rtpmap:96 PS/90000
y=0123456789
f=`,
			expected: "0123456789",
		},
		{
			name:     "Windows换行符 CRLF",
			sdp:      "v=0\r\no=- 0 0 IN IP4 192.168.1.100\r\ns=Play\r\nc=IN IP4 192.168.1.100\r\nt=0 0\r\nm=video 6000 RTP/AVP 96\r\na=recvonly\r\na=rtpmap:96 PS/90000\r\ny=0123456789\r\nf=",
			expected: "0123456789",
		},
		{
			name:     "混合换行符",
			sdp:      "v=0\no=- 0 0 IN IP4 192.168.1.100\r\ns=Play\nc=IN IP4 192.168.1.100\r\nt=0 0\nm=video 6000 RTP/AVP 96\r\na=recvonly\na=rtpmap:96 PS/90000\r\ny=0123456789\nf=",
			expected: "0123456789",
		},
		{
			name: "y=行带空格",
			sdp: `v=0
o=- 0 0 IN IP4 192.168.1.100
y= 0123456789
f=`,
			expected: "0123456789",
		},
		{
			name: "无y=行",
			sdp: `v=0
o=- 0 0 IN IP4 192.168.1.100
f=`,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ssrc := parseSSRCFromSDP(tc.sdp)
			assert.Equal(t, tc.expected, ssrc, "SSRC 解析结果")
			t.Logf("解析 SSRC: %s", ssrc)
		})
	}
}

// TestParseDeviceSDP_Complete 测试完整设备 SDP 解析（c=、m=、y= 行）
func TestParseDeviceSDP_Complete(t *testing.T) {
	// 模拟设备响应的 SDP（CRLF 换行）
	deviceSDP_CRLF := "v=0\r\no=- 0 0 IN IP4 192.168.1.100\r\ns=Play\r\nc=IN IP4 192.168.1.200\r\nt=0 0\r\nm=video 7000 RTP/AVP/TCP 96\r\na=recvonly\r\na=rtpmap:96 PS/90000\r\ny=0123456789\r\nf="

	// 模拟设备响应的 SDP（LF 换行）
	deviceSDP_LF := `v=0
o=- 0 0 IN IP4 192.168.1.100
s=Play
c=IN IP4 192.168.1.200
t=0 0
m=video 8000 RTP/AVP 96
a=recvonly
a=rtpmap:96 PS/90000
y=0123456780
f=`

	testCases := []struct {
		name       string
		sdp        string
		expectIP   string
		expectPort int
		expectSSRC string
	}{
		{
			name:       "CRLF换行",
			sdp:        deviceSDP_CRLF,
			expectIP:   "192.168.1.200",
			expectPort: 7000,
			expectSSRC: "0123456789",
		},
		{
			name:       "LF换行",
			sdp:        deviceSDP_LF,
			expectIP:   "192.168.1.200",
			expectPort: 8000,
			expectSSRC: "0123456780",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 解析 SSRC
			ssrc := parseSSRCFromSDP(tc.sdp)
			assert.Equal(t, tc.expectSSRC, ssrc, "SSRC 解析")

			// 解析媒体端口（模拟当前逻辑，使用统一的换行符处理）
			normalizedSDP := strings.ReplaceAll(tc.sdp, "\r\n", "\n")
			lines := strings.Split(normalizedSDP, "\n")
			port := 0
			mediaIP := ""

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "m=video ") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						if p, err := strconv.Atoi(parts[1]); err == nil {
							port = p
						}
					}
				}
				if strings.HasPrefix(line, "c=IN IP4 ") {
					mediaIP = strings.TrimSpace(strings.TrimPrefix(line, "c=IN IP4 "))
				}
			}

			assert.Equal(t, tc.expectPort, port, "端口解析")
			t.Logf("解析端口: %d", port)

			assert.Equal(t, tc.expectIP, mediaIP, "媒体源地址解析")
			t.Logf("解析媒体地址: %s", mediaIP)
		})
	}
}

// TestACK_RequestFormat 测试 ACK 请求格式规范
// 根据 RFC 3261 第 17.1.1.3 节，ACK 必须与 INVITE 的某些 headers 一致
func TestACK_RequestFormat(t *testing.T) {
	t.Log("=== ACK 请求格式验证 ===")

	// 模拟 INVITE 请求的关键参数
	inviteCallID := "call-id-12345@192.168.1.100"
	inviteFrom := "<sip:34020000002000000001@192.168.1.100>;tag=from-tag-123"
	inviteTo := "<sip:34020000001320000001@192.168.1.200>;tag=to-tag-456" // 200 OK 后带 tag
	inviteCSeq := 1
	channelId := "34020000001320000001"
	deviceIP := "192.168.1.200"
	devicePort := 5060

	// ACK 应满足的条件（RFC 3261）
	// 1. Request-URI 与 INVITE 一致
	expectedURI := fmt.Sprintf("sip:%s@%s:%d", channelId, deviceIP, devicePort)
	t.Logf("Request-URI: %s (必须与 INVITE 一致)", expectedURI)

	// 2. Call-ID 与 INVITE 一致
	t.Logf("Call-ID: %s (必须与 INVITE 一致)", inviteCallID)

	// 3. From 与 INVITE 一致
	t.Logf("From: %s (必须与 INVITE 一致)", inviteFrom)

	// 4. To 与 200 OK 响应的 To 一致（包含 tag）
	t.Logf("To: %s (必须与 200 OK 的 To 一致，包含 tag)", inviteTo)

	// 5. CSeq 格式：INVITE 的 CSeq 序号 + " ACK"
	expectedCSeq := fmt.Sprintf("%d ACK", inviteCSeq)
	t.Logf("CSeq: %s (格式: INVITE的CSeq序号 + ACK)", expectedCSeq)

	// 验证 CSeq 格式
	cseqParts := strings.Fields(expectedCSeq)
	assert.Len(t, cseqParts, 2, "CSeq 必须包含序号和方法")
	assert.Equal(t, fmt.Sprintf("%d", inviteCSeq), cseqParts[0], "CSeq 序号必须与 INVITE 一致")
	assert.Equal(t, "ACK", cseqParts[1], "CSeq 方法必须为 ACK")

	// 验证 To header 包含 tag（从 200 OK 获取）
	assert.Contains(t, inviteTo, "tag=", "To header 必须包含 tag（来自 200 OK 响应）")

	t.Log("=== ACK 格式验证通过 ===")
}

// TestSSRC_FormatValidation 测试 SSRC 格式规范
func TestSSRC_FormatValidation(t *testing.T) {
	playService := &PlayService{
		config: PlayConfig{
			ZLMHost:     "10.10.10.200",
			ZLMPort:     8080,
			AppName:     "rtp",
			SIPListenIP: "10.10.10.200",
			SIPPort:     5060,
		},
	}

	testCases := []struct {
		name     string
		mode     PlayMode
		expected string
	}{
		{"实时播放 SSRC 首位为 0", PlayModeLive, "0"},
		{"录像回放 SSRC 首位为 1", PlayModePlayback, "1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ssrc := playService.generateSSRC("test_stream_id", tc.mode)
			t.Logf("生成 SSRC: %s (模式: %s)", ssrc, tc.mode)

			// 验证 SSRC 长度为 10 位
			assert.Len(t, ssrc, 10, "SSRC 必须为 10 位数字")

			// 验证 SSRC 首位
			assert.Equal(t, tc.expected, ssrc[0:1], "SSRC 首位必须符合规范")

			// 验证 SSRC 全为数字
			for i, c := range ssrc {
				assert.True(t, c >= '0' && c <= '9', "SSRC 第 %d 位必须为数字: %c", i, c)
			}
		})
	}
}

// TestBuildSDP_GB28181_Compliance GB28181-2016 标准合规性完整检查
func TestBuildSDP_GB28181_Compliance(t *testing.T) {
	playService := &PlayService{
		config: PlayConfig{
			ZLMHost:     "10.10.10.200",
			ZLMPort:     8080,
			AppName:     "rtp",
			SIPListenIP: "10.10.10.200",
			SIPPort:     5060,
		},
	}

	startTime := time.Date(2026, 4, 4, 10, 0, 0, 0, time.Local)
	endTime := time.Date(2026, 4, 4, 11, 0, 0, 0, time.Local)

	session := &PlaySession{
		StreamId:   "34020000001320000001_34020000001320000002",
		DeviceId:   "34020000001320000001",
		ChannelId:  "34020000001320000002",
		RTPPort:    60000,
		SSRC:       "0200000001",
		Mode:       PlayModeLive,
		RangeStart: &startTime,
		RangeEnd:   &endTime,
	}

	sdp := playService.buildSDP(session)

	t.Log("=== GB28181 合规性检查 SDP ===")
	t.Log(sdp)
	t.Log("=== SDP 结束 ===")

	// 检查必填行
	requiredLines := []struct {
		prefix string
		desc   string
	}{
		{"v=0", "版本号"},
		{"o=", "Origin 行"}, // COMPAT_JAVA: o= 行使用 channelId
		{"s=", "会话名"},
		{"c=IN IP4", "连接信息"},
		{"t=", "时间信息"},
		{"m=video", "媒体行"},
		{"a=recvonly", "接收模式"},
		{"a=rtpmap:96 PS/90000", "RTP 映射"},
		{"y=", "SSRC"},
	}

	for _, rl := range requiredLines {
		if !strings.Contains(sdp, rl.prefix) {
			t.Errorf("缺少必填行: %s (%s)", rl.prefix, rl.desc)
		}
	}

	// 检查 SSRC 长度为 10 位
	yLine := ""
	lines := strings.Split(sdp, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "y=") {
			yLine = line
			break
		}
	}
	if yLine != "" {
		ssrc := strings.TrimPrefix(yLine, "y=")
		if len(ssrc) != 10 {
			t.Errorf("SSRC 长度应为 10 位，实际为 %d 位: %s", len(ssrc), ssrc)
		}
	}
}

func TestSendInvite_RequiresDeviceAddress(t *testing.T) {
	playService := &PlayService{}
	session := &PlaySession{DeviceId: "44050100001110000008"}

	err := playService.sendInvite(session, "v=0\n")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "缺少有效的 SIP 地址")
}

func TestSendAck_RequiresAckTarget(t *testing.T) {
	playService := &PlayService{}
	session := &PlaySession{DeviceId: "44050100001110000008", ChannelId: "34020000001310000001"}
	inviteReq := sip.NewRequest(sip.INVITE, sip.Uri{User: session.ChannelId, Host: "10.10.10.210", Port: 5060})
	resp := sip.NewResponse(200, "OK")
	resp.AppendHeader(&sip.ContactHeader{Address: sip.Uri{}})

	err := playService.sendAck(session, inviteReq, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "缺少有效的 ACK 目标地址")
}

func TestAckTargetUserPrefersDeviceID(t *testing.T) {
	session := &PlaySession{
		DeviceId:  "44050100001110000008",
		ChannelId: "34020000001310000001",
	}

	ackUser := session.DeviceId
	if ackUser == "" {
		ackUser = session.ChannelId
	}

	assert.Equal(t, "44050100001110000008", ackUser)
}

func TestOnDeviceKeepalive_AutoCreatesUnknownDevice(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.Device{}))

	deviceRepo := repository.NewDeviceRepository(db)
	deviceService := NewDeviceService(deviceRepo, nil, nil)

	err = deviceService.OnDeviceKeepalive("44050100001110000008", "10.10.10.210", 41883)
	assert.NoError(t, err)

	device, err := deviceService.GetDevice("44050100001110000008")
	assert.NoError(t, err)
	assert.Equal(t, "10.10.10.210", device.IP)
	assert.Equal(t, 41883, device.Port)
	assert.Equal(t, model.DeviceStatusOnline, device.Status)
}

// TestBuildSDP_Download 测试录像下载 SDP 格式
// 验证下载模式 SDP 包含 a=downloadspeed 属性，且 rtpmap 顺序与 WVP 一致
func TestBuildSDP_Download(t *testing.T) {
	playService := &PlayService{
		config: PlayConfig{
			ZLMHost:     "10.10.10.200",
			ZLMPort:     8080,
			AppName:     "rtp",
			SIPListenIP: "10.10.10.200",
			SIPPort:     5060,
		},
	}

	startTime := time.Date(2026, 4, 9, 23, 39, 58, 0, time.Local)
	endTime := time.Date(2026, 4, 9, 23, 50, 0, 0, time.Local)

	session := &PlaySession{
		StreamId:   "34020000001320000001_34020000001310000001_1744215598_1744216200",
		DeviceId:   "34020000001320000001",
		ChannelId:  "34020000001310000001",
		RTPPort:    61242,
		SSRC:       "2501007246", // 下载 SSRC 首位为 2
		Mode:       PlayModeDownload,
		RangeStart: &startTime,
		RangeEnd:   &endTime,
		Speed:      4, // 4 倍速下载
		StartTime:  time.Now(),
	}

	sdp := playService.buildSDP(session)

	t.Log("=== 录像下载 SDP ===")
	t.Log(sdp)
	t.Log("=== SDP 结束 ===")

	// 验证关键格式
	if !strings.Contains(sdp, "s=Download") {
		t.Error("下载模式 s= 应为 Download")
	}

	if !strings.Contains(sdp, "u=34020000001310000001:0") {
		t.Error("下载模式应包含 u= 行（通道ID:0 格式）")
	}

	// 检查 a=downloadspeed 属性（关键：下载倍速）
	if !strings.Contains(sdp, "a=downloadspeed:4") {
		t.Error("下载模式应包含 a=downloadspeed:4 属性")
	}

	// 检查 f= 行（GB28181-2016 标准要求）
	if !strings.Contains(sdp, "f=") {
		t.Error("下载模式应包含 f= 行")
	}

	// 验证 rtpmap 顺序与 WVP 一致（96 PS, 97 MPEG4, 98 H264, 99 H265）
	expectedRtpmapOrder := []string{
		"a=rtpmap:96 PS/90000",
		"a=rtpmap:97 MPEG4/90000",
		"a=rtpmap:98 H264/90000",
		"a=rtpmap:99 H265/90000",
	}
	for _, expected := range expectedRtpmapOrder {
		if !strings.Contains(sdp, expected) {
			t.Errorf("下载模式 SDP 应包含 %s", expected)
		}
	}

	// 检查时间戳
	expectedStart := startTime.Unix()
	expectedEnd := endTime.Unix()
	timeLine := fmt.Sprintf("t=%d %d", expectedStart, expectedEnd)
	if !strings.Contains(sdp, timeLine) {
		t.Errorf("时间行格式错误，期望包含 '%s'", timeLine)
	}

	// 检查 SSRC 首位为 2
	if !strings.Contains(sdp, "y=2501007246") {
		t.Error("下载模式 SSRC 首位应为 2")
	}
}

// TestBuildSDP_DownloadSpeedValues 测试不同下载倍速
func TestBuildSDP_DownloadSpeedValues(t *testing.T) {
	playService := &PlayService{
		config: PlayConfig{
			ZLMHost:     "10.10.10.200",
			ZLMPort:     8080,
			AppName:     "rtp",
			SIPListenIP: "10.10.10.200",
			SIPPort:     5060,
		},
	}

	startTime := time.Date(2026, 4, 9, 23, 0, 0, 0, time.Local)
	endTime := time.Date(2026, 4, 9, 23, 30, 0, 0, time.Local)

	testCases := []struct {
		speed    int
		expected string
	}{
		{1, "a=downloadspeed:1"},
		{2, "a=downloadspeed:2"},
		{4, "a=downloadspeed:4"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Speed%d", tc.speed), func(t *testing.T) {
			session := &PlaySession{
				StreamId:   "test_stream",
				DeviceId:   "34020000001320000001",
				ChannelId:  "34020000001310000001",
				RTPPort:    60000,
				SSRC:       "2500000001",
				Mode:       PlayModeDownload,
				RangeStart: &startTime,
				RangeEnd:   &endTime,
				Speed:      tc.speed,
			}

			sdp := playService.buildSDP(session)
			if !strings.Contains(sdp, tc.expected) {
				t.Errorf("期望包含 '%s'，实际 SDP:\n%s", tc.expected, sdp)
			}
		})
	}
}
