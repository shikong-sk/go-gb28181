package handler

import (
	"git.skcks.cn/Shikong/go-gb28181/internal/client/config"
	"git.skcks.cn/Shikong/go-gb28181/internal/client/services"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils/charset"
	"github.com/duke-git/lancet/v2/random"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

func SetupKeepalive(client *sipgo.Client, clientConfig *config.ClientConfig) {
	services.AddKeepaliveSender(clientConfig.DeviceId, func(client *sipgo.Client) {
		sn := random.RandNumeral(6)
		data := manscdp.NewKeepAliveReqWithOK(sn, clientConfig.DeviceId)
		target := sip.Uri{
			User:    clientConfig.ServerId,
			Host:    clientConfig.ServerIp,
			Port:    clientConfig.ServerPort,
			Headers: sip.NewParams(),
		}

		req := sip.NewRequest(sip.MESSAGE, target)
		marshal, _ := utils.XMLMarshal(data, charset.GBK)
		req.SetBody(marshal)
		_ = client.WriteRequest(req)
	})
}
