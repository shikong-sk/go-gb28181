package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// TestSsrcService_PlaySsrc 测试实时播放 SSRC 分配
func TestSsrcService_PlaySsrc(t *testing.T) {
	// 创建 Redis 测试客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	// 测试 Redis 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis 未启动，跳过测试")
	}

	// 清理测试数据
	testDomain := "34020000002000000001"
	testKey := "gb28181:ssrc:" + testDomain
	redisClient.Del(ctx, testKey)

	// 创建 SSRC 服务
	service := NewSsrcService(redisClient, testDomain)

	// 等待初始化完成
	time.Sleep(100 * time.Millisecond)

	// 分配实时播放 SSRC
	ssrc, err := service.GetPlaySsrc()
	assert.NoError(t, err)
	assert.NotEmpty(t, ssrc)
	assert.Len(t, ssrc, 10)         // 0 + 9位序号
	assert.Equal(t, "0", ssrc[0:1]) // 首位应为 0

	t.Logf("分配实时播放 SSRC: %s", ssrc)

	// 释放 SSRC
	err = service.ReleaseSsrc(ssrc)
	assert.NoError(t, err)

	t.Logf("释放 SSRC: %s", ssrc)

	// 清理
	redisClient.Del(ctx, testKey)
	redisClient.Close()
}

// TestSsrcService_PlaybackSsrc 测试录像回放 SSRC 分配
func TestSsrcService_PlaybackSsrc(t *testing.T) {
	// 创建 Redis 测试客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	// 测试 Redis 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis 未启动，跳过测试")
	}

	// 清理测试数据
	testDomain := "34020000002000000001"
	testKey := "gb28181:ssrc:" + testDomain
	redisClient.Del(ctx, testKey)

	// 创建 SSRC 服务
	service := NewSsrcService(redisClient, testDomain)

	// 等待初始化完成
	time.Sleep(100 * time.Millisecond)

	// 分配录像回放 SSRC
	ssrc, err := service.GetPlaybackSsrc()
	assert.NoError(t, err)
	assert.NotEmpty(t, ssrc)
	assert.Len(t, ssrc, 10)         // 1 + 9位序号
	assert.Equal(t, "1", ssrc[0:1]) // 首位应为 1

	t.Logf("分配录像回放 SSRC: %s", ssrc)

	// 释放 SSRC
	err = service.ReleaseSsrc(ssrc)
	assert.NoError(t, err)

	t.Logf("释放 SSRC: %s", ssrc)

	// 清理
	redisClient.Del(ctx, testKey)
	redisClient.Close()
}

// TestSsrcService_PoolSize 测试 SSRC 池大小
func TestSsrcService_PoolSize(t *testing.T) {
	// 创建 Redis 测试客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	// 测试 Redis 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis 未启动，跳过测试")
	}

	// 清理测试数据
	testDomain := "34020000002000000001"
	testKey := "gb28181:ssrc:" + testDomain
	redisClient.Del(ctx, testKey)

	// 创建 SSRC 服务
	service := NewSsrcService(redisClient, testDomain)

	// 等待初始化完成
	time.Sleep(100 * time.Millisecond)

	// 检查池大小
	size, err := service.GetPoolSize()
	assert.NoError(t, err)
	assert.Equal(t, int64(9999), size) // 初始池大小应为 9999

	t.Logf("初始 SSRC 池大小: %d", size)

	// 分配一个 SSRC
	ssrc1, err := service.GetPlaySsrc()
	assert.NoError(t, err)

	// 再次检查池大小
	size, err = service.GetPoolSize()
	assert.NoError(t, err)
	assert.Equal(t, int64(9998), size) // 分配后应为 9998

	t.Logf("分配后 SSRC 池大小: %d", size)

	// 释放 SSRC
	err = service.ReleaseSsrc(ssrc1)
	assert.NoError(t, err)

	// 检查池大小恢复
	size, err = service.GetPoolSize()
	assert.NoError(t, err)
	assert.Equal(t, int64(9999), size) // 释放后恢复为 9999

	t.Logf("释放后 SSRC 池大小: %d", size)

	// 清理
	redisClient.Del(ctx, testKey)
	redisClient.Close()
}

// TestSsrcService_NoDuplicate 测试 SSRC 不重复分配
func TestSsrcService_NoDuplicate(t *testing.T) {
	// 创建 Redis 测试客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	// 测试 Redis 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis 未启动，跳过测试")
	}

	// 清理测试数据
	testDomain := "34020000002000000001"
	testKey := "gb28181:ssrc:" + testDomain
	redisClient.Del(ctx, testKey)

	// 创建 SSRC 服务
	service := NewSsrcService(redisClient, testDomain)

	// 等待初始化完成
	time.Sleep(100 * time.Millisecond)

	// 分配多个 SSRC 并检查是否重复
	ssrcMap := make(map[string]bool)
	for i := 0; i < 10; i++ {
		ssrc, err := service.GetPlaySsrc()
		assert.NoError(t, err)
		assert.False(t, ssrcMap[ssrc], "SSRC 不应重复: %s", ssrc)
		ssrcMap[ssrc] = true
	}

	t.Logf("成功分配 10 个不重复的 SSRC")

	// 释放所有 SSRC
	for ssrc := range ssrcMap {
		err := service.ReleaseSsrc(ssrc)
		assert.NoError(t, err)
	}

	// 清理
	redisClient.Del(ctx, testKey)
	redisClient.Close()
}
