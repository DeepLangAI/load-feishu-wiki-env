package main

import (
	"fmt"
	"strings"
)

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

	var parts []string
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
