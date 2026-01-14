#!/bin/bash
set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# 默认版本
VERSION=${1:-v0.1.0}

echo -e "${BLUE}=== GitHub CLI Release 工具 ===${NC}"

# 检查 gh 是否安装
if ! command -v gh &> /dev/null; then
    echo -e "${RED}错误: GitHub CLI (gh) 未安装${NC}"
    echo "请访问 https://cli.github.com 安装"
    exit 1
fi

# 检查是否已登录
if ! gh auth status &> /dev/null; then
    echo -e "${YELLOW}需要登录 GitHub${NC}"
    gh auth login
fi

echo -e "版本: ${GREEN}$VERSION${NC}"
echo ""

# 1. 构建二进制文件
echo -e "${YELLOW}步骤 1: 构建二进制文件${NC}"
make build-all

# 2. 创建标签
echo -e "\n${YELLOW}步骤 2: 创建 Git 标签${NC}"
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo "标签 $VERSION 已存在"
else
    git tag -a $VERSION -m "Release $VERSION"
    git push origin $VERSION
    echo "✓ 标签已创建并推送"
fi

# 3. 创建 Release
echo -e "\n${YELLOW}步骤 3: 创建 GitHub Release${NC}"

# 生成发布说明
NOTES=$(cat <<EOF
## 🎉 MCP $VERSION

### ✨ Features
- Multi-database support (MySQL, Redis, Pulsar)
- Natural language interface for AI assistants
- Granular security controls with command-line flags
- SQL parser for accurate permission validation

### 📦 安装

\`\`\`bash
# Linux/macOS
chmod +x cowork-database-*
sudo mv cowork-database-* /usr/local/bin/cowork-database

# 验证
cowork-database --version
\`\`\`

### 📚 文档
- [安装指南](https://github.com/comcpwork/mcp/blob/main/docs/zh/installation.md)
- [使用文档](https://github.com/comcpwork/mcp/tree/main/docs)
EOF
)

# 创建 release 并上传文件
echo "创建 Release..."
gh release create $VERSION \
    --title "MCP $VERSION" \
    --notes "$NOTES" \
    dist/cowork-database-linux-amd64 \
    dist/cowork-database-linux-arm64 \
    dist/cowork-database-darwin-amd64 \
    dist/cowork-database-darwin-arm64 \
    dist/cowork-database-windows-amd64.exe

echo -e "\n${GREEN}✓ Release 发布成功！${NC}"
echo -e "查看: https://github.com/comcpwork/mcp/releases/tag/$VERSION"