package repository

import (
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"gorm.io/gorm"
)

// AlarmRepository 报警数据访问层
type AlarmRepository struct {
	db *gorm.DB
}

// NewAlarmRepository 创建报警仓库
func NewAlarmRepository(db *gorm.DB) *AlarmRepository {
	return &AlarmRepository{db: db}
}

// Create 创建报警记录
func (r *AlarmRepository) Create(alarm *model.Alarm) error {
	return r.db.Create(alarm).Error
}

// GetByID 根据 ID 获取报警
func (r *AlarmRepository) GetByID(id uint) (*model.Alarm, error) {
	var alarm model.Alarm
	err := r.db.First(&alarm, id).Error
	if err != nil {
		return nil, err
	}
	return &alarm, nil
}

// List 查询报警列表
func (r *AlarmRepository) List(query AlarmQuery) ([]model.Alarm, int64, error) {
	var alarms []model.Alarm
	var total int64

	db := r.db.Model(&model.Alarm{})

	// 过滤条件
	if query.DeviceID != "" {
		db = db.Where("device_id = ?", query.DeviceID)
	}
	if query.Priority != "" {
		db = db.Where("alarm_priority = ?", query.Priority)
	}
	if query.StartTime != "" {
		db = db.Where("created_at >= ?", query.StartTime)
	}
	if query.EndTime != "" {
		db = db.Where("created_at <= ?", query.EndTime)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&alarms).Error; err != nil {
		return nil, 0, err
	}

	return alarms, total, nil
}

// Delete 删除报警
func (r *AlarmRepository) Delete(id uint) error {
	return r.db.Delete(&model.Alarm{}, id).Error
}

// DeleteByDeviceID 删除设备的所有报警
func (r *AlarmRepository) DeleteByDeviceID(deviceID string) error {
	return r.db.Where("device_id = ?", deviceID).Delete(&model.Alarm{}).Error
}

// CountByDeviceID 统计设备报警数量
func (r *AlarmRepository) CountByDeviceID(deviceID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.Alarm{}).Where("device_id = ?", deviceID).Count(&count).Error
	return count, err
}

// DeleteAll 清空所有报警记录
func (r *AlarmRepository) DeleteAll() error {
	return r.db.Where("1 = 1").Delete(&model.Alarm{}).Error
}

// DeleteExpired 删除过期报警记录
func (r *AlarmRepository) DeleteExpired(retentionDays int) error {
	if retentionDays <= 0 {
		return nil // 0 表示永久保留
	}
	expiryTime := time.Now().AddDate(0, 0, -retentionDays)
	return r.db.Where("created_at < ?", expiryTime).Delete(&model.Alarm{}).Error
}

// AlarmQuery 报警查询条件
type AlarmQuery struct {
	DeviceID  string
	Priority  string
	StartTime string
	EndTime   string
	Page      int
	PageSize  int
}
