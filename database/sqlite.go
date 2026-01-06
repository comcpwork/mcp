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

	// 打开数据库连接
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
