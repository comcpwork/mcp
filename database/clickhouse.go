package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleClickHouseExec 处理 ClickHouse 执行请求
func handleClickHouseExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取 DSN 参数
	dsn, err := req.RequireString("dsn")
	if err != nil {
		return mcp.NewToolResultError("Missing dsn parameter"), nil
	}

	// 获取 SQL 参数
	sqlQuery, err := req.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError("Missing sql parameter"), nil
	}

	// 检查是否需要SSH隧道
	sshURI := req.GetString("ssh", "")
	var tunnel *PooledSSHTunnel
	if sshURI != "" {
		// 从DSN中提取目标地址
		remoteHost, remotePort, err := ExtractClickHouseHostPort(dsn)
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
		dsn = ReplaceClickHouseDSNHostPort(dsn, tunnel.LocalAddr())
	}

	// 打开数据库连接
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Database connection failed: %v", err)), nil
	}
	defer db.Close()

	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Database connection failed: %v", err)), nil
	}

	// 判断是查询还是执行操作
	if isClickHouseQueryStatement(sqlQuery) {
		return executeClickHouseQuery(ctx, db, sqlQuery)
	}
	return executeClickHouseModification(ctx, db, sqlQuery)
}

// isClickHouseQueryStatement 判断是否为查询语句
func isClickHouseQueryStatement(sqlQuery string) bool {
	lowerSQL := strings.ToLower(strings.TrimSpace(sqlQuery))
	return strings.HasPrefix(lowerSQL, "select") ||
		strings.HasPrefix(lowerSQL, "show") ||
		strings.HasPrefix(lowerSQL, "describe") ||
		strings.HasPrefix(lowerSQL, "desc") ||
		strings.HasPrefix(lowerSQL, "explain") ||
		strings.HasPrefix(lowerSQL, "exists")
}

// executeClickHouseQuery 执行查询操作
func executeClickHouseQuery(ctx context.Context, db *sql.DB, sqlQuery string) (*mcp.CallToolResult, error) {
	rows, err := db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Query failed: %v", err)), nil
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get columns: %v", err)), nil
	}

	// 读取结果
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val == nil {
				row[col] = nil
			} else if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error iterating results: %v", err)), nil
	}

	// 格式化输出
	output := formatClickHouseQueryResult(columns, results, sqlQuery)
	return mcp.NewToolResultText(output), nil
}

// executeClickHouseModification 执行修改操作
func executeClickHouseModification(ctx context.Context, db *sql.DB, sqlQuery string) (*mcp.CallToolResult, error) {
	result, err := db.ExecContext(ctx, sqlQuery)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Execute failed: %v", err)), nil
	}

	rowsAffected, _ := result.RowsAffected()

	output := formatClickHouseModificationResult(sqlQuery, rowsAffected)
	return mcp.NewToolResultText(output), nil
}

// formatClickHouseQueryResult 格式化查询结果
func formatClickHouseQueryResult(columns []string, results []map[string]interface{}, sqlQuery string) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Query: %s\n", sqlQuery))
	output.WriteString(fmt.Sprintf("Result: %d rows x %d columns\n", len(results), len(columns)))

	if len(results) == 0 {
		output.WriteString("No data\n")
		return output.String()
	}

	// 输出列名（数组格式）
	output.WriteString("Columns: [")
	for i, col := range columns {
		if i > 0 {
			output.WriteString(", ")
		}
		output.WriteString(fmt.Sprintf("%q", col))
	}
	output.WriteString("]\n\n")

	// 输出数据行（数组格式）
	for _, row := range results {
		output.WriteString("[")
		for j, col := range columns {
			if j > 0 {
				output.WriteString(", ")
			}
			output.WriteString(formatClickHouseValue(row[col]))
		}
		output.WriteString("]\n")
	}

	return output.String()
}

// formatClickHouseValue 格式化单个值为紧凑格式
func formatClickHouseValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case []byte:
		return fmt.Sprintf("%q", string(val))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%v", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%q", fmt.Sprintf("%v", val))
	}
}

// formatClickHouseModificationResult 格式化修改操作结果
func formatClickHouseModificationResult(sqlQuery string, rowsAffected int64) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Execute: %s\n", sqlQuery))

	lowerSQL := strings.ToLower(strings.TrimSpace(sqlQuery))
	if strings.HasPrefix(lowerSQL, "insert") {
		output.WriteString("✓ Insert successful")
		if rowsAffected > 0 {
			output.WriteString(fmt.Sprintf(", affected %d rows", rowsAffected))
		}
	} else if strings.HasPrefix(lowerSQL, "alter") {
		output.WriteString("✓ Alter successful")
	} else if strings.HasPrefix(lowerSQL, "create") {
		output.WriteString("✓ Create successful")
	} else if strings.HasPrefix(lowerSQL, "drop") {
		output.WriteString("✓ Drop successful")
	} else if strings.HasPrefix(lowerSQL, "truncate") {
		output.WriteString("✓ Truncate successful")
	} else {
		output.WriteString("✓ Execute successful")
		if rowsAffected > 0 {
			output.WriteString(fmt.Sprintf(", affected %d rows", rowsAffected))
		}
	}

	output.WriteString("\n")
	return output.String()
}