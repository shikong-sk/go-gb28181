package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// DataSource 数据来源标识
type DataSource string

const (
	DataSourceCache  DataSource = "cache"  // 来自缓存
	DataSourceDB     DataSource = "db"     // 来自数据库（缓存已存储但标记为过期）
	DataSourceDevice DataSource = "device" // 来自设备实时查询
)

// RecordQueryResult 录像查询结果，包含来源信息
type RecordQueryResult struct {
	Items     []model.RecordItem `json:"items"`
	Source    DataSource         `json:"source"`     // 数据来源
	CachedAt  *time.Time         `json:"cached_at"`  // 缓存时间（仅缓存数据有）
	ExpiresAt *time.Time         `json:"expires_at"` // 过期时间（仅缓存数据有）
	ExpiresIn int                `json:"expires_in"` // 距离过期剩余秒数（仅缓存数据有）
	ItemCount int                `json:"item_count"` // 录像项数量
}

const recordOutputLayout = "2006-01-02 15:04:05"

// 使用 model.RecordItem 作为录像项类型
type RecordItem = model.RecordItem

type recordQueryContext struct {
	deviceID  string
	channelID string
	total     int
	items     []manscdp.RecordInfoItem
	done      bool
	resultCh  chan []RecordItem
}

// RecordService 历史录像查询服务
type RecordService struct {
	client           *sipgo.Client
	deviceService    *DeviceService
	cacheRepo        *repository.RecordCacheRepository // 录像缓存仓库
	localID          string
	localIP          string
	localPort        int
	cacheExpiryHours int // 缓存有效期（小时）

	mu      sync.RWMutex
	queries map[string]*recordQueryContext
}

// NewRecordService 创建历史录像查询服务
func NewRecordService(client *sipgo.Client, deviceService *DeviceService, cacheRepo *repository.RecordCacheRepository, localID, localIP string, localPort int) *RecordService {
	cacheExpiryHours := model.DefaultCacheExpiryHours
	return &RecordService{
		client:           client,
		deviceService:    deviceService,
		cacheRepo:        cacheRepo,
		localID:          localID,
		localIP:          localIP,
		localPort:        localPort,
		cacheExpiryHours: cacheExpiryHours,
		queries:          make(map[string]*recordQueryContext),
	}
}

// SetCacheExpiry 设置缓存有效期
func (s *RecordService) SetCacheExpiry(hours int) {
	if hours > 0 {
		s.cacheExpiryHours = hours
	}
}

// GetCacheRepo 获取缓存仓库（供外部查询缓存状态）
func (s *RecordService) GetCacheRepo() *repository.RecordCacheRepository {
	return s.cacheRepo
}

