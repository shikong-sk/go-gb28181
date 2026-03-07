package repository

import (
	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"gorm.io/gorm"
)

// ChannelRepository 通道数据访问层
type ChannelRepository struct {
	db *gorm.DB
}

// NewChannelRepository 创建通道仓库
func NewChannelRepository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

// Create 创建通道
func (r *ChannelRepository) Create(channel *model.Channel) error {
	return r.db.Create(channel).Error
}

// CreateBatch 批量创建通道
func (r *ChannelRepository) CreateBatch(channels []model.Channel) error {
	if len(channels) == 0 {
		return nil
	}
	return r.db.Create(&channels).Error
}

// Update 更新通道
func (r *ChannelRepository) Update(channel *model.Channel) error {
	return r.db.Save(channel).Error
}

// UpdateStatus 更新通道状态
func (r *ChannelRepository) UpdateStatus(channelID, deviceID, status string) error {
	return r.db.Model(&model.Channel{}).
		Where("channel_id = ? AND device_id = ?", channelID, deviceID).
		Update("status", status).Error
}

// Delete 删除通道
func (r *ChannelRepository) Delete(channelID, deviceID string) error {
	return r.db.Where("channel_id = ? AND device_id = ?", channelID, deviceID).
		Delete(&model.Channel{}).Error
}

// DeleteByDeviceID 删除设备的所有通道
func (r *ChannelRepository) DeleteByDeviceID(deviceID string) error {
	return r.db.Where("device_id = ?", deviceID).Delete(&model.Channel{}).Error
}

// GetByID 根据 ID 获取通道
func (r *ChannelRepository) GetByID(id uint) (*model.Channel, error) {
	var channel model.Channel
	err := r.db.First(&channel, id).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// GetByChannelID 根据通道 ID 获取通道
func (r *ChannelRepository) GetByChannelID(channelID, deviceID string) (*model.Channel, error) {
	var channel model.Channel
	err := r.db.Where("channel_id = ? AND device_id = ?", channelID, deviceID).
		First(&channel).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// ListByDeviceID 获取设备的通道列表
func (r *ChannelRepository) ListByDeviceID(deviceID string, offset, limit int) ([]model.Channel, int64, error) {
	var channels []model.Channel
	var total int64

	query := r.db.Model(&model.Channel{}).Where("device_id = ?", deviceID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset(offset).Limit(limit).Find(&channels).Error; err != nil {
		return nil, 0, err
	}

	return channels, total, nil
}

// List 获取所有通道列表
func (r *ChannelRepository) List(offset, limit int) ([]model.Channel, int64, error) {
	var channels []model.Channel
	var total int64

	if err := r.db.Model(&model.Channel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Offset(offset).Limit(limit).Find(&channels).Error; err != nil {
		return nil, 0, err
	}

	return channels, total, nil
}

// CountByDeviceID 统计设备的通道数量
func (r *ChannelRepository) CountByDeviceID(deviceID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.Channel{}).Where("device_id = ?", deviceID).Count(&count).Error
	return count, err
}

// CountOnlineByDeviceID 统计设备的在线通道数量
func (r *ChannelRepository) CountOnlineByDeviceID(deviceID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.Channel{}).
		Where("device_id = ? AND status = ?", deviceID, model.ChannelStatusOnline).
		Count(&count).Error
	return count, err
}

// Exists 检查通道是否存在
func (r *ChannelRepository) Exists(channelID, deviceID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Channel{}).
		Where("channel_id = ? AND device_id = ?", channelID, deviceID).
		Count(&count).Error
	return count > 0, err
}

// Upsert 创建或更新通道
func (r *ChannelRepository) Upsert(channel *model.Channel) error {
	existing, err := r.GetByChannelID(channel.ChannelID, channel.DeviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.Create(channel)
		}
		return err
	}

	channel.ID = existing.ID
	return r.Update(channel)
}

// UpsertBatch 批量创建或更新通道
func (r *ChannelRepository) UpsertBatch(channels []model.Channel) error {
	if len(channels) == 0 {
		return nil
	}

	// 使用事务
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, channel := range channels {
			if err := r.upsertInTransaction(tx, &channel); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ChannelRepository) upsertInTransaction(tx *gorm.DB, channel *model.Channel) error {
	var existing model.Channel
	err := tx.Where("channel_id = ? AND device_id = ?", channel.ChannelID, channel.DeviceID).
		First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return tx.Create(channel).Error
		}
		return err
	}

	channel.ID = existing.ID
	return tx.Save(channel).Error
}
