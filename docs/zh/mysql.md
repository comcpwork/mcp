# MySQL 使用指南

## 开始使用

### 连接到 MySQL

告诉 Claude：
- "连接到 192.168.1.100:3306 的 MySQL，用户名 root，密码 mypass"
- "连接到本地的 MySQL 数据库"

### 基本操作

#### 查看表
- "显示所有表"
- "列出数据库中的表"

#### 查询数据
- "查询 users 表"
- "显示 orders 表的前 10 条记录"
- "查找今天注册的用户"

#### 修改数据
- "插入新用户张三，邮箱 zhangsan@example.com"
- "将 ID 为 123 的用户状态更新为活跃"
- "删除 logs 表中 30 天前的记录"

## 高级功能

### 表信息
- "描述 users 表的结构"
- "显示 products 表的索引"
- "查看表的大小和行数"

### 数据库管理
- "创建一个名为 test_db 的新数据库"
- "显示所有数据库"
- "切换到 production 数据库"

### 复杂查询
- "显示最近 6 个月的月度销售汇总"
- "查找 users 表中的重复邮箱地址"
- "关联 orders 和 customers 表，显示最近的购买记录"

## 安全选项

启动 MCP 服务器时，可以添加安全标志：

```bash
mcp mysql --disable-drop --disable-truncate
```

可用标志：
- `--disable-create` - 禁止 CREATE 操作
- `--disable-drop` - 禁止 DROP 操作
- `--disable-alter` - 禁止 ALTER 操作
- `--disable-truncate` - 禁止 TRUNCATE 操作
- `--disable-update` - 禁止 UPDATE 操作
- `--disable-delete` - 禁止 DELETE 操作

## 使用技巧

1. **自然语言**：用自然语言描述你想要的操作
2. **上下文**：Claude 会在对话中记住你的数据库上下文
3. **安全性**：危险操作会要求确认
4. **历史记录**：使用"显示连接历史"查看之前的连接