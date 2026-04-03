package manscdp

import (
	"encoding/xml"
	"testing"
	"time"
)

func TestMobilePositionReq(t *testing.T) {
	// 测试 MobilePositionReq 的 XML 编码
	req := NewMobilePositionReq("1", "34020000001110000001")

	// 验证结构体字段
	if req.CmdType != "MobilePosition" {
		t.Errorf("expected CmdType 'MobilePosition', got '%s'", req.CmdType)
	}
	if req.SN != "1" {
		t.Errorf("expected SN '1', got '%s'", req.SN)
	}
	if req.DeviceID != "34020000001110000001" {
		t.Errorf("expected DeviceID '34020000001110000001', got '%s'", req.DeviceID)
	}
	if req.XMLName.Local != "Query" {
		t.Errorf("expected XMLName.Local 'Query', got '%s'", req.XMLName.Local)
	}

	// 测试 XML 编码
	xmlBytes, err := xml.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal MobilePositionReq: %v", err)
	}

	// 验证 XML 内容包含必要的元素
	xmlStr := string(xmlBytes)
	expectedElements := []string{"<Query>", "<CmdType>MobilePosition</CmdType>", "<SN>1</SN>", "<DeviceID>34020000001110000001</DeviceID>"}
	for _, elem := range expectedElements {
		if !containsString(xmlStr, elem) {
			t.Errorf("expected XML to contain %s, got %s", elem, xmlStr)
		}
	}

	// 测试 XML 解码
	var decodedReq MobilePositionReq
	err = xml.Unmarshal(xmlBytes, &decodedReq)
	if err != nil {
		t.Fatalf("failed to unmarshal MobilePositionReq: %v", err)
	}

	// 验证解码后的字段
	if decodedReq.SN != req.SN {
		t.Errorf("expected SN '%s', got '%s'", req.SN, decodedReq.SN)
	}
	if decodedReq.DeviceID != req.DeviceID {
		t.Errorf("expected DeviceID '%s', got '%s'", req.DeviceID, decodedReq.DeviceID)
	}
}

func TestMobilePositionNotify(t *testing.T) {
	// 测试 MobilePositionNotify 的 XML 编码
	gpsTime := time.Date(2026, 4, 3, 10, 30, 0, 0, time.UTC)
	notify := NewMobilePositionNotify("2", "34020000001110000001", 117.123456, 31.654321, 45.5, 50.0, 90, gpsTime)

	// 验证结构体字段
	if notify.CmdType != "MobilePosition" {
		t.Errorf("expected CmdType 'MobilePosition', got '%s'", notify.CmdType)
	}
	if notify.SN != "2" {
		t.Errorf("expected SN '2', got '%s'", notify.SN)
	}
	if notify.DeviceID != "34020000001110000001" {
		t.Errorf("expected DeviceID '34020000001110000001', got '%s'", notify.DeviceID)
	}
	if notify.Longitude != 117.123456 {
		t.Errorf("expected Longitude 117.123456, got %f", notify.Longitude)
	}
	if notify.Latitude != 31.654321 {
		t.Errorf("expected Latitude 31.654321, got %f", notify.Latitude)
	}
	if notify.Speed != 45.5 {
		t.Errorf("expected Speed 45.5, got %f", notify.Speed)
	}
	if notify.Direction != 90 {
		t.Errorf("expected Direction 90, got %d", notify.Direction)
	}
	if notify.Altitude != 50.0 {
		t.Errorf("expected Altitude 50.0, got %f", notify.Altitude)
	}
	if notify.GPSTime != "2026-04-03T10:30:00" {
		t.Errorf("expected GPSTime '2026-04-03T10:30:00', got '%s'", notify.GPSTime)
	}

	// 测试 XML 编码
	xmlBytes, err := xml.Marshal(notify)
	if err != nil {
		t.Fatalf("failed to marshal MobilePositionNotify: %v", err)
	}

	xmlStr := string(xmlBytes)

	// 验证 XML 内容包含必要的元素
	if !containsString(xmlStr, "<Notify>") {
		t.Errorf("expected XML to contain <Notify>")
	}
	if !containsString(xmlStr, "<CmdType>MobilePosition</CmdType>") {
		t.Errorf("expected XML to contain <CmdType>MobilePosition</CmdType>")
	}
	if !containsString(xmlStr, "<SN>2</SN>") {
		t.Errorf("expected XML to contain <SN>2</SN>")
	}

	// 测试 XML 解码
	var decodedNotify MobilePositionNotify
	err = xml.Unmarshal(xmlBytes, &decodedNotify)
	if err != nil {
		t.Fatalf("failed to unmarshal MobilePositionNotify: %v", err)
	}

	// 验证 float64 字段解析正确性
	if decodedNotify.Longitude != 117.123456 {
		t.Errorf("expected Longitude 117.123456, got %f", decodedNotify.Longitude)
	}
	if decodedNotify.Latitude != 31.654321 {
		t.Errorf("expected Latitude 31.654321, got %f", decodedNotify.Latitude)
	}
	if decodedNotify.Speed != 45.5 {
		t.Errorf("expected Speed 45.5, got %f", decodedNotify.Speed)
	}
	if decodedNotify.Altitude != 50.0 {
		t.Errorf("expected Altitude 50.0, got %f", decodedNotify.Altitude)
	}
}

func TestMobilePositionNotifyFromXML(t *testing.T) {
	// 测试从原始 XML 解析（使用 UTF-8 编码，符合国标 XML 通常处理方式）
	xmlStr := `<Notify>
  <CmdType>MobilePosition</CmdType>
  <SN>3</SN>
  <DeviceID>34020000001110000001</DeviceID>
  <Longitude>117.123456</Longitude>
  <Latitude>31.654321</Latitude>
  <Speed>45.5</Speed>
  <Direction>90</Direction>
  <Altitude>50.0</Altitude>
  <GPSTime>2026-04-03T10:30:00</GPSTime>
</Notify>`

	var notify MobilePositionNotify
	err := xml.Unmarshal([]byte(xmlStr), &notify)
	if err != nil {
		t.Fatalf("failed to unmarshal from XML: %v", err)
	}

	// 验证字段映射
	if notify.CmdType != "MobilePosition" {
		t.Errorf("expected CmdType 'MobilePosition', got '%s'", notify.CmdType)
	}
	if notify.SN != "3" {
		t.Errorf("expected SN '3', got '%s'", notify.SN)
	}
	if notify.DeviceID != "34020000001110000001" {
		t.Errorf("expected DeviceID '34020000001110000001', got '%s'", notify.DeviceID)
	}
	if notify.Longitude != 117.123456 {
		t.Errorf("expected Longitude 117.123456, got %f", notify.Longitude)
	}
	if notify.Latitude != 31.654321 {
		t.Errorf("expected Latitude 31.654321, got %f", notify.Latitude)
	}
	if notify.Speed != 45.5 {
		t.Errorf("expected Speed 45.5, got %f", notify.Speed)
	}
	if notify.Direction != 90 {
		t.Errorf("expected Direction 90, got %d", notify.Direction)
	}
	if notify.Altitude != 50.0 {
		t.Errorf("expected Altitude 50.0, got %f", notify.Altitude)
	}
	if notify.GPSTime != "2026-04-03T10:30:00" {
		t.Errorf("expected GPSTime '2026-04-03T10:30:00', got '%s'", notify.GPSTime)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
