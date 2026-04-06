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
	"git.skcks.cn/Shikong/go-gb28181/internal/server/websocket"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// App 应用管理器 - 统一管理所有服务
type App struct {
	config *config.Config

	// 核心服务
	deviceService              *service.DeviceService
	catalogService             *service.CatalogService
	catalogSubscriptionService *service.CatalogSubscriptionService
	playService                *service.PlayService
	recordService              *service.RecordService
	downloadService            *service.DownloadService // 录像下载服务
	alarmService               *service.AlarmService
	alarmSubscriptionService   *service.AlarmSubscriptionService // 报警订阅服务
	ptzService                 *service.PTZService
	subscriptionService        *service.SubscriptionService
	positionService            *service.PositionService
	statusService              *service.DeviceStatusService
	infoService                *service.DeviceInfoService
	eventService               *service.EventService       // 事件发布服务
	ssrcService                *service.SsrcService        // SSRC 管理服务
	recordFetchService         *service.RecordFetchService // 录像缓存拉取服务

	// 服务器
	sipServer   *sip.SIPServer
	httpServer  *http.Server
	zlmClient   *zlmediakit.ZLMediaKit
	wsManager   *websocket.WebSocketManager // WebSocket 管理器
	redisClient *redis.Client               // Redis 客户端

	// 仓库（用于 Start 中初始化服务）
	deviceRepo      *repository.DeviceRepository
	channelRepo     *repository.ChannelRepository
	positionRepo    repository.PositionRepository
	recordCacheRepo *repository.RecordCacheRepository

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
	a.channelRepo = repository.NewChannelRepository(db)
	a.positionRepo = repository.NewPositionRepository(db)
	a.recordCacheRepo = repository.NewRecordCacheRepository(db)

	// 3. 初始化事件服务（WebSocketManager 暂未初始化，事件仅记录日志）
	a.eventService = service.NewEventService()

	// 4. 初始化业务服务
	a.deviceService = service.NewDeviceService(a.deviceRepo, a.channelRepo, a.eventService)

	// 初始化报警服务
	alarmRepo := repository.NewAlarmRepository(db)
	a.alarmService = service.NewAlarmService(alarmRepo, &a.config.Alarm, a.eventService)

	// 5. 初始化 Redis 客户端
	a.redisClient = redis.NewClient(&redis.Options{
		Addr:     a.config.Redis.Addr,
		Password: a.config.Redis.Password,
		DB:       a.config.Redis.DB,
	})

	// 测试 Redis 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.redisClient.Ping(ctx).Err(); err != nil {
		log.Warn().Err(err).Msg("Redis 连接失败，SSRC 管理服务将不可用")
	} else {
		log.Info().Str("addr", a.config.Redis.Addr).Msg("Redis 连接成功")
		// 初始化 SSRC 服务 - 使用 SIP.DeviceID 提取域编码前缀
		// COMPAT_WVP: WVP 从域编码提取 SSRC 前缀（如 44050100002000000002 → 50100）
		a.ssrcService = service.NewSsrcService(a.redisClient, a.config.SIP.DeviceID)
	}

	// 6. 初始化 ZLMediaKit 客户端
	zlmCfg := &zlmediakit.Config{
		Url:    a.config.ZLMediaKit.Url,
		Secret: a.config.ZLMediaKit.Secret,
		Id:     a.config.ZLMediaKit.Id,
	}
	zlmediakit.SetupZLMediaKitService(zlmCfg)
	a.zlmClient = zlmediakit.GetZLMediaKitService()

	// 7. 初始化 SIP 服务
	if a.config.SIP.Enabled {
		a.sipServer = sip.NewSIPServer(a.config, a.deviceService, a.alarmService)
	}

	// 8. 初始化播放服务
	// 使用 buildSIPLocalIP 获取有效的 SIP 本地 IP（避免 0.0.0.0）
	localIP := buildSIPLocalIP(a.config)
	if a.zlmClient != nil {
		a.playService = service.NewPlayService(nil, a.zlmClient, buildPlayConfig(a.config.ZLMediaKit.Url, a.config.SIP.DeviceID, localIP, a.config.SIP.ListenPort, a.config.ZLMediaKit.RtpPort), a.deviceService, a.ssrcService, a.config.SIP.InviteTimeout)
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

	// 1. 初始化并启动 WebSocket 服务
	a.wsManager = websocket.NewWebSocketManager()
	a.eventService.SetWebSocketManager(a.wsManager)
	go a.wsManager.Start()
	log.Info().Msg("WebSocket 服务已启动")

	// 2. 启动 SIP 服务
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
		a.playService = service.NewPlayService(sipClient, a.zlmClient, buildPlayConfig(a.config.ZLMediaKit.Url, a.config.SIP.DeviceID, localIP, a.config.SIP.ListenPort, a.config.ZLMediaKit.RtpPort), a.deviceService, a.ssrcService, a.config.SIP.InviteTimeout)
		a.downloadService = service.NewDownloadService(a.playService)
		a.ptzService = service.NewPTZService(sipClient, a.deviceService)
		a.recordService = service.NewRecordService(sipClient, a.deviceService, a.recordCacheRepo, a.config.SIP.DeviceID, localIP, a.config.SIP.ListenPort)
		a.statusService = service.NewDeviceStatusService(sipClient, a.deviceService, a.subscriptionService, a.config.SIP.DeviceID, localIP, a.config.SIP.ListenPort)
		a.infoService = service.NewDeviceInfoService(sipClient, a.deviceService, a.subscriptionService, a.config.SIP.DeviceID, localIP, a.config.SIP.ListenPort)

		// 初始化报警订阅服务
		db := database.GetDB()
		alarmRepo := repository.NewAlarmRepository(db)
		a.alarmSubscriptionService = service.NewAlarmSubscriptionService(
			sipClient,
			a.deviceRepo,
			alarmRepo,
			a.alarmService,
			a.eventService,
			a.config.SIP.DeviceID,
			localIP,
			a.config.SIP.ListenPort,
		)

		// 注入服务到 SIP Server
		a.sipServer.SetRecordService(a.recordService)
		a.sipServer.SetSubscriptionService(a.subscriptionService)
		a.sipServer.SetPositionService(a.positionService)
		a.sipServer.SetPlayService(a.playService)
		a.sipServer.SetAlarmSubscriptionService(a.alarmSubscriptionService)

		// 初始化录像缓存拉取服务
		a.recordFetchService = service.NewRecordFetchService(
			a.recordService,
			a.deviceService,
			a.recordCacheRepo,
			a.channelRepo,
			24, // 默认缓存 24 小时
		)
	}

	// 3. 启动 HTTP 服务
	if a.config.HTTP.Enabled {
		if err := a.startHTTP(); err != nil {
			return fmt.Errorf("启动 HTTP 服务失败: %w", err)
		}
	}

	// 4. 动态配置 ZLM Hook（在 ZLM 初始化和 HTTP 服务启动后）
	if a.zlmClient != nil && a.config.HTTP.Enabled {
		// 先获取当前配置，避免清空其他配置项
		currentConfig, err := a.zlmClient.GetServerConfig()
		if err != nil {
			log.Warn().Err(err).Msg("获取 ZLM 配置失败，跳过 Hook 配置")
		} else if len(currentConfig.Data) > 0 {
			// 只修改 Hook 相关配置
			hookBaseURL := fmt.Sprintf("http://%s:%d/index/api/hook",
				a.config.HTTP.Host, a.config.HTTP.Port)

			config := &currentConfig.Data[0]
			config.HookEnable = "1"
			config.HookOnPublish = hookBaseURL + "/on_publish"
			config.HookOnPlay = hookBaseURL + "/on_play"
			config.HookOnStreamChanged = hookBaseURL + "/on_stream_changed"
			config.HookOnStreamNoneReader = hookBaseURL + "/on_stream_none_reader"
			config.HookOnRtpServerTimeout = hookBaseURL + "/on_rtp_server_timeout"

			if _, err := a.zlmClient.SetServerConfig(config); err != nil {
				log.Warn().Err(err).Msg("配置 ZLM Hook 失败，Hook 功能可能不可用")
			} else {
				log.Info().Str("hook_base_url", hookBaseURL).Msg("ZLM Hook 配置成功")
			}
		}
	}

	a.running = true
	log.Info().Msg("应用启动成功")

	go a.startAlarmCleanup()
	go a.startOfflineCheck()
	go a.startSubscriptionCleanup()
	go a.startDownloadCleanup()
	go a.startPlaySessionCleanup()
	go a.startRecordFetch()

	return nil
}

