package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"update-server/internal/cache"
	"update-server/internal/config"
	"update-server/internal/handler"
	"update-server/internal/version"
)

func main() {
	cfg := config.Load()

	// 初始化缓存目录
	if err := os.MkdirAll(cfg.Cache.Dir, 0755); err != nil {
		log.Fatalf("创建缓存目录失败: %v", err)
	}

	// 启动时获取版本信息
	if err := version.Refresh(); err != nil {
		log.Printf("警告: 初始化版本信息失败: %v", err)
	}

	// 启动时同步缓存
	go func() {
		if err := cache.Sync(); err != nil {
			log.Printf("警告: 同步缓存失败: %v", err)
		}
	}()

	// 路由
	http.HandleFunc("/", handler.Root)
	http.HandleFunc("/api/v1/check-update", handler.CheckUpdate)
	http.HandleFunc("/api/v1/version", handler.Version)
	http.HandleFunc("/api/v1/download/", handler.Download)
	http.HandleFunc("/api/v1/webhook", handler.Webhook)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// Swagger UI (仅开发模式)
	registerSwagger(addr)

	log.Printf("服务器启动: http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
