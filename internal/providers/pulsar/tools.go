package pulsar

import (
	"context"
	"fmt"
	"mcp/internal/common"
	"mcp/pkg/log"
	pulsaradmin "mcp/pkg/pulsar-admin"
	"sort"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/viper"
)





// handleListTenants 处理列出租户的请求
func (s *PulsarServer) handleListTenants(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理list_tenants请求")

	client, err := s.getClient(ctx)
	if err != nil {
		if common.IsNoConfigError(err) {
			return mcp.NewToolResultText("No Pulsar configuration found. Use add_pulsar to configure a Pulsar instance first."), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenants, err := client.Tenants().List(ctx)
	if err != nil {
		log.Error(ctx, "获取租户列表失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error listing tenants: %s", err.Error())), nil
	}

	if len(tenants) == 0 {
		return mcp.NewToolResultText("No tenants found"), nil
	}

	sort.Strings(tenants)
	output := fmt.Sprintf("Tenants (%d):\n", len(tenants))
	for _, tenant := range tenants {
		output += fmt.Sprintf("  - %s\n", tenant)
	}

	return mcp.NewToolResultText(output[:len(output)-1]), nil // 去掉最后的换行符
}

// handleCreateTenant 处理创建租户的请求
func (s *PulsarServer) handleCreateTenant(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理create_tenant请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "create_tenant"),
		log.String(common.FieldOperation, "create"))

	// 安全检查
	if err := s.validatePulsarOperationSecurity(ctx, "CREATE_TENANT"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tenant := request.GetString("tenant", "")
	if tenant == "" {
		return mcp.NewToolResultError("Error: tenant is required"), nil
	}

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenantInfo := &pulsaradmin.TenantInfo{}

	// 获取管理员角色
	args := request.GetArguments()
	if adminRoles, ok := args["admin_roles"].([]interface{}); ok {
		for _, role := range adminRoles {
			if roleStr, ok := role.(string); ok {
				tenantInfo.AdminRoles = append(tenantInfo.AdminRoles, roleStr)
			}
		}
	}

	// 获取允许的集群
	if allowedClusters, ok := args["allowed_clusters"].([]interface{}); ok {
		for _, cluster := range allowedClusters {
			if clusterStr, ok := cluster.(string); ok {
				tenantInfo.AllowedClusters = append(tenantInfo.AllowedClusters, clusterStr)
			}
		}
	}

	if err := client.Tenants().Create(ctx, tenant, tenantInfo); err != nil {
		log.Error(ctx, "创建租户失败", 
			log.String("tenant", tenant),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error creating tenant: %s", err.Error())), nil
	}

	log.Info(ctx, "租户创建成功", log.String("tenant", tenant))
	return mcp.NewToolResultText(fmt.Sprintf("✓ Tenant '%s' created successfully", tenant)), nil
}

// handleDeleteTenant 处理删除租户的请求
func (s *PulsarServer) handleDeleteTenant(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理delete_tenant请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "delete_tenant"),
		log.String(common.FieldOperation, "delete"))

	// 安全检查
	if err := s.validatePulsarOperationSecurity(ctx, "DELETE_TENANT"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tenant := request.GetString("tenant", "")
	if tenant == "" {
		return mcp.NewToolResultError("Error: tenant is required"), nil
	}

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	if err := client.Tenants().Delete(ctx, tenant); err != nil {
		log.Error(ctx, "删除租户失败",
			log.String("tenant", tenant),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error deleting tenant: %s", err.Error())), nil
	}

	log.Info(ctx, "租户删除成功", log.String("tenant", tenant))
	return mcp.NewToolResultText(fmt.Sprintf("✓ Tenant '%s' deleted successfully", tenant)), nil
}

// handleListNamespaces 处理列出命名空间的请求
func (s *PulsarServer) handleListNamespaces(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理list_namespaces请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "list_namespaces"),
		log.String(common.FieldOperation, "list"))

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenant, _, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant: %s", err.Error())), nil
	}

	namespaces, err := client.Namespaces().List(ctx, tenant)
	if err != nil {
		log.Error(ctx, "获取命名空间列表失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error listing namespaces: %s", err.Error())), nil
	}

	if len(namespaces) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No namespaces found in tenant '%s'", tenant)), nil
	}

	sort.Strings(namespaces)
	output := fmt.Sprintf("Namespaces in tenant '%s' (%d):\n", tenant, len(namespaces))
	for _, namespace := range namespaces {
		output += fmt.Sprintf("  - %s\n", namespace)
	}

	return mcp.NewToolResultText(output[:len(output)-1]), nil // 去掉最后的换行符
}

// handleCreateNamespace 处理创建命名空间的请求
func (s *PulsarServer) handleCreateNamespace(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理create_namespace请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "create_namespace"),
		log.String(common.FieldOperation, "create"))

	// 安全检查
	if err := s.validatePulsarOperationSecurity(ctx, "CREATE_NAMESPACE"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	namespace := request.GetString("namespace", "")
	if namespace == "" {
		return mcp.NewToolResultError("Error: namespace is required"), nil
	}

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenant, _, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant: %s", err.Error())), nil
	}

	if err := client.Namespaces().Create(ctx, tenant, namespace, nil); err != nil {
		log.Error(ctx, "创建命名空间失败",
			log.String("tenant", tenant),
			log.String("namespace", namespace),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error creating namespace: %s", err.Error())), nil
	}

	log.Info(ctx, "命名空间创建成功", 
		log.String("tenant", tenant),
		log.String("namespace", namespace))

	return mcp.NewToolResultText(fmt.Sprintf("✓ Namespace '%s/%s' created successfully", tenant, namespace)), nil
}

// handleDeleteNamespace 处理删除命名空间的请求
func (s *PulsarServer) handleDeleteNamespace(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理delete_namespace请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "delete_namespace"),
		log.String(common.FieldOperation, "delete"))

	// 安全检查
	if err := s.validatePulsarOperationSecurity(ctx, "DELETE_NAMESPACE"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	namespace := request.GetString("namespace", "")
	if namespace == "" {
		return mcp.NewToolResultError("Error: namespace is required"), nil
	}

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenant, _, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant: %s", err.Error())), nil
	}

	if err := client.Namespaces().Delete(ctx, tenant, namespace); err != nil {
		log.Error(ctx, "删除命名空间失败",
			log.String("tenant", tenant),
			log.String("namespace", namespace),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error deleting namespace: %s", err.Error())), nil
	}

	log.Info(ctx, "命名空间删除成功",
		log.String("tenant", tenant),
		log.String("namespace", namespace))

	return mcp.NewToolResultText(fmt.Sprintf("✓ Namespace '%s/%s' deleted successfully", tenant, namespace)), nil
}

