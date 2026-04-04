package model

import (
	"time"

	"gorm.io/gorm"
)

// Device 设备模型
type Device struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 设备信息
	DeviceID     string  `gorm:"uniqueIndex;size:20;not null" json:"deviceId"` // 设备国标编码 (20位)
	Name         string  `gorm:"size:255" json:"name"`                         // 设备名称
	Manufacturer string  `gorm:"size:255" json:"manufacturer"`                 // 厂商
	Model        string  `gorm:"size:255" json:"model"`                        // 型号
	Firmware     string  `gorm:"size:255" json:"firmware"`                     // 固件版本
	Owner        string  `gorm:"size:255" json:"owner"`                        // 归属
	CivilCode    string  `gorm:"size:20" json:"civilCode"`                     // 行政区划代码
	Block        string  `gorm:"size:255" json:"block"`                        // 警区
	Address      string  `gorm:"size:255" json:"address"`                      // 安装地址
	ParentID     string  `gorm:"size:20;index" json:"parentId"`                // 父设备 ID
	SafetyWay    string  `gorm:"size:1" json:"safetyWay"`                      // 安全传输方式
	RegisterWay  string  `gorm:"size:1" json:"registerWay"`                    // 注册方式
	Secrecy      string  `gorm:"size:1" json:"secrecy"`                        // 保密属性
	IP           string  `gorm:"size:45" json:"ip"`                            // IP 地址 (IPv6 最长 45)
	Port         int     `gorm:"default:0" json:"port"`                        // 端口
	Password     string  `gorm:"size:255" json:"-"`                            // 密码 (敏感信息不输出)
	Status       string  `gorm:"size:1;default:'0'" json:"status"`             // 状态: 0=离线, 1=在线
	Longitude    float64 `gorm:"type:decimal(10,6)" json:"longitude"`          // 经度
	Latitude     float64 `gorm:"type:decimal(10,6)" json:"latitude"`           // 纬度

	// 业务字段
	LastRegisterTime  time.Time `json:"lastRegisterTime"`                        // 最后注册时间
	LastKeepaliveTime time.Time `json:"lastKeepaliveTime"`                       // 最后心跳时间
	ChannelCount      int       `gorm:"default:0" json:"channelCount"`           // 通道数量
	StreamMode        string    `gorm:"size:20;default:'UDP'" json:"streamMode"` // 流传输模式: UDP, TCP-ACTIVE, TCP-PASSIVE
}

// TableName 指定表名
func (Device) TableName() string {
	return "devices"
}

// IsOnline 判断设备是否在线
func (d *Device) IsOnline() bool {
	return d.Status == "1"
}

// DeviceStatus 设备状态常量
const (
	DeviceStatusOffline = "0" // 离线
	DeviceStatusOnline  = "1" // 在线
)

// Channel 通道模型
type Channel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 通道信息
	ChannelID    string  `gorm:"uniqueIndex:idx_device_channel;size:20;not null" json:"channelId"`      // 通道国标编码 (20位)
	DeviceID     string  `gorm:"uniqueIndex:idx_device_channel;size:20;not null;index" json:"deviceId"` // 所属设备 ID
	Name         string  `gorm:"size:255" json:"name"`                                                  // 通道名称
	Manufacturer string  `gorm:"size:255" json:"manufacturer"`                                          // 厂商
	Model        string  `gorm:"size:255" json:"model"`                                                 // 型号
	Owner        string  `gorm:"size:255" json:"owner"`                                                 // 归属
	CivilCode    string  `gorm:"size:20" json:"civilCode"`                                              // 行政区划代码
	Block        string  `gorm:"size:255" json:"block"`                                                 // 警区
	Address      string  `gorm:"size:255" json:"address"`                                               // 安装地址
	ParentID     string  `gorm:"size:20" json:"parentId"`                                               // 父设备 ID
	SafetyWay    string  `gorm:"size:1" json:"safetyWay"`                                               // 安全传输方式
	RegisterWay  string  `gorm:"size:1" json:"registerWay"`                                             // 注册方式
	Secrecy      string  `gorm:"size:1" json:"secrecy"`                                                 // 保密属性
	IP           string  `gorm:"size:45" json:"ip"`                                                     // IP 地址
	Port         int     `gorm:"default:0" json:"port"`                                                 // 端口
	Status       string  `gorm:"size:1;default:'0'" json:"status"`                                      // 状态: 0=离线, 1=在线
	Longitude    float64 `gorm:"type:decimal(10,6)" json:"longitude"`                                   // 经度
	Latitude     float64 `gorm:"type:decimal(10,6)" json:"latitude"`                                    // 纬度

	// 业务字段
	PTZType    int  `gorm:"default:0" json:"ptzType"`        // PTZ 类型: 0=未知, 1=球机, 2=半球, 3=固定机, 4=遥控机
	StreamType int  `gorm:"default:0" json:"streamType"`     // 流类型: 0=普通, 1=高清
	HasAudio   bool `gorm:"default:false" json:"hasAudio"`   // 是否有音频
	GPSChannel bool `gorm:"default:false" json:"gpsChannel"` // 是否 GPS 通道
}

// TableName 指定表名
func (Channel) TableName() string {
	return "channels"
}

// IsOnline 判断通道是否在线
func (c *Channel) IsOnline() bool {
	return c.Status == "1"
}

// ChannelStatus 通道状态常量
const (
	ChannelStatusOffline = "0" // 离线
	ChannelStatusOnline  = "1" // 在线
)

// PTZType PTZ 类型常量
const (
	PTZTypeUnknown  = 0 // 未知
	PTZTypeBall     = 1 // 球机
	PTZTypeHalfBall = 2 // 半球
	PTZTypeFixed    = 3 // 固定机
	PTZTypeRemote   = 4 // 遥控机
)

// StreamType 流类型常量
const (
	StreamTypeNormal = 0 // 普通
	StreamTypeHD     = 1 // 高清
)
