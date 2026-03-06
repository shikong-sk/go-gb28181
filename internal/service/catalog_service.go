package service

import (
	"fmt"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp/cmdtype"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// CatalogService 目录同步服务
type CatalogService struct {
	client *sipgo.Client
	config CatalogConfig
}

// CatalogConfig 目录同步配置
type CatalogConfig struct {
	ServerID   string
	ServerIP   string
	ServerPort int
}

// NewCatalogService 创建目录同步服务
func NewCatalogService(client *sipgo.Client, config CatalogConfig) *CatalogService {
	return &CatalogService{
		client: client,
		config: config,
	}
}

// SyncCatalog 同步设备目录
func (s *CatalogService) SyncCatalog(deviceID string) error {
	// 构建目录查询请求
	sn := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	req := manscdp.NewCatalogReq(cmdtype.Catalog, sn, deviceID)

	// 编码为 XML (GBK)
	body, err := utils.XMLMarshal(req, "gbk")
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("编码目录查询请求失败")
		return fmt.Errorf("编码目录查询请求失败: %w", err)
	}

	// 构建 SIP MESSAGE 请求
	target := sip.Uri{
		User: deviceID,
		Host: s.config.ServerIP,
		Port: s.config.ServerPort,
	}

	// 创建 MESSAGE 请求
	messageReq := sip.NewRequest(sip.MESSAGE, target)
	messageReq.SetTransport("UDP")
	messageReq.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))
	messageReq.SetBody(body)

	if err := sipgo.ClientRequestBuild(s.client, messageReq); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("构建 MESSAGE 请求失败")
		return fmt.Errorf("构建 MESSAGE 请求失败: %w", err)
	}

	// 发送请求
	if err := s.client.WriteRequest(messageReq); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("发送目录查询请求失败")
		return fmt.Errorf("发送目录查询请求失败: %w", err)
	}

	log.Info().Str("device_id", deviceID).Msg("目录查询请求发送成功")
	return nil
}
