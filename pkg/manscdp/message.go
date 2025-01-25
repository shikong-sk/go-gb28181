package manscdp

import "encoding/xml"

// MessageReq 定义了查询消息的请求结构体，使用XML格式进行序列化和反序列化
type MessageReq struct {
	XMLName  xml.Name `xml:"Query"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
}
