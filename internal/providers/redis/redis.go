package redis

import (
	"context"
	"fmt"
	"mcp/internal/common"
	"mcp/internal/config"
	"mcp/internal/pool"
	"mcp/internal/server"
	"mcp/pkg/log"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// 使用 common 包的错误，不再定义本地错误

// RedisServer Redis MCP服务器
type RedisServer struct {
	*server.BaseServer
	config      *config.Config
	activeRedis string           // 当前激活的Redis实例名称
	redisPool   *pool.RedisPool  // Redis连接池
	
	// 细分的安全选项 (对于Redis，某些选项可能不适用，保持一致性)
	disableCreate   bool // 对Redis不太适用，但保持接口一致
	disableDrop     bool // 禁用FLUSHDB/FLUSHALL等清空操作
	disableAlter    bool // 禁用CONFIG SET等配置修改操作
	disableTruncate bool // 对Redis不太适用，但保持接口一致
	disableUpdate   bool // 禁用SET/HSET等更新操作
	disableDelete   bool // 禁用DEL/HDEL等删除操作
}

// NewServer 创建Redis服务器
func NewServer() server.MCPServer {
	return &RedisServer{
		BaseServer: server.NewBaseServer("Redis MCP Server", "1.0.0"),
		// 默认开放所有权限
		disableCreate:   false,
		disableDrop:     false,
		disableAlter:    false,
		disableTruncate: false,
		disableUpdate:   false,
		disableDelete:   false,
	}
}

// SetSecurityOptions 设置细分的安全选项
func (s *RedisServer) SetSecurityOptions(disableCreate, disableDrop, disableAlter, disableTruncate, disableUpdate, disableDelete bool) {
	s.disableCreate = disableCreate
	s.disableDrop = disableDrop
	s.disableAlter = disableAlter
	s.disableTruncate = disableTruncate
	s.disableUpdate = disableUpdate
	s.disableDelete = disableDelete
}

// Init 初始化服务器
func (s *RedisServer) Init(ctx context.Context) error {
	// 初始化配置
	s.config = config.NewConfig("redis")

	// 使用自定义的默认配置
	if err := s.initConfig(ctx); err != nil {
		return errors.Wrap(err, "初始化配置失败")
	}

	// 初始化Redis连接池（懒加载）
	s.redisPool = pool.NewRedisPool("redis")

	// 不在启动时初始化Redis连接，使用懒加载机制
	log.Info(ctx, "使用懒加载机制，Redis连接将在需要时创建",
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "init"))

	// 初始化MCP服务器
	s.InitMCPServer(
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(true, false),
	)

	log.Info(ctx, "Redis服务器初始化完成",
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "init"),
		log.String(common.FieldStatus, "success"))
	return nil
}

// initConfig 初始化配置（使用自定义默认配置）
func (s *RedisServer) initConfig(ctx context.Context) error {
	// 确保配置目录存在
	configPath := s.config.GetConfigPath()

	// 如果配置文件不存在，写入默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(configPath, []byte(DefaultConfig), 0644); err != nil {
			return err
		}
		log.Info(ctx, "创建Redis配置文件",
			log.String("path", configPath),
			log.String(common.FieldProvider, "redis"),
			log.String(common.FieldOperation, "create_config"))
	}

	// 加载配置
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}

// getConnection 获取Redis连接（懒加载）
func (s *RedisServer) getConnection(ctx context.Context) (*redis.Client, error) {
	// 检查是否有Redis配置 - 适配现有的配置格式
	redisInstances := viper.GetStringMap("databases")
	if len(redisInstances) == 0 {
		return nil, common.NewNoConfigError("redis")
	}

	// 设置激活Redis实例名称 - 使用 active_database 字段
	s.activeRedis = viper.GetString("active_database")
	if s.activeRedis == "" {
		s.activeRedis = "default"
	}

	// 获取对应的Redis配置 - 使用 databases 格式
	redisKey := fmt.Sprintf("databases.%s", s.activeRedis)
	if !viper.IsSet(redisKey) {
		return nil, errors.Newf("Redis配置 '%s' 不存在", s.activeRedis)
	}

	// 使用连接池获取连接
	client, err := s.redisPool.GetConnection(ctx, s.activeRedis)
	if err != nil {
		return nil, err
	}

	log.Info(ctx, "Redis连接成功",
		log.String("name", s.activeRedis),
		log.String("host", viper.GetString(redisKey+".host")),
		log.Int("port", viper.GetInt(redisKey+".port")),
		log.Int("database", viper.GetInt(redisKey+".database")),
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "connect"),
		log.String(common.FieldStatus, "success"),
	)

	return client, nil
}

