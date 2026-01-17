package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleMySQLExec 处理 MySQL 执行请求
func handleMySQLExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		remoteHost, remotePort, err := ExtractMySQLHostPort(dsn)
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
		dsn = ReplaceMySQLDSNHostPort(dsn, tunnel.LocalAddr())
	}

	// 打开数据库连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Database connection failed: %v", err)), nil
	}
	defer db.Close()

	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Database connection failed: %v", err)), nil
	}

	// 判断是查询还是执行操作
	if isQueryStatement(sqlQuery) {
		return executeMySQLQuery(ctx, db, sqlQuery)
	}
	return executeMySQLModification(ctx, db, sqlQuery)
}

// isQueryStatement 判断是否为查询语句
func isQueryStatement(sqlQuery string) bool {
	lowerSQL := strings.ToLower(strings.TrimSpace(sqlQuery))
	return strings.HasPrefix(lowerSQL, "select") ||
		strings.HasPrefix(lowerSQL, "show") ||
		strings.HasPrefix(lowerSQL, "describe") ||
		strings.HasPrefix(lowerSQL, "desc") ||
		strings.HasPrefix(lowerSQL, "explain")
}

// executeMySQLQuery 执行查询操作
func executeMySQLQuery(ctx context.Context, db *sql.DB, sqlQuery string) (*mcp.CallToolResult, error) {
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
	output := formatMySQLQueryResult(columns, results, sqlQuery)
	return mcp.NewToolResultText(output), nil
}

// executeMySQLModification 执行修改操作
func executeMySQLModification(ctx context.Context, db *sql.DB, sqlQuery string) (*mcp.CallToolResult, error) {
	result, err := db.ExecContext(ctx, sqlQuery)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Execute failed: %v", err)), nil
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	output := formatMySQLModificationResult(sqlQuery, rowsAffected, lastInsertId)
	return mcp.NewToolResultText(output), nil
}

// formatMySQLQueryResult 格式化查询结果
func formatMySQLQueryResult(columns []string, results []map[string]interface{}, sqlQuery string) string {
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
	output.WriteString("]\n")

	// 输出数据行（数组格式，每行带逗号）
	for _, row := range results {
		output.WriteString("[")
		for j, col := range columns {
			if j > 0 {
				output.WriteString(", ")
			}
			output.WriteString(formatValue(row[col]))
		}
		output.WriteString("],\n")
	}

	return output.String()
}

// formatValue 格式化单个值为紧凑格式
func formatValue(v interface{}) string {
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

// formatMySQLModificationResult 格式化修改操作结果
func formatMySQLModificationResult(sqlQuery string, rowsAffected, lastInsertId int64) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Execute: %s\n", sqlQuery))

	lowerSQL := strings.ToLower(strings.TrimSpace(sqlQuery))
	if strings.HasPrefix(lowerSQL, "insert") {
		output.WriteString("✓ Insert successful")
		if rowsAffected > 0 {
			output.WriteString(fmt.Sprintf(", affected %d rows", rowsAffected))
		}
		if lastInsertId > 0 {
			output.WriteString(fmt.Sprintf(", new ID: %d", lastInsertId))
		}
	} else if strings.HasPrefix(lowerSQL, "update") {
		output.WriteString("✓ Update successful")
		if rowsAffected > 0 {
			output.WriteString(fmt.Sprintf(", affected %d rows", rowsAffected))
		}
	} else if strings.HasPrefix(lowerSQL, "delete") {
		output.WriteString("✓ Delete successful")
		if rowsAffected > 0 {
			output.WriteString(fmt.Sprintf(", deleted %d rows", rowsAffected))
		}
	} else {
		output.WriteString("✓ Execute successful")
		if rowsAffected > 0 {
			output.WriteString(fmt.Sprintf(", affected %d rows", rowsAffected))
		}
	}

	output.WriteString("\n")
	return output.String()
}