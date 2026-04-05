package hook

import (
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// OnRtpServerTimeout 处理 RTP 服务器超时 Hook
// 当 RTP 服务器长时间未收到数据时，ZLM 会调用此接口
func (h *HookHandler) OnRtpServerTimeout(c *gin.Context) {
	var req RtpServerTimeoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("解析 on_rtp_server_timeout 请求失败")
		c.JSON(200, RejectHookResponse("参数解析失败"))
		return
	}

	log.Warn().
		Str("app", req.App).
		Str("stream", req.Stream).
		Interface("request", req).
		Msg("收到 RTP 服务器超时 Hook")

	// 解析流 ID
	deviceId, channelId := parseStreamId(req.Stream)
	if deviceId == "" || channelId == "" {
		log.Warn().Str("stream", req.Stream).Msg("无法解析流 ID")
		c.JSON(200, SuccessHookResponse("忽略无效流 ID"))
		return
	}

	// 清理会话信息并释放资源
	if h.playService != nil {
		session, exists := h.playService.GetSession(req.Stream)
		if exists {
			log.Warn().
				Str("device_id", deviceId).
				Str("channel_id", channelId).
				Str("stream_id", req.Stream).
				Msg("RTP 服务器超时，清理播放会话")

			// 释放 SSRC
			if h.ssrcService != nil && session.SSRC != "" {
				if err := h.ssrcService.ReleaseSsrc(session.SSRC); err != nil {
					log.Warn().Err(err).Str("ssrc", session.SSRC).Msg("释放 SSRC 失败")
				} else {
					log.Info().Str("ssrc", session.SSRC).Msg("SSRC 已释放")
				}
			}

			// 关闭 RTP 服务器（重要：ZLM 超时时需要主动关闭）
			if h.zlmClient != nil {
				if _, err := h.zlmClient.CloseRtpServer(req.Stream); err != nil {
					log.Warn().Err(err).Str("stream_id", req.Stream).Msg("关闭 RTP 服务器失败")
				} else {
					log.Info().Str("stream_id", req.Stream).Msg("RTP 服务器已关闭")
				}
			}

			// 按 stream_id 清理会话。
			// RTP Server 超时回调给出的就是 stream 主键，
			// 直接按 stream 清理才能和 ZLM 的生命周期保持一致。
			h.playService.RemoveSessionByStreamID(req.Stream)
		} else {
			log.Warn().
				Str("stream_id", req.Stream).
				Msg("未找到对应播放会话")
		}
	}

	// 返回成功响应
	c.JSON(200, SuccessHookResponse("会话已清理"))
}
