package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"update-server/internal/config"
)

func TestDomains(t *testing.T) {
	// 设置配置文件路径
	os.Setenv("CONFIG_PATH", "../../config.yaml")
	config.Load()

	req := httptest.NewRequest("GET", "/api/v1/redirect/domains", nil)
	w := httptest.NewRecorder()

	Domains(w, req)

	resp := w.Result()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 200, 实际: %d", resp.StatusCode)
		t.Logf("响应内容: %s", w.Body.String())
		return
	}

	// 检查 Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("期望 Content-Type application/json, 实际: %s", contentType)
	}

	// 检查返回的 JSON 是否有效
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("返回的不是有效 JSON: %v", err)
		return
	}

	// 检查必要字段
	if _, ok := result["panelType"]; !ok {
		t.Error("缺少 panelType 字段")
	}
	if _, ok := result["panels"]; !ok {
		t.Error("缺少 panels 字段")
	}

	t.Logf("成功获取 domains.json，包含 %d 个顶级字段", len(result))
}

func TestDomainsNoConfig(t *testing.T) {
	// 测试未配置 repo 的情况
	cfg := config.Get()
	originalRepo := cfg.Domains.Repo
	cfg.Domains.Repo = ""
	defer func() { cfg.Domains.Repo = originalRepo }()

	req := httptest.NewRequest("GET", "/api/v1/redirect/domains", nil)
	w := httptest.NewRecorder()

	Domains(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("期望状态码 500, 实际: %d", w.Result().StatusCode)
	}
}
