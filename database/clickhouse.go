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
	output.WriteString(fmt.Sprintf("Result: %d rows x %d columns\n\n", len(results), len(columns)))

	if len(results) == 0 {
		output.WriteString("No data\n")
		return output.String()
	}

	// 表格形式显示（最多5列）
	if len(columns) <= 5 {
		// 计算列宽
		colWidths := make([]int, len(columns))
		for i, col := range columns {
			colWidths[i] = len(col)
		}

		for _, row := range results {
			for j, col := range columns {
				val := fmt.Sprintf("%v", row[col])
				if len(val) > colWidths[j] {
					colWidths[j] = len(val)
				}
			}
		}

		// 表头
		for i, col := range columns {
			output.WriteString(fmt.Sprintf("%-*s", colWidths[i]+2, col))
		}
		output.WriteString("\n")

		// 分隔线
		for i := range columns {
			output.WriteString(strings.Repeat("-", colWidths[i]+2))
		}
		output.WriteString("\n")

		// 数据行
		for _, row := range results {
			for j, col := range columns {
				val := fmt.Sprintf("%v", row[col])
				output.WriteString(fmt.Sprintf("%-*s", colWidths[j]+2, val))
			}
			output.WriteString("\n")
		}
	} else {
		// 列数较多时使用键值对形式
		for i, row := range results {
			output.WriteString(fmt.Sprintf("--- Row %d ---\n", i+1))
			for _, col := range columns {
				output.WriteString(fmt.Sprintf("%s: %v\n", col, row[col]))
			}
			output.WriteString("\n")
		}
	}

	return output.String()
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