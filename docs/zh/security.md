# 安全选项

## 概述

MCP 工具提供细粒度的安全控制来防止危险操作。默认情况下，所有操作都是允许的，但您可以使用命令行标志选择性地禁用特定操作。

## MySQL 安全标志

```bash
# 禁用 DROP 操作
mcp mysql --disable-drop

# 禁用多个操作
mcp mysql --disable-drop --disable-truncate --disable-delete

# 最大限制
mcp mysql --disable-create --disable-drop --disable-alter --disable-truncate --disable-update --disable-delete
```

### 可用标志

| 标志 | 描述 | 阻止的操作 |
|------|------|-----------|
| `--disable-create` | 禁止 CREATE 操作 | CREATE DATABASE, CREATE TABLE, CREATE INDEX |
| `--disable-drop` | 禁止 DROP 操作 | DROP DATABASE, DROP TABLE, DROP INDEX |
| `--disable-alter` | 禁止 ALTER 操作 | ALTER TABLE, ALTER DATABASE |
| `--disable-truncate` | 禁止 TRUNCATE 操作 | TRUNCATE TABLE |
| `--disable-update` | 禁止 UPDATE 操作 | UPDATE 语句 |
| `--disable-delete` | 禁止 DELETE 操作 | DELETE 语句 |

## Redis 安全标志

```bash
# 禁用危险命令
mcp redis --disable-delete --disable-update
```

### 阻止的命令

- **--disable-delete**: DEL, UNLINK, FLUSHDB, FLUSHALL
- **--disable-update**: CONFIG, EVAL, EVALSHA, SCRIPT

## Pulsar 安全标志

```bash
# 禁用管理操作
mcp pulsar --disable-create --disable-drop
```

### 阻止的操作

- **--disable-create**: 创建租户/命名空间/主题/订阅
- **--disable-drop**: 删除租户/命名空间/主题/订阅
- **--disable-update**: 更新配置

## 最佳实践

1. **开发环境**: 使用默认设置（允许所有操作）
2. **测试环境**: 启用一些限制以提高安全性
3. **生产环境**: 最大限制，仅允许读取操作
4. **CI/CD**: 根据流水线需求自定义

## 示例

### 只读 MySQL 访问
```bash
mcp mysql --disable-create --disable-drop --disable-alter --disable-truncate --disable-update --disable-delete
```

### 安全的 Redis 访问
```bash
mcp redis --disable-delete --disable-update
```

### 受限的 Pulsar 管理
```bash
mcp pulsar --disable-drop --disable-update
```