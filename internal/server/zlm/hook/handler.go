package hook

import (
	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit"
	"github.com/gin-gonic/gin"
)

// HookHandler ZLM Hook 处理器
type HookHandler struct {
	playService   *service.PlayService
	deviceService *service.DeviceService
	ssrcService   *service.SsrcService   // SSRC 管理服务
	zlmClient     *zlmediakit.ZLMediaKit // ZLMediaKit 客户端
}

// NewHookHandler 创建 Hook 处理器
func NewHookHandler(playService *service.PlayService, deviceService *service.DeviceService, ssrcService *service.SsrcService, zlmClient *zlmediakit.ZLMediaKit) *HookHandler {
	return &HookHandler{
		playService:   playService,
		deviceService: deviceService,
		ssrcService:   ssrcService,
		zlmClient:     zlmClient,
	}
}

// RegisterRoutes 注册 Hook 路由
func (h *HookHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/on_publish", h.OnPublish)
	r.POST("/on_stream_changed", h.OnStreamChanged)
	r.POST("/on_stream_none_reader", h.OnStreamNoneReader)
	r.POST("/on_rtp_server_timeout", h.OnRtpServerTimeout)
}
