package model

import (
	"time"
)

// DevicePosition 设备定位历史记录
type DevicePosition struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	DeviceID  string    `gorm:"index;size:20;not null" json:"device_id"`      // 设备编码
	Longitude float64   `gorm:"type:decimal(10,6);not null" json:"longitude"` // 经度
	Latitude  float64   `gorm:"type:decimal(10,6);not null" json:"latitude"`  // 纬度
	Speed     float64   `gorm:"type:decimal(6,2)" json:"speed"`               // 速度 km/h
	Direction int       `gorm:"type:smallint" json:"direction"`               // 方向 0-360度
	Altitude  float64   `gorm:"type:decimal(8,2)" json:"altitude"`            // 高度 米
	GPSTime   time.Time `gorm:"index;not null" json:"gps_time"`               // 定位时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (DevicePosition) TableName() string {
	return "device_positions"
}
