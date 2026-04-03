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
	DeviceID  string `form:"device_id" binding:"required"`
	ChannelID string `form:"channel_id" binding:"required"`
	Date      string `form:"date"`    // 查询日期，格式 yyyy-MM-dd
	Timeout   int    `form:"timeout"` // 超时时间(秒)，默认 30
}

// RecordListResponse 历史录像查询响应
type RecordListResponse struct {
	DeviceID  string       `json:"device_id"`
	ChannelID string       `json:"channel_id"`
	Total     int          `json:"total"`
	Items     []RecordItem `json:"items"`
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

	items, err := h.recordService.QueryRecords(req.DeviceID, req.ChannelID, startTime, endTime, timeout)
	if err != nil {
		log.Error().Err(err).Str("device_id", req.DeviceID).Str("channel_id", req.ChannelID).Msg("查询历史录像失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "查询历史录像失败: " + err.Error(),
		})
		return
	}

	// 转换响应
	result := make([]RecordItem, 0, len(items))
	for _, item := range items {
		result = append(result, RecordItem{
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

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: RecordListResponse{
			DeviceID:  req.DeviceID,
			ChannelID: req.ChannelID,
			Total:     len(result),
			Items:     result,
		},
	})
}
