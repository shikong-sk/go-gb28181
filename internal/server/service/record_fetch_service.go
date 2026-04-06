package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
)

// RecordFetchService 录像拉取服务
// 主动从设备拉取录像信息并缓存到数据库
type RecordFetchService struct {
	recordService   *RecordService
	deviceService   *DeviceService
	recordCacheRepo *repository.RecordCacheRepository
	channelRepo     *repository.ChannelRepository

	// 后台拉取控制
	fetchInterval    time.Duration
	cacheExpiryHours int
	running          bool
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup

	// 拉取统计
	fetchStats map[string]*FetchStat
	statsMu    sync.RWMutex
}

// FetchStat 拉取统计信息
type FetchStat struct {
	DeviceID     string    `json:"deviceId"`
	ChannelID    string    `json:"channelId"`
	LastFetchAt  time.Time `json:"lastFetchAt"`
	ItemCount    int       `json:"itemCount"`
	SuccessCount int       `json:"successCount"`
	FailCount    int       `json:"failCount"`
	LastError    string    `json:"lastError"`
}

// NewRecordFetchService 创建录像拉取服务
func NewRecordFetchService(
	recordService *RecordService,
	deviceService *DeviceService,
	recordCacheRepo *repository.RecordCacheRepository,
	channelRepo *repository.ChannelRepository,
	cacheExpiryHours int,
) *RecordFetchService {
	if cacheExpiryHours <= 0 {
		cacheExpiryHours = model.DefaultCacheExpiryHours
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &RecordFetchService{
		recordService:    recordService,
		deviceService:    deviceService,
		recordCacheRepo:  recordCacheRepo,
		channelRepo:      channelRepo,
		cacheExpiryHours: cacheExpiryHours,
		ctx:              ctx,
		cancel:           cancel,
		fetchStats:       make(map[string]*FetchStat),
	}
}

// FetchChannelRecords 拉取单个通道的录像
// 查询指定时间范围内的录像并缓存到数据库
func (s *RecordFetchService) FetchChannelRecords(deviceID, channelID string, startTime, endTime time.Time) ([]model.RecordItem, error) {
	log.Info().
		Str("device_id", deviceID).
		Str("channel_id", channelID).
		Str("start_time", startTime.Format(recordOutputLayout)).
		Str("end_time", endTime.Format(recordOutputLayout)).
		Msg("开始拉取通道录像")

	// 检查设备是否在线
	device, err := s.deviceService.GetDevice(deviceID)
	if err != nil {
		s.recordFetchError(deviceID, channelID, fmt.Sprintf("获取设备失败: %v", err))
		return nil, fmt.Errorf("获取设备失败: %w", err)
	}
	if !device.IsOnline() {
		s.recordFetchError(deviceID, channelID, "设备离线")
		return nil, fmt.Errorf("设备离线")
	}

	// 调用 RecordService 查询录像（forceRefresh=true 强制从设备获取）
	result, err := s.recordService.QueryRecords(deviceID, channelID, startTime, endTime, 60*time.Second, true)
	if err != nil {
		s.recordFetchError(deviceID, channelID, fmt.Sprintf("查询录像失败: %v", err))
		return nil, fmt.Errorf("查询录像失败: %w", err)
	}
	items := result.Items

	// 缓存查询结果
	queryDate := startTime.Format("2006-01-02")
	startStr := startTime.Format(recordOutputLayout)
	endStr := endTime.Format(recordOutputLayout)

	if err := s.recordCacheRepo.SaveCache(deviceID, channelID, queryDate, startStr, endStr, items, s.cacheExpiryHours); err != nil {
		log.Error().Err(err).
			Str("device_id", deviceID).
			Str("channel_id", channelID).
			Msg("保存录像缓存失败")
		// 缓存失败不影响返回结果
	} else {
		log.Info().
			Str("device_id", deviceID).
			Str("channel_id", channelID).
			Str("query_date", queryDate).
			Int("item_count", len(items)).
			Msg("录像缓存保存成功")
	}

	// 记录拉取成功统计
	s.recordFetchSuccess(deviceID, channelID, len(items))

	return items, nil
}

// FetchDeviceRecords 拉取设备所有通道的录像
// 查询设备所有在线通道在指定时间范围内的录像
func (s *RecordFetchService) FetchDeviceRecords(deviceID string, startTime, endTime time.Time) (map[string][]model.RecordItem, error) {
	log.Info().
		Str("device_id", deviceID).
		Str("start_time", startTime.Format(recordOutputLayout)).
		Str("end_time", endTime.Format(recordOutputLayout)).
		Msg("开始拉取设备所有通道录像")

	// 检查设备是否在线
	device, err := s.deviceService.GetDevice(deviceID)
	if err != nil {
		return nil, fmt.Errorf("获取设备失败: %w", err)
	}
	if !device.IsOnline() {
		return nil, fmt.Errorf("设备离线")
	}

	// 获取设备所有通道
	channels, _, err := s.deviceService.GetDeviceChannels(deviceID, 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("获取通道列表失败: %w", err)
	}

	results := make(map[string][]model.RecordItem)
	var fetchErrors []error

	for _, channel := range channels {
		// 只拉取在线通道的录像
		if !channel.IsOnline() {
			log.Debug().
				Str("device_id", deviceID).
				Str("channel_id", channel.ChannelID).
				Msg("通道离线，跳过录像拉取")
			continue
		}

		items, err := s.FetchChannelRecords(deviceID, channel.ChannelID, startTime, endTime)
		if err != nil {
			fetchErrors = append(fetchErrors, fmt.Errorf("通道 %s: %w", channel.ChannelID, err))
			continue
		}
		results[channel.ChannelID] = items
	}

	// 如果所有通道都失败，返回汇总错误
	if len(fetchErrors) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("所有通道录像拉取失败: %v", fetchErrors)
	}

	log.Info().
		Str("device_id", deviceID).
		Int("success_channels", len(results)).
		Int("failed_channels", len(fetchErrors)).
		Msg("设备录像拉取完成")

	return results, nil
}

