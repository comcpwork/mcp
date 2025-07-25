# MCP Database Tools

[English](README.md) | [中文](README_CN.md)

Manage MySQL, Redis, and Pulsar through natural language conversations with AI assistants.

## Quick Start

```bash
# Install
git clone https://github.com/yourname/mcp.git
cd mcp
make install

# Configure (Claude Desktop)
# Edit: ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "database": {
      "command": "~/.local/bin/mcp",
      "args": ["mysql"]  # or "redis", "pulsar"
    }
  }
}
```

## Basic Usage

Just talk to Claude:
- "Connect to MySQL at localhost:3306"
- "Show all tables"
- "Query the users table"

## Documentation

- [Installation Guide](docs/en/installation.md)
- [MySQL Usage](docs/en/mysql.md)
- [Redis Usage](docs/en/redis.md)
- [Pulsar Usage](docs/en/pulsar.md)
- [Security Options](docs/en/security.md)

## License

MIT License - see [LICENSE](LICENSE) file for details.