package test

import (
	"github.com/pion/sdp"
	"io"
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
}