// handleListTopics 处理列出主题的请求
func (s *PulsarServer) handleListTopics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理list_topics请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "list_topics"),
		log.String(common.FieldOperation, "list"))

	persistent := request.GetBool("persistent", true)

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenant, namespace, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant/namespace: %s", err.Error())), nil
	}

	topics, err := client.Topics().List(ctx, tenant, namespace, persistent)
	if err != nil {
		log.Error(ctx, "获取主题列表失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error listing topics: %s", err.Error())), nil
	}

	if len(topics) == 0 {
		topicType := "persistent"
		if !persistent {
			topicType = "non-persistent"
		}
		return mcp.NewToolResultText(fmt.Sprintf("No %s topics found in %s/%s", topicType, tenant, namespace)), nil
	}

	sort.Strings(topics)
	topicType := "persistent"
	if !persistent {
		topicType = "non-persistent"
	}
	output := fmt.Sprintf("%s topics in %s/%s (%d):\n", strings.Title(topicType), tenant, namespace, len(topics))
	for _, topic := range topics {
		output += fmt.Sprintf("  - %s\n", topic)
	}

	return mcp.NewToolResultText(output[:len(output)-1]), nil // 去掉最后的换行符
}

// handleCreateTopic 处理创建主题的请求
func (s *PulsarServer) handleCreateTopic(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理create_topic请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "create_topic"),
		log.String(common.FieldOperation, "create"))

	// 安全检查
	if err := s.validatePulsarOperationSecurity(ctx, "CREATE_TOPIC"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	topic := request.GetString("topic", "")
	if topic == "" {
		return mcp.NewToolResultError("Error: topic is required"), nil
	}

	persistent := request.GetBool("persistent", true)
	partitions := int(request.GetFloat("partitions", 0))

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenant, namespace, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant/namespace: %s", err.Error())), nil
	}

	if partitions > 0 {
		// 创建分区主题
		err = client.Topics().CreatePartitioned(ctx, tenant, namespace, topic, partitions, persistent)
	} else {
		// 创建非分区主题
		err = client.Topics().CreateNonPartitioned(ctx, tenant, namespace, topic, persistent)
	}

	if err != nil {
		log.Error(ctx, "创建主题失败",
			log.String("tenant", tenant),
			log.String("namespace", namespace),
			log.String("topic", topic),
			log.Bool("persistent", persistent),
			log.Int("partitions", partitions),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error creating topic: %s", err.Error())), nil
	}

	log.Info(ctx, "主题创建成功",
		log.String("tenant", tenant),
		log.String("namespace", namespace),
		log.String("topic", topic),
		log.Bool("persistent", persistent),
		log.Int("partitions", partitions))

	topicType := "persistent"
	if !persistent {
		topicType = "non-persistent"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("✓ %s topic '%s/%s/%s' created successfully", strings.Title(topicType), tenant, namespace, topic))
	if partitions > 0 {
		result.WriteString(fmt.Sprintf(" with %d partitions", partitions))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// handleDeleteTopic 处理删除主题的请求
func (s *PulsarServer) handleDeleteTopic(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理delete_topic请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "delete_topic"),
		log.String(common.FieldOperation, "delete"))

	// 安全检查
	if err := s.validatePulsarOperationSecurity(ctx, "DELETE_TOPIC"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	topic := request.GetString("topic", "")
	if topic == "" {
		return mcp.NewToolResultError("Error: topic is required"), nil
	}

	persistent := request.GetBool("persistent", true)

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenant, namespace, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant/namespace: %s", err.Error())), nil
	}

	// 首先尝试检查是否为分区主题
	metadata, err := client.Topics().GetPartitionedMetadata(ctx, tenant, namespace, topic, persistent)
	if err == nil && metadata.Partitions > 0 {
		// 是分区主题，使用分区主题删除方法
		err = client.Topics().DeletePartitioned(ctx, tenant, namespace, topic, persistent)
	} else {
		// 是非分区主题，使用普通删除方法
		err = client.Topics().Delete(ctx, tenant, namespace, topic, persistent)
	}

	if err != nil {
		log.Error(ctx, "删除主题失败",
			log.String("tenant", tenant),
			log.String("namespace", namespace),
			log.String("topic", topic),
			log.Bool("persistent", persistent),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error deleting topic: %s", err.Error())), nil
	}

	log.Info(ctx, "主题删除成功",
		log.String("tenant", tenant),
		log.String("namespace", namespace),
		log.String("topic", topic),
		log.Bool("persistent", persistent))

	topicType := "persistent"
	if !persistent {
		topicType = "non-persistent"
	}

	return mcp.NewToolResultText(fmt.Sprintf("✓ %s topic '%s/%s/%s' deleted successfully", strings.Title(topicType), tenant, namespace, topic)), nil
}

// handleGetTopicStats 处理获取主题统计的请求
func (s *PulsarServer) handleGetTopicStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理get_topic_stats请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "get_topic_stats"),
		log.String(common.FieldOperation, "get_stats"))

	topic := request.GetString("topic", "")
	if topic == "" {
		return mcp.NewToolResultError("Error: topic is required"), nil
	}

	persistent := request.GetBool("persistent", true)

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenant, namespace, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant/namespace: %s", err.Error())), nil
	}

	stats, err := client.Topics().GetStats(ctx, tenant, namespace, topic, persistent)
	if err != nil {
		log.Error(ctx, "获取主题统计失败",
			log.String("tenant", tenant),
			log.String("namespace", namespace),
			log.String("topic", topic),
			log.Bool("persistent", persistent),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error getting topic stats: %s", err.Error())), nil
	}

	output := formatTopicStats(tenant, namespace, topic, persistent, stats)
	return mcp.NewToolResultText(output), nil
}

// handleListSubscriptions 处理列出订阅的请求
func (s *PulsarServer) handleListSubscriptions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理list_subscriptions请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "list_subscriptions"),
		log.String(common.FieldOperation, "list"))

	topic := request.GetString("topic", "")
	if topic == "" {
		return mcp.NewToolResultError("Error: topic is required"), nil
	}

	persistent := request.GetBool("persistent", true)

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenant, namespace, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant/namespace: %s", err.Error())), nil
	}

	subscriptions, err := client.Subscriptions().List(ctx, tenant, namespace, topic, persistent)
	if err != nil {
		log.Error(ctx, "获取订阅列表失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error listing subscriptions: %s", err.Error())), nil
	}

	if len(subscriptions) == 0 {
		topicType := "persistent"
		if !persistent {
			topicType = "non-persistent"
		}
		return mcp.NewToolResultText(fmt.Sprintf("No subscriptions found for %s topic '%s/%s/%s'", topicType, tenant, namespace, topic)), nil
	}

	sort.Strings(subscriptions)
	topicType := "persistent"
	if !persistent {
		topicType = "non-persistent"
	}
	output := fmt.Sprintf("Subscriptions for %s topic '%s/%s/%s' (%d):\n", topicType, tenant, namespace, topic, len(subscriptions))
	for _, sub := range subscriptions {
		output += fmt.Sprintf("  - %s\n", sub)
	}

	return mcp.NewToolResultText(output[:len(output)-1]), nil // 去掉最后的换行符
}

