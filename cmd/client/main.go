package main

import (
	"git.skcks.cn/Shikong/go-gb28181/pkg/config"
	"git.skcks.cn/Shikong/go-gb28181/pkg/handler/message"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/rs/zerolog"
	"math"
	"time"

	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 解决 sip go udp 包 > 1500 报错
	sip.UDPMTUSize = math.MaxInt
	output := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stdout
		w.TimeFormat = time.RFC3339
	})
	logger := zerolog.New(output).With().Timestamp().Logger()

	clientConfig, err := config.ReadClientConfig()
	if err != nil {
		logger.Fatal().Any("%s", err)
		return
	}
	fmt.Printf("%+v\n", clientConfig)

	if clientConfig.Debug {
		sip.SIPDebug = clientConfig.Debug
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	log.SetLogger(&logger)

	ctx := context.Background()
	addr := fmt.Sprintf("%s:%d", clientConfig.ListenIp, clientConfig.ListenPort)
	ua, _ := sipgo.NewUA(
		sipgo.WithUserAgent(clientConfig.DeviceId),
		sipgo.WithUserAgentHostname(addr))

	client, _ := sipgo.NewClient(ua,
		sipgo.WithClientHostname(clientConfig.ListenIp),
		sipgo.WithClientPort(clientConfig.ListenPort))

	srv, _ := sipgo.NewServer(ua, sipgo.WithServerLogger(logger))

	message.SetupMessageHandler(srv, client, clientConfig)

	//srv.OnMessage(func(req *sip.Request, tx sip.ServerTransaction) {
	//	query := new(manscdp.CatalogReq)
	//	_ = utils.XMLUnmarshal(req.Body(), query)
	//	logger.Info().Msgf("收到查询指令: %s\n%+v\n", query.CmdType, query)
	//
	//	tx.Done()
	//
	//	go func() {
	//		resp := new(manscdp.CatalogResp)
	//		resp.XMLName = xml.Name{Local: "Response"}
	//		resp.DeviceID = clientConfig.DeviceId
	//		resp.CmdType = "Catalog"
	//		resp.SumNum = "1"
	//		resp.DeviceList = new(manscdp.CateLogDeviceList)
	//		resp.DeviceList.XMLName = xml.Name{Local: "DeviceList"}
	//		resp.DeviceList.Num = "1"
	//		resp.DeviceList.Item = make([]manscdp.CateLogDevice, 0)
	//
	//		device := manscdp.CateLogDevice{}
	//		device.DeviceID = clientConfig.DeviceId
	//		device.Name = "设备名称"
	//		device.Manufacturer = "设备厂商"
	//		device.ErrCode = "0"
	//		device.Port = fmt.Sprintf("%d", clientConfig.ListenPort)
	//
	//		resp.DeviceList.Item = append(resp.DeviceList.Item, device)
	//
	//		resp.SN = query.SN
	//
	//		marshal, _ := utils.XMLMarshal(resp, "gbk")
	//		logger.Info().Msgf("回复查询指令: %s\n%+v\n", query.CmdType, resp)
	//
	//		target := sip.Uri{
	//			User:    clientConfig.ServerId,
	//			Host:    clientConfig.ServerIp,
	//			Port:    clientConfig.ServerPort,
	//			Headers: sip.NewParams(),
	//		}
	//
	//		//uri := sip.Uri{User: "44050100002000000002", Host: "10.10.10.20", Port: 5099}
	//		nReq := sip.NewRequest(sip.MESSAGE, target)
	//		nReq.SetTransport("UDP")
	//		to := sip.NewHeader("To", req.GetHeader("From").Value())
	//		from := sip.NewHeader("From", req.GetHeader("To").Value())
	//		nReq.AppendHeader(to)
	//		nReq.AppendHeader(from)
	//		//nReq.AppendHeader(req.GetHeader("Call-ID"))
	//		nReq.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))
	//
	//		nReq.SetBody(marshal)
	//		err := sipgo.ClientRequestBuild(client, nReq)
	//		if err != nil {
	//			logger.Error().Msgf("向服务器发送查询指令失败: %s", err)
	//		}
	//
	//		logger.Debug().Msgf("向服务器发送查询指令: %s\n%+v\n", query.CmdType, nReq)
	//
	//		err = client.WriteRequest(nReq)
	//		if err != nil {
	//			logger.Error().Msgf("向服务器发送查询指令失败: %s", err)
	//			return
	//		}
	//	}()
	//})

	quit := make(chan os.Signal, 1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Fatal().Any("%s", err)
				quit <- syscall.SIGKILL
			}
		}()

		if err := srv.ListenAndServe(ctx, "udp", addr); err != nil {
			logger.Error().Err(err)
			quit <- syscall.SIGTERM
		}
	}()

	signal.Notify(quit, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	<-quit
}
