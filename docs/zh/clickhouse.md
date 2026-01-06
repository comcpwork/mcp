# ClickHouse 使用指南

## MCP 工具

**工具名称:** `clickhouse_exec`

## DSN 格式

```
clickhouse://username:password@host:port/database?options
```

### 示例

| 场景 | DSN |
|------|-----|
| 本地默认 | `clickhouse://default:@localhost:9000/default` |
| 带密码 | `clickhouse://default:password@localhost:9000/mydb` |
| 远程服务器 | `clickhouse://admin:pass123@192.168.1.100:9000/production` |
| 带超时 | `clickhouse://default:@localhost:9000/default?dial_timeout=10s` |
| 带读取超时 | `clickhouse://default:@localhost:9000/default?read_timeout=20s` |

### 常用选项

| 选项 | 描述 | 示例 |
|------|------|------|
| `dial_timeout` | 连接超时 | `dial_timeout=10s` |
| `read_timeout` | 读取超时 | `read_timeout=20s` |
| `write_timeout` | 写入超时 | `write_timeout=20s` |
| `compress` | 启用压缩 | `compress=lz4` |

## 使用示例

### 查询操作

向你的 AI 助手提问：

- "使用 DSN `clickhouse://default:@localhost:9000/default` 执行 ClickHouse: `SELECT 1`"
- "使用 DSN `clickhouse://default:@localhost:9000/default` 执行 ClickHouse: `SHOW DATABASES`"
- "使用 DSN `clickhouse://default:@localhost:9000/default` 执行 ClickHouse: `SHOW TABLES`"
- "使用 DSN `clickhouse://default:@localhost:9000/mydb` 执行 ClickHouse: `SELECT * FROM events LIMIT 10`"

### 表操作

- "使用 DSN `clickhouse://default:@localhost:9000/default` 执行 ClickHouse: `DESCRIBE TABLE events`"
- "使用 DSN `clickhouse://default:@localhost:9000/default` 执行 ClickHouse: `EXISTS TABLE events`"

### 数据修改

- "使用 DSN `clickhouse://default:@localhost:9000/mydb` 执行 ClickHouse: `INSERT INTO events (id, name, timestamp) VALUES (1, 'click', now())`"
- "使用 DSN `clickhouse://default:@localhost:9000/mydb` 执行 ClickHouse: `ALTER TABLE events ADD COLUMN category String`"

### 分析查询

- "使用 DSN `clickhouse://default:@localhost:9000/mydb` 执行 ClickHouse: `SELECT toDate(timestamp) as date, count() as cnt FROM events GROUP BY date ORDER BY date`"
- "使用 DSN `clickhouse://default:@localhost:9000/mydb` 执行 ClickHouse: `SELECT uniq(user_id) FROM events WHERE timestamp > now() - INTERVAL 1 DAY`"

## 支持的操作

### 查询语句
- `SELECT` - 查询数据
- `SHOW` - 显示数据库、表等
- `DESCRIBE` / `DESC` - 描述表结构
- `EXPLAIN` - 解释查询计划
- `EXISTS` - 检查表是否存在

### 修改语句
- `INSERT` - 插入数据
- `ALTER` - 修改表结构
- `CREATE` - 创建数据库/表
- `DROP` - 删除数据库/表
- `TRUNCATE` - 清空表

## 输出格式

### 查询结果

**≤5 列:** 表格格式
```
date          cnt
----------    -----
2024-01-01    1000
2024-01-02    1500
2024-01-03    1200
```

**>5 列:** 键值对格式
```
Row 1:
  id: 1
  name: click
  user_id: 12345
  timestamp: 2024-01-01 12:00:00
  ...
```

### 修改结果

```
Query OK, 1 row affected
```

## 使用技巧

1. **端口**: ClickHouse 原生协议使用端口 9000（不是 HTTP 的 8123）
2. **默认用户**: ClickHouse 默认用户是 `default`，密码为空
3. **压缩**: 使用 `compress=lz4` 提高网络传输性能
4. **LIMIT 子句**: 对大表始终使用 `LIMIT` 避免内存问题
5. **时间函数**: ClickHouse 有丰富的时间函数如 `now()`、`today()`、`toDate()`
