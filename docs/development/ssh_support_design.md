# SSH 隧道支持设计方案

## 一、背景

当前 MCP 数据库工具（MySQL、Redis、ClickHouse）直接通过 DSN 连接目标数据库。但在生产环境中，数据库通常部署在内网，需要通过 SSH 跳板机访问。本方案为现有工具添加可选的 SSH 隧道支持。

## 二、现状分析

### 2.1 现有工具

| 工具名 | 文件 | DSN格式 | SSH支持 | SSH模式 |
|--------|------|---------|---------|---------|
| `mysql_exec` | `database/mysql.go` | `user:pass@tcp(host:port)/db` | ✅ | TCP隧道 |
| `redis_exec` | `database/redis.go` | `redis://host:port/db` | ✅ | TCP隧道 |
| `clickhouse_exec` | `database/clickhouse.go` | `clickhouse://user:pass@host:port/db` | ✅ | TCP隧道 |
| `sqlite_exec` | `database/sqlite.go` | 本地/远程文件路径 | ✅ | 远程命令 |

### 2.2 当前连接流程

```
请求 → 解析DSN → 建立连接 → 执行SQL → 关闭连接 → 返回结果
```

## 三、设计方案

### 3.1 新增参数

为 MySQL、Redis、ClickHouse 工具添加单个可选参数 `ssh`：

| 参数名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `ssh` | string | 否 | SSH隧道URI，格式见下方 |

### 3.2 SSH URI 格式

支持两种格式：

#### 格式1：完整URI
```
ssh://[user[:password]@]host[:port][?key=path&passphrase=xxx]
```

| 部分 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `user` | 是 | - | SSH用户名 |
| `password` | 否 | - | SSH密码（与key二选一） |
| `host` | 是 | - | SSH服务器地址 |
| `port` | 否 | 22 | SSH端口 |
| `key` | 否 | - | 私钥文件路径（URL编码） |
| `passphrase` | 否 | - | 私钥密码（URL编码） |

#### 格式2：引用SSH配置（推荐）
```
ssh://config-name
```

直接引用 `~/.ssh/config` 中的Host配置。例如：

**~/.ssh/config 内容：**
```
Host myserver
    HostName jump.example.com
    User admin
    Port 2222
    IdentityFile ~/.ssh/id_rsa

Host prod-db
    HostName 192.168.1.100
    User deploy
    IdentityFile ~/.ssh/prod_key
```

**使用方式：**
```json
{
  "dsn": "root:pass@tcp(10.0.0.100:3306)/mydb",
  "sql": "SELECT 1",
  "ssh": "ssh://myserver"
}
```

**解析优先级：**
1. 如果URI中包含 `@`（如 `ssh://user@host`），按完整URI解析
2. 如果URI只有名称（如 `ssh://myserver`），从 `~/.ssh/config` 读取配置

### 3.3 DSN说明

**重要**：DSN中的地址应该是**从SSH服务器角度能访问的地址**，而不是从本地能访问的地址。工具不会重写DSN，会直接通过SSH隧道转发到DSN指定的地址。

```
┌─────────┐      SSH隧道       ┌──────────────┐      TCP       ┌──────────────┐
│  本地   │  ───────────────►  │  SSH服务器   │  ───────────►  │   数据库     │
│         │                    │ jump.example │                │ 10.0.0.100   │
└─────────┘                    └──────────────┘                └──────────────┘

DSN中填写: 10.0.0.100:3306 （SSH服务器能访问的地址）
```

### 3.4 使用示例

#### MySQL + SSH密钥认证
```json
{
  "dsn": "root:password@tcp(10.0.0.100:3306)/mydb",
  "sql": "SELECT * FROM users LIMIT 10",
  "ssh": "ssh://admin@jump.example.com?key=/home/user/.ssh/id_rsa"
}
```

#### MySQL + SSH密钥认证（带密码保护的私钥）
```json
{
  "dsn": "root:password@tcp(10.0.0.100:3306)/mydb",
  "sql": "SELECT * FROM users LIMIT 10",
  "ssh": "ssh://admin@jump.example.com:2222?key=/home/user/.ssh/id_rsa&passphrase=myKeyPass"
}
```

#### Redis + SSH密码认证
```json
{
  "dsn": "redis://:password@10.0.0.101:6379/0",
  "command": "GET mykey",
  "ssh": "ssh://admin:ssh_password@jump.example.com:2222"
}
```

