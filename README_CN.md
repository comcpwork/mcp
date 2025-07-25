# MCP 数据库工具

[English](README.md) | [中文](README_CN.md)

通过与 AI 助手的自然语言对话管理 MySQL、Redis 和 Pulsar。

## 快速开始

```bash
# 安装
git clone https://github.com/yourname/mcp.git
cd mcp
make install

# 配置（Claude Desktop）
# 编辑：~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "database": {
      "command": "~/.local/bin/mcp",
      "args": ["mysql"]  # 或 "redis"、"pulsar"
    }
  }
}
```

## 基本使用

直接对 Claude 说：
- "连接到本地的 MySQL 数据库"
- "显示所有表"
- "查询 users 表的数据"

## 文档

- [安装指南](docs/zh/installation.md)
- [MySQL 使用](docs/zh/mysql.md)
- [Redis 使用](docs/zh/redis.md)
- [Pulsar 使用](docs/zh/pulsar.md)
- [安全选项](docs/zh/security.md)

## 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。