package handler

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"update-server/internal/config"
)

// Domains 获取域名列表
// @Summary 获取域名列表
// @Description 从 GitHub 获取 domains.json 并返回
// @Tags redirect
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /api/v1/redirect/domains [get]
func Domains(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()

	if cfg.Domains.Repo == "" {
		httpError(w, http.StatusInternalServerError, "domains repo 未配置")
		return
	}

	// 构造 GitHub API URL
	apiURL := "https://api.github.com/repos/" + cfg.Domains.Repo + "/contents/domains.json"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "创建请求失败")
		return
	}

	if cfg.Domains.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Domains.Token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		httpError(w, http.StatusBadGateway, "无法连接到 GitHub")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		httpError(w, resp.StatusCode, "GitHub 返回错误: "+string(body))
		return
	}

	// GitHub API 返回的是 base64 编码的内容
	var apiResp struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		httpError(w, http.StatusInternalServerError, "解析响应失败")
		return
	}

	// 解码 base64 内容 (GitHub 返回的 base64 包含换行符，需要先去掉)
	cleanContent := strings.ReplaceAll(apiResp.Content, "\n", "")
	content, err := base64.StdEncoding.DecodeString(cleanContent)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "解码内容失败: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}
