package repository

import (
	"git.skcks.cn/Shikong/go-gb28181/internal/model"
	"gorm.io/gorm"
)

// DeviceRepository 设备数据访问层
type DeviceRepository struct {
	db *gorm.DB
}

// NewDeviceRepository 创建设备仓库
func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Create 创建设备
func (r *DeviceRepository) Create(device *model.Device) error {
	return r.db.Create(device).Error
}

// Update 更新设备
func (r *DeviceRepository) Update(device *model.Device) error {
	return r.db.Save(device).Error
}

// UpdateStatus 更新设备状态
func (r *DeviceRepository) UpdateStatus(deviceID, status string) error {
	return r.db.Model(&model.Device{}).
		Where("device_id = ?", deviceID).
		Update("status", status).Error
}

// UpdateKeepaliveTime 更新心跳时间
func (r *DeviceRepository) UpdateKeepaliveTime(deviceID string) error {
	return r.db.Model(&model.Device{}).
		Where("device_id = ?", deviceID).
		Update("last_keepalive_time", gorm.Expr("datetime('now')")).Error
}

// UpdateRegisterTime 更新注册时间
func (r *DeviceRepository) UpdateRegisterTime(deviceID string) error {
	return r.db.Model(&model.Device{}).
		Where("device_id = ?", deviceID).
		Update("last_register_time", gorm.Expr("datetime('now')")).Error
}

// Delete 删除设备
func (r *DeviceRepository) Delete(deviceID string) error {
	return r.db.Where("device_id = ?", deviceID).Delete(&model.Device{}).Error
}

// GetByID 根据 ID 获取设备
func (r *DeviceRepository) GetByID(id uint) (*model.Device, error) {
	var device model.Device
	err := r.db.First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByDeviceID 根据设备 ID 获取设备
func (r *DeviceRepository) GetByDeviceID(deviceID string) (*model.Device, error) {
	var device model.Device
	err := r.db.Where("device_id = ?", deviceID).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// List 获取设备列表
func (r *DeviceRepository) List(offset, limit int) ([]model.Device, int64, error) {
	var devices []model.Device
	var total int64

	if err := r.db.Model(&model.Device{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Model(&model.Device{}).Offset(offset).Limit(limit).Find(&devices).Error; err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

// ListByStatus 根据状态获取设备列表
func (r *DeviceRepository) ListByStatus(status string, offset, limit int) ([]model.Device, int64, error) {
	var devices []model.Device
	var total int64

	query := r.db.Model(&model.Device{}).Where("status = ?", status)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset(offset).Limit(limit).Find(&devices).Error; err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

// CountOnline 统计在线设备数量
func (r *DeviceRepository) CountOnline() (int64, error) {
	var count int64
	err := r.db.Model(&model.Device{}).Where("status = ?", model.DeviceStatusOnline).Count(&count).Error
	return count, err
}

// CountTotal 统计设备总数
func (r *DeviceRepository) CountTotal() (int64, error) {
	var count int64
	err := r.db.Model(&model.Device{}).Count(&count).Error
	return count, err
}

// Exists 检查设备是否存在
func (r *DeviceRepository) Exists(deviceID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Device{}).Where("device_id = ?", deviceID).Count(&count).Error
	return count > 0, err
}

// Upsert 创建或更新设备
func (r *DeviceRepository) Upsert(device *model.Device) error {
	existing, err := r.GetByDeviceID(device.DeviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.Create(device)
		}
		return err
	}

	device.ID = existing.ID
	return r.Update(device)
}
