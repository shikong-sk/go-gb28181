package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSubscriptionService_Subscribe(t *testing.T) {
	svc := NewSubscriptionService(30 * time.Second)

	tests := []struct {
		name     string
		cmdType  string
		deviceID string
		sn       string
		timeout  time.Duration
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "正常订阅",
			cmdType:  "RecordInfo",
			deviceID: "34020000001320000001",
			sn:       "123456",
			timeout:  10 * time.Second,
			wantErr:  false,
		},
		{
			name:     "使用默认超时",
			cmdType:  "DeviceStatus",
			deviceID: "34020000001320000002",
			sn:       "234567",
			timeout:  0, // 使用默认
			wantErr:  false,
		},
		{
			name:     "deviceID为空",
			cmdType:  "RecordInfo",
			deviceID: "",
			sn:       "123456",
			timeout:  10 * time.Second,
			wantErr:  true,
			errMsg:   "deviceID 和 sn 不能为空",
		},
		{
			name:     "sn为空",
			cmdType:  "RecordInfo",
			deviceID: "34020000001320000001",
			sn:       "",
			timeout:  10 * time.Second,
			wantErr:  true,
			errMsg:   "deviceID 和 sn 不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, err := svc.Subscribe(tt.cmdType, tt.deviceID, tt.sn, tt.timeout)

			if tt.wantErr {
				if err == nil {
					t.Errorf("期望返回错误，但没有")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("错误消息不匹配，got: %s, want: %s", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("不期望返回错误，但返回: %v", err)
				return
			}

			if sub == nil {
				t.Error("订阅不应为 nil")
				return
			}

			if sub.CmdType != tt.cmdType {
				t.Errorf("CmdType 不匹配，got: %s, want: %s", sub.CmdType, tt.cmdType)
			}
			if sub.DeviceID != tt.deviceID {
				t.Errorf("DeviceID 不匹配，got: %s, want: %s", sub.DeviceID, tt.deviceID)
			}
			if sub.SN != tt.sn {
				t.Errorf("SN 不匹配，got: %s, want: %s", sub.SN, tt.sn)
			}
			if sub.ResultCh == nil {
				t.Error("ResultCh 不应为 nil")
			}
			if sub.Context == nil {
				t.Error("Context 不应为 nil")
			}
			if sub.Cancel == nil {
				t.Error("Cancel 不应为 nil")
			}

			// 清理
			svc.Unsubscribe(tt.deviceID, tt.sn)
		})
	}
}

func TestSubscriptionService_DuplicateSubscribe(t *testing.T) {
	svc := NewSubscriptionService(30 * time.Second)

	deviceID := "34020000001320000001"
	sn := "123456"

	// 第一次订阅
	_, err := svc.Subscribe("RecordInfo", deviceID, sn, 10*time.Second)
	if err != nil {
		t.Fatalf("第一次订阅失败: %v", err)
	}

	// 第二次订阅相同 key，应该失败
	_, err = svc.Subscribe("RecordInfo", deviceID, sn, 10*time.Second)
	if err == nil {
		t.Error("期望重复订阅返回错误，但没有")
	}

	// 清理
	svc.Unsubscribe(deviceID, sn)
}

func TestSubscriptionService_Unsubscribe(t *testing.T) {
	svc := NewSubscriptionService(30 * time.Second)

	deviceID := "34020000001320000001"
	sn := "123456"

	// 创建订阅
	sub, err := svc.Subscribe("RecordInfo", deviceID, sn, 10*time.Second)
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}

	// 验证订阅存在
	_, exists := svc.GetSubscription(deviceID, sn)
	if !exists {
		t.Error("订阅应该存在")
	}

	// 取消订阅
	svc.Unsubscribe(deviceID, sn)

	// 验证订阅已删除
	_, exists = svc.GetSubscription(deviceID, sn)
	if exists {
		t.Error("订阅应该已被删除")
	}

	// 验证 context 已取消
	if sub.Context.Err() != context.Canceled {
		t.Error("订阅的 context 应该已被取消")
	}
}

