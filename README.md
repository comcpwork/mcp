# MCP Database Tools

[English](README.md) | [中文](README_CN.md)

Manage MySQL, Redis, and Pulsar through natural language conversations with AI assistants.

## Quick Start

### Option 1: One-line Install (Recommended)

**Linux/macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/comcpwork/mcp/main/scripts/install.sh | bash
```

**Windows (PowerShell as Administrator):**
```powershell
iwr -useb https://raw.githubusercontent.com/comcpwork/mcp/main/scripts/install.ps1 | iex
```

### Option 2: Manual Download

Download the latest release from [GitHub Releases](https://github.com/comcpwork/mcp/releases)

```bash
# Linux/macOS
chmod +x mcp-*
sudo mv mcp-* /usr/local/bin/mcp

# Verify
mcp --version
```

### Option 3: Build from Source

```bash
git clone https://github.com/comcpwork/mcp.git
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