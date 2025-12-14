package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"update-server/internal/config"
)

func setupWebhookTestConfig(t *testing.T, secret string) func() {
	tmpDir, err := os.MkdirTemp("", "webhook-test")
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
  webhook_secret: "` + secret + `"
cache:
  dir: "` + filepath.Join(tmpDir, "cache") + `"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("CONFIG_PATH", configPath)
	config.Load()

	return func() {
		os.RemoveAll(tmpDir)
		os.Unsetenv("CONFIG_PATH")
	}
}

func TestWebhook_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/webhook", nil)
	w := httptest.NewRecorder()

	Webhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态码 405, 得到 %d", w.Code)
	}
}

func TestWebhook_InvalidSignature(t *testing.T) {
	cleanup := setupWebhookTestConfig(t, "test-secret")
	defer cleanup()

	payload := []byte(`{"action":"published"}`)
	req := httptest.NewRequest("POST", "/api/v1/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")
	req.Header.Set("X-GitHub-Event", "release")
	w := httptest.NewRecorder()

	Webhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401, 得到 %d", w.Code)
	}
}

func TestWebhook_ValidSignature(t *testing.T) {
	secret := "test-secret"
	cleanup := setupWebhookTestConfig(t, secret)
	defer cleanup()

	payload := []byte(`{"action":"published","release":{"tag_name":"v1.0.0"}}`)

	// 计算正确的签名
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/api/v1/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", signature)
	req.Header.Set("X-GitHub-Event", "release")
	w := httptest.NewRecorder()

	Webhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 得到 %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	if resp["status"] != "ok" {
		t.Errorf("期望 status=ok, 得到 %s", resp["status"])
	}
}

func TestWebhook_IgnoreNonReleaseEvent(t *testing.T) {
	cleanup := setupWebhookTestConfig(t, "")
	defer cleanup()

	payload := []byte(`{}`)
	req := httptest.NewRequest("POST", "/api/v1/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "push")
	w := httptest.NewRecorder()

	Webhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 得到 %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ignored" {
		t.Errorf("期望 status=ignored, 得到 %s", resp["status"])
	}
}

func TestWebhook_IgnoreNonPublishedAction(t *testing.T) {
	cleanup := setupWebhookTestConfig(t, "")
	defer cleanup()

	payload := []byte(`{"action":"created"}`)
	req := httptest.NewRequest("POST", "/api/v1/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "release")
	w := httptest.NewRecorder()

	Webhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 得到 %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ignored" {
		t.Errorf("期望 status=ignored, 得到 %s", resp["status"])
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "my-secret"
	payload := []byte("test payload")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		signature string
		expected  bool
	}{
		{"有效签名", validSig, true},
		{"无效签名", "sha256=invalid", false},
		{"缺少前缀", hex.EncodeToString(mac.Sum(nil)), false},
		{"空签名", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifySignature(payload, tt.signature, secret)
			if result != tt.expected {
				t.Errorf("期望 %v, 得到 %v", tt.expected, result)
			}
		})
	}
}