// handleCreateSubscription 处理创建订阅的请求
func (s *PulsarServer) handleCreateSubscription(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理create_subscription请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "create_subscription"),
		log.String(common.FieldOperation, "create"))

	// 安全检查
	if err := s.validatePulsarOperationSecurity(ctx, "CREATE_SUBSCRIPTION"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	topic := request.GetString("topic", "")
	if topic == "" {
		return mcp.NewToolResultError("Error: topic is required"), nil
	}

	subscription := request.GetString("subscription", "")
	if subscription == "" {
		return mcp.NewToolResultError("Error: subscription is required"), nil
	}

	persistent := request.GetBool("persistent", true)

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenant, namespace, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant/namespace: %s", err.Error())), nil
	}

	if err := client.Subscriptions().Create(ctx, tenant, namespace, topic, subscription, persistent); err != nil {
		log.Error(ctx, "创建订阅失败",
			log.String("tenant", tenant),
			log.String("namespace", namespace),
			log.String("topic", topic),
			log.String("subscription", subscription),
			log.Bool("persistent", persistent),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error creating subscription: %s", err.Error())), nil
	}

	log.Info(ctx, "订阅创建成功",
		log.String("tenant", tenant),
		log.String("namespace", namespace),
		log.String("topic", topic),
		log.String("subscription", subscription),
		log.Bool("persistent", persistent))

	topicType := "persistent"
	if !persistent {
		topicType = "non-persistent"
	}

	return mcp.NewToolResultText(fmt.Sprintf("✓ Subscription '%s' created for %s topic '%s/%s/%s'", subscription, topicType, tenant, namespace, topic)), nil
}

// handleDeleteSubscription 处理删除订阅的请求
func (s *PulsarServer) handleDeleteSubscription(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理delete_subscription请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "delete_subscription"),
		log.String(common.FieldOperation, "delete"))

	// 安全检查
	if err := s.validatePulsarOperationSecurity(ctx, "DELETE_SUBSCRIPTION"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	topic := request.GetString("topic", "")
	if topic == "" {
		return mcp.NewToolResultError("Error: topic is required"), nil
	}

	subscription := request.GetString("subscription", "")
	if subscription == "" {
		return mcp.NewToolResultError("Error: subscription is required"), nil
	}

	persistent := request.GetBool("persistent", true)

	client, err := s.getClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	tenant, namespace, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant/namespace: %s", err.Error())), nil
	}

	if err := client.Subscriptions().Delete(ctx, tenant, namespace, topic, subscription, persistent); err != nil {
		log.Error(ctx, "删除订阅失败",
			log.String("tenant", tenant),
			log.String("namespace", namespace),
			log.String("topic", topic),
			log.String("subscription", subscription),
			log.Bool("persistent", persistent),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error deleting subscription: %s", err.Error())), nil
	}

	log.Info(ctx, "订阅删除成功",
		log.String("tenant", tenant),
		log.String("namespace", namespace),
		log.String("topic", topic),
		log.String("subscription", subscription),
		log.Bool("persistent", persistent))

	topicType := "persistent"
	if !persistent {
		topicType = "non-persistent"
	}

	return mcp.NewToolResultText(fmt.Sprintf("✓ Subscription '%s' deleted from %s topic '%s/%s/%s'", subscription, topicType, tenant, namespace, topic)), nil
}

// 工具函数

// ensureToolsConfig 确保工具配置存在
func ensureToolsConfig() error {
	if !viper.IsSet("tools") {
		viper.Set("tools", map[string]interface{}{})
	}
	if !viper.IsSet("databases") {
		viper.Set("databases", map[string]interface{}{})
	}
	return nil
}


