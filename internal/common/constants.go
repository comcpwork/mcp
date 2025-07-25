package common

// 查询相关常量
const (
	// DefaultQueryLimit 默认查询限制
	DefaultQueryLimit = 1000
	
	// MaxQueryLimit 最大查询限制
	MaxQueryLimit = 10000
	
	// DefaultMaxRows 默认最大行数
	DefaultMaxRows = 1000
	
	// MaxOutputLength 最大输出长度
	MaxOutputLength = 30000
	
	// DefaultTimeout 默认超时时间（秒）
	DefaultTimeout = 30
	
	// MaxTimeout 最大超时时间（秒）
	MaxTimeout = 300
)

// 并发相关常量
const (
	// MaxBatchConcurrency 批量查询最大并发数
	MaxBatchConcurrency = 5
	
	// DefaultBatchSize 默认批量大小
	DefaultBatchSize = 100
)

// 数据库相关常量
const (
	// DefaultMySQLPort MySQL默认端口
	DefaultMySQLPort = 3306
	
	// DefaultRedisPort Redis默认端口
	DefaultRedisPort = 6379
	
	// DefaultPulsarPort Pulsar默认端口
	DefaultPulsarPort = 8080
	
	// DefaultCharset 默认字符集
	DefaultCharset = "utf8mb4"
	
	// MaxRedisDatabase Redis最大数据库编号
	MaxRedisDatabase = 15
)

// 配置相关常量
const (
	// ConfigFileName 配置文件名模板
	ConfigFileName = "%s.yaml"
	
	// ConfigDirPath 配置目录路径
	ConfigDirPath = "~/.co-mcp"
	
	// LogDirPath 日志目录路径
	LogDirPath = "~/.co-mcp/logs"
	
	// DefaultInstanceName 默认实例名称
	DefaultInstanceName = "default"
)