#### ClickHouse + SSH密钥认证
```json
{
  "dsn": "clickhouse://default:@10.0.0.102:9000/mydb",
  "sql": "SELECT * FROM events LIMIT 10",
  "ssh": "ssh://root@bastion.example.com?key=/root/.ssh/id_ed25519"
}
```

#### SQLite + SSH远程执行
```json
{
  "dsn": "/data/app.db",
  "sql": "SELECT * FROM users LIMIT 10",
  "ssh": "ssh://admin@server.example.com?key=/home/user/.ssh/id_rsa"
}
```
**注意**：SQLite的SSH模式通过在远程服务器执行`sqlite3`命令实现，远程服务器需要安装sqlite3命令行工具。

### 3.5 SSH URI 示例汇总

```
# 引用SSH配置（推荐，最简洁）
ssh://myserver
ssh://prod-db

# 密码认证
ssh://user:password@host
ssh://user:password@host:2222

# 密钥认证（默认端口22）
ssh://user@host?key=/path/to/id_rsa

# 密钥认证（自定义端口）
ssh://user@host:2222?key=/path/to/id_rsa

# 加密私钥
ssh://user@host?key=/path/to/id_rsa&passphrase=keypass

# 路径包含特殊字符（需URL编码）
ssh://user@host?key=%2Fhome%2Fuser%2F.ssh%2Fid_rsa
```

### 3.6 连接流程（启用SSH时）

#### MySQL/Redis/ClickHouse（TCP隧道模式）
```
请求 → 解析SSH URI → 建立SSH隧道(本地端口转发到DSN地址) → 建立数据库连接(通过本地隧道端口) → 执行SQL → 关闭数据库 → 关闭隧道 → 返回结果
```

#### SQLite（远程命令执行模式）
```
请求 → 解析SSH URI → 建立SSH连接 → 执行远程sqlite3命令 → 原样返回输出 → 关闭SSH连接
```

## 四、实现设计

### 4.1 新增文件

#### `database/ssh.go` - SSH隧道核心实现

```go
package database

import (
    "fmt"
    "io"
    "net"
    "os"
    "golang.org/x/crypto/ssh"
)

// SSHConfig SSH连接配置
type SSHConfig struct {
    Host       string // SSH服务器地址
    Port       int    // SSH端口，默认22
    User       string // SSH用户名
    Password   string // SSH密码
    KeyPath    string // 私钥文件路径
    Passphrase string // 私钥密码
}

// SSHTunnel SSH隧道
type SSHTunnel struct {
    config     *SSHConfig
    client     *ssh.Client
    listener   net.Listener
    localPort  int
    done       chan struct{}
}

// NewSSHTunnel 创建SSH隧道实例
func NewSSHTunnel(config *SSHConfig) *SSHTunnel

// Start 启动隧道到指定远程地址，返回本地监听端口
func (t *SSHTunnel) Start(remoteHost string, remotePort int) (localPort int, err error)

// LocalAddr 获取本地隧道地址 127.0.0.1:port
func (t *SSHTunnel) LocalAddr() string

// Close 关闭隧道
func (t *SSHTunnel) Close() error

// getSSHAuthMethods 根据配置获取认证方法
func getSSHAuthMethods(config *SSHConfig) ([]ssh.AuthMethod, error)

// readPrivateKey 读取并解析私钥文件
func readPrivateKey(path string, passphrase string) (ssh.Signer, error)
```

#### `database/ssh_helper.go` - 辅助函数

```go
package database

import (
    "net/url"
    "os"
    "path/filepath"
    "strconv"
)

// ParseSSHURI 解析SSH URI字符串为SSHConfig
// 支持两种格式:
//   1. ssh://config-name - 从~/.ssh/config读取
//   2. ssh://[user[:password]@]host[:port][?key=path&passphrase=xxx] - 完整URI
// 如果uri为空，返回nil表示不使用SSH
func ParseSSHURI(uri string) (*SSHConfig, error)

// ParseSSHConfig 解析~/.ssh/config文件，查找指定Host的配置
func ParseSSHConfig(hostName string) (*SSHConfig, error)

// isSSHConfigReference 判断URI是否是SSH配置引用（不含@符号）
func isSSHConfigReference(uri string) bool

// ExtractHostPort 从DSN中提取host:port用于隧道转发
func ExtractMySQLHostPort(dsn string) (host string, port int, err error)
func ExtractRedisHostPort(dsn string) (host string, port int, err error)
func ExtractClickHouseHostPort(dsn string) (host string, port int, err error)

// ReplaceDSNHostPort 替换DSN中的host:port为本地隧道地址
func ReplaceMySQLDSNHostPort(dsn string, localAddr string) string
func ReplaceRedisDSNHostPort(dsn string, localAddr string) string
func ReplaceClickHouseDSNHostPort(dsn string, localAddr string) string
```