// formatTopicStats 格式化主题统计信息
func formatTopicStats(tenant, namespace, topic string, persistent bool, stats *pulsaradmin.TopicStats) string {
	topicType := "persistent"
	if !persistent {
		topicType = "non-persistent"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Stats for %s topic '%s/%s/%s':\n\n", topicType, tenant, namespace, topic))

	// 基本统计
	result.WriteString("Message Stats:\n")
	result.WriteString(fmt.Sprintf("  In Rate: %.2f msg/s (%.2f MB/s)\n", stats.MsgRateIn, stats.MsgThroughputIn/1024/1024))
	result.WriteString(fmt.Sprintf("  Out Rate: %.2f msg/s (%.2f MB/s)\n", stats.MsgRateOut, stats.MsgThroughputOut/1024/1024))
	result.WriteString(fmt.Sprintf("  In Counter: %d messages\n", stats.MsgInCounter))
	result.WriteString(fmt.Sprintf("  Out Counter: %d messages\n", stats.MsgOutCounter))
	result.WriteString(fmt.Sprintf("  Avg Size: %.2f bytes\n", stats.AverageMsgSize))

	// 存储统计
	result.WriteString("\nStorage Stats:\n")
	result.WriteString(fmt.Sprintf("  Storage Size: %.2f MB\n", float64(stats.StorageSize)/1024/1024))
	result.WriteString(fmt.Sprintf("  Backlog Size: %.2f MB\n", float64(stats.BacklogSize)/1024/1024))

	// 发布者统计
	if len(stats.Publishers) > 0 {
		result.WriteString(fmt.Sprintf("\nPublishers (%d):\n", len(stats.Publishers)))
		for i, pub := range stats.Publishers {
			result.WriteString(fmt.Sprintf("  #%d: %.2f msg/s (%.2f MB/s) from %s\n", 
				i+1, pub.MsgRateIn, pub.MsgThroughputIn/1024/1024, pub.Address))
		}
	}

	// 订阅统计
	if len(stats.Subscriptions) > 0 {
		result.WriteString(fmt.Sprintf("\nSubscriptions (%d):\n", len(stats.Subscriptions)))
		for subName, sub := range stats.Subscriptions {
			result.WriteString(fmt.Sprintf("  %s: %.2f msg/s, %d backlog, %d consumers\n",
				subName, sub.MsgRateOut, sub.MsgBacklog, len(sub.Consumers)))
		}
	}

	return result.String()
}

// handleGetNamespaceInfo 处理获取命名空间详细信息的请求，支持批量查询
func (s *PulsarServer) handleGetNamespaceInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理get_namespace_info请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "get_namespace_info"),
		log.String(common.FieldOperation, "get_info"))

	// 获取客户端
	client, err := s.getClient(ctx)
	if err != nil {
		if common.IsNoConfigError(err) {
			return mcp.NewToolResultError("Error: No Pulsar configurations found. Please add a Pulsar instance first using 'add_pulsar' tool."), nil
		}
		log.Error(ctx, "获取Pulsar客户端失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	// 获取参数
	tenant := request.GetString("tenant", "")
	namespaces := getStringArray(request, "namespaces")
	
	// 优先使用批量查询
	if len(namespaces) > 0 {
		return s.handleBatchNamespaceInfo(ctx, client, tenant, namespaces)
	}
	
	// 单个查询（向后兼容）
	namespace := request.GetString("namespace", "")
	return s.handleSingleNamespaceInfo(ctx, client, tenant, namespace)
}

// handleSingleNamespaceInfo 处理单个命名空间信息查询
func (s *PulsarServer) handleSingleNamespaceInfo(ctx context.Context, client *pulsaradmin.Client, tenant, namespace string) (*mcp.CallToolResult, error) {
	// 如果未指定，使用配置中的默认值
	if tenant == "" || namespace == "" {
		defaultTenant, defaultNamespace, err := s.getTenantNamespace()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error getting default tenant/namespace: %s", err.Error())), nil
		}
		if tenant == "" {
			tenant = defaultTenant
		}
		if namespace == "" {
			namespace = defaultNamespace
		}
	}

	// 获取命名空间策略
	policies, err := client.Namespaces().GetPolicies(ctx, tenant, namespace)
	if err != nil {
		log.Error(ctx, "获取命名空间策略失败", 
			log.String("tenant", tenant),
			log.String("namespace", namespace),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error getting namespace policies: %s", err.Error())), nil
	}

	// 获取权限信息
	permissions, err := client.Namespaces().GetPermissions(ctx, tenant, namespace)
	if err != nil {
		log.Warn(ctx, "获取命名空间权限失败", 
			log.String("tenant", tenant),
			log.String("namespace", namespace),
			log.String("error", err.Error()))
		// 权限获取失败不影响主要功能，继续执行
	}

	output := formatNamespaceInfoCompact(tenant, namespace, policies, permissions)
	return mcp.NewToolResultText(output), nil
}

// handleBatchNamespaceInfo 处理批量命名空间信息查询
func (s *PulsarServer) handleBatchNamespaceInfo(ctx context.Context, client *pulsaradmin.Client, tenant string, namespaces []string) (*mcp.CallToolResult, error) {
	// 如果未指定租户，使用配置中的默认值
	if tenant == "" {
		defaultTenant, _, err := s.getTenantNamespace()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error getting default tenant: %s", err.Error())), nil
		}
		tenant = defaultTenant
	}

	// 创建查询函数
	queryFunc := func(namespace string) (string, error) {
		// 获取命名空间策略
		policies, err := client.Namespaces().GetPolicies(ctx, tenant, namespace)
		if err != nil {
			return "", err
		}

		// 获取权限信息
		permissions, err := client.Namespaces().GetPermissions(ctx, tenant, namespace)
		if err != nil {
			log.Warn(ctx, "获取命名空间权限失败", 
				log.String("tenant", tenant),
				log.String("namespace", namespace),
				log.String("error", err.Error()))
			// 权限获取失败不影响主要功能，设为nil继续
			permissions = nil
		}

		return formatNamespaceInfoCompact(tenant, namespace, policies, permissions), nil
	}

	// 执行批量查询
	results := batchQuery(namespaces, queryFunc)
	
	// 格式化结果
	output := formatBatchResults(results, "Namespace Info")
	return mcp.NewToolResultText(output), nil
}

// handleGetTopicInfo 处理获取主题详细信息的请求，支持批量查询
func (s *PulsarServer) handleGetTopicInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理get_topic_info请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "get_topic_info"),
		log.String(common.FieldOperation, "get_info"))

	// 获取客户端
	client, err := s.getClient(ctx)
	if err != nil {
		if common.IsNoConfigError(err) {
			return mcp.NewToolResultError("Error: No Pulsar configurations found. Please add a Pulsar instance first using 'add_pulsar' tool."), nil
		}
		log.Error(ctx, "获取Pulsar客户端失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	// 获取参数
	topics := getStringArray(request, "topics")
	persistent := request.GetBool("persistent", true)
	
	// 优先使用批量查询
	if len(topics) > 0 {
		return s.handleBatchTopicInfo(ctx, client, topics, persistent)
	}
	
	// 单个查询（向后兼容）
	topic := request.GetString("topic", "")
	if topic == "" {
		return mcp.NewToolResultError("Error: topic or topics parameter is required"), nil
	}
	
	return s.handleSingleTopicInfo(ctx, client, topic, persistent)
}

// handleSingleTopicInfo 处理单个主题信息查询
func (s *PulsarServer) handleSingleTopicInfo(ctx context.Context, client *pulsaradmin.Client, topic string, persistent bool) (*mcp.CallToolResult, error) {
	// 获取当前租户和命名空间
	tenant, namespace, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant/namespace: %s", err.Error())), nil
	}

	// 获取主题元数据（尝试获取分区主题元数据）
	metadata, err := client.Topics().GetPartitionedMetadata(ctx, tenant, namespace, topic, persistent)
	if err != nil {
		log.Warn(ctx, "获取分区主题元数据失败，可能是非分区主题",
			log.String("topic", topic),
			log.Bool("persistent", persistent),
			log.String("error", err.Error()))
		// 如果获取分区元数据失败，设为 nil
		metadata = nil
	}

	// 获取主题统计信息（可选，如果失败不影响主要功能）
	stats, err := client.Topics().GetStats(ctx, tenant, namespace, topic, persistent)
	if err != nil {
		log.Warn(ctx, "获取主题统计信息失败",
			log.String("topic", topic),
			log.Bool("persistent", persistent),
			log.String("error", err.Error()))
		// 统计信息获取失败不影响主要功能
		stats = nil
	}

	output := formatTopicInfoCompact(tenant, namespace, topic, persistent, metadata, stats)
	return mcp.NewToolResultText(output), nil
}

