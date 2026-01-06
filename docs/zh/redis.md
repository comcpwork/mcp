# Redis 使用指南

## MCP 工具

**工具名称:** `redis_exec`

## DSN 格式

```
redis://[:password@]host:port/database
```

### 示例

| 场景 | DSN |
|------|-----|
| 本地无密码 | `redis://localhost:6379/0` |
| 带密码 | `redis://:mypassword@localhost:6379/0` |
| 远程服务器 | `redis://:pass123@192.168.1.100:6379/1` |
| 带用户名 | `redis://user:password@localhost:6379/0` |

## 使用示例

### 字符串操作

向你的 AI 助手提问：

- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `PING`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `SET user:1001 张三`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `GET user:1001`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `DEL session:abc123`"

### 列表操作

- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `LPUSH queue task1`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `LRANGE todo_list 0 -1`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `RPOP queue`"

### 哈希操作

- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `HSET user:1001 name 张三 age 30`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `HGETALL user:1001`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `HGET user:1001 name`"

### 集合操作

- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `SADD online_users user123`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `SMEMBERS online_users`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `SISMEMBER online_users user123`"

### 键管理

- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `KEYS user:*`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `TTL session:abc123`"
- "在 `redis://localhost:6379/0` 上执行 Redis 命令 `EXPIRE session:abc123 3600`"

## 输出格式

### 字符串值

```
"张三"
```

### 整数值

```
(integer) 1
```

### 空值

```
(nil)
```

### 数组值

```
(3 items)
1) "item1"
2) "item2"
3) "item3"
```

### 哈希值

```
(4 fields)
name: 张三
age: 30
email: zhangsan@example.com
status: active
```

## 常用命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `PING` | 测试连接 | `PING` |
| `SET` | 设置字符串值 | `SET key value` |
| `GET` | 获取字符串值 | `GET key` |
| `DEL` | 删除键 | `DEL key1 key2` |
| `EXISTS` | 检查键是否存在 | `EXISTS key` |
| `KEYS` | 按模式查找键 | `KEYS user:*` |
| `TTL` | 获取过期时间 | `TTL key` |
| `EXPIRE` | 设置过期时间 | `EXPIRE key seconds` |
| `HSET` | 设置哈希字段 | `HSET hash field value` |
| `HGET` | 获取哈希字段 | `HGET hash field` |
| `HGETALL` | 获取所有哈希字段 | `HGETALL hash` |
| `LPUSH` | 从头部推入列表 | `LPUSH list value` |
| `RPUSH` | 从尾部推入列表 | `RPUSH list value` |
| `LRANGE` | 获取列表范围 | `LRANGE list 0 -1` |
| `SADD` | 添加到集合 | `SADD set member` |
| `SMEMBERS` | 获取集合所有成员 | `SMEMBERS set` |

## 使用技巧

1. **数据库选择**: Redis 有 16 个数据库（0-15），在 DSN 路径中指定
2. **键模式匹配**: 生产环境谨慎使用 `KEYS pattern`（会阻塞服务器）
3. **TTL 管理**: 使用 `EXPIRE` 管理会话/缓存
4. **数据类型**: 根据使用场景选择合适的数据结构
