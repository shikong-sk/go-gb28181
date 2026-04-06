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

		// 通知 PlayService 流注册成功，同时更新活跃时间
		if h.playService != nil {
			h.playService.NotifyStreamRegistered(req.Stream)
			// 流持续活跃时更新活跃时间（用于健康检查）
			h.playService.UpdateStreamActiveTime(req.Stream)
			// 更新观看人数
			h.playService.UpdateReaderCount(req.Stream, req.TotalReaderCount)
		}
	} else {
		// 流注销通知：只有流存活超过10秒才触发断流检测
		// ZLM 推流开始时会先发 regist=false (alive_second=0) 表示旧流注销
		// 然后发 regist=true 表示新流注册，这是正常流程，不应触发断流检测
		if req.AliveSecond > 10 {
			log.Warn().
				Str("device_id", deviceId).
				Str("channel_id", channelId).
				Str("schema", req.Schema).
				Int("alive_second", req.AliveSecond).
				Msg("收到流注销通知，触发断流检测")

			// 触发断流检测和重连
			if h.playService != nil {
				h.playService.OnStreamDisconnected(req.Stream)
			}
		} else {
			log.Debug().
				Str("device_id", deviceId).
				Str("channel_id", channelId).
				Str("schema", req.Schema).
				Int("alive_second", req.AliveSecond).
				Msg("收到流注销通知，流存活时间短，忽略（可能是正常推流流程）")
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
