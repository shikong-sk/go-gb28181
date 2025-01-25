package cmdtype

type CmdType = string

const (
	Keepalive      CmdType = "Keepalive"
	DeviceConfig   CmdType = "DeviceConfig"
	DeviceControl  CmdType = "DeviceControl"
	DeviceStatus   CmdType = "DeviceStatus"
	Catalog        CmdType = "Catalog"
	Alarm          CmdType = "Alarm"
	MobilePosition CmdType = "MobilePosition"
	Broadcast      CmdType = "Broadcast"
	RecordInfo     CmdType = "RecordInfo"
	MediaStatus    CmdType = "MediaStatus"
	ConfigDownload CmdType = "ConfigDownload"
	PresetQuery    CmdType = "PresetQuery"
)
