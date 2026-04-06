package http

import (
	"net/http"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// RecordHandler 历史录像 HTTP 处理器
type RecordHandler struct {
	recordService *service.RecordService
}

// NewRecordHandler 创建历史录像处理器
func NewRecordHandler(recordService *service.RecordService) *RecordHandler {
	return &RecordHandler{recordService: recordService}
}

// RecordListRequest 历史录像查询请求
type RecordListRequest struct {
	DeviceID     string `form:"device_id" binding:"required"`
	ChannelID    string `form:"channel_id" binding:"required"`
	Date         string `form:"date"`    // 查询日期，格式 yyyy-MM-dd
	Timeout      int    `form:"timeout"` // 超时时间(秒)，默认 30
	ForceRefresh bool   `form:"force"`   // 强制刷新缓存，默认 false
}

// RecordListResponse 历史录像查询响应
type RecordListResponse struct {
	DeviceID  string       `json:"device_id"`
	ChannelID string       `json:"channel_id"`
	Total     int          `json:"total"`
	Items     []RecordItem `json:"items"`
	Source    string       `json:"source"`     // 数据来源: cache, db, device
	CachedAt  string       `json:"cached_at"`  // 缓存时间（仅缓存数据有）
	ExpiresAt string       `json:"expires_at"` // 过期时间（仅缓存数据有）
	ExpiresIn int          `json:"expires_in"` // 距离过期剩余秒数（仅缓存数据有）
}

// RecordItem 录像条目
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

// List 查询历史录像列表
// @Summary 查询历史录像
// @Tags 历史录像
// @Produce json
// @Param device_id query string true "设备ID"
// @Param channel_id query string true "通道ID"
// @Param date query string false "查询日期 (yyyy-MM-dd)，默认今天"
// @Param timeout query int false "超时时间(秒)，默认30"
// @Param force query bool false "强制刷新缓存，默认false"
// @Success 200 {object} Response{data=RecordListResponse}
// @Router /api/record/list [get]
func (h *RecordHandler) List(c *gin.Context) {
	var req RecordListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	if h.recordService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "录像查询服务不可用",
		})
		return
	}

	// 解析日期
	var queryDate time.Time
	if req.Date != "" {
		parsed, err := time.ParseInLocation("2006-01-02", req.Date, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    http.StatusBadRequest,
				Message: "日期格式错误，应为 yyyy-MM-dd",
			})
			return
		}
		queryDate = parsed
	} else {
		queryDate = time.Now()
	}

	// 设置超时
	timeout := 30 * time.Second
	if req.Timeout > 30 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	// 计算时间范围（当天 00:00:00 到 23:59:59）
	startTime := time.Date(queryDate.Year(), queryDate.Month(), queryDate.Day(), 0, 0, 0, 0, queryDate.Location())
	endTime := time.Date(queryDate.Year(), queryDate.Month(), queryDate.Day(), 23, 59, 59, 0, queryDate.Location())

	// 使用缓存优先查询
	result, err := h.recordService.QueryRecords(req.DeviceID, req.ChannelID, startTime, endTime, timeout, req.ForceRefresh)
	if err != nil {
		log.Error().Err(err).Str("device_id", req.DeviceID).Str("channel_id", req.ChannelID).Msg("查询历史录像失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "查询历史录像失败: " + err.Error(),
		})
		return
	}

	// 转换响应
	items := make([]RecordItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, RecordItem{
			DeviceID:  item.DeviceID,
			Name:      item.Name,
			Address:   item.Address,
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Secrecy:   item.Secrecy,
			Type:      item.Type,
			FileSize:  item.FileSize,
		})
	}

	// 构建响应，包含缓存信息
	resp := RecordListResponse{
		DeviceID:  req.DeviceID,
		ChannelID: req.ChannelID,
		Total:     len(items),
		Items:     items,
		Source:    string(result.Source),
		ExpiresIn: result.ExpiresIn,
	}

	// 格式化缓存时间（仅缓存数据有）
	if result.CachedAt != nil {
		resp.CachedAt = result.CachedAt.Format("2006-01-02 15:04:05")
	}
	if result.ExpiresAt != nil {
		resp.ExpiresAt = result.ExpiresAt.Format("2006-01-02 15:04:05")
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// FetchRequest 录像拉取请求
type FetchRequest struct {
	DeviceID  string `json:"device_id" binding:"required"`
	ChannelID string `json:"channel_id" binding:"required"`
	Date      string `json:"date"`    // 查询日期，格式 yyyy-MM-dd
	Timeout   int    `json:"timeout"` // 超时时间(秒)，默认 30
}

// FetchResponse 录像拉取响应
type FetchResponse struct {
	DeviceID  string `json:"device_id"`
	ChannelID string `json:"channel_id"`
	Date      string `json:"date"`
	Status    string `json:"status"`     // 拉取状态: pending, fetching, completed, failed
	Message   string `json:"message"`    // 状态消息
	ItemCount int    `json:"item_count"` // 已拉取的录像数量
}

// Fetch 触发录像拉取
// @Summary 触发录像拉取
// @Tags 历史录像
// @Accept json
// @Produce json
// @Param body body FetchRequest true "拉取请求"
// @Success 200 {object} Response{data=FetchResponse}
// @Router /api/record/fetch [post]
func (h *RecordHandler) Fetch(c *gin.Context) {
	var req FetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	if h.recordService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "录像查询服务不可用",
		})
		return
	}

	// 解析日期
	var queryDate time.Time
	if req.Date != "" {
		parsed, err := time.ParseInLocation("2006-01-02", req.Date, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    http.StatusBadRequest,
				Message: "日期格式错误，应为 yyyy-MM-dd",
			})
			return
		}
		queryDate = parsed
	} else {
		queryDate = time.Now()
	}

	// 设置超时
	timeout := 30 * time.Second
	if req.Timeout > 30 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	// 计算时间范围
	startTime := time.Date(queryDate.Year(), queryDate.Month(), queryDate.Day(), 0, 0, 0, 0, queryDate.Location())
	endTime := time.Date(queryDate.Year(), queryDate.Month(), queryDate.Day(), 23, 59, 59, 0, queryDate.Location())

	// 执行拉取（强制刷新）
	result, err := h.recordService.QueryRecords(req.DeviceID, req.ChannelID, startTime, endTime, timeout, true)
	if err != nil {
		log.Error().Err(err).Str("device_id", req.DeviceID).Str("channel_id", req.ChannelID).Msg("拉取录像失败")
		c.JSON(http.StatusOK, Response{
			Code:    http.StatusOK,
			Message: "拉取完成",
			Data: FetchResponse{
				DeviceID:  req.DeviceID,
				ChannelID: req.ChannelID,
				Date:      req.Date,
				Status:    "failed",
				Message:   err.Error(),
				ItemCount: 0,
			},
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "拉取完成",
		Data: FetchResponse{
			DeviceID:  req.DeviceID,
			ChannelID: req.ChannelID,
			Date:      req.Date,
			Status:    "completed",
			Message:   "录像拉取成功",
			ItemCount: result.ItemCount,
		},
	})
}

