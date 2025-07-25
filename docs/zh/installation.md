# 安装指南

## 系统要求

- Go 1.19 或更高版本
- Git

## 安装方法

### 方法一：从源码安装（推荐）

```bash
# 克隆仓库
git clone https://github.com/yourname/mcp.git
cd mcp

# 构建并安装
make install

# 验证安装
mcp --version
```

### 方法二：系统级安装

```bash
# 需要 sudo 权限
make install-system
```

## 配置

### Claude Desktop（macOS/Windows）

编辑配置文件：
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

### 其他 AI 助手

请参考您的 AI 助手的 MCP 配置文档。

## 故障排除

### 找不到命令

添加到 PATH：
```bash
export PATH="$HOME/.local/bin:$PATH"
```

### 权限被拒绝

```bash
chmod +x ~/.local/bin/mcp
```

## 卸载

```bash
make uninstall
```