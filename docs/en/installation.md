# Installation Guide

## Requirements

- Go 1.19 or higher
- Git

## Installation Methods

### Method 1: From Source (Recommended)

```bash
# Clone repository
git clone https://github.com/yourname/mcp.git
cd mcp

# Build and install
make install

# Verify installation
mcp --version
```

### Method 2: System-wide Installation

```bash
# Requires sudo
make install-system
```

## Configuration

### Claude Desktop (macOS/Windows)

Edit configuration file:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "mysql": {
      "command": "~/.local/bin/mcp",
      "args": ["mysql"]
    },
    "redis": {
      "command": "~/.local/bin/mcp",
      "args": ["redis"]
    },
    "pulsar": {
      "command": "~/.local/bin/mcp",
      "args": ["pulsar"]
    }
  }
}
```

### Other AI Assistants

Refer to your AI assistant's MCP configuration documentation.

## Troubleshooting

### Command not found

Add to your PATH:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

### Permission denied

```bash
chmod +x ~/.local/bin/mcp
```

## Uninstall

```bash
make uninstall
```