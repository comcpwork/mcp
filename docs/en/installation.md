# Installation Guide

## Requirements

- Go 1.21 or higher
- MCP Client (Claude Code, Cursor, Cline, etc.)

## Installation

### Quick Install

```bash
go install github.com/comcpwork/mcp/cmd/mcp@latest
```

### Verify Installation

```bash
mcp --version
```

## Configuration

### Claude Code

Add the MCP server directly:

```bash
claude mcp add database -- mcp database
```

### Claude Desktop (macOS/Windows)

Edit configuration file:
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

Go to **Settings > Features > MCP Servers** and add:

```json
{
  "database": {
    "command": "mcp",
    "args": ["database"]
  }
}
```

### Cline (VS Code)

Edit `.vscode/mcp.json` or VS Code settings:

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

## Available Tools

After configuration, the following MCP tools will be available:

| Tool | Description |
|------|-------------|
| `mysql_exec` | Execute MySQL SQL statements |
| `redis_exec` | Execute Redis commands |
| `clickhouse_exec` | Execute ClickHouse SQL statements |
| `sqlite_exec` | Execute SQLite SQL statements |

## Troubleshooting

### Command not found

Ensure Go bin directory is in your PATH:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Add this line to your `~/.bashrc` or `~/.zshrc` for persistence.

### Permission denied

```bash
chmod +x ~/go/bin/mcp
```

### MCP client not detecting tools

1. Restart your MCP client after configuration
2. Verify the `mcp` command is accessible from the client's environment
3. Check the configuration file syntax (valid JSON)

## Uninstall

```bash
rm ~/go/bin/mcp
```
