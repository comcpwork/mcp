package database

import (
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// NewServer 创建数据库 MCP 服务器
func NewServer() *mcpserver.MCPServer {
	server := mcpserver.NewMCPServer(
		"Database MCP Server",
		"1.0.4",
		mcpserver.WithToolCapabilities(true),
	)

	// SSH参数描述（供复用）
	sshDescription := "SSH tunnel URI for connecting through bastion host. " +
		"Format 1: ssh://config-name (use ~/.ssh/config). " +
		"Format 2: ssh://user[:password]@host[:port][?key=/path/to/key&passphrase=xxx]. " +
		"Example: ssh://myserver or ssh://admin@jump.example.com?key=~/.ssh/id_rsa"

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
			mcp.WithString("ssh",
				mcp.Description(sshDescription),
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
			mcp.WithString("ssh",
				mcp.Description(sshDescription),
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
			mcp.WithString("ssh",
				mcp.Description(sshDescription),
			),
		),
		handleClickHouseExec,
	)

	// SQLite SSH参数描述（特殊说明远程命令执行模式）
	sqliteSSHDescription := "SSH connection for remote sqlite3 command execution. " +
		"When SSH is provided, the tool executes sqlite3 command on remote server (requires sqlite3 installed). " +
		"Format 1: ssh://config-name (use ~/.ssh/config). " +
		"Format 2: ssh://user[:password]@host[:port][?key=/path/to/key&passphrase=xxx]. " +
		"Example: ssh://myserver or ssh://admin@server.example.com?key=~/.ssh/id_rsa"

	// 注册 SQLite 工具
	server.AddTool(
		mcp.NewTool("sqlite_exec",
			mcp.WithDescription(
				"Execute SQLite SQL statements. "+
					"Local mode: uses SQLite driver directly. "+
					"SSH mode: executes sqlite3 command on remote server (requires sqlite3 installed). "+
					"Note: If no LIMIT is specified in your SELECT query, all matching rows will be returned.",
			),
			mcp.WithString("dsn",
				mcp.Required(),
				mcp.Description(
					"SQLite database file path or :memory: for in-memory database. "+
						"For local: /path/to/database.db or :memory:. "+
						"For SSH: remote file path like /data/mydb.db. "+
						"Example: /Users/<username>/data/mydb.db or :memory:",
				),
			),
			mcp.WithString("sql",
				mcp.Required(),
				mcp.Description(
					"SQL statement to execute. "+
						"For SELECT queries without LIMIT clause, all rows will be returned.",
				),
			),
			mcp.WithString("ssh",
				mcp.Description(sqliteSSHDescription),
			),
		),
		handleSQLiteExec,
	)

	// 注册 Prometheus 工具
	server.AddTool(
		mcp.NewTool("prometheus_exec",
			mcp.WithDescription(
				"Query Prometheus metrics using PromQL or built-in commands. "+
					"Built-in commands (support match filter): "+
					"SHOW METRICS - list all metric names; "+
					"SHOW LABELS - list all label names; "+
					"SHOW LABEL VALUES <label> - get all values for a label; "+
					"DESCRIBE <metric> - get metric metadata (type, help). "+
					"PromQL queries are also supported for instant and range queries (no match filter). "+
					"Match filter (built-in commands only): append '| {selector}' to filter by PromQL selector. "+
					"Example: 'SHOW METRICS | {job=\"prometheus\"}'.",
			),
			mcp.WithString("dsn",
				mcp.Required(),
				mcp.Description(
					"Prometheus server address. "+
						"Format: prometheus://[user:pass@]host:port. "+
						"Example: prometheus://localhost:9090 or prometheus://admin:secret@prometheus.example.com:9090",
				),
			),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description(
					"Query expression: built-in command or PromQL. "+
						"Built-in commands (support '| {selector}' match filter): SHOW METRICS, SHOW LABELS, SHOW LABEL VALUES <label>, DESCRIBE <metric>. "+
						"PromQL (no match filter): any valid PromQL like 'up', 'rate(http_requests_total[5m])'. "+
						"Examples: 'SHOW METRICS | {job=\"prometheus\"}', 'SHOW LABEL VALUES job', 'up'.",
				),
			),
			mcp.WithString("start",
				mcp.Description(
					"Range query start time in RFC3339 format. "+
						"Required for range queries along with 'end' and 'step'. "+
						"Example: 2024-01-14T09:00:00Z",
				),
			),
			mcp.WithString("end",
				mcp.Description(
					"Range query end time in RFC3339 format. "+
						"Required for range queries along with 'start' and 'step'. "+
						"Example: 2024-01-14T10:00:00Z",
				),
			),
			mcp.WithString("step",
				mcp.Description(
					"Range query step duration. "+
						"Required for range queries along with 'start' and 'end'. "+
						"Example: 1m, 5m, 1h",
				),
			),
			mcp.WithString("ssh",
				mcp.Description(sshDescription),
			),
		),
		handlePrometheusExec,
	)

	return server
}