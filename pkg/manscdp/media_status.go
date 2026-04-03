package manscdp

import (
	"encoding/xml"

	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
)

// MediaStatusNotifyType 媒体状态通知类型
type MediaStatusNotifyType int

const (
	MediaStatusNotifyTypeRecordEnd MediaStatusNotifyType = 121 // 录像结束
)

// MediaStatusNotify 媒体状态通知
type MediaStatusNotify struct {
	XMLName    xml.Name              `xml:"Notify"`
	CmdType    string                `xml:"CmdType"`
	SN         string                `xml:"SN"`
	DeviceID   string                `xml:"DeviceID"`
	NotifyType MediaStatusNotifyType `xml:"NotifyType"` // 121=录像结束
	StreamId   string                `xml:"StreamId"`   // 流ID
}

// NewMediaStatusNotify 创建一个新的 MediaStatusNotify 实例
func NewMediaStatusNotify(sn, deviceID string, notifyType MediaStatusNotifyType, streamId string) *MediaStatusNotify {
	return &MediaStatusNotify{
		XMLName:    xml.Name{Local: "Notify"},
		CmdType:    cmdtype.MediaStatus,
		SN:         sn,
		DeviceID:   deviceID,
		NotifyType: notifyType,
		StreamId:   streamId,
	}
}
