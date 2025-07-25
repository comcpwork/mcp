package pulsar

import (
	"context"
	"fmt"
	"mcp/internal/common"
	"mcp/pkg/log"
	pulsaradmin "mcp/pkg/pulsar-admin"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/viper"
)

// handleConnectPulsar 处理直接连接请求
func (s *PulsarServer) handleConnectPulsar(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "connect_pulsar"),
		log.String(common.FieldOperation, "connect"))
	
	log.Info(ctx, "处理Pulsar连接请求")
	
	// 获取连接参数
	adminURL := request.GetString("admin_url", fmt.Sprintf("http://localhost:%d", common.DefaultPulsarPort))
	tenant := request.GetString("tenant", "")
	namespace := request.GetString("namespace", "")
	username := request.GetString("username", "")
	password := request.GetString("password", "")
	name := request.GetString("name", "") // 配置名称
	
	// 验证必需参数
	if tenant == "" {
		return mcp.NewToolResultError("Tenant name is required"), nil
	}
	if namespace == "" {
		return mcp.NewToolResultError("Namespace name is required"), nil
	}
	
	// 如果没有指定名称，生成一个
	if name == "" {
		// 从 adminURL 提取主机名
		host := adminURL
		if strings.HasPrefix(host, "http://") {
			host = strings.TrimPrefix(host, "http://")
		} else if strings.HasPrefix(host, "https://") {
			host = strings.TrimPrefix(host, "https://")
		}
		// 移除端口号
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
		
		name = fmt.Sprintf("%s_%s_%s", host, tenant, namespace)
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
		"admin_url": adminURL,
		"tenant":    tenant,
		"namespace": namespace,
		"username":  username,
		"password":  password,
		"timeout":   "30s",
	}
	
	// 测试连接
	log.Info(ctx, "测试Pulsar连接",
		log.String("admin_url", adminURL),
		log.String("tenant", tenant),
		log.String("namespace", namespace))
	
	// 创建Pulsar Admin客户端
	var options []pulsaradmin.ClientOption
	if username != "" && password != "" {
		options = append(options, pulsaradmin.WithAuth(username, password))
	}
	timeout, _ := time.ParseDuration("30s")
	options = append(options, pulsaradmin.WithTimeout(timeout))
	
	client := pulsaradmin.NewClient(adminURL, options...)
	
	// 测试连接 - 尝试获取Leader Broker信息
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	// 获取Leader Broker信息以验证连接
	leaderBroker, err := client.Brokers().GetLeaderBroker(testCtx)
	if err != nil {
		log.Error(ctx, "Pulsar连接测试失败",
			log.String(common.FieldError, err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Connection test failed: %v", err)), nil
	}
	
	// 从连接成功推断集群信息
	var clusters []string
	if leaderBroker != nil && leaderBroker.PulsarVersion != "" {
		// 如果连接成功，至少有一个默认集群
		clusters = []string{"standalone"}
	}
	
	// 获取租户信息
	tenants, err := client.Tenants().List(testCtx)
	if err != nil {
		log.Warn(ctx, "获取租户列表失败",
			log.String(common.FieldError, err.Error()))
		tenants = []string{} // 不影响连接
	}
	
	// 检查指定的租户是否存在
	tenantExists := false
	for _, t := range tenants {
		if t == tenant {
			tenantExists = true
			break
		}
	}
	
	// 如果租户存在，检查命名空间
	var namespaces []string
	namespaceExists := false
	if tenantExists {
		namespaces, err = client.Namespaces().List(testCtx, tenant)
		if err != nil {
			log.Warn(ctx, "获取命名空间列表失败",
				log.String(common.FieldError, err.Error()))
		} else {
			fullNamespace := fmt.Sprintf("%s/%s", tenant, namespace)
			for _, ns := range namespaces {
				if ns == fullNamespace || ns == namespace {
					namespaceExists = true
					break
				}
			}
		}
	}
	
	// 获取broker信息 - 使用默认集群名称
	var brokers []string
	for _, cluster := range []string{"standalone", "cluster", "default"} {
		b, err := client.Brokers().List(testCtx, cluster)
		if err == nil && len(b) > 0 {
			brokers = b
			break
		}
	}
	
	// 如果都失败，不影响连接
	if len(brokers) == 0 {
		log.Warn(ctx, "获取Broker列表失败，可能需要指定正确的集群名称")
	}
	
	// 保存配置
	for k, v := range config {
		viper.Set(fmt.Sprintf("%s.%s", configKey, k), v)
	}
	
	// 设置为激活实例
	viper.Set("active_database", name)
	s.activePulsar = name
	
	// 记录到历史
	s.addToHistory(name, config)
	
	// 保存配置文件
	if err := viper.WriteConfig(); err != nil {
		log.Error(ctx, "保存配置失败",
			log.String(common.FieldError, err.Error()))
	}
	
	// 清除缓存的连接（如果有）
	if s.pulsarPool != nil {
		s.pulsarPool.CloseConnection(name)
	}
	
	// 构建成功响应
	var result strings.Builder
	result.WriteString("✅ Successfully connected to Pulsar\n")
	result.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	// 基本连接信息
	result.WriteString("📌 Connection Info:\n")
	result.WriteString(fmt.Sprintf("  Config Name: %s (active)\n", name))
	result.WriteString(fmt.Sprintf("  Admin URL: %s\n", adminURL))
	result.WriteString(fmt.Sprintf("  Tenant: %s", tenant))
	if !tenantExists {
		result.WriteString(" ⚠️ (not exists)")
	}
	result.WriteString("\n")
	result.WriteString(fmt.Sprintf("  Namespace: %s", namespace))
	if tenantExists && !namespaceExists {
		result.WriteString(" ⚠️ (not exists)")
	}
	result.WriteString("\n")
	if username != "" {
		result.WriteString("  Authentication: Yes\n")
	}
	
	// 集群信息
	result.WriteString("\n📊 Cluster Info:\n")
	if len(clusters) > 0 {
		result.WriteString(fmt.Sprintf("  Available Clusters: %s\n", strings.Join(clusters, ", ")))
	}
	result.WriteString(fmt.Sprintf("  Active Brokers: %d\n", len(brokers)))
	
	// 租户信息
	if len(tenants) > 0 {
		result.WriteString(fmt.Sprintf("  Total Tenants: %d\n", len(tenants)))
		if len(tenants) <= 10 {
			result.WriteString(fmt.Sprintf("  Tenants: %s\n", strings.Join(tenants, ", ")))
		}
	}
	
	result.WriteString("\n💡 Note: Connection will be established when needed (lazy loading)")
	
	log.Info(ctx, "Pulsar连接配置成功",
		log.String(common.FieldStatus, "success"),
		log.String(common.FieldInstance, name),
		log.Int("broker_count", len(brokers)))
	
	return mcp.NewToolResultText(result.String()), nil
}

// handleCurrentPulsar 查看当前Pulsar配置
func (s *PulsarServer) handleCurrentPulsar(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "current_pulsar"),
		log.String(common.FieldOperation, "get_current"))
	
	log.Info(ctx, "查看当前Pulsar配置")
	
	activeDB := viper.GetString("active_database")
	if activeDB == "" {
		activeDB = common.DefaultInstanceName
	}
	
	// 获取配置
	configKey := fmt.Sprintf("databases.%s", activeDB)
	if !viper.IsSet(configKey) {
		return mcp.NewToolResultText("No active Pulsar configuration"), nil
	}
	
	// 构建输出
	var result strings.Builder
	result.WriteString("📊 Current Pulsar Configuration\n")
	result.WriteString(strings.Repeat("=", 40) + "\n\n")
	
	result.WriteString(fmt.Sprintf("Active: %s\n", activeDB))
	
	// 检查连接状态
	isConnected := false
	if s.pulsarPool != nil {
		if _, err := s.pulsarPool.GetConnection(ctx, activeDB); err == nil {
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
	adminURL := viper.GetString(configKey + ".admin_url")
	tenant := viper.GetString(configKey + ".tenant")
	namespace := viper.GetString(configKey + ".namespace")
	
	result.WriteString(fmt.Sprintf("Admin URL: %s\n", adminURL))
	result.WriteString(fmt.Sprintf("Tenant: %s\n", tenant))
	result.WriteString(fmt.Sprintf("Namespace: %s\n", namespace))
	
	if viper.IsSet(configKey + ".username") && viper.GetString(configKey + ".username") != "" {
		result.WriteString("Authentication: Enabled\n")
	}
	
	// 显示历史使用信息
	if history := s.getHistoryInfo(activeDB); history != nil {
		result.WriteString(fmt.Sprintf("Last Used: %s\n", history.LastUsed.Format("2006-01-02 15:04:05")))
		result.WriteString(fmt.Sprintf("Use Count: %d times\n", history.UseCount))
	}
	
	return mcp.NewToolResultText(result.String()), nil
}

// handleHistoryPulsar 查看Pulsar连接历史
func (s *PulsarServer) handleHistoryPulsar(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "history_pulsar"),
		log.String(common.FieldOperation, "list_history"))
	
	log.Info(ctx, "查看Pulsar连接历史")
	
	// 获取所有数据库配置
	databases := viper.GetStringMap("databases")
	if len(databases) == 0 {
		return mcp.NewToolResultText("No Pulsar configuration history"), nil
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
	result.WriteString("📚 Pulsar Configuration History\n")
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
		
		adminURL := getString(db.config, "admin_url", "")
		tenant := getString(db.config, "tenant", "")
		namespace := getString(db.config, "namespace", "")
		
		result.WriteString(fmt.Sprintf("    Admin URL: %s\n", adminURL))
		result.WriteString(fmt.Sprintf("    Tenant: %s\n", tenant))
		result.WriteString(fmt.Sprintf("    Namespace: %s\n", namespace))
		
		if getString(db.config, "username", "") != "" {
			result.WriteString("    Authentication: Enabled\n")
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
	
	result.WriteString("\nTip: Use 'switch_pulsar' to switch between configurations")
	
	return mcp.NewToolResultText(result.String()), nil
}

// handleSwitchPulsar 切换到其他Pulsar配置
func (s *PulsarServer) handleSwitchPulsar(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = log.WithFields(ctx,
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "switch_pulsar"),
		log.String(common.FieldOperation, "switch"))
	
	name := request.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("Please provide configuration name"), nil
	}
	
	log.Info(ctx, "切换Pulsar配置",
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
	
	// 设置为激活实例
	viper.Set("active_database", name)
	oldActive := s.activePulsar
	s.activePulsar = name
	
	// 关闭旧连接
	if oldActive != "" && oldActive != name && s.pulsarPool != nil {
		s.pulsarPool.CloseConnection(oldActive)
	}
	
	// 更新历史
	s.updateHistory(name)
	
	// 保存配置
	if err := viper.WriteConfig(); err != nil {
		log.Error(ctx, "保存配置失败",
			log.String(common.FieldError, err.Error()))
	}
	
	// 获取配置信息
	adminURL := viper.GetString(configKey + ".admin_url")
	tenant := viper.GetString(configKey + ".tenant")
	namespace := viper.GetString(configKey + ".namespace")
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("✅ Switched to Pulsar configuration: %s\n\n", name))
	result.WriteString(fmt.Sprintf("Admin URL: %s\n", adminURL))
	result.WriteString(fmt.Sprintf("Tenant: %s\n", tenant))
	result.WriteString(fmt.Sprintf("Namespace: %s\n", namespace))
	
	log.Info(ctx, "Pulsar配置切换成功",
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
func (s *PulsarServer) addToHistory(name string, config map[string]interface{}) {
	history := &historyInfo{
		LastUsed: time.Now(),
		UseCount: 1,
	}
	
	// 读取现有历史
	key := fmt.Sprintf("history.pulsar.%s", name)
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
func (s *PulsarServer) updateHistory(name string) {
	key := fmt.Sprintf("history.pulsar.%s", name)
	
	useCount := viper.GetInt(key + ".use_count")
	viper.Set(key+".last_used", time.Now())
	viper.Set(key+".use_count", useCount+1)
}

// getHistoryInfo 获取历史信息
func (s *PulsarServer) getHistoryInfo(name string) *historyInfo {
	key := fmt.Sprintf("history.pulsar.%s", name)
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