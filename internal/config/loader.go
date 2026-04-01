package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load 加载基础配置
func Load() (*Config, error) {
	configPath := "configs/config.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("读取配置文件失败: %v", err)
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Printf("解析配置文件失败: %v", err)
		return nil, err
	}
	// 从数据库加载
	config.Services = []Service{}
	config.Routes = []Route{}
	return &config, nil
}

// MergeConfig 合并配置
func MergeConfig(base *Config, services []Service, routes []Route) *Config {
	base.Services = services
	base.Routes = routes
	return base
}
