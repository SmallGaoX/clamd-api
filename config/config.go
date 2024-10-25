package config

import (
	"errors"

	"github.com/spf13/viper"
)

// Config 结构体存储应用程序配置
type Config struct {
	ClamAVAddress string
	TempDir       string
	Port          string
	APIKeyFile    string
	LogFile       string
}

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
	// 设置默认值
	viper.SetDefault("clamav_address", "10.10.101.50:3310")
	viper.SetDefault("temp_dir", "/tmp")
	viper.SetDefault("port", "8080")
	viper.SetDefault("api_key_file", "api_keys.txt") // 修改这里，使用相对路径
	viper.SetDefault("log_file", "clamd-api.log")

	// 读取配置文件
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	// 从环境变量读取配置
	viper.AutomaticEnv()

	// 创建配置结构体
	config := &Config{
		ClamAVAddress: viper.GetString("clamav_address"),
		TempDir:       viper.GetString("temp_dir"),
		Port:          viper.GetString("port"),
		APIKeyFile:    viper.GetString("api_key_file"),
		LogFile:       viper.GetString("log_file"),
	}

	return config, nil
}
