package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"update-server/internal/cache"
	"update-server/internal/config"
	"update-server/internal/handler"
	"update-server/internal/version"

	_ "update-server/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title Update Server API
// @version 1.0
// @description GitHub Release 缓存和更新检查服务
// @host localhost:8001
// @BasePath /

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

	// 定时刷新 (每5分钟)
	version.StartAutoRefresh(5 * time.Minute)

	// 路由
	http.HandleFunc("/", handler.Root)
	http.HandleFunc("/api/v1/check-update", handler.CheckUpdate)
	http.HandleFunc("/api/v1/version", handler.Version)
	http.HandleFunc("/api/v1/download/", handler.Download)
	http.HandleFunc("/api/v1/webhook", handler.Webhook)

	// Swagger UI
	http.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("服务器启动: http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