// startHTTP 启动 HTTP 服务
func (a *App) startHTTP() error {
	if a.config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := router.SetupRouterWithServices(a.catalogService, a.catalogSubscriptionService, a.playService, a.alarmService, a.alarmSubscriptionService, a.ptzService, a.recordService, a.downloadService, a.subscriptionService, a.positionService, a.statusService, a.deviceService, a.infoService, a.wsManager, a.ssrcService, a.zlmClient)
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

	if a.wsManager != nil {
		a.wsManager.Stop()
	}

	if a.recordFetchService != nil {
		a.recordFetchService.StopBackgroundFetcher()
	}

	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			log.Error().Err(err).Msg("Redis 连接关闭失败")
		}
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

// startSubscriptionCleanup 启动订阅清理定时任务 (每分钟执行)
func (a *App) startSubscriptionCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			log.Info().Msg("订阅清理任务停止")
			return
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Error().Any("panic", r).Msg("订阅清理任务 panic")
					}
				}()
				if a.subscriptionService != nil {
					count := a.subscriptionService.CleanupExpired()
					if count > 0 {
						log.Info().Int("cleaned", count).Msg("清理过期订阅")
					}
				}
			}()
		}
	}
}

// startDownloadCleanup 启动下载会话清理定时任务 (每小时执行)
func (a *App) startDownloadCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			log.Info().Msg("下载会话清理任务停止")
			return
		case <-ticker.C:
			if a.downloadService != nil {
				// 清理超过 24 小时的已完成/已取消/错误会话
				a.downloadService.CleanupStaleSessions(24 * time.Hour)
			}
		}
	}
}

