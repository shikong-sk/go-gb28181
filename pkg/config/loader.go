package config

import (
	"errors"
	"git.skcks.cn/Shikong/go-gb28181/pkg/constants"
	"git.skcks.cn/Shikong/go-gb28181/pkg/logger"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

func GenerateConfig() error {
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
	viper.SetConfigName(constants.ConfigFileName)
	viper.SetConfigType(constants.ConfigType)
	for _, path := range constants.ConfigPaths {
		viper.AddConfigPath(path)
	}

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			_ = GenerateConfig()
			logger.Log().Fatalf("未找到配置文件, 已生成示例配置文件于运行路径下")
		}
	}

	conf := new(ClientConfig)
	return conf, viper.Unmarshal(conf)
}
