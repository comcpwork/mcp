package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"mcp/internal/common"
	"mcp/internal/config"
	"mcp/internal/pool"
	"mcp/internal/server"
	"mcp/pkg/log"
	"os"
	"path/filepath"
	"time"

	"github.com/cockroachdb/errors"
	_ "github.com/go-sql-driver/mysql" // MySQL驱动
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/viper"
)

// MySQL 相关错误定义已移至 common.errors
// 使用 common.NewNoConfigError("mysql") 替代 ErrNoDatabaseConfig

// MySQLServer MySQL MCP服务器
type MySQLServer struct {
	*server.BaseServer
	db             *sql.DB
	config         *config.Config
	activeDatabase string // 当前激活的数据库名称
	dbPool         *pool.DatabasePool // 数据库连接池
	
	// 细分的安全选项
	disableCreate   bool // 禁用CREATE操作
	disableDrop     bool // 禁用DROP操作
	disableAlter    bool // 禁用ALTER操作
	disableTruncate bool // 禁用TRUNCATE操作
	disableUpdate   bool // 禁用UPDATE操作
	disableDelete   bool // 禁用DELETE操作
}

// NewServer 创建MySQL服务器
func NewServer() server.MCPServer {
	return &MySQLServer{
		BaseServer: server.NewBaseServer("MySQL MCP Server", "1.0.0"),
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
func (s *MySQLServer) SetSecurityOptions(disableCreate, disableDrop, disableAlter, disableTruncate, disableUpdate, disableDelete bool) {
	s.disableCreate = disableCreate
	s.disableDrop = disableDrop
	s.disableAlter = disableAlter
	s.disableTruncate = disableTruncate
	s.disableUpdate = disableUpdate
	s.disableDelete = disableDelete
}

// Init 初始化服务器
func (s *MySQLServer) Init(ctx context.Context) error {
	// 初始化配置
	s.config = config.NewConfig("mysql")

	// 使用自定义的默认配置
	if err := s.initConfig(ctx); err != nil {
		return errors.Wrap(err, "初始化配置失败")
	}

	// 初始化数据库连接池（懒加载）
	s.dbPool = pool.NewDatabasePool("mysql")

	// 不在启动时初始化数据库连接，使用懒加载机制
	log.Info(ctx, "使用懒加载机制，数据库连接将在需要时创建",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldOperation, "init"))

	// 初始化MCP服务器
	s.InitMCPServer(
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(true, false),
	)

	log.Info(ctx, "MySQL服务器初始化完成",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldOperation, "init"),
		log.String(common.FieldStatus, "success"))
	return nil
}

// initConfig 初始化配置（使用自定义默认配置）
func (s *MySQLServer) initConfig(ctx context.Context) error {
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
		log.Info(ctx, "创建MySQL配置文件",
			log.String(common.FieldProvider, "mysql"),
			log.String(common.FieldOperation, "create_config"),
			log.String("path", configPath))
	}

	// 加载配置
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}

// getConnection 获取数据库连接（懒加载）
func (s *MySQLServer) getConnection(ctx context.Context) (*sql.DB, error) {
	// 检查是否有数据库配置
	databases := viper.GetStringMap("databases")
	if len(databases) == 0 {
		return nil, common.NewNoConfigError("mysql")
	}
	
	// 设置激活数据库名称（但不连接）
	s.activeDatabase = viper.GetString("active_database")
	if s.activeDatabase == "" {
		s.activeDatabase = "default"
	}

	// 获取对应的数据库配置
	dbKey := fmt.Sprintf("databases.%s", s.activeDatabase)
	if !viper.IsSet(dbKey) {
		return nil, errors.Newf("数据库配置 '%s' 不存在", s.activeDatabase)
	}

	// 获取连接超时设置
	connectionTimeout := viper.GetString(dbKey + ".connection_timeout")
	if connectionTimeout == "" {
		connectionTimeout = "5s" // 默认 5 秒超时
	}
	
	// 构建DSN（包含连接超时）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&timeout=%s",
		viper.GetString(dbKey+".user"),
		viper.GetString(dbKey+".password"),
		viper.GetString(dbKey+".host"),
		viper.GetInt(dbKey+".port"),
		viper.GetString(dbKey+".database"),
		viper.GetString(dbKey+".charset"),
		connectionTimeout,
	)

	// 打开数据库连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// 设置连接池参数
	db.SetMaxOpenConns(viper.GetInt(dbKey + ".max_connections"))
	db.SetMaxIdleConns(viper.GetInt(dbKey + ".max_idle_connections"))

	// 测试连接（使用超时上下文）
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, errors.Wrap(err, "连接数据库失败")
	}

	log.Info(ctx, "数据库连接成功",
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldOperation, "connect"),
		log.String(common.FieldStatus, "success"),
		log.String("name", s.activeDatabase),
		log.String(common.FieldHost, viper.GetString(dbKey+".host")),
		log.Int(common.FieldPort, viper.GetInt(dbKey+".port")),
		log.String(common.FieldDatabase, viper.GetString(dbKey+".database")),
	)

	return db, nil
}