// startPlaySessionCleanup 启动播放会话清理定时任务 (每小时执行)
func (a *App) startPlaySessionCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			log.Info().Msg("播放会话清理任务停止")
			return
		case <-ticker.C:
			if a.playService != nil {
				// 清理超过 24 小时的过期会话
				a.playService.CleanupStaleSessions(24 * time.Hour)
			}
		}
	}
}

// startRecordFetch 启动录像缓存后台拉取 (每小时执行)
func (a *App) startRecordFetch() {
	if a.recordFetchService == nil {
		return
	}
	// 默认每小时拉取一次录像缓存
	a.recordFetchService.StartBackgroundFetcher(1 * time.Hour)
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

// GetDeviceInfoService 获取设备信息查询服务
func (a *App) GetDeviceInfoService() *service.DeviceInfoService {
	return a.infoService
}

// GetSIPServer 获取 SIP 服务
func (a *App) GetSIPServer() *sip.SIPServer {
	return a.sipServer
}

// GetSsrcService 获取 SSRC 管理服务
func (a *App) GetSsrcService() *service.SsrcService {
	return a.ssrcService
}

// GetRecordFetchService 获取录像缓存拉取服务
func (a *App) GetRecordFetchService() *service.RecordFetchService {
	return a.recordFetchService
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

func buildPlayConfig(rawURL string, localId string, sipListenIP string, sipPort int, rtpPort int) service.PlayConfig {
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
		ZLMHost:     host,
		ZLMPort:     port,
		AppName:     "rtp",
		LocalId:     localId,
		SIPListenIP: sipListenIP,
		SIPPort:     sipPort,
		RtpPort:     rtpPort,
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
