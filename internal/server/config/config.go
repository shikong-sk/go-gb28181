package config

import (
	"fmt"
	"os"

	"git.skcks.cn/Shikong/go-gb28181/pkg/services/zlmediakit"
	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	// 调试模式
	Debug bool `mapstructure:"debug"`

	// SIP 配置
	SIP SIPConfig `mapstructure:"sip"`

	// HTTP API 配置
	HTTP HTTPConfig `mapstructure:"http"`

	// 数据库配置
	Database DatabaseConfig `mapstructure:"database"`

	// ZLMediaKit 配置
	ZLMediaKit *zlmediakit.Config `mapstructure:"zlmediakit"`
}

// SIPConfig SIP 相关配置
type SIPConfig struct {
	// 服务器配置
	ServerID   string `mapstructure:"server_id"`   // 服务器 ID (20位国标编码)
	ServerIP   string `mapstructure:"server_ip"`   // 服务器 IP
	ServerPort int    `mapstructure:"server_port"` // 服务器端口

	// 本地配置
	DeviceID   string `mapstructure:"device_id"`   // 设备 ID (20位国标编码)
	ListenIP   string `mapstructure:"listen_ip"`   // 监听 IP (0.0.0.0 监听所有)
	ListenPort int    `mapstructure:"listen_port"` // 监听端口
	ExternalIP string `mapstructure:"external_ip"` // 对外 IP (SIP 消息中的 IP)

	// 认证配置
	Password string `mapstructure:"password"` // 密码

	// 功能开关
	Enabled       bool `mapstructure:"enabled"`        // 是否启用 SIP 服务
	RegisterCycle int  `mapstructure:"register_cycle"` // 注册周期(秒)
	KeepaliveInt  int  `mapstructure:"keepalive_int"`  // 心跳间隔(秒)
}

// HTTPConfig HTTP API 配置
type HTTPConfig struct {
	Enabled bool   `mapstructure:"enabled"` // 是否启用 HTTP API
	Host    string `mapstructure:"host"`    // 监听地址
	Port    int    `mapstructure:"port"`    // 监听端口
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     string `mapstructure:"type"`     // 数据库类型: sqlite, mysql, postgres
	Host     string `mapstructure:"host"`     // 主机地址 (mysql/postgres)
	Port     int    `mapstructure:"port"`     // 端口 (mysql/postgres)
	User     string `mapstructure:"user"`     // 用户名 (mysql/postgres)
	Password string `mapstructure:"password"` // 密码 (mysql/postgres)
	Name     string `mapstructure:"name"`     // 数据库名/文件路径
	SSLMode  string `mapstructure:"ssl_mode"` // SSL 模式 (postgres)

	// 连接池配置
	MaxOpenConns int `mapstructure:"max_open_conns"` // 最大打开连接数
	MaxIdleConns int `mapstructure:"max_idle_conns"` // 最大空闲连接数
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Debug: false,
		SIP: SIPConfig{
			ServerID:      "34020000002000000001",
			ServerIP:      "127.0.0.1",
			ServerPort:    5060,
			DeviceID:      "34020000001320000001",
			ListenIP:      "0.0.0.0",
			ListenPort:    5099,
			ExternalIP:    "", // 空则自动检测本机 IP
			Password:      "12345678",
			Enabled:       true,
			RegisterCycle: 3600,
			KeepaliveInt:  30,
		},
		HTTP: HTTPConfig{
			Enabled: true,
			Host:    "0.0.0.0",
			Port:    8080,
		},
		Database: DatabaseConfig{
			Type:         "sqlite",
			Name:         "data/gb28181.db",
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
		ZLMediaKit: &zlmediakit.Config{
			Url: "http://127.0.0.1:80",
		},
	}
}

// ReadConfig 读取配置文件
func ReadConfig() (*Config, error) {
	v := viper.New()

	// 设置默认值
	defaultConfig := DefaultConfig()
	setDefaults(v, defaultConfig)

	// 配置文件搜索路径
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(".")
	v.AddConfigPath("./conf")
	v.AddConfigPath("./config")

	// 环境变量支持
	v.SetEnvPrefix("GB28181")
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件不存在，生成默认配置
			if err := generateDefaultConfig(); err != nil {
				return nil, fmt.Errorf("生成默认配置失败: %w", err)
			}
			// 重新读取
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("读取配置文件失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	// 解析配置
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper, c *Config) {
	v.SetDefault("debug", c.Debug)

	v.SetDefault("sip.server_id", c.SIP.ServerID)
	v.SetDefault("sip.server_ip", c.SIP.ServerIP)
	v.SetDefault("sip.server_port", c.SIP.ServerPort)
	v.SetDefault("sip.device_id", c.SIP.DeviceID)
	v.SetDefault("sip.listen_ip", c.SIP.ListenIP)
	v.SetDefault("sip.listen_port", c.SIP.ListenPort)
	v.SetDefault("sip.password", c.SIP.Password)
	v.SetDefault("sip.enabled", c.SIP.Enabled)
	v.SetDefault("sip.register_cycle", c.SIP.RegisterCycle)
	v.SetDefault("sip.keepalive_int", c.SIP.KeepaliveInt)

	v.SetDefault("http.enabled", c.HTTP.Enabled)
	v.SetDefault("http.host", c.HTTP.Host)
	v.SetDefault("http.port", c.HTTP.Port)

	v.SetDefault("database.type", c.Database.Type)
	v.SetDefault("database.name", c.Database.Name)
	v.SetDefault("database.max_open_conns", c.Database.MaxOpenConns)
	v.SetDefault("database.max_idle_conns", c.Database.MaxIdleConns)

	v.SetDefault("zlmediakit.url", c.ZLMediaKit.Url)
}

// generateDefaultConfig 生成默认配置文件
func generateDefaultConfig() error {
	configContent := `# GB28181 服务配置文件

# 调试模式
debug = false

[sip]
# SIP 服务器配置
server_id = "34020000002000000001"   # 服务器 ID (20位国标编码)
server_ip = "127.0.0.1"               # 服务器 IP
server_port = 5060                    # 服务器端口

# 本地配置
device_id = "34020000001320000001"    # 设备 ID (20位国标编码)
listen_ip = "0.0.0.0"                 # 监听 IP
listen_port = 5099                    # 监听端口

# 认证配置
password = "12345678"                 # 密码

# 注册周期(秒)
register_cycle = 3600

[http]
# HTTP API 配置
enabled = true                        # 是否启用 HTTP API
host = "0.0.0.0"                      # 监听地址
port = 8080                           # 监听端口

[database]
# 数据库配置
type = "sqlite"                       # 数据库类型: sqlite, mysql, postgres
name = "data/gb28181.db"              # 数据库文件路径 (sqlite) 或数据库名 (mysql/postgres)
max_open_conns = 10                   # 最大打开连接数
max_idle_conns = 5                    # 最大空闲连接数

# MySQL 示例配置:
# type = "mysql"
# host = "127.0.0.1"
# port = 3306
# user = "root"
# password = "password"
# name = "gb28181"

[zlmediakit]
# ZLMediaKit 流媒体服务器配置
url = "http://127.0.0.1:80"
secret = ""
id = ""
`

	// 确保目录存在
	if err := os.MkdirAll(".", 0755); err != nil {
		return err
	}

	return os.WriteFile("config.toml", []byte(configContent), 0644)
}
