package app

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	deviceService       *service.DeviceService
	catalogService      *service.CatalogService
	playService         *service.PlayService
	recordService       *service.RecordService
	alarmService        *service.AlarmService
	ptzService          *service.PTZService
	subscriptionService *service.SubscriptionService
	positionService     *service.PositionService
	statusService       *service.DeviceStatusService

	// 服务器
	sipServer  *sip.SIPServer
	httpServer *http.Server
	zlmClient  *zlmediakit.ZLMediaKit

	// 仓库（用于 Start 中初始化服务）
	deviceRepo   *repository.DeviceRepository
	positionRepo repository.PositionRepository

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
	a.positionRepo = repository.NewPositionRepository(db)

	// 3. 初始化业务服务
	a.deviceService = service.NewDeviceService(a.deviceRepo, channelRepo)

	// 初始化报警服务
	alarmRepo := repository.NewAlarmRepository(db)
	a.alarmService = service.NewAlarmService(alarmRepo, &a.config.Alarm)

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
		a.sipServer = sip.NewSIPServer(a.config, a.deviceService, a.alarmService)
	}

	// 6. 初始化播放服务
	if a.zlmClient != nil {
		a.playService = service.NewPlayService(nil, a.zlmClient, buildPlayConfig(a.config.ZLMediaKit.Url), a.deviceService)
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

		sipClient := a.sipServer.GetClient()
		localIP := buildSIPLocalIP(a.config)

		// 初始化订阅服务
		a.subscriptionService = service.NewSubscriptionService(30 * time.Second)

		// 初始化定位服务 (默认保留 7 天)
		a.positionService = service.NewPositionService(a.positionRepo, 7)

		// 获取 SIP client 用于其他服务
		a.catalogService = service.NewCatalogService(
			sipClient,
			a.deviceRepo,
			a.config.SIP.DeviceID,
			localIP,
			a.config.SIP.ListenPort,
		)
		a.playService = service.NewPlayService(sipClient, a.zlmClient, buildPlayConfig(a.config.ZLMediaKit.Url), a.deviceService)
		a.ptzService = service.NewPTZService(sipClient, a.deviceService)
		a.recordService = service.NewRecordService(sipClient, a.deviceService, a.config.SIP.DeviceID, localIP, a.config.SIP.ListenPort)
		a.statusService = service.NewDeviceStatusService(sipClient, a.deviceService, a.subscriptionService, a.config.SIP.DeviceID, localIP, a.config.SIP.ListenPort)

		// 注入服务到 SIP Server
		a.sipServer.SetRecordService(a.recordService)
		a.sipServer.SetSubscriptionService(a.subscriptionService)
		a.sipServer.SetPositionService(a.positionService)
		a.sipServer.SetPlayService(a.playService)
	}

	// 2. 启动 HTTP 服务
	if a.config.HTTP.Enabled {
		if err := a.startHTTP(); err != nil {
			return fmt.Errorf("启动 HTTP 服务失败: %w", err)
		}
	}

	a.running = true
	log.Info().Msg("应用启动成功")

	go a.startAlarmCleanup()
	go a.startOfflineCheck()

	return nil
}

// startHTTP 启动 HTTP 服务
func (a *App) startHTTP() error {
	if a.config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := router.SetupRouterWithServices(a.catalogService, a.playService, a.alarmService, a.ptzService, a.recordService, a.subscriptionService, a.positionService, a.statusService, a.deviceService)
	addr := fmt.Sprintf("%s:%d", a.config.HTTP.Host, a.config.HTTP.Port)

	a.httpServer = &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

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

	if a.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.httpServer.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("HTTP 服务关闭失败")
		}
	}

	if a.sipServer != nil {
		a.sipServer.Stop()
	}

	database.Close()

	if a.cancel != nil {
		a.cancel()
	}

	a.running = false
	log.Info().Msg("应用已停止")
}

// startAlarmCleanup 启动报警清理定时任务 (每天凌晨执行)
func (a *App) startAlarmCleanup() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if a.alarmService != nil {
				if err := a.alarmService.CleanupExpired(); err != nil {
					log.Error().Err(err).Msg("清理过期报警失败")
				}
			}
		}
	}
}

// startOfflineCheck 启动设备离线检测定时任务 (每分钟执行)
func (a *App) startOfflineCheck() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if a.deviceService != nil {
				// GB28181 心跳间隔 30s，允许丢失 5 次，超时阈值 3 分钟
				if err := a.deviceService.CheckOfflineDevices(3); err != nil {
					log.Error().Err(err).Msg("设备离线检测失败")
				}
			}
		}
	}
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

// GetRecordService 获取录像服务
func (a *App) GetRecordService() *service.RecordService {
	return a.recordService
}

// GetAlarmService 获取报警服务
func (a *App) GetAlarmService() *service.AlarmService {
	return a.alarmService
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

func buildPlayConfig(rawURL string) service.PlayConfig {
	host := "127.0.0.1"
	port := 80
	parsedURL := rawURL
	if parsedURL != "" && !strings.Contains(parsedURL, "://") {
		parsedURL = "http://" + parsedURL
	}
	if parsedURL != "" {
		if parsed, err := url.Parse(parsedURL); err == nil {
			if parsed.Hostname() != "" {
				host = parsed.Hostname()
			}
			if parsed.Port() != "" {
				if parsedPort, err := strconv.Atoi(parsed.Port()); err == nil {
					port = parsedPort
				}
			} else if parsed.Scheme == "https" {
				port = 443
			}
		}
	}
	return service.PlayConfig{
		ZLMHost: host,
		ZLMPort: port,
		AppName: "rtp",
	}
}

func buildSIPLocalIP(cfg *config.Config) string {
	if cfg.SIP.ExternalIP != "" {
		return cfg.SIP.ExternalIP
	}
	if cfg.SIP.ServerIP != "" && cfg.SIP.ServerIP != "0.0.0.0" {
		return cfg.SIP.ServerIP
	}
	if cfg.SIP.ListenIP != "" && cfg.SIP.ListenIP != "0.0.0.0" {
		return cfg.SIP.ListenIP
	}
	return "127.0.0.1"
}
