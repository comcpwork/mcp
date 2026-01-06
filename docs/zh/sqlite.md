# SQLite 使用指南

## MCP 工具

**工具名称:** `sqlite_exec`

## DSN 格式

```
/path/to/database.db
:memory:
```

### 示例

| 场景 | DSN |
|------|-----|
| 内存数据库 | `:memory:` |
| 绝对路径 | `/Users/<username>/data/mydb.db` |
| 相对路径 | `./data/test.db` |
| 用户目录 | `~/databases/app.db` |

**注意:** 请将 `<username>` 替换为实际的用户名。

## 使用示例

### 内存数据库

向你的 AI 助手提问：

- "使用 DSN `:memory:` 执行 SQLite: `SELECT 1`"
- "使用 DSN `:memory:` 执行 SQLite: `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`"
- "使用 DSN `:memory:` 执行 SQLite: `INSERT INTO users (name) VALUES ('张三')`"

**注意:** 内存数据库是临时的，连接关闭后数据会丢失。

### 文件数据库

- "使用 DSN `/Users/john/data/mydb.db` 执行 SQLite: `SHOW TABLES`"
- "使用 DSN `./test.db` 执行 SQLite: `SELECT * FROM users`"
- "使用 DSN `~/app.db` 执行 SQLite: `DESCRIBE users`"

### 表操作

- "使用 DSN `./mydb.db` 执行 SQLite: `CREATE TABLE products (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, price REAL)`"
- "使用 DSN `./mydb.db` 执行 SQLite: `DROP TABLE IF EXISTS temp_data`"

### 数据操作

- "使用 DSN `./mydb.db` 执行 SQLite: `INSERT INTO products (name, price) VALUES ('小组件', 9.99)`"
- "使用 DSN `./mydb.db` 执行 SQLite: `UPDATE products SET price = 12.99 WHERE name = '小组件'`"
- "使用 DSN `./mydb.db` 执行 SQLite: `DELETE FROM products WHERE id = 1`"

## 支持的操作

### 查询语句
- `SELECT` - 查询数据
- `SHOW TABLES` - 列出所有表（SQLite 特定实现）
- `DESCRIBE` / `DESC` - 描述表结构
- `EXPLAIN` - 解释查询计划

### 修改语句
- `INSERT` - 插入数据
- `UPDATE` - 更新数据
- `DELETE` - 删除数据
- `CREATE` - 创建表
- `DROP` - 删除表
- `ALTER` - 修改表结构

## 输出格式

### 查询结果

**≤5 列:** 表格格式
```
id    name     price
----  -------  ------
1     小组件   9.99
2     大组件   19.99
```

**>5 列:** 键值对格式
```
Row 1:
  id: 1
  name: 小组件
  description: 一个有用的小组件
  price: 9.99
  stock: 100
  created_at: 2024-01-01
```

### 修改结果

```
Query OK, 1 row affected
Last Insert ID: 42
```

## SQLite 数据类型

| 类型 | 描述 |
|------|------|
| `INTEGER` | 有符号整数 |
| `REAL` | 浮点数 |
| `TEXT` | 文本字符串 |
| `BLOB` | 二进制数据 |
| `NULL` | 空值 |

## 常用模式

### 自增主键

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);
```

### 时间戳

```sql
CREATE TABLE logs (
    id INTEGER PRIMARY KEY,
    message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 外键

```sql
CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

## 使用技巧

1. **内存 vs 文件**: 测试使用 `:memory:`，持久化使用文件路径
2. **文件权限**: 确保目录存在且可写
3. **自增**: 使用 `INTEGER PRIMARY KEY AUTOINCREMENT` 实现自增 ID
4. **并发访问**: SQLite 支持有限的并发写入
5. **备份**: SQLite 数据库是单个文件 - 易于备份/复制
