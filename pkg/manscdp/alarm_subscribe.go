package manscdp

import (
	"encoding/xml"

	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
)

// AlarmSubscribeReq 报警订阅请求
// 平台向设备发起报警订阅，设备通过 Notify 消息实时推送报警
type AlarmSubscribeReq struct {
	XMLName     xml.Name `xml:"Subscribe"`
	CmdType     string   `xml:"CmdType"`
	SN          string   `xml:"SN"`
	DeviceID    string   `xml:"DeviceID"`
	Expires     string   `xml:"Expires"`     // 订阅过期时间（秒）
	SubscribeID string   `xml:"SubscribeID"` // 订阅ID，用于标识订阅
}

// NewAlarmSubscribeReq 创建报警订阅请求
func NewAlarmSubscribeReq(sn, deviceID, expires, subscribeID string) *AlarmSubscribeReq {
	return &AlarmSubscribeReq{
		XMLName:     xml.Name{Local: "Subscribe"},
		CmdType:     cmdtype.Alarm,
		SN:          sn,
		DeviceID:    deviceID,
		Expires:     expires,
		SubscribeID: subscribeID,
	}
}

// AlarmSubscribeResp 报警订阅响应
// 设备响应平台的报警订阅请求
type AlarmSubscribeResp struct {
	XMLName     xml.Name `xml:"Response"`
	CmdType     string   `xml:"CmdType"`
	SN          string   `xml:"SN"`
	DeviceID    string   `xml:"DeviceID"`
	Result      string   `xml:"Result"`      // 订阅结果：OK 或 ERROR
	Expires     string   `xml:"Expires"`     // 订阅过期时间（秒）
	SubscribeID string   `xml:"SubscribeID"` // 订阅ID
}

// NewAlarmSubscribeResp 创建报警订阅响应
func NewAlarmSubscribeResp(sn, deviceID, result, expires, subscribeID string) *AlarmSubscribeResp {
	return &AlarmSubscribeResp{
		XMLName:     xml.Name{Local: "Response"},
		CmdType:     cmdtype.Alarm,
		SN:          sn,
		DeviceID:    deviceID,
		Result:      result,
		Expires:     expires,
		SubscribeID: subscribeID,
	}
}
