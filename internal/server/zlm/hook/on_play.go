package hook

import (
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// OnPlayRequest 播放鉴权请求
type OnPlayRequest struct {
	App           string `json:"app"`
	Stream        string `json:"stream"`
	IP            string `json:"ip"`
	Port          int    `json:"port"`
	Params        string `json:"params"`
	Schema        string `json:"schema"`
	Vhost         string `json:"vhost"`
	MediaServerId string `json:"mediaServerId"`
	PlayerId      string `json:"player_id"`
}

// OnPlayResponse 播放鉴权响应
// 参考 ZLM 文档：https://github.com/ZLMediaKit/ZLMediaKit/wiki/ZLMediaKit%E4%B8%AD%E7%9A%84Hook-%E6%9C%BA%E5%88%B6
type OnPlayResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg,omitempty"`
}

// OnPlay 处理播放鉴权 Hook
// 当客户端请求播放流时，ZLM 会调用此接口进行鉴权
func (h *HookHandler) OnPlay(c *gin.Context) {
	var req OnPlayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("解析 on_play 请求失败")
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
		Str("playerId", req.PlayerId).
		Msg("[ZLM HOOK] 播放鉴权请求")

	// 默认允许所有播放请求
	response := OnPlayResponse{
		Code: 0, // 0 表示允许播放
		Msg:  "success",
	}

	log.Info().
		Str("stream", req.Stream).
		Str("playerId", req.PlayerId).
		Int("code", response.Code).
		Msg("[ZLM HOOK] 播放鉴权通过")

	c.JSON(200, response)
}
