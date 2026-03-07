package config

import "git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit"

type ClientConfig struct {
	Debug bool `json:"debug" toml:"debug" yaml:"debug" comment:"调试模式"`

	ServerIp   string `json:"serverIp" toml:"serverIp" yaml:"serverIp" comment:"服务端Ip"`
	ServerPort int    `json:"serverPort" toml:"serverPort" yaml:"serverPort" comment:"服务端端口号"`
	ServerId   string `json:"serverId" toml:"serverId" yaml:"serverId" comment:"服务端国标 Id"`
	Password   string `json:"password" toml:"password" yaml:"password" comment:"服务端认证密码"`

	DeviceId     string `json:"deviceId" toml:"deviceId" yaml:"deviceId" comment:"设备Id"`
	Name         string `json:"name" toml:"name" yaml:"name" comment:"设备名称"`
	Manufacturer string `json:"manufacturer" toml:"manufacturer" yaml:"manufacturer" comment:"设备厂商"`
	Model        string `json:"model" toml:"model" yaml:"model" comment:"设备型号"`
	IPAddress    string `json:"ipAddress" toml:"ipAddress" yaml:"ipAddress" comment:"设备IP地址"`
	ListenIp     string `json:"listenIp" toml:"listenIp" yaml:"listenIp" comment:"监听Ip"`
	ListenPort   int    `json:"listenPort" toml:"listenPort" yaml:"listenPort" comment:"监听端口号"`

	ZLMediaKit *zlmediakit.Config `json:"zlmediakit" toml:"zlmediakit" yaml:"zlmediakit" comment:"ZLMediaKit配置"`
}

func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Debug: false,

		ServerIp:   "10.10.10.20",
		ServerPort: 5060,
		ServerId:   "44050100002000000003",
		Password:   "123456",

		DeviceId:     "44050100002000000002",
		Name:         "GB28181设备",
		Manufacturer: "Shikong",
		Model:        "SK-GB28181-001",
		IPAddress:    "10.10.10.100",
		ListenIp:     "0.0.0.0",
		ListenPort:   5099,

		ZLMediaKit: &zlmediakit.Config{
			Id:     "zlmediakit",
			Url:    "http://10.10.10.200:5081",
			Secret: "zlmediakit",
		},
	}
}
