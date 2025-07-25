# MCP 项目开发要求

## 零、文档规范
- **必须**在需要用户填写的内容处使用 `<>` 标记，如 `/home/<username>/`
- **必须**明确告知用户需要替换占位符为实际值
- **必须**同步更新 README.md 和 README_CN.md，保持中英文版本一致
- **必须**在修改 README 时，确保两个语言版本的内容完全对应

## 一、强制要求（MUST DO）

### 1. 成功证明要求
**当宣布任何功能或修改成功时，必须提供充分且不可辩驳的证据**

### 2. 命名规范
- **必须**文件名使用英文
- **必须**代码注释使用中文
- **必须**文档内容使用中文

### 3. 文档查阅
- **必须**使用 Context7 查看最新的编码文档和 API 文档
- **必须**在不知道如何使用某个库时，先通过 Context7 获取使用方法
- **禁止**直接猜测 API 用法

### 4. 日志规范
- **必须**使用 `mcp/pkg/log` 包进行日志记录
- **必须**将日志输出到文件 `~/.co-mcp/logs/<server>.log`
- **必须**使用统一的字段名常量（如 `common.FieldTool`, `common.FieldSQL`）
- **必须**为关键操作添加日志（工具开始、连接状态、错误等）
- **禁止**使用标准库的 `log` 包
- **禁止**输出日志到 stdout/stderr

### 5. 代码规范
- **必须**使用 `common/constants.go` 中定义的常量
- **必须**使用 `common/errors.go` 中的错误类型和函数
- **必须**使用 `common/batch_query.go` 实现批量查询
- **必须**使用 `common/config_handler.go` 的基础配置处理器
- **禁止**使用全局变量
- **禁止**使用 `init()` 函数
- **禁止**硬编码数字（如 1000, 3306, 30000）
- **禁止**自定义错误类型（使用 common 包中的）
- **禁止**在多个 provider 中重复相同代码

### 6. 编译规范
- **必须**使用 `make` 命令进行编译
- **必须**将二进制文件输出到 `~/go/bin/mcp/`
- **禁止**直接使用 `go build`
- **禁止**在源码目录中保留二进制文件

### 7. MCP 服务器规范
- **必须**使用懒加载机制连接数据库
- **必须**在没有配置时能正常启动
- **必须**在连接失败时返回清晰错误信息
- **禁止**在启动时立即初始化数据库连接

### 8. 输出格式
- **必须**使用紧凑输出格式
- **必须**所有 MCP 工具的返回都必须使用 compact 格式
- **必须**所有输出使用英文（表头、字段名、状态信息）
- **必须**所有参数描述使用英文
- **必须**在 describe_table 中包含 COMMENT 信息

### 9. 测试规范
- **必须**将测试文件放在 `test/` 目录下
- **必须**测试文件命名为 `test_<功能>.py`
- **必须**覆盖关键功能和边界情况

## 二、建议做（SHOULD DO）

### 1. 日志级别
- **建议**使用正确的日志级别：
  - Debug: 详细调试信息
  - Info: 正常操作信息
  - Warn: 可能的问题
  - Error: 操作失败

### 2. 错误处理
- **建议**使用 `errors.Wrap()` 保留错误上下文
- **建议**为用户返回友好的错误消息
- **建议**在日志中记录详细错误信息

### 3. 代码组织
- **建议**依赖注入优于全局状态
- **建议**提取通用功能到 common 包
- **建议**保持函数简短，单一职责

## 三、可以做（MAY DO）

### 1. 测试工具
- **可以**使用 Python 脚本进行功能测试
- **可以**使用 MCP 客户端库进行测试
- **可以**使用自定义客户端封装

### 2. 扩展功能
- **可以**添加新的工具和资源
- **可以**扩展现有功能
- **可以**优化性能

## 四、操作清单

### 修改代码前
- [ ] 查看 Context7 了解相关 API
- [ ] 检查 common 包是否有可复用功能
- [ ] 确认没有硬编码数值

### 修改代码后
- [ ] 运行 `make build` 确保编译通过
- [ ] 运行相关测试脚本
- [ ] 检查日志输出是否规范
- [ ] 确认没有重复代码
- [ ] 如果修改了 README.md，必须同步更新 README_CN.md

### 宣布成功前
- [ ] 必须有充分的证据
- [ ] 必须验证功能正常
- [ ] 必须确认无错误

## 五、常用示例

### 使用常量
```go
// 错误 ❌
limit := 1000
port := 3306

// 正确 ✅
limit := common.DefaultMaxRows
port := common.DefaultMySQLPort
```

### 错误处理
```go
// 错误 ❌
return errors.New("没有数据库配置")

// 正确 ✅
return common.NewNoConfigError("mysql")
```

### 日志记录
```go
// 错误 ❌
log.Info(ctx, "处理请求")  // 缺少必要的字段信息

// 正确 ✅
ctx = log.WithFields(ctx, 
    log.String(common.FieldTool, "query"),
    log.String(common.FieldProvider, "mysql"))
log.Info(ctx, "处理查询请求")

// 或直接传入字段
log.Info(ctx, "执行查询", 
    log.String(common.FieldSQL, sqlQuery),
    log.Int("limit", limit))
```

### 批量查询
```go
// 正确 ✅
results := common.BatchQuery(ctx, items, queryFunc)
```