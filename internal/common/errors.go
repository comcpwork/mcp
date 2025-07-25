package common

import (
	"fmt"
	"github.com/cockroachdb/errors"
)

// 通用错误定义
var (
	// ErrNoConfig 没有配置错误
	ErrNoConfig = errors.New("no configuration found")
	
	// ErrInstanceNotFound 实例不存在错误
	ErrInstanceNotFound = errors.New("instance not found")
	
	// ErrConnectionFailed 连接失败错误
	ErrConnectionFailed = errors.New("connection failed")
	
	// ErrInvalidParameter 参数无效错误
	ErrInvalidParameter = errors.New("invalid parameter")
	
	// ErrOperationNotSupported 操作不支持错误
	ErrOperationNotSupported = errors.New("operation not supported")
	
	// ErrQueryFailed 查询失败错误
	ErrQueryFailed = errors.New("query failed")
	
	// ErrTimeout 超时错误
	ErrTimeout = errors.New("operation timeout")
)

// ProviderError 提供者特定错误
type ProviderError struct {
	Provider string
	Type     string
	Message  string
	Cause    error
}

// Error 实现 error 接口
func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s %s: %s (caused by: %v)", e.Provider, e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s %s: %s", e.Provider, e.Type, e.Message)
}

// Unwrap 实现错误解包
func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// NewNoConfigError 创建没有配置错误
func NewNoConfigError(provider string) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Type:     "NoConfig",
		Message:  fmt.Sprintf("No %s configuration found, please use add_%s tool to add configuration", provider, provider),
		Cause:    ErrNoConfig,
	}
}

// NewInstanceNotFoundError 创建实例不存在错误
func NewInstanceNotFoundError(provider, instance string) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Type:     "InstanceNotFound",
		Message:  fmt.Sprintf("%s instance '%s' does not exist", provider, instance),
		Cause:    ErrInstanceNotFound,
	}
}

// NewConnectionError 创建连接错误
func NewConnectionError(provider string, err error) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Type:     "ConnectionFailed",
		Message:  "Failed to establish connection",
		Cause:    errors.Wrap(err, "connection error"),
	}
}

// NewInvalidParameterError 创建参数无效错误
func NewInvalidParameterError(provider, param, reason string) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Type:     "InvalidParameter",
		Message:  fmt.Sprintf("Invalid parameter '%s': %s", param, reason),
		Cause:    ErrInvalidParameter,
	}
}

// NewQueryError 创建查询错误
func NewQueryError(provider string, err error) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Type:     "QueryFailed",
		Message:  "Query execution failed",
		Cause:    errors.Wrap(err, "query error"),
	}
}

// NewInvalidConfigError 创建无效配置错误
func NewInvalidConfigError(provider string, details string) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Type:     "InvalidConfig",
		Message:  fmt.Sprintf("Invalid %s configuration: %s", provider, details),
		Cause:    errors.New("invalid configuration"),
	}
}

// IsNoConfigError 检查是否是没有配置错误
func IsNoConfigError(err error) bool {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Type == "NoConfig"
	}
	return errors.Is(err, ErrNoConfig)
}

// IsConnectionError 检查是否是连接错误
func IsConnectionError(err error) bool {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Type == "ConnectionFailed"
	}
	return errors.Is(err, ErrConnectionFailed)
}

// FormatErrorMessage 格式化错误消息（用于返回给用户）
func FormatErrorMessage(err error) string {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Message
	}
	return err.Error()
}