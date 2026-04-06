package repository

import (
	"encoding/json"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"gorm.io/gorm"
)

// RecordCacheRepository 录像缓存数据访问层
type RecordCacheRepository struct {
	db *gorm.DB
}

// NewRecordCacheRepository 创建录像缓存仓库
func NewRecordCacheRepository(db *gorm.DB) *RecordCacheRepository {
	return &RecordCacheRepository{db: db}
}

// SaveCache 保存录像缓存
// 如果已存在相同设备、通道、日期的缓存，则更新；否则创建新记录
func (r *RecordCacheRepository) SaveCache(deviceID, channelID string, queryDate, startTime, endTime string, items []model.RecordItem, expiryHours int) error {
	if expiryHours <= 0 {
		expiryHours = model.DefaultCacheExpiryHours
	}

	// 序列化录像项为 JSON
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return err
	}

	now := time.Now()
	cache := &model.RecordCache{
		DeviceID:        deviceID,
		ChannelID:       channelID,
		QueryDate:       queryDate,
		StartTime:       startTime,
		EndTime:         endTime,
		RecordItemsJSON: string(itemsJSON),
		ItemCount:       len(items),
		FetchedAt:       now,
		ExpiresAt:       now.Add(time.Duration(expiryHours) * time.Hour),
		CacheStatus:     model.RecordCacheStatusValid,
	}

	// 查找是否存在相同设备、通道、日期的缓存
	var existing model.RecordCache
	err = r.db.Where("device_id = ? AND channel_id = ? AND query_date = ?", deviceID, channelID, queryDate).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		return r.db.Create(cache).Error
	}
	if err != nil {
		return err
	}

	// 更新现有记录
	cache.ID = existing.ID
	cache.CreatedAt = existing.CreatedAt
	return r.db.Save(cache).Error
}

// GetCache 获取录像缓存
// 返回缓存记录和解析后的录像项列表
func (r *RecordCacheRepository) GetCache(deviceID, channelID, queryDate string) (*model.RecordCache, []model.RecordItem, error) {
	var cache model.RecordCache
	err := r.db.Where("device_id = ? AND channel_id = ? AND query_date = ?", deviceID, channelID, queryDate).First(&cache).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	// 解析录像项 JSON
	var items []model.RecordItem
	if cache.RecordItemsJSON != "" {
		if err := json.Unmarshal([]byte(cache.RecordItemsJSON), &items); err != nil {
			return &cache, nil, err
		}
	}

	return &cache, items, nil
}

// GetValidCache 获取有效缓存（仅返回未过期的缓存）
func (r *RecordCacheRepository) GetValidCache(deviceID, channelID, queryDate string) (*model.RecordCache, []model.RecordItem, error) {
	cache, items, err := r.GetCache(deviceID, channelID, queryDate)
	if err != nil || cache == nil {
		return nil, nil, err
	}

	// 检查缓存是否过期
	if cache.IsExpired() {
		return nil, nil, nil
	}

	return cache, items, nil
}

// QueryByTimeRange 按时间范围查询缓存
// 返回指定设备和通道在时间范围内的所有缓存记录
func (r *RecordCacheRepository) QueryByTimeRange(deviceID, channelID, startTime, endTime string) ([]model.RecordCache, error) {
	var caches []model.RecordCache
	err := r.db.Where("device_id = ? AND channel_id = ? AND start_time >= ? AND end_time <= ?",
		deviceID, channelID, startTime, endTime).
		Order("query_date ASC").
		Find(&caches).Error
	return caches, err
}

