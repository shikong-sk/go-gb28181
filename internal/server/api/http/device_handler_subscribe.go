package http

import (
	"strconv"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// SubscribeCatalog 订阅目录
// POST /api/devices/:device_id/subscribe/catalog
func (h *DeviceHandler) SubscribeCatalog(c *gin.Context) {
	deviceID := c.Param("device_id")

	// 获取过期时间参数（秒），默认 3600 秒（1小时）
	expiresStr := c.DefaultQuery("expires", "3600")
	expires, err := strconv.Atoi(expiresStr)
	if err != nil || expires <= 0 {
		c.JSON(400, gin.H{
			"error": "expires 参数无效，必须为正整数",
		})
		return
	}

	// 检查服务是否初始化
	if h.catalogSubscriptionService == nil {
		c.JSON(500, gin.H{
			"error": "目录订阅服务未初始化",
		})
		return
	}

	// 发起订阅
	if err := h.catalogSubscriptionService.Subscribe(deviceID, expires); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("订阅目录失败")
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	log.Info().Str("device_id", deviceID).Int("expires", expires).Msg("目录订阅成功")
	c.JSON(200, gin.H{
		"message":   "目录订阅成功",
		"device_id": deviceID,
		"expires":   expires,
	})
}

// UnsubscribeCatalog 取消目录订阅
// DELETE /api/devices/:device_id/subscribe/catalog
func (h *DeviceHandler) UnsubscribeCatalog(c *gin.Context) {
	deviceID := c.Param("device_id")
	sn := c.Query("sn")

	// 如果没有提供 sn，则取消该设备的所有订阅
	if sn == "" {
		// 获取该设备的所有订阅
		subscriptions := h.catalogSubscriptionService.ListSubscriptions()
		var count int
		for _, sub := range subscriptions {
			if sub.DeviceID == deviceID {
				if err := h.catalogSubscriptionService.Unsubscribe(deviceID, sub.SN); err != nil {
					log.Error().Err(err).Str("device_id", deviceID).Str("sn", sub.SN).Msg("取消订阅失败")
					continue
				}
				count++
			}
		}
		log.Info().Str("device_id", deviceID).Int("count", count).Msg("取消所有目录订阅")
		c.JSON(200, gin.H{
			"message":   "取消订阅成功",
			"device_id": deviceID,
			"count":     count,
		})
		return
	}

	// 取消指定订阅
	if err := h.catalogSubscriptionService.Unsubscribe(deviceID, sn); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Str("sn", sn).Msg("取消订阅失败")
		c.JSON(404, gin.H{
			"error": err.Error(),
		})
		return
	}

	log.Info().Str("device_id", deviceID).Str("sn", sn).Msg("取消目录订阅成功")
	c.JSON(200, gin.H{
		"message":   "取消订阅成功",
		"device_id": deviceID,
		"sn":        sn,
	})
}

// ListSubscriptions 列出所有订阅
// GET /api/subscriptions
func (h *DeviceHandler) ListSubscriptions(c *gin.Context) {
	// 检查服务是否初始化
	if h.catalogSubscriptionService == nil {
		c.JSON(500, gin.H{
			"error": "目录订阅服务未初始化",
		})
		return
	}

	subscriptions := h.catalogSubscriptionService.ListSubscriptions()

	// 转换为响应格式
	var list []gin.H
	for _, sub := range subscriptions {
		list = append(list, gin.H{
			"device_id":     sub.DeviceID,
			"sn":            sub.SN,
			"event_id":      sub.EventID,
			"expires":       sub.Expires,
			"subscribed_at": sub.SubscribedAt,
		})
	}

	c.JSON(200, gin.H{
		"total": len(list),
		"list":  list,
	})
}
