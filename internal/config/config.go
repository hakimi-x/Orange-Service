package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port    int    `yaml:"port"`
		Host    string `yaml:"host"`
		BaseURL string `yaml:"base_url"`
	} `yaml:"server"`
	GitHub struct {
		Token         string `yaml:"token"`
		Repo          string `yaml:"repo"`
		WebhookSecret string `yaml:"webhook_secret"`
	} `yaml:"github"`
	Cache struct {
		Dir string `yaml:"dir"`
	} `yaml:"cache"`
}

var cfg Config

func Load() *Config {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	// 默认值
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Cache.Dir == "" {
		cfg.Cache.Dir = "cache"
	}

	return &cfg
}

func Get() *Config {
	return &cfg
}
