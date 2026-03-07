package http

import (
	"net/http"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// PlayHandler 播放 HTTP 处理器
type PlayHandler struct {
	playService *service.PlayService
}

// NewPlayHandler 创建播放处理器
func NewPlayHandler(playService *service.PlayService) *PlayHandler {
	return &PlayHandler{playService: playService}
}

// PlayRequest 播放请求
type PlayRequest struct {
	DeviceId  string `json:"device_id" binding:"required"`
	ChannelId string `json:"channel_id" binding:"required"`
}

// PlayResponse 播放响应
type PlayResponse struct {
	StreamId string   `json:"stream_id"`
	Urls     []string `json:"urls"`
	RTPPort  int      `json:"rtp_port"`
}

// MediaInfoResponse 媒体信息响应
type MediaInfoResponse struct {
	HasStream bool        `json:"has_stream"`
	Info      interface{} `json:"info,omitempty"`
}

// Play 开始播放
// @Summary 开始视频播放
// @Tags 视频播放
// @Accept json
// @Produce json
// @Param request body PlayRequest true "播放请求"
// @Success 200 {object} Response{data=PlayResponse}
// @Router /api/play/start [post]
func (h *PlayHandler) Play(c *gin.Context) {
	var req PlayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	if h.playService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "播放服务不可用",
		})
		return
	}

	result, err := h.playService.Play(req.DeviceId, req.ChannelId)
	if err != nil {
		log.Error().Err(err).Str("device_id", req.DeviceId).Str("channel_id", req.ChannelId).Msg("开始播放失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "开始播放失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: PlayResponse{
			StreamId: result.StreamId,
			Urls:     result.Urls,
			RTPPort:  result.RTPPort,
		},
	})
}

// StopRequest 停止播放请求
type StopRequest struct {
	StreamId string `json:"stream_id" binding:"required"`
}

// Stop 停止播放
// @Summary 停止视频播放
// @Tags 视频播放
// @Accept json
// @Produce json
// @Param request body StopRequest true "停止请求"
// @Success 200 {object} Response
// @Router /api/play/stop [post]
func (h *PlayHandler) Stop(c *gin.Context) {
	var req StopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	if h.playService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "播放服务不可用",
		})
		return
	}

	if err := h.playService.Stop(req.StreamId); err != nil {
		log.Error().Err(err).Str("stream_id", req.StreamId).Msg("停止播放失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "停止播放失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "停止成功",
	})
}

// SessionResponse 会话响应
type SessionResponse struct {
	StreamId  string `json:"stream_id"`
	DeviceId  string `json:"device_id"`
	ChannelId string `json:"channel_id"`
	RTPPort   int    `json:"rtp_port"`
	Status    string `json:"status"`
	StartTime string `json:"start_time"`
}

// ListSessions 列出所有播放会话
// @Summary 列出所有播放会话
// @Tags 视频播放
// @Produce json
// @Success 200 {object} Response{data=[]SessionResponse}
// @Router /api/play/sessions [get]
func (h *PlayHandler) ListSessions(c *gin.Context) {
	if h.playService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "播放服务不可用",
		})
		return
	}

	sessions := h.playService.ListSessions()
	result := make([]SessionResponse, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, SessionResponse{
			StreamId:  s.StreamId,
			DeviceId:  s.DeviceId,
			ChannelId: s.ChannelId,
			RTPPort:   s.RTPPort,
			Status:    s.Status,
			StartTime: s.StartTime.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    result,
	})
}

// GetMediaInfo 获取媒体信息
// @Summary 获取流媒体信息
// @Tags 视频播放
// @Produce json
// @Param stream_id path string true "流 ID"
// @Success 200 {object} Response{data=MediaInfoResponse}
// @Router /api/play/media/{stream_id} [get]
func (h *PlayHandler) GetMediaInfo(c *gin.Context) {
	streamId := c.Param("stream_id")
	if streamId == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "流 ID 不能为空",
		})
		return
	}

	if h.playService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "播放服务不可用",
		})
		return
	}

	info, err := h.playService.GetMediaInfo(streamId)
	if err != nil {
		log.Error().Err(err).Str("stream_id", streamId).Msg("获取媒体信息失败")
		c.JSON(http.StatusOK, Response{
			Code:    http.StatusOK,
			Message: "success",
			Data: MediaInfoResponse{
				HasStream: false,
			},
		})
		return
	}

	hasStream := info.Code == 0
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: MediaInfoResponse{
			HasStream: hasStream,
			Info:      info.Data,
		},
	})
}