// QueryRecords 查询历史录像（缓存优先策略）
// 参数：
//   - deviceID: 设备ID
//   - channelID: 通道ID
//   - startTime: 开始时间
//   - endTime: 结束时间
//   - timeout: 超时时间
//   - forceRefresh: 是否强制刷新（跳过缓存）
func (s *RecordService) QueryRecords(deviceID, channelID string, startTime, endTime time.Time, timeout time.Duration, forceRefresh bool) (*RecordQueryResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("SIP 客户端未初始化")
	}
	if endTime.Before(startTime) {
		return nil, fmt.Errorf("结束时间不能早于开始时间")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 格式化查询日期
	queryDate := startTime.Format("2006-01-02")
	startTimeStr := startTime.Format(recordOutputLayout)
	endTimeStr := endTime.Format(recordOutputLayout)

	// 缓存优先策略：先查缓存
	if !forceRefresh && s.cacheRepo != nil {
		cache, items, err := s.cacheRepo.GetValidCache(deviceID, channelID, queryDate)
		if err != nil {
			log.Warn().Err(err).Str("device_id", deviceID).Str("channel_id", channelID).Msg("查询缓存失败，继续回源查询")
		} else if cache != nil && items != nil {
			// 缓存有效，直接返回
			log.Info().Str("device_id", deviceID).Str("channel_id", channelID).Str("query_date", queryDate).Int("count", len(items)).Msg("使用缓存数据")
			now := time.Now()
			expiresIn := int(cache.ExpiresAt.Sub(now).Seconds())
			if expiresIn < 0 {
				expiresIn = 0
			}
			return &RecordQueryResult{
				Items:     items,
				Source:    DataSourceCache,
				CachedAt:  &cache.FetchedAt,
				ExpiresAt: &cache.ExpiresAt,
				ExpiresIn: expiresIn,
				ItemCount: len(items),
			}, nil
		}
	}

	// 缓存不存在或过期，回源设备查询
	log.Info().Str("device_id", deviceID).Str("channel_id", channelID).Str("query_date", queryDate).Msg("缓存无效，回源设备查询")

	items, err := s.queryFromDevice(deviceID, channelID, startTime, endTime, timeout)
	if err != nil {
		return nil, err
	}

	// 保存到缓存
	if s.cacheRepo != nil && len(items) > 0 {
		if err := s.cacheRepo.SaveCache(deviceID, channelID, queryDate, startTimeStr, endTimeStr, items, s.cacheExpiryHours); err != nil {
			log.Warn().Err(err).Str("device_id", deviceID).Str("channel_id", channelID).Msg("保存缓存失败")
		} else {
			log.Info().Str("device_id", deviceID).Str("channel_id", channelID).Int("count", len(items)).Msg("录像缓存已保存")
		}
	}

	return &RecordQueryResult{
		Items:     items,
		Source:    DataSourceDevice,
		ItemCount: len(items),
	}, nil
}

