package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/app"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/config"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config", "", "配置文件路径")
}

func main() {
	flag.Parse()

	// 加载配置
	cfg, err := config.ReadConfig()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger := log.InitLogger(cfg.Debug)
	log.SetLogger(&logger)

	log.Info().Msg("GB28181 服务启动中...")

	// 创建应用
	application := app.NewApp(cfg)

	// 初始化应用
	if err := application.Init(); err != nil {
		log.Fatal().Err(err).Msg("初始化应用失败")
		return
	}

	// 启动应用
	if err := application.Start(); err != nil {
		log.Fatal().Err(err).Msg("启动应用失败")
		return
	}

	log.Info().Msg("GB28181 服务启动完成")
	log.Info().Msg("按 Ctrl+C 停止服务")

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 停止应用
	application.Stop()

	log.Info().Msg("服务已退出")
}
