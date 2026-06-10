package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
