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
