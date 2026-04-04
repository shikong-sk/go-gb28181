package http

import (
	"net/http"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// DownloadHandler 录像下载 HTTP 处理器
type DownloadHandler struct {
	downloadService *service.DownloadService
}

// NewDownloadHandler 创建录像下载处理器
func NewDownloadHandler(downloadService *service.DownloadService) *DownloadHandler {
	return &DownloadHandler{downloadService: downloadService}
}

// DownloadRequest 录像下载请求
type DownloadRequest struct {
	DeviceId  string `json:"device_id" binding:"required"`
	ChannelId string `json:"channel_id" binding:"required"`
	StartTime string `json:"start_time" binding:"required"` // 格式: yyyy-MM-dd HH:mm:ss
	EndTime   string `json:"end_time" binding:"required"`
	Speed     int    `json:"speed"` // 倍速：1/2/4，默认 1
}

// DownloadResponse 录像下载响应
type DownloadResponse struct {
	StreamId  string  `json:"stream_id"`
	DeviceId  string  `json:"device_id"`
	ChannelId string  `json:"channel_id"`
	Status    string  `json:"status"`
	Speed     int     `json:"speed"`
	Progress  float64 `json:"progress"`
	CreatedAt string  `json:"created_at"`
}

// ProgressResponse 下载进度响应
type ProgressResponse struct {
	StreamId  string  `json:"stream_id"`
	DeviceId  string  `json:"device_id"`
	ChannelId string  `json:"channel_id"`
	Status    string  `json:"status"`
	Progress  float64 `json:"progress"`
	Speed     int     `json:"speed"`
	Error     string  `json:"error,omitempty"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// Start 开始录像下载
// @Summary 开始录像下载
// @Tags 录像下载
// @Accept json
// @Produce json
// @Param request body DownloadRequest true "下载请求"
// @Success 200 {object} Response{data=DownloadResponse}
// @Router /api/download/start [post]
func (h *DownloadHandler) Start(c *gin.Context) {
	var req DownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	if h.downloadService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "下载服务不可用",
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

	// 默认倍速为 1
	speed := req.Speed
	if speed == 0 {
		speed = 1
	}

	// 倍速限制：仅支持 1/2/4
	if speed != 1 && speed != 2 && speed != 4 {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "下载倍速仅支持 1/2/4",
		})
		return
	}

	session, err := h.downloadService.StartDownload(req.DeviceId, req.ChannelId, startTime, endTime, speed)
	if err != nil {
		log.Error().Err(err).Str("device_id", req.DeviceId).Str("channel_id", req.ChannelId).Msg("开始录像下载失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "开始录像下载失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: DownloadResponse{
			StreamId:  session.StreamId,
			DeviceId:  session.DeviceId,
			ChannelId: session.ChannelId,
			Status:    session.Status,
			Speed:     session.Speed,
			Progress:  session.Progress,
			CreatedAt: session.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// Progress 查询下载进度
// @Summary 查询下载进度
// @Tags 录像下载
// @Produce json
// @Param stream_id path string true "流 ID"
// @Success 200 {object} Response{data=ProgressResponse}
// @Router /api/download/progress/{stream_id} [get]
func (h *DownloadHandler) Progress(c *gin.Context) {
	streamId := c.Param("stream_id")
	if streamId == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "流 ID 不能为空",
		})
		return
	}

	if h.downloadService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "下载服务不可用",
		})
		return
	}

	session, err := h.downloadService.GetProgress(streamId)
	if err != nil {
		log.Error().Err(err).Str("stream_id", streamId).Msg("查询下载进度失败")
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "下载会话不存在: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: ProgressResponse{
			StreamId:  session.StreamId,
			DeviceId:  session.DeviceId,
			ChannelId: session.ChannelId,
			Status:    session.Status,
			Progress:  session.Progress,
			Speed:     session.Speed,
			Error:     session.Error,
			StartTime: session.StartTime.Format("2006-01-02 15:04:05"),
			EndTime:   session.EndTime.Format("2006-01-02 15:04:05"),
			CreatedAt: session.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: session.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// Cancel 取消下载
// @Summary 取消录像下载
// @Tags 录像下载
// @Produce json
// @Param stream_id path string true "流 ID"
// @Success 200 {object} Response
// @Router /api/download/cancel/{stream_id} [post]
func (h *DownloadHandler) Cancel(c *gin.Context) {
	streamId := c.Param("stream_id")
	if streamId == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "流 ID 不能为空",
		})
		return
	}

	if h.downloadService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "下载服务不可用",
		})
		return
	}

	if err := h.downloadService.CancelDownload(streamId); err != nil {
		log.Error().Err(err).Str("stream_id", streamId).Msg("取消录像下载失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "取消录像下载失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "下载已取消",
	})
}

// DownloadSessionResponse 下载会话响应
type DownloadSessionResponse struct {
	StreamId  string  `json:"stream_id"`
	DeviceId  string  `json:"device_id"`
	ChannelId string  `json:"channel_id"`
	Status    string  `json:"status"`
	Progress  float64 `json:"progress"`
	Speed     int     `json:"speed"`
	Error     string  `json:"error,omitempty"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// ListSessions 列出所有下载会话
// @Summary 列出所有下载会话
// @Tags 录像下载
// @Produce json
// @Success 200 {object} Response{data=[]DownloadSessionResponse}
// @Router /api/download/sessions [get]
func (h *DownloadHandler) ListSessions(c *gin.Context) {
	if h.downloadService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "下载服务不可用",
		})
		return
	}

	sessions := h.downloadService.ListSessions()
	result := make([]DownloadSessionResponse, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, DownloadSessionResponse{
			StreamId:  s.StreamId,
			DeviceId:  s.DeviceId,
			ChannelId: s.ChannelId,
			Status:    s.Status,
			Progress:  s.Progress,
			Speed:     s.Speed,
			Error:     s.Error,
			StartTime: s.StartTime.Format("2006-01-02 15:04:05"),
			EndTime:   s.EndTime.Format("2006-01-02 15:04:05"),
			CreatedAt: s.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: s.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    result,
	})
}
