package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/internal/repository"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// CatalogService 目录同步服务
type CatalogService struct {
	client     *sipgo.Client
	deviceRepo *repository.DeviceRepository
	localID    string // 本地设备 ID
	localIP    string // 本地 IP
	localPort  int    // 本地端口
}

// NewCatalogService 创建目录服务
func NewCatalogService(client *sipgo.Client, deviceRepo *repository.DeviceRepository, localID, localIP string, localPort int) *CatalogService {
	return &CatalogService{
		client:     client,
		deviceRepo: deviceRepo,
		localID:    localID,
		localIP:    localIP,
		localPort:  localPort,
	}
}

// SyncCatalog 同步设备目录
func (s *CatalogService) SyncCatalog(deviceID string) error {
	if s.client == nil {
		return fmt.Errorf("SIP 客户端未初始化")
	}

	// 从数据库获取设备信息
	device, err := s.deviceRepo.GetByDeviceID(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}

	// 构建目录查询请求
	sn := fmt.Sprintf("%06d", rand.Intn(1000000))
	catalogReq := manscdp.NewCatalogReq("Catalog", sn, deviceID)

	// 编码为 XML (GBK)
	body, err := utils.XMLMarshal(catalogReq, "gbk")
	if err != nil {
		return fmt.Errorf("XML 编码失败: %w", err)
	}

	// 构建目标 URI - 发送给设备
	target := sip.Uri{
		User: deviceID,
		Host: device.IP,
		Port: device.Port,
	}

	// 创建 MESSAGE 请求
	req := sip.NewRequest(sip.MESSAGE, target)

	// 设置 From/To 头
	from := sip.NewHeader("From", fmt.Sprintf("<sip:%s@%s:%d>", s.localID, s.localIP, s.localPort))
	to := sip.NewHeader("To", fmt.Sprintf("<sip:%s@%s:%d>", deviceID, device.IP, device.Port))
	callID := sip.NewHeader("Call-ID", fmt.Sprintf("%d@%s", time.Now().UnixNano(), s.localIP))
	cseq := sip.NewHeader("CSeq", "1 MESSAGE")
	contentType := sip.NewHeader("Content-Type", "Application/MANSCDP+xml")

	req.AppendHeader(from)
	req.AppendHeader(to)
	req.AppendHeader(callID)
	req.AppendHeader(cseq)
	req.AppendHeader(contentType)
	req.SetBody(body)

	log.Info().Str("device_id", deviceID).Str("ip", device.IP).Int("port", device.Port).Msg("发送目录查询请求")

	// 使用 TransactionRequest 发送请求（复用传输层连接）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := s.client.TransactionRequest(ctx, req)
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("发送目录查询请求失败")
		return fmt.Errorf("发送目录查询请求失败: %w", err)
	}
	defer tx.Terminate()

	log.Info().Str("device_id", deviceID).Msg("目录查询请求已发送")
	return nil
}
