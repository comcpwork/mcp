package log

import (
	"context"
	"testing"
)

// 这个文件演示如何使用日志系统
func Example() {
	ctx := context.Background()

	// 方式1：使用WithFields添加字段到context
	ctx = WithFields(ctx,
		String("service", "mysql"),
		String("version", "1.0.0"),
	)

	logger := Get(ctx)
	logger.Info("服务启动")

	// 方式2：直接在日志方法中传入字段
	logger.Info("用户连接成功",
		String("user", "admin"),
		Int("connection_id", 12345),
	)

	// 方式3：便捷函数直接传入字段
	Info(ctx, "处理查询请求", String("query", "SELECT * FROM users"))

	// 方式4：混合使用context字段和方法字段
	ctx = WithFields(ctx, String("module", "auth"))
	Error(ctx, "认证失败", String("reason", "invalid_token"))
}

func TestLoggerCreation(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Error("NewLogger() 返回 nil")
	}
}

func TestFieldCreation(t *testing.T) {
	tests := []struct {
		name   string
		field  Field
		expect string
	}{
		{"string", String("key", "value"), "key"},
		{"int", Int("count", 42), "count"},
		{"bool", Bool("active", true), "active"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != tt.expect {
				t.Errorf("期望 key=%s, 实际 key=%s", tt.expect, tt.field.Key)
			}
		})
	}
}
