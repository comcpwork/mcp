# Quick Install Commands

## Linux/macOS

```bash
curl -fsSL https://raw.githubusercontent.com/comcpwork/mcp/main/scripts/install.sh | bash
```

Or with wget:
```bash
wget -qO- https://raw.githubusercontent.com/comcpwork/mcp/main/scripts/install.sh | bash
```

## Windows (PowerShell as Administrator)

```powershell
iwr -useb https://raw.githubusercontent.com/comcpwork/mcp/main/scripts/install.ps1 | iex
```

Or download and run:
```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/comcpwork/mcp/main/scripts/install.ps1" -OutFile "install.ps1"
.\install.ps1
```

## Manual Installation

1. Download from [Releases](https://github.com/comcpwork/mcp/releases/latest)
2. Extract and move to PATH
3. Configure Claude Desktop