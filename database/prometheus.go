package database

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// handlePrometheusExec 处理 Prometheus 执行请求
func handlePrometheusExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取 DSN 参数
	dsn, err := req.RequireString("dsn")
	if err != nil {
		return mcp.NewToolResultError("Missing dsn parameter"), nil
	}

	// 获取 Query 参数
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("Missing query parameter"), nil
	}

	// 获取可选的时间范围参数
	startStr := req.GetString("start", "")
	endStr := req.GetString("end", "")
	stepStr := req.GetString("step", "")

	// 检查是否需要SSH隧道
	sshURI := req.GetString("ssh", "")
	var tunnel *PooledSSHTunnel
	if sshURI != "" {
		// 从DSN中提取目标地址
		remoteHost, remotePort, err := ExtractPrometheusHostPort(dsn)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse DSN: %v", err)), nil
		}

		// 从连接池获取SSH隧道
		tunnel, err = GetSSHPool().GetTunnel(sshURI, remoteHost, remotePort)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("SSH tunnel failed: %v", err)), nil
		}
		defer tunnel.Close()

		// 替换DSN中的地址为本地隧道地址
		dsn = ReplacePrometheusDSNHostPort(dsn, tunnel.LocalAddr())
	}

	// 解析 DSN
	addr, basicAuth, err := parsePrometheusDSN(dsn)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid DSN: %v", err)), nil
	}

	// 创建带超时的 HTTP 客户端
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
	}

	// 创建 Prometheus API 客户端
	config := api.Config{
		Address:      addr,
		RoundTripper: httpClient.Transport,
	}
	if basicAuth != nil {
		config.RoundTripper = &basicAuthRoundTripper{
			username:  basicAuth.username,
			password:  basicAuth.password,
			transport: httpClient.Transport,
		}
	}

	client, err := api.NewClient(config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create client: %v", err)), nil
	}

	apiClient := v1.NewAPI(client)

	// 添加超时控制
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 解析 Query，分离命令和过滤器
	command, filter := parsePrometheusQuery(query)

	// 判断是内置命令还是 PromQL
	if isBuiltinCommand(command) {
		return executeBuiltinCommand(ctx, apiClient, command, filter)
	}

	// PromQL 查询（不支持管道过滤）
	if startStr != "" && endStr != "" && stepStr != "" {
		return executeRangeQuery(ctx, apiClient, query, startStr, endStr, stepStr)
	}
	return executeInstantQuery(ctx, apiClient, query)
}

// basicAuthInfo 基本认证信息
type basicAuthInfo struct {
	username string
	password string
}

// basicAuthRoundTripper 添加基本认证的 RoundTripper
type basicAuthRoundTripper struct {
	username  string
	password  string
	transport http.RoundTripper
}

func (rt *basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// 克隆请求以避免修改原始请求
	req2 := req.Clone(req.Context())
	req2.SetBasicAuth(rt.username, rt.password)
	transport := rt.transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return transport.RoundTrip(req2)
}

// parsePrometheusDSN 解析 Prometheus DSN
// 格式: prometheus://[user:pass@]host:port
func parsePrometheusDSN(dsn string) (string, *basicAuthInfo, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", nil, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "prometheus" {
		return "", nil, fmt.Errorf("invalid scheme: expected 'prometheus', got '%s'", u.Scheme)
	}

	// 构建 HTTP 地址
	addr := fmt.Sprintf("http://%s", u.Host)

	// 检查是否有认证信息
	var auth *basicAuthInfo
	if u.User != nil {
		password, _ := u.User.Password()
		auth = &basicAuthInfo{
			username: u.User.Username(),
			password: password,
		}
	}

	return addr, auth, nil
}

// ExtractPrometheusHostPort 从 Prometheus DSN 中提取主机和端口
func ExtractPrometheusHostPort(dsn string) (string, int, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", 0, fmt.Errorf("invalid URL: %w", err)
	}

	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "9090" // Prometheus 默认端口
	}

	port := 9090
	fmt.Sscanf(portStr, "%d", &port)

	return host, port, nil
}

