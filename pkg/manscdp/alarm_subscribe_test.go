package manscdp

import (
	"encoding/xml"
	"testing"
)

func TestAlarmSubscribeReq_Marshal(t *testing.T) {
	req := NewAlarmSubscribeReq("12345", "34020000001110000001", "3600", "sub001")

	data, err := xml.MarshalIndent(req, "", " ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("Generated XML:\n%s", string(data))

	// 验证关键字段
	if req.CmdType != "Alarm" {
		t.Errorf("Expected CmdType='Alarm', got '%s'", req.CmdType)
	}
	if req.SN != "12345" {
		t.Errorf("Expected SN='12345', got '%s'", req.SN)
	}
	if req.DeviceID != "34020000001110000001" {
		t.Errorf("Expected DeviceID='34020000001110000001', got '%s'", req.DeviceID)
	}
	if req.Expires != "3600" {
		t.Errorf("Expected Expires='3600', got '%s'", req.Expires)
	}
	if req.SubscribeID != "sub001" {
		t.Errorf("Expected SubscribeID='sub001', got '%s'", req.SubscribeID)
	}
}

func TestAlarmSubscribeReq_Unmarshal(t *testing.T) {
	xmlData := `<Subscribe>
 <CmdType>Alarm</CmdType>
 <SN>12345</SN>
 <DeviceID>34020000001110000001</DeviceID>
 <Expires>3600</Expires>
 <SubscribeID>sub001</SubscribeID>
</Subscribe>`

	var req AlarmSubscribeReq
	err := xml.Unmarshal([]byte(xmlData), &req)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if req.CmdType != "Alarm" {
		t.Errorf("Expected CmdType='Alarm', got '%s'", req.CmdType)
	}
	if req.SN != "12345" {
		t.Errorf("Expected SN='12345', got '%s'", req.SN)
	}
	if req.DeviceID != "34020000001110000001" {
		t.Errorf("Expected DeviceID='34020000001110000001', got '%s'", req.DeviceID)
	}
	if req.Expires != "3600" {
		t.Errorf("Expected Expires='3600', got '%s'", req.Expires)
	}
	if req.SubscribeID != "sub001" {
		t.Errorf("Expected SubscribeID='sub001', got '%s'", req.SubscribeID)
	}
}

func TestAlarmSubscribeResp_Marshal(t *testing.T) {
	resp := NewAlarmSubscribeResp("12345", "34020000001110000001", "OK", "3600", "sub001")

	data, err := xml.MarshalIndent(resp, "", " ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("Generated XML:\n%s", string(data))

	// 验证关键字段
	if resp.CmdType != "Alarm" {
		t.Errorf("Expected CmdType='Alarm', got '%s'", resp.CmdType)
	}
	if resp.Result != "OK" {
		t.Errorf("Expected Result='OK', got '%s'", resp.Result)
	}
}

func TestAlarmSubscribeResp_Unmarshal(t *testing.T) {
	xmlData := `<Response>
 <CmdType>Alarm</CmdType>
 <SN>12345</SN>
 <DeviceID>34020000001110000001</DeviceID>
 <Result>OK</Result>
 <Expires>3600</Expires>
 <SubscribeID>sub001</SubscribeID>
</Response>`

	var resp AlarmSubscribeResp
	err := xml.Unmarshal([]byte(xmlData), &resp)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.CmdType != "Alarm" {
		t.Errorf("Expected CmdType='Alarm', got '%s'", resp.CmdType)
	}
	if resp.Result != "OK" {
		t.Errorf("Expected Result='OK', got '%s'", resp.Result)
	}
	if resp.Expires != "3600" {
		t.Errorf("Expected Expires='3600', got '%s'", resp.Expires)
	}
	if resp.SubscribeID != "sub001" {
		t.Errorf("Expected SubscribeID='sub001', got '%s'", resp.SubscribeID)
	}
}

func TestNewAlarmSubscribeReq(t *testing.T) {
	req := NewAlarmSubscribeReq("12345", "34020000001110000001", "3600", "sub001")

	if req == nil {
		t.Fatal("NewAlarmSubscribeReq returned nil")
	}

	if req.XMLName.Local != "Subscribe" {
		t.Errorf("Expected XMLName.Local='Subscribe', got '%s'", req.XMLName.Local)
	}
}

func TestNewAlarmSubscribeResp(t *testing.T) {
	resp := NewAlarmSubscribeResp("12345", "34020000001110000001", "OK", "3600", "sub001")

	if resp == nil {
		t.Fatal("NewAlarmSubscribeResp returned nil")
	}

	if resp.XMLName.Local != "Response" {
		t.Errorf("Expected XMLName.Local='Response', got '%s'", resp.XMLName.Local)
	}
}
