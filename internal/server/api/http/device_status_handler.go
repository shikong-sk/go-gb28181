package http

import (
	"net/http"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// DeviceStatusHandler 设备状态 HTTP 处理器
type DeviceStatusHandler struct {
	statusService *service.DeviceStatusService
	deviceService *service.DeviceService
}

// NewDeviceStatusHandler 创建设备状态处理器
func NewDeviceStatusHandler(statusService *service.DeviceStatusService, deviceService *service.DeviceService) *DeviceStatusHandler {
	return &DeviceStatusHandler{
		statusService: statusService,
		deviceService: deviceService,
	}
}

// GetDeviceStatus 获取设备状态（从数据库缓存）
func (h *DeviceStatusHandler) GetDeviceStatus(c *gin.Context) {
	if h.deviceService == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "设备服务未初始化",
		})
		return
	}

	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "设备ID不能为空",
		})
		return
	}

	// 从数据库获取设备当前状态
	device, err := h.deviceService.GetDevice(deviceID)
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("获取设备状态失败")
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "设备不存在",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: gin.H{
			"device_id":           device.DeviceID,
			"online":              device.IsOnline(),
			"status":              device.Status,
			"ip":                  device.IP,
			"port":                device.Port,
			"last_register_time":  device.LastRegisterTime,
			"last_keepalive_time": device.LastKeepaliveTime,
		},
	})
}

// QueryDeviceStatus 查询设备状态（发送 SIP MESSAGE）
func (h *DeviceStatusHandler) QueryDeviceStatus(c *gin.Context) {
	if h.statusService == nil || h.deviceService == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "服务未初始化",
		})
		return
	}

	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "设备ID不能为空",
		})
		return
	}

	// 检查设备是否存在
	device, err := h.deviceService.GetDevice(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "设备不存在",
		})
		return
	}

	if !device.IsOnline() {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "设备离线，无法查询状态",
		})
		return
	}

	// 发起状态查询
	resp, err := h.statusService.QueryDeviceStatus(deviceID, 30*time.Second)
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("查询设备状态失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "查询设备状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}
