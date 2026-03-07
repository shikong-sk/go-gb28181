package model

import (
	"time"

	"gorm.io/gorm"
)

// Alarm 报警记录模型
type Alarm struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 报警信息
	DeviceID         string `gorm:"index;size:20;not null" json:"deviceId"` // 设备国标编码
	AlarmPriority    string `gorm:"size:1;not null" json:"alarmPriority"`   // 报警级别: 1=一级, 2=二级, 3=三级
	AlarmMethod      string `gorm:"size:2;not null" json:"alarmMethod"`     // 报警方式: 见 GB/T 28181 附录A
	AlarmTime        string `gorm:"size:32;not null" json:"alarmTime"`      // 报警时间
	AlarmDescription string `gorm:"size:512" json:"alarmDescription"`       // 报警描述
	AlarmInfo        string `gorm:"type:text" json:"alarmInfo"`             // 报警附加信息
	Longitude        string `gorm:"size:16" json:"longitude"`               // 经度
	Latitude         string `gorm:"size:16" json:"latitude"`                // 纬度
}

// TableName 指定表名
func (Alarm) TableName() string {
	return "alarms"
}

// AlarmPriority 报警级别常量
const (
	AlarmPriorityOne   = "1" // 一级报警 (最高)
	AlarmPriorityTwo   = "2" // 二级报警
	AlarmPriorityThree = "3" // 三级报警 (最低)
)
