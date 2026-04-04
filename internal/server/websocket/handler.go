package websocket

import (
	"net/http"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketHandler WebSocket HTTP 处理器
type WebSocketHandler struct {
	manager  *WebSocketManager
	upgrader websocket.Upgrader
}

// NewWebSocketHandler 创建 WebSocket 处理器
func NewWebSocketHandler(manager *WebSocketManager) *WebSocketHandler {
	return &WebSocketHandler{
		manager: manager,
		upgrader: websocket.Upgrader{
			// 允许所有来源连接
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			// 缓冲区大小
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// HandleWebSocket 处理 WebSocket 连接升级
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// 1. 升级 HTTP 连接为 WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket 升级失败")
		return
	}

	// 2. 创建 WebSocketClient
	clientID := utils.GenerateSN()
	client := NewWebSocketClient(clientID, conn)

	// 3. 注册到 Manager
	h.manager.Register(client)

	log.Info().
		Str("client_id", clientID).
		Str("remote_addr", c.Request.RemoteAddr).
		Msg("WebSocket 客户端连接成功")

	// 4. 启动读取 goroutine (处理 pong)
	go h.readMessages(client)

	// 5. 启动写入 goroutine (发送消息)
	go h.writeMessages(client)
}

// readMessages 读取客户端消息
func (h *WebSocketHandler) readMessages(client *WebSocketClient) {
	defer func() {
		// 连接断开时注销客户端
		h.manager.Unregister(client)
		client.Close()
	}()

	// 设置 pong 处理器
	client.Conn.SetPongHandler(func(appData string) error {
		log.Debug().
			Str("client_id", client.ID).
			Msg("收到客户端 pong 响应")
		return nil
	})

	// 循环读取客户端消息
	for {
		select {
		case <-client.ctx.Done():
			return
		default:
			messageType, message, err := client.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error().
						Err(err).
						Str("client_id", client.ID).
						Msg("WebSocket 读取错误")
				} else {
					log.Info().
						Str("client_id", client.ID).
						Msg("WebSocket 客户端关闭连接")
				}
				return
			}

			// 处理客户端消息（目前只记录，不处理业务）
			log.Debug().
				Str("client_id", client.ID).
				Int("message_type", messageType).
				Int("message_len", len(message)).
				Msg("收到 WebSocket 消息")
		}
	}
}

// writeMessages 向客户端发送消息
func (h *WebSocketHandler) writeMessages(client *WebSocketClient) {
	for {
		select {
		case <-client.ctx.Done():
			// Context 已取消，尝试发送关闭消息
			err := client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			if err != nil {
				log.Debug().
					Err(err).
					Str("client_id", client.ID).
					Msg("发送关闭消息失败")
			}
			return

		case message, ok := <-client.SendChan:
			// 设置写超时
			err := client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				log.Error().
					Err(err).
					Str("client_id", client.ID).
					Msg("设置写超时失败")
				return
			}

			if !ok {
				// 通道已关闭，发送关闭消息
				err := client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					log.Debug().
						Err(err).
						Str("client_id", client.ID).
						Msg("发送关闭消息失败")
				}
				return
			}

			// 发送消息
			err = client.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Error().
					Err(err).
					Str("client_id", client.ID).
					Msg("发送消息失败")
				return
			}

			log.Debug().
				Str("client_id", client.ID).
				Int("message_len", len(message)).
				Msg("消息已发送到客户端")
		}
	}
}
