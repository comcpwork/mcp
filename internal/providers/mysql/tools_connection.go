package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"mcp/internal/common"
	"mcp/pkg/log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/viper"
)

// handleConnectMySQL 处理直接连接请求
func (s *MySQLServer) handleConnectMySQL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "connect_mysql"),
		log.String(common.FieldOperation, "connect"))
	
	log.Info(ctx, "处理MySQL连接请求")
	
	// 获取连接参数
	host := request.GetString("host", "localhost")
	port := request.GetInt("port", common.DefaultMySQLPort)
	user := request.GetString("user", "root")
	password := request.GetString("password", "")
	database := request.GetString("database", "")
	name := request.GetString("name", "") // 配置名称
	
	// 如果没有指定名称，生成一个
	if name == "" {
		name = fmt.Sprintf("%s_%d", host, port)
		// 如果已存在，添加序号
		if viper.IsSet(fmt.Sprintf("databases.%s", name)) {
			for i := 2; ; i++ {
				candidate := fmt.Sprintf("%s_%d", name, i)
				if !viper.IsSet(fmt.Sprintf("databases.%s", candidate)) {
					name = candidate
					break
				}
			}
		}
	}
	
	// 构建配置
	configKey := fmt.Sprintf("databases.%s", name)
	config := map[string]interface{}{
		"host":     host,
		"port":     port,
		"user":     user,
		"password": password,
		"database": database,
		"charset":  "utf8mb4",
		"max_connections": 10,
		"max_idle_connections": 5,
		"connection_timeout": "30s",
	}
	
	// 测试连接（懒加载方式）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, host, port)
	if database != "" {
		dsn += database
	}
	dsn += "?charset=utf8mb4&parseTime=true&loc=Local"
	
	log.Info(ctx, "测试数据库连接",
		log.String(common.FieldHost, host),
		log.Int(common.FieldPort, port),
		log.String(common.FieldUser, user),
		log.String(common.FieldDatabase, database))
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Error(ctx, "创建数据库连接失败",
			log.String(common.FieldError, err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create connection: %v", err)), nil
	}
	
	// 设置连接参数
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	
	// 测试连接
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		log.Error(ctx, "数据库连接测试失败",
			log.String(common.FieldError, err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Connection test failed: %v", err)), nil
	}
	
	// 获取数据库基本信息
	var version, charset, timezone string
	var currentDB sql.NullString
	
	// 版本信息
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		version = "unknown"
	}
	
	// 字符集
	if err := db.QueryRowContext(ctx, "SELECT @@character_set_server").Scan(&charset); err != nil {
		charset = "unknown"  
	}
	
	// 时区
	if err := db.QueryRowContext(ctx, "SELECT @@time_zone").Scan(&timezone); err != nil {
		timezone = "unknown"
	}
	
	// 当前数据库
	db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&currentDB)
	
	// 获取数据库列表
	var databases []string
	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var dbName string
			if err := rows.Scan(&dbName); err == nil {
				databases = append(databases, dbName)
			}
		}
	}
	
	// 关闭测试连接（实际使用时会重新创建）
	db.Close()
	
	// 保存配置
	for k, v := range config {
		viper.Set(fmt.Sprintf("%s.%s", configKey, k), v)
	}
	
	// 设置为激活数据库
	viper.Set("active_database", name)
	s.activeDatabase = name
	
	// 记录到历史
	s.addToHistory(name, config)
	
	// 保存配置文件
	if err := viper.WriteConfig(); err != nil {
		log.Error(ctx, "保存配置失败",
			log.String(common.FieldError, err.Error()))
	}
	
	// 清除缓存的连接（如果有）
	s.dbPool.CloseConnection(name)
	
	// 构建成功响应
	var result strings.Builder
	result.WriteString("✅ Successfully connected to MySQL\n")
	result.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	// 基本连接信息
	result.WriteString("📌 Connection Info:\n")
	result.WriteString(fmt.Sprintf("  Config Name: %s (active)\n", name))
	result.WriteString(fmt.Sprintf("  Host: %s:%d\n", host, port))
	result.WriteString(fmt.Sprintf("  User: %s\n", user))
	
	// 服务器信息
	result.WriteString("\n📊 Server Info:\n")
	result.WriteString(fmt.Sprintf("  Version: %s\n", version))
	result.WriteString(fmt.Sprintf("  Charset: %s\n", charset))
	result.WriteString(fmt.Sprintf("  Timezone: %s\n", timezone))
	
	// 当前数据库
	if currentDB.Valid && currentDB.String != "" {
		result.WriteString(fmt.Sprintf("  Current Database: %s\n", currentDB.String))
	} else if database != "" {
		result.WriteString(fmt.Sprintf("  Default Database: %s\n", database))
	}
	
	// 可用数据库列表
	if len(databases) > 0 {
		result.WriteString(fmt.Sprintf("\n📁 Available Databases (%d):\n", len(databases)))
		// 过滤系统数据库，显示前10个
		userDbs := []string{}
		systemDbs := map[string]bool{
			"information_schema": true,
			"mysql":             true,
			"performance_schema": true,
			"sys":               true,
		}
		
		for _, db := range databases {
			if !systemDbs[db] {
				userDbs = append(userDbs, db)
			}
		}
		
		// 显示用户数据库
		for i, db := range userDbs {
			if i < 10 {
				result.WriteString(fmt.Sprintf("  - %s\n", db))
			}
		}
		
		if len(userDbs) > 10 {
			result.WriteString(fmt.Sprintf("  ... and %d more\n", len(userDbs)-10))
		}
		
		// 系统数据库数量
		systemCount := len(databases) - len(userDbs)
		if systemCount > 0 {
			result.WriteString(fmt.Sprintf("\n  System databases: %d\n", systemCount))
		}
	}
	
	result.WriteString("\n💡 Note: Connection will be established when needed (lazy loading)")
	
	log.Info(ctx, "MySQL连接配置成功",
		log.String(common.FieldStatus, "success"),
		log.String(common.FieldInstance, name),
		log.String("version", version))
	
	return mcp.NewToolResultText(result.String()), nil
}

