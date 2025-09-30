# MCP Database Tools

[English](README.md) | [中文](README_CN.md)

Execute MySQL, Redis and ClickHouse commands through natural language conversations with AI assistants.

## Quick Start

### Installation

**Option 1: One-line Install (Recommended)**

**Linux/macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/comcpwork/mcp/main/scripts/install.sh | bash
```

**Windows (PowerShell as Administrator):**
```powershell
iwr -useb https://raw.githubusercontent.com/comcpwork/mcp/main/scripts/install.ps1 | iex
```

**Option 2: From Source**

```bash
git clone https://github.com/comcpwork/mcp.git
cd mcp
make install
```

### Configuration

Edit your Claude Desktop config file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "database": {
      "command": "mcp",
      "args": ["database"]
    }
  }
}
```

### Usage

Just talk to Claude:

**MySQL Examples:**
- "Execute MySQL with DSN root:password@tcp(localhost:3306)/mydb and SQL: SELECT * FROM users"
- "Run MySQL query: SELECT COUNT(*) FROM orders WHERE status='completed' using DSN root:pass@tcp(localhost:3306)/shop"
- "Create a new table in MySQL: CREATE TABLE products (id INT PRIMARY KEY, name VARCHAR(100))"

**Redis Examples:**
- "Execute Redis command GET user:123 on redis://localhost:6379/0"
- "Set a Redis key: SET session:abc xyz using DSN redis://localhost:6379/0"
- "Get all hash fields: HGETALL user:profile:456 from redis://localhost:6379/0"

**ClickHouse Examples:**
- "Execute ClickHouse with DSN clickhouse://default:password@localhost:9000/mydb and SQL: SELECT * FROM events"
- "Run ClickHouse query: SELECT count() FROM logs WHERE date >= today() using DSN clickhouse://default:@localhost:9000/analytics"
- "Create a ClickHouse table: CREATE TABLE events (timestamp DateTime, user_id UInt64, action String) ENGINE = MergeTree() ORDER BY timestamp"

## Tools

### mysql_exec

Execute MySQL SQL statements using Go database/sql driver DSN.

**Parameters:**
- `dsn` (required): MySQL connection string
  - Format: `username:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true`
  - Example: `root:password@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=true`
- `sql` (required): SQL statement to execute
  - Supports: SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, etc.

### redis_exec

Execute Redis commands using Redis connection string.

**Parameters:**
- `dsn` (required): Redis connection string
  - Format: `redis://[username:password@]host:port/database`
  - Examples:
    - `redis://localhost:6379/0`
    - `redis://:password@localhost:6379/0`
    - `redis://user:pass@localhost:6379/1`
- `command` (required): Redis command to execute
  - Examples: `GET key`, `SET key value`, `HGETALL myhash`, `LPUSH mylist value`

### clickhouse_exec

Execute ClickHouse SQL statements using ClickHouse driver DSN.

**Parameters:**
- `dsn` (required): ClickHouse connection string
  - Format: `clickhouse://username:password@host:port/database?options`
  - Examples:
    - `clickhouse://default:@localhost:9000/mydb`
    - `clickhouse://default:password@localhost:9000/analytics?dial_timeout=10s&read_timeout=20s`
    - `clickhouse://user:pass@host1:9000,host2:9000/cluster_db?connection_open_strategy=round_robin`
- `sql` (required): SQL statement to execute
  - Supports: SELECT, INSERT, CREATE, DROP, ALTER, etc.

## Features

- ✅ **No Configuration Files** - Pass DSN directly, no local storage needed
- ✅ **Stateless Design** - Every execution creates a new connection
- ✅ **Simple & Clean** - Only ~650 lines of core code
- ✅ **Full SQL Support** - Execute any MySQL and ClickHouse statement
- ✅ **Full Redis Support** - Execute any Redis command
- ✅ **ClickHouse Support** - Native protocol with advanced features (compression, load balancing, etc.)

## Architecture

```
mcp/
├── cmd/mcp/
│   └── main.go          # Entry point (~40 lines)
├── database/
│   ├── server.go        # MCP server & tool registration (~95 lines)
│   ├── mysql.go         # MySQL execution logic (~220 lines)
│   ├── redis.go         # Redis execution logic (~120 lines)
│   └── clickhouse.go    # ClickHouse execution logic (~220 lines)
└── go.mod
```

**Total:** ~650 lines of code (excluding comments and blank lines)

## Requirements

- Go 1.21 or higher (for building from source)
- MySQL 5.7+ or MariaDB 10.2+ (for MySQL operations)
- Redis 5.0+ (for Redis operations)
- ClickHouse 20.3+ (for ClickHouse operations)

## Development

```bash
# Build
make build

# Install to user directory
make install

# Run tests
make test

# Clean build files
make clean
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Links

- [GitHub Repository](https://github.com/comcpwork/mcp)
- [Report Issues](https://github.com/comcpwork/mcp/issues)
- [Model Context Protocol](https://modelcontextprotocol.io/)