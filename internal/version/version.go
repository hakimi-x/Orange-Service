package version

import (
	"fmt"
	"log"
	"sync"
	"time"

	"update-server/internal/config"
	"update-server/internal/github"
)

// 构建时注入的版本号
var (
	AppVersion   = "dev"
	BuildTime    = "unknown"
	GitCommit    = "unknown"
)

type Asset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
}

type Info struct {
	Version      string    `json:"version"`
	ReleaseNotes string    `json:"release_notes"`
	PublishedAt  string    `json:"published_at"`
	Assets       []Asset   `json:"assets"`
	UpdatedAt    time.Time `json:"-"`
}

var (
	current *Info
	mu      sync.RWMutex
)

func Refresh() error {
	release, err := github.FetchLatestRelease()
	if err != nil {
		return err
	}

	cfg := config.Get()

	mu.Lock()
	defer mu.Unlock()

	assets := make([]Asset, 0, len(release.Assets))
	for _, a := range release.Assets {
		assets = append(assets, Asset{
			Name:        a.Name,
			Size:        a.Size,
			DownloadURL: fmt.Sprintf("%s/api/v1/download/%s/%s", cfg.Server.BaseURL, release.TagName, a.Name),
		})
	}

	current = &Info{
		Version:      release.TagName,
		ReleaseNotes: release.Body,
		PublishedAt:  release.PublishedAt,
		Assets:       assets,
		UpdatedAt:    time.Now(),
	}

	log.Printf("版本信息已更新: %s", release.TagName)
	return nil
}

func Get() *Info {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func StartAutoRefresh(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		for range ticker.C {
			if err := Refresh(); err != nil {
				log.Printf("刷新版本信息失败: %v", err)
			}
		}
	}()
}