#### SSH Config 解析示例

```go
// ParseSSHConfig 从~/.ssh/config解析指定Host的配置
func ParseSSHConfig(hostName string) (*SSHConfig, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }

    configPath := filepath.Join(homeDir, ".ssh", "config")
    // 使用第三方库解析，如 github.com/kevinburke/ssh_config
    // 或手动解析

    cfg, err := ssh_config.DecodeFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to parse SSH config: %w", err)
    }

    host := cfg.Get(hostName, "HostName")
    if host == "" {
        return nil, fmt.Errorf("SSH config host '%s' not found", hostName)
    }

    config := &SSHConfig{
        Host: host,
        Port: 22,
        User: cfg.Get(hostName, "User"),
    }

    if portStr := cfg.Get(hostName, "Port"); portStr != "" {
        config.Port, _ = strconv.Atoi(portStr)
    }

    if keyPath := cfg.Get(hostName, "IdentityFile"); keyPath != "" {
        // 展开 ~ 为home目录
        config.KeyPath = expandPath(keyPath)
    }

    return config, nil
}
```

#### SSH URI 解析示例

```go
// ParseSSHURI 解析SSH URI
func ParseSSHURI(uri string) (*SSHConfig, error) {
    if uri == "" {
        return nil, nil
    }

    u, err := url.Parse(uri)
    if err != nil {
        return nil, fmt.Errorf("invalid SSH URI: %w", err)
    }

    if u.Scheme != "ssh" {
        return nil, fmt.Errorf("invalid scheme: expected 'ssh', got '%s'", u.Scheme)
    }

    config := &SSHConfig{
        Host: u.Hostname(),
        Port: 22, // 默认端口
    }

    // 解析端口
    if portStr := u.Port(); portStr != "" {
        port, err := strconv.Atoi(portStr)
        if err != nil {
            return nil, fmt.Errorf("invalid SSH port: %s", portStr)
        }
        config.Port = port
    }

    // 解析用户名和密码
    if u.User != nil {
        config.User = u.User.Username()
        if password, ok := u.User.Password(); ok {
            config.Password = password
        }
    }

    // 解析query参数 (key, passphrase)
    query := u.Query()
    if keyPath := query.Get("key"); keyPath != "" {
        config.KeyPath = keyPath
    }
    if passphrase := query.Get("passphrase"); passphrase != "" {
        config.Passphrase = passphrase
    }

    // 验证: 必须有用户名
    if config.User == "" {
        return nil, fmt.Errorf("SSH user is required")
    }

    // 验证: 必须有密码或私钥
    if config.Password == "" && config.KeyPath == "" {
        return nil, fmt.Errorf("SSH password or key is required")
    }

    return config, nil
}
```

### 4.2 修改现有文件

#### `database/server.go` - 添加SSH参数定义

为 mysql_exec、redis_exec、clickhouse_exec 添加单个 `ssh` 参数：

```go
server.AddTool(
    mcp.NewTool("mysql_exec",
        mcp.WithDescription("Execute MySQL SQL statements..."),
        mcp.WithString("dsn", mcp.Required(), mcp.Description("MySQL DSN string...")),
        mcp.WithString("sql", mcp.Required(), mcp.Description("SQL statement to execute...")),
        // 新增SSH参数（可选）
        mcp.WithString("ssh", mcp.Description(
            "SSH tunnel URI for connecting through bastion host. "+
            "Format: ssh://user[:password]@host[:port][?key=/path/to/key&passphrase=xxx]. "+
            "Example: ssh://admin@jump.example.com?key=/home/user/.ssh/id_rsa")),
    ),
    handleMySQLExec,
)
```

#### `database/mysql.go` - 集成SSH隧道

