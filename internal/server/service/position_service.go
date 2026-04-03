package service

import (
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
)

// PositionService 设备定位服务层
type PositionService struct {
	positionRepo  repository.PositionRepository
	retentionDays int // 定位数据保留天数
}

// NewPositionService 创建设备定位服务
func NewPositionService(positionRepo repository.PositionRepository, retentionDays int) *PositionService {
	if retentionDays <= 0 {
		retentionDays = 7 // 默认保留 7 天
	}
	return &PositionService{
		positionRepo:  positionRepo,
		retentionDays: retentionDays,
	}
}

// OnMobilePositionReceived 处理设备定位上报
func (s *PositionService) OnMobilePositionReceived(notify *manscdp.MobilePositionNotify) error {
	// 1. 解析 GPSTime
	gpsTime, err := parseGPSTime(notify.GPSTime)
	if err != nil {
		log.Error().Err(err).Str("device_id", notify.DeviceID).Str("gps_time", notify.GPSTime).Msg("解析 GPS 时间失败")
		return err
	}

	// 2. 构建 DevicePosition
	position := &model.DevicePosition{
		DeviceID:  notify.DeviceID,
		Longitude: notify.Longitude,
		Latitude:  notify.Latitude,
		Speed:     notify.Speed,
		Direction: notify.Direction,
		Altitude:  notify.Altitude,
		GPSTime:   gpsTime,
	}

	// 3. 保存到数据库
	err = s.positionRepo.SavePosition(position)
	if err != nil {
		log.Error().Err(err).Str("device_id", notify.DeviceID).Msg("保存设备定位失败")
		return err
	}

	log.Info().
		Str("device_id", notify.DeviceID).
		Float64("longitude", notify.Longitude).
		Float64("latitude", notify.Latitude).
		Msg("设备定位已保存")

	// 4. 清理超过保留期的历史记录
	cutoffTime := time.Now().AddDate(0, 0, -s.retentionDays)
	if err := s.positionRepo.DeleteOldPositions(cutoffTime); err != nil {
		log.Error().Err(err).Int("retention_days", s.retentionDays).Msg("清理过期定位数据失败")
		// 不返回错误，因为主流程已成功
	}

	return nil
}

// GetDeviceLatestPosition 获取设备最新位置
func (s *PositionService) GetDeviceLatestPosition(deviceId string) (*model.DevicePosition, error) {
	return s.positionRepo.GetLatestPosition(deviceId)
}

// GetDevicePositionHistory 获取设备历史轨迹
func (s *PositionService) GetDevicePositionHistory(deviceId string, startTime, endTime time.Time) ([]model.DevicePosition, error) {
	return s.positionRepo.GetPositions(deviceId, startTime, endTime)
}

// parseGPSTime 解析 GPS 时间
// GB28181 的 GPSTime 格式通常为 ISO 8601 格式：2006-01-02T15:04:05
func parseGPSTime(gpsTimeStr string) (time.Time, error) {
	// 尝试解析 GB28181 标准格式
	layout := "2006-01-02T15:04:05"
	t, err := time.Parse(layout, gpsTimeStr)
	if err == nil {
		return t, nil
	}

	// 尝试其他常见格式
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, gpsTimeStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, err
}
