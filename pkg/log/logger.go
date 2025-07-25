package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

// SimpleLogger 简单日志实现
type SimpleLogger struct {
	fields map[string]interface{}
	writer io.Writer
	level  LogLevel
	mu     sync.Mutex
}

// loggerState 日志器状态（避免全局变量）
type loggerState struct {
	writers    map[string]io.Writer
	writersMu  sync.RWMutex
	currentCmd string // 当前运行的命令
	cmdMu      sync.RWMutex
}

// 使用单例模式管理状态
var (
	state     *loggerState
	stateOnce sync.Once
)

// getState 获取日志器状态
func getState() *loggerState {
	stateOnce.Do(func() {
		state = &loggerState{
			writers: make(map[string]io.Writer),
		}
	})
	return state
}

// SetCommand 设置当前运行的命令（应该在程序启动时调用）
func SetCommand(cmd string) {
	s := getState()
	s.cmdMu.Lock()
	s.currentCmd = cmd
	s.cmdMu.Unlock()
}

// getCommand 获取当前命令
func getCommand() string {
	s := getState()
	s.cmdMu.RLock()
	defer s.cmdMu.RUnlock()
	if s.currentCmd == "" {
		return "mcp"
	}
	return s.currentCmd
}

// getLogWriter 根据命令获取日志写入器
func getLogWriter() io.Writer {
	cmd := getCommand()
	s := getState()

	s.writersMu.RLock()
	if writer, exists := s.writers[cmd]; exists {
		s.writersMu.RUnlock()
		return writer
	}
	s.writersMu.RUnlock()

	// 需要创建新的writer
	s.writersMu.Lock()
	defer s.writersMu.Unlock()

	// 双重检查
	if writer, exists := s.writers[cmd]; exists {
		return writer
	}

	// 创建日志目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		s.writers[cmd] = os.Stderr
		return os.Stderr
	}

	logDir := filepath.Join(homeDir, ".co-mcp")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		s.writers[cmd] = os.Stderr
		return os.Stderr
	}

	// 创建日志文件（根据命令名）
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", cmd))

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		s.writers[cmd] = os.Stderr
		return os.Stderr
	}

	// 只输出到文件，不输出到stderr（MCP服务器规范）
	s.writers[cmd] = file

	return file
}

// NewLogger 创建新的日志器
func NewLogger() *SimpleLogger {
	return &SimpleLogger{
		fields: make(map[string]interface{}),
		writer: getLogWriter(),
		level:  InfoLevel,
	}
}

// WithFields 添加字段（返回新的logger）
func (l *SimpleLogger) WithFields(fields map[string]interface{}) *SimpleLogger {
	newFields := make(map[string]interface{})

	// 复制原有字段
	for k, v := range l.fields {
		newFields[k] = v
	}

	// 添加新字段
	for k, v := range fields {
		newFields[k] = v
	}

	return &SimpleLogger{
		fields: newFields,
		writer: l.writer,
		level:  l.level,
	}
}

// log 统一日志输出
func (l *SimpleLogger) log(level LogLevel, msg string, fields ...Field) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// 基础日志格式
	logLine := fmt.Sprintf("[%s] %s: %s", timestamp, level.String(), msg)

	// 合并字段：先添加logger中的字段，再添加方法参数中的字段
	allFields := make(map[string]interface{})

	// 复制logger中的字段
	for k, v := range l.fields {
		allFields[k] = v
	}

	// 添加方法参数中的字段（会覆盖同名字段）
	for _, f := range fields {
		allFields[f.Key] = f.Value
	}

	// 输出字段
	if len(allFields) > 0 {
		logLine += " "
		first := true
		for k, v := range allFields {
			if !first {
				logLine += " "
			}
			logLine += fmt.Sprintf("%s=%v", k, v)
			first = false
		}
	}

	logLine += "\n"

	// 写入日志
	if _, err := l.writer.Write([]byte(logLine)); err != nil {
		// 如果写入失败，尝试写入stderr
		fmt.Fprintf(os.Stderr, "日志写入失败: %v\n", err)
		fmt.Fprint(os.Stderr, logLine)
	}
}

// 实现Logger接口
func (l *SimpleLogger) Debug(msg string, fields ...Field) {
	l.log(DebugLevel, msg, fields...)
}

func (l *SimpleLogger) Info(msg string, fields ...Field) {
	l.log(InfoLevel, msg, fields...)
}

func (l *SimpleLogger) Warn(msg string, fields ...Field) {
	l.log(WarnLevel, msg, fields...)
}

func (l *SimpleLogger) Error(msg string, fields ...Field) {
	l.log(ErrorLevel, msg, fields...)
}
