package message

import (
	"encoding/xml"
	"fmt"
	"git.skcks.cn/Shikong/go-gb28181/pkg/config"
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

	resp := new(manscdp.CatalogResp)
	resp.XMLName = xml.Name{Local: "Response"}
	resp.DeviceID = clientConfig.DeviceId
	resp.CmdType = "Catalog"
	resp.SumNum = "1"
	resp.DeviceList = new(manscdp.CateLogDeviceList)
	resp.DeviceList.XMLName = xml.Name{Local: "DeviceList"}
	resp.DeviceList.Num = "1"
	resp.DeviceList.Item = make([]manscdp.CateLogDevice, 0)

	device := manscdp.CateLogDevice{}
	device.DeviceID = clientConfig.DeviceId
	device.Name = "设备名称"
	device.Manufacturer = "设备厂商"
	device.ErrCode = "0"
	device.Port = fmt.Sprintf("%d", clientConfig.ListenPort)

	resp.DeviceList.Item = append(resp.DeviceList.Item, device)

	resp.SN = query.SN

	marshal, _ := utils.XMLMarshal(resp, "gbk")
	log.Log().Info().Msgf("回复查询指令: %s\n%+v\n", query.CmdType, resp)

	target := sip.Uri{
		User:    clientConfig.ServerId,
		Host:    clientConfig.ServerIp,
		Port:    clientConfig.ServerPort,
		Headers: sip.NewParams(),
	}

	//uri := sip.Uri{User: "44050100002000000002", Host: "10.10.10.20", Port: 5099}
	nReq := sip.NewRequest(sip.MESSAGE, target)
	nReq.SetTransport("UDP")
	to := sip.NewHeader("To", req.GetHeader("From").Value())
	from := sip.NewHeader("From", req.GetHeader("To").Value())
	nReq.AppendHeader(to)
	nReq.AppendHeader(from)
	//nReq.AppendHeader(req.GetHeader("Call-ID"))
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
