package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleSQLiteExec 处理 SQLite 执行请求
func handleSQLiteExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取 DSN 参数（SQLite 的 DSN 就是文件路径或 :memory:）
	dsn, err := req.RequireString("dsn")
	if err != nil {
		return mcp.NewToolResultError("Missing dsn parameter"), nil
	}

	// 获取 SQL 参数
	sqlQuery, err := req.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError("Missing sql parameter"), nil
	}

	// 检查是否需要SSH远程执行
	sshURI := req.GetString("ssh", "")
	if sshURI != "" {
		return handleSQLiteSSHExec(ctx, sshURI, dsn, sqlQuery)
	}

	// 本地模式：打开数据库连接
	db, err := sql.Open("sqlite3", dsn)
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
		return executeSQLiteQuery(ctx, db, sqlQuery)
	}
	return executeSQLiteModification(ctx, db, sqlQuery)
}

// handleSQLiteSSHExec 通过SSH远程执行sqlite3命令
func handleSQLiteSSHExec(ctx context.Context, sshURI, dbPath, sqlQuery string) (*mcp.CallToolResult, error) {
	// 从连接池获取SSH客户端
	client, err := GetSSHPool().GetSSHExecClient(sshURI)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("SSH connection failed: %v", err)), nil
	}
	defer client.Close()

	// 构建sqlite3命令（使用-header -column格式输出）
	// 转义单引号防止命令注入
	escapedPath := strings.ReplaceAll(dbPath, "'", "'\\''")
	escapedSQL := strings.ReplaceAll(sqlQuery, "'", "'\\''")
	cmd := fmt.Sprintf("sqlite3 -header -column '%s' '%s'", escapedPath, escapedSQL)

	// 执行远程命令
	output, err := client.RunWithContext(ctx, cmd)
	if err != nil {
		errStr := err.Error()
		// 检查是否是sqlite3不存在
		if strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "command not found") ||
			strings.Contains(errStr, "No such file or directory") {
			return mcp.NewToolResultError("sqlite3 command not found on remote server"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Remote execution failed: %v", err)), nil
	}

	// 原样返回输出，不做格式化
	if output == "" {
		output = "Query executed successfully (no output)"
	}
	return mcp.NewToolResultText(output), nil
}

// executeSQLiteQuery 执行查询操作
func executeSQLiteQuery(ctx context.Context, db *sql.DB, sqlQuery string) (*mcp.CallToolResult, error) {
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
	output := formatSQLiteQueryResult(columns, results, sqlQuery)
	return mcp.NewToolResultText(output), nil
}

// executeSQLiteModification 执行修改操作
func executeSQLiteModification(ctx context.Context, db *sql.DB, sqlQuery string) (*mcp.CallToolResult, error) {
	result, err := db.ExecContext(ctx, sqlQuery)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Execute failed: %v", err)), nil
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	output := formatSQLiteModificationResult(sqlQuery, rowsAffected, lastInsertId)
	return mcp.NewToolResultText(output), nil
}

// formatSQLiteQueryResult 格式化查询结果
func formatSQLiteQueryResult(columns []string, results []map[string]interface{}, sqlQuery string) string {
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
			output.WriteString(formatSQLiteValue(row[col]))
		}
		output.WriteString("],\n")
	}

	return output.String()
}

// formatSQLiteValue 格式化单个值为紧凑格式
func formatSQLiteValue(v interface{}) string {
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

// formatSQLiteModificationResult 格式化修改操作结果
func formatSQLiteModificationResult(sqlQuery string, rowsAffected, lastInsertId int64) string {
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
