package common

// 日志字段名常量定义
// 这些常量用于统一日志字段名，确保日志格式一致性
// 使用示例：
//   log.Info(ctx, "处理请求", log.String(common.FieldTool, "query"))
//   log.Error(ctx, "连接失败", log.String(common.FieldError, err.Error()))
const (
	// FieldTool 工具名称
	FieldTool = "tool"
	
	// FieldProvider 提供者名称
	FieldProvider = "provider"
	
	// FieldInstance 实例名称
	FieldInstance = "instance"
	
	// FieldDatabase 数据库名称
	FieldDatabase = "database"
	
	// FieldTable 表名
	FieldTable = "table"
	
	// FieldOperation 操作类型
	FieldOperation = "operation"
	
	// FieldSQL SQL语句
	FieldSQL = "sql"
	
	// FieldAffectedRows 影响行数
	FieldAffectedRows = "affected_rows"
	
	// FieldDuration 执行时间
	FieldDuration = "duration_ms"
	
	// FieldError 错误信息
	FieldError = "error"
	
	// FieldHost 主机地址
	FieldHost = "host"
	
	// FieldPort 端口号
	FieldPort = "port"
	
	// FieldUser 用户名
	FieldUser = "user"
	
	// FieldStatus 状态
	FieldStatus = "status"
	
	// FieldCount 数量
	FieldCount = "count"
	
	// FieldFields 字段列表
	FieldFields = "fields"
	
	// FieldQuery 查询条件
	FieldQuery = "query"
	
	// FieldResult 结果
	FieldResult = "result"
)

// 日志级别使用规范（仅作为指导，不是代码）
// - Debug: 详细的调试信息，生产环境应关闭
// - Info: 正常的操作信息，如工具调用、连接建立等
// - Warn: 可能的问题，但不影响功能，如连接重试、性能问题等
// - Error: 操作失败，需要处理，如查询失败、连接失败等