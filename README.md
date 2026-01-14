# MCP Database Tools

[English](README.md) | [中文](README_CN.md)

Execute MySQL, Redis, ClickHouse and SQLite commands through natural language conversations with AI assistants.

## Features

- **MySQL** - Execute SQL queries and modifications
- **Redis** - Execute Redis commands
- **ClickHouse** - Execute ClickHouse SQL statements
- **SQLite** - Execute SQLite SQL (file-based or in-memory)
- **SSH Tunneling** - Connect to databases through SSH bastion hosts

## Installation

### Requirements

- Go 1.21 or higher
- MCP Client (Claude Code, Cursor, Cline, etc.)

### Step 1: Install the Tool

```bash
go install github.com/comcpwork/mcp/cmd/cowork-database@latest
```

### Step 2: Configure Your MCP Client

#### Claude Code

Add the MCP server:

```bash
claude mcp add database -- cowork-database database
```

#### Cursor / Cline / Other MCP Clients

Add to your MCP configuration file:

```json
{
  "mcpServers": {
    "database": {
      "command": "cowork-database",
      "args": ["database"]
    }
  }
}
```

Configuration file locations:
- **Claude Desktop (macOS):** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Claude Desktop (Windows):** `%APPDATA%\Claude\claude_desktop_config.json`
- **Cursor:** Settings > Features > MCP Servers
- **Cline (VS Code):** `.vscode/mcp.json` or VS Code settings

### Step 3: Restart Your Client

Restart your MCP client to load the database tools.

## Quick Start

Ask your AI assistant:

- **MySQL:** "Execute MySQL with DSN `root:password@tcp(localhost:3306)/test` and SQL: `SELECT * FROM users`"
- **Redis:** "Execute Redis command `PING` on `redis://localhost:6379/0`"
- **ClickHouse:** "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/default` and SQL: `SELECT 1`"
- **SQLite:** "Execute SQLite with DSN `:memory:` and SQL: `SELECT 1`"

## DSN Formats

| Database | Format | Example |
|----------|--------|---------|
| MySQL | `user:pass@tcp(host:port)/dbname` | `root:password@tcp(localhost:3306)/mydb` |
| Redis | `redis://[:password@]host:port/db` | `redis://localhost:6379/0` |
| ClickHouse | `clickhouse://user:pass@host:port/db` | `clickhouse://default:@localhost:9000/default` |
| SQLite | `/path/to/file.db` or `:memory:` | `/data/mydb.db` or `:memory:` |

## SSH Connection

All tools support optional SSH tunneling through the `ssh` parameter.

### SSH URI Formats

| Format | Example | Description |
|--------|---------|-------------|
| Config reference | `ssh://myserver` | Use `~/.ssh/config` entry |
| Password auth | `ssh://user:pass@host:port` | Direct password authentication |
| Key auth | `ssh://user@host?key=/path/to/key` | Private key authentication |

### Example

```json
{
  "dsn": "root:password@tcp(10.0.0.100:3306)/mydb",
  "sql": "SELECT * FROM users",
  "ssh": "ssh://admin@jump.example.com?key=~/.ssh/id_rsa"
}
```

> **Note:** SQLite uses remote command execution mode (requires `sqlite3` on remote server).

## Documentation

- [Installation Guide](docs/en/installation.md)
- [MySQL Guide](docs/en/mysql.md)
- [Redis Guide](docs/en/redis.md)
- [ClickHouse Guide](docs/en/clickhouse.md)
- [SQLite Guide](docs/en/sqlite.md)

## License

MIT License - see [LICENSE](LICENSE) file for details.
