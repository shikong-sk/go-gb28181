package manscdp

import (
	"encoding/xml"
	"testing"
)

func TestSubscribeReq_Marshal(t *testing.T) {
	req := NewSubscribeReq("12345", "34020000001110000001", "3600", "event001")

	data, err := xml.MarshalIndent(req, "", " ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("Generated XML:\n%s", data)

	// 验证必需字段
	if req.CmdType != "Catalog" {
		t.Errorf("Expected CmdType 'Catalog', got '%s'", req.CmdType)
	}
	if req.SN != "12345" {
		t.Errorf("Expected SN '12345', got '%s'", req.SN)
	}
	if req.DeviceID != "34020000001110000001" {
		t.Errorf("Expected DeviceID '34020000001110000001', got '%s'", req.DeviceID)
	}
	if req.Expires != "3600" {
		t.Errorf("Expected Expires '3600', got '%s'", req.Expires)
	}
	if req.EventID != "event001" {
		t.Errorf("Expected EventID 'event001', got '%s'", req.EventID)
	}
}

func TestSubscribeReq_Unmarshal(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Subscribe>
 <CmdType>Catalog</CmdType>
 <SN>12345</SN>
 <DeviceID>34020000001110000001</DeviceID>
 <Expires>3600</Expires>
 <EventID>event001</EventID>
</Subscribe>`

	var req SubscribeReq
	err := xml.Unmarshal([]byte(xmlData), &req)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if req.CmdType != "Catalog" {
		t.Errorf("Expected CmdType 'Catalog', got '%s'", req.CmdType)
	}
	if req.SN != "12345" {
		t.Errorf("Expected SN '12345', got '%s'", req.SN)
	}
	if req.DeviceID != "34020000001110000001" {
		t.Errorf("Expected DeviceID '34020000001110000001', got '%s'", req.DeviceID)
	}
	if req.Expires != "3600" {
		t.Errorf("Expected Expires '3600', got '%s'", req.Expires)
	}
	if req.EventID != "event001" {
		t.Errorf("Expected EventID 'event001', got '%s'", req.EventID)
	}
}

func TestSubscribeResp_Marshal(t *testing.T) {
	resp := NewSubscribeResp("12345", "34020000001110000001", "OK", "3600")

	data, err := xml.MarshalIndent(resp, "", " ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("Generated XML:\n%s", data)

	if resp.CmdType != "Catalog" {
		t.Errorf("Expected CmdType 'Catalog', got '%s'", resp.CmdType)
	}
	if resp.Result != "OK" {
		t.Errorf("Expected Result 'OK', got '%s'", resp.Result)
	}
}

func TestSubscribeResp_Unmarshal(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Response>
 <CmdType>Catalog</CmdType>
 <SN>12345</SN>
 <DeviceID>34020000001110000001</DeviceID>
 <Result>OK</Result>
 <Expires>3600</Expires>
</Response>`

	var resp SubscribeResp
	err := xml.Unmarshal([]byte(xmlData), &resp)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.CmdType != "Catalog" {
		t.Errorf("Expected CmdType 'Catalog', got '%s'", resp.CmdType)
	}
	if resp.SN != "12345" {
		t.Errorf("Expected SN '12345', got '%s'", resp.SN)
	}
	if resp.DeviceID != "34020000001110000001" {
		t.Errorf("Expected DeviceID '34020000001110000001', got '%s'", resp.DeviceID)
	}
	if resp.Result != "OK" {
		t.Errorf("Expected Result 'OK', got '%s'", resp.Result)
	}
	if resp.Expires != "3600" {
		t.Errorf("Expected Expires '3600', got '%s'", resp.Expires)
	}
}

func TestCatalogNotify_Marshal(t *testing.T) {
	deviceList := NewCateLogDeviceList([]CateLogDevice{
		*NewCateLogDevice(func(d *CateLogDevice) {
			d.DeviceID = "34020000001110000002"
			d.Name = "Camera 01"
			d.Status = "ON"
		}),
	})

	notify := NewCatalogNotify("12345", "34020000001110000001", 1, deviceList)

	data, err := xml.MarshalIndent(notify, "", " ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("Generated XML:\n%s", data)

	if notify.CmdType != "Catalog" {
		t.Errorf("Expected CmdType 'Catalog', got '%s'", notify.CmdType)
	}
	if notify.SumNum != "1" {
		t.Errorf("Expected SumNum '1', got '%s'", notify.SumNum)
	}
}

func TestCatalogNotify_Unmarshal(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Notify>
 <CmdType>Catalog</CmdType>
 <SN>12345</SN>
 <DeviceID>34020000001110000001</DeviceID>
 <SumNum>1</SumNum>
 <DeviceList Num="1">
  <Item>
   <DeviceID>34020000001110000002</DeviceID>
   <Name>Camera 01</Name>
   <Status>ON</Status>
  </Item>
 </DeviceList>
</Notify>`

	var notify CatalogNotify
	err := xml.Unmarshal([]byte(xmlData), &notify)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if notify.CmdType != "Catalog" {
		t.Errorf("Expected CmdType 'Catalog', got '%s'", notify.CmdType)
	}
	if notify.SN != "12345" {
		t.Errorf("Expected SN '12345', got '%s'", notify.SN)
	}
	if notify.DeviceID != "34020000001110000001" {
		t.Errorf("Expected DeviceID '34020000001110000001', got '%s'", notify.DeviceID)
	}
	if notify.SumNum != "1" {
		t.Errorf("Expected SumNum '1', got '%s'", notify.SumNum)
	}
	if notify.DeviceList == nil {
		t.Fatal("DeviceList should not be nil")
	}
	if notify.DeviceList.Num != "1" {
		t.Errorf("Expected DeviceList.Num '1', got '%s'", notify.DeviceList.Num)
	}
	if len(notify.DeviceList.Item) != 1 {
		t.Errorf("Expected 1 item, got %d", len(notify.DeviceList.Item))
	}
	if notify.DeviceList.Item[0].DeviceID != "34020000001110000002" {
		t.Errorf("Expected item DeviceID '34020000001110000002', got '%s'", notify.DeviceList.Item[0].DeviceID)
	}
}

func TestNewSubscribeReq(t *testing.T) {
	req := NewSubscribeReq("sn123", "device001", "1800", "event123")

	if req.XMLName.Local != "Subscribe" {
		t.Errorf("Expected XMLName.Local 'Subscribe', got '%s'", req.XMLName.Local)
	}
	if req.CmdType != "Catalog" {
		t.Errorf("Expected CmdType 'Catalog', got '%s'", req.CmdType)
	}
	if req.SN != "sn123" {
		t.Errorf("Expected SN 'sn123', got '%s'", req.SN)
	}
}

func TestNewSubscribeResp(t *testing.T) {
	resp := NewSubscribeResp("sn123", "device001", "OK", "1800")

	if resp.XMLName.Local != "Response" {
		t.Errorf("Expected XMLName.Local 'Response', got '%s'", resp.XMLName.Local)
	}
	if resp.CmdType != "Catalog" {
		t.Errorf("Expected CmdType 'Catalog', got '%s'", resp.CmdType)
	}
	if resp.Result != "OK" {
		t.Errorf("Expected Result 'OK', got '%s'", resp.Result)
	}
}

func TestNewCatalogNotify(t *testing.T) {
	deviceList := NewCateLogDeviceList([]CateLogDevice{
		*NewCateLogDevice(func(d *CateLogDevice) {
			d.DeviceID = "device002"
			d.Name = "Test Device"
		}),
	})

	notify := NewCatalogNotify("sn456", "device001", 1, deviceList)

	if notify.XMLName.Local != "Notify" {
		t.Errorf("Expected XMLName.Local 'Notify', got '%s'", notify.XMLName.Local)
	}
	if notify.CmdType != "Catalog" {
		t.Errorf("Expected CmdType 'Catalog', got '%s'", notify.CmdType)
	}
	if notify.SumNum != "1" {
		t.Errorf("Expected SumNum '1', got '%s'", notify.SumNum)
	}
}
