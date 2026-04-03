package manscdp

import (
	"encoding/xml"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
)

// DeviceStatusReq 设备状态查询请求结构
type DeviceStatusReq struct {
	XMLName  xml.Name `xml:"Query"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
}

// NewDeviceStatusReq 创建一个新的设备状态查询请求
func NewDeviceStatusReq(sn, deviceID string) *DeviceStatusReq {
	return &DeviceStatusReq{
		XMLName:  xml.Name{Local: "Query"},
		CmdType:  cmdtype.DeviceStatus,
		SN:       sn,
		DeviceID: deviceID,
	}
}

// DeviceStatusResp 设备状态查询响应结构
type DeviceStatusResp struct {
	XMLName       xml.Name `xml:"Response"`
	CmdType       string   `xml:"CmdType"`
	SN            string   `xml:"SN"`
	DeviceID      string   `xml:"DeviceID"`
	Result        string   `xml:"Result"`        // OK/ERROR
	Online        string   `xml:"Online"`        // ONLINE/OFFLINE
	Status        string   `xml:"Status"`        // OK/ERROR
	DeviceTime    string   `xml:"DeviceTime"`    // 设备时间，格式：yyyy-MM-ddTHH:mm:ss
	RecordStatus  string   `xml:"RecordStatus"`  // 录像状态，REC/STOP/PAUSE
	StorageStatus string   `xml:"StorageStatus"` // 存储状态 OK/ERROR/DISK_ERROR/LOW_SPACE
	NetStatus     string   `xml:"NetStatus"`     // 网络状态 OK/ERROR
}

// NewDeviceStatusResp 创建一个新的设备状态查询响应
func NewDeviceStatusResp(sn, deviceID, result string) *DeviceStatusResp {
	return &DeviceStatusResp{
		XMLName:  xml.Name{Local: "Response"},
		CmdType:  cmdtype.DeviceStatus,
		SN:       sn,
		DeviceID: deviceID,
		Result:   result,
	}
}
