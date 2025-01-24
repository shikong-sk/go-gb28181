package main

import (
	"context"
	"fmt"
	"git.skcks.cn/Shikong/go-gb28181/pkg/config"
	"git.skcks.cn/Shikong/go-gb28181/pkg/logger"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	clientConfig, err := config.ReadClientConfig()
	if err != nil {
		logger.Log().Fatal(err)
		return
	}
	fmt.Printf("%+v\n", clientConfig)

	sip.SIPDebug = clientConfig.Debug

	ctx := context.Background()

	addr := fmt.Sprintf("%s:%d", clientConfig.ListenIp, clientConfig.ListenPort)
	ua, _ := sipgo.NewUA(
		sipgo.WithUserAgent(clientConfig.DeviceId),
		sipgo.WithUserAgentHostname(addr))

	srv, _ := sipgo.NewServer(ua)

	quit := make(chan os.Signal, 1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Log().Error(err)
				quit <- syscall.SIGKILL
			}
		}()

		if err := srv.ListenAndServe(ctx, "udp", addr); err != nil {
			logger.Log().Error(err)
			quit <- syscall.SIGTERM
		}
	}()

	signal.Notify(quit, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	<-quit
}
