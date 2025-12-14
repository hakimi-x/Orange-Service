package cache

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"update-server/internal/config"
	"update-server/internal/github"
)

// Sync 同步最新版本的所有文件到本地缓存
func Sync() error {
	release, err := github.FetchLatestRelease()
	if err != nil {
		return fmt.Errorf("获取 release 失败: %w", err)
	}

	cfg := config.Get()
	cacheDir := filepath.Join(cfg.Cache.Dir, release.TagName)

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	log.Printf("开始同步版本 %s 的文件 (%d 个)", release.TagName, len(release.Assets))

	for _, asset := range release.Assets {
		cachePath := filepath.Join(cacheDir, asset.Name)

		// 检查文件是否已存在且大小一致
		if info, err := os.Stat(cachePath); err == nil {
			if info.Size() == asset.Size {
				log.Printf("  [跳过] %s (已缓存)", asset.Name)
				continue
			}
		}

		log.Printf("  [下载] %s (%d MB)", asset.Name, asset.Size/1024/1024)

		downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
			cfg.GitHub.Repo, release.TagName, asset.Name)

		if err := downloadFile(downloadURL, cachePath, cfg.GitHub.Token); err != nil {
			log.Printf("  [失败] %s: %v", asset.Name, err)
			continue
		}

		log.Printf("  [完成] %s", asset.Name)

		// 每个文件下载后强制 GC 释放内存
		runtime.GC()
	}

	log.Printf("版本 %s 同步完成", release.TagName)
	return nil
}

func downloadFile(url, dest, token string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	tmpPath := dest + ".tmp"
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

	return os.Rename(tmpPath, dest)
}
