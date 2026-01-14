# MCP 数据库工具

[English](README.md) | [中文](README_CN.md)

通过与 AI 助手的自然语言对话执行 MySQL、Redis、ClickHouse 和 SQLite 命令。

## 功能特性

- **MySQL** - 执行 SQL 查询和修改操作
- **Redis** - 执行 Redis 命令
- **ClickHouse** - 执行 ClickHouse SQL 语句
- **SQLite** - 执行 SQLite SQL（文件数据库或内存数据库）
- **SSH 隧道** - 通过 SSH 跳板机连接数据库

## 安装

### 系统要求

- Go 1.21 或更高版本
- MCP 客户端（Claude Code、Cursor、Cline 等）

### 步骤 1：安装工具

```bash
go install github.com/comcpwork/mcp/cmd/cowork-database@latest
```

### 步骤 2：配置 MCP 客户端

#### Claude Code

添加 MCP 服务器：

```bash
claude mcp add database -- cowork-database database
```

#### Cursor / Cline / 其他 MCP 客户端

添加到 MCP 配置文件：

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

配置文件位置：
- **Claude Desktop (macOS):** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Claude Desktop (Windows):** `%APPDATA%\Claude\claude_desktop_config.json`
- **Cursor:** Settings > Features > MCP Servers
- **Cline (VS Code):** `.vscode/mcp.json` 或 VS Code 设置

### 步骤 3：重启客户端

重启你的 MCP 客户端以加载数据库工具。

## 快速开始

向你的 AI 助手提问：

- **MySQL:** "使用 DSN `root:password@tcp(localhost:3306)/test` 执行 MySQL: `SELECT * FROM users`"
- **Redis:** "在 `redis://localhost:6379/0` 上执行 Redis 命令 `PING`"
- **ClickHouse:** "使用 DSN `clickhouse://default:@localhost:9000/default` 执行 ClickHouse: `SELECT 1`"
- **SQLite:** "使用 DSN `:memory:` 执行 SQLite: `SELECT 1`"

## DSN 格式

| 数据库 | 格式 | 示例 |
|--------|------|------|
| MySQL | `user:pass@tcp(host:port)/dbname` | `root:password@tcp(localhost:3306)/mydb` |
| Redis | `redis://[:password@]host:port/db` | `redis://localhost:6379/0` |
| ClickHouse | `clickhouse://user:pass@host:port/db` | `clickhouse://default:@localhost:9000/default` |
| SQLite | `/path/to/file.db` 或 `:memory:` | `/data/mydb.db` 或 `:memory:` |

## SSH 连接

所有工具都支持通过 `ssh` 参数进行可选的 SSH 隧道连接。

### SSH URI 格式

| 格式 | 示例 | 说明 |
|------|------|------|
| 配置引用 | `ssh://myserver` | 使用 `~/.ssh/config` 中的配置 |
| 密码认证 | `ssh://user:pass@host:port` | 直接密码认证 |
| 密钥认证 | `ssh://user@host?key=/path/to/key` | 私钥认证 |

### 示例

```json
{
  "dsn": "root:password@tcp(10.0.0.100:3306)/mydb",
  "sql": "SELECT * FROM users",
  "ssh": "ssh://admin@jump.example.com?key=~/.ssh/id_rsa"
}
```

> **注意：** SQLite 使用远程命令执行模式（需要远程服务器安装 `sqlite3`）。

## 文档

- [安装指南](docs/zh/installation.md)
- [MySQL 指南](docs/zh/mysql.md)
- [Redis 指南](docs/zh/redis.md)
- [ClickHouse 指南](docs/zh/clickhouse.md)
- [SQLite 指南](docs/zh/sqlite.md)

## 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。
