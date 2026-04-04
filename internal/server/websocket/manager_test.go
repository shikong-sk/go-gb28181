package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// TestNewWebSocketManager 测试管理器创建
func TestNewWebSocketManager(t *testing.T) {
	manager := NewWebSocketManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.register)
	assert.NotNil(t, manager.unregister)
	assert.NotNil(t, manager.broadcast)
	assert.Equal(t, 30*time.Second, manager.pingInterval)
	assert.Equal(t, 10*time.Second, manager.writeTimeout)
}

// TestWebSocketManager_StartStop 测试启动和停止
func TestWebSocketManager_StartStop(t *testing.T) {
	manager := NewWebSocketManager()

	// 启动管理器
	manager.Start()
	time.Sleep(50 * time.Millisecond)

	// 验证管理器正在运行
	assert.NoError(t, manager.ctx.Err())

	// 停止管理器
	manager.Stop()

	// 验证上下文已取消
	assert.Error(t, manager.ctx.Err(), context.Canceled)
}

// TestWebSocketManager_RegisterUnregister 测试注册和注销
func TestWebSocketManager_RegisterUnregister(t *testing.T) {
	manager := NewWebSocketManager()
	defer manager.Stop()

	manager.Start()

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		client := NewWebSocketClient("test-client-1", conn)
		manager.Register(client)

		// 等待注销信号
		<-client.CloseChan
	}))
	defer server.Close()

	// 连接到测试服务器
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)

	// 等待注册完成
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, manager.GetClientCount())

	// 关闭连接
	conn.Close()
	time.Sleep(100 * time.Millisecond)
}

// TestWebSocketManager_Broadcast 测试广播功能
func TestWebSocketManager_Broadcast(t *testing.T) {
	manager := NewWebSocketManager()
	defer manager.Stop()

	manager.Start()

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		client := NewWebSocketClient("test-client", conn)
		manager.Register(client)

		// 保持连接
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
	defer server.Close()

	// 连接到测试服务器
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// 等待注册完成
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, manager.GetClientCount())

	// 广播消息（验证不会崩溃）
	manager.Broadcast([]byte("test message"))
	time.Sleep(50 * time.Millisecond)
}

// TestWebSocketManager_SendPing 测试心跳功能
func TestWebSocketManager_SendPing(t *testing.T) {
	manager := NewWebSocketManager()
	defer manager.Stop()

	manager.Start()

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		client := NewWebSocketClient("test-client", conn)
		manager.Register(client)

		// 保持连接
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
	defer server.Close()

	// 连接到测试服务器
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// 等待注册完成
	time.Sleep(100 * time.Millisecond)

	// 手动触发心跳（验证不会崩溃）
	manager.SendPing()
	time.Sleep(50 * time.Millisecond)
}

// TestWebSocketClient_New 测试客户端创建
func TestWebSocketClient_New(t *testing.T) {
	// 创建模拟连接
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	client := NewWebSocketClient("test-id", conn)
	assert.NotNil(t, client)
	assert.Equal(t, "test-id", client.ID)
	assert.NotNil(t, client.Conn)
	assert.NotNil(t, client.SendChan)
	assert.NotNil(t, client.CloseChan)
	assert.NotNil(t, client.ctx)
	assert.NotNil(t, client.cancel)
}

// TestWebSocketClient_Close 测试客户端关闭
func TestWebSocketClient_Close(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)

	client := NewWebSocketClient("test", conn)
	client.Close()

	// 验证上下文已取消
	assert.Error(t, client.ctx.Err(), context.Canceled)
}

// TestWebSocketManager_GetClientCount 测试客户端计数
func TestWebSocketManager_GetClientCount(t *testing.T) {
	manager := NewWebSocketManager()

	// 验证初始计数为 0
	assert.Equal(t, 0, manager.GetClientCount())

	// 手动添加客户端到 sync.Map（仅用于测试计数功能）
	mockConn1 := &websocket.Conn{}
	mockConn2 := &websocket.Conn{}

	client1 := NewWebSocketClient("client-1", mockConn1)
	client2 := NewWebSocketClient("client-2", mockConn2)

	manager.clients.Store(client1.ID, client1)
	manager.clients.Store(client2.ID, client2)

	assert.Equal(t, 2, manager.GetClientCount())

	// 删除一个客户端
	manager.clients.Delete(client1.ID)
	assert.Equal(t, 1, manager.GetClientCount())
}

// TestWebSocketManager_ConcurrentBroadcast 测试并发广播
func TestWebSocketManager_ConcurrentBroadcast(t *testing.T) {
	manager := NewWebSocketManager()
	defer manager.Stop()

	manager.Start()

	// 并发广播多条消息
	for i := 0; i < 10; i++ {
		go func() {
			manager.Broadcast([]byte("test message"))
		}()
	}

	time.Sleep(100 * time.Millisecond)

	// 验证管理器仍然正常运行
	assert.NoError(t, manager.ctx.Err())
}
