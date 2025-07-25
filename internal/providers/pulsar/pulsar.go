package pulsar

import (
	"context"
	"fmt"
	"mcp/internal/common"
	"mcp/internal/config"
	"mcp/internal/pool"
	"mcp/internal/server"
	"mcp/pkg/log"
	pulsaradmin "mcp/pkg/pulsar-admin"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/viper"
)

// 使用 common 包的错误，不再定义本地错误

// PulsarServer Pulsar MCP服务器
type PulsarServer struct {
	*server.BaseServer
	config       *config.Config
	activePulsar string             // 当前激活的Pulsar实例名称
	pulsarPool   *pool.PulsarPool   // Pulsar连接池
	
	// 细分的安全选项 (对于Pulsar，某些选项可能不太适用，但保持接口一致)
	disableCreate   bool // 禁用CREATE操作（创建租户、命名空间、主题等）
	disableDrop     bool // 禁用DROP操作（删除租户、命名空间、主题等）
	disableAlter    bool // 禁用ALTER操作（修改配置等，对Pulsar不太适用）
	disableTruncate bool // 禁用TRUNCATE操作（对Pulsar不太适用）
	disableUpdate   bool // 禁用UPDATE操作（修改配置等）
	disableDelete   bool // 禁用DELETE操作（删除订阅等）
}

// NewServer 创建Pulsar服务器
func NewServer() server.MCPServer {
	return &PulsarServer{
		BaseServer: server.NewBaseServer("Pulsar MCP Server", "1.0.0"),
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
func (s *PulsarServer) SetSecurityOptions(disableCreate, disableDrop, disableAlter, disableTruncate, disableUpdate, disableDelete bool) {
	s.disableCreate = disableCreate
	s.disableDrop = disableDrop
	s.disableAlter = disableAlter
	s.disableTruncate = disableTruncate
	s.disableUpdate = disableUpdate
	s.disableDelete = disableDelete
}

// Init 初始化服务器
func (s *PulsarServer) Init(ctx context.Context) error {
	// 初始化配置
	s.config = config.NewConfig("pulsar")

	// 使用自定义的默认配置
	if err := s.initConfig(ctx); err != nil {
		return errors.Wrap(err, "初始化配置失败")
	}

	// 初始化Pulsar连接池（懒加载）
	s.pulsarPool = pool.NewPulsarPool("pulsar")

	// 不在启动时初始化Pulsar连接，使用懒加载机制
	log.Info(ctx, "使用懒加载机制，Pulsar连接将在需要时创建",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldOperation, "init"))

	// 初始化MCP服务器
	s.InitMCPServer(
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(true, false),
	)

	log.Info(ctx, "Pulsar服务器初始化完成",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldOperation, "init"),
		log.String(common.FieldStatus, "success"))
	return nil
}

