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
)

// 中间件：日志 + CORS
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	cfg := config.Load()

	// 初始化缓存目录
	if err := os.MkdirAll(cfg.CacheDir, 0755); err != nil {
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
	http.HandleFunc("/api/v1/resources", handler.Resources)
	http.HandleFunc("/api/v1/resources/", handler.Resources) // 兼容 /api/v1/resources/{brand}/{inviteCode}
	http.HandleFunc("/api/v1/download/", handler.Download)
	http.HandleFunc("/api/v1/webhook", handler.Webhook)
	http.HandleFunc("/api/v1/redirect/domains", handler.Domains)
	http.HandleFunc("/api/v1/redirect/", handler.RedirectBrand)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// Swagger UI (仅开发模式)
	registerSwagger(addr)

	log.Printf("服务器启动: http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, middleware(http.DefaultServeMux)))
}
