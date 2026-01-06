# 安装指南

## 系统要求

- Go 1.21 或更高版本
- MCP 客户端（Claude Code、Cursor、Cline 等）

## 安装

### 快速安装

```bash
go install github.com/comcpwork/mcp/cmd/mcp@latest
```

### 验证安装

```bash
mcp --version
```

## 配置

### Claude Code

直接添加 MCP 服务器：

```bash
claude mcp add database -- mcp database
```

### Claude Desktop（macOS/Windows）

编辑配置文件：
- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

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

### Cursor

进入 **Settings > Features > MCP Servers** 并添加：

```json
{
  "database": {
    "command": "mcp",
    "args": ["database"]
  }
}
```

### Cline (VS Code)

编辑 `.vscode/mcp.json` 或 VS Code 设置：

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

## 可用工具

配置完成后，以下 MCP 工具将可用：

| 工具 | 描述 |
|------|------|
| `mysql_exec` | 执行 MySQL SQL 语句 |
| `redis_exec` | 执行 Redis 命令 |
| `clickhouse_exec` | 执行 ClickHouse SQL 语句 |
| `sqlite_exec` | 执行 SQLite SQL 语句 |

## 故障排除

### 找不到命令

确保 Go bin 目录在 PATH 中：

```bash
export PATH="$HOME/go/bin:$PATH"
```

将此行添加到 `~/.bashrc` 或 `~/.zshrc` 以持久化。

### 权限被拒绝

```bash
chmod +x ~/go/bin/mcp
```

### MCP 客户端未检测到工具

1. 配置后重启 MCP 客户端
2. 确认 `mcp` 命令在客户端环境中可访问
3. 检查配置文件语法（有效的 JSON）

## 卸载

```bash
rm ~/go/bin/mcp
```
