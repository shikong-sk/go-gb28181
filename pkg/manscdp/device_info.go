package manscdp

import (
	"encoding/xml"

	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
)

// DeviceInfoReq 设备信息查询请求结构
type DeviceInfoReq struct {
	XMLName  xml.Name `xml:"Query"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
}

// NewDeviceInfoReq 创建一个新的设备信息查询请求
func NewDeviceInfoReq(sn, deviceID string) *DeviceInfoReq {
	return &DeviceInfoReq{
		XMLName:  xml.Name{Local: "Query"},
		CmdType:  cmdtype.DeviceInfo,
		SN:       sn,
		DeviceID: deviceID,
	}
}

// DeviceInfoResp 设备信息查询响应结构
type DeviceInfoResp struct {
	XMLName      xml.Name `xml:"Response"`
	CmdType      string   `xml:"CmdType"`
	SN           string   `xml:"SN"`
	DeviceID     string   `xml:"DeviceID"`
	Result       string   `xml:"Result"`       // OK/ERROR
	DeviceName   string   `xml:"DeviceName"`   // 设备名称
	Manufacturer string   `xml:"Manufacturer"` // 厂商
	Model        string   `xml:"Model"`        // 型号
	Firmware     string   `xml:"Firmware"`     // 固件版本
	Channel      int      `xml:"Channel"`      // 通道数
}

// NewDeviceInfoResp 创建一个新的设备信息查询响应
func NewDeviceInfoResp(sn, deviceID, result string) *DeviceInfoResp {
	return &DeviceInfoResp{
		XMLName:  xml.Name{Local: "Response"},
		CmdType:  cmdtype.DeviceInfo,
		SN:       sn,
		DeviceID: deviceID,
		Result:   result,
	}
}