// RegisterTools 注册工具
func (s *RedisServer) RegisterTools() error {
	srv := s.GetServer()

	// Redis命令执行工具（类似MySQL的query工具）
	srv.AddTool(
		mcp.NewTool("exec",
			mcp.WithDescription("Execute Redis commands (supports single commands and pipeline with | or ; separator)"),
			mcp.WithString("command",
				mcp.Required(),
				mcp.Description("Redis command to execute. Examples: 'GET mykey', 'SET mykey value', 'HGETALL myhash'. For multiple commands use pipeline: 'SET key1 value1 | GET key1 | DEL key1' or 'SET key2 value2; GET key2; DEL key2'"),
			),
		),
		s.handleExec,
	)

	// 配置更新管理工具
	srv.AddTool(
		mcp.NewTool("update_config",
			mcp.WithDescription("Update Redis instance configuration properties"),
			mcp.WithString("name",
				mcp.Description("Redis instance name to update (defaults to active instance)"),
			),
			mcp.WithString("host",
				mcp.Description("Update Redis server host address"),
			),
			mcp.WithNumber("port",
				mcp.Description("Update Redis server port"),
			),
			mcp.WithString("password",
				mcp.Description("Update Redis password"),
			),
			mcp.WithNumber("database",
				mcp.Description(fmt.Sprintf("Update Redis database number (0-%d)", common.MaxRedisDatabase)),
			),
		),
		s.handleUpdateConfig,
	)

	srv.AddTool(
		mcp.NewTool("get_config_details",
			mcp.WithDescription("Get detailed configuration information for Redis instances"),
			mcp.WithString("name",
				mcp.Description("Redis instance name (defaults to active instance, 'all' for all instances)"),
			),
			mcp.WithBoolean("include_sensitive",
				mcp.Description("Whether to include sensitive information like passwords (default: false)"),
			),
		),
		s.handleGetConfigDetails,
	)

	// 新的连接管理工具
	srv.AddTool(
		mcp.NewTool("connect_redis",
			mcp.WithDescription("Connect to Redis server"),
			mcp.WithString("host",
				mcp.Description("Redis server host address (default: localhost)"),
			),
			mcp.WithNumber("port",
				mcp.Description(fmt.Sprintf("Redis server port (default: %d)", common.DefaultRedisPort)),
			),
			mcp.WithString("password",
				mcp.Description("Redis password"),
			),
			mcp.WithNumber("database",
				mcp.Description(fmt.Sprintf("Redis database number (default: 0, range: 0-%d)", common.MaxRedisDatabase)),
			),
			mcp.WithString("name",
				mcp.Description("Configuration name (auto-generated if not provided)"),
			),
		),
		s.handleConnectRedis,
	)

	srv.AddTool(
		mcp.NewTool("current_redis",
			mcp.WithDescription("Show current Redis connection information"),
		),
		s.handleCurrentRedis,
	)

	srv.AddTool(
		mcp.NewTool("history_redis",
			mcp.WithDescription("Show Redis connection history"),
		),
		s.handleHistoryRedis,
	)

	srv.AddTool(
		mcp.NewTool("switch_redis",
			mcp.WithDescription("Switch to another Redis configuration"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Configuration name or index number"),
			),
		),
		s.handleSwitchRedis,
	)

	return nil
}

// RegisterResources 注册资源
func (s *RedisServer) RegisterResources() error {
	// 暂时不注册资源，等待API稳定
	return nil
}

// Start 启动服务器
func (s *RedisServer) Start(ctx context.Context, transport string) error {
	// 启动逻辑已在registry中实现
	return nil
}