```go
func handleMySQLExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    dsn, err := req.RequireString("dsn")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    sqlQuery, err := req.RequireString("sql")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    // 检查是否需要SSH隧道
    sshURI, _ := req.GetString("ssh")
    if sshURI != "" {
        sshConfig, err := ParseSSHURI(sshURI)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Invalid SSH URI: %v", err)), nil
        }

        // 提取目标数据库地址
        remoteHost, remotePort, err := ExtractMySQLHostPort(dsn)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to parse DSN: %v", err)), nil
        }

        // 建立SSH隧道
        tunnel := NewSSHTunnel(sshConfig)
        localPort, err := tunnel.Start(remoteHost, remotePort)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("SSH tunnel failed: %v", err)), nil
        }
        defer tunnel.Close()

        // 重写DSN为本地隧道地址
        dsn = RewriteMySQLDSN(dsn, tunnel.LocalAddr())
    }

    // 后续逻辑保持不变...
    db, err := sql.Open("mysql", dsn)
    // ...
}
```

#### `database/redis.go` - 集成SSH隧道

```go
func handleRedisExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    dsn, err := req.RequireString("dsn")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    command, err := req.RequireString("command")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    // 检查是否需要SSH隧道
    sshURI, _ := req.GetString("ssh")
    if sshURI != "" {
        sshConfig, err := ParseSSHURI(sshURI)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Invalid SSH URI: %v", err)), nil
        }

        remoteHost, remotePort, err := ExtractRedisHostPort(dsn)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to parse DSN: %v", err)), nil
        }

        tunnel := NewSSHTunnel(sshConfig)
        if _, err := tunnel.Start(remoteHost, remotePort); err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("SSH tunnel failed: %v", err)), nil
        }
        defer tunnel.Close()

        dsn = RewriteRedisDSN(dsn, tunnel.LocalAddr())
    }

    // 后续逻辑保持不变...
}
```

#### `database/clickhouse.go` - 集成SSH隧道

与 MySQL 类似，使用 `ExtractClickHouseHostPort` 和 `ReplaceClickHouseDSNHostPort`。

#### `database/sqlite.go` - SSH远程命令执行

```go
func handleSQLiteExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    dsn, err := req.RequireString("dsn")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    sqlQuery, err := req.RequireString("sql")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    // 检查是否需要SSH远程执行
    sshURI, _ := req.GetString("ssh")
    if sshURI != "" {
        return handleSQLiteSSHExec(ctx, sshURI, dsn, sqlQuery)
    }

    // 本地执行逻辑保持不变...
    db, err := sql.Open("sqlite3", dsn)
    // ...
}

// handleSQLiteSSHExec 通过SSH远程执行sqlite3命令
func handleSQLiteSSHExec(ctx context.Context, sshURI, dbPath, sqlQuery string) (*mcp.CallToolResult, error) {
    sshConfig, err := ParseSSHURI(sshURI)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Invalid SSH URI: %v", err)), nil
    }

    // 建立SSH连接
    client, err := NewSSHClient(sshConfig)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("SSH connection failed: %v", err)), nil
    }
    defer client.Close()

    // 构建sqlite3命令（使用-header -column格式输出）
    cmd := fmt.Sprintf("sqlite3 -header -column '%s' '%s'",
        escapeSingleQuote(dbPath),
        escapeSingleQuote(sqlQuery))

    // 执行远程命令
    output, err := client.Run(cmd)
    if err != nil {
        // 检查是否是sqlite3不存在
        if strings.Contains(err.Error(), "not found") ||
           strings.Contains(err.Error(), "command not found") {
            return mcp.NewToolResultError("sqlite3 command not found on remote server"), nil
        }
        return mcp.NewToolResultError(fmt.Sprintf("Remote execution failed: %v", err)), nil
    }

    // 原样返回输出，不做格式化
    return mcp.NewToolResultText(output), nil
}
```

### 4.3 依赖添加

```bash
# SSH连接核心库
go get golang.org/x/crypto/ssh

# SSH config文件解析（可选，支持ssh://config-name格式）
go get github.com/kevinburke/ssh_config
```

## 五、连接方式详解

### 5.1 MySQL/Redis/ClickHouse（TCP隧道模式）

**流程**：
```
1. 从DSN中提取目标 host:port（如 10.0.0.100:3306）
2. 建立SSH隧道：本地随机端口 → SSH服务器 → 目标 host:port
3. 替换DSN中的地址为本地隧道地址（如 127.0.0.1:54321）
4. 使用替换后的DSN建立数据库连接
```