// FetchStatus 获取拉取状态
// @Summary 获取录像拉取状态
// @Tags 历史录像
// @Produce json
// @Param device_id query string true "设备ID"
// @Param channel_id query string true "通道ID"
// @Param date query string false "查询日期 (yyyy-MM-dd)，默认今天"
// @Success 200 {object} Response{data=FetchResponse}
// @Router /api/record/fetch/status [get]
func (h *RecordHandler) FetchStatus(c *gin.Context) {
	deviceID := c.Query("device_id")
	channelID := c.Query("channel_id")
	date := c.Query("date")

	if deviceID == "" || channelID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "参数错误: device_id 和 channel_id 必填",
		})
		return
	}

	if h.recordService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "录像查询服务不可用",
		})
		return
	}

	// 解析日期
	var queryDate time.Time
	if date != "" {
		parsed, err := time.ParseInLocation("2006-01-02", date, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    http.StatusBadRequest,
				Message: "日期格式错误，应为 yyyy-MM-dd",
			})
			return
		}
		queryDate = parsed
	} else {
		queryDate = time.Now()
	}

	dateStr := queryDate.Format("2006-01-02")

	// 检查缓存状态
	cacheRepo := h.recordService.GetCacheRepo()
	if cacheRepo == nil {
		c.JSON(http.StatusOK, Response{
			Code:    http.StatusOK,
			Message: "success",
			Data: FetchResponse{
				DeviceID:  deviceID,
				ChannelID: channelID,
				Date:      dateStr,
				Status:    "unknown",
				Message:   "缓存服务不可用",
			},
		})
		return
	}

	cache, _, err := cacheRepo.GetCache(deviceID, channelID, dateStr)
	if err != nil {
		c.JSON(http.StatusOK, Response{
			Code:    http.StatusOK,
			Message: "success",
			Data: FetchResponse{
				DeviceID:  deviceID,
				ChannelID: channelID,
				Date:      dateStr,
				Status:    "unknown",
				Message:   "查询缓存状态失败: " + err.Error(),
			},
		})
		return
	}

	if cache == nil {
		c.JSON(http.StatusOK, Response{
			Code:    http.StatusOK,
			Message: "success",
			Data: FetchResponse{
				DeviceID:  deviceID,
				ChannelID: channelID,
				Date:      dateStr,
				Status:    "pending",
				Message:   "无缓存数据，需要拉取",
			},
		})
		return
	}

	status := "completed"
	message := "缓存有效"
	if cache.IsExpired() {
		status = "expired"
		message = "缓存已过期，需要重新拉取"
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: FetchResponse{
			DeviceID:  deviceID,
			ChannelID: channelID,
			Date:      dateStr,
			Status:    status,
			Message:   message,
			ItemCount: cache.ItemCount,
		},
	})
}