// handleBatchTopicInfo 处理批量主题信息查询
func (s *PulsarServer) handleBatchTopicInfo(ctx context.Context, client *pulsaradmin.Client, topics []string, persistent bool) (*mcp.CallToolResult, error) {
	// 获取当前租户和命名空间
	tenant, namespace, err := s.getTenantNamespace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant/namespace: %s", err.Error())), nil
	}

	// 创建查询函数
	queryFunc := func(topic string) (string, error) {
		// 获取主题元数据
		metadata, err := client.Topics().GetPartitionedMetadata(ctx, tenant, namespace, topic, persistent)
		if err != nil {
			log.Warn(ctx, "获取分区主题元数据失败",
				log.String("topic", topic),
				log.Bool("persistent", persistent),
				log.String("error", err.Error()))
			metadata = nil
		}

		// 获取主题统计信息
		stats, err := client.Topics().GetStats(ctx, tenant, namespace, topic, persistent)
		if err != nil {
			log.Warn(ctx, "获取主题统计信息失败",
				log.String("topic", topic),
				log.Bool("persistent", persistent),
				log.String("error", err.Error()))
			stats = nil
		}

		return formatTopicInfoCompact(tenant, namespace, topic, persistent, metadata, stats), nil
	}

	// 执行批量查询
	results := batchQuery(topics, queryFunc)
	
	// 格式化结果
	output := formatBatchResults(results, "Topic Info")
	return mcp.NewToolResultText(output), nil
}

// handleGetTenantInfo 处理获取租户详细信息的请求，支持批量查询
func (s *PulsarServer) handleGetTenantInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理get_tenant_info请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "get_tenant_info"),
		log.String(common.FieldOperation, "get_info"))

	// 获取客户端
	client, err := s.getClient(ctx)
	if err != nil {
		if common.IsNoConfigError(err) {
			return mcp.NewToolResultError("Error: No Pulsar configurations found. Please add a Pulsar instance first using 'add_pulsar' tool."), nil
		}
		log.Error(ctx, "获取Pulsar客户端失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	// 获取参数
	tenants := getStringArray(request, "tenants")
	
	// 优先使用批量查询
	if len(tenants) > 0 {
		return s.handleBatchTenantInfo(ctx, client, tenants)
	}
	
	// 单个查询（向后兼容）
	tenant := request.GetString("tenant", "")
	if tenant == "" {
		defaultTenant, _, err := s.getTenantNamespace()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error getting default tenant: %s", err.Error())), nil
		}
		tenant = defaultTenant
	}
	
	return s.handleSingleTenantInfo(ctx, client, tenant)
}

// handleSingleTenantInfo 处理单个租户信息查询
func (s *PulsarServer) handleSingleTenantInfo(ctx context.Context, client *pulsaradmin.Client, tenant string) (*mcp.CallToolResult, error) {
	// 获取租户信息
	tenantInfo, err := client.Tenants().Get(ctx, tenant)
	if err != nil {
		log.Error(ctx, "获取租户信息失败",
			log.String("tenant", tenant),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error getting tenant info: %s", err.Error())), nil
	}

	// 获取租户下的命名空间列表
	namespaces, err := client.Namespaces().List(ctx, tenant)
	if err != nil {
		log.Warn(ctx, "获取租户命名空间失败",
			log.String("tenant", tenant),
			log.String("error", err.Error()))
		// 命名空间获取失败不影响主要功能
		namespaces = nil
	}

	output := formatTenantInfoCompact(tenant, tenantInfo, namespaces)
	return mcp.NewToolResultText(output), nil
}

// handleBatchTenantInfo 处理批量租户信息查询
func (s *PulsarServer) handleBatchTenantInfo(ctx context.Context, client *pulsaradmin.Client, tenants []string) (*mcp.CallToolResult, error) {
	// 创建查询函数
	queryFunc := func(tenant string) (string, error) {
		// 获取租户信息
		tenantInfo, err := client.Tenants().Get(ctx, tenant)
		if err != nil {
			return "", err
		}

		// 获取租户下的命名空间列表
		namespaces, err := client.Namespaces().List(ctx, tenant)
		if err != nil {
			log.Warn(ctx, "获取租户命名空间失败",
				log.String("tenant", tenant),
				log.String("error", err.Error()))
			namespaces = nil
		}

		return formatTenantInfoCompact(tenant, tenantInfo, namespaces), nil
	}

	// 执行批量查询
	results := batchQuery(tenants, queryFunc)
	
	// 格式化结果
	output := formatBatchResults(results, "Tenant Info")
	return mcp.NewToolResultText(output), nil
}

// handleListBrokers 处理列出Broker的请求
func (s *PulsarServer) handleListBrokers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理list_brokers请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "list_brokers"),
		log.String(common.FieldOperation, "list"))

	// 获取客户端
	client, err := s.getClient(ctx)
	if err != nil {
		if common.IsNoConfigError(err) {
			return mcp.NewToolResultError("Error: No Pulsar configurations found. Please add a Pulsar instance first using 'add_pulsar' tool."), nil
		}
		log.Error(ctx, "获取Pulsar客户端失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	// 获取参数
	cluster := request.GetString("cluster", "")

	// 获取Broker列表
	brokers, err := client.Brokers().List(ctx, cluster)
	if err != nil {
		log.Error(ctx, "获取Broker列表失败",
			log.String("cluster", cluster),
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error getting brokers list: %s", err.Error())), nil
	}

	if len(brokers) == 0 {
		return mcp.NewToolResultText("No brokers found"), nil
	}

	output := formatListBrokersCompact(brokers, cluster)
	return mcp.NewToolResultText(output), nil
}

// handleGetBrokerInfo 处理获取Broker详细信息的请求，支持批量查询
func (s *PulsarServer) handleGetBrokerInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理get_broker_info请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "get_broker_info"),
		log.String(common.FieldOperation, "get_info"))

	// 获取客户端
	client, err := s.getClient(ctx)
	if err != nil {
		if common.IsNoConfigError(err) {
			return mcp.NewToolResultError("Error: No Pulsar configurations found. Please add a Pulsar instance first using 'add_pulsar' tool."), nil
		}
		log.Error(ctx, "获取Pulsar客户端失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	// 获取参数
	brokers := getStringArray(request, "brokers")
	
	// 优先使用批量查询
	if len(brokers) > 0 {
		return s.handleBatchBrokerInfo(ctx, client, brokers)
	}
	
	// 单个查询（向后兼容）
	broker := request.GetString("broker", "")
	if broker == "" {
		return mcp.NewToolResultError("Error: broker or brokers parameter is required"), nil
	}
	
	return s.handleSingleBrokerInfo(ctx, client, broker)
}

// handleSingleBrokerInfo 处理单个Broker信息查询
func (s *PulsarServer) handleSingleBrokerInfo(ctx context.Context, client *pulsaradmin.Client, broker string) (*mcp.CallToolResult, error) {
	// 获取所有Broker负载数据
	allLoadData, err := client.Brokers().GetLoadData(ctx)
	if err != nil {
		log.Error(ctx, "获取Broker负载数据失败",
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error getting broker load data: %s", err.Error())), nil
	}

	// 找到指定Broker的数据
	loadData, exists := allLoadData[broker]
	if !exists {
		return mcp.NewToolResultError(fmt.Sprintf("Broker '%s' not found in load data", broker)), nil
	}

	output := formatBrokerInfoCompact(broker, loadData)
	return mcp.NewToolResultText(output), nil
}

// handleBatchBrokerInfo 处理批量Broker信息查询
func (s *PulsarServer) handleBatchBrokerInfo(ctx context.Context, client *pulsaradmin.Client, brokers []string) (*mcp.CallToolResult, error) {
	// 获取所有Broker负载数据（一次性获取，避免重复调用）
	allLoadData, err := client.Brokers().GetLoadData(ctx)
	if err != nil {
		log.Error(ctx, "获取Broker负载数据失败",
			log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error getting broker load data: %s", err.Error())), nil
	}

	// 创建查询函数
	queryFunc := func(broker string) (string, error) {
		// 查找指定Broker的数据
		loadData, exists := allLoadData[broker]
		if !exists {
			return "", fmt.Errorf("Broker '%s' not found in load data", broker)
		}

		return formatBrokerInfoCompact(broker, loadData), nil
	}

	// 执行批量查询
	results := batchQuery(brokers, queryFunc)
	
	// 格式化结果
	output := formatBatchResults(results, "Broker Info")
	return mcp.NewToolResultText(output), nil
}

// handleBrokerHealthcheck 处理Broker健康检查的请求
func (s *PulsarServer) handleBrokerHealthcheck(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理broker_healthcheck请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "broker_healthcheck"),
		log.String(common.FieldOperation, "healthcheck"))

	// 获取客户端
	client, err := s.getClient(ctx)
	if err != nil {
		if common.IsNoConfigError(err) {
			return mcp.NewToolResultError("Error: No Pulsar configurations found. Please add a Pulsar instance first using 'add_pulsar' tool."), nil
		}
		log.Error(ctx, "获取Pulsar客户端失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error connecting to Pulsar: %s", err.Error())), nil
	}

	// 执行健康检查
	err = client.Brokers().Healthcheck(ctx)
	if err != nil {
		log.Error(ctx, "Broker健康检查失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Broker healthcheck failed: %s", err.Error())), nil
	}

	return mcp.NewToolResultText("✓ Broker healthcheck passed - All brokers are healthy"), nil
}

