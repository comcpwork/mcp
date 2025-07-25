// Package log 提供基于context的结构化日志功能
//
// 基本用法:
//
//	ctx = log.WithFields(ctx, log.String("user", "john"), log.Int("age", 30))
//	logger := log.Get(ctx)
//	logger.Info("用户登录成功")
//
// 便捷用法:
//
//	log.Info(ctx, "用户登录成功")
//
// 日志文件保存在: ~/.co-mcp/mcp-YYYY-MM-DD.log
package log

// 重新导出核心类型，方便外部使用
type (
	// Level 日志级别
	Level = LogLevel
)

// 导出日志级别常量
const (
	LevelDebug = DebugLevel
	LevelInfo  = InfoLevel
	LevelWarn  = WarnLevel
	LevelError = ErrorLevel
)