// RegisterTools 注册工具
func (s *MySQLServer) RegisterTools() error {
	srv := s.GetServer()

	// SQL执行工具
	srv.AddTool(
		mcp.NewTool("exec",
			mcp.WithDescription("Execute SQL statements (SELECT, INSERT, UPDATE, DELETE, etc.)"),
			mcp.WithString("sql",
				mcp.Required(),
				mcp.Description("SQL statement to execute"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Limit the number of returned rows (for SELECT queries)"),
				mcp.DefaultNumber(float64(viper.GetInt("tools.query.max_rows"))),
			),
		),
		s.handleExec,
	)

	// Schema工具
	srv.AddTool(
		mcp.NewTool("show_tables",
			mcp.WithDescription("Show all tables in database with detailed information (comments, engine, row count, etc.)"),
			mcp.WithString("database",
				mcp.Description("Database name (optional, defaults to current active database)"),
			),
			mcp.WithBoolean("exact_count",
				mcp.Description("Whether to get exact row count (default false, uses estimate, true may impact performance)"),
			),
		),
		s.handleShowTables,
	)

	srv.AddTool(
		mcp.NewTool("describe_table",
			mcp.WithDescription("Show table structure with column details and comments"),
			mcp.WithString("table",
				mcp.Required(),
				mcp.Description("Table name"),
			),
		),
		s.handleDescribeTable,
	)

	srv.AddTool(
		mcp.NewTool("describe_tables",
			mcp.WithDescription("Show structure information for multiple tables with column details and comments"),
			mcp.WithString("tables",
				mcp.Required(),
				mcp.Description("Comma-separated table names, e.g.: 'tb_admin,tb_user,tb_product'"),
			),
			mcp.WithBoolean("include_indexes",
				mcp.Description("Whether to include index information (default false)"),
			),
			mcp.WithBoolean("include_foreign_keys",
				mcp.Description("Whether to include foreign key information (default false)"),
			),
		),
		s.handleDescribeTables,
	)

	// 配置更新管理工具
	srv.AddTool(
		mcp.NewTool("update_config",
			mcp.WithDescription("Update MySQL database configuration properties"),
			mcp.WithString("name",
				mcp.Description("Database configuration name to update (defaults to active database)"),
			),
			mcp.WithString("host",
				mcp.Description("Update database host address"),
			),
			mcp.WithNumber("port",
				mcp.Description("Update database port"),
			),
			mcp.WithString("user",
				mcp.Description("Update database username"),
			),
			mcp.WithString("password",
				mcp.Description("Update database password"),
			),
			mcp.WithString("database",
				mcp.Description("Update default database name"),
			),
			mcp.WithString("charset",
				mcp.Description("Update character set"),
			),
		),
		s.handleUpdateConfig,
	)

	srv.AddTool(
		mcp.NewTool("get_config_details",
			mcp.WithDescription("Get detailed configuration information for MySQL databases"),
			mcp.WithString("name",
				mcp.Description("Database configuration name (defaults to active database, 'all' for all databases)"),
			),
			mcp.WithBoolean("include_sensitive",
				mcp.Description("Whether to include sensitive information like passwords (default: false)"),
			),
		),
		s.handleGetConfigDetails,
	)

	// 新的连接管理工具
	srv.AddTool(
		mcp.NewTool("connect_mysql",
			mcp.WithDescription("Connect to MySQL database server"),
			mcp.WithString("host",
				mcp.Description("Database host address (default: localhost)"),
			),
			mcp.WithNumber("port",
				mcp.Description("Database port (default: 3306)"),
			),
			mcp.WithString("user",
				mcp.Description("Database username (default: root)"),
			),
			mcp.WithString("password",
				mcp.Description("Database password"),
			),
			mcp.WithString("database",
				mcp.Description("Default database name"),
			),
			mcp.WithString("name",
				mcp.Description("Configuration name (auto-generated if not provided)"),
			),
		),
		s.handleConnectMySQL,
	)

	srv.AddTool(
		mcp.NewTool("current_mysql",
			mcp.WithDescription("Show current MySQL connection information"),
		),
		s.handleCurrentMySQL,
	)

	srv.AddTool(
		mcp.NewTool("history_mysql",
			mcp.WithDescription("Show MySQL connection history"),
		),
		s.handleHistoryMySQL,
	)

	srv.AddTool(
		mcp.NewTool("switch_mysql",
			mcp.WithDescription("Switch to another MySQL configuration"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Configuration name or index number"),
			),
		),
		s.handleSwitchMySQL,
	)

	return nil
}

// RegisterResources 注册资源
func (s *MySQLServer) RegisterResources() error {
	srv := s.GetServer()

	// 表资源
	srv.AddResource(
		mcp.NewResource(
			"tables://list",
			"数据库表列表",
			mcp.WithResourceDescription("获取所有数据库表的列表"),
			mcp.WithMIMEType("application/json"),
		),
		s.handleTablesResource,
	)

	return nil
}

// Start 启动服务器
func (s *MySQLServer) Start(ctx context.Context, transport string) error {
	// 启动逻辑已在registry中实现
	return nil
}
