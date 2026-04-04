package hook

import (
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// OnPublishRequest 推流鉴权请求
type OnPublishRequest struct {
	App           string `json:"app"`
	Stream        string `json:"stream"`
	IP            string `json:"ip"`
	Port          int    `json:"port"`
	Params        string `json:"params"`
	Schema        string `json:"schema"`
	Vhost         string `json:"vhost"`
	MediaServerId string `json:"mediaServerId"`
}

// OnPublishResponse 推流鉴权响应
// 参考 ZLM 文档：https://github.com/ZLMediaKit/ZLMediaKit/wiki/ZLMediaKit%E4%B8%AD%E7%9A%84Hook-%E6%9C%BA%E5%88%B6
type OnPublishResponse struct {
	Code           int    `json:"code"`
	Msg            string `json:"msg,omitempty"`
	EnableHls      bool   `json:"enable_hls"`       // 是否转换成 hls-mpegts 协议
	EnableHlsFmp4  bool   `json:"enable_hls_fmp4"`  // 是否转换成 hls-fmp4 协议
	EnableMp4      bool   `json:"enable_mp4"`       // 是否允许 mp4 录制
	EnableRtsp     bool   `json:"enable_rtsp"`      // 是否转 rtsp 协议
	EnableRtmp     bool   `json:"enable_rtmp"`      // 是否转 rtmp/flv 协议
	EnableTs       bool   `json:"enable_ts"`        // 是否转 http-ts/ws-ts 协议
	EnableFmp4     bool   `json:"enable_fmp4"`      // 是否转 http-fmp4/ws-fmp4 协议
	EnableAudio    bool   `json:"enable_audio"`     // 转协议时是否开启音频
	AddMuteAudio   bool   `json:"add_mute_audio"`   // 转协议时，无音频是否添加静音 aac 音频
	Mp4MaxSecond   int    `json:"mp4_max_second"`   // mp4 录制切片大小，单位秒
	AutoClose      bool   `json:"auto_close"`       // 无人观看是否自动关闭流(不触发无人观看 hook)
	ContinuePushMs int    `json:"continue_push_ms"` // 断连续推延时，单位毫秒
}

// OnPublish 处理推流鉴权 Hook
// 当设备推流到 ZLM 时，ZLM 会调用此接口进行鉴权
func (h *HookHandler) OnPublish(c *gin.Context) {
	var req OnPublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("解析 on_publish 请求失败")
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  "参数解析失败",
		})
		return
	}

	// 打印完整请求参数用于调试
	log.Info().
		Str("app", req.App).
		Str("stream", req.Stream).
		Str("ip", req.IP).
		Int("port", req.Port).
		Str("schema", req.Schema).
		Str("params", req.Params).
		Str("vhost", req.Vhost).
		Str("mediaServerId", req.MediaServerId).
		Msg("[ZLM HOOK] 推流鉴权完整参数")

	// 默认允许推流，并启用所有协议转换
	// 对于 GB28181 PS流，需要特殊处理时间戳
	response := OnPublishResponse{
		Code:           0, // 0 表示允许推流
		Msg:            "success",
		EnableHls:      true,  // 启用 HLS
		EnableHlsFmp4:  false, // 不启用 HLS-FMP4
		EnableMp4:      false, // 不录制 MP4
		EnableRtsp:     true,  // 启用 RTSP
		EnableRtmp:     true,  // 启用 RTMP/FLV
		EnableTs:       true,  // 启用 HTTP-TS
		EnableFmp4:     true,  // 启用 HTTP-FMP4
		EnableAudio:    true,  // 启用音频（PS流通常包含音频）
		AddMuteAudio:   false, // 不添加静音音频
		Mp4MaxSecond:   3600,  // MP4 最大时长1小时
		AutoClose:      false, // 不自动关闭流（关键！）
		ContinuePushMs: 30000, // 断连续推延时 30秒
	}

	log.Info().
		Str("stream", req.Stream).
		Int("code", response.Code).
		Msg("[ZLM HOOK] 推流鉴权通过")

	c.JSON(200, response)
}