// formatNamespaceInfoCompact 格式化命名空间信息的紧凑输出
func formatNamespaceInfoCompact(tenant, namespace string, policies *pulsaradmin.NamespacePolicy, permissions map[string]pulsaradmin.PermissionActions) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Namespace Info: %s/%s\n\n", tenant, namespace))

	if policies != nil {
		result.WriteString("Policies:\n")
		if policies.MessageTTL != nil {
			result.WriteString(fmt.Sprintf("  TTL: %d seconds\n", *policies.MessageTTL))
		}
		if policies.RetentionPolicies != nil {
			result.WriteString(fmt.Sprintf("  Retention: %d min, %d MB\n", 
				policies.RetentionPolicies.RetentionTimeInMinutes,
				policies.RetentionPolicies.RetentionSizeInMB))
		}
		if policies.DeduplicationEnabled != nil {
			result.WriteString(fmt.Sprintf("  Deduplication: %t\n", *policies.DeduplicationEnabled))
		}
		if policies.MaxConsumersPerTopic != nil {
			result.WriteString(fmt.Sprintf("  Max Consumers/Topic: %d\n", *policies.MaxConsumersPerTopic))
		}
		if policies.MaxProducersPerTopic != nil {
			result.WriteString(fmt.Sprintf("  Max Producers/Topic: %d\n", *policies.MaxProducersPerTopic))
		}
	}

	if permissions != nil && len(permissions) > 0 {
		result.WriteString("\nPermissions:\n")
		for role, actions := range permissions {
			result.WriteString(fmt.Sprintf("  %s: %v\n", role, actions))
		}
	}

	return result.String()
}

// formatTopicInfoCompact 格式化主题信息的紧凑输出
func formatTopicInfoCompact(tenant, namespace, topic string, persistent bool, metadata *pulsaradmin.PartitionedTopicMetadata, stats *pulsaradmin.TopicStats) string {
	var result strings.Builder
	
	topicType := "persistent"
	if !persistent {
		topicType = "non-persistent"
	}
	
	result.WriteString(fmt.Sprintf("Topic Info: %s/%s/%s (%s)\n\n", tenant, namespace, topic, topicType))

	if metadata != nil {
		result.WriteString("Metadata:\n")
		result.WriteString(fmt.Sprintf("  Partitions: %d\n", metadata.Partitions))
	} else {
		result.WriteString("Metadata: Non-partitioned topic\n")
	}

	if stats != nil {
		result.WriteString("\nCurrent Stats:\n")
		result.WriteString(fmt.Sprintf("  In Rate: %.2f msg/s (%.2f MB/s)\n", stats.MsgRateIn, stats.MsgThroughputIn/1024/1024))
		result.WriteString(fmt.Sprintf("  Out Rate: %.2f msg/s (%.2f MB/s)\n", stats.MsgRateOut, stats.MsgThroughputOut/1024/1024))
		result.WriteString(fmt.Sprintf("  Storage Size: %.2f MB\n", float64(stats.StorageSize)/1024/1024))
		result.WriteString(fmt.Sprintf("  Publishers: %d\n", len(stats.Publishers)))
		result.WriteString(fmt.Sprintf("  Subscriptions: %d\n", len(stats.Subscriptions)))
	}

	return result.String()
}

// formatTenantInfoCompact 格式化租户信息的紧凑输出
func formatTenantInfoCompact(tenant string, tenantInfo *pulsaradmin.TenantInfo, namespaces []string) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Tenant Info: %s\n\n", tenant))

	if tenantInfo != nil {
		if len(tenantInfo.AdminRoles) > 0 {
			result.WriteString(fmt.Sprintf("Admin Roles: %s\n", strings.Join(tenantInfo.AdminRoles, ", ")))
		}
		if len(tenantInfo.AllowedClusters) > 0 {
			result.WriteString(fmt.Sprintf("Allowed Clusters: %s\n", strings.Join(tenantInfo.AllowedClusters, ", ")))
		}
	}

	if namespaces != nil && len(namespaces) > 0 {
		result.WriteString(fmt.Sprintf("\nNamespaces (%d):\n", len(namespaces)))
		sort.Strings(namespaces)
		for _, ns := range namespaces {
			result.WriteString(fmt.Sprintf("  - %s\n", ns))
		}
	}

	return result.String()
}

