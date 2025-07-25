package log

import "context"

// 日志context key
type loggerKey struct{}

// WithFields 向context添加日志字段
func WithFields(ctx context.Context, fields ...Field) context.Context {
	// 从context获取现有logger
	var logger *SimpleLogger
	if l, ok := ctx.Value(loggerKey{}).(*SimpleLogger); ok {
		logger = l
	} else {
		logger = NewLogger()
	}

	// 转换字段格式
	fieldMap := make(map[string]interface{})
	for _, f := range fields {
		fieldMap[f.Key] = f.Value
	}

	// 创建新的logger并放入context
	newLogger := logger.WithFields(fieldMap)
	return context.WithValue(ctx, loggerKey{}, newLogger)
}

// Get 从context获取logger
func Get(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*SimpleLogger); ok {
		return logger
	}
	return NewLogger()
}

// 便捷函数，直接从context记录日志
func Debug(ctx context.Context, msg string, fields ...Field) {
	Get(ctx).Debug(msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...Field) {
	Get(ctx).Info(msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...Field) {
	Get(ctx).Warn(msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...Field) {
	Get(ctx).Error(msg, fields...)
}
