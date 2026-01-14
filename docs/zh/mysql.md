# MySQL 使用指南

## MCP 工具

**工具名称:** `mysql_exec`

## DSN 格式

```
username:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true
```

### 示例

| 场景 | DSN |
|------|-----|
| 本地数据库 | `root:password@tcp(localhost:3306)/mydb` |
| 远程服务器 | `admin:pass123@tcp(192.168.1.100:3306)/production` |
| 指定字符集 | `root:pass@tcp(localhost:3306)/mydb?charset=utf8mb4` |
| 解析时间 | `root:pass@tcp(localhost:3306)/mydb?parseTime=true` |

## 使用示例

### 查询操作

向你的 AI 助手提问：

- "使用 DSN `root:password@tcp(localhost:3306)/mydb` 执行 MySQL: `SHOW TABLES`"
- "使用 DSN `root:password@tcp(localhost:3306)/mydb` 执行 MySQL: `SELECT * FROM users LIMIT 10`"
- "使用 DSN `root:password@tcp(localhost:3306)/mydb` 执行 MySQL: `DESCRIBE users`"

### 修改操作

- "使用 DSN `root:password@tcp(localhost:3306)/mydb` 执行 MySQL: `INSERT INTO users (name, email) VALUES ('张三', 'zhangsan@example.com')`"
- "使用 DSN `root:password@tcp(localhost:3306)/mydb` 执行 MySQL: `UPDATE users SET status = 'active' WHERE id = 123`"
- "使用 DSN `root:password@tcp(localhost:3306)/mydb` 执行 MySQL: `DELETE FROM logs WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY)`"

## 支持的操作

### 查询语句
- `SELECT` - 查询数据
- `SHOW` - 显示数据库对象
- `DESCRIBE` / `DESC` - 描述表结构
- `EXPLAIN` - 解释查询计划

### 修改语句
- `INSERT` - 插入数据
- `UPDATE` - 更新数据
- `DELETE` - 删除数据
- `CREATE` - 创建数据库/表
- `DROP` - 删除数据库/表
- `ALTER` - 修改表结构
- `TRUNCATE` - 清空表

## 输出格式

### 查询结果

**≤5 列:** 表格格式
```
ID    Name    Email
----  ------  -----------------
1     张三    zhangsan@example.com
2     李四    lisi@example.com
```

**>5 列:** 键值对格式
```
Row 1:
  id: 1
  name: 张三
  email: zhangsan@example.com
  ...
```

### 修改结果

```
Query OK, 1 row affected
Last Insert ID: 42
```

## SSH 连接

通过可选的 `ssh` 参数，使用 SSH 跳板机连接 MySQL。

### SSH URI 格式

| 格式 | 示例 | 说明 |
|------|------|------|
| 配置引用 | `ssh://myserver` | 使用 `~/.ssh/config` 中的配置 |
| 密码认证 | `ssh://user:pass@host:port` | 直接密码认证 |
| 密钥认证 | `ssh://user@host?key=/path/to/key` | 私钥认证 |
| 加密密钥 | `ssh://user@host?key=/path/to/key&passphrase=xxx` | 加密私钥 |

### 示例

**使用 SSH 配置：**
```
DSN: root:password@tcp(10.0.0.100:3306)/mydb
SSH: ssh://myserver
```

**使用 SSH 密钥：**
```
DSN: root:password@tcp(10.0.0.100:3306)/mydb
SSH: ssh://admin@jump.example.com?key=~/.ssh/id_rsa
```

**使用 SSH 密码：**
```
DSN: root:password@tcp(10.0.0.100:3306)/mydb
SSH: ssh://admin:sshpass@jump.example.com:2222
```

> **注意：** DSN 中的 host:port 应该是从 SSH 服务器可访问的地址（如内网 IP）。

## 使用技巧

1. **DSN 安全**: 不要在日志或共享环境中暴露包含敏感凭据的 DSN
2. **LIMIT 子句**: 对大表使用 `LIMIT` 控制结果集大小
3. **自然语言**: 你可以用自然语言描述需求，AI 会构建适当的 SQL
