# MCP 数据库工具

[English](README.md) | [中文](README_CN.md)

通过与 AI 助手的自然语言对话执行 MySQL、Redis 和 ClickHouse 命令。

## 快速开始

### 安装

**方式一：一键安装（推荐）**

**Linux/macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/comcpwork/mcp/main/scripts/install.sh | bash
```

**Windows（以管理员身份运行 PowerShell）:**
```powershell
iwr -useb https://raw.githubusercontent.com/comcpwork/mcp/main/scripts/install.ps1 | iex
```

**方式二：从源码构建**

```bash
git clone https://github.com/comcpwork/mcp.git
cd mcp
make install
```

### 配置

编辑 Claude Desktop 配置文件：

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

### 使用

直接对 Claude 说：

**MySQL 示例：**
- "使用 DSN root:password@tcp(localhost:3306)/mydb 执行 SQL: SELECT * FROM users"
- "运行 MySQL 查询：SELECT COUNT(*) FROM orders WHERE status='completed'，DSN 是 root:pass@tcp(localhost:3306)/shop"
- "在 MySQL 中创建新表：CREATE TABLE products (id INT PRIMARY KEY, name VARCHAR(100))"

**Redis 示例：**
- "在 redis://localhost:6379/0 上执行 Redis 命令 GET user:123"
- "设置 Redis 键：使用 DSN redis://localhost:6379/0 执行 SET session:abc xyz"
- "获取所有哈希字段：从 redis://localhost:6379/0 执行 HGETALL user:profile:456"

**ClickHouse 示例：**
- "使用 DSN clickhouse://default:password@localhost:9000/mydb 执行 SQL: SELECT * FROM events"
- "运行 ClickHouse 查询：SELECT count() FROM logs WHERE date >= today()，DSN 是 clickhouse://default:@localhost:9000/analytics"
- "创建 ClickHouse 表：CREATE TABLE events (timestamp DateTime, user_id UInt64, action String) ENGINE = MergeTree() ORDER BY timestamp"

## 工具

### mysql_exec

使用 Go database/sql 驱动 DSN 执行 MySQL SQL 语句。

**参数：**
- `dsn`（必需）：MySQL 连接字符串
  - 格式：`username:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true`
  - 示例：`root:password@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=true`
- `sql`（必需）：要执行的 SQL 语句
  - 支持：SELECT、INSERT、UPDATE、DELETE、CREATE、DROP 等

### redis_exec

使用 Redis 连接字符串执行 Redis 命令。

**参数：**
- `dsn`（必需）：Redis 连接字符串
  - 格式：`redis://[username:password@]host:port/database`
  - 示例：
    - `redis://localhost:6379/0`
    - `redis://:password@localhost:6379/0`
    - `redis://user:pass@localhost:6379/1`
- `command`（必需）：要执行的 Redis 命令
  - 示例：`GET key`、`SET key value`、`HGETALL myhash`、`LPUSH mylist value`

### clickhouse_exec

使用 ClickHouse 驱动 DSN 执行 ClickHouse SQL 语句。

**参数：**
- `dsn`（必需）：ClickHouse 连接字符串
  - 格式：`clickhouse://username:password@host:port/database?options`
  - 示例：
    - `clickhouse://default:@localhost:9000/mydb`
    - `clickhouse://default:password@localhost:9000/analytics?dial_timeout=10s&read_timeout=20s`
    - `clickhouse://user:pass@host1:9000,host2:9000/cluster_db?connection_open_strategy=round_robin`
- `sql`（必需）：要执行的 SQL 语句
  - 支持：SELECT、INSERT、CREATE、DROP、ALTER 等

## 特性

- ✅ **无配置文件** - 直接传递 DSN，无需本地存储
- ✅ **无状态设计** - 每次执行创建新连接
- ✅ **简洁清晰** - 核心代码仅约 650 行
- ✅ **完整 SQL 支持** - 执行任何 MySQL 和 ClickHouse 语句
- ✅ **完整 Redis 支持** - 执行任何 Redis 命令
- ✅ **ClickHouse 支持** - 原生协议，支持高级特性（压缩、负载均衡等）

## 架构

```
mcp/
├── cmd/mcp/
│   └── main.go          # 入口（约 40 行）
├── database/
│   ├── server.go        # MCP 服务器和工具注册（约 95 行）
│   ├── mysql.go         # MySQL 执行逻辑（约 220 行）
│   ├── redis.go         # Redis 执行逻辑（约 120 行）
│   └── clickhouse.go    # ClickHouse 执行逻辑（约 220 行）
└── go.mod
```

**总计：** 约 650 行代码（不含注释和空行）

## 系统要求

- Go 1.21 或更高版本（从源码构建时需要）
- MySQL 5.7+ 或 MariaDB 10.2+（用于 MySQL 操作）
- Redis 5.0+（用于 Redis 操作）
- ClickHouse 20.3+（用于 ClickHouse 操作）

## 开发

```bash
# 构建
make build

# 安装到用户目录
make install

# 运行测试
make test

# 清理构建文件
make clean
```

## 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 贡献

欢迎贡献！请随时提交 Pull Request。

## 链接

- [GitHub 仓库](https://github.com/comcpwork/mcp)
- [报告问题](https://github.com/comcpwork/mcp/issues)
- [Model Context Protocol](https://modelcontextprotocol.io/)