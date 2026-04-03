package http

import (
	"net/http"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"github.com/gin-gonic/gin"
)

// PTZHandler 云台控制 HTTP 处理器
type PTZHandler struct {
	ptzService *service.PTZService
}

// NewPTZHandler 创建云台控制处理器
func NewPTZHandler(ptzService *service.PTZService) *PTZHandler {
	return &PTZHandler{ptzService: ptzService}
}

// PTZRequest 云台控制请求
type PTZRequest struct {
	DeviceId        string `json:"device_id" binding:"required"`
	ChannelId       string `json:"channel_id" binding:"required"`
	Direction       string `json:"direction" binding:"required"` // Stop, Up, Down, Left, Right, ZoomIn, ZoomOut
	HorizontalSpeed int    `json:"horizontal_speed,omitempty"`   // 0-15
	VerticalSpeed   int    `json:"vertical_speed,omitempty"`     // 0-255
	Speed           int    `json:"speed,omitempty"`              // 通用速度 (用于简化调用)
}

// Control 云台控制
// @Summary 云台控制
// @Tags 云台控制
// @Accept json
// @Produce json
// @Param request body PTZRequest true "云台控制请求"
// @Success 200 {object} Response
// @Router /api/ptz/control [post]
func (h *PTZHandler) Control(c *gin.Context) {
	var req PTZRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	if h.ptzService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "云台控制服务不可用",
		})
		return
	}

	// 解析方向
	direction := manscdp.PTZDirection(req.Direction)

	// 如果使用通用速度，根据方向自动设置
	hSpeed := req.HorizontalSpeed
	vSpeed := req.VerticalSpeed
	if req.Speed > 0 && hSpeed == 0 && vSpeed == 0 {
		hSpeed = req.Speed
		vSpeed = req.Speed * 16 // 转换为 0-255 范围
		if vSpeed > 255 {
			vSpeed = 255
		}
	}

	// 执行云台控制
	if err := h.ptzService.PTZControl(req.DeviceId, req.ChannelId, direction, hSpeed, vSpeed); err != nil {
		log.Error().Err(err).Str("device_id", req.DeviceId).Str("channel_id", req.ChannelId).Msg("云台控制失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "云台控制失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

// Stop 停止云台
// @Summary 停止云台
// @Tags 云台控制
// @Accept json
// @Produce json
// @Param request body PTZRequest true "停止请求"
// @Success 200 {object} Response
// @Router /api/ptz/stop [post]
func (h *PTZHandler) Stop(c *gin.Context) {
	var req PTZRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	if h.ptzService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "云台控制服务不可用",
		})
		return
	}

	if err := h.ptzService.PTZStop(req.DeviceId, req.ChannelId); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "停止云台失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}
