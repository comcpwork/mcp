package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/redis/go-redis/v9"
)

// handleRedisExec 处理 Redis 执行请求
func handleRedisExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取 DSN 参数
	dsn, err := req.RequireString("dsn")
	if err != nil {
		return mcp.NewToolResultError("Missing dsn parameter"), nil
	}

	// 获取 command 参数
	command, err := req.RequireString("command")
	if err != nil {
		return mcp.NewToolResultError("Missing command parameter"), nil
	}

	// 检查是否需要SSH隧道
	sshURI := req.GetString("ssh", "")
	var tunnel *PooledSSHTunnel
	if sshURI != "" {
		// 从DSN中提取目标地址
		remoteHost, remotePort, err := ExtractRedisHostPort(dsn)
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
		dsn = ReplaceRedisDSNHostPort(dsn, tunnel.LocalAddr())
	}

	// 解析 DSN
	opt, err := redis.ParseURL(dsn)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid DSN: %v", err)), nil
	}

	// 创建 Redis 客户端
	client := redis.NewClient(opt)
	defer client.Close()

	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Connection failed: %v", err)), nil
	}

	// 解析命令
	args, err := parseRedisCommand(command)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Command parse failed: %v", err)), nil
	}

	if len(args) == 0 {
		return mcp.NewToolResultError("Empty command"), nil
	}

	// 执行命令
	result := client.Do(ctx, args...)
	if result.Err() != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Command failed: %v", result.Err())), nil
	}

	// 格式化输出
	output := formatRedisResult(command, result.Val())
	return mcp.NewToolResultText(output), nil
}

// parseRedisCommand 解析 Redis 命令
func parseRedisCommand(command string) ([]interface{}, error) {
	if command == "" {
		return nil, fmt.Errorf("empty command")
	}

	// 简单的命令解析，按空格分割
	parts := strings.Fields(command)
	args := make([]interface{}, len(parts))
	for i, part := range parts {
		// 去除引号
		part = strings.Trim(part, "\"'")
		args[i] = part
	}

	return args, nil
}

// formatRedisResult 格式化 Redis 结果
func formatRedisResult(command string, value interface{}) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Command: %s\n", command))
	output.WriteString("Result: ")

	switch v := value.(type) {
	case nil:
		output.WriteString("(nil)\n")
	case string:
		output.WriteString(fmt.Sprintf("%s\n", v))
	case int64:
		output.WriteString(fmt.Sprintf("(integer) %d\n", v))
	case []interface{}:
		if len(v) == 0 {
			output.WriteString("(empty array)\n")
		} else {
			output.WriteString(fmt.Sprintf("(%d items)\n", len(v)))
			for i, item := range v {
				output.WriteString(fmt.Sprintf("%d) %v\n", i+1, item))
			}
		}
	case map[string]interface{}:
		if len(v) == 0 {
			output.WriteString("(empty hash)\n")
		} else {
			output.WriteString(fmt.Sprintf("(%d fields)\n", len(v)))
			for key, val := range v {
				output.WriteString(fmt.Sprintf("%s: %v\n", key, val))
			}
		}
	default:
		output.WriteString(fmt.Sprintf("%v\n", v))
	}

	return output.String()
}