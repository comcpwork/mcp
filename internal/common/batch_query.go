package common

import (
	"context"
	"fmt"
	"mcp/pkg/log"
	"strings"
	"sync"
	"time"
)

// BatchQueryResult 批量查询结果
type BatchQueryResult struct {
	ID      string // 查询项目的标识符
	Success bool   // 是否成功
	Content string // 成功时的内容
	Error   string // 失败时的错误信息
}

// BatchQuery 执行批量查询的通用函数
func BatchQuery(ctx context.Context, items []string, queryFunc func(context.Context, string) (string, error)) []BatchQueryResult {
	return BatchQueryWithConcurrency(ctx, items, queryFunc, MaxBatchConcurrency)
}

// BatchQueryWithConcurrency 执行批量查询（可指定并发数）
func BatchQueryWithConcurrency(ctx context.Context, items []string, queryFunc func(context.Context, string) (string, error), maxConcurrency int) []BatchQueryResult {
	if len(items) == 0 {
		return []BatchQueryResult{}
	}
	
	// 记录批量查询开始
	startTime := time.Now()
	log.Info(ctx, "开始批量查询",
		log.String(FieldOperation, "batch_query"),
		log.Int(FieldCount, len(items)),
		log.Int("concurrency", maxConcurrency))
	
	// 创建结果切片
	results := make([]BatchQueryResult, len(items))
	
	// 如果只有一个项目，直接执行
	if len(items) == 1 {
		content, err := queryFunc(ctx, items[0])
		results[0] = BatchQueryResult{
			ID:      items[0],
			Success: err == nil,
			Content: content,
		}
		if err != nil {
			results[0].Error = err.Error()
		}
		return results
	}
	
	// 使用 goroutines 并发执行
	sem := make(chan bool, maxConcurrency)
	var wg sync.WaitGroup
	
	for i, item := range items {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()
			
			// 获取信号量
			sem <- true
			defer func() { <-sem }()
			
			// 执行查询
			content, err := queryFunc(ctx, id)
			
			// 保存结果
			results[index] = BatchQueryResult{
				ID:      id,
				Success: err == nil,
				Content: content,
			}
			
			if err != nil {
				results[index].Error = err.Error()
			}
		}(i, item)
	}
	
	// 等待所有查询完成
	wg.Wait()
	
	// 统计成功和失败的数量
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}
	
	// 记录批量查询完成
	duration := time.Since(startTime).Milliseconds()
	log.Info(ctx, "批量查询完成",
		log.String(FieldOperation, "batch_query"),
		log.String(FieldStatus, "success"),
		log.Int("success_count", successCount),
		log.Int("failed_count", len(items)-successCount),
		log.Int64(FieldDuration, duration))
	
	return results
}

// FormatBatchResults 格式化批量查询结果
func FormatBatchResults(results []BatchQueryResult, queryType string) string {
	if len(results) == 0 {
		return "No results"
	}
	
	var result strings.Builder
	
	// 统计信息
	total := len(results)
	successCount := 0
	failedCount := 0
	
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failedCount++
		}
	}
	
	// 头部信息
	result.WriteString(fmt.Sprintf("Batch %s Results (%d total, %d successful, %d failed)\n\n",
		queryType, total, successCount, failedCount))
	
	// 详细结果
	for _, r := range results {
		if r.Success {
			result.WriteString(fmt.Sprintf("=== %s ===\n", r.ID))
			result.WriteString(r.Content)
			result.WriteString("\n\n")
		} else {
			result.WriteString(fmt.Sprintf("=== %s (FAILED) ===\n", r.ID))
			result.WriteString(fmt.Sprintf("Error: %s\n\n", r.Error))
		}
	}
	
	// 添加统计摘要
	percentage := float64(successCount) / float64(total) * 100
	result.WriteString(fmt.Sprintf("Summary: %d/%d successful (%.1f%%)", 
		successCount, total, percentage))
	
	return result.String()
}

// ExtractArrayParameter 从请求中提取数组参数
func ExtractArrayParameter(request map[string]interface{}, key string) []string {
	var result []string
	
	if args, ok := request["arguments"].(map[string]interface{}); ok {
		if param, exists := args[key]; exists {
			// 尝试转换为字符串切片
			if slice, ok := param.([]interface{}); ok {
				for _, item := range slice {
					if str, ok := item.(string); ok {
						result = append(result, str)
					}
				}
			}
		}
	}
	
	return result
}