package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// GitHubRepo GitHub 仓库配置
type GitHubRepo struct {
	Repo          string `yaml:"repo"`           // owner/repo 格式
	Token         string `yaml:"token"`          // 访问令牌 (私有仓库需要)
	WebhookSecret string `yaml:"webhook_secret"` // Webhook 签名密钥
}

type Config struct {
	Server struct {
		Port    int    `yaml:"port"`
		Host    string `yaml:"host"`
		BaseURL string `yaml:"base_url"`
	} `yaml:"server"`

	// 构建/发布仓库 (公开仓库，用于 check-update/download)
	Release GitHubRepo `yaml:"release"`

	// 域名配置仓库 (私有仓库，用于 redirect/domains)
	Domains GitHubRepo `yaml:"domains"`

	// 缓存目录 (内部使用，默认 "github_cache")
	CacheDir string `yaml:"-"`
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
	cfg.CacheDir = "github_cache"

	return &cfg
}

func Get() *Config {
	return &cfg
}
