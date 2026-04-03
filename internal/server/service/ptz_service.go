package service

import (
	"context"
	"fmt"
	"time"

	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/manscdp"
	"git.skcks.cn/Shikong/go-gb28181/pkg/utils"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// PTZService 云台控制服务
type PTZService struct {
	client        *sipgo.Client
	deviceService *DeviceService
}

// NewPTZService 创建云台控制服务
func NewPTZService(client *sipgo.Client, deviceService *DeviceService) *PTZService {
	return &PTZService{
		client:        client,
		deviceService: deviceService,
	}
}

// PTZControl 云台控制
// deviceId: 设备 ID
// channelId: 通道 ID
// direction: 云台方向
// horizontalSpeed: 水平速度 (0-15)
// verticalSpeed: 垂直速度 (0-255)
func (s *PTZService) PTZControl(deviceId, channelId string, direction manscdp.PTZDirection, horizontalSpeed, verticalSpeed int) error {
	// 获取设备真实地址
	deviceIP := "127.0.0.1"
	devicePort := 5060

	if s.deviceService != nil {
		device, err := s.deviceService.GetDevice(deviceId)
		if err == nil && device.IP != "" {
			deviceIP = device.IP
			devicePort = device.Port
		} else {
			return fmt.Errorf("设备不存在或无法获取地址: %w", err)
		}
	}

	// 构建云台控制命令
	ptzCmd := manscdp.BuildPTZCmd(direction, horizontalSpeed, verticalSpeed)

	// 生成安全序列号
	sn := utils.GenerateSN()

	// 构建控制请求
	controlReq := manscdp.NewPTZControlReq(sn, channelId, ptzCmd)

	// 序列化 XML (使用 GBK 编码)
	body, err := utils.XMLMarshal(controlReq, "gbk")
	if err != nil {
		return fmt.Errorf("序列化云台控制请求失败: %w", err)
	}

	// 构建目标 URI - 使用通道 ID
	target := sip.Uri{
		User: channelId,
		Host: deviceIP,
		Port: devicePort,
	}

	// 创建 MESSAGE 请求
	req := sip.NewRequest(sip.MESSAGE, target)
	req.SetTransport("UDP")
	req.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))
	req.SetBody(body)

	// 发送请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.client.TransactionRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("发送云台控制请求失败: %w", err)
	}
	defer tx.Terminate()

	// 等待响应
	select {
	case <-tx.Done():
		return fmt.Errorf("云台控制请求超时")
	case resp := <-tx.Responses():
		if resp.StatusCode == 200 {
			log.Info().Str("device_id", deviceId).Str("channel_id", channelId).Str("direction", string(direction)).Msg("云台控制成功")
			return nil
		}
		return fmt.Errorf("云台控制失败: %d", resp.StatusCode)
	}
}

// PTZStop 停止云台
func (s *PTZService) PTZStop(deviceId, channelId string) error {
	return s.PTZControl(deviceId, channelId, manscdp.PTZStop, 0, 0)
}

// PTZUp 向上
func (s *PTZService) PTZUp(deviceId, channelId string, speed int) error {
	return s.PTZControl(deviceId, channelId, manscdp.PTZUp, 0, speed)
}

// PTZDown 向下
func (s *PTZService) PTZDown(deviceId, channelId string, speed int) error {
	return s.PTZControl(deviceId, channelId, manscdp.PTZDown, 0, speed)
}

// PTZLeft 向左
func (s *PTZService) PTZLeft(deviceId, channelId string, speed int) error {
	return s.PTZControl(deviceId, channelId, manscdp.PTZLeft, speed, 0)
}

// PTZRight 向右
func (s *PTZService) PTZRight(deviceId, channelId string, speed int) error {
	return s.PTZControl(deviceId, channelId, manscdp.PTZRight, speed, 0)
}

// PTZZoomIn 放大
func (s *PTZService) PTZZoomIn(deviceId, channelId string, speed int) error {
	return s.PTZControl(deviceId, channelId, manscdp.PTZZoomIn, speed, 0)
}

// PTZZoomOut 缩小
func (s *PTZService) PTZZoomOut(deviceId, channelId string, speed int) error {
	return s.PTZControl(deviceId, channelId, manscdp.PTZZoomOut, speed, 0)
}
