package service

import (
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"gorm.io/gorm"
)

// DeviceService 设备服务层
type DeviceService struct {
	deviceRepo  *repository.DeviceRepository
	channelRepo *repository.ChannelRepository
}

// NewDeviceService 创建设备服务
func NewDeviceService(deviceRepo *repository.DeviceRepository, channelRepo *repository.ChannelRepository) *DeviceService {
	return &DeviceService{
		deviceRepo:  deviceRepo,
		channelRepo: channelRepo,
	}
}

// OnDeviceRegister 设备注册事件处理
func (s *DeviceService) OnDeviceRegister(deviceID, ip string, port int) error {
	// 查询设备是否存在
	_, err := s.deviceRepo.GetByDeviceID(deviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 设备不存在，创建新设备
			device := &model.Device{
				DeviceID:          deviceID,
				IP:                ip,
				Port:              port,
				Status:            model.DeviceStatusOnline,
				LastRegisterTime:  time.Now(),
				LastKeepaliveTime: time.Now(),
			}

			if err := s.deviceRepo.Create(device); err != nil {
				log.Error().Err(err).Str("device_id", deviceID).Msg("创建设备失败")
				return err
			}

			log.Info().Str("device_id", deviceID).Str("ip", ip).Int("port", port).Msg("新设备注册成功")
			return nil
		}
		return err
	}

	// 设备已存在，更新状态
	device := &model.Device{
		DeviceID:          deviceID,
		IP:                ip,
		Port:              port,
		Status:            model.DeviceStatusOnline,
		LastRegisterTime:  time.Now(),
		LastKeepaliveTime: time.Now(),
	}

	if err := s.deviceRepo.Upsert(device); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("更新设备注册信息失败")
		return err
	}

	log.Info().Str("device_id", deviceID).Str("ip", ip).Int("port", port).Msg("设备重新注册成功")
	return nil
}

// OnDeviceKeepalive 设备心跳事件处理
func (s *DeviceService) OnDeviceKeepalive(deviceID string) error {
	// 检查设备是否存在
	exists, err := s.deviceRepo.Exists(deviceID)
	if err != nil {
		return err
	}

	if !exists {
		log.Warn().Str("device_id", deviceID).Msg("收到未知设备的心跳")
		return nil
	}

	// 更新心跳时间和状态
	if err := s.deviceRepo.UpdateKeepaliveTime(deviceID); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("更新心跳时间失败")
		return err
	}

	// 同时更新状态为在线
	if err := s.deviceRepo.UpdateStatus(deviceID, model.DeviceStatusOnline); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("更新设备状态失败")
		return err
	}

	log.Debug().Str("device_id", deviceID).Msg("设备心跳更新成功")
	return nil
}

// OnDeviceOffline 设备离线事件处理
func (s *DeviceService) OnDeviceOffline(deviceID string) error {
	if err := s.deviceRepo.UpdateStatus(deviceID, model.DeviceStatusOffline); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("更新设备离线状态失败")
		return err
	}

	log.Info().Str("device_id", deviceID).Msg("设备离线")
	return nil
}

// OnCatalogReceived 设备目录收到事件处理
func (s *DeviceService) OnCatalogReceived(deviceID string, channels []model.Channel) error {
	// 批量插入/更新通道
	if len(channels) > 0 {
		if err := s.channelRepo.UpsertBatch(channels); err != nil {
			log.Error().Err(err).Str("device_id", deviceID).Int("count", len(channels)).Msg("保存通道失败")
			return err
		}
	}

	// 更新设备通道数
	channelCount := len(channels)
	device, err := s.deviceRepo.GetByDeviceID(deviceID)
	if err != nil {
		return err
	}

	if device != nil {
		device.ChannelCount = channelCount
		if err := s.deviceRepo.Update(device); err != nil {
			log.Error().Err(err).Str("device_id", deviceID).Msg("更新通道数量失败")
			return err
		}
	}

	log.Info().Str("device_id", deviceID).Int("count", channelCount).Msg("设备目录同步成功")
	return nil
}

// GetDevice 获取设备
func (s *DeviceService) GetDevice(deviceID string) (*model.Device, error) {
	return s.deviceRepo.GetByDeviceID(deviceID)
}

// ListDevices 获取设备列表
func (s *DeviceService) ListDevices(page, pageSize int) ([]model.Device, int64, error) {
	offset := (page - 1) * pageSize
	return s.deviceRepo.List(offset, pageSize)
}

// ListOnlineDevices 获取在线设备列表
func (s *DeviceService) ListOnlineDevices(page, pageSize int) ([]model.Device, int64, error) {
	offset := (page - 1) * pageSize
	return s.deviceRepo.ListByStatus(model.DeviceStatusOnline, offset, pageSize)
}

// GetDeviceChannels 获取设备通道列表
func (s *DeviceService) GetDeviceChannels(deviceID string, page, pageSize int) ([]model.Channel, int64, error) {
	offset := (page - 1) * pageSize
	return s.channelRepo.ListByDeviceID(deviceID, offset, pageSize)
}

// GetChannel 获取单个通道
func (s *DeviceService) GetChannel(deviceID, channelID string) (*model.Channel, error) {
	return s.channelRepo.GetByChannelID(channelID, deviceID)
}

// DeleteDevice 删除设备
func (s *DeviceService) DeleteDevice(deviceID string) error {
	// 删除设备的所有通道
	if err := s.channelRepo.DeleteByDeviceID(deviceID); err != nil {
		return err
	}

	// 删除设备
	return s.deviceRepo.Delete(deviceID)
}

// GetDeviceStats 获取设备统计
func (s *DeviceService) GetDeviceStats() (total, online int64, err error) {
	total, err = s.deviceRepo.CountTotal()
	if err != nil {
		return
	}
	online, err = s.deviceRepo.CountOnline()
	return
}

// CheckOfflineDevices 检查离线设备（心跳超时）
// timeoutMinutes: 心跳超时阈值（分钟），默认 3 分钟（GB28181 心跳间隔 30s，允许丢失 5 次）
func (s *DeviceService) CheckOfflineDevices(timeoutMinutes int) error {
	if timeoutMinutes <= 0 {
		timeoutMinutes = 3 // 默认 3 分钟
	}

	// 获取心跳超时的设备
	staleDevices, err := s.deviceRepo.ListStale(timeoutMinutes)
	if err != nil {
		log.Error().Err(err).Msg("查询超时设备失败")
		return err
	}

	// 将这些设备标记为离线
	for _, device := range staleDevices {
		if err := s.OnDeviceOffline(device.DeviceID); err != nil {
			log.Error().Err(err).Str("device_id", device.DeviceID).Msg("标记设备离线失败")
		} else {
			log.Info().Str("device_id", device.DeviceID).Msg("设备心跳超时，标记为离线")
		}
	}

	return nil
}
