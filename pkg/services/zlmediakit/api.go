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