// QueryByTimeRangeWithItems 按时间范围查询缓存并解析录像项
func (r *RecordCacheRepository) QueryByTimeRangeWithItems(deviceID, channelID, startTime, endTime string) ([]model.RecordCache, []model.RecordItem, error) {
	caches, err := r.QueryByTimeRange(deviceID, channelID, startTime, endTime)
	if err != nil {
		return nil, nil, err
	}

	// 合并所有缓存中的录像项
	var allItems []model.RecordItem
	for _, cache := range caches {
		if cache.RecordItemsJSON != "" {
			var items []model.RecordItem
			if err := json.Unmarshal([]byte(cache.RecordItemsJSON), &items); err != nil {
				continue // 跳过解析失败的缓存
			}
			allItems = append(allItems, items...)
		}
	}

	return caches, allItems, nil
}

// DeleteExpiredCache 删除过期缓存
// 返回删除的记录数
func (r *RecordCacheRepository) DeleteExpiredCache() (int64, error) {
	result := r.db.Where("cache_status = ? OR expires_at < ?", model.RecordCacheStatusExpired, time.Now()).
		Delete(&model.RecordCache{})
	return result.RowsAffected, result.Error
}

// MarkExpired 标记缓存为过期
func (r *RecordCacheRepository) MarkExpired(deviceID, channelID, queryDate string) error {
	return r.db.Model(&model.RecordCache{}).
		Where("device_id = ? AND channel_id = ? AND query_date = ?", deviceID, channelID, queryDate).
		Update("cache_status", model.RecordCacheStatusExpired).Error
}

// MarkAllExpired 标记指定设备通道的所有缓存为过期
func (r *RecordCacheRepository) MarkAllExpired(deviceID, channelID string) error {
	return r.db.Model(&model.RecordCache{}).
		Where("device_id = ? AND channel_id = ?", deviceID, channelID).
		Update("cache_status", model.RecordCacheStatusExpired).Error
}

// DeleteByDeviceID 删除设备的所有缓存
func (r *RecordCacheRepository) DeleteByDeviceID(deviceID string) error {
	return r.db.Where("device_id = ?", deviceID).Delete(&model.RecordCache{}).Error
}

// DeleteByChannelID 删除通道的所有缓存
func (r *RecordCacheRepository) DeleteByChannelID(deviceID, channelID string) error {
	return r.db.Where("device_id = ? AND channel_id = ?", deviceID, channelID).Delete(&model.RecordCache{}).Error
}

// CountByDeviceID 统计设备缓存数量
func (r *RecordCacheRepository) CountByDeviceID(deviceID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.RecordCache{}).Where("device_id = ?", deviceID).Count(&count).Error
	return count, err
}

// CountValidByDeviceID 统计设备有效缓存数量
func (r *RecordCacheRepository) CountValidByDeviceID(deviceID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.RecordCache{}).
		Where("device_id = ? AND cache_status = ? AND expires_at > ?", deviceID, model.RecordCacheStatusValid, time.Now()).
		Count(&count).Error
	return count, err
}

// List 获取缓存列表（分页）
func (r *RecordCacheRepository) List(query RecordCacheQuery) ([]model.RecordCache, int64, error) {
	var caches []model.RecordCache
	var total int64

	db := r.db.Model(&model.RecordCache{})

	// 过滤条件
	if query.DeviceID != "" {
		db = db.Where("device_id = ?", query.DeviceID)
	}
	if query.ChannelID != "" {
		db = db.Where("channel_id = ?", query.ChannelID)
	}
	if query.CacheStatus != "" {
		db = db.Where("cache_status = ?", query.CacheStatus)
	}
	if query.StartTime != "" {
		db = db.Where("query_date >= ?", query.StartTime)
	}
	if query.EndTime != "" {
		db = db.Where("query_date <= ?", query.EndTime)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("fetched_at DESC").Offset(offset).Limit(query.PageSize).Find(&caches).Error; err != nil {
		return nil, 0, err
	}

	return caches, total, nil
}

// RecordCacheQuery 缓存查询条件
type RecordCacheQuery struct {
	DeviceID    string
	ChannelID   string
	CacheStatus string
	StartTime   string // 查询日期范围开始
	EndTime     string // 查询日期范围结束
	Page        int
	PageSize    int
}
