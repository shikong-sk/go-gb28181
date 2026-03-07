package http

import (
	"net/http"
	"strconv"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// ChannelHandler 通道 HTTP 处理器
type ChannelHandler struct {
	repo *repository.ChannelRepository
}

// NewChannelHandler 创建通道处理器
func NewChannelHandler(repo *repository.ChannelRepository) *ChannelHandler {
	return &ChannelHandler{repo: repo}
}

// ListChannelsRequest 通道列表请求
type ListChannelsRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	DeviceID string `form:"device_id"`
}

// ListChannelsResponse 通道列表响应
type ListChannelsResponse struct {
	Total int64           `json:"total"`
	List  []model.Channel `json:"list"`
}

// List 获取通道列表
// @Summary 获取通道列表
// @Tags 通道管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param device_id query string false "设备ID"
// @Success 200 {object} Response{data=ListChannelsResponse}
// @Router /api/channels [get]
func (h *ChannelHandler) List(c *gin.Context) {
	// 手动解析参数，兼容 camelCase 和 snake_case
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize == 0 {
		pageSize, _ = strconv.Atoi(c.Query("pageSize")) // 兼容 camelCase
	}
	deviceID := c.Query("device_id")
	if deviceID == "" {
		deviceID = c.Query("deviceId") // 兼容 camelCase
	}

	// 默认值
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	var channels []model.Channel
	var total int64
	var err error

	// 根据设备筛选
	if deviceID != "" {
		channels, total, err = h.repo.ListByDeviceID(deviceID, offset, pageSize)
	} else {
		channels, total, err = h.repo.List(offset, pageSize)
	}

	if err != nil {
		log.Error().Err(err).Msg("获取通道列表失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "获取通道列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: ListChannelsResponse{
			Total: total,
			List:  channels,
		},
	})
}

// GetChannel 获取通道详情
// @Summary 获取通道详情
// @Tags 通道管理
// @Accept json
// @Produce json
// @Param channel_id path string true "通道ID"
// @Param device_id query string true "设备ID"
// @Success 200 {object} Response{data=model.Channel}
// @Router /api/channels/{channel_id} [get]
func (h *ChannelHandler) GetChannel(c *gin.Context) {
	channelID := c.Param("channel_id")
	deviceID := c.Query("device_id")

	if channelID == "" || deviceID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "通道ID和设备ID不能为空",
		})
		return
	}

	channel, err := h.repo.GetByChannelID(channelID, deviceID)
	if err != nil {
		log.Error().Err(err).Str("channel_id", channelID).Str("device_id", deviceID).Msg("获取通道详情失败")
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "通道不存在",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    channel,
	})
}

// DeleteChannel 删除通道
// @Summary 删除通道
// @Tags 通道管理
// @Accept json
// @Produce json
// @Param channel_id path string true "通道ID"
// @Param device_id query string true "设备ID"
// @Success 200 {object} Response
// @Router /api/channels/{channel_id} [delete]
func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	channelID := c.Param("channel_id")
	deviceID := c.Query("device_id")

	if channelID == "" || deviceID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "通道ID和设备ID不能为空",
		})
		return
	}

	if err := h.repo.Delete(channelID, deviceID); err != nil {
		log.Error().Err(err).Str("channel_id", channelID).Str("device_id", deviceID).Msg("删除通道失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "删除通道失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "删除成功",
	})
}

// Stats 获取通道统计
// @Summary 获取通道统计
// @Tags 通道管理
// @Accept json
// @Produce json
// @Param device_id query string false "设备ID"
// @Success 200 {object} Response{data=ChannelStatsResponse}
// @Router /api/channels/stats [get]
func (h *ChannelHandler) Stats(c *gin.Context) {
	deviceID := c.Query("device_id")

	var total, online int64
	var err error

	if deviceID != "" {
		total, err = h.repo.CountByDeviceID(deviceID)
		if err != nil {
			log.Error().Err(err).Msg("获取通道统计失败")
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "获取统计失败",
			})
			return
		}
		online, err = h.repo.CountOnlineByDeviceID(deviceID)
	} else {
		// 全局统计
		total, err = h.repo.CountByDeviceID("")
		if err != nil {
			log.Error().Err(err).Msg("获取通道统计失败")
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "获取统计失败",
			})
			return
		}
		online, _ = h.repo.CountOnlineByDeviceID("")
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: ChannelStatsResponse{
			Total:   total,
			Online:  online,
			Offline: total - online,
		},
	})
}

// ChannelStatsResponse 通道统计响应
type ChannelStatsResponse struct {
	Total   int64 `json:"total"`
	Online  int64 `json:"online"`
	Offline int64 `json:"offline"`
}
