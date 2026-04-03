package http

import (
	"net/http"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"github.com/gin-gonic/gin"
)

// PositionHandler 设备定位 HTTP 处理器
type PositionHandler struct {
	positionService *service.PositionService
}

// NewPositionHandler 创建设备定位处理器
func NewPositionHandler(positionService *service.PositionService) *PositionHandler {
	return &PositionHandler{positionService: positionService}
}

// GetLatestPosition 获取设备最新位置
// GET /api/devices/:id/position
func (h *PositionHandler) GetLatestPosition(c *gin.Context) {
	if h.positionService == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "定位服务未初始化",
		})
		return
	}

	deviceId := c.Param("device_id")

	position, err := h.positionService.GetDeviceLatestPosition(deviceId)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "无定位数据",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    position,
	})
}

// GetPositionHistory 获取设备历史轨迹
// GET /api/devices/:id/positions?start=xxx&end=xxx
func (h *PositionHandler) GetPositionHistory(c *gin.Context) {
	if h.positionService == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "定位服务未初始化",
		})
		return
	}

	deviceId := c.Param("device_id")

	// 解析时间范围
	startTimeStr := c.Query("start")
	endTimeStr := c.Query("end")

	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    http.StatusBadRequest,
				Message: "开始时间格式错误",
			})
			return
		}
	} else {
		startTime = time.Now().AddDate(0, 0, -1) // 默认最近1天
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    http.StatusBadRequest,
				Message: "结束时间格式错误",
			})
			return
		}
	} else {
		endTime = time.Now()
	}

	positions, err := h.positionService.GetDevicePositionHistory(deviceId, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    positions,
	})
}
