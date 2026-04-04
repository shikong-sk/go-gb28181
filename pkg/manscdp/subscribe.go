package manscdp

import (
	"encoding/xml"
	"strconv"

	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
)

// SubscribeReq 目录订阅请求
// 平台向设备发起目录订阅，设备通过 NOTIFY 消息实时推送目录变化
type SubscribeReq struct {
	XMLName  xml.Name `xml:"Subscribe"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
	Expires  string   `xml:"Expires"` // 订阅过期时间（秒）
	EventID  string   `xml:"EventID"` // 事件ID，用于标识订阅
}

// NewSubscribeReq 创建目录订阅请求
func NewSubscribeReq(sn, deviceID, expires, eventID string) *SubscribeReq {
	return &SubscribeReq{
		XMLName:  xml.Name{Local: "Subscribe"},
		CmdType:  cmdtype.Catalog,
		SN:       sn,
		DeviceID: deviceID,
		Expires:  expires,
		EventID:  eventID,
	}
}

// SubscribeResp 目录订阅响应
// 设备响应平台的订阅请求
type SubscribeResp struct {
	XMLName  xml.Name `xml:"Response"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
	Result   string   `xml:"Result"`  // 订阅结果：OK 或 ERROR
	Expires  string   `xml:"Expires"` // 订阅过期时间（秒）
}

// NewSubscribeResp 创建目录订阅响应
func NewSubscribeResp(sn, deviceID, result, expires string) *SubscribeResp {
	return &SubscribeResp{
		XMLName:  xml.Name{Local: "Response"},
		CmdType:  cmdtype.Catalog,
		SN:       sn,
		DeviceID: deviceID,
		Result:   result,
		Expires:  expires,
	}
}

// CatalogNotify 目录变化通知
// 设备主动上报目录变化（新增、更新、删除）
type CatalogNotify struct {
	XMLName    xml.Name           `xml:"Notify"`
	CmdType    string             `xml:"CmdType"`
	SN         string             `xml:"SN"`
	DeviceID   string             `xml:"DeviceID"`
	SumNum     string             `xml:"SumNum"` // 总设备数
	DeviceList *CateLogDeviceList `xml:"DeviceList"`
}

// NewCatalogNotify 创建目录变化通知
func NewCatalogNotify(sn, deviceID string, sumNum int, deviceList *CateLogDeviceList) *CatalogNotify {
	return &CatalogNotify{
		XMLName:    xml.Name{Local: "Notify"},
		CmdType:    cmdtype.Catalog,
		SN:         sn,
		DeviceID:   deviceID,
		SumNum:     strconv.Itoa(sumNum),
		DeviceList: deviceList,
	}
}
