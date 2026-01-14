# MCP Windows 安装脚本

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\CoworkDatabase",
    [switch]$AddToPath = $true
)

# 颜色输出函数
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

Write-ColorOutput Green "=== MCP Windows 安装脚本 ==="
Write-Output ""

# 检测系统架构
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
Write-Output "系统架构: $arch"

# 获取最新版本
Write-ColorOutput Yellow "获取最新版本信息..."
try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/comcpwork/mcp/releases/latest"
    $version = $release.tag_name
    Write-Output "最新版本: $version"
} catch {
    Write-ColorOutput Red "错误: 无法获取最新版本"
    exit 1
}

# 构建下载URL
$binaryFile = "cowork-database-windows-$arch.exe"
$downloadUrl = "https://github.com/comcpwork/mcp/releases/download/$version/$binaryFile"

# 创建安装目录
if (!(Test-Path $InstallDir)) {
    Write-ColorOutput Yellow "创建安装目录..."
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# 下载文件
$tempFile = Join-Path $env:TEMP "cowork-database.exe"
Write-ColorOutput Yellow "下载 MCP..."
Write-Output "URL: $downloadUrl"

try {
    $ProgressPreference = 'SilentlyContinue'
    Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -UseBasicParsing
    $ProgressPreference = 'Continue'
} catch {
    Write-ColorOutput Red "错误: 下载失败"
    Write-Output $_.Exception.Message
    exit 1
}

# 复制到安装目录
$targetFile = Join-Path $InstallDir "cowork-database.exe"
Write-ColorOutput Yellow "安装 MCP..."
try {
    Copy-Item $tempFile $targetFile -Force
    Remove-Item $tempFile -Force
} catch {
    Write-ColorOutput Red "错误: 安装失败，可能需要管理员权限"
    Write-Output $_.Exception.Message
    exit 1
}

# 添加到用户 PATH
if ($AddToPath) {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$InstallDir*") {
        Write-ColorOutput Yellow "添加到用户 PATH..."
        try {
            [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "User")
            Write-ColorOutput Green "✓ 已添加到 PATH"
            Write-Output "注意: 请重启终端或运行 'refreshenv' 使 PATH 生效"
        } catch {
            Write-ColorOutput Red "警告: 无法自动添加到 PATH"
            Write-Output "请手动将以下路径添加到用户环境变量 PATH："
            Write-Output $InstallDir
        }
    }
}

# 验证安装
Write-Output ""
if (Test-Path $targetFile) {
    Write-ColorOutput Green "✓ MCP 安装成功！"
    Write-Output "安装位置: $targetFile"
    
    # 尝试获取版本
    try {
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
        $versionOutput = & $targetFile --version 2>$null
        if ($versionOutput) {
            Write-Output "版本: $versionOutput"
        }
    } catch {
        # 忽略版本检查错误
    }
} else {
    Write-ColorOutput Red "错误: 安装验证失败"
    exit 1
}

# 显示配置说明
Write-Output ""
Write-ColorOutput Blue "=== 配置 Claude Desktop ==="
Write-Output "编辑配置文件："
Write-Output "$env:APPDATA\Claude\claude_desktop_config.json"
Write-Output ""
Write-Output "添加以下配置："
Write-Output @"
{
  "mcpServers": {
    "database": {
      "command": "$targetFile",
      "args": ["database"]
    }
  }
}
"@

Write-Output ""
Write-ColorOutput Green "安装完成！"
Write-Output "注意: 可能需要重启终端或系统才能使 PATH 生效"