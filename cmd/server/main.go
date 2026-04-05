package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/app"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/config"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/rs/zerolog"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config", "", "配置文件路径")
}

func main() {
	flag.Parse()

	cfg, err := config.ReadConfig()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	if cfg.HTTP.Daemonize && os.Getenv("GB28181_DAEMONIZED") != "1" {
		if err := relaunchInBackground(); err != nil {
			fmt.Printf("后台启动失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("GB28181 服务已切换到后台运行")
		return
	}

	logger := log.InitLogger(cfg.Debug)
	log.SetLogger(&logger)

	logFile, err := log.SetupFileLogging(cfg.HTTP.LogFile)
	if err != nil {
		fmt.Printf("配置日志文件失败: %v\n", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer func() {
			_ = logFile.Close()
		}()
		log.Info().Str("log_file", cfg.HTTP.LogFile).Msg("已启用日志文件输出")
	}

	zerolog.TimeFieldFormat = time.RFC3339

	application := app.NewApp(cfg)
	if err := application.Init(); err != nil {
		log.Fatal().Err(err).Msg("初始化应用失败")
		return
	}
	if err := application.Start(); err != nil {
		log.Fatal().Err(err).Msg("启动应用失败")
		return
	}

	log.Info().Msg("GB28181 服务启动完成")
	log.Info().Msg("按 Ctrl+C 停止服务")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	application.Stop()
	log.Info().Msg("服务已退出")
}

func relaunchInBackground() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	args := make([]string, 0, len(os.Args)-1)
	for _, arg := range os.Args[1:] {
		if strings.TrimSpace(arg) == "" {
			continue
		}
		args = append(args, arg)
	}

	cmd := exec.Command(execPath, args...)
	cmd.Env = append(os.Environ(), "GB28181_DAEMONIZED=1")
	cmd.Dir = filepath.Dir(execPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
