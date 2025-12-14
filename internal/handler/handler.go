package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"update-server/internal/config"
	"update-server/internal/version"
)

// UpdateCheckResponse 更新检查响应
type UpdateCheckResponse struct {
	UpdateAvailable bool   `json:"update_available" example:"true"`
	LatestVersion   string `json:"latest_version" example:"v1.2.0"`
	ReleaseNotes    string `json:"release_notes,omitempty" example:"Bug fixes and improvements"`
	DownloadURL     string `json:"download_url" example:"https://example.com"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error" example:"错误信息"`
}

// RootResponse 根路径响应
type RootResponse struct {
	App     string `json:"app" example:"orange-service"`
	Version string `json:"version" example:"1.0.0"`
}

// Root 服务信息
// @Summary 获取服务信息
// @Description 返回服务名称和版本
// @Tags system
// @Produce json
// @Success 200 {object} RootResponse
// @Router / [get]
func Root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	jsonResponse(w, map[string]any{
		"app":     "orange-service",
		"version": "1.0.0",
	})
}

// CheckUpdate 检查更新
// @Summary 检查客户端是否有新版本
// @Description 根据客户端版本号判断是否需要更新
// @Tags update
// @Produce json
// @Param version query string true "客户端当前版本号" example("v1.0.0")
// @Success 200 {object} UpdateCheckResponse
// @Failure 400 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/v1/check-update [get]
func CheckUpdate(w http.ResponseWriter, r *http.Request) {
	clientVersion := r.URL.Query().Get("version")
	if clientVersion == "" {
		httpError(w, http.StatusBadRequest, "缺少 version 参数")
		return
	}

	info := version.Get()
	if info == nil {
		httpError(w, http.StatusServiceUnavailable, "版本信息暂不可用")
		return
	}

	cfg := config.Get()
	latestVer := strings.TrimPrefix(info.Version, "v")
	clientVer := strings.TrimPrefix(clientVersion, "v")
	updateAvailable := latestVer != clientVer && latestVer > clientVer

	jsonResponse(w, UpdateCheckResponse{
		UpdateAvailable: updateAvailable,
		LatestVersion:   info.Version,
		ReleaseNotes:    info.ReleaseNotes,
		DownloadURL:     cfg.Server.BaseURL,
	})
}

// Version 获取最新版本信息
// @Summary 获取最新版本详情
// @Description 返回最新版本的完整信息，包括版本号、发布说明、资源列表等
// @Tags update
// @Produce json
// @Success 200 {object} version.Info
// @Failure 503 {object} ErrorResponse
// @Router /api/v1/version [get]
func Version(w http.ResponseWriter, r *http.Request) {
	info := version.Get()
	if info == nil {
		httpError(w, http.StatusServiceUnavailable, "版本信息暂不可用")
		return
	}
	jsonResponse(w, info)
}

// Download 下载文件
// @Summary 下载指定版本的文件
// @Description 从缓存或 GitHub 下载指定版本的文件
// @Tags download
// @Produce octet-stream
// @Param version path string true "版本号" example("v1.0.0")
// @Param filename path string true "文件名" example("app-linux-amd64.tar.gz")
// @Success 200 {file} binary
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/download/{version}/{filename} [get]
func Download(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/download/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		httpError(w, http.StatusBadRequest, "无效的下载路径")
		return
	}

	ver, filename := parts[0], parts[1]

	if strings.Contains(filename, "..") || strings.Contains(ver, "..") {
		httpError(w, http.StatusBadRequest, "非法路径")
		return
	}

	cachePath := filepath.Join(cfg.Cache.Dir, ver, filename)
	if _, err := os.Stat(cachePath); err == nil {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		http.ServeFile(w, r, cachePath)
		return
	}

	info := version.Get()
	if info == nil || info.Version != ver {
		httpError(w, http.StatusNotFound, "版本不存在")
		return
	}

	var found bool
	for _, asset := range info.Assets {
		if asset.Name == filename {
			found = true
			break
		}
	}

	if !found {
		httpError(w, http.StatusNotFound, "文件不存在")
		return
	}

	downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
		cfg.GitHub.Repo, ver, filename)

	if err := downloadAndCache(downloadURL, cachePath, cfg.GitHub.Token); err != nil {
		httpError(w, http.StatusInternalServerError, "下载文件失败")
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	http.ServeFile(w, r, cachePath)
}

func downloadAndCache(url, cachePath, token string) error {
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: %d", resp.StatusCode)
	}

	tmpPath := cachePath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, cachePath)
}

func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func httpError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
