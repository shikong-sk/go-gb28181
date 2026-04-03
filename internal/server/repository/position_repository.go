package repository

import (
	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"time"

	"gorm.io/gorm"
)

// PositionRepository 定位数据访问层接口
type PositionRepository interface {
	SavePosition(position *model.DevicePosition) error
	GetLatestPosition(deviceId string) (*model.DevicePosition, error)
	GetPositions(deviceId string, startTime, endTime time.Time) ([]model.DevicePosition, error)
	DeleteOldPositions(beforeTime time.Time) error
}

// positionRepository 定位数据仓库实现
type positionRepository struct {
	db *gorm.DB
}

// NewPositionRepository 创建设位仓库
func NewPositionRepository(db *gorm.DB) PositionRepository {
	return &positionRepository{db: db}
}

// SavePosition 保存定位数据
func (r *positionRepository) SavePosition(position *model.DevicePosition) error {
	return r.db.Create(position).Error
}

// GetLatestPosition 获取设备最新定位
func (r *positionRepository) GetLatestPosition(deviceId string) (*model.DevicePosition, error) {
	var position model.DevicePosition
	err := r.db.Where("device_id = ?", deviceId).
		Order("gps_time DESC").
		First(&position).Error
	if err != nil {
		return nil, err
	}
	return &position, nil
}

// GetPositions 获取设备定位历史
func (r *positionRepository) GetPositions(deviceId string, startTime, endTime time.Time) ([]model.DevicePosition, error) {
	var positions []model.DevicePosition
	err := r.db.Where("device_id = ? AND gps_time >= ? AND gps_time <= ?", deviceId, startTime, endTime).
		Order("gps_time ASC").
		Find(&positions).Error
	return positions, err
}

// DeleteOldPositions 删除旧定位数据
func (r *positionRepository) DeleteOldPositions(beforeTime time.Time) error {
	return r.db.Where("created_at < ?", beforeTime).Delete(&model.DevicePosition{}).Error
}
