package handler

import (
	"fmt"
	"git.skcks.cn/Shikong/go-gb28181/internal/client/config"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

func init() {
	handlers["Catalog"] = CatalogHandler
}

func CatalogHandler(client *sipgo.Client, clientConfig *config.ClientConfig, req *sip.Request, tx sip.ServerTransaction) {
	query := new(manscdp.CatalogReq)
	_ = utils.XMLUnmarshal(req.Body(), query)
	log.Log().Info().Msgf("收到查询指令: %s\n%+v\n", query.CmdType, query)
	tx.Done()

	device := manscdp.NewCateLogDevice(func(device *manscdp.CateLogDevice) {
		device.DeviceID = clientConfig.DeviceId
		device.Name = clientConfig.Name
		device.Manufacturer = clientConfig.Manufacturer
		device.Model = clientConfig.Model
		device.IPAddress = clientConfig.IPAddress
		device.ErrCode = "0"
		device.Port = fmt.Sprintf("%d", clientConfig.ListenPort)
	})

	list := manscdp.NewCateLogDeviceList([]manscdp.CateLogDevice{*device})

	resp := manscdp.NewCatalogResp(1, list, query.SN, clientConfig.DeviceId)

	marshal, _ := utils.XMLMarshal(resp, "gbk")
	log.Log().Info().Msgf("回复查询指令: %s\n%+v\n", query.CmdType, resp)

	target := sip.Uri{
		User:    clientConfig.ServerId,
		Host:    clientConfig.ServerIp,
		Port:    clientConfig.ServerPort,
		Headers: sip.NewParams(),
	}

	nReq := sip.NewRequest(sip.MESSAGE, target)
	nReq.SetTransport("UDP")
	to := sip.NewHeader("To", req.GetHeader("From").Value())
	from := sip.NewHeader("From", req.GetHeader("To").Value())
	nReq.AppendHeader(to)
	nReq.AppendHeader(from)
	nReq.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))

	nReq.SetBody(marshal)
	err := sipgo.ClientRequestBuild(client, nReq)
	if err != nil {
		log.Log().Error().Msgf("向服务器发送查询指令失败: %s", err)
	}

	log.Log().Debug().Msgf("向服务器发送查询指令: %s\n%+v\n", query.CmdType, nReq)

	err = client.WriteRequest(nReq)
	if err != nil {
		log.Log().Error().Msgf("向服务器发送查询指令失败: %s", err)
		return
	}
}
