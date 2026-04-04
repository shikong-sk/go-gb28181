package http

import (
	"net/http"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// DeviceInfoHandler 设备信息 HTTP 处理器
type DeviceInfoHandler struct {
	infoService   *service.DeviceInfoService
	deviceService *service.DeviceService
}

// NewDeviceInfoHandler 创建设备信息处理器
func NewDeviceInfoHandler(infoService *service.DeviceInfoService, deviceService *service.DeviceService) *DeviceInfoHandler {
	return &DeviceInfoHandler{
		infoService:   infoService,
		deviceService: deviceService,
	}
}

// QueryDeviceInfo 发起设备信息查询（发送 SIP MESSAGE）
func (h *DeviceInfoHandler) QueryDeviceInfo(c *gin.Context) {
	if h.infoService == nil || h.deviceService == nil {
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
		log.Error().Err(err).Str("device_id", deviceID).Msg("获取设备失败")
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "设备不存在",
		})
		return
	}

	if !device.IsOnline() {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "设备离线，无法查询设备信息",
		})
		return
	}

	// 发起设备信息查询
	resp, err := h.infoService.QueryDeviceInfo(deviceID, 30*time.Second)
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("查询设备信息失败")
		c.JSON(http.StatusGatewayTimeout, Response{
			Code:    http.StatusGatewayTimeout,
			Message: "查询设备信息超时: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}

// GetDeviceInfo 获取设备信息（从数据库）
func (h *DeviceInfoHandler) GetDeviceInfo(c *gin.Context) {
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

	// 从数据库获取设备信息
	device, err := h.deviceService.GetDevice(deviceID)
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("获取设备失败")
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
			"device_id":     device.DeviceID,
			"name":          device.Name,
			"manufacturer":  device.Manufacturer,
			"model":         device.Model,
			"firmware":      device.Firmware,
			"channel_count": device.ChannelCount,
		},
	})
}
