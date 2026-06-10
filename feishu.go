package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

var feishuAPI = "https://open.feishu.cn/open-apis"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func getTenantAccessToken(appID, appSecret string) (string, error) {
	var token string
	err := withRetry("获取 tenant_access_token", func() error {
		body, _ := json.Marshal(map[string]string{"app_id": appID, "app_secret": appSecret})
		resp, err := httpClient.Post(feishuAPI+"/auth/v3/tenant_access_token/internal", "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("请求失败: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("触发限流 (HTTP 429)")
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("服务器错误 (HTTP %d)", resp.StatusCode)
		}

		var result struct {
			Code              int    `json:"code"`
			Msg               string `json:"msg"`
			TenantAccessToken string `json:"tenant_access_token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
		if result.Code != 0 {
			return fmt.Errorf("API 错误: code=%d msg=%s", result.Code, result.Msg)
		}
		token = result.TenantAccessToken
		return nil
	})
	return token, err
}

func resolveWikiAppToken(token, wikiNodeToken string) (string, error) {
	var appToken string
	err := withRetry("解析 wiki 节点", func() error {
		req, _ := http.NewRequest("GET", feishuAPI+"/wiki/v2/spaces/get_node", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		q := req.URL.Query()
		q.Set("token", wikiNodeToken)
		req.URL.RawQuery = q.Encode()

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("请求失败: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("触发限流 (HTTP 429)")
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("服务器错误 (HTTP %d)", resp.StatusCode)
		}

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				Node struct {
					ObjType  string `json:"obj_type"`
					ObjToken string `json:"obj_token"`
				} `json:"node"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
		if result.Code != 0 {
			return fmt.Errorf("API 错误: code=%d msg=%s\n请确认应用已开通权限: wiki:node:read 或 wiki:wiki:readonly", result.Code, result.Msg)
		}
		node := result.Data.Node
		if node.ObjType != "bitable" {
			return fmt.Errorf("该 wiki 节点不是多维表格，obj_type=%s", node.ObjType)
		}
		appToken = node.ObjToken
		return nil
	})
	return appToken, err
}

func listRecords(token, appToken, tableID, viewID, pageToken string) (items []map[string]any, hasMore bool, nextToken string, err error) {
	err = withRetry("读取表格记录", func() error {
		url := fmt.Sprintf("%s/bitable/v1/apps/%s/tables/%s/records", feishuAPI, appToken, tableID)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		q := req.URL.Query()
		q.Set("page_size", "500")
		if viewID != "" {
			q.Set("view_id", viewID)
		}
		if pageToken != "" {
			q.Set("page_token", pageToken)
		}
		req.URL.RawQuery = q.Encode()

		resp, reqErr := httpClient.Do(req)
		if reqErr != nil {
			return fmt.Errorf("请求失败: %w", reqErr)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("触发限流 (HTTP 429)")
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("服务器错误 (HTTP %d)", resp.StatusCode)
		}

		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				Items []struct {
					Fields map[string]any `json:"fields"`
				} `json:"items"`
				HasMore   bool   `json:"has_more"`
				PageToken string `json:"page_token"`
			} `json:"data"`
		}
		if decErr := json.NewDecoder(resp.Body).Decode(&result); decErr != nil {
			return fmt.Errorf("解析响应失败: %w", decErr)
		}
		if result.Code != 0 {
			return fmt.Errorf("API 错误: code=%d msg=%s", result.Code, result.Msg)
		}

		items = items[:0]
		for _, item := range result.Data.Items {
			items = append(items, item.Fields)
		}
		hasMore = result.Data.HasMore
		nextToken = result.Data.PageToken
		return nil
	})
	return
}

func fetchAllRecords(token, appToken, tableID, viewID string) ([]map[string]any, error) {
	var all []map[string]any
	pageToken := ""
	page := 1
	for {
		fmt.Fprintf(os.Stderr, "  读取第 %d 页...\n", page)
		items, hasMore, next, err := listRecords(token, appToken, tableID, viewID, pageToken)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		fmt.Fprintf(os.Stderr, "  已获取 %d 条\n", len(all))
		if !hasMore {
			break
		}
		pageToken = next
		page++
	}
	return all, nil
}
