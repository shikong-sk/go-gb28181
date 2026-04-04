package service

import (
	"fmt"
	"sync"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
)

// DownloadService 录像下载服务
type DownloadService struct {
	playService *PlayService
	sessions    map[string]*DownloadSession // 下载会话（stream_id -> session）
	mu          sync.RWMutex
}

// DownloadSession 下载会话
type DownloadSession struct {
	StreamId  string    // 流 ID
	DeviceId  string    // 设备 ID
	ChannelId string    // 通道 ID
	StartTime time.Time // 下载开始时间
	EndTime   time.Time // 录像结束时间（计划）
	Speed     int       // 下载倍速
	Status    string    // 状态: pending, downloading, completed, cancelled, error
	Progress  float64   // 下载进度（0-100）
	CreatedAt time.Time // 会话创建时间
	UpdatedAt time.Time // 最后更新时间
	Error     string    // 错误信息
}

// 下载状态常量
const (
	DownloadStatusPending     = "pending"
	DownloadStatusDownloading = "downloading"
	DownloadStatusCompleted   = "completed"
	DownloadStatusCancelled   = "cancelled"
	DownloadStatusError       = "error"
)

// NewDownloadService 创建下载服务
func NewDownloadService(playService *PlayService) *DownloadService {
	return &DownloadService{
		playService: playService,
		sessions:    make(map[string]*DownloadSession),
	}
}

// StartDownload 开始录像下载
func (s *DownloadService) StartDownload(deviceId, channelId string, startTime, endTime time.Time, speed int) (*DownloadSession, error) {
	if s.playService == nil {
		return nil, fmt.Errorf("播放服务未初始化")
	}

	// 调用 PlayService 开始下载
	result, err := s.playService.Download(deviceId, channelId, startTime, endTime, speed)
	if err != nil {
		return nil, fmt.Errorf("开始下载失败: %w", err)
	}

	session := &DownloadSession{
		StreamId:  result.StreamId,
		DeviceId:  deviceId,
		ChannelId: channelId,
		StartTime: startTime,
		EndTime:   endTime,
		Speed:     speed,
		Status:    DownloadStatusDownloading,
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.mu.Lock()
	s.sessions[result.StreamId] = session
	s.mu.Unlock()

	log.Info().
		Str("stream_id", result.StreamId).
		Str("device_id", deviceId).
		Str("channel_id", channelId).
		Int("speed", speed).
		Msg("录像下载已开始")

	return session, nil
}

// GetProgress 查询下载进度
func (s *DownloadService) GetProgress(streamId string) (*DownloadSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return nil, fmt.Errorf("下载会话不存在: %s", streamId)
	}

	// 检查 PlayService 会话状态以更新进度
	playSession, playExists := s.playService.GetSession(streamId)
	if playExists {
		// 如果 PlaySession 不存在，说明下载已结束
		if playSession.Status == PlayStatusStopped {
			if session.Status == DownloadStatusDownloading {
				session.Status = DownloadStatusCompleted
				session.Progress = 100
				session.UpdatedAt = time.Now()
				log.Info().Str("stream_id", streamId).Msg("录像下载已完成")
			}
		}
	} else {
		// PlaySession 已清理，认为下载已完成
		if session.Status == DownloadStatusDownloading {
			session.Status = DownloadStatusCompleted
			session.Progress = 100
			session.UpdatedAt = time.Now()
			log.Info().Str("stream_id", streamId).Msg("录像下载已完成（会话已清理）")
		}
	}

	return session, nil
}

// CancelDownload 取消下载
func (s *DownloadService) CancelDownload(streamId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return fmt.Errorf("下载会话不存在: %s", streamId)
	}

	if session.Status == DownloadStatusCompleted || session.Status == DownloadStatusCancelled {
		return fmt.Errorf("下载已完成或已取消，无法再次取消")
	}

	// 停止 PlayService 会话
	if err := s.playService.Stop(streamId); err != nil {
		log.Warn().Err(err).Str("stream_id", streamId).Msg("停止播放会话失败")
	}

	session.Status = DownloadStatusCancelled
	session.UpdatedAt = time.Now()

	log.Info().Str("stream_id", streamId).Msg("录像下载已取消")

	return nil
}

// ListSessions 列出所有下载会话
func (s *DownloadService) ListSessions() []*DownloadSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*DownloadSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		result = append(result, session)
	}
	return result
}

// UpdateProgress 更新下载进度（内部方法，供 Hook 调用）
func (s *DownloadService) UpdateProgress(streamId string, progress float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return
	}

	session.Progress = progress
	session.UpdatedAt = time.Now()

	log.Debug().
		Str("stream_id", streamId).
		Float64("progress", progress).
		Msg("更新下载进度")
}

// MarkCompleted 标记下载完成（内部方法，供 Hook 调用）
func (s *DownloadService) MarkCompleted(streamId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return
	}

	session.Status = DownloadStatusCompleted
	session.Progress = 100
	session.UpdatedAt = time.Now()

	log.Info().Str("stream_id", streamId).Msg("录像下载已完成")
}

// MarkError 标记下载错误（内部方法，供 Hook 调用）
func (s *DownloadService) MarkError(streamId string, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[streamId]
	if !exists {
		return
	}

	session.Status = DownloadStatusError
	session.Error = errMsg
	session.UpdatedAt = time.Now()

	log.Error().Str("stream_id", streamId).Str("error", errMsg).Msg("录像下载失败")
}

// CleanupStaleSessions 清理过期会话
func (s *DownloadService) CleanupStaleSessions(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for streamId, session := range s.sessions {
		// 已完成/已取消/错误的会话超过指定时间则清理
		if (session.Status == DownloadStatusCompleted ||
			session.Status == DownloadStatusCancelled ||
			session.Status == DownloadStatusError) &&
			now.Sub(session.UpdatedAt) > timeout {
			delete(s.sessions, streamId)
			log.Info().Str("stream_id", streamId).Msg("清理过期下载会话")
		}
	}
}
