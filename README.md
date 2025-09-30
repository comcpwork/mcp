# MCP Database Tools

[English](README.md) | [中文](README_CN.md)

Execute MySQL, Redis and ClickHouse commands through natural language conversations with AI assistants.

## Installation

### Requirements

- Go 1.21 or higher
- MCP Client (Claude Code, Cursor, Cline, etc.)

### Step 1: Install the Tool

```bash
go install github.com/comcpwork/mcp/cmd/mcp@latest
```

### Step 2: Configure Your MCP Client

#### Claude Code

Add the MCP server:

```bash
claude mcp add database -- mcp database
```

#### Cursor / Cline / Other MCP Clients

Add to your MCP configuration file:

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

Configuration file locations:
- **Claude Desktop (macOS):** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Claude Desktop (Windows):** `%APPDATA%\Claude\claude_desktop_config.json`
- **Cursor:** Settings > Features > MCP Servers
- **Cline (VS Code):** `.vscode/mcp.json` or VS Code settings

### Step 3: Restart Your Client

Restart your MCP client to load the database tools.

### Verification

Ask your AI assistant:
- "Execute MySQL with DSN root:password@tcp(localhost:3306)/test and SQL: SELECT 1"
- "Execute Redis command PING on redis://localhost:6379/0"

## License

MIT License - see [LICENSE](LICENSE) file for details.