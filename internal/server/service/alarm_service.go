package service

import (
	"git.skcks.cn/Shikong/go-gb28181/internal/server/config"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
)

// AlarmService 报警服务层
type AlarmService struct {
	alarmRepo *repository.AlarmRepository
	config    *config.AlarmConfig
}

// NewAlarmService 创建报警服务
func NewAlarmService(alarmRepo *repository.AlarmRepository, cfg *config.AlarmConfig) *AlarmService {
	if cfg == nil {
		cfg = &config.AlarmConfig{Enabled: true, RetentionDays: 3}
	}
	return &AlarmService{
		alarmRepo: alarmRepo,
		config:    cfg,
	}
}

// IsEnabled 检查报警记录是否启用
func (s *AlarmService) IsEnabled() bool {
	return s.config.Enabled
}

// SaveAlarm 保存报警信息
func (s *AlarmService) SaveAlarm(alarm *model.Alarm) error {
	// 检查开关
	if !s.config.Enabled {
		log.Debug().Str("device_id", alarm.DeviceID).Msg("报警记录已禁用，跳过保存")
		return nil
	}

	if err := s.alarmRepo.Create(alarm); err != nil {
		log.Error().Err(err).Str("device_id", alarm.DeviceID).Msg("保存报警失败")
		return err
	}
	log.Info().Str("device_id", alarm.DeviceID).Str("priority", alarm.AlarmPriority).Msg("报警已保存")
	return nil
}

// GetAlarm 获取报警详情
func (s *AlarmService) GetAlarm(id uint) (*model.Alarm, error) {
	return s.alarmRepo.GetByID(id)
}

// ListAlarms 查询报警列表
func (s *AlarmService) ListAlarms(deviceID, priority, startTime, endTime string, page, pageSize int) ([]model.Alarm, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := repository.AlarmQuery{
		DeviceID:  deviceID,
		Priority:  priority,
		StartTime: startTime,
		EndTime:   endTime,
		Page:      page,
		PageSize:  pageSize,
	}

	return s.alarmRepo.List(query)
}

// DeleteAlarm 删除报警
func (s *AlarmService) DeleteAlarm(id uint) error {
	return s.alarmRepo.Delete(id)
}

// DeleteAlarmsByDevice 删除设备的所有报警
func (s *AlarmService) DeleteAlarmsByDevice(deviceID string) error {
	return s.alarmRepo.DeleteByDeviceID(deviceID)
}

// DeleteAll 清空所有报警记录
func (s *AlarmService) DeleteAll() error {
	return s.alarmRepo.DeleteAll()
}

// CleanupExpired 清理过期报警记录
func (s *AlarmService) CleanupExpired() error {
	if s.config.RetentionDays <= 0 {
		log.Debug().Msg("报警记录永久保留，跳过清理")
		return nil
	}

	err := s.alarmRepo.DeleteExpired(s.config.RetentionDays)
	if err != nil {
		log.Error().Err(err).Int("retention_days", s.config.RetentionDays).Msg("清理过期报警失败")
		return err
	}

	log.Info().Int("retention_days", s.config.RetentionDays).Msg("过期报警清理完成")
	return nil
}

// GetRetentionDays 获取保留天数
func (s *AlarmService) GetRetentionDays() int {
	return s.config.RetentionDays
}