// QueryRecordsSimple 简单查询（不区分来源，兼容旧接口）
func (s *RecordService) QueryRecordsSimple(deviceID, channelID string, startTime, endTime time.Time, timeout time.Duration) ([]RecordItem, error) {
	result, err := s.QueryRecords(deviceID, channelID, startTime, endTime, timeout, false)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// queryFromDevice 从设备实时查询录像
func (s *RecordService) queryFromDevice(deviceID, channelID string, startTime, endTime time.Time, timeout time.Duration) ([]RecordItem, error) {
	if s.client == nil {
		return nil, fmt.Errorf("SIP 客户端未初始化")
	}
	if endTime.Before(startTime) {
		return nil, fmt.Errorf("结束时间不能早于开始时间")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	device, err := s.deviceService.GetDevice(deviceID)
	if err != nil {
		return nil, fmt.Errorf("获取设备失败: %w", err)
	}
	if _, err := s.deviceService.GetChannel(deviceID, channelID); err != nil {
		return nil, fmt.Errorf("获取通道失败: %w", err)
	}

	sn := utils.GenerateSN()
	query := manscdp.NewRecordInfoReq(sn, channelID, startTime, endTime)
	body, err := utils.XMLMarshal(query, "gbk")
	if err != nil {
		return nil, fmt.Errorf("序列化历史录像查询失败: %w", err)
	}

	key := s.buildQueryKey(channelID, sn)
	ctx := &recordQueryContext{
		deviceID:  deviceID,
		channelID: channelID,
		items:     make([]manscdp.RecordInfoItem, 0),
		resultCh:  make(chan []RecordItem, 1),
	}

	s.mu.Lock()
	s.queries[key] = ctx
	s.mu.Unlock()
	defer s.deleteQuery(key)

	target := sip.Uri{
		User: deviceID,
		Host: device.IP,
		Port: device.Port,
	}
	req := sip.NewRequest(sip.MESSAGE, target)
	from := sip.NewHeader("From", fmt.Sprintf("<sip:%s@%s:%d>;tag=%s", s.localID, s.localIP, s.localPort, utils.GenerateFromTag()))
	to := sip.NewHeader("To", fmt.Sprintf("<sip:%s@%s:%d>", deviceID, device.IP, device.Port))
	callID := sip.NewHeader("Call-ID", fmt.Sprintf("%d@%s", time.Now().UnixNano(), s.localIP))
	cseq := sip.NewHeader("CSeq", "1 MESSAGE")
	contentType := sip.NewHeader("Content-Type", "Application/MANSCDP+xml")
	req.AppendHeader(from)
	req.AppendHeader(to)
	req.AppendHeader(callID)
	req.AppendHeader(cseq)
	req.AppendHeader(contentType)
	req.SetBody(body)

	requestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := s.client.TransactionRequest(requestCtx, req)
	if err != nil {
		return nil, fmt.Errorf("发送历史录像查询失败: %w", err)
	}
	defer tx.Terminate()

	log.Info().Str("device_id", deviceID).Str("channel_id", channelID).Str("sn", sn).Msg("历史录像查询请求已发送")

	select {
	case result := <-ctx.resultCh:
		return result, nil
	case <-time.After(timeout):
		partial := s.snapshotItems(key)
		if len(partial) > 0 {
			log.Warn().Str("device_id", deviceID).Str("channel_id", channelID).Str("sn", sn).Int("count", len(partial)).Msg("历史录像查询超时，返回部分结果")
			return partial, nil
		}
		return nil, fmt.Errorf("历史录像查询超时")
	}
}

// HandleRecordInfoResponse 处理历史录像查询响应
func (s *RecordService) HandleRecordInfoResponse(resp *manscdp.RecordInfoResp) {
	if resp == nil {
		return
	}

	key := s.buildQueryKey(resp.DeviceID, resp.SN)

	s.mu.Lock()
	ctx, exists := s.queries[key]
	if !exists || ctx.done {
		s.mu.Unlock()
		return
	}

	if total := resp.Total(); total > ctx.total {
		ctx.total = total
	}
	if resp.RecordList != nil && len(resp.RecordList.Item) > 0 {
		ctx.items = append(ctx.items, resp.RecordList.Item...)
	}

	shouldComplete := ctx.total > 0 && len(ctx.items) >= ctx.total
	items := make([]manscdp.RecordInfoItem, len(ctx.items))
	copy(items, ctx.items)
	if shouldComplete {
		ctx.done = true
	}
	s.mu.Unlock()

	log.Info().
		Str("channel_id", resp.DeviceID).
		Str("sn", resp.SN).
		Int("received", len(items)).
		Int("total", ctx.total).
		Msg("收到历史录像查询响应")

	if shouldComplete {
		result := convertRecordItems(items)
		select {
		case ctx.resultCh <- result:
		default:
		}
	}
}

func (s *RecordService) buildQueryKey(channelID, sn string) string {
	return channelID + ":" + sn
}

func (s *RecordService) deleteQuery(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.queries, key)
}

func (s *RecordService) snapshotItems(key string) []RecordItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ctx, exists := s.queries[key]
	if !exists {
		return nil
	}
	items := make([]manscdp.RecordInfoItem, len(ctx.items))
	copy(items, ctx.items)
	return convertRecordItems(items)
}

func convertRecordItems(items []manscdp.RecordInfoItem) []RecordItem {
	result := make([]RecordItem, 0, len(items))
	for _, item := range items {
		secrecy, _ := strconv.Atoi(item.Secrecy)
		fileSize, _ := strconv.ParseInt(item.FileSize, 10, 64)
		result = append(result, RecordItem{
			DeviceID:  item.DeviceID,
			Name:      item.Name,
			Address:   item.Address,
			StartTime: normalizeRecordTime(item.StartTime),
			EndTime:   normalizeRecordTime(item.EndTime),
			Secrecy:   secrecy,
			Type:      item.Type,
			FileSize:  fileSize,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return parseRecordTime(result[i].StartTime).Before(parseRecordTime(result[j].StartTime))
	})
	return result
}

func normalizeRecordTime(value string) string {
	parsed := parseRecordTime(value)
	if parsed.IsZero() {
		return value
	}
	return parsed.Format(recordOutputLayout)
}

func parseRecordTime(value string) time.Time {
	layouts := []string{
		recordOutputLayout,
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
