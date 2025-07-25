package redis

import (
	"context"
	"fmt"
	"mcp/internal/common"
	"mcp/pkg/log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// handleConnectRedis 处理直接连接请求
func (s *RedisServer) handleConnectRedis(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldTool, "connect_redis"),
		log.String(common.FieldOperation, "connect"))
	
	log.Info(ctx, "处理Redis连接请求")
	
	// 获取连接参数
	host := request.GetString("host", "localhost")
	port := request.GetInt("port", common.DefaultRedisPort)
	password := request.GetString("password", "")
	database := request.GetInt("database", 0)
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
		"password": password,
		"database": database,
		"connection_timeout": "5s",
		"read_timeout": "3s",
		"write_timeout": "3s",
		"max_connections": 10,
		"max_idle_connections": 5,
		"max_idle_time": "300s",
	}
	
	// 测试连接
	log.Info(ctx, "测试Redis连接",
		log.String(common.FieldHost, host),
		log.Int(common.FieldPort, port),
		log.Int(common.FieldDatabase, database))
	
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           database,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MaxIdleConns: 5,
		ConnMaxIdleTime: 5 * time.Minute,
	})
	
	// 测试连接
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	if err := client.Ping(testCtx).Err(); err != nil {
		client.Close()
		log.Error(ctx, "Redis连接测试失败",
			log.String(common.FieldError, err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Connection test failed: %v", err)), nil
	}
	
	// 获取Redis信息
	info, err := client.Info(testCtx, "server").Result()
	if err != nil {
		info = "unknown"
	}
	
	// 解析版本信息
	var version string
	for _, line := range strings.Split(info, "\r\n") {
		if strings.HasPrefix(line, "redis_version:") {
			version = strings.TrimPrefix(line, "redis_version:")
			break
		}
	}
	if version == "" {
		version = "unknown"
	}
	
	// 获取数据库大小
	var dbSize int64
	if result := client.DBSize(testCtx); result.Err() == nil {
		dbSize = result.Val()
	}
	
	// 关闭测试连接（实际使用时会重新创建）
	client.Close()
	
	// 保存配置
	for k, v := range config {
		viper.Set(fmt.Sprintf("%s.%s", configKey, k), v)
	}
	
	// 设置为激活数据库
	viper.Set("active_database", name)
	s.activeRedis = name
	
	// 记录到历史
	s.addToHistory(name, config)
	
	// 保存配置文件
	if err := viper.WriteConfig(); err != nil {
		log.Error(ctx, "保存配置失败",
			log.String(common.FieldError, err.Error()))
	}
	
	// 清除缓存的连接（如果有）
	s.redisPool.CloseConnection(name)
	
	// 构建成功响应
	var result strings.Builder
	result.WriteString("✅ Successfully connected to Redis\n")
	result.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	// 基本连接信息
	result.WriteString("📌 Connection Info:\n")
	result.WriteString(fmt.Sprintf("  Config Name: %s (active)\n", name))
	result.WriteString(fmt.Sprintf("  Host: %s:%d\n", host, port))
	result.WriteString(fmt.Sprintf("  Database: %d\n", database))
	if password != "" {
		result.WriteString("  Password: ***\n")
	}
	
	// 服务器信息
	result.WriteString("\n📊 Server Info:\n")
	result.WriteString(fmt.Sprintf("  Version: %s\n", version))
	result.WriteString(fmt.Sprintf("  Keys in DB: %d\n", dbSize))
	
	result.WriteString("\n💡 Note: Connection will be established when needed (lazy loading)")
	
	log.Info(ctx, "Redis连接配置成功",
		log.String(common.FieldStatus, "success"),
		log.String(common.FieldInstance, name),
		log.String("version", version))
	
	return mcp.NewToolResultText(result.String()), nil
}

// handleCurrentRedis 查看当前Redis配置
func (s *RedisServer) handleCurrentRedis(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldTool, "current_redis"),
		log.String(common.FieldOperation, "get_current"))
	
	log.Info(ctx, "查看当前Redis配置")
	
	activeDB := viper.GetString("active_database")
	if activeDB == "" {
		activeDB = common.DefaultInstanceName
	}
	
	// 获取配置
	configKey := fmt.Sprintf("databases.%s", activeDB)
	if !viper.IsSet(configKey) {
		return mcp.NewToolResultText("No active Redis configuration"), nil
	}
	
	// 构建输出
	var result strings.Builder
	result.WriteString("📊 Current Redis Configuration\n")
	result.WriteString(strings.Repeat("=", 40) + "\n\n")
	
	result.WriteString(fmt.Sprintf("Active: %s\n", activeDB))
	
	// 检查连接状态
	isConnected := false
	if client, err := s.redisPool.GetConnection(ctx, activeDB); err == nil && client != nil {
		if err := client.Ping(ctx).Err(); err == nil {
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
	database := viper.GetInt(configKey + ".database")
	
	result.WriteString(fmt.Sprintf("Host: %s:%d\n", host, port))
	result.WriteString(fmt.Sprintf("Database: %d\n", database))
	
	if viper.IsSet(configKey + ".password") && viper.GetString(configKey + ".password") != "" {
		result.WriteString("Password: ***\n")
	}
	
	// 显示历史使用信息
	if history := s.getHistoryInfo(activeDB); history != nil {
		result.WriteString(fmt.Sprintf("Last Used: %s\n", history.LastUsed.Format("2006-01-02 15:04:05")))
		result.WriteString(fmt.Sprintf("Use Count: %d times\n", history.UseCount))
	}
	
	return mcp.NewToolResultText(result.String()), nil
}

// handleHistoryRedis 查看Redis连接历史
func (s *RedisServer) handleHistoryRedis(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldTool, "history_redis"),
		log.String(common.FieldOperation, "list_history"))
	
	log.Info(ctx, "查看Redis连接历史")
	
	// 获取所有数据库配置
	databases := viper.GetStringMap("databases")
	if len(databases) == 0 {
		return mcp.NewToolResultText("No Redis configuration history"), nil
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
	result.WriteString("📚 Redis Configuration History\n")
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
		port := getInt(db.config, "port", common.DefaultRedisPort)
		database := getInt(db.config, "database", 0)
		
		result.WriteString(fmt.Sprintf("    Host: %s:%d\n", host, port))
		result.WriteString(fmt.Sprintf("    Database: %d\n", database))
		
		if getString(db.config, "password", "") != "" {
			result.WriteString("    Password: ***\n")
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
	
	result.WriteString("\nTip: Use 'switch_redis' to switch between configurations")
	
	return mcp.NewToolResultText(result.String()), nil
}

// handleSwitchRedis 切换到其他Redis配置
func (s *RedisServer) handleSwitchRedis(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldTool, "switch_redis"),
		log.String(common.FieldOperation, "switch"))
	
	name := request.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("Please provide configuration name"), nil
	}
	
	log.Info(ctx, "切换Redis配置",
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
	oldActive := s.activeRedis
	s.activeRedis = name
	
	// 关闭旧连接
	if oldActive != "" && oldActive != name {
		s.redisPool.CloseConnection(oldActive)
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
	database := viper.GetInt(configKey + ".database")
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("✅ Switched to Redis configuration: %s\n\n", name))
	result.WriteString(fmt.Sprintf("Host: %s:%d\n", host, port))
	result.WriteString(fmt.Sprintf("Database: %d\n", database))
	
	log.Info(ctx, "Redis配置切换成功",
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
func (s *RedisServer) addToHistory(name string, config map[string]interface{}) {
	history := &historyInfo{
		LastUsed: time.Now(),
		UseCount: 1,
	}
	
	// 读取现有历史
	key := fmt.Sprintf("history.redis.%s", name)
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
func (s *RedisServer) updateHistory(name string) {
	key := fmt.Sprintf("history.redis.%s", name)
	
	useCount := viper.GetInt(key + ".use_count")
	viper.Set(key+".last_used", time.Now())
	viper.Set(key+".use_count", useCount+1)
}

// getHistoryInfo 获取历史信息
func (s *RedisServer) getHistoryInfo(name string) *historyInfo {
	key := fmt.Sprintf("history.redis.%s", name)
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