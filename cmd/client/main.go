package main

import (
	"git.skcks.cn/Shikong/go-gb28181/pkg/config"
	"git.skcks.cn/Shikong/go-gb28181/pkg/handler/message"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit"
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
	fmt.Printf("%#v\n", clientConfig)

	if clientConfig.Debug {
		sip.SIPDebug = clientConfig.Debug
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	log.SetLogger(&logger)
	zlmediakit.SetupZLMediaKitService(clientConfig.ZLMediaKit)

	ctx := context.Background()
	addr := fmt.Sprintf("%s:%d", clientConfig.ListenIp, clientConfig.ListenPort)
	ua, _ := sipgo.NewUA(
		sipgo.WithUserAgent(clientConfig.DeviceId),
		sipgo.WithUserAgentHostname(addr))

	// 客户端, 发送/回复 SIP 指令
	client, _ := sipgo.NewClient(ua,
		sipgo.WithClientHostname(clientConfig.ListenIp),
		sipgo.WithClientPort(clientConfig.ListenPort))

	// 服务端, 接受 SIP 指令
	srv, _ := sipgo.NewServer(ua, sipgo.WithServerLogger(logger))

	message.SetupMessageHandler(srv, client, clientConfig)

	quit := make(chan os.Signal, 1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Fatal().Any("%s", err)
				quit <- syscall.SIGKILL
			}
		}()

		// 暂时 默认 udp
		// 启动 SIP 服务
		if err := srv.ListenAndServe(ctx, "udp", addr); err != nil {
			logger.Error().Err(err)
			quit <- syscall.SIGTERM
		}
	}()

	signal.Notify(quit, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	<-quit
}