// ReplacePrometheusDSNHostPort 替换 Prometheus DSN 中的主机和端口
func ReplacePrometheusDSNHostPort(dsn string, newHostPort string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}

	u.Host = newHostPort
	return u.String()
}

// parsePrometheusQuery 解析查询表达式，分离命令和匹配器
// 只支持一个管道符，管道后面是 PromQL 匹配器（如 {job="prometheus"}）
func parsePrometheusQuery(query string) (command string, matcher string) {
	// 只分割成2部分，确保只有一个管道
	parts := strings.SplitN(query, "|", 2)
	command = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		matcher = strings.TrimSpace(parts[1])
	}
	return
}

// isBuiltinCommand 判断是否为内置命令
func isBuiltinCommand(cmd string) bool {
	upper := strings.ToUpper(cmd)
	return strings.HasPrefix(upper, "SHOW ") ||
		strings.HasPrefix(upper, "DESCRIBE ")
}

// executeBuiltinCommand 执行内置命令
func executeBuiltinCommand(ctx context.Context, api v1.API, command, matcher string) (*mcp.CallToolResult, error) {
	upper := strings.ToUpper(command)

	// 构建 matches 参数
	var matches []string
	if matcher != "" {
		matches = []string{matcher}
	}

	switch {
	case strings.HasPrefix(upper, "SHOW METRICS"):
		return executeShowMetrics(ctx, api, matches, matcher)
	case strings.HasPrefix(upper, "SHOW LABEL VALUES "):
		// 提取标签名
		parts := strings.Fields(command)
		if len(parts) < 4 {
			return mcp.NewToolResultError("Invalid command: SHOW LABEL VALUES requires a label name"), nil
		}
		labelName := parts[3]
		return executeShowLabelValues(ctx, api, labelName, matches, matcher)
	case strings.HasPrefix(upper, "SHOW LABELS"):
		return executeShowLabels(ctx, api, matches, matcher)
	case strings.HasPrefix(upper, "DESCRIBE "):
		// 提取指标名
		parts := strings.Fields(command)
		if len(parts) < 2 {
			return mcp.NewToolResultError("Invalid command: DESCRIBE requires a metric name"), nil
		}
		metricName := parts[1]
		return executeDescribe(ctx, api, metricName)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("Unknown command: %s", command)), nil
	}
}

// executeShowMetrics 执行 SHOW METRICS 命令
func executeShowMetrics(ctx context.Context, api v1.API, matches []string, matcher string) (*mcp.CallToolResult, error) {
	// 获取指标名（使用 matches 参数过滤）
	labelValues, warnings, err := api.LabelValues(ctx, "__name__", matches, time.Time{}, time.Time{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get metrics: %v", err)), nil
	}

	// 转换为字符串切片
	metrics := make([]string, 0, len(labelValues))
	for _, v := range labelValues {
		metrics = append(metrics, string(v))
	}

	// 排序
	sort.Strings(metrics)

	// 格式化输出
	output := formatMetricsList(metrics, matcher, warnings)
	return mcp.NewToolResultText(output), nil
}

// executeShowLabels 执行 SHOW LABELS 命令
func executeShowLabels(ctx context.Context, api v1.API, matches []string, matcher string) (*mcp.CallToolResult, error) {
	// 获取标签名（使用 matches 参数过滤）
	labels, warnings, err := api.LabelNames(ctx, matches, time.Time{}, time.Time{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get labels: %v", err)), nil
	}

	// 排序
	sort.Strings(labels)

	// 格式化输出
	output := formatLabelsList(labels, matcher, warnings)
	return mcp.NewToolResultText(output), nil
}

// executeShowLabelValues 执行 SHOW LABEL VALUES 命令
func executeShowLabelValues(ctx context.Context, api v1.API, labelName string, matches []string, matcher string) (*mcp.CallToolResult, error) {
	// 获取标签的所有值（使用 matches 参数过滤）
	labelValues, warnings, err := api.LabelValues(ctx, labelName, matches, time.Time{}, time.Time{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get label values: %v", err)), nil
	}

	// 转换为字符串切片
	values := make([]string, 0, len(labelValues))
	for _, v := range labelValues {
		values = append(values, string(v))
	}

	// 排序
	sort.Strings(values)

	// 格式化输出
	output := formatLabelValuesList(labelName, values, matcher, warnings)
	return mcp.NewToolResultText(output), nil
}

// executeDescribe 执行 DESCRIBE 命令
func executeDescribe(ctx context.Context, api v1.API, metricName string) (*mcp.CallToolResult, error) {
	// 获取指标元数据
	metadata, err := api.Metadata(ctx, metricName, "")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get metadata: %v", err)), nil
	}

	metaList, ok := metadata[metricName]
	if !ok || len(metaList) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("Metric not found: %s", metricName)), nil
	}

	// 格式化输出
	output := formatMetadata(metricName, metaList)
	return mcp.NewToolResultText(output), nil
}