func TestSubscriptionService_NotifyResponse(t *testing.T) {
	svc := NewSubscriptionService(30 * time.Second)

	deviceID := "34020000001320000001"
	sn := "123456"

	// 测试不存在的订阅
	result := svc.NotifyResponse("nonexistent", "123", "response")
	if result {
		t.Error("不存在的订阅应该返回 false")
	}

	// 创建订阅
	_, err := svc.Subscribe("RecordInfo", deviceID, sn, 10*time.Second)
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer svc.Unsubscribe(deviceID, sn)

	// 测试正常通知
	testResponse := map[string]string{"status": "ok"}
	result = svc.NotifyResponse(deviceID, sn, testResponse)
	if !result {
		t.Error("通知响应应该成功")
	}

	// 验证结果通道接收到响应
	sub, exists := svc.GetSubscription(deviceID, sn)
	if !exists {
		t.Fatal("订阅应该存在")
	}
	select {
	case r := <-sub.ResultCh:
		respMap, ok := r.(map[string]string)
		if !ok {
			t.Errorf("响应类型不匹配，got: %T", r)
		} else if respMap["status"] != "ok" {
			t.Errorf("响应内容不匹配，got: %v", respMap)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("应该能接收到响应")
	}
}

func TestSubscriptionService_NotifyResponse_ChannelFull(t *testing.T) {
	svc := NewSubscriptionService(30 * time.Second)

	deviceID := "34020000001320000001"
	sn := "123456"

	// 创建订阅
	_, err := svc.Subscribe("RecordInfo", deviceID, sn, 10*time.Second)
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer svc.Unsubscribe(deviceID, sn)

	// 填满通道（容量为1）
	svc.NotifyResponse(deviceID, sn, "first")

	// 再次写入应该失败（通道已满）
	result := svc.NotifyResponse(deviceID, sn, "second")
	if result {
		t.Error("通道已满时应该返回 false")
	}
}

func TestSubscriptionService_CleanupExpired(t *testing.T) {
	// 使用很短的超时时间
	svc := NewSubscriptionService(100 * time.Millisecond)

	deviceID := "34020000001320000001"
	sn := "123456"

	// 创建订阅
	_, err := svc.Subscribe("RecordInfo", deviceID, sn, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}

	// 验证订阅存在
	if svc.Count() != 1 {
		t.Errorf("订阅数量应为 1，实际为 %d", svc.Count())
	}

	// 等待超时
	time.Sleep(150 * time.Millisecond)

	// 清理过期订阅
	count := svc.CleanupExpired()
	if count != 1 {
		t.Errorf("应清理 1 个订阅，实际清理 %d 个", count)
	}

	// 验证订阅已删除
	if svc.Count() != 0 {
		t.Errorf("订阅数量应为 0，实际为 %d", svc.Count())
	}
}

func TestSubscriptionService_ConcurrentSafety(t *testing.T) {
	svc := NewSubscriptionService(30 * time.Second)

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	// 并发创建订阅
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				deviceID := "device_" + string(rune(id))
				sn := "sn_" + string(rune(j))
				_, _ = svc.Subscribe("RecordInfo", deviceID, sn, 10*time.Second)
				_ = svc.NotifyResponse(deviceID, sn, "response")
				svc.Unsubscribe(deviceID, sn)
			}
		}(i)
	}

	// 并发读取 Count
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = svc.Count()
			}
		}()
	}

	// 并发清理
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = svc.CleanupExpired()
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
}

func TestSubscriptionService_DefaultTimeout(t *testing.T) {
	// 测试默认超时设置
	svc := NewSubscriptionService(0) // 传入0，应使用默认值

	if svc.timeout <= 0 {
		t.Error("默认超时时间应该大于 0")
	}

	// 验证默认超时是 30 秒
	if svc.timeout != 30*time.Second {
		t.Errorf("默认超时时间应为 30s，实际为 %v", svc.timeout)
	}
}

func TestSubscriptionService_GetSubscription(t *testing.T) {
	svc := NewSubscriptionService(30 * time.Second)

	deviceID := "34020000001320000001"
	sn := "123456"

	// 获取不存在的订阅
	_, exists := svc.GetSubscription(deviceID, sn)
	if exists {
		t.Error("订阅不应该存在")
	}

	// 创建订阅
	_, err := svc.Subscribe("RecordInfo", deviceID, sn, 10*time.Second)
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer svc.Unsubscribe(deviceID, sn)

	// 获取存在的订阅
	sub, exists := svc.GetSubscription(deviceID, sn)
	if !exists {
		t.Error("订阅应该存在")
	}
	if sub == nil {
		t.Error("订阅不应为 nil")
	}
	if sub.CmdType != "RecordInfo" {
		t.Errorf("CmdType 应为 RecordInfo，实际为 %s", sub.CmdType)
	}
}

func TestSubscriptionService_Count(t *testing.T) {
	svc := NewSubscriptionService(30 * time.Second)

	// 初始计数应为 0
	if svc.Count() != 0 {
		t.Errorf("初始计数应为 0，实际为 %d", svc.Count())
	}

	// 创建多个订阅
	for i := 0; i < 5; i++ {
		deviceID := "device_" + string(rune('0'+i))
		sn := "sn_" + string(rune('0'+i))
		_, _ = svc.Subscribe("RecordInfo", deviceID, sn, 10*time.Second)
	}

	// 验证计数
	if svc.Count() != 5 {
		t.Errorf("计数应为 5，实际为 %d", svc.Count())
	}

	// 清理
	for i := 0; i < 5; i++ {
		deviceID := "device_" + string(rune('0'+i))
		sn := "sn_" + string(rune('0'+i))
		svc.Unsubscribe(deviceID, sn)
	}

	if svc.Count() != 0 {
		t.Errorf("清理后计数应为 0，实际为 %d", svc.Count())
	}
}
