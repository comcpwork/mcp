package database

import (
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// NewServer 创建数据库 MCP 服务器
func NewServer() *mcpserver.MCPServer {
	server := mcpserver.NewMCPServer(
		"Database MCP Server",
		"1.0.3",
		mcpserver.WithToolCapabilities(true),
	)

	// 注册 MySQL 工具
	server.AddTool(
		mcp.NewTool("mysql_exec",
			mcp.WithDescription(
				"Execute MySQL SQL statements using Go database/sql driver DSN. "+
					"Supports SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, etc. "+
					"Note: If no LIMIT is specified in your SELECT query, all matching rows will be returned.",
			),
			mcp.WithString("dsn",
				mcp.Required(),
				mcp.Description(
					"Go MySQL driver DSN string. "+
						"Format: username:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true. "+
						"Example: root:password@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=true",
				),
			),
			mcp.WithString("sql",
				mcp.Required(),
				mcp.Description(
					"SQL statement to execute. "+
						"For SELECT queries without LIMIT clause, all rows will be returned.",
				),
			),
		),
		handleMySQLExec,
	)

	// 注册 Redis 工具
	server.AddTool(
		mcp.NewTool("redis_exec",
			mcp.WithDescription(
				"Execute Redis commands using Redis connection string. "+
					"Supports all Redis commands like GET, SET, HGET, LPUSH, etc. "+
					"For commands that return multiple values, all values will be returned.",
			),
			mcp.WithString("dsn",
				mcp.Required(),
				mcp.Description(
					"Redis connection string. "+
						"Format: redis://username:password@host:port/database or redis://host:port/database. "+
						"Example: redis://localhost:6379/0 or redis://:password@localhost:6379/0",
				),
			),
			mcp.WithString("command",
				mcp.Required(),
				mcp.Description(
					"Redis command to execute. "+
						"Example: GET key, SET key value, HGETALL myhash, LPUSH mylist value",
				),
			),
		),
		handleRedisExec,
	)

	// 注册 ClickHouse 工具
	server.AddTool(
		mcp.NewTool("clickhouse_exec",
			mcp.WithDescription(
				"Execute ClickHouse SQL statements using ClickHouse driver DSN. "+
					"Supports SELECT, INSERT, CREATE, DROP, ALTER, etc. "+
					"Note: If no LIMIT is specified in your SELECT query, all matching rows will be returned.",
			),
			mcp.WithString("dsn",
				mcp.Required(),
				mcp.Description(
					"ClickHouse driver DSN string. "+
						"Format: clickhouse://username:password@host:port/database?options. "+
						"Example: clickhouse://default:password@localhost:9000/mydb?dial_timeout=10s&read_timeout=20s",
				),
			),
			mcp.WithString("sql",
				mcp.Required(),
				mcp.Description(
					"SQL statement to execute. "+
						"For SELECT queries without LIMIT clause, all rows will be returned.",
				),
			),
		),
		handleClickHouseExec,
	)

	return server
}