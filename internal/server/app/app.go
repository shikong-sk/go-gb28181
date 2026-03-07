package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/api/router"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/config"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/database"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/sip"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit"
	"github.com/gin-gonic/gin"
)

// App 应用管理器 - 统一管理所有服务
type App struct {
	config *config.Config

	// 核心服务
	deviceService  *service.DeviceService
	catalogService *service.CatalogService
	playService    *service.PlayService

	// 服务器
	sipServer  *sip.SIPServer
	httpServer *http.Server
	zlmClient  *zlmediakit.ZLMediaKit

	// 仓库（用于 Start 中初始化服务）
	deviceRepo *repository.DeviceRepository

	// 状态
	running bool
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewApp 创建应用
func NewApp(cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Init 初始化应用
func (a *App) Init() error {
	log.Info().Msg("正在初始化应用...")

	// 1. 初始化数据库
	if _, err := database.Init(&a.config.Database); err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}

	// 2. 初始化仓库
	db := database.GetDB()
	a.deviceRepo = repository.NewDeviceRepository(db)
	channelRepo := repository.NewChannelRepository(db)

	// 3. 初始化业务服务
	a.deviceService = service.NewDeviceService(a.deviceRepo, channelRepo)

	// 4. 初始化 ZLMediaKit 客户端
	zlmCfg := &zlmediakit.Config{
		Url:    a.config.ZLMediaKit.Url,
		Secret: a.config.ZLMediaKit.Secret,
		Id:     a.config.ZLMediaKit.Id,
	}
	zlmediakit.SetupZLMediaKitService(zlmCfg)
	a.zlmClient = zlmediakit.GetZLMediaKitService()

	// 5. 初始化 SIP 服务
	if a.config.SIP.Enabled {
		a.sipServer = sip.NewSIPServer(a.config, a.deviceService)
	}

	// 6. 初始化播放服务
	if a.zlmClient != nil {
		a.playService = service.NewPlayService(nil, a.zlmClient, service.PlayConfig{
			ZLMHost: a.config.ZLMediaKit.Url,
			ZLMPort: 80,
			AppName: "rtp",
		})
	}

	log.Info().Msg("应用初始化完成")
	return nil
}

// Start 启动应用
func (a *App) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	// 1. 启动 SIP 服务
	if a.sipServer != nil {
		if err := a.sipServer.Start(); err != nil {
			return fmt.Errorf("启动 SIP 服务失败: %w", err)
		}
		// 获取 SIP client 用于其他服务
		a.catalogService = service.NewCatalogService(
			a.sipServer.GetClient(),
			a.deviceRepo,
			a.config.SIP.DeviceID,
			a.config.SIP.ListenIP,
			a.config.SIP.ListenPort,
		)
		a.playService = service.NewPlayService(a.sipServer.GetClient(), a.zlmClient, service.PlayConfig{
			ZLMHost: "127.0.0.1",
			ZLMPort: 80,
			AppName: "rtp",
		})
	}

	// 2. 启动 HTTP 服务
	if a.config.HTTP.Enabled {
		if err := a.startHTTP(); err != nil {
			return fmt.Errorf("启动 HTTP 服务失败: %w", err)
		}
	}

	a.running = true
	log.Info().Msg("应用启动成功")
	return nil
}

// startHTTP 启动 HTTP 服务
func (a *App) startHTTP() error {
	// 设置 Gin 模式
	if a.config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := router.SetupRouterWithServices(a.catalogService, a.playService)

	// HTTP 服务地址
	addr := fmt.Sprintf("%s:%d", a.config.HTTP.Host, a.config.HTTP.Port)

	// 创建 HTTP 服务器
	a.httpServer = &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 启动 HTTP 服务 (goroutine)
	go func() {
		log.Info().Str("addr", addr).Msg("HTTP 服务启动")
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP 服务异常停止")
		}
	}()

	return nil
}

// Stop 停止应用
func (a *App) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return
	}

	log.Info().Msg("正在停止应用...")

	// 1. 停止 HTTP 服务
	if a.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.httpServer.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("HTTP 服务关闭失败")
		}
	}

	// 2. 停止 SIP 服务
	if a.sipServer != nil {
		a.sipServer.Stop()
	}

	// 3. 关闭数据库
	database.Close()

	// 4. 取消上下文
	if a.cancel != nil {
		a.cancel()
	}

	a.running = false
	log.Info().Msg("应用已停止")
}

// GetDeviceService 获取设备服务
func (a *App) GetDeviceService() *service.DeviceService {
	return a.deviceService
}

// GetCatalogService 获取目录服务
func (a *App) GetCatalogService() *service.CatalogService {
	return a.catalogService
}

// GetPlayService 获取播放服务
func (a *App) GetPlayService() *service.PlayService {
	return a.playService
}

// GetSIPServer 获取 SIP 服务
func (a *App) GetSIPServer() *sip.SIPServer {
	return a.sipServer
}

// IsRunning 检查是否运行中
func (a *App) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// Wait 等待应用停止
func (a *App) Wait() {
	<-a.ctx.Done()
}
