package device

import (
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/emiago/sipgo"
	"time"
)

var keepaliveTimer *time.Ticker

type SendKeepAlive = func(client *sipgo.Client)

var senders = make(map[string]SendKeepAlive)

func AddKeepaliveSender(deviceId string, sender SendKeepAlive) {
	log.Log().Info().Msgf("添加 keealive 发送器, deviceId: %s", deviceId)
	senders[deviceId] = sender
}

// StartKeepAlive 启动 keepalive 定时器
// 每 30 秒发送一次 keepalive 消息到服务器
func StartKeepAlive(client *sipgo.Client) {
	keepaliveTimer = time.NewTicker(time.Second * 30)
	go func() {
		for range keepaliveTimer.C {
			// 遍历所有注册的设备，发送 keepalive 消息
			for deviceId, sender := range senders {
				log.Log().Debug().Msgf("执行 发送 keealive 消息, deviceId: %s", deviceId)
				go sender(client)
			}
		}
	}()
}

// StopKeepAlive 停止 keepalive 定时器
func StopKeepAlive() {
	if keepaliveTimer == nil {
		return
	}
	keepaliveTimer.Stop()
}
