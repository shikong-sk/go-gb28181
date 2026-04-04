package service

import (
	"context"
	"fmt"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/redis/go-redis/v9"
)

// SsrcService SSRC 管理服务 - 基于 Redis 的 SSRC 池化管理
type SsrcService struct {
	redisClient *redis.Client
	domain      string // SIP 域编码（20位国标编码）
	ssrcKey     string // Redis key for SSRC pool
}

// NewSsrcService 创建 SSRC 服务并初始化 SSRC 池
func NewSsrcService(redisClient *redis.Client, domain string) *SsrcService {
	// 提取域编码第3-7位作为 SSRC 前缀（共5位）
	// 例如：34020000002000000001 -> 20000
	if len(domain) < 8 {
		log.Warn().Str("domain", domain).Msg("域编码长度不足，使用默认前缀")
		domain = "34020000002000000001" // 使用默认值
	}

	ssrcPrefix := domain[3:8] // 第3-7位（索引3-7），共5位
	ssrcKey := fmt.Sprintf("gb28181:ssrc:%s", domain)

	// 创建服务实例
	service := &SsrcService{
		redisClient: redisClient,
		domain:      domain,
		ssrcKey:     ssrcKey,
	}

	// 初始化 SSRC 池（异步执行，避免阻塞启动）
	go service.initSsrcPool(ssrcPrefix)

	log.Info().
		Str("domain", domain).
		Str("ssrc_prefix", ssrcPrefix).
		Str("redis_key", ssrcKey).
		Msg("SSRC 服务初始化")

	return service
}

// initSsrcPool 初始化 SSRC 池（0001-9999）
func (s *SsrcService) initSsrcPool(ssrcPrefix string) {
	ctx := context.Background()

	// 检查池是否已初始化
	exists, err := s.redisClient.Exists(ctx, s.ssrcKey).Result()
	if err != nil {
		log.Error().Err(err).Msg("检查 SSRC 池失败")
		return
	}

	if exists > 0 {
		log.Info().Msg("SSRC 池已存在，跳过初始化")
		return
	}

	// 生成 SSRC 序号列表（前缀+序号，共9位，0001-9999，共9999个）
	ssrcList := make([]interface{}, 0, 9999)
	for i := 1; i < 10000; i++ {
		// 格式化为：前缀（5位）+ 序号（4位）= 共9位
		// 例如：20000 + 0001 = 200000001
		ssrcSn := fmt.Sprintf("%s%04d", ssrcPrefix, i)
		ssrcList = append(ssrcList, ssrcSn)
	}

	// 批量添加到 Redis Set
	if err := s.redisClient.SAdd(ctx, s.ssrcKey, ssrcList...).Err(); err != nil {
		log.Error().Err(err).Msg("初始化 SSRC 池失败")
		return
	}

	log.Info().
		Int("count", 9999).
		Str("prefix", ssrcPrefix).
		Msg("SSRC 池初始化完成")
}

// GetPlaySsrc 分配实时播放 SSRC（首位为 0）
// 返回格式：0 + Redis存储内容（前缀+序号，9位）= 10位 SSRC
func (s *SsrcService) GetPlaySsrc() (string, error) {
	ctx := context.Background()

	// 检查池剩余数量，低于阈值告警
	if size, err := s.GetPoolSize(); err == nil && size < 100 {
		log.Warn().Int64("remaining", size).Msg("SSRC 池即将耗尽")
	}

	// 从 Redis Set 原子弹出一个 SSRC 序号（格式：前缀+序号，共9位）
	sn, err := s.redisClient.SPop(ctx, s.ssrcKey).Result()
	if err != nil {
		if err == redis.Nil {
			// SSRC 池耗尽是严重错误，影响播放功能
			log.Error().Msg("SSRC 池已耗尽，无法分配新的实时播放会话")
			return "", fmt.Errorf("SSRC 池已耗尽")
		}
		log.Error().Err(err).Msg("分配 SSRC 失败")
		return "", fmt.Errorf("分配 SSRC 失败: %w", err)
	}

	// 实时播放 SSRC 格式：0 + 前缀+序号（9位）= 10位
	// 例如：0 + 200000001 = 0200000001
	ssrc := fmt.Sprintf("0%s", sn)

	log.Debug().
		Str("ssrc", ssrc).
		Str("sn", sn).
		Msg("分配实时播放 SSRC")

	return ssrc, nil
}

