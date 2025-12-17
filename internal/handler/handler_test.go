package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"update-server/internal/config"
)

// 初始化测试配置
func setupTestConfig(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "update-server-test")
	if err != nil {
		t.Fatal(err)
	}

	configContent := `
server:
  port: 8080
  host: "127.0.0.1"
  base_url: "http://localhost:8080"
github:
  repo: "test/repo"
cache:
  dir: "` + filepath.Join(tmpDir, "cache") + `"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("CONFIG_PATH", configPath)
	config.Load()

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
		os.Unsetenv("CONFIG_PATH")
	}
}

func TestRoot(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	Root(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 得到 %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	if resp["app"] != "update-server" {
		t.Errorf("期望 app=update-server, 得到 %v", resp["app"])
	}
}

func TestRoot_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/invalid", nil)
	w := httptest.NewRecorder()

	Root(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404, 得到 %d", w.Code)
	}
}

func TestCheckUpdate_MissingVersion(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/check-update", nil)
	w := httptest.NewRecorder()

	CheckUpdate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 得到 %d", w.Code)
	}
}

func TestCheckUpdate_NoVersionInfo(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/check-update?version=1.0.0", nil)
	w := httptest.NewRecorder()

	CheckUpdate(w, req)

	// 没有版本信息时应返回 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("期望状态码 503, 得到 %d", w.Code)
	}
}

func TestVersion_NoVersionInfo(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/version", nil)
	w := httptest.NewRecorder()

	Version(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("期望状态码 503, 得到 %d", w.Code)
	}
}

func TestDownload_InvalidPath(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	tests := []struct {
		name string
		path string
		code int
	}{
		{"缺少版本和文件名", "/api/v1/download/", http.StatusBadRequest},
		{"只有版本", "/api/v1/download/v1.0.0", http.StatusBadRequest},
		{"路径穿越-版本", "/api/v1/download/../etc/passwd", http.StatusBadRequest},
		{"路径穿越-文件名", "/api/v1/download/v1.0.0/../../etc/passwd", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			Download(w, req)

			if w.Code != tt.code {
				t.Errorf("期望状态码 %d, 得到 %d", tt.code, w.Code)
			}
		})
	}
}

func TestDownload_VersionNotFound(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/download/v999.0.0/test.zip", nil)
	w := httptest.NewRecorder()

	Download(w, req)

	// 版本不存在时返回 404
	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404, 得到 %d", w.Code)
	}
}

func TestDownload_FromCache(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := config.Get()

	// 创建缓存文件
	cacheDir := filepath.Join(cfg.CacheDir, "v1.0.0")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}

	testContent := []byte("test file content")
	cachePath := filepath.Join(cacheDir, "test.zip")
	if err := os.WriteFile(cachePath, testContent, 0644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/v1/download/v1.0.0/test.zip", nil)
	w := httptest.NewRecorder()

	Download(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 得到 %d", w.Code)
	}

	if w.Body.String() != string(testContent) {
		t.Errorf("文件内容不匹配")
	}
}

// 测试版本比较逻辑
func TestVersionComparison(t *testing.T) {
	tests := []struct {
		client   string
		latest   string
		expected bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.1", "1.0.0", false},
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.0", "v1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.client+"_vs_"+tt.latest, func(t *testing.T) {
			latestVer := trimV(tt.latest)
			clientVer := trimV(tt.client)
			result := latestVer != clientVer && latestVer > clientVer

			if result != tt.expected {
				t.Errorf("版本比较 %s vs %s: 期望 %v, 得到 %v",
					tt.client, tt.latest, tt.expected, result)
			}
		})
	}
}

func trimV(v string) string {
	if len(v) > 0 && v[0] == 'v' {
		return v[1:]
	}
	return v
}

// 测试 JSON 响应格式
func TestJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	jsonResponse(w, data)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type 应为 application/json")
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	if resp["key"] != "value" {
		t.Error("JSON 响应内容不正确")
	}
}

// 测试错误响应格式
func TestHTTPError(t *testing.T) {
	w := httptest.NewRecorder()

	httpError(w, http.StatusBadRequest, "测试错误")

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 得到 %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	if resp["error"] != "测试错误" {
		t.Error("错误消息不正确")
	}
}