// handleCurrentMySQL 查看当前MySQL配置
func (s *MySQLServer) handleCurrentMySQL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "current_mysql"),
		log.String(common.FieldOperation, "get_current"))
	
	log.Info(ctx, "查看当前MySQL配置")
	
	activeDB := viper.GetString("active_database")
	if activeDB == "" {
		activeDB = common.DefaultInstanceName
	}
	
	// 获取配置
	configKey := fmt.Sprintf("databases.%s", activeDB)
	if !viper.IsSet(configKey) {
		return mcp.NewToolResultText("No active MySQL configuration"), nil
	}
	
	// 构建输出
	var result strings.Builder
	result.WriteString("📊 Current MySQL Configuration\n")
	result.WriteString(strings.Repeat("=", 40) + "\n\n")
	
	result.WriteString(fmt.Sprintf("Active: %s\n", activeDB))
	
	// 检查连接状态
	isConnected := false
	if db, err := s.dbPool.GetConnection(activeDB); err == nil && db != nil {
		if err := db.Ping(); err == nil {
			isConnected = true
		}
	}
	
	result.WriteString(fmt.Sprintf("Status: %s\n", func() string {
		if isConnected {
			return "🟢 Connected"
		}
		return "⚪ Not connected (will connect on use)"
	}()))
	
	// 显示配置详情
	host := viper.GetString(configKey + ".host")
	port := viper.GetInt(configKey + ".port")
	user := viper.GetString(configKey + ".user")
	database := viper.GetString(configKey + ".database")
	
	result.WriteString(fmt.Sprintf("Host: %s:%d\n", host, port))
	result.WriteString(fmt.Sprintf("User: %s\n", user))
	if database != "" {
		result.WriteString(fmt.Sprintf("Database: %s\n", database))
	}
	
	// 显示历史使用信息
	if history := s.getHistoryInfo(activeDB); history != nil {
		result.WriteString(fmt.Sprintf("Last Used: %s\n", history.LastUsed.Format("2006-01-02 15:04:05")))
		result.WriteString(fmt.Sprintf("Use Count: %d times\n", history.UseCount))
	}
	
	return mcp.NewToolResultText(result.String()), nil
}

// handleHistoryMySQL 查看MySQL连接历史
func (s *MySQLServer) handleHistoryMySQL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "history_mysql"),
		log.String(common.FieldOperation, "list_history"))
	
	log.Info(ctx, "查看MySQL连接历史")
	
	// 获取所有数据库配置
	databases := viper.GetStringMap("databases")
	if len(databases) == 0 {
		return mcp.NewToolResultText("No MySQL configuration history"), nil
	}
	
	activeDB := viper.GetString("active_database")
	if activeDB == "" {
		activeDB = common.DefaultInstanceName
	}
	
	// 获取历史信息并排序
	type dbInfo struct {
		name     string
		config   map[string]interface{}
		history  *historyInfo
		isActive bool
	}
	
	var dbList []dbInfo
	for name, cfg := range databases {
		if configMap, ok := cfg.(map[string]interface{}); ok {
			info := dbInfo{
				name:     name,
				config:   configMap,
				history:  s.getHistoryInfo(name),
				isActive: name == activeDB,
			}
			dbList = append(dbList, info)
		}
	}
	
	// 按最后使用时间排序
	for i := 0; i < len(dbList)-1; i++ {
		for j := i + 1; j < len(dbList); j++ {
			if dbList[i].history != nil && dbList[j].history != nil {
				if dbList[i].history.LastUsed.Before(dbList[j].history.LastUsed) {
					dbList[i], dbList[j] = dbList[j], dbList[i]
				}
			}
		}
	}
	
	// 构建输出
	var result strings.Builder
	result.WriteString("📚 MySQL Configuration History\n")
	result.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	for i, db := range dbList {
		if i > 0 {
			result.WriteString("\n" + strings.Repeat("-", 30) + "\n\n")
		}
		
		status := ""
		if db.isActive {
			status = " [ACTIVE]"
		}
		
		result.WriteString(fmt.Sprintf("[%d] %s%s\n", i+1, db.name, status))
		
		host := getString(db.config, "host", "localhost")
		port := getInt(db.config, "port", 3306)
		user := getString(db.config, "user", "")
		database := getString(db.config, "database", "")
		
		result.WriteString(fmt.Sprintf("    Host: %s:%d\n", host, port))
		result.WriteString(fmt.Sprintf("    User: %s\n", user))
		if database != "" {
			result.WriteString(fmt.Sprintf("    Database: %s\n", database))
		}
		
		if db.history != nil {
			result.WriteString(fmt.Sprintf("    Last Used: %s\n", db.history.LastUsed.Format("2006-01-02 15:04:05")))
			result.WriteString(fmt.Sprintf("    Use Count: %d times\n", db.history.UseCount))
		}
		
		// 最多显示10条
		if i >= 9 {
			remaining := len(dbList) - 10
			if remaining > 0 {
				result.WriteString(fmt.Sprintf("\n... and %d more configurations\n", remaining))
			}
			break
		}
	}
	
	result.WriteString("\nTip: Use 'switch_mysql' to switch between configurations")
	
	return mcp.NewToolResultText(result.String()), nil
}

