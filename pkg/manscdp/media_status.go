package manscdp

import (
	"encoding/xml"

	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
)

// MediaStatusNotifyType 媒体状态通知类型
type MediaStatusNotifyType int

const (
	// MediaStatusNotifyTypePlaybackStart 录像文件回放开始
	// 设备开始播放录像文件，表示媒体流即将就绪
	MediaStatusNotifyTypePlaybackStart MediaStatusNotifyType = 121
	// MediaStatusNotifyTypePlaybackEnd 录像文件回放结束
	// 设备播放录像文件结束，表示媒体流已结束
	MediaStatusNotifyTypePlaybackEnd MediaStatusNotifyType = 122
	// 兼容旧常量名（已废弃，请使用 PlaybackEnd）
	MediaStatusNotifyTypeRecordEnd MediaStatusNotifyType = 122
)

// MediaStatusNotify 媒体状态通知
type MediaStatusNotify struct {
	XMLName    xml.Name              `xml:"Notify"`
	CmdType    string                `xml:"CmdType"`
	SN         string                `xml:"SN"`
	DeviceID   string                `xml:"DeviceID"`
	NotifyType MediaStatusNotifyType `xml:"NotifyType"`         // 121=回放开始, 122=回放结束
	StreamId   string                `xml:"StreamId,omitempty"` // 流ID（设备可能不发送）
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
