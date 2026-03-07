package handler

import (
	"git.skcks.cn/Shikong/go-gb28181/internal/client/config"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// messageHandler 是一个处理SIP消息的函数类型
type messageHandler = func(client *sipgo.Client, clientConfig *config.ClientConfig, req *sip.Request, tx sip.ServerTransaction)

// handlers 是一个存储消息处理器的映射，键为命令类型，值为消息处理器函数
var handlers = make(map[string]messageHandler)

// SetupMessageHandler 设置SIP服务器的消息处理器，根据消息的命令类型调用相应的处理器
func SetupMessageHandler(srv *sipgo.Server, client *sipgo.Client, clientConfig *config.ClientConfig) {
	srv.OnMessage(func(req *sip.Request, tx sip.ServerTransaction) {
		body := req.Body()
		if len(body) == 0 {
			log.Log().Warn().Msgf("消息体为空")
			_ = tx.Respond(sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil))
			return
		}

		message := new(manscdp.MessageReq)
		err := utils.XMLUnmarshal(body, message)
		if err != nil {
			log.Log().Error().Msgf("XML Unmarshal error: %v", err)
			_ = tx.Respond(sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Bad Request", nil))
			return
		}

		handlers, ok := handlers[message.CmdType]
		if ok {
			handlers(client, clientConfig, req, tx)
		} else {
			log.Log().Warn().Msgf("未找到可用的消息处理器: %s", message.CmdType)
		}

		_ = tx.Respond(sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil))
	})
}