// handleSwitchMySQL 切换到其他MySQL配置
func (s *MySQLServer) handleSwitchMySQL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "mysql"),
		log.String(common.FieldTool, "switch_mysql"),
		log.String(common.FieldOperation, "switch"))
	
	name := request.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("Please provide configuration name"), nil
	}
	
	log.Info(ctx, "切换MySQL配置",
		log.String("target", name))
	
	// 检查配置是否存在
	configKey := fmt.Sprintf("databases.%s", name)
	if !viper.IsSet(configKey) {
		// 尝试按索引查找
		if idx := request.GetInt("name", 0); idx > 0 {
			databases := viper.GetStringMap("databases")
			i := 1
			for n := range databases {
				if i == idx {
					name = n
					configKey = fmt.Sprintf("databases.%s", name)
					break
				}
				i++
			}
		}
		
		if !viper.IsSet(configKey) {
			return mcp.NewToolResultError(fmt.Sprintf("Configuration '%s' not found", name)), nil
		}
	}
	
	// 设置为激活数据库
	viper.Set("active_database", name)
	oldActive := s.activeDatabase
	s.activeDatabase = name
	
	// 关闭旧连接
	if oldActive != "" && oldActive != name {
		s.dbPool.CloseConnection(oldActive)
	}
	
	// 更新历史
	s.updateHistory(name)
	
	// 保存配置
	if err := viper.WriteConfig(); err != nil {
		log.Error(ctx, "保存配置失败",
			log.String(common.FieldError, err.Error()))
	}
	
	// 获取配置信息
	host := viper.GetString(configKey + ".host")
	port := viper.GetInt(configKey + ".port")
	user := viper.GetString(configKey + ".user")
	database := viper.GetString(configKey + ".database")
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("✅ Switched to MySQL configuration: %s\n\n", name))
	result.WriteString(fmt.Sprintf("Host: %s:%d\n", host, port))
	result.WriteString(fmt.Sprintf("User: %s\n", user))
	if database != "" {
		result.WriteString(fmt.Sprintf("Database: %s\n", database))
	}
	
	log.Info(ctx, "MySQL配置切换成功",
		log.String(common.FieldStatus, "success"),
		log.String(common.FieldInstance, name))
	
	return mcp.NewToolResultText(result.String()), nil
}

// 历史信息结构
type historyInfo struct {
	LastUsed time.Time `yaml:"last_used"`
	UseCount int       `yaml:"use_count"`
}

// addToHistory 添加到历史记录
func (s *MySQLServer) addToHistory(name string, config map[string]interface{}) {
	history := &historyInfo{
		LastUsed: time.Now(),
		UseCount: 1,
	}
	
	// 读取现有历史
	key := fmt.Sprintf("history.mysql.%s", name)
	if viper.IsSet(key) {
		var existing historyInfo
		if err := viper.UnmarshalKey(key, &existing); err == nil {
			history.UseCount = existing.UseCount + 1
		}
	}
	
	// 保存历史
	viper.Set(key+".last_used", history.LastUsed)
	viper.Set(key+".use_count", history.UseCount)
}

// updateHistory 更新历史记录
func (s *MySQLServer) updateHistory(name string) {
	key := fmt.Sprintf("history.mysql.%s", name)
	
	useCount := viper.GetInt(key + ".use_count")
	viper.Set(key+".last_used", time.Now())
	viper.Set(key+".use_count", useCount+1)
}

// getHistoryInfo 获取历史信息
func (s *MySQLServer) getHistoryInfo(name string) *historyInfo {
	key := fmt.Sprintf("history.mysql.%s", name)
	if !viper.IsSet(key) {
		return nil
	}
	
	var info historyInfo
	if err := viper.UnmarshalKey(key, &info); err != nil {
		return nil
	}
	
	return &info
}

// 辅助函数
func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return defaultValue
}