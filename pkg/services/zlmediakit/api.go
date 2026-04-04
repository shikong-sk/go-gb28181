package zlmediakit

import (
	"encoding/json"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit/types"
	"github.com/go-resty/resty/v2"
	"strconv"
	"time"
)

type ZLMediaKit struct {
	client *resty.Client
}

var zLMediaKitService *ZLMediaKit

func SetupZLMediaKitService(config *Config) {
	client := resty.New()
	client.EnableTrace()
	client.SetBaseURL(config.Url)
	client.SetQueryParam("secret", config.Secret)
	client.SetTimeout(10 * time.Second)
	client.SetRetryCount(3)
	zLMediaKitService = &ZLMediaKit{
		client: client,
	}
}

func GetZLMediaKitService() *ZLMediaKit {
	return zLMediaKitService
}

func (z *ZLMediaKit) GetApiList() (data *types.Data[[]string], err error) {
	resp, err := z.client.R().Get("/index/api/getApiList")
	if err != nil {
		return nil, err
	}

	data = new(types.Data[[]string])
	err = json.Unmarshal(resp.Body(), data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (z *ZLMediaKit) GetServerConfig() (data *types.ServerConfigResp, err error) {
	resp, err := z.client.R().Get("/index/api/getServerConfig")
	if err != nil {
		return nil, err
	}

	data = new(types.ServerConfigResp)
	err = json.Unmarshal(resp.Body(), data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (z *ZLMediaKit) SetServerConfig(config *types.ServerConfig) (data *types.SetServerConfigResp, err error) {
	resp, err := z.client.R().
		SetBody(config).
		Post("/index/api/setServerConfig")
	if err != nil {
		return nil, err
	}

	data = new(types.SetServerConfigResp)
	err = json.Unmarshal(resp.Body(), data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// OpenRtpServer 打开 RTP 服务器
// tcpMode: 0=UDP, 1=TCP被动(平台监听TCP端口), 2=TCP主动(平台主动连接设备)
func (z *ZLMediaKit) OpenRtpServer(streamId string, port int, tcpMode int) (*types.OpenRtpServerResp, error) {
	// 重要：使用 port=0 让 ZLM 自动分配端口
	// Java 平台测试证明：自动分配的端口（如55728）可以成功收流
	// 不要强制指定端口范围

	// ZLMediaKit API 使用 GET 方法，不是 POST
	req := z.client.R().
		SetQueryParam("port", strconv.Itoa(port)).
		SetQueryParam("stream_id", streamId).
		SetQueryParam("tcp_mode", strconv.Itoa(tcpMode)).
		SetQueryParam("re_use_port", "1") // 允许端口复用（关键！）

	// 记录请求详情
	log.Info().
		Str("stream_id", streamId).
		Int("port", port).
		Int("tcp_mode", tcpMode).
		Msg("调用 OpenRtpServer API (GET)")

	resp, err := req.Get("/index/api/openRtpServer")
	if err != nil {
		log.Error().Err(err).Str("stream_id", streamId).Msg("OpenRtpServer API 调用失败")
		return nil, err
	}

	// 记录原始响应
	log.Info().
		Str("stream_id", streamId).
		Str("response", string(resp.Body())).
		Int("status_code", resp.StatusCode()).
		Msg("OpenRtpServer API 响应")

	data := new(types.OpenRtpServerResp)
	if err := json.Unmarshal(resp.Body(), data); err != nil {
		log.Error().Err(err).Str("stream_id", streamId).Msg("解析 OpenRtpServer 响应失败")
		return nil, err
	}

	// 记录 ZLM 分配的实际端口
	log.Info().
		Str("stream_id", streamId).
		Int("assigned_port", data.Port).
		Msg("ZLM 自动分配的 RTP 端口")

	return data, nil
}

// CloseRtpServer 关闭 RTP 服务器
func (z *ZLMediaKit) CloseRtpServer(streamId string) (*types.CloseRtpServerResp, error) {
	req := &types.CloseRtpServerReq{
		StreamId: streamId,
	}

	resp, err := z.client.R().
		SetBody(req).
		Post("/index/api/closeRtpServer")
	if err != nil {
		return nil, err
	}

	data := new(types.CloseRtpServerResp)
	if err := json.Unmarshal(resp.Body(), data); err != nil {
		return nil, err
	}
	return data, nil
}

// ConnectRtpServer 主动连接 RTP 服务器（TCP-ACTIVE 模式）
func (z *ZLMediaKit) ConnectRtpServer(streamId, dstUrl string, dstPort int) (*types.ConnectRtpServerResp, error) {
	req := &types.ConnectRtpServerReq{
		StreamId: streamId,
		DstUrl:   dstUrl,
		DstPort:  dstPort,
	}

	resp, err := z.client.R().
		SetBody(req).
		Post("/index/api/connectRtpServer")
	if err != nil {
		return nil, err
	}

	data := new(types.ConnectRtpServerResp)
	if err := json.Unmarshal(resp.Body(), data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetMediaList 获取媒体列表
func (z *ZLMediaKit) GetMediaList(app, stream string) (*types.GetMediaListResp, error) {
	req := &types.GetMediaListReq{
		App:    app,
		Stream: stream,
	}

	resp, err := z.client.R().
		SetBody(req).
		Post("/index/api/getMediaList")
	if err != nil {
		return nil, err
	}

	data := new(types.GetMediaListResp)
	if err := json.Unmarshal(resp.Body(), data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetRtpInfo 获取 RTP 流信息
func (z *ZLMediaKit) GetRtpInfo(streamId string) (*types.RtpInfoResp, error) {
	resp, err := z.client.R().
		SetQueryParam("stream_id", streamId).
		Get("/index/api/getRtpInfo")
	if err != nil {
		return nil, err
	}

	data := new(types.RtpInfoResp)
	if err := json.Unmarshal(resp.Body(), data); err != nil {
		return nil, err
	}
	return data, nil
}
