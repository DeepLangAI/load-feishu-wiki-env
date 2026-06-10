package main

import (
	"fmt"
	"os"
	"time"
)

const maxRetries = 3

// withRetry 以指数退避（1s / 2s）重试 fn，共最多 maxRetries 次。
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
		wait := time.Duration(1<<uint(attempt-1)) * time.Second
		fmt.Fprintf(os.Stderr, "  [%s] 第 %d 次失败，%v 后重试: %v\n", opName, attempt, wait, lastErr)
		time.Sleep(wait)
	}
	return fmt.Errorf("%s 失败（重试 %d 次仍未成功）: %w", opName, maxRetries, lastErr)
}
