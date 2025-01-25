package test

import (
	"github.com/duke-git/lancet/v2/formatter"
	"github.com/pion/sdp"
	"io"
	"net"
	"regexp"
	"strings"
	"testing"
)

func TestSDP(t *testing.T) {
	req := `v=0
o=00000000000000000002 0 0 IN IP4 10.10.10.200
s=Play
c=IN IP4 10.10.10.200
t=0 0
m=video 5080 RTP/AVP 99 125 126 96 97 98
a=recvonly
a=rtpmap:99 H265/90000
a=fmtp:125 profile-level-id=42e01e
a=rtpmap:125 H264S/90000
a=fmtp:126 profile-level-id=42e01e
a=rtpmap:126 H264/90000
a=rtpmap:96 PS/90000
a=rtpmap:97 MPEG4/90000
a=rtpmap:98 H264/90000
a=setup:active
a=connection:new
a=streamnumber:0
y=501000001
f=
`
	// 匹配 28181 协议的 ssrc
	ssrcReg := regexp.MustCompile("y=(.*)")
	matchSsrc := ssrcReg.FindStringSubmatch(req)
	ssrc := matchSsrc[1]
	t.Log("y= ", ssrc)
	req = strings.Replace(req, matchSsrc[0], "", 1)

	// 匹配 28181 协议的 format
	formatReg := regexp.MustCompile("f=(.*)")
	matchFormat := formatReg.FindStringSubmatch(req)
	format := matchFormat[1]
	t.Log("f= ", format)
	req = strings.Replace(req, matchFormat[0], "", 1)

	session := &sdp.SessionDescription{}
	err := session.Unmarshal(req)
	if err != nil && err != io.EOF {
		t.Error(err)
	}
	t.Logf("%#v\n", session)

	pretty, _ := formatter.Pretty(session)
	t.Logf("%s\n", pretty)
}

func TestSDP_Marshal(t *testing.T) {
	description := sdp.NewJSEPSessionDescription(false)
	description.SessionName = "Play"

	description.ConnectionInformation = &sdp.ConnectionInformation{
		NetworkType: "IN",
		AddressType: "IP4",
		Address:     &sdp.Address{IP: net.ParseIP("10.10.10.200")},
	}

	description.Origin.Username = "00000000000000000002"
	description.Origin.SessionID = 0
	description.Origin.SessionVersion = 0
	description.Origin.UnicastAddress = "10.10.10.200"

	media := &sdp.MediaDescription{
		MediaName: sdp.MediaName{
			Media:   "video",
			Port:    sdp.RangedPort{Value: 5080},
			Protos:  []string{"RTP", "AVP"},
			Formats: []string{"99", "125", "126", "96", "97", "98"},
		},
		Attributes: []sdp.Attribute{
			{Key: "rtpmap", Value: "99 H265/90000"},
			{Key: "fmtp", Value: "125 profile-level-id=42e01e"},
			{Key: "rtpmap", Value: "125 H264S/90000"},
			{Key: "fmtp", Value: "126 profile-level-id=42e01e"},
			{Key: "rtpmap", Value: "126 H264/90000"},
			{Key: "rtpmap", Value: "96 PS/90000"},
			{Key: "rtpmap", Value: "97 MPEG4/90000"},
			{Key: "rtpmap", Value: "98 H264/90000"},
			{Key: "sendonly"},
		},
	}

	description.MediaDescriptions = append(description.MediaDescriptions, media)
	sdpResp := description.Marshal()

	// 添加 y= 和 f= 字段
	sdpResp += "y=501000001\r\n"
	sdpResp += "f=\r\n"

	t.Logf("\n%s\n", sdpResp)
}
