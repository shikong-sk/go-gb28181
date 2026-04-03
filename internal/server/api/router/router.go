package router

import (
	"git.skcks.cn/Shikong/go-gb28181/internal/server/api/http"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/database"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter() *gin.Engine {
	return SetupRouterWithServices(nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

// SetupRouterWithServices 设置路由（带完整服务）
func SetupRouterWithServices(catalogService *service.CatalogService, playService *service.PlayService, alarmService *service.AlarmService, ptzService *service.PTZService, recordService *service.RecordService, subscriptionService *service.SubscriptionService, positionService *service.PositionService, statusService *service.DeviceStatusService, deviceService *service.DeviceService) *gin.Engine {
	router := gin.Default()

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 初始化仓库
	db := database.GetDB()
	deviceRepo := repository.NewDeviceRepository(db)
	channelRepo := repository.NewChannelRepository(db)

	// 初始化处理器
	deviceHandler := http.NewDeviceHandler(deviceRepo, catalogService)
	channelHandler := http.NewChannelHandler(channelRepo)
	playHandler := http.NewPlayHandler(playService)
	alarmHandler := http.NewAlarmHandler(alarmService)
	ptzHandler := http.NewPTZHandler(ptzService)
	recordHandler := http.NewRecordHandler(recordService)
	positionHandler := http.NewPositionHandler(positionService)
	statusHandler := http.NewDeviceStatusHandler(statusService, deviceService)

	// API 路由组
	api := router.Group("/api")
	{
		// 设备管理
		devices := api.Group("/devices")
		{
			devices.GET("", deviceHandler.List)
			devices.GET("/stats", deviceHandler.Stats)
			devices.GET("/:device_id", deviceHandler.GetDevice)
			devices.DELETE("/:device_id", deviceHandler.DeleteDevice)
			devices.POST("/:device_id/sync", deviceHandler.SyncCatalog)
			devices.GET("/:device_id/position", positionHandler.GetLatestPosition)
			devices.GET("/:device_id/positions", positionHandler.GetPositionHistory)
			devices.GET("/:device_id/status", statusHandler.GetDeviceStatus)
			devices.POST("/:device_id/query-status", statusHandler.QueryDeviceStatus)
		}

		// 通道管理
		channels := api.Group("/channels")
		{
			channels.GET("", channelHandler.List)
			channels.GET("/stats", channelHandler.Stats)
			channels.GET("/:channel_id", channelHandler.GetChannel)
			channels.DELETE("/:channel_id", channelHandler.DeleteChannel)
		}

		// 视频播放
		play := api.Group("/play")
		{
			play.POST("/start", playHandler.Play)
			play.POST("/stop", playHandler.Stop)
			play.POST("/playback", playHandler.PlayBack)
			play.GET("/sessions", playHandler.ListSessions)
			play.GET("/media/:stream_id", playHandler.GetMediaInfo)
		}

		// 报警管理
		alarms := api.Group("/alarms")
		{
			alarms.GET("", alarmHandler.List)
			alarms.GET("/config", alarmHandler.GetConfig)
			alarms.GET("/:id", alarmHandler.GetAlarm)
			alarms.DELETE("/:id", alarmHandler.DeleteAlarm)
		}

		// 云台控制
		ptz := api.Group("/ptz")
		{
			ptz.POST("/control", ptzHandler.Control)
			ptz.POST("/stop", ptzHandler.Stop)
		}

		// 历史录像
		record := api.Group("/record")
		{
			record.GET("/list", recordHandler.List)
		}
	}

	return router
}

// SetupRouterWithCatalog 设置路由（带目录同步服务）
// 已废弃，请使用 SetupRouterWithServices
func SetupRouterWithCatalog(catalogService *service.CatalogService) *gin.Engine {
	return SetupRouterWithServices(catalogService, nil, nil, nil, nil, nil, nil, nil, nil)
}
