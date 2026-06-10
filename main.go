package main

// 从飞书多维表格读取 key/value 记录，输出为环境变量格式。
//
// 用法:
//
//	export FEISHU_APP_ID="cli_xxx"
//	export FEISHU_APP_SECRET="xxx"
//
//	# 输出 export 语句，可直接 eval
//	eval $(load_feishu_wiki_env "https://xxx.feishu.cn/wiki/XYZ?table=tblABC")
//
//	# 输出 .env 格式
//	load_feishu_wiki_env --format dotenv "https://..." > .env

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const maxRetries = 3

var feishuAPI = "https://open.feishu.cn/open-apis"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// ---------- 重试 ----------

// withRetry 以指数退避（1s / 2s / 4s）重试 fn，共最多 maxRetries 次。
// 所有错误均重试（网络抖动、429 限流、5xx 等），因为这是幂等读操作。
func withRetry(opName string, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			if attempt > 1 {
				fmt.Fprintf(os.Stderr, "  [%s] 第 %d 次尝试成功\n", opName, attempt)
			}
			return nil
		}
		if attempt == maxRetries {
			break
		}
		wait := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s
		fmt.Fprintf(os.Stderr, "  [%s] 第 %d 次失败，%v 后重试: %v\n", opName, attempt, wait, lastErr)
		time.Sleep(wait)
	}
	return fmt.Errorf("%s 失败（重试 %d 次仍未成功）: %w", opName, maxRetries, lastErr)
}

// ---------- API helpers ----------

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

// ---------- URL parsing ----------

func parseFeishuURL(rawURL string) (urlType, nodeToken, tableID, viewID string, err error) {
	idx := strings.Index(rawURL, "?")
	path := rawURL
	query := ""
	if idx >= 0 {
		path, query = rawURL[:idx], rawURL[idx+1:]
	}

	if fi := strings.Index(query, "#"); fi >= 0 {
		query = query[:fi]
	}
	if fi := strings.Index(path, "#"); fi >= 0 {
		path = path[:fi]
	}

	tableID = queryParam(query, "table")
	if tableID == "" {
		tableID = queryParam(query, "tableId")
	}
	viewID = queryParam(query, "view")
	if viewID == "" {
		viewID = queryParam(query, "viewId")
	}

	if i := strings.Index(path, "://"); i >= 0 {
		path = path[i+3:]
	}
	if i := strings.Index(path, "/"); i >= 0 {
		path = path[i:]
	}

	parts := []string{}
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) < 2 {
		return "", "", "", "", fmt.Errorf("无法从 URL 中提取 token，路径: %s", path)
	}
	urlType = parts[0]
	if urlType != "base" && urlType != "wiki" {
		return "", "", "", "", fmt.Errorf("不支持的 URL 类型 /%s/，仅支持 /base/ 和 /wiki/", urlType)
	}
	nodeToken = parts[1]
	return urlType, nodeToken, tableID, viewID, nil
}

func queryParam(query, key string) string {
	for _, part := range strings.Split(query, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == key {
			return kv[1]
		}
	}
	return ""
}

// ---------- Field value extraction ----------

// quoteDotenvValue 将值包裹在双引号中，转义 \ 和 "，使其在 docker compose
// env_file（godotenv 解析）中被正确读取。
// 值包含换行符时直接报错——密钥类数据不应包含换行，出现说明数据有误。
func quoteDotenvValue(k, v string) (string, error) {
	if strings.ContainsAny(v, "\n\r") {
		return "", fmt.Errorf("key %q 的值包含换行符，env_file 格式不支持多行值", k)
	}
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return `"` + v + `"`, nil
}

// fieldStr 将飞书字段值转为字符串。
// 文本字段有时返回 []any（富文本 segment 数组），需拼接 .text。
func fieldStr(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case []any:
		var sb strings.Builder
		for _, seg := range val {
			if m, ok := seg.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					sb.WriteString(t)
				}
			}
		}
		return sb.String()
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// ---------- main ----------