**DSN地址替换示例**：
```
MySQL:
  原始: user:password@tcp(10.0.0.100:3306)/mydb
  替换: user:password@tcp(127.0.0.1:54321)/mydb

Redis:
  原始: redis://:password@10.0.0.101:6379/0
  替换: redis://:password@127.0.0.1:54321/0

ClickHouse:
  原始: clickhouse://user:pass@10.0.0.102:9000/mydb
  替换: clickhouse://user:pass@127.0.0.1:54321/mydb
```

### 5.2 SQLite（远程命令执行模式）

**流程**：
```
1. 建立SSH连接
2. 执行远程命令: sqlite3 -header -column '<db_path>' '<sql>'
3. 原样返回命令输出
4. 关闭SSH连接
```

**注意**：
- DSN为远程服务器上的数据库文件路径（如 `/data/app.db`）
- 远程服务器需要安装 `sqlite3` 命令行工具
- 输出不做格式化，保持sqlite3原始输出格式

## 六、错误处理

### 6.1 SSH相关错误

| 错误场景 | 错误信息 |
|---------|---------|
| SSH连接失败 | `SSH connection failed: <detail>` |
| 认证失败 | `SSH authentication failed: <detail>` |
| 私钥读取失败 | `Failed to read SSH key: <detail>` |
| 私钥解密失败 | `Failed to decrypt SSH key: invalid passphrase` |
| 隧道建立失败 | `SSH tunnel setup failed: <detail>` |

### 6.2 参数校验

- 如果提供了 `ssh_host`，则 `ssh_user` 必需
- `ssh_password` 和 `ssh_key` 至少提供一个
- 如果提供了 `ssh_key_passphrase`，则 `ssh_key` 必需

## 七、安全考虑

1. **私钥文件权限**：检查私钥文件权限是否为600
2. **密码传输**：密码在内存中使用后应及时清理
3. **日志脱敏**：日志中不输出密码和私钥内容
4. **超时控制**：SSH连接和隧道应有合理的超时设置

## 八、实现步骤

### Phase 1: 核心实现
- [ ] 创建 `database/ssh.go` - SSH隧道核心
- [ ] 创建 `database/ssh_helper.go` - 辅助函数
- [ ] 添加依赖 `golang.org/x/crypto/ssh`

### Phase 2: 集成
- [ ] 修改 `database/server.go` - 添加SSH参数
- [ ] 修改 `database/mysql.go` - 集成SSH隧道
- [ ] 修改 `database/redis.go` - 集成SSH隧道
- [ ] 修改 `database/clickhouse.go` - 集成SSH隧道

### Phase 3: 测试与文档
- [ ] 创建 `test/test_ssh.py` - SSH功能测试
- [ ] 更新 `docs/en/*.md` - 英文文档
- [ ] 更新 `docs/zh/*.md` - 中文文档
- [ ] 更新 `README.md` 和 `README_CN.md`

## 九、待讨论问题

1. **SSH Agent支持**：是否需要支持SSH Agent？
   - 可通过 `ssh://user@host?agent=true` 启用
   - 或自动检测 `SSH_AUTH_SOCK` 环境变量

2. **Known Hosts验证**：是否强制验证服务器指纹？
   - 选项A：默认跳过验证（方便使用，但有安全风险）
   - 选项B：默认验证，可通过 `?insecure=true` 跳过
   - 选项C：首次连接提示用户确认

3. **连接复用**：多次请求是否复用同一SSH连接？
   - 当前设计：每次请求独立建立和关闭隧道（无状态）
   - 优点：简单，无状态
   - 缺点：频繁建立SSH连接有性能开销

4. **超时配置**：是否需要暴露SSH连接超时参数？
   - 可通过 `?timeout=30s` 设置
   - 或使用固定默认值（如30秒）

5. **跳板机链**：是否需要支持多级跳板机？
   - 格式提案：`ssh://user@host1?proxy=ssh://user@host2`
   - 复杂度较高，建议后续版本考虑

6. **默认私钥路径**：是否支持自动查找默认私钥？
   - 当未指定 `key` 和 `password` 时，尝试 `~/.ssh/id_rsa`、`~/.ssh/id_ed25519`
   - 需要讨论是否符合预期

---

*文档创建时间: 2026-01-14*
*状态: 草案*
