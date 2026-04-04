package zlmediakit

type Config struct {
	Id      string `json:"id" yaml:"id" toml:"id" comment:"ZLMediaKit服务ID"`
	Url     string `json:"url" yaml:"url" toml:"url" comment:"ZLMediaKit服务地址"` //...
	Secret  string `json:"secret" yaml:"secret" toml:"secret" comment:"ZLMediaKit服务密钥"`
	HookUrl string `json:"hook_url" yaml:"hook_url" toml:"hook_url" comment:"ZLMediaKit Hook回调地址"`
}
