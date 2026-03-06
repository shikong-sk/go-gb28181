package manscdp

import "encoding/xml"

// MessageHeader 通用消息头，用于解析所有类型的消息
// 支持 Query, Response, Notify, Control 等根元素
type MessageHeader struct {
	XMLName  xml.Name // 不限制元素名，动态解析
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
}

// NotifyReq Notify 消息结构（设备主动上报）
type NotifyReq struct {
	XMLName  xml.Name `xml:"Notify"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
}

// ResponseReq Response 消息结构（设备响应查询）
type ResponseReq struct {
	XMLName  xml.Name `xml:"Response"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
}

// AlarmNotify 报警通知消息
type AlarmNotify struct {
	XMLName          xml.Name `xml:"Notify"`
	CmdType          string   `xml:"CmdType"`
	SN               string   `xml:"SN"`
	DeviceID         string   `xml:"DeviceID"`
	AlarmPriority    string   `xml:"AlarmPriority"`
	AlarmMethod      string   `xml:"AlarmMethod"`
	AlarmTime        string   `xml:"AlarmTime"`
	AlarmDescription string   `xml:"AlarmDescription"`
	AlarmInfo        string   `xml:"AlarmInfo"`
}
