package manscdp

import (
	"encoding/xml"
	"strings"
	"testing"
)

const mediaStatusXML = `<?xml version="1.0" encoding="UTF-8"?>
<Notify>
  <CmdType>MediaStatus</CmdType>
  <SN>1</SN>
  <DeviceID>44050100001310000001</DeviceID>
  <NotifyType>121</NotifyType>
  <StreamId>stream123456</StreamId>
</Notify>`

func TestMediaStatusNotifyUnmarshal(t *testing.T) {
	notify := &MediaStatusNotify{}
	err := xml.Unmarshal([]byte(mediaStatusXML), notify)
	if err != nil {
		t.Fatalf("XML 解码失败: %v", err)
	}

	// 验证字段解析
	if notify.CmdType != "MediaStatus" {
		t.Errorf("CmdType 解析失败: 期望 MediaStatus, 实际 %s", notify.CmdType)
	}
	if notify.SN != "1" {
		t.Errorf("SN 解析失败: 期望 1, 实际 %s", notify.SN)
	}
	if notify.DeviceID != "44050100001310000001" {
		t.Errorf("DeviceID 解析失败: 期望 44050100001310000001, 实际 %s", notify.DeviceID)
	}
	// 121 = PlaybackStart（回放开始）
	if notify.NotifyType != MediaStatusNotifyTypePlaybackStart {
		t.Errorf("NotifyType 解析失败: 期望 %d (PlaybackStart), 实际 %d", MediaStatusNotifyTypePlaybackStart, notify.NotifyType)
	}
	if notify.StreamId != "stream123456" {
		t.Errorf("StreamId 解析失败: 期望 stream123456, 实际 %s", notify.StreamId)
	}
}

func TestMediaStatusNotifyMarshal(t *testing.T) {
	notify := NewMediaStatusNotify("2", "44050100001310000002", MediaStatusNotifyTypePlaybackEnd, "stream789")

	xmlBytes, err := xml.Marshal(notify)
	if err != nil {
		t.Fatalf("XML 编码失败: %v", err)
	}

	xmlStr := string(xmlBytes)

	// 验证关键字段存在
	if !strings.Contains(xmlStr, "<CmdType>MediaStatus</CmdType>") {
		t.Error("编码结果缺少 CmdType")
	}
	if !strings.Contains(xmlStr, "<SN>2</SN>") {
		t.Error("编码结果缺少 SN")
	}
	if !strings.Contains(xmlStr, "<DeviceID>44050100001310000002</DeviceID>") {
		t.Error("编码结果缺少 DeviceID")
	}
	// 122 = PlaybackEnd（回放结束）
	if !strings.Contains(xmlStr, "<NotifyType>122</NotifyType>") {
		t.Error("编码结果缺少 NotifyType")
	}
	if !strings.Contains(xmlStr, "<StreamId>stream789</StreamId>") {
		t.Error("编码结果缺少 StreamId")
	}
}

func TestMediaStatusNotifyTypeConstant(t *testing.T) {
	// 验证常量值（GB28181-2016 标准）
	// 121 = 回放开始，122 = 回放结束
	if MediaStatusNotifyTypePlaybackStart != 121 {
		t.Errorf("MediaStatusNotifyTypePlaybackStart 常量值错误: 期望 121, 实际 %d", MediaStatusNotifyTypePlaybackStart)
	}
	if MediaStatusNotifyTypePlaybackEnd != 122 {
		t.Errorf("MediaStatusNotifyTypePlaybackEnd 常量值错误: 期望 122, 实际 %d", MediaStatusNotifyTypePlaybackEnd)
	}
	// 兼容旧常量名
	if MediaStatusNotifyTypeRecordEnd != 122 {
		t.Errorf("MediaStatusNotifyTypeRecordEnd 常量值错误: 期望 122, 实际 %d", MediaStatusNotifyTypeRecordEnd)
	}
}

func TestNewMediaStatusNotify(t *testing.T) {
	notify := NewMediaStatusNotify("10", "device001", MediaStatusNotifyTypePlaybackEnd, "test-stream")

	if notify.XMLName.Local != "Notify" {
		t.Errorf("XMLName 错误: 期望 Notify, 实际 %s", notify.XMLName.Local)
	}
	if notify.CmdType != "MediaStatus" {
		t.Errorf("CmdType 错误: 期望 MediaStatus, 实际 %s", notify.CmdType)
	}
	if notify.SN != "10" {
		t.Errorf("SN 错误: 期望 10, 实际 %s", notify.SN)
	}
	if notify.DeviceID != "device001" {
		t.Errorf("DeviceID 错误: 期望 device001, 实际 %s", notify.DeviceID)
	}
	if notify.NotifyType != MediaStatusNotifyTypePlaybackEnd {
		t.Errorf("NotifyType 错误: 期望 %d, 实际 %d", MediaStatusNotifyTypePlaybackEnd, notify.NotifyType)
	}
	if notify.StreamId != "test-stream" {
		t.Errorf("StreamId 错误: 期望 test-stream, 实际 %s", notify.StreamId)
	}
}
