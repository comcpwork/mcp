# GitHub CLI Release 命令

## 创建 Release

```bash
# 基本创建
gh release create v1.0.0

# 带标题和说明
gh release create v1.0.0 \
  --title "MCP v1.0.0" \
  --notes "First release"

# 从文件读取说明
gh release create v1.0.0 \
  --title "MCP v1.0.0" \
  --notes-file RELEASE_NOTES.md

# 上传多个文件
gh release create v1.0.0 \
  --title "MCP v1.0.0" \
  --notes "First release" \
  mcp-linux-amd64 \
  mcp-darwin-amd64 \
  mcp-windows-amd64.exe

# 创建预发布版本
gh release create v1.0.0-beta.1 \
  --prerelease \
  --title "MCP v1.0.0 Beta 1"

# 创建草稿
gh release create v1.0.0 \
  --draft \
  --title "MCP v1.0.0"
```

## 查看 Release

```bash
# 列出所有 releases
gh release list

# 查看特定 release
gh release view v1.0.0

# 查看最新 release
gh release view --latest
```

## 下载 Release 文件

```bash
# 下载特定版本的所有文件
gh release download v1.0.0

# 下载到指定目录
gh release download v1.0.0 --dir ./downloads

# 只下载特定文件
gh release download v1.0.0 --pattern "*.tar.gz"
```

## 编辑 Release

```bash
# 编辑 release 说明
gh release edit v1.0.0 --notes "Updated release notes"

# 上传额外的文件
gh release upload v1.0.0 checksums.txt

# 将草稿发布为正式版本
gh release edit v1.0.0 --draft=false
```

## 删除 Release

```bash
# 删除 release（保留标签）
gh release delete v1.0.0

# 删除 release 和标签
gh release delete v1.0.0 --delete-tag
```

## 实用示例

### 1. 自动生成更新日志

```bash
# 基于提交历史生成
gh release create v1.0.0 --generate-notes
```

### 2. 批量上传文件

```bash
# 使用通配符
gh release create v1.0.0 ~/go/bin/mcp/mcp-*
```

### 3. 从 CI/CD 发布

```bash
# 使用环境变量中的 token
GH_TOKEN=$GITHUB_TOKEN gh release create v1.0.0
```

### 4. 检查文件是否上传成功

```bash
# 查看 release 的文件列表
gh release view v1.0.0 --json assets --jq '.assets[].name'
```