func main() {
	var (
		appID      = flag.String("app-id", os.Getenv("FEISHU_APP_ID"), "飞书 App ID（或 FEISHU_APP_ID 环境变量）")
		appSecret  = flag.String("app-secret", os.Getenv("FEISHU_APP_SECRET"), "飞书 App Secret（或 FEISHU_APP_SECRET 环境变量）")
		tableIDArg = flag.String("table-id", "", "覆盖 URL 中的 table_id")
		viewIDArg  = flag.String("view-id", "", "覆盖 URL 中的 view_id")
		keyField   = flag.String("key-field", "key", "key 字段名")
		valueField = flag.String("value-field", "value", "value 字段名")
		format     = flag.String("format", "export", "输出格式: export | dotenv | json")
		outputFile = flag.String("output", "", "输出到文件（默认 stdout）")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: %s [flags] <飞书多维表格链接>\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "示例:")
		fmt.Fprintln(os.Stderr, `  eval $(load_feishu_wiki_env "https://xxx.feishu.cn/wiki/XYZ?table=tblABC")`)
		fmt.Fprintln(os.Stderr, `  load_feishu_wiki_env --format dotenv "https://..." > .env`)
		fmt.Fprintln(os.Stderr, "")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	rawURL := flag.Arg(0)

	if *appID == "" || *appSecret == "" {
		fmt.Fprintln(os.Stderr, "错误: 需要 --app-id / --app-secret，或设置环境变量 FEISHU_APP_ID / FEISHU_APP_SECRET")
		os.Exit(1)
	}

	urlType, nodeToken, tableID, viewID, err := parseFeishuURL(rawURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
	if *tableIDArg != "" {
		tableID = *tableIDArg
	}
	if *viewIDArg != "" {
		viewID = *viewIDArg
	}
	if tableID == "" {
		fmt.Fprintln(os.Stderr, "错误: URL 中未包含 table 参数，请用 --table-id 手动指定")
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "获取 tenant_access_token ...")
	token, err := getTenantAccessToken(*appID, *appSecret)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "tenant_access_token 获取成功")

	appToken := nodeToken
	if urlType == "wiki" {
		fmt.Fprintf(os.Stderr, "解析 wiki 节点 %s ...\n", nodeToken)
		appToken, err = resolveWikiAppToken(token, nodeToken)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "app_token: %s\n", appToken)
	}

	fmt.Fprintf(os.Stderr, "读取表格 app=%s table=%s ...\n", appToken, tableID)
	records, err := fetchAllRecords(token, appToken, tableID, viewID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "共 %d 条记录\n", len(records))

	var out io.Writer = os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	switch *format {
	case "export":
		writeExport(out, records, *keyField, *valueField)
	case "dotenv":
		if err := writeDotenv(out, records, *keyField, *valueField); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	case "json":
		writeJSON(out, records, *keyField, *valueField)
	default:
		fmt.Fprintf(os.Stderr, "错误: 未知输出格式 %q，支持: export | dotenv | json\n", *format)
		os.Exit(1)
	}
}

// ---------- output formatters ----------

func writeExport(out io.Writer, records []map[string]any, keyField, valueField string) {
	for _, fields := range records {
		k := fieldStr(fields[keyField])
		v := fieldStr(fields[valueField])
		if k == "" {
			continue
		}
		fmt.Fprintf(out, "export %s=%q\n", k, v)
	}
}

func writeDotenv(out io.Writer, records []map[string]any, keyField, valueField string) error {
	for _, fields := range records {
		k := fieldStr(fields[keyField])
		v := fieldStr(fields[valueField])
		if k == "" {
			continue
		}
		quoted, err := quoteDotenvValue(k, v)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s=%s\n", k, quoted)
	}
	return nil
}

func writeJSON(out io.Writer, records []map[string]any, keyField, valueField string) {
	envMap := make(map[string]string, len(records))
	for _, fields := range records {
		k := fieldStr(fields[keyField])
		v := fieldStr(fields[valueField])
		if k == "" {
			continue
		}
		envMap[k] = v
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(envMap)
}
