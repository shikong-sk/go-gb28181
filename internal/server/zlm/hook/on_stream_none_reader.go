package hook

import (
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// OnStreamNoneReader 处理无观看者 Hook
// 当流没有观看者时，ZLM 会调用此接口询问是否关闭流
func (h *HookHandler) OnStreamNoneReader(c *gin.Context) {
	var req StreamNoneReaderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("解析 on_stream_none_reader 请求失败")
		c.JSON(200, RejectHookResponse("参数解析失败"))
		return
	}

	log.Info().
		Str("app", req.App).
		Str("stream", req.Stream).
		Msg("收到无观看者 Hook")

	// 解析流 ID
	deviceId, channelId := parseStreamId(req.Stream)
	if deviceId == "" || channelId == "" {
		log.Warn().Str("stream", req.Stream).Msg("无法解析流 ID")
		c.JSON(200, SuccessHookResponse("忽略无效流 ID"))
		return
	}

	// 检查会话是否存在
	if h.playService != nil {
		if _, exists := h.playService.GetSession(req.Stream); exists {
			log.Info().
				Str("device_id", deviceId).
				Str("channel_id", channelId).
				Str("stream_id", req.Stream).
				Msg("流无观看者，更新观看人数为0")

			// 更新观看人数为0（用于流复用决策）
			h.playService.UpdateReaderCount(req.Stream, 0)
		} else {
			log.Warn().
				Str("stream_id", req.Stream).
				Msg("未找到对应播放会话")
		}
	}

	// 返回 Code=-1，不允许关闭流（保持流供后续播放）
	// GB28181 设备推流后需要保持流，等待客户端连接
	// 流的健康检查由 startStreamHealthCheck 定时任务负责
	c.JSON(200, RejectHookResponse("保持流不关闭"))
}
