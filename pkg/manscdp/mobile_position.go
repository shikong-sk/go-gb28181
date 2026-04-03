package manscdp

import (
	"encoding/xml"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
)

const mobilePositionTimeLayout = "2006-01-02T15:04:05"

// MobilePositionReq 移动设备定位查询请求
// 平台向设备发送查询请求，设备响应自身的 GPS 定位信息
type MobilePositionReq struct {
	XMLName  xml.Name `xml:"Query"`
	CmdType  string   `xml:"CmdType"`
	SN       string   `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
}

// NewMobilePositionReq 创建移动设备定位查询请求
func NewMobilePositionReq(sn, deviceID string) *MobilePositionReq {
	return &MobilePositionReq{
		XMLName:  xml.Name{Local: "Query"},
		CmdType:  cmdtype.MobilePosition,
		SN:       sn,
		DeviceID: deviceID,
	}
}

// MobilePositionNotify 移动设备定位上报
// 设备主动上报或响应查询的 GPS 定位信息
type MobilePositionNotify struct {
	XMLName   xml.Name `xml:"Notify"`
	CmdType   string   `xml:"CmdType"`
	SN        string   `xml:"SN"`
	DeviceID  string   `xml:"DeviceID"`
	Longitude float64  `xml:"Longitude"` // 经度
	Latitude  float64  `xml:"Latitude"`  // 纬度
	Speed     float64  `xml:"Speed"`     // 速度 km/h
	Direction int      `xml:"Direction"` // 方向 0-360度
	Altitude  float64  `xml:"Altitude"`  // 高度 米
	GPSTime   string   `xml:"GPSTime"`   // 定位时间
}

// NewMobilePositionNotify 创建移动设备定位上报
func NewMobilePositionNotify(sn, deviceID string, lon, lat, speed, altitude float64, direction int, gpsTime time.Time) *MobilePositionNotify {
	return &MobilePositionNotify{
		XMLName:   xml.Name{Local: "Notify"},
		CmdType:   cmdtype.MobilePosition,
		SN:        sn,
		DeviceID:  deviceID,
		Longitude: lon,
		Latitude:  lat,
		Speed:     speed,
		Direction: direction,
		Altitude:  altitude,
		GPSTime:   gpsTime.Format(mobilePositionTimeLayout),
	}
}
