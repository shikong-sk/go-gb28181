package manscdp

import (
	"testing"

	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceStatusReq_Marshal(t *testing.T) {
	req := NewDeviceStatusReq("12345", "34020000001110000001")

	xmlBytes, err := utils.XMLMarshal(req, "UTF-8")
	require.NoError(t, err)
	xmlStr := string(xmlBytes)

	t.Logf("Generated XML:\n%s", xmlStr)

	// 验证 XML 包含必要的元素
	assert.Contains(t, xmlStr, "<Query>")
	assert.Contains(t, xmlStr, "<CmdType>DeviceStatus</CmdType>")
	assert.Contains(t, xmlStr, "<SN>12345</SN>")
	assert.Contains(t, xmlStr, "<DeviceID>34020000001110000001</DeviceID>")
}

func TestDeviceStatusReq_Unmarshal(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Query>
  <CmdType>DeviceStatus</CmdType>
  <SN>12345</SN>
  <DeviceID>34020000001110000001</DeviceID>
</Query>`

	var req DeviceStatusReq
	err := utils.XMLUnmarshal([]byte(xmlData), &req)
	require.NoError(t, err)

	assert.Equal(t, "DeviceStatus", req.CmdType)
	assert.Equal(t, "12345", req.SN)
	assert.Equal(t, "34020000001110000001", req.DeviceID)
}

func TestDeviceStatusResp_Marshal(t *testing.T) {
	resp := &DeviceStatusResp{
		CmdType:       "DeviceStatus",
		SN:            "12345",
		DeviceID:      "34020000001110000001",
		Result:        "OK",
		Online:        "ONLINE",
		Status:        "OK",
		DeviceTime:    "2024-01-01T12:00:00",
		RecordStatus:  "REC",
		StorageStatus: "OK",
		NetStatus:     "OK",
	}

	xmlBytes, err := utils.XMLMarshal(resp, "UTF-8")
	require.NoError(t, err)
	xmlStr := string(xmlBytes)

	t.Logf("Generated XML:\n%s", xmlStr)

	// 验证 XML 包含必要的元素
	assert.Contains(t, xmlStr, "<Response>")
	assert.Contains(t, xmlStr, "<CmdType>DeviceStatus</CmdType>")
	assert.Contains(t, xmlStr, "<SN>12345</SN>")
	assert.Contains(t, xmlStr, "<DeviceID>34020000001110000001</DeviceID>")
	assert.Contains(t, xmlStr, "<Result>OK</Result>")
	assert.Contains(t, xmlStr, "<Online>ONLINE</Online>")
	assert.Contains(t, xmlStr, "<Status>OK</Status>")
	assert.Contains(t, xmlStr, "<DeviceTime>2024-01-01T12:00:00</DeviceTime>")
	assert.Contains(t, xmlStr, "<RecordStatus>REC</RecordStatus>")
	assert.Contains(t, xmlStr, "<StorageStatus>OK</StorageStatus>")
	assert.Contains(t, xmlStr, "<NetStatus>OK</NetStatus>")
}

func TestDeviceStatusResp_Unmarshal(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Response>
  <CmdType>DeviceStatus</CmdType>
  <SN>12345</SN>
  <DeviceID>34020000001110000001</DeviceID>
  <Result>OK</Result>
  <Online>ONLINE</Online>
  <Status>OK</Status>
  <DeviceTime>2024-01-01T12:00:00</DeviceTime>
  <RecordStatus>REC</RecordStatus>
  <StorageStatus>OK</StorageStatus>
  <NetStatus>OK</NetStatus>
</Response>`

	var resp DeviceStatusResp
	err := utils.XMLUnmarshal([]byte(xmlData), &resp)
	require.NoError(t, err)

	assert.Equal(t, "DeviceStatus", resp.CmdType)
	assert.Equal(t, "12345", resp.SN)
	assert.Equal(t, "34020000001110000001", resp.DeviceID)
	assert.Equal(t, "OK", resp.Result)
	assert.Equal(t, "ONLINE", resp.Online)
	assert.Equal(t, "OK", resp.Status)
	assert.Equal(t, "2024-01-01T12:00:00", resp.DeviceTime)
	assert.Equal(t, "REC", resp.RecordStatus)
	assert.Equal(t, "OK", resp.StorageStatus)
	assert.Equal(t, "OK", resp.NetStatus)
}

func TestDeviceStatusResp_Unmarshal_Offline(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Response>
  <CmdType>DeviceStatus</CmdType>
  <SN>67890</SN>
  <DeviceID>34020000001110000002</DeviceID>
  <Result>OK</Result>
  <Online>OFFLINE</Online>
  <Status>ERROR</Status>
  <DeviceTime>2024-01-01T00:00:00</DeviceTime>
  <RecordStatus>STOP</RecordStatus>
  <StorageStatus>DISK_ERROR</StorageStatus>
  <NetStatus>ERROR</NetStatus>
</Response>`

	var resp DeviceStatusResp
	err := utils.XMLUnmarshal([]byte(xmlData), &resp)
	require.NoError(t, err)

	assert.Equal(t, "OFFLINE", resp.Online)
	assert.Equal(t, "ERROR", resp.Status)
	assert.Equal(t, "STOP", resp.RecordStatus)
	assert.Equal(t, "DISK_ERROR", resp.StorageStatus)
	assert.Equal(t, "ERROR", resp.NetStatus)
}

func TestNewDeviceStatusReq(t *testing.T) {
	req := NewDeviceStatusReq("99999", "34020000009999999999")

	assert.Equal(t, "DeviceStatus", req.CmdType)
	assert.Equal(t, "99999", req.SN)
	assert.Equal(t, "34020000009999999999", req.DeviceID)
}

func TestNewDeviceStatusResp(t *testing.T) {
	resp := NewDeviceStatusResp("99999", "34020000009999999999", "OK")

	assert.Equal(t, "DeviceStatus", resp.CmdType)
	assert.Equal(t, "99999", resp.SN)
	assert.Equal(t, "34020000009999999999", resp.DeviceID)
	assert.Equal(t, "OK", resp.Result)
}
