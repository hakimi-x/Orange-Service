package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"update-server/internal/cache"
	"update-server/internal/config"
	"update-server/internal/version"
)

type webhookPayload struct {
	Action  string `json:"action"`
	Release struct {
		TagName string `json:"tag_name"`
	} `json:"release"`
}

// WebhookResponse webhook 响应
type WebhookResponse struct {
	Status  string `json:"status" example:"ok"`
	Version string `json:"version,omitempty" example:"v1.0.0"`
	Reason  string `json:"reason,omitempty" example:"not a release event"`
}

// Webhook 处理 GitHub webhook 回调
// @Summary GitHub Webhook 回调
// @Description 接收 GitHub release 事件，自动更新版本信息和缓存
// @Tags webhook
// @Accept json
// @Produce json
// @Param X-GitHub-Event header string true "GitHub 事件类型" example("release")
// @Param X-Hub-Signature-256 header string false "GitHub 签名"
// @Success 200 {object} WebhookResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 405 {object} ErrorResponse
// @Router /api/v1/webhook [post]
func Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "只支持 POST")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "读取请求体失败")
		return
	}

	// 验证签名 (如果配置了 secret)
	cfg := config.Get()
	if cfg.GitHub.WebhookSecret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if !verifySignature(body, signature, cfg.GitHub.WebhookSecret) {
			httpError(w, http.StatusUnauthorized, "签名验证失败")
			return
		}
	}

	// 检查事件类型
	event := r.Header.Get("X-GitHub-Event")
	if event != "release" {
		jsonResponse(w, map[string]string{"status": "ignored", "reason": "not a release event"})
		return
	}

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		httpError(w, http.StatusBadRequest, "解析 payload 失败")
		return
	}

	// 只处理 published 事件
	if payload.Action != "published" {
		jsonResponse(w, map[string]string{"status": "ignored", "reason": "action is " + payload.Action})
		return
	}

	log.Printf("收到 release webhook: %s", payload.Release.TagName)

	// 异步更新版本信息和缓存
	go func() {
		if err := version.Refresh(); err != nil {
			log.Printf("刷新版本信息失败: %v", err)
		}
		if err := cache.Sync(); err != nil {
			log.Printf("同步缓存失败: %v", err)
		}
	}()

	jsonResponse(w, map[string]string{
		"status":  "ok",
		"version": payload.Release.TagName,
	})
}

func verifySignature(payload []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}