// formatListBrokersCompact 格式化Broker列表的紧凑输出
func formatListBrokersCompact(brokers []string, cluster string) string {
	var result strings.Builder
	
	if cluster != "" {
		result.WriteString(fmt.Sprintf("Brokers in cluster '%s' (%d):\n", cluster, len(brokers)))
	} else {
		result.WriteString(fmt.Sprintf("Active Brokers (%d):\n", len(brokers)))
	}

	sort.Strings(brokers)
	for i, broker := range brokers {
		result.WriteString(fmt.Sprintf("  %d. %s\n", i+1, broker))
	}

	return result.String()
}

// formatBrokerInfoCompact 格式化Broker详细信息的紧凑输出
func formatBrokerInfoCompact(broker string, loadData pulsaradmin.BrokerLoadData) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Broker Info: %s\n\n", broker))

	local := loadData.LocalBrokerData
	result.WriteString("Service URLs:\n")
	result.WriteString(fmt.Sprintf("  Pulsar: %s\n", local.PulsarServiceUrl))
	if local.PulsarServiceUrlTls != "" {
		result.WriteString(fmt.Sprintf("  Pulsar TLS: %s\n", local.PulsarServiceUrlTls))
	}
	result.WriteString(fmt.Sprintf("  Web: %s\n", local.WebServiceUrl))
	if local.WebServiceUrlTls != "" {
		result.WriteString(fmt.Sprintf("  Web TLS: %s\n", local.WebServiceUrlTls))
	}

	result.WriteString("\nResource Usage:\n")
	result.WriteString(fmt.Sprintf("  CPU: %.1f%% (limit: %.1f%%)\n", local.CPU.Usage, local.CPU.Limit))
	result.WriteString(fmt.Sprintf("  Memory: %.1f%% (limit: %.1f%%)\n", local.Memory.Usage, local.Memory.Limit))
	result.WriteString(fmt.Sprintf("  Direct Memory: %.1f%% (limit: %.1f%%)\n", local.DirectMemory.Usage, local.DirectMemory.Limit))

	result.WriteString("\nLoad Stats:\n")
	result.WriteString(fmt.Sprintf("  Topics: %d\n", local.NumTopics))
	result.WriteString(fmt.Sprintf("  Bundles: %d\n", local.NumBundles))
	result.WriteString(fmt.Sprintf("  Producers: %d\n", local.NumProducers))
	result.WriteString(fmt.Sprintf("  Consumers: %d\n", local.NumConsumers))

	return result.String()
}

// getStringArray 从请求参数中获取字符串数组
func getStringArray(request mcp.CallToolRequest, key string) []string {
	var result []string
	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if param, exists := args[key]; exists {
			if slice, ok := param.([]interface{}); ok {
				for _, item := range slice {
					if str, ok := item.(string); ok {
						result = append(result, str)
					}
				}
			}
		}
	}
	return result
}

// BatchQueryResult 批量查询结果
type BatchQueryResult struct {
	ID      string // 查询项目的标识符
	Success bool   // 是否成功
	Content string // 成功时的内容
	Error   string // 失败时的错误信息
}

// batchQuery 执行批量查询的通用函数
func batchQuery(items []string, queryFunc func(string) (string, error)) []BatchQueryResult {
	const maxConcurrency = 5 // 最大并发数
	
	sem := make(chan bool, maxConcurrency)
	results := make([]BatchQueryResult, len(items))
	
	var wg sync.WaitGroup
	
	for i, item := range items {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()
			
			sem <- true // 获取信号量
			defer func() { <-sem }() // 释放信号量
			
			content, err := queryFunc(id)
			if err != nil {
				results[index] = BatchQueryResult{
					ID:      id,
					Success: false,
					Error:   err.Error(),
				}
			} else {
				results[index] = BatchQueryResult{
					ID:      id,
					Success: true,
					Content: content,
				}
			}
		}(i, item)
	}
	
	wg.Wait()
	return results
}

// formatBatchResults 格式化批量查询结果
func formatBatchResults(results []BatchQueryResult, queryType string) string {
	var result strings.Builder
	
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	
	total := len(results)
	failedCount := total - successCount
	
	result.WriteString(fmt.Sprintf("%s (batch: %d requested, %d successful, %d failed):\n\n", 
		queryType, total, successCount, failedCount))
	
	for _, r := range results {
		if r.Success {
			result.WriteString(fmt.Sprintf("=== %s ===\n", r.ID))
			result.WriteString(r.Content)
			result.WriteString("\n")
		} else {
			result.WriteString(fmt.Sprintf("=== %s (FAILED) ===\n", r.ID))
			result.WriteString(fmt.Sprintf("Error: %s\n\n", r.Error))
		}
	}
	
	// 添加统计摘要
	percentage := float64(successCount) / float64(total) * 100
	result.WriteString(fmt.Sprintf("Summary: %d/%d successful (%.1f%%)", 
		successCount, total, percentage))
	
	return result.String()
}

