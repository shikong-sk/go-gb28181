package http

import (
	"net/http"
	"time"

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

// PlayBackRequest 录像回放请求
type PlayBackRequest struct {
	DeviceId  string `json:"device_id" binding:"required"`
	ChannelId string `json:"channel_id" binding:"required"`
	StartTime string `json:"start_time" binding:"required"` // 格式: yyyy-MM-dd HH:mm:ss
	EndTime   string `json:"end_time" binding:"required"`
}

// PlayResponse 播放响应
type PlayResponse struct {
	StreamId string   `json:"stream_id"`
	Urls     []string `json:"urls"`
	RTPPort  int      `json:"rtp_port"`
	FLVURL   string   `json:"flv_url"`
	HLSURL   string   `json:"hls_url"`
	RTSPURL  string   `json:"rtsp_url"`
	RTMPURL  string   `json:"rtmp_url"`
	Mode     string   `json:"mode"`
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
			FLVURL:   result.FLVURL,
			HLSURL:   result.HLSURL,
			RTSPURL:  result.RTSPURL,
			RTMPURL:  result.RTMPURL,
			Mode:     result.Mode,
		},
	})
}

// PlayBack 开始录像回放
// @Summary 开始历史录像回放
// @Tags 视频播放
// @Accept json
// @Produce json
// @Param request body PlayBackRequest true "回放请求"
// @Success 200 {object} Response{data=PlayResponse}
// @Router /api/play/playback [post]
func (h *PlayHandler) PlayBack(c *gin.Context) {
	var req PlayBackRequest
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

	// 解析时间
	startTime, err := time.ParseInLocation("2006-01-02 15:04:05", req.StartTime, time.Local)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "开始时间格式错误，应为 yyyy-MM-dd HH:mm:ss",
		})
		return
	}

	endTime, err := time.ParseInLocation("2006-01-02 15:04:05", req.EndTime, time.Local)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "结束时间格式错误，应为 yyyy-MM-dd HH:mm:ss",
		})
		return
	}

	result, err := h.playService.PlayBack(req.DeviceId, req.ChannelId, startTime, endTime)
	if err != nil {
		log.Error().Err(err).Str("device_id", req.DeviceId).Str("channel_id", req.ChannelId).Msg("开始录像回放失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "开始录像回放失败: " + err.Error(),
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
			FLVURL:   result.FLVURL,
			HLSURL:   result.HLSURL,
			RTSPURL:  result.RTSPURL,
			RTMPURL:  result.RTMPURL,
			Mode:     result.Mode,
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
	StreamId      string `json:"stream_id"`
	DeviceId      string `json:"device_id"`
	ChannelId     string `json:"channel_id"`
	RTPPort       int    `json:"rtp_port"`
	Status        string `json:"status"`
	StartTime     string `json:"start_time"`
	Mode          string `json:"mode"`
	PlaybackStart string `json:"playback_start,omitempty"`
	PlaybackEnd   string `json:"playback_end,omitempty"`
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
			Mode:      string(s.Mode),
			PlaybackStart: func() string {
				if s.RangeStart == nil {
					return ""
				}
				return s.RangeStart.Format("2006-01-02 15:04:05")
			}(),
			PlaybackEnd: func() string {
				if s.RangeEnd == nil {
					return ""
				}
				return s.RangeEnd.Format("2006-01-02 15:04:05")
			}(),
		})
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    result,
	})
}

// StreamStatusResponse 流状态响应
type StreamStatusResponse struct {
	StreamId     string `json:"stream_id"`
	Ready        bool   `json:"ready"`         // 流是否已就绪（FLV 可播放）
	StreamActive bool   `json:"stream_active"` // RTP 流是否存在
	FlvReady     bool   `json:"flv_ready"`     // FLV/RTMP 流是否已生成
	Mode         string `json:"mode"`
	Status       string `json:"status"`
}

// GetStreamStatus 获取流就绪状态
// @Summary 获取流就绪状态
// @Tags 视频播放
// @Produce json
// @Param stream_id path string true "流 ID"
// @Success 200 {object} Response{data=StreamStatusResponse}
// @Router /api/play/stream_status/{stream_id} [get]
func (h *PlayHandler) GetStreamStatus(c *gin.Context) {
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

	session, exists := h.playService.GetSession(streamId)
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "会话不存在",
		})
		return
	}

	streamActive := h.playService.IsStreamActive(streamId)

	// 检查 FLV/RTMP 流是否已生成（前端播放需要）
	flvReady := h.playService.IsFlvStreamReady(streamId)

	// 流就绪条件：会话状态正常 + FLV 流已生成
	ready := session.Status == service.PlayStatusPlaying && flvReady

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: StreamStatusResponse{
			StreamId:     streamId,
			Ready:        ready,
			StreamActive: streamActive,
			FlvReady:     flvReady,
			Mode:         string(session.Mode),
			Status:       session.Status,
		},
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