// executeInstantQuery 执行即时查询
func executeInstantQuery(ctx context.Context, api v1.API, query string) (*mcp.CallToolResult, error) {
	result, warnings, err := api.Query(ctx, query, time.Now())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Query failed: %v", err)), nil
	}

	// 格式化输出
	output := formatInstantResult(query, result, warnings)
	return mcp.NewToolResultText(output), nil
}

// executeRangeQuery 执行范围查询
func executeRangeQuery(ctx context.Context, api v1.API, query, startStr, endStr, stepStr string) (*mcp.CallToolResult, error) {
	// 解析时间参数
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid start time: %v", err)), nil
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid end time: %v", err)), nil
	}

	step, err := model.ParseDuration(stepStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid step: %v", err)), nil
	}

	r := v1.Range{
		Start: start,
		End:   end,
		Step:  time.Duration(step),
	}

	result, warnings, err := api.QueryRange(ctx, query, r)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Query failed: %v", err)), nil
	}

	// 格式化输出
	output := formatRangeResult(query, r, result, warnings)
	return mcp.NewToolResultText(output), nil
}

// formatMetricsList 格式化指标列表
func formatMetricsList(metrics []string, matcher string, warnings v1.Warnings) string {
	var output strings.Builder

	if matcher != "" {
		output.WriteString(fmt.Sprintf("Query: SHOW METRICS | %s\n", matcher))
	} else {
		output.WriteString("Query: SHOW METRICS\n")
	}
	output.WriteString(fmt.Sprintf("Result: %d metrics\n", len(metrics)))

	if len(warnings) > 0 {
		output.WriteString(fmt.Sprintf("Warnings: %v\n", warnings))
	}

	output.WriteString("\n")

	for _, m := range metrics {
		output.WriteString(m)
		output.WriteString("\n")
	}

	return output.String()
}

// formatLabelsList 格式化标签列表
func formatLabelsList(labels []string, matcher string, warnings v1.Warnings) string {
	var output strings.Builder

	if matcher != "" {
		output.WriteString(fmt.Sprintf("Query: SHOW LABELS | %s\n", matcher))
	} else {
		output.WriteString("Query: SHOW LABELS\n")
	}
	output.WriteString(fmt.Sprintf("Result: %d labels\n", len(labels)))

	if len(warnings) > 0 {
		output.WriteString(fmt.Sprintf("Warnings: %v\n", warnings))
	}

	output.WriteString("\n")

	for _, l := range labels {
		output.WriteString(l)
		output.WriteString("\n")
	}

	return output.String()
}

// formatLabelValuesList 格式化标签值列表
func formatLabelValuesList(labelName string, values []string, matcher string, warnings v1.Warnings) string {
	var output strings.Builder

	if matcher != "" {
		output.WriteString(fmt.Sprintf("Query: SHOW LABEL VALUES %s | %s\n", labelName, matcher))
	} else {
		output.WriteString(fmt.Sprintf("Query: SHOW LABEL VALUES %s\n", labelName))
	}
	output.WriteString(fmt.Sprintf("Result: %d values\n", len(values)))

	if len(warnings) > 0 {
		output.WriteString(fmt.Sprintf("Warnings: %v\n", warnings))
	}

	output.WriteString("\n")

	for _, v := range values {
		output.WriteString(v)
		output.WriteString("\n")
	}

	return output.String()
}