// initConfig 初始化配置（使用自定义默认配置）
func (s *PulsarServer) initConfig(ctx context.Context) error {
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
		log.Info(ctx, "创建Pulsar配置文件",
			log.String("path", configPath),
			log.String(common.FieldProvider, "pulsar"),
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

// getClient 获取Pulsar Admin客户端（懒加载）
func (s *PulsarServer) getClient(ctx context.Context) (*pulsaradmin.Client, error) {
	// 检查是否有Pulsar配置
	pulsarInstances := viper.GetStringMap("databases")
	if len(pulsarInstances) == 0 {
		return nil, common.NewNoConfigError("pulsar")
	}

	// 设置激活Pulsar实例名称
	s.activePulsar = viper.GetString("active_database")
	if s.activePulsar == "" {
		s.activePulsar = common.DefaultInstanceName
	}

	// 获取对应的Pulsar配置
	pulsarKey := fmt.Sprintf("databases.%s", s.activePulsar)
	if !viper.IsSet(pulsarKey) {
		return nil, errors.Newf("Pulsar配置 '%s' 不存在", s.activePulsar)
	}

	// 使用连接池获取连接
	client, err := s.pulsarPool.GetConnection(ctx, s.activePulsar)
	if err != nil {
		return nil, err
	}

	log.Info(ctx, "Pulsar连接成功",
		log.String("name", s.activePulsar),
		log.String("admin_url", viper.GetString(pulsarKey+".admin_url")),
		log.String("tenant", viper.GetString(pulsarKey+".tenant")),
		log.String("namespace", viper.GetString(pulsarKey+".namespace")),
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldOperation, "connect"),
		log.String(common.FieldStatus, "success"),
	)

	return client, nil
}

// getTenantNamespace 获取当前配置的租户和命名空间
func (s *PulsarServer) getTenantNamespace() (string, string, error) {
	if s.activePulsar == "" {
		return "", "", errors.New("没有激活的Pulsar配置")
	}

	pulsarKey := fmt.Sprintf("databases.%s", s.activePulsar)
	tenant := viper.GetString(pulsarKey + ".tenant")
	namespace := viper.GetString(pulsarKey + ".namespace")

	if tenant == "" {
		return "", "", errors.New("租户配置不能为空")
	}
	if namespace == "" {
		return "", "", errors.New("命名空间配置不能为空")
	}

	return tenant, namespace, nil
}

// RegisterTools 注册工具
func (s *PulsarServer) RegisterTools() error {
	srv := s.GetServer()


	// 租户管理工具
	srv.AddTool(
		mcp.NewTool("list_tenants",
			mcp.WithDescription("List all tenants in Pulsar cluster"),
		),
		s.handleListTenants,
	)

	srv.AddTool(
		mcp.NewTool("create_tenant",
			mcp.WithDescription("Create a new tenant"),
			mcp.WithString("tenant",
				mcp.Required(),
				mcp.Description("Tenant name"),
			),
			mcp.WithArray("admin_roles",
				mcp.Description("Admin roles for the tenant"),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithArray("allowed_clusters",
				mcp.Description("Allowed clusters for the tenant"),
				mcp.Items(map[string]any{"type": "string"}),
			),
		),
		s.handleCreateTenant,
	)

	srv.AddTool(
		mcp.NewTool("delete_tenant",
			mcp.WithDescription("Delete a tenant"),
			mcp.WithString("tenant",
				mcp.Required(),
				mcp.Description("Tenant name to delete"),
			),
		),
		s.handleDeleteTenant,
	)

	// 命名空间管理工具
	srv.AddTool(
		mcp.NewTool("list_namespaces",
			mcp.WithDescription("List all namespaces in current tenant"),
		),
		s.handleListNamespaces,
	)

	srv.AddTool(
		mcp.NewTool("create_namespace",
			mcp.WithDescription("Create a new namespace in current tenant"),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("Namespace name"),
			),
		),
		s.handleCreateNamespace,
	)

	srv.AddTool(
		mcp.NewTool("delete_namespace",
			mcp.WithDescription("Delete a namespace in current tenant"),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("Namespace name to delete"),
			),
		),
		s.handleDeleteNamespace,
	)

	// 主题管理工具
	srv.AddTool(
		mcp.NewTool("list_topics",
			mcp.WithDescription("List all topics in current namespace"),
			mcp.WithBoolean("persistent",
				mcp.Description("Whether to list persistent topics (default: true)"),
			),
		),
		s.handleListTopics,
	)

	srv.AddTool(
		mcp.NewTool("create_topic",
			mcp.WithDescription("Create a new topic"),
			mcp.WithString("topic",
				mcp.Required(),
				mcp.Description("Topic name"),
			),
			mcp.WithBoolean("persistent",
				mcp.Description("Whether to create persistent topic (default: true)"),
			),
			mcp.WithNumber("partitions",
				mcp.Description("Number of partitions (0 for non-partitioned topic)"),
			),
		),
		s.handleCreateTopic,
	)

	srv.AddTool(
		mcp.NewTool("delete_topic",
			mcp.WithDescription("Delete a topic"),
			mcp.WithString("topic",
				mcp.Required(),
				mcp.Description("Topic name to delete"),
			),
			mcp.WithBoolean("persistent",
				mcp.Description("Whether it's a persistent topic (default: true)"),
			),
		),
		s.handleDeleteTopic,
	)

	srv.AddTool(
		mcp.NewTool("get_topic_stats",
			mcp.WithDescription("Get topic statistics"),
			mcp.WithString("topic",
				mcp.Required(),
				mcp.Description("Topic name"),
			),
			mcp.WithBoolean("persistent",
				mcp.Description("Whether it's a persistent topic (default: true)"),
			),
		),
		s.handleGetTopicStats,
	)

	// 订阅管理工具
	srv.AddTool(
		mcp.NewTool("list_subscriptions",
			mcp.WithDescription("List all subscriptions for a topic"),
			mcp.WithString("topic",
				mcp.Required(),
				mcp.Description("Topic name"),
			),
			mcp.WithBoolean("persistent",
				mcp.Description("Whether it's a persistent topic (default: true)"),
			),
		),
		s.handleListSubscriptions,
	)

	srv.AddTool(
		mcp.NewTool("create_subscription",
			mcp.WithDescription("Create a new subscription"),
			mcp.WithString("topic",
				mcp.Required(),
				mcp.Description("Topic name"),
			),
			mcp.WithString("subscription",
				mcp.Required(),
				mcp.Description("Subscription name"),
			),
			mcp.WithBoolean("persistent",
				mcp.Description("Whether it's a persistent topic (default: true)"),
			),
		),
		s.handleCreateSubscription,
	)

	srv.AddTool(
		mcp.NewTool("delete_subscription",
			mcp.WithDescription("Delete a subscription"),
			mcp.WithString("topic",
				mcp.Required(),
				mcp.Description("Topic name"),
			),
			mcp.WithString("subscription",
				mcp.Required(),
				mcp.Description("Subscription name to delete"),
			),
			mcp.WithBoolean("persistent",
				mcp.Description("Whether it's a persistent topic (default: true)"),
			),
		),
		s.handleDeleteSubscription,
	)

	// 详细信息查询工具
	srv.AddTool(
		mcp.NewTool("get_namespace_info",
			mcp.WithDescription("Get detailed namespace information including policies and permissions. Supports both single and batch queries."),
			mcp.WithString("namespace",
				mcp.Description("Single namespace name (defaults to configured namespace)"),
			),
			mcp.WithArray("namespaces",
				mcp.Description("Multiple namespace names for batch query (takes priority over single namespace)"),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithString("tenant",
				mcp.Description("Tenant name (defaults to configured tenant)"),
			),
		),
		s.handleGetNamespaceInfo,
	)

	srv.AddTool(
		mcp.NewTool("get_topic_info",
			mcp.WithDescription("Get detailed topic information and metadata. Supports both single and batch queries."),
			mcp.WithString("topic",
				mcp.Description("Single topic name"),
			),
			mcp.WithArray("topics",
				mcp.Description("Multiple topic names for batch query (takes priority over single topic)"),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithBoolean("persistent",
				mcp.Description("Whether to get persistent topic info (default: true)"),
			),
		),
		s.handleGetTopicInfo,
	)

	srv.AddTool(
		mcp.NewTool("get_tenant_info",
			mcp.WithDescription("Get detailed tenant information. Supports both single and batch queries."),
			mcp.WithString("tenant",
				mcp.Description("Single tenant name (defaults to configured tenant)"),
			),
			mcp.WithArray("tenants",
				mcp.Description("Multiple tenant names for batch query (takes priority over single tenant)"),
				mcp.Items(map[string]any{"type": "string"}),
			),
		),
		s.handleGetTenantInfo,
	)

	// Broker管理工具
	srv.AddTool(
		mcp.NewTool("list_brokers",
			mcp.WithDescription("List all active brokers in the cluster"),
			mcp.WithString("cluster",
				mcp.Description("Cluster name (optional)"),
			),
		),
		s.handleListBrokers,
	)

	srv.AddTool(
		mcp.NewTool("get_broker_info",
			mcp.WithDescription("Get detailed broker information and load data. Supports both single and batch queries."),
			mcp.WithString("broker",
				mcp.Description("Single broker identifier (e.g., 'localhost:8080')"),
			),
			mcp.WithArray("brokers",
				mcp.Description("Multiple broker identifiers for batch query (takes priority over single broker)"),
				mcp.Items(map[string]any{"type": "string"}),
			),
		),
		s.handleGetBrokerInfo,
	)

	srv.AddTool(
		mcp.NewTool("broker_healthcheck",
			mcp.WithDescription("Perform broker health check"),
		),
		s.handleBrokerHealthcheck,
	)

	// 配置更新管理工具
	srv.AddTool(
		mcp.NewTool("update_config",
			mcp.WithDescription("Update Pulsar instance configuration properties"),
			mcp.WithString("name",
				mcp.Description("Pulsar instance name to update (defaults to active instance)"),
			),
			mcp.WithString("admin_url",
				mcp.Description("Update Pulsar Admin API URL"),
			),
			mcp.WithString("tenant",
				mcp.Description("Update default tenant name"),
			),
			mcp.WithString("namespace",
				mcp.Description("Update default namespace name"),
			),
			mcp.WithString("username",
				mcp.Description("Update authentication username"),
			),
			mcp.WithString("password",
				mcp.Description("Update authentication password"),
			),
			mcp.WithString("timeout",
				mcp.Description("Update connection timeout (e.g., '30s', '1m')"),
			),
		),
		s.handleUpdateConfig,
	)

	srv.AddTool(
		mcp.NewTool("get_config_details",
			mcp.WithDescription("Get detailed configuration information for Pulsar instances"),
			mcp.WithString("name",
				mcp.Description("Pulsar instance name (defaults to active instance, 'all' for all instances)"),
			),
			mcp.WithBoolean("include_sensitive",
				mcp.Description("Whether to include sensitive information like passwords (default: false)"),
			),
		),
		s.handleGetConfigDetails,
	)

	// 新的连接管理工具
	srv.AddTool(
		mcp.NewTool("connect_pulsar",
			mcp.WithDescription("Connect to Pulsar server"),
			mcp.WithString("admin_url",
				mcp.Description(fmt.Sprintf("Pulsar Admin API URL (default: http://localhost:%d)", common.DefaultPulsarPort)),
			),
			mcp.WithString("tenant",
				mcp.Required(),
				mcp.Description("Pulsar tenant name"),
			),
			mcp.WithString("namespace", 
				mcp.Required(),
				mcp.Description("Pulsar namespace name"),
			),
			mcp.WithString("username",
				mcp.Description("Pulsar authentication username"),
			),
			mcp.WithString("password",
				mcp.Description("Pulsar authentication password"),
			),
			mcp.WithString("name",
				mcp.Description("Configuration name (auto-generated if not provided)"),
			),
		),
		s.handleConnectPulsar,
	)

	srv.AddTool(
		mcp.NewTool("current_pulsar",
			mcp.WithDescription("Show current Pulsar connection information"),
		),
		s.handleCurrentPulsar,
	)

	srv.AddTool(
		mcp.NewTool("history_pulsar",
			mcp.WithDescription("Show Pulsar connection history"),
		),
		s.handleHistoryPulsar,
	)

	srv.AddTool(
		mcp.NewTool("switch_pulsar",
			mcp.WithDescription("Switch to another Pulsar configuration"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Configuration name or index number"),
			),
		),
		s.handleSwitchPulsar,
	)

	return nil
}

// RegisterResources 注册资源
func (s *PulsarServer) RegisterResources() error {
	// 暂时不注册资源，等待API稳定
	return nil
}

// Start 启动服务器
func (s *PulsarServer) Start(ctx context.Context, transport string) error {
	// 启动逻辑已在registry中实现
	return nil
}