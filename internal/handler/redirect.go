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

// fetchDomainsJSON 获取并解析 domains.json
func fetchDomainsJSON() (map[string]interface{}, error) {
	cfg := config.Get()

	apiURL := "https://api.github.com/repos/" + cfg.Domains.Repo + "/contents/domains.json"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	if cfg.Domains.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Domains.Token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var apiResp struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	cleanContent := strings.ReplaceAll(apiResp.Content, "\n", "")
	content, err := base64.StdEncoding.DecodeString(cleanContent)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	return data, nil
}

// RedirectBrand 根据品牌重定向到第一个面板 URL
// @Summary 品牌重定向
// @Description 根据品牌名称重定向到该品牌的第一个面板 URL
// @Tags redirect
// @Param brand path string true "品牌名称" example("v2x")
// @Success 302 "重定向到面板 URL"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/redirect/{brand} [get]
func RedirectBrand(w http.ResponseWriter, r *http.Request) {
	// 从 URL 路径提取品牌名
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/redirect/")
	brand := strings.Split(path, "/")[0]

	if brand == "" || brand == "domains" {
		httpError(w, http.StatusBadRequest, "缺少品牌参数")
		return
	}

	data, err := fetchDomainsJSON()
	if err != nil {
		httpError(w, http.StatusBadGateway, "获取域名配置失败")
		return
	}

	panels, ok := data["panels"].(map[string]interface{})
	if !ok {
		httpError(w, http.StatusInternalServerError, "域名配置格式错误")
		return
	}

	brandPanels, ok := panels[brand].([]interface{})
	if !ok || len(brandPanels) == 0 {
		httpError(w, http.StatusNotFound, "品牌不存在或无可用域名")
		return
	}

	firstPanel, ok := brandPanels[0].(map[string]interface{})
	if !ok {
		httpError(w, http.StatusInternalServerError, "域名配置格式错误")
		return
	}

	targetURL, ok := firstPanel["url"].(string)
	if !ok || targetURL == "" {
		httpError(w, http.StatusInternalServerError, "域名 URL 无效")
		return
	}

	http.Redirect(w, r, targetURL, http.StatusFound)
}
