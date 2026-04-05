package zlmediakit

type Config struct {
	Id      string `json:"id" yaml:"id" toml:"id" mapstructure:"id" comment:"ZLMediaKit服务ID"`
	Url     string `json:"url" yaml:"url" toml:"url" mapstructure:"url" comment:"ZLMediaKit服务地址"`
	Secret  string `json:"secret" yaml:"secret" toml:"secret" mapstructure:"secret" comment:"ZLMediaKit服务密钥"`
	HookUrl string `json:"hook_url" yaml:"hook_url" toml:"hook_url" mapstructure:"hook_url" comment:"ZLMediaKit Hook回调地址"`
	RtpPort int    `json:"rtp_port" yaml:"rtp_port" toml:"rtp_port" mapstructure:"rtp_port" comment:"RTP收流端口(0=自动分配)"`
}
