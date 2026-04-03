package service

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

const recordOutputLayout = "2006-01-02 15:04:05"

// RecordItem 历史录像记录
type RecordItem struct {
	DeviceID  string `json:"device_id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Secrecy   int    `json:"secrecy"`
	Type      string `json:"type"`
	FileSize  int64  `json:"file_size"`
}

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
	client        *sipgo.Client
	deviceService *DeviceService
	localID       string
	localIP       string
	localPort     int

	mu      sync.RWMutex
	queries map[string]*recordQueryContext
}

// NewRecordService 创建历史录像查询服务
func NewRecordService(client *sipgo.Client, deviceService *DeviceService, localID, localIP string, localPort int) *RecordService {
	return &RecordService{
		client:        client,
		deviceService: deviceService,
		localID:       localID,
		localIP:       localIP,
		localPort:     localPort,
		queries:       make(map[string]*recordQueryContext),
	}
}

// QueryRecords 查询历史录像
func (s *RecordService) QueryRecords(deviceID, channelID string, startTime, endTime time.Time, timeout time.Duration) ([]RecordItem, error) {
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

	sn := fmt.Sprintf("%06d", rand.Intn(1000000))
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