// handleUpdateConfig 处理更新配置的请求
func (s *PulsarServer) handleUpdateConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理update_config请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "update_config"),
		log.String(common.FieldOperation, "update"))

	// 安全检查
	if err := s.validatePulsarOperationSecurity(ctx, "UPDATE_CONFIG"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	name := request.GetString("name", "")
	if name == "" {
		// 使用当前激活的实例
		name = viper.GetString("active_database")
		if name == "" {
			name = "default"
		}
	}

	// 检查实例是否存在
	pulsarKey := fmt.Sprintf("databases.%s", name)
	if !viper.IsSet(pulsarKey) {
		return mcp.NewToolResultError(fmt.Sprintf("Error: Pulsar instance '%s' does not exist", name)), nil
	}

	// 收集需要更新的配置
	updates := make(map[string]interface{})
	var updatedFields []string

	if adminURL := request.GetString("admin_url", ""); adminURL != "" {
		updates[pulsarKey+".admin_url"] = adminURL
		updatedFields = append(updatedFields, "admin_url")
	}

	if tenant := request.GetString("tenant", ""); tenant != "" {
		updates[pulsarKey+".tenant"] = tenant
		updatedFields = append(updatedFields, "tenant")
	}

	if namespace := request.GetString("namespace", ""); namespace != "" {
		updates[pulsarKey+".namespace"] = namespace
		updatedFields = append(updatedFields, "namespace")
	}

	if username := request.GetString("username", ""); username != "" {
		updates[pulsarKey+".username"] = username
		updatedFields = append(updatedFields, "username")
	}

	if password := request.GetString("password", ""); password != "" {
		updates[pulsarKey+".password"] = password
		updatedFields = append(updatedFields, "password")
	}

	if timeout := request.GetString("timeout", ""); timeout != "" {
		updates[pulsarKey+".timeout"] = timeout
		updatedFields = append(updatedFields, "timeout")
	}

	if len(updates) == 0 {
		return mcp.NewToolResultError("Error: No configuration properties to update"), nil
	}

	// 应用更新
	for key, value := range updates {
		viper.Set(key, value)
	}

	// 保存配置到文件
	if err := viper.WriteConfig(); err != nil {
		log.Error(ctx, "保存配置失败", log.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error saving configuration: %s", err.Error())), nil
	}

	// 清除连接缓存，强制重新连接
	if s.pulsarPool != nil {
		s.pulsarPool.CloseConnection(name)
		log.Info(ctx, "清除连接缓存", 
			log.String("instance", name),
			log.String(common.FieldProvider, "pulsar"))
	}

	// 格式化输出
	output := fmt.Sprintf("✅ Configuration updated for Pulsar instance '%s'\n", name)
	output += fmt.Sprintf("Updated fields (%d): %s\n", len(updatedFields), strings.Join(updatedFields, ", "))
	
	// 显示更新后的关键配置
	output += "\nCurrent configuration:\n"
	if adminURL := viper.GetString(pulsarKey + ".admin_url"); adminURL != "" {
		output += fmt.Sprintf("  Admin URL: %s\n", adminURL)
	}
	if tenant := viper.GetString(pulsarKey + ".tenant"); tenant != "" {
		output += fmt.Sprintf("  Tenant: %s\n", tenant)
	}
	if namespace := viper.GetString(pulsarKey + ".namespace"); namespace != "" {
		output += fmt.Sprintf("  Namespace: %s\n", namespace)
	}
	if username := viper.GetString(pulsarKey + ".username"); username != "" {
		output += fmt.Sprintf("  Username: %s\n", username)
	}
	if timeout := viper.GetString(pulsarKey + ".timeout"); timeout != "" {
		output += fmt.Sprintf("  Timeout: %s\n", timeout)
	}

	log.Info(ctx, "配置更新成功", 
		log.String("instance", name),
		log.String("fields", strings.Join(updatedFields, ",")))

	return mcp.NewToolResultText(output[:len(output)-1]), nil // 去掉最后的换行符
}

// handleGetConfigDetails 处理获取配置详情的请求
func (s *PulsarServer) handleGetConfigDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理get_config_details请求",
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldTool, "get_config_details"),
		log.String(common.FieldOperation, "get_details"))

	name := request.GetString("name", "")
	includeSensitive := request.GetBool("include_sensitive", false)

	pulsarInstances := viper.GetStringMap("databases")
	if len(pulsarInstances) == 0 {
		return mcp.NewToolResultText("No Pulsar instances configured"), nil
	}

	activePulsar := viper.GetString("active_database")
	if activePulsar == "" {
		activePulsar = common.DefaultInstanceName
	}

	var output strings.Builder

	if name == "" {
		name = activePulsar
	}

	if name == "all" {
		// 显示所有实例的配置
		output.WriteString("📋 All Pulsar Instance Configurations\n")
		output.WriteString("=====================================\n\n")

		for instanceName, _ := range pulsarInstances {
			output.WriteString(s.formatInstanceConfig(instanceName, instanceName == activePulsar, includeSensitive))
			output.WriteString("\n")
		}
	} else {
		// 显示指定实例的配置
		if _, exists := pulsarInstances[name]; !exists {
			return mcp.NewToolResultError(fmt.Sprintf("Error: Pulsar instance '%s' does not exist", name)), nil
		}
		
		output.WriteString(fmt.Sprintf("📋 Pulsar Instance Configuration: %s\n", name))
		output.WriteString("========================================\n\n")
		output.WriteString(s.formatInstanceConfig(name, name == activePulsar, includeSensitive))
	}

	return mcp.NewToolResultText(output.String()), nil
}

// formatInstanceConfig 格式化实例配置信息
func (s *PulsarServer) formatInstanceConfig(name string, isActive bool, includeSensitive bool) string {
	pulsarKey := fmt.Sprintf("databases.%s", name)
	
	var output strings.Builder
	
	// 实例名称和状态
	status := "inactive"
	if isActive {
		status = "🟢 ACTIVE"
	} else {
		status = "⚪ inactive"
	}
	output.WriteString(fmt.Sprintf("Instance: %s (%s)\n", name, status))
	
	// 基本配置
	if adminURL := viper.GetString(pulsarKey + ".admin_url"); adminURL != "" {
		output.WriteString(fmt.Sprintf("  Admin URL: %s\n", adminURL))
	}
	
	if tenant := viper.GetString(pulsarKey + ".tenant"); tenant != "" {
		output.WriteString(fmt.Sprintf("  Tenant: %s\n", tenant))
	}
	
	if namespace := viper.GetString(pulsarKey + ".namespace"); namespace != "" {
		output.WriteString(fmt.Sprintf("  Namespace: %s\n", namespace))
	}
	
	// 认证信息
	if username := viper.GetString(pulsarKey + ".username"); username != "" {
		output.WriteString(fmt.Sprintf("  Username: %s\n", username))
	}
	
	if password := viper.GetString(pulsarKey + ".password"); password != "" {
		if includeSensitive {
			output.WriteString(fmt.Sprintf("  Password: %s\n", password))
		} else {
			output.WriteString("  Password: *** (hidden)\n")
		}
	}
	
	// 连接配置
	if timeout := viper.GetString(pulsarKey + ".timeout"); timeout != "" {
		output.WriteString(fmt.Sprintf("  Timeout: %s\n", timeout))
	}
	
	// 连接状态检查（仅对激活的实例）
	if isActive {
		if client, err := s.getClient(context.Background()); err == nil && client != nil {
			output.WriteString("  Connection: ✅ Available\n")
		} else {
			output.WriteString(fmt.Sprintf("  Connection: ❌ Failed (%s)\n", err.Error()))
		}
	}
	
	return output.String()
}

// validatePulsarOperationSecurity 验证Pulsar操作的安全性
func (s *PulsarServer) validatePulsarOperationSecurity(ctx context.Context, operation string) error {
	operation = strings.ToUpper(operation)

	switch operation {
	case "CREATE_TENANT", "CREATE_NAMESPACE", "CREATE_TOPIC", "CREATE_SUBSCRIPTION":
		if s.disableCreate {
			return errors.New(fmt.Sprintf("%s操作已被禁用", operation))
		}
	case "DELETE_TENANT", "DELETE_NAMESPACE", "DELETE_TOPIC", "DELETE_SUBSCRIPTION":
		if s.disableDrop {
			return errors.New(fmt.Sprintf("%s操作已被禁用", operation))
		}
	case "UPDATE_CONFIG":
		if s.disableUpdate {
			return errors.New("UPDATE_CONFIG操作已被禁用")
		}
	}

	log.Debug(ctx, "Pulsar操作安全检查通过",
		log.String("operation", operation),
		log.String(common.FieldProvider, "pulsar"),
		log.String(common.FieldOperation, "validate_security"))

	return nil
}