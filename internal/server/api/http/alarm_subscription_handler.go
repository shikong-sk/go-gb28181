package http

import (
	"net/http"
	"strconv"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// AlarmSubscriptionHandler 报警订阅处理器
type AlarmSubscriptionHandler struct {
	alarmSubscriptionService *service.AlarmSubscriptionService
}

// NewAlarmSubscriptionHandler 创建报警订阅处理器
func NewAlarmSubscriptionHandler(alarmSubscriptionService *service.AlarmSubscriptionService) *AlarmSubscriptionHandler {
	return &AlarmSubscriptionHandler{
		alarmSubscriptionService: alarmSubscriptionService,
	}
}

// Subscribe 报警订阅
// POST /api/devices/:device_id/subscribe/alarm
func (h *AlarmSubscriptionHandler) Subscribe(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备ID不能为空"})
		return
	}

	// 解析过期时间参数
	expiresStr := c.DefaultQuery("expires", "3600") // 默认 1 小时
	expires, err := strconv.Atoi(expiresStr)
	if err != nil || expires <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "过期时间格式错误"})
		return
	}

	// 发起订阅
	if err := h.alarmSubscriptionService.Subscribe(deviceID, expires); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("发起报警订阅失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().Str("device_id", deviceID).Int("expires", expires).Msg("报警订阅成功")
	c.JSON(http.StatusOK, gin.H{
		"message":   "报警订阅成功",
		"device_id": deviceID,
		"expires":   expires,
	})
}

// Unsubscribe 取消报警订阅
// DELETE /api/devices/:device_id/subscribe/alarm
func (h *AlarmSubscriptionHandler) Unsubscribe(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备ID不能为空"})
		return
	}

	sn := c.Query("sn")
	if sn == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订阅SN不能为空"})
		return
	}

	// 取消订阅
	if err := h.alarmSubscriptionService.Unsubscribe(deviceID, sn); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Str("sn", sn).Msg("取消报警订阅失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().Str("device_id", deviceID).Str("sn", sn).Msg("报警订阅已取消")
	c.JSON(http.StatusOK, gin.H{"message": "报警订阅已取消"})
}

// ListSubscriptions 列出所有报警订阅
// GET /api/alarms/subscriptions
func (h *AlarmSubscriptionHandler) ListSubscriptions(c *gin.Context) {
	subscriptions := h.alarmSubscriptionService.ListSubscriptions()

	// 构建响应数据
	result := make([]gin.H, 0, len(subscriptions))
	for _, sub := range subscriptions {
		result = append(result, gin.H{
			"device_id":     sub.DeviceID,
			"sn":            sub.SN,
			"subscribe_id":  sub.SubscribeID,
			"expires":       sub.Expires,
			"subscribed_at": sub.SubscribedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total":         len(result),
		"subscriptions": result,
	})
}

// GlobalSubscribe 全局报警订阅（订阅所有在线设备）
// POST /api/alarms/subscribe
func (h *AlarmSubscriptionHandler) GlobalSubscribe(c *gin.Context) {
	// 解析过期时间参数
	expiresStr := c.DefaultQuery("expires", "3600") // 默认 1 小时
	expires, err := strconv.Atoi(expiresStr)
	if err != nil || expires <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "过期时间格式错误"})
		return
	}

	// TODO: 实现全局订阅逻辑（需要从 DeviceService 获取所有在线设备列表）
	// 当前版本返回提示信息
	c.JSON(http.StatusOK, gin.H{
		"message": "全局报警订阅功能待实现，请使用设备级订阅",
		"hint":    "POST /api/devices/:device_id/subscribe/alarm",
	})
}