// StartBackgroundFetcher 启动后台定时拉取
// interval: 拉取间隔，默认 1 小时
func (s *RecordFetchService) StartBackgroundFetcher(interval time.Duration) {
	if interval <= 0 {
		interval = 1 * time.Hour
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		log.Warn().Msg("录像后台拉取服务已在运行")
		return
	}
	s.running = true
	s.fetchInterval = interval
	s.mu.Unlock()

	s.wg.Add(1)
	go s.backgroundFetchLoop()

	log.Info().
		Dur("interval", interval).
		Int("cache_expiry_hours", s.cacheExpiryHours).
		Msg("录像后台拉取服务已启动")
}

// StopBackgroundFetcher 停止后台拉取
func (s *RecordFetchService) StopBackgroundFetcher() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()
	log.Info().Msg("录像后台拉取服务已停止")
}

// backgroundFetchLoop 后台拉取循环
func (s *RecordFetchService) backgroundFetchLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.fetchInterval)
	defer ticker.Stop()

	// 启动后立即执行一次
	s.fetchAllDevicesRecords()

	for {
		select {
		case <-s.ctx.Done():
			log.Info().Msg("录像后台拉取循环停止")
			return
		case <-ticker.C:
			s.fetchAllDevicesRecords()
		}
	}
}

// fetchAllDevicesRecords 拉取所有在线设备的录像
func (s *RecordFetchService) fetchAllDevicesRecords() {
	log.Info().Msg("开始批量拉取所有设备录像")

	// 获取所有在线设备
	devices, _, err := s.deviceService.ListOnlineDevices(1, 1000)
	if err != nil {
		log.Error().Err(err).Msg("获取在线设备列表失败")
		return
	}

	// 计算查询时间范围：当天 00:00:00 到当前时间
	now := time.Now()
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endTime := now

	for _, device := range devices {
		// 为每个设备拉取录像
		_, err := s.FetchDeviceRecords(device.DeviceID, startTime, endTime)
		if err != nil {
			log.Error().Err(err).
				Str("device_id", device.DeviceID).
				Msg("拉取设备录像失败")
			continue
		}
	}

	log.Info().
		Int("device_count", len(devices)).
		Msg("批量拉取录像完成")

	// 清理过期缓存
	s.cleanupExpiredCache()
}

// cleanupExpiredCache 清理过期缓存
func (s *RecordFetchService) cleanupExpiredCache() {
	count, err := s.recordCacheRepo.DeleteExpiredCache()
	if err != nil {
		log.Error().Err(err).Msg("清理过期录像缓存失败")
		return
	}
	if count > 0 {
		log.Info().Int64("count", count).Msg("清理过期录像缓存")
	}
}

// FetchRecordsForDateRange 按日期范围拉取录像
// 用于手动触发拉取特定日期的录像
func (s *RecordFetchService) FetchRecordsForDateRange(deviceID, channelID string, startDate, endDate time.Time) ([]model.RecordItem, error) {
	// 将日期转换为时间范围
	startTime := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endTime := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())

	return s.FetchChannelRecords(deviceID, channelID, startTime, endTime)
}

// GetFetchStats 获取拉取统计信息
func (s *RecordFetchService) GetFetchStats(deviceID, channelID string) *FetchStat {
	key := deviceID + ":" + channelID
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()
	return s.fetchStats[key]
}

// GetAllFetchStats 获取所有拉取统计信息
func (s *RecordFetchService) GetAllFetchStats() map[string]*FetchStat {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()
	result := make(map[string]*FetchStat)
	for k, v := range s.fetchStats {
		result[k] = v
	}
	return result
}

// recordFetchSuccess 记录拉取成功统计
func (s *RecordFetchService) recordFetchSuccess(deviceID, channelID string, itemCount int) {
	key := deviceID + ":" + channelID
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	stat, exists := s.fetchStats[key]
	if !exists {
		stat = &FetchStat{
			DeviceID:  deviceID,
			ChannelID: channelID,
		}
		s.fetchStats[key] = stat
	}

	stat.LastFetchAt = time.Now()
	stat.ItemCount = itemCount
	stat.SuccessCount++
	stat.LastError = ""
}

// recordFetchError 记录拉取失败统计
func (s *RecordFetchService) recordFetchError(deviceID, channelID string, errMsg string) {
	key := deviceID + ":" + channelID
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	stat, exists := s.fetchStats[key]
	if !exists {
		stat = &FetchStat{
			DeviceID:  deviceID,
			ChannelID: channelID,
		}
		s.fetchStats[key] = stat
	}

	stat.LastFetchAt = time.Now()
	stat.FailCount++
	stat.LastError = errMsg
}

// IsRunning 检查服务是否在运行
func (s *RecordFetchService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetCacheExpiryHours 获取缓存过期时间（小时）
func (s *RecordFetchService) GetCacheExpiryHours() int {
	return s.cacheExpiryHours
}

// SetCacheExpiryHours 设置缓存过期时间（小时）
func (s *RecordFetchService) SetCacheExpiryHours(hours int) {
	if hours <= 0 {
		hours = model.DefaultCacheExpiryHours
	}
	s.cacheExpiryHours = hours
}
