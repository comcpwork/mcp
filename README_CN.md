# MCP 数据库工具

[English](README.md) | [中文](README_CN.md)

通过与 AI 助手的自然语言对话执行 MySQL、Redis、ClickHouse 和 SQLite 命令。

## 安装

### 系统要求

- Go 1.21 或更高版本
- MCP 客户端（Claude Code、Cursor、Cline 等）

### 步骤 1：安装工具

```bash
go install github.com/comcpwork/mcp/cmd/mcp@latest
```

### 步骤 2：配置 MCP 客户端

#### Claude Code

添加 MCP 服务器：

```bash
claude mcp add database -- mcp database
```

#### Cursor / Cline / 其他 MCP 客户端

添加到 MCP 配置文件：

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

配置文件位置：
- **Claude Desktop (macOS):** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Claude Desktop (Windows):** `%APPDATA%\Claude\claude_desktop_config.json`
- **Cursor:** Settings > Features > MCP Servers
- **Cline (VS Code):** `.vscode/mcp.json` 或 VS Code 设置

### 步骤 3：重启客户端

重启你的 MCP 客户端以加载数据库工具。

### 验证

向你的 AI 助手提问：
- "使用 DSN root:password@tcp(localhost:3306)/test 执行 MySQL: SELECT 1"
- "在 redis://localhost:6379/0 上执行 Redis 命令 PING"
- "使用 DSN :memory: 执行 SQLite: SELECT 1"

## 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。