package zlmediakit

import (
	"encoding/json"
	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit/types"
	"github.com/go-resty/resty/v2"
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
func (z *ZLMediaKit) OpenRtpServer(streamId string, port int) (*types.OpenRtpServerResp, error) {
	req := &types.OpenRtpServerReq{
		Port:     port,
		StreamId: streamId,
		TcpMode:  1,
	}

	resp, err := z.client.R().
		SetBody(req).
		Post("/index/api/openRtpServer")
	if err != nil {
		return nil, err
	}

	data := new(types.OpenRtpServerResp)
	if err := json.Unmarshal(resp.Body(), data); err != nil {
		return nil, err
	}
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
