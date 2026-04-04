package manscdp

import (
	"testing"

	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceInfoReq_Marshal(t *testing.T) {
	req := NewDeviceInfoReq("12345", "34020000001110000001")

	xmlBytes, err := utils.XMLMarshal(req, "UTF-8")
	require.NoError(t, err)
	xmlStr := string(xmlBytes)

	t.Logf("Generated XML:\n%s", xmlStr)

	// 验证 XML 包含必要的元素
	assert.Contains(t, xmlStr, "<Query>")
	assert.Contains(t, xmlStr, "<CmdType>DeviceInfo</CmdType>")
	assert.Contains(t, xmlStr, "<SN>12345</SN>")
	assert.Contains(t, xmlStr, "<DeviceID>34020000001110000001</DeviceID>")
}

func TestDeviceInfoReq_Unmarshal(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Query>
  <CmdType>DeviceInfo</CmdType>
  <SN>12345</SN>
  <DeviceID>34020000001110000001</DeviceID>
</Query>`

	var req DeviceInfoReq
	err := utils.XMLUnmarshal([]byte(xmlData), &req)
	require.NoError(t, err)

	assert.Equal(t, "DeviceInfo", req.CmdType)
	assert.Equal(t, "12345", req.SN)
	assert.Equal(t, "34020000001110000001", req.DeviceID)
}

func TestDeviceInfoResp_Marshal(t *testing.T) {
	resp := &DeviceInfoResp{
		CmdType:      "DeviceInfo",
		SN:           "12345",
		DeviceID:     "34020000001110000001",
		Result:       "OK",
		DeviceName:   "IPC Camera 01",
		Manufacturer: "Hikvision",
		Model:        "DS-2CD3T86FWDV2-I3S",
		Firmware:     "V5.5.800",
		Channel:      4,
	}

	xmlBytes, err := utils.XMLMarshal(resp, "UTF-8")
	require.NoError(t, err)
	xmlStr := string(xmlBytes)

	t.Logf("Generated XML:\n%s", xmlStr)

	// 验证 XML 包含必要的元素
	assert.Contains(t, xmlStr, "<Response>")
	assert.Contains(t, xmlStr, "<CmdType>DeviceInfo</CmdType>")
	assert.Contains(t, xmlStr, "<SN>12345</SN>")
	assert.Contains(t, xmlStr, "<DeviceID>34020000001110000001</DeviceID>")
	assert.Contains(t, xmlStr, "<Result>OK</Result>")
	assert.Contains(t, xmlStr, "<DeviceName>IPC Camera 01</DeviceName>")
	assert.Contains(t, xmlStr, "<Manufacturer>Hikvision</Manufacturer>")
	assert.Contains(t, xmlStr, "<Model>DS-2CD3T86FWDV2-I3S</Model>")
	assert.Contains(t, xmlStr, "<Firmware>V5.5.800</Firmware>")
	assert.Contains(t, xmlStr, "<Channel>4</Channel>")
}

func TestDeviceInfoResp_Unmarshal(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Response>
  <CmdType>DeviceInfo</CmdType>
  <SN>12345</SN>
  <DeviceID>34020000001110000001</DeviceID>
  <Result>OK</Result>
  <DeviceName>IPC Camera 01</DeviceName>
  <Manufacturer>Hikvision</Manufacturer>
  <Model>DS-2CD3T86FWDV2-I3S</Model>
  <Firmware>V5.5.800</Firmware>
  <Channel>4</Channel>
</Response>`

	var resp DeviceInfoResp
	err := utils.XMLUnmarshal([]byte(xmlData), &resp)
	require.NoError(t, err)

	assert.Equal(t, "DeviceInfo", resp.CmdType)
	assert.Equal(t, "12345", resp.SN)
	assert.Equal(t, "34020000001110000001", resp.DeviceID)
	assert.Equal(t, "OK", resp.Result)
	assert.Equal(t, "IPC Camera 01", resp.DeviceName)
	assert.Equal(t, "Hikvision", resp.Manufacturer)
	assert.Equal(t, "DS-2CD3T86FWDV2-I3S", resp.Model)
	assert.Equal(t, "V5.5.800", resp.Firmware)
	assert.Equal(t, 4, resp.Channel)
}

func TestNewDeviceInfoReq(t *testing.T) {
	req := NewDeviceInfoReq("99999", "34020000009999999999")

	assert.Equal(t, "DeviceInfo", req.CmdType)
	assert.Equal(t, "99999", req.SN)
	assert.Equal(t, "34020000009999999999", req.DeviceID)
}

func TestNewDeviceInfoResp(t *testing.T) {
	resp := NewDeviceInfoResp("99999", "34020000009999999999", "OK")

	assert.Equal(t, "DeviceInfo", resp.CmdType)
	assert.Equal(t, "99999", resp.SN)
	assert.Equal(t, "34020000009999999999", resp.DeviceID)
	assert.Equal(t, "OK", resp.Result)
}
