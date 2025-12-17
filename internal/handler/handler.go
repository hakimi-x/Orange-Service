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
		"version": version.AppVersion,
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

// BuildInfo 构建文件信息
type BuildInfo struct {
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	FileURL      string `json:"file_url"`
	DownloadLink string `json:"download_link"`
	FileSize     int64  `json:"file_size"`
	UploadTime   string `json:"upload_time"`
	FileType     string `json:"file_type"`
	Architecture string `json:"architecture"`
}

// ResourcesResponse 资源列表响应
type ResourcesResponse struct {
	Status  string                `json:"status"`
	Version string                `json:"version"`
	Builds  map[string][]BuildInfo `json:"builds"`
}

// Resources 获取构建资源列表
// @Summary 获取构建资源列表
// @Description 返回按平台分类的构建文件列表
// @Tags resources
// @Produce json
// @Success 200 {object} ResourcesResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/v1/resources [get]
func Resources(w http.ResponseWriter, r *http.Request) {
	info := version.Get()
	if info == nil {
		httpError(w, http.StatusServiceUnavailable, "版本信息暂不可用")
		return
	}

	cfg := config.Get()
	builds := make(map[string][]BuildInfo)

	for _, asset := range info.Assets {
		platform, fileType, arch := parseAssetName(asset.Name)
		if platform == "" {
			continue
		}

		build := BuildInfo{
			FileName:     asset.Name,
			FilePath:     asset.Name,
			FileURL:      asset.DownloadURL,
			DownloadLink: fmt.Sprintf("%s/api/v1/download/%s/%s", cfg.Server.BaseURL, info.Version, asset.Name),
			FileSize:     asset.Size,
			UploadTime:   info.PublishedAt,
			FileType:     fileType,
			Architecture: arch,
		}

		builds[platform] = append(builds[platform], build)
	}

	jsonResponse(w, ResourcesResponse{
		Status:  "success",
		Version: info.Version,
		Builds:  builds,
	})
}

// parseAssetName 解析文件名，返回平台、文件类型、架构
func parseAssetName(name string) (platform, fileType, arch string) {
	nameLower := strings.ToLower(name)

	// 跳过 sha256 校验文件
	if strings.HasSuffix(nameLower, ".sha256") {
		return "", "", ""
	}

	// 判断平台
	switch {
	case strings.Contains(nameLower, "android"):
		platform = "android"
	case strings.Contains(nameLower, "windows"):
		platform = "windows"
	case strings.Contains(nameLower, "macos") || strings.Contains(nameLower, "darwin"):
		platform = "macos"
	case strings.Contains(nameLower, "linux"):
		platform = "linux"
	case strings.Contains(nameLower, "ios"):
		platform = "ios"
	default:
		return "", "", ""
	}

	// 判断文件类型
	switch {
	case strings.HasSuffix(nameLower, ".apk"):
		fileType = "apk"
	case strings.HasSuffix(nameLower, ".exe"):
		fileType = "exe"
	case strings.HasSuffix(nameLower, ".dmg"):
		fileType = "dmg"
	case strings.HasSuffix(nameLower, ".zip"):
		fileType = "zip"
	case strings.HasSuffix(nameLower, ".tar.gz"):
		fileType = "tar.gz"
	case strings.HasSuffix(nameLower, ".deb"):
		fileType = "deb"
	case strings.HasSuffix(nameLower, ".rpm"):
		fileType = "rpm"
	case strings.HasSuffix(nameLower, ".ipa"):
		fileType = "ipa"
	default:
		fileType = "unknown"
	}

	// 判断架构
	switch {
	case strings.Contains(nameLower, "arm64") || strings.Contains(nameLower, "aarch64"):
		arch = "arm64"
	case strings.Contains(nameLower, "amd64") || strings.Contains(nameLower, "x86_64") || strings.Contains(nameLower, "x64"):
		arch = "amd64"
	case strings.Contains(nameLower, "x86") || strings.Contains(nameLower, "i386") || strings.Contains(nameLower, "i686"):
		arch = "x86"
	case strings.Contains(nameLower, "universal"):
		arch = "universal"
	default:
		arch = "unknown"
	}

	return platform, fileType, arch
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
	parts := strings.Split(path, "/")

	// 支持多种格式:
	// - /api/v1/download/{version}/{filename}
	// - /api/v1/download/{brand}/{version}/{invite_code}/{filename}
	var ver, filename string
	switch len(parts) {
	case 2:
		// 格式: {version}/{filename}
		ver, filename = parts[0], parts[1]
	case 4:
		// 格式: {brand}/{version}/{invite_code}/{filename}
		ver, filename = parts[1], parts[3]
	default:
		httpError(w, http.StatusBadRequest, "无效的下载路径")
		return
	}

	if strings.Contains(filename, "..") || strings.Contains(ver, "..") {
		httpError(w, http.StatusBadRequest, "非法路径")
		return
	}

	cachePath := filepath.Join(cfg.CacheDir, ver, filename)
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
		cfg.Release.Repo, ver, filename)

	if err := downloadAndCache(downloadURL, cachePath, cfg.Release.Token); err != nil {
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

	// 使用固定 32KB buffer 避免内存膨胀
	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(f, resp.Body, buf)
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