// GetPlaybackSsrc 分配录像回放 SSRC（首位为 1）
// 返回格式：1 + Redis存储内容（前缀+序号，9位）= 10位 SSRC
func (s *SsrcService) GetPlaybackSsrc() (string, error) {
	ctx := context.Background()

	// 检查池剩余数量，低于阈值告警
	if size, err := s.GetPoolSize(); err == nil && size < 100 {
		log.Warn().Int64("remaining", size).Msg("SSRC 池即将耗尽")
	}

	// 从 Redis Set 原子弹出一个 SSRC 序号（格式：前缀+序号，共9位）
	sn, err := s.redisClient.SPop(ctx, s.ssrcKey).Result()
	if err != nil {
		if err == redis.Nil {
			// SSRC 池耗尽是严重错误，影响播放功能
			log.Error().Msg("SSRC 池已耗尽，无法分配新的录像回放会话")
			return "", fmt.Errorf("SSRC 池已耗尽")
		}
		log.Error().Err(err).Msg("分配 SSRC 失败")
		return "", fmt.Errorf("分配 SSRC 失败: %w", err)
	}

	// 录像回放 SSRC 格式：1 + 前缀+序号（9位）= 10位
	// 例如：1 + 200000001 = 1200000001
	ssrc := fmt.Sprintf("1%s", sn)

	log.Debug().
		Str("ssrc", ssrc).
		Str("sn", sn).
		Msg("分配录像回放 SSRC")

	return ssrc, nil
}

// GetDownloadSsrc 分配录像下载 SSRC（首位为 2，符合 GB28181-2016 附录A.4.2）
// 返回格式：2 + Redis存储内容（前缀+序号，9位）= 10位 SSRC
func (s *SsrcService) GetDownloadSsrc() (string, error) {
	ctx := context.Background()

	// 检查池剩余数量，低于阈值告警
	if size, err := s.GetPoolSize(); err == nil && size < 100 {
		log.Warn().Int64("remaining", size).Msg("SSRC 池即将耗尽")
	}

	// 从 Redis Set 原子弹出一个 SSRC 序号（格式：前缀+序号，共9位）
	sn, err := s.redisClient.SPop(ctx, s.ssrcKey).Result()
	if err != nil {
		if err == redis.Nil {
			// SSRC 池耗尽是严重错误，影响播放功能
			log.Error().Msg("SSRC 池已耗尽，无法分配新的录像下载会话")
			return "", fmt.Errorf("SSRC 池已耗尽")
		}
		log.Error().Err(err).Msg("分配 SSRC 失败")
		return "", fmt.Errorf("分配 SSRC 失败: %w", err)
	}

	// 录像下载 SSRC 格式：2 + 前缀+序号（9位）= 10位
	// 例如：2 + 200000001 = 2200000001
	ssrc := fmt.Sprintf("2%s", sn)

	log.Debug().
		Str("ssrc", ssrc).
		Str("sn", sn).
		Msg("分配录像下载 SSRC")

	return ssrc, nil
}

// ReleaseSsrc 释放 SSRC（归还到池）
// 输入格式：0/1 + 前缀+序号（共10位）
// 归还格式：去掉首位标识，归还 前缀+序号 部分（共9位）
func (s *SsrcService) ReleaseSsrc(ssrc string) error {
	ctx := context.Background()

	// SSRC 格式验证（应为10位：首位标识 + 前缀+序号）
	if len(ssrc) != 10 {
		log.Warn().Str("ssrc", ssrc).Msg("SSRC 格式错误，无法释放")
		return fmt.Errorf("SSRC 格式错误: %s", ssrc)
	}

	// 提取序号部分（去掉首位标识，保留剩余9位）
	// 例如：0200000001 -> 200000001
	sn := ssrc[1:]

	// 归还到 Redis Set
	if err := s.redisClient.SAdd(ctx, s.ssrcKey, sn).Err(); err != nil {
		log.Error().Err(err).Str("ssrc", ssrc).Msg("释放 SSRC 失败")
		return fmt.Errorf("释放 SSRC 失败: %w", err)
	}

	log.Debug().
		Str("ssrc", ssrc).
		Str("sn", sn).
		Msg("释放 SSRC 成功")

	return nil
}

// GetPoolSize 获取 SSRC 池剩余数量
func (s *SsrcService) GetPoolSize() (int64, error) {
	ctx := context.Background()
	size, err := s.redisClient.SCard(ctx, s.ssrcKey).Result()
	if err != nil {
		log.Error().Err(err).Msg("获取 SSRC 池大小失败")
		return 0, fmt.Errorf("获取 SSRC 池大小失败: %w", err)
	}
	return size, nil
}
