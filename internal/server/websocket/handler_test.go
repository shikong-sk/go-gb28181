package websocket

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// TestNewWebSocketHandler 测试 Handler 创建
func TestNewWebSocketHandler(t *testing.T) {
	manager := NewWebSocketManager()
	handler := NewWebSocketHandler(manager)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.manager)
	assert.NotNil(t, handler.upgrader)
	assert.True(t, handler.upgrader.CheckOrigin(nil))
}

// TestHandleWebSocket_Upgrade 测试 WebSocket 升级
func TestHandleWebSocket_Upgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewWebSocketManager()
	defer manager.Stop()
	manager.Start()

	handler := NewWebSocketHandler(manager)

	// 创建 Gin 路由
	router := gin.New()
	router.GET("/ws", handler.HandleWebSocket)

	// 创建测试服务器
	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 WebSocket 端点
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	// 等待注册完成
	time.Sleep(100 * time.Millisecond)

	// 验证客户端已注册
	assert.Equal(t, 1, manager.GetClientCount())

	// 关闭连接
	conn.Close()
	time.Sleep(100 * time.Millisecond)

	// 验证客户端已注销
	assert.Equal(t, 0, manager.GetClientCount())
}

// TestHandleWebSocket_PongHandler 测试 Pong 处理
func TestHandleWebSocket_PongHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewWebSocketManager()
	defer manager.Stop()
	manager.Start()

	handler := NewWebSocketHandler(manager)

	// 创建 Gin 路由
	router := gin.New()
	router.GET("/ws", handler.HandleWebSocket)

	// 创建测试服务器
	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 WebSocket 端点
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// 等待注册完成
	time.Sleep(100 * time.Millisecond)

	// 发送 pong 消息（验证处理器正常工作）
	err = conn.WriteMessage(websocket.PongMessage, []byte{})
	assert.NoError(t, err)

	// 等待一下确保不会崩溃
	time.Sleep(50 * time.Millisecond)

	// 验证连接仍然存活
	assert.Equal(t, 1, manager.GetClientCount())
}

// TestHandleWebSocket_MessageRead 测试消息读取
func TestHandleWebSocket_MessageRead(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewWebSocketManager()
	defer manager.Stop()
	manager.Start()

	handler := NewWebSocketHandler(manager)

	// 创建 Gin 路由
	router := gin.New()
	router.GET("/ws", handler.HandleWebSocket)

	// 创建测试服务器
	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 WebSocket 端点
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// 等待注册完成
	time.Sleep(100 * time.Millisecond)

	// 发送测试消息
	err = conn.WriteMessage(websocket.TextMessage, []byte("test message"))
	assert.NoError(t, err)

	// 等待消息被读取
	time.Sleep(50 * time.Millisecond)

	// 验证连接仍然存活
	assert.Equal(t, 1, manager.GetClientCount())
}

// TestHandleWebSocket_BroadcastToClient 测试广播消息到客户端
func TestHandleWebSocket_BroadcastToClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewWebSocketManager()
	defer manager.Stop()
	manager.Start()

	handler := NewWebSocketHandler(manager)

	// 创建 Gin 路由
	router := gin.New()
	router.GET("/ws", handler.HandleWebSocket)

	// 创建测试服务器
	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 WebSocket 端点
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// 等待注册完成
	time.Sleep(100 * time.Millisecond)

	// 广播消息
	testMessage := []byte("{\"type\":\"test\",\"data\":\"hello\"}")
	manager.Broadcast(testMessage)

	// 等待消息发送
	time.Sleep(100 * time.Millisecond)

	// 读取消息
	messageType, message, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, websocket.TextMessage, messageType)
	assert.Equal(t, testMessage, message)
}

// TestHandleWebSocket_MultipleClients 测试多个客户端连接
func TestHandleWebSocket_MultipleClients(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewWebSocketManager()
	defer manager.Stop()
	manager.Start()

	handler := NewWebSocketHandler(manager)

	// 创建 Gin 路由
	router := gin.New()
	router.GET("/ws", handler.HandleWebSocket)

	// 创建测试服务器
	server := httptest.NewServer(router)
	defer server.Close()

	// 连接多个客户端
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	clients := make([]*websocket.Conn, 3)
	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		clients[i] = conn
	}

	// 等待所有客户端注册完成
	time.Sleep(200 * time.Millisecond)

	// 验证所有客户端已注册
	assert.Equal(t, 3, manager.GetClientCount())

	// 关闭所有连接
	for _, conn := range clients {
		conn.Close()
	}

	// 等待所有客户端注销完成
	time.Sleep(200 * time.Millisecond)

	// 验证所有客户端已注销
	assert.Equal(t, 0, manager.GetClientCount())
}

// TestHandleWebSocket_AbnormalClose 测试异常关闭处理
func TestHandleWebSocket_AbnormalClose(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewWebSocketManager()
	defer manager.Stop()
	manager.Start()

	handler := NewWebSocketHandler(manager)

	// 创建 Gin 路由
	router := gin.New()
	router.GET("/ws", handler.HandleWebSocket)

	// 创建测试服务器
	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 WebSocket 端点
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)

	// 等待注册完成
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, manager.GetClientCount())

	// 异常关闭（不发送关闭帧）
	conn.Close()

	// 等待客户端注销
	time.Sleep(100 * time.Millisecond)

	// 验证客户端已正确注销
	assert.Equal(t, 0, manager.GetClientCount())
}
