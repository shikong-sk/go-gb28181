package model

import (
	"time"

	"gorm.io/gorm"
)

// RecordCache 录像缓存模型
// 用于缓存设备录像查询结果，减少对设备的重复查询
type RecordCache struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 设备和通道信息
	DeviceID  string `gorm:"uniqueIndex:idx_device_channel_date;size:20;not null;index" json:"deviceId"`  // 设备国标编码 (20位)
	ChannelID string `gorm:"uniqueIndex:idx_device_channel_date;size:20;not null;index" json:"channelId"` // 通道国标编码 (20位)

	// 缓存时间范围
	QueryDate string `gorm:"uniqueIndex:idx_device_channel_date;size:10;not null" json:"queryDate"` // 查询日期 (格式: 2006-01-02)
	StartTime string `gorm:"size:19;not null" json:"startTime"`                                     // 查询开始时间 (格式: 2006-01-02 15:04:05)
	EndTime   string `gorm:"size:19;not null" json:"endTime"`                                       // 查询结束时间 (格式: 2006-01-02 15:04:05)

	// 录像数据
	RecordItemsJSON string `gorm:"type:text;not null" json:"recordItemsJson"` // 录像项列表 JSON (序列化的 []RecordItem)
	ItemCount       int    `gorm:"default:0" json:"itemCount"`                // 录像项数量

	// 缓存状态
	FetchedAt   time.Time `json:"fetchedAt"`                             // 拉取时间 (用于判断缓存新鲜度)
	ExpiresAt   time.Time `json:"expiresAt"`                             // 过期时间
	CacheStatus string    `gorm:"size:1;default:'1'" json:"cacheStatus"` // 缓存状态: 0=过期, 1=有效
}

// TableName 指定表名
func (RecordCache) TableName() string {
	return "record_caches"
}

// IsExpired 判断缓存是否过期
func (c *RecordCache) IsExpired() bool {
	return c.CacheStatus == RecordCacheStatusExpired || time.Now().After(c.ExpiresAt)
}

// IsValid 判断缓存是否有效
func (c *RecordCache) IsValid() bool {
	return c.CacheStatus == RecordCacheStatusValid && !time.Now().After(c.ExpiresAt)
}

// RecordCacheStatus 缓存状态常量
const (
	RecordCacheStatusExpired = "0" // 过期
	RecordCacheStatusValid   = "1" // 有效
)

// 默认缓存有效期（小时）
const DefaultCacheExpiryHours = 24

// RecordItem 录像项结构
// 用于存储单条录像信息，便于跨包使用
type RecordItem struct {
	DeviceID  string `json:"device_id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Secrecy   int    `json:"secrecy"`
	Type      string `json:"type"`
	FileSize  int64  `json:"file_size"`
}
