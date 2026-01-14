#!/bin/bash
set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
REPO="comcpwork/mcp"
INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="cowork-database"

# 检测系统信息
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# 转换架构名称
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}错误: 不支持的架构 $ARCH${NC}"
        exit 1
        ;;
esac

# 转换系统名称
case $OS in
    darwin)
        PLATFORM="darwin"
        ;;
    linux)
        PLATFORM="linux"
        ;;
    mingw*|msys*|cygwin*|windows*)
        PLATFORM="windows"
        echo -e "${RED}请使用 Windows 安装方法${NC}"
        exit 1
        ;;
    *)
        echo -e "${RED}错误: 不支持的操作系统 $OS${NC}"
        exit 1
        ;;
esac

echo -e "${BLUE}=== MCP 自动安装脚本 ===${NC}"
echo -e "系统: ${GREEN}$PLATFORM${NC}"
echo -e "架构: ${GREEN}$ARCH${NC}"
echo ""

# 获取最新版本
echo -e "${YELLOW}获取最新版本信息...${NC}"
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}错误: 无法获取最新版本${NC}"
    exit 1
fi

echo -e "最新版本: ${GREEN}$LATEST_VERSION${NC}"

# 构建下载URL
BINARY_FILE="${BINARY_NAME}-${PLATFORM}-${ARCH}"
if [ "$PLATFORM" == "windows" ]; then
    BINARY_FILE="${BINARY_FILE}.exe"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/$BINARY_FILE"

# 创建临时目录
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# 下载二进制文件
echo -e "\n${YELLOW}下载 MCP...${NC}"
echo "URL: $DOWNLOAD_URL"

cd $TEMP_DIR
if command -v wget > /dev/null; then
    wget -q --show-progress "$DOWNLOAD_URL" -O "$BINARY_NAME"
elif command -v curl > /dev/null; then
    curl -L --progress-bar "$DOWNLOAD_URL" -o "$BINARY_NAME"
else
    echo -e "${RED}错误: 需要 wget 或 curl${NC}"
    exit 1
fi

# 检查下载是否成功
if [ ! -f "$BINARY_NAME" ]; then
    echo -e "${RED}错误: 下载失败${NC}"
    exit 1
fi

# 设置执行权限
chmod +x "$BINARY_NAME"

# 创建安装目录（如果不存在）
if [ ! -d "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}创建安装目录 $INSTALL_DIR${NC}"
    mkdir -p "$INSTALL_DIR"
fi

# 安装
echo -e "\n${YELLOW}安装 MCP 到 $INSTALL_DIR${NC}"
mv "$BINARY_NAME" "$INSTALL_DIR/"

# 检查 PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "\n${YELLOW}添加 $INSTALL_DIR 到 PATH${NC}"
    
    # 检测使用的 shell
    SHELL_NAME=$(basename "$SHELL")
    case $SHELL_NAME in
        bash)
            PROFILE="$HOME/.bashrc"
            ;;
        zsh)
            PROFILE="$HOME/.zshrc"
            ;;
        *)
            PROFILE="$HOME/.profile"
            ;;
    esac
    
    # 添加到配置文件
    echo "" >> "$PROFILE"
    echo "# MCP Database Tools" >> "$PROFILE"
    echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$PROFILE"
    
    echo -e "${GREEN}✓ 已添加到 $PROFILE${NC}"
    echo -e "${YELLOW}请运行以下命令使其生效：${NC}"
    echo -e "${GREEN}source $PROFILE${NC}"
    
    # 临时添加到当前 session
    export PATH="$HOME/.local/bin:$PATH"
fi

# 验证安装
if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    echo -e "\n${GREEN}✓ MCP 安装成功！${NC}"
    echo -e "版本: $LATEST_VERSION"
    echo -e "位置: $INSTALL_DIR/$BINARY_NAME"
else
    echo -e "${RED}错误: 安装失败${NC}"
    exit 1
fi

# 显示配置提示
echo -e "\n${BLUE}=== 配置 Claude Desktop ===${NC}"
echo "编辑配置文件："
if [ "$PLATFORM" == "darwin" ]; then
    echo "~/Library/Application Support/Claude/claude_desktop_config.json"
else
    echo "~/.config/Claude/claude_desktop_config.json"
fi

echo -e "\n添加以下配置："
cat << EOF
{
  "mcpServers": {
    "database": {
      "command": "$INSTALL_DIR/$BINARY_NAME",
      "args": ["database"]
    }
  }
}
EOF

echo -e "\n${GREEN}安装完成！${NC}"