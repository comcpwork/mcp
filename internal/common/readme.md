# Common 包说明

本包提供了 MCP 项目的通用功能组件，用于减少代码重复并统一规范。

## 包含的组件

### 1. constants.go - 常量定义
- 查询相关常量（DefaultQueryLimit, MaxQueryLimit, DefaultMaxRows 等）
- 并发相关常量（MaxBatchConcurrency, DefaultBatchSize）
- 数据库相关常量（DefaultMySQLPort, DefaultRedisPort, DefaultPulsarPort 等）
- 配置相关常量（ConfigFileName, ConfigDirPath, LogDirPath 等）

### 2. errors.go - 统一错误处理
- 通用错误定义（ErrNoConfig, ErrInstanceNotFound, ErrConnectionFailed 等）
- ProviderError 结构体，支持错误包装和上下文
- 错误创建函数（NewNoConfigError, NewInstanceNotFoundError 等）
- 错误检查函数（IsNoConfigError, IsConnectionError）
- 错误格式化函数（FormatErrorMessage）

### 3. logging.go - 日志规范
- 统一的日志字段名常量定义
- LogHelper 结构体，提供标准化的日志记录方法
- 日志级别使用规范说明
- 常用日志记录函数（LogToolStart, LogToolSuccess, LogToolError 等）

### 4. config_handler.go - 配置管理基础
- ConfigManager 接口定义
- BaseConfigHandler 通用配置处理器
- 通用的 HandleUpdateConfig 和 HandleGetConfigDetails 实现
- 配置格式化函数

### 5. batch_query.go - 批量查询组件
- BatchQueryResult 结构体定义
- BatchQuery 通用批量查询函数
- BatchQueryWithConcurrency 可指定并发数的批量查询
- FormatBatchResults 批量结果格式化函数
- ExtractArrayParameter 数组参数提取函数

## 使用示例

### 使用常量替代魔法数字
```go
// 之前
configMaxRows = 1000

// 之后
configMaxRows = common.DefaultMaxRows
```

### 使用统一错误处理
```go
// 之前
return nil, errors.New("没有数据库配置")

// 之后
return nil, common.NewNoConfigError("mysql")
```

### 使用日志规范
```go
// 之前
log.Info(ctx, "处理 query 请求", log.String("tool", "query"))

// 之后
ctx = common.Logger.LogToolStart(ctx, "mysql", "query")
```

### 使用批量查询
```go
results := common.BatchQuery(ctx, items, func(ctx context.Context, item string) (string, error) {
    return queryItem(ctx, item)
})
```