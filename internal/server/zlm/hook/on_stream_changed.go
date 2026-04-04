package hook

import (
	"strings"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// OnStreamChanged 处理流状态变化 Hook
// 当流注册或注销时，ZLM 会调用此接口
func (h *HookHandler) OnStreamChanged(c *gin.Context) {
	var req StreamChangedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("解析 on_stream_changed 请求失败")
		c.JSON(200, RejectHookResponse("参数解析失败"))
		return
	}

	log.Info().
		Str("app", req.App).
		Str("stream", req.Stream).
		Bool("register", req.Register).
		Str("schema", req.Schema).
		Msg("收到流状态变化 Hook")

	// 解析流 ID（格式：deviceId_channelId）
	deviceId, channelId := parseStreamId(req.Stream)
	if deviceId == "" || channelId == "" {
		log.Warn().Str("stream", req.Stream).Msg("无法解析流 ID")
		c.JSON(200, SuccessHookResponse("忽略无效流 ID"))
		return
	}

	// 判断流是注册还是注销
	if req.Register {
		// 流注册：设备推流成功
		log.Info().
			Str("device_id", deviceId).
			Str("channel_id", channelId).
			Msg("设备推流成功，流已注册")

		// 可以在这里更新通道状态为"正在播放"
		// 目前仅记录日志，后续可扩展
	} else {
		// 流注销：设备停止推流
		log.Info().
			Str("device_id", deviceId).
			Str("channel_id", channelId).
			Msg("设备停止推流，流已注销")

		// 清理播放会话并释放资源
		if h.playService != nil {
			session, exists := h.playService.GetSession(req.Stream)
			if exists {
				log.Info().Str("stream_id", req.Stream).Msg("清理播放会话")

				// 释放 SSRC
				if h.ssrcService != nil && session.SSRC != "" {
					if err := h.ssrcService.ReleaseSsrc(session.SSRC); err != nil {
						log.Warn().Err(err).Str("ssrc", session.SSRC).Msg("释放 SSRC 失败")
					} else {
						log.Info().Str("ssrc", session.SSRC).Msg("SSRC 已释放")
					}
				}

				// 关闭 RTP 服务器
				if h.zlmClient != nil {
					if _, err := h.zlmClient.CloseRtpServer(req.Stream); err != nil {
						log.Warn().Err(err).Str("stream_id", req.Stream).Msg("关闭 RTP 服务器失败")
					} else {
						log.Info().Str("stream_id", req.Stream).Msg("RTP 服务器已关闭")
					}
				}

				// 移除会话
				h.playService.RemoveSession(session.CallID)
			}
		}
	}

	// 返回成功响应，允许流状态变化
	c.JSON(200, SuccessHookResponse("ok"))
}

// parseStreamId 解析流 ID（格式：deviceId_channelId）
func parseStreamId(streamId string) (deviceId, channelId string) {
	// 回放流格式：deviceId_channelId_startTime_endTime
	// 实时流格式：deviceId_channelId
	parts := strings.Split(streamId, "_")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