// formatMetadata 格式化元数据
func formatMetadata(metricName string, metaList []v1.Metadata) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Query: DESCRIBE %s\n\n", metricName))
	output.WriteString(fmt.Sprintf("Metric: %s\n", metricName))

	// 可能有多个元数据条目（来自不同的 target）
	if len(metaList) > 0 {
		meta := metaList[0]
		output.WriteString(fmt.Sprintf("Type:   %s\n", meta.Type))
		output.WriteString(fmt.Sprintf("Help:   %s\n", meta.Help))
		if meta.Unit != "" {
			output.WriteString(fmt.Sprintf("Unit:   %s\n", meta.Unit))
		}
	}

	return output.String()
}

// formatInstantResult 格式化即时查询结果
func formatInstantResult(query string, result model.Value, warnings v1.Warnings) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Query: %s\n", query))
	output.WriteString(fmt.Sprintf("Time: %s\n", time.Now().Format(time.RFC3339)))

	if len(warnings) > 0 {
		output.WriteString(fmt.Sprintf("Warnings: %v\n", warnings))
	}

	switch v := result.(type) {
	case model.Vector:
		output.WriteString(fmt.Sprintf("Result: %d series\n\n", len(v)))

		if len(v) == 0 {
			output.WriteString("No data\n")
		} else {
			// 计算最大指标名长度
			maxLen := 6 // "Metric"
			for _, sample := range v {
				metricStr := sample.Metric.String()
				if len(metricStr) > maxLen {
					maxLen = len(metricStr)
				}
			}

			// 限制最大宽度
			if maxLen > 60 {
				maxLen = 60
			}

			// 表头
			output.WriteString(fmt.Sprintf("%-*s  Value\n", maxLen, "Metric"))
			output.WriteString(strings.Repeat("-", maxLen))
			output.WriteString("  -----\n")

			// 数据行
			for _, sample := range v {
				metricStr := sample.Metric.String()
				if len(metricStr) > maxLen {
					metricStr = metricStr[:maxLen-3] + "..."
				}
				output.WriteString(fmt.Sprintf("%-*s  %v\n", maxLen, metricStr, sample.Value))
			}
		}

	case *model.Scalar:
		output.WriteString("Result: scalar\n\n")
		output.WriteString(fmt.Sprintf("Value: %v\n", v.Value))

	case *model.String:
		output.WriteString("Result: string\n\n")
		output.WriteString(fmt.Sprintf("Value: %s\n", v.Value))

	default:
		output.WriteString(fmt.Sprintf("Result type: %T\n", result))
		output.WriteString(fmt.Sprintf("Value: %v\n", result))
	}

	return output.String()
}

// formatRangeResult 格式化范围查询结果
func formatRangeResult(query string, r v1.Range, result model.Value, warnings v1.Warnings) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Query: %s\n", query))
	output.WriteString(fmt.Sprintf("Range: %s - %s (step: %s)\n",
		r.Start.Format(time.RFC3339),
		r.End.Format(time.RFC3339),
		r.Step.String()))

	if len(warnings) > 0 {
		output.WriteString(fmt.Sprintf("Warnings: %v\n", warnings))
	}

	switch v := result.(type) {
	case model.Matrix:
		output.WriteString(fmt.Sprintf("Result: %d series\n\n", len(v)))

		if len(v) == 0 {
			output.WriteString("No data\n")
		} else {
			for _, stream := range v {
				output.WriteString(fmt.Sprintf("--- %s ---\n", stream.Metric.String()))

				// 表头
				output.WriteString("Time                      Value\n")
				output.WriteString("------------------------  -------\n")

				// 数据行
				for _, sample := range stream.Values {
					output.WriteString(fmt.Sprintf("%-24s  %v\n",
						sample.Timestamp.Time().Format(time.RFC3339),
						sample.Value))
				}
				output.WriteString("\n")
			}
		}

	default:
		output.WriteString(fmt.Sprintf("Result type: %T\n", result))
		output.WriteString(fmt.Sprintf("Value: %v\n", result))
	}

	return output.String()
}
