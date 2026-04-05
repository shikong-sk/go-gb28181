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
		Bool("regist", req.Regist).
		Str("schema", req.Schema).
		Int("alive_second", req.AliveSecond).
		Int("reader_count", req.ReaderCount).
		Msg("收到流状态变化 Hook")

	// 解析流 ID（格式：deviceId_channelId）
	deviceId, channelId := parseStreamId(req.Stream)
	if deviceId == "" || channelId == "" {
		log.Warn().Str("stream", req.Stream).Msg("无法解析流 ID")
		c.JSON(200, SuccessHookResponse("忽略无效流 ID"))
		return
	}

	// 判断流是注册还是注销
	// 注意：regist=false 不代表设备真正停止推流，可能是 ZLM 启动过程的正常行为
	// 真正的流停止应该通过 on_stream_none_reader 或 on_rtp_server_timeout 判断
	if req.Regist {
		// 流注册：设备推流成功
		log.Info().
			Str("device_id", deviceId).
			Str("channel_id", channelId).
			Int("alive_second", req.AliveSecond).
			Int("total_reader_count", req.TotalReaderCount).
			Msg("设备推流成功，流已注册")

		// 通知 PlayService 流注册成功
		if h.playService != nil {
			h.playService.NotifyStreamRegistered(req.Stream)
		}
	} else {
		// 流注销通知：但不要立即清理，等待 on_stream_none_reader 或 on_rtp_server_timeout
		// 原因：ZLM 刚开始推流时可能先触发 regist=false，这是启动过程的正常行为
		log.Warn().
			Str("device_id", deviceId).
			Str("channel_id", channelId).
			Str("schema", req.Schema).
			Msg("收到流注销通知，但不立即清理（等待 none_reader 或 timeout hook）")

		// 不执行清理逻辑，让流自然超时或通过其他 hook 清理
		// 这样可以避免误杀刚启动的流

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

				// 按 stream_id 清理会话。
				// 这里的 hook 请求主键就是 req.Stream，
				// 如果错误地改用 Call-ID 删除，会导致会话残留，后续复用旧会话时误判“播放已建立”。
				h.playService.RemoveSessionByStreamID(req.Stream)
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
