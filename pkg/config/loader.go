package config

import (
	"errors"
	"git.skcks.cn/Shikong/go-gb28181/pkg/constants"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
)

func GenerateConfig() error {
	// 生成配置文件
	p, _ := filepath.Abs("./config.toml")

	flag := os.O_RDWR
	_, err := os.Stat(p)
	exist := !os.IsNotExist(err)

	if !exist {
		f, err := os.OpenFile(p, flag|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()
		encoder := toml.NewEncoder(f)
		encoder.SetIndentTables(true)
		_ = encoder.Encode(DefaultClientConfig())
		_ = f.Sync()
	}

	return nil
}

func ReadClientConfig() (*ClientConfig, error) {
	// 读取客户端配置文件，如果不存在则生成示例配置
	viper.SetConfigName(constants.ConfigFileName)
	viper.SetConfigType(constants.ConfigType)
	for _, path := range constants.ConfigPaths {
		viper.AddConfigPath(path)
	}

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			_ = GenerateConfig()
			log.Fatal("未找到配置文件, 已生成示例配置文件于运行路径下")
		}
	}

	conf := new(ClientConfig)
	return conf, viper.Unmarshal(conf)
}
