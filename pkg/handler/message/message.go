package message

import (
	"git.skcks.cn/Shikong/go-gb28181/pkg/config"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

type messageHandler = func(client *sipgo.Client, clientConfig *config.ClientConfig, req *sip.Request, tx sip.ServerTransaction)

var handlers = make(map[string]messageHandler)

func SetupMessageHandler(srv *sipgo.Server, client *sipgo.Client, clientConfig *config.ClientConfig) {
	srv.OnMessage(func(req *sip.Request, tx sip.ServerTransaction) {
		body := req.Body()
		message := new(manscdp.MessageReq)
		err := utils.XMLUnmarshal(body, message)
		if err != nil {
			log.Log().Error().Msgf("XML Unmarshal error: %v", err)
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
