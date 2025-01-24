package zlmediakit

import (
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
