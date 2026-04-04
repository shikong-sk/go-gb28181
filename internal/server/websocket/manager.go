package websocket

import (
	"context"
	"sync"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gorilla/websocket"
)

// WebSocketClient 表示一个 WebSocket 客户端连接
type WebSocketClient struct {
	ID        string          // 客户端唯一标识
	Conn      *websocket.Conn // WebSocket 连接
	SendChan  chan []byte     // 发送消息通道
	CloseChan chan struct{}   // 关闭信号通道
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once // 确保 Close 只执行一次
}

// NewWebSocketClient 创建 WebSocket 客户端
func NewWebSocketClient(id string, conn *websocket.Conn) *WebSocketClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketClient{
		ID:        id,
		Conn:      conn,
		SendChan:  make(chan []byte, 100),
		CloseChan: make(chan struct{}),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Close 关闭客户端连接
func (c *WebSocketClient) Close() {
	c.closeOnce.Do(func() {
		if c.cancel != nil {
			c.cancel()
		}
		close(c.CloseChan)
		close(c.SendChan)
		if c.Conn != nil {
			c.Conn.Close()
		}
	})
}

// WebSocketManager WebSocket 连接管理器
type WebSocketManager struct {
	clients    sync.Map // key: clientID, value: *WebSocketClient
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan []byte

	// 心跳配置
	pingInterval time.Duration // 默认 30s
	writeTimeout time.Duration // 默认 10s

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewWebSocketManager 创建 WebSocket 管理器
func NewWebSocketManager() *WebSocketManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketManager{
		register:     make(chan *WebSocketClient, 10),
		unregister:   make(chan *WebSocketClient, 10),
		broadcast:    make(chan []byte, 100),
		pingInterval: 30 * time.Second,
		writeTimeout: 10 * time.Second,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Run 启动 WebSocket 管理器主循环
func (m *WebSocketManager) Run() {
	// 启动心跳定时器
	pingTicker := time.NewTicker(m.pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			log.Info().Msg("WebSocket 管理器停止")
			return

		case client := <-m.register:
			// 注册新客户端
			m.clients.Store(client.ID, client)
			log.Info().
				Str("client_id", client.ID).
				Msg("WebSocket 客户端已连接")

		case client := <-m.unregister:
			// 注销客户端
			if _, ok := m.clients.LoadAndDelete(client.ID); ok {
				log.Info().
					Str("client_id", client.ID).
					Msg("WebSocket 客户端已断开")
			}

		case message := <-m.broadcast:
			// 广播消息到所有客户端
			m.broadcastMessage(message)

		case <-pingTicker.C:
			// 定时发送心跳
			m.SendPing()
		}
	}
}

// Register 注册新客户端
func (m *WebSocketManager) Register(client *WebSocketClient) {
	m.register <- client
}

// Unregister 注销客户端
func (m *WebSocketManager) Unregister(client *WebSocketClient) {
	m.unregister <- client
}

// Broadcast 向所有客户端广播消息
func (m *WebSocketManager) Broadcast(message []byte) {
	m.broadcast <- message
}

// SendPing 向所有客户端发送心跳
func (m *WebSocketManager) SendPing() {
	m.clients.Range(func(key, value interface{}) bool {
		client := value.(*WebSocketClient)

		// 设置写超时
		err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(m.writeTimeout))
		if err != nil {
			log.Error().
				Err(err).
				Str("client_id", client.ID).
				Msg("发送心跳失败")
			// 心跳失败，关闭连接
			go m.unregisterClient(client)
		}

		return true
	})
}

// unregisterClient 异步注销客户端
func (m *WebSocketManager) unregisterClient(client *WebSocketClient) {
	select {
	case m.unregister <- client:
	case <-m.ctx.Done():
	}
}

// broadcastMessage 向所有客户端发送消息
func (m *WebSocketManager) broadcastMessage(message []byte) {
	m.clients.Range(func(key, value interface{}) bool {
		client := value.(*WebSocketClient)

		select {
		case client.SendChan <- message:
			// 消息已发送到客户端发送通道
		case <-m.ctx.Done():
			return false
		default:
			// 发送通道已满，客户端可能阻塞
			log.Warn().
				Str("client_id", client.ID).
				Msg("客户端发送通道已满，跳过消息")
		}

		return true
	})
}

// Start 启动管理器
func (m *WebSocketManager) Start() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.Run()
	}()
}

// Stop 停止管理器
func (m *WebSocketManager) Stop() {
	// 取消上下文
	m.cancel()

	// 关闭所有客户端连接
	m.clients.Range(func(key, value interface{}) bool {
		client := value.(*WebSocketClient)
		client.Close()
		return true
	})

	// 等待主循环退出
	m.wg.Wait()

	log.Info().Msg("WebSocket 管理器已停止")
}

// GetClientCount 获取当前连接的客户端数量
func (m *WebSocketManager) GetClientCount() int {
	count := 0
	m.clients.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
