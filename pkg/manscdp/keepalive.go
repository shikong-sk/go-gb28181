package manscdp

import (
	"encoding/xml"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
)

type KeepAliveReq struct {
	XMLName  xml.Name `xml:"Notify"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
	Status   string   `xml:"Status"`
}

func NewKeepAliveReqWithOK(sn, deviceID string) *KeepAliveReq {
	return &KeepAliveReq{
		XMLName:  xml.Name{Local: "Notify"},
		CmdType:  cmdtype.Keepalive,
		SN:       sn,
		DeviceID: deviceID,
		Status:   "OK",
	}
}

func NewKeepAliveReq(sn, deviceID, status string) *KeepAliveReq {
	return &KeepAliveReq{
		XMLName:  xml.Name{Local: "Notify"},
		CmdType:  cmdtype.Keepalive,
		SN:       sn,
		DeviceID: deviceID,
		Status:   status,
	}
}
