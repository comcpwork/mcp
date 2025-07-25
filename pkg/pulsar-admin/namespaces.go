package pulsaradmin

import (
	"context"
	"fmt"
	"net/http"
)

// NamespacesAPI 命名空间管理 API
type NamespacesAPI struct {
	client *Client
}

// NamespacePolicy 命名空间策略
type NamespacePolicy struct {
	Replication              *ReplicationPolicy      `json:"replication,omitempty"`
	MessageTTL               *int                    `json:"message_ttl_in_seconds,omitempty"`
	RetentionPolicies        *RetentionPolicies      `json:"retention_policies,omitempty"`
	DeduplicationEnabled     *bool                   `json:"deduplication_enabled,omitempty"`
	PersistencePolicies      *PersistencePolicies    `json:"persistence,omitempty"`
	DispatchRate             *DispatchRate           `json:"dispatch_rate,omitempty"`
	SubscriptionDispatchRate *DispatchRate           `json:"subscription_dispatch_rate,omitempty"`
	MaxConsumersPerTopic     *int                    `json:"max_consumers_per_topic,omitempty"`
	MaxConsumersPerSubscription *int                 `json:"max_consumers_per_subscription,omitempty"`
	MaxProducersPerTopic     *int                    `json:"max_producers_per_topic,omitempty"`
}

// ReplicationPolicy 复制策略
type ReplicationPolicy struct {
	Clusters []string `json:"clusters"`
}

// RetentionPolicies 保留策略
type RetentionPolicies struct {
	RetentionTimeInMinutes int `json:"retentionTimeInMinutes"`
	RetentionSizeInMB      int `json:"retentionSizeInMB"`
}

// PersistencePolicies 持久化策略
type PersistencePolicies struct {
	BookkeeperEnsemble          int     `json:"bookkeeperEnsemble"`
	BookkeeperWriteQuorum       int     `json:"bookkeeperWriteQuorum"`
	BookkeeperAckQuorum         int     `json:"bookkeeperAckQuorum"`
	ManagedLedgerMaxMarkDeleteRate float64 `json:"managedLedgerMaxMarkDeleteRate"`
}

// DispatchRate 消息分发速率
type DispatchRate struct {
	DispatchThrottlingRatePerTopicInMsg  int `json:"dispatchThrottlingRatePerTopicInMsg"`
	DispatchThrottlingRatePerTopicInByte int `json:"dispatchThrottlingRatePerTopicInByte"`
	RatePeriodInSecond                   int `json:"ratePeriodInSecond"`
}

// PermissionActions 权限操作
type PermissionActions []string

// List 获取租户下的所有命名空间
func (n *NamespacesAPI) List(ctx context.Context, tenant string) ([]string, error) {
	path := fmt.Sprintf("/admin/v2/namespaces/%s", tenant)
	resp, err := n.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var namespaces []string
	if err := n.client.parseResponse(resp, &namespaces); err != nil {
		return nil, err
	}

	return namespaces, nil
}

// GetPolicies 获取命名空间策略
func (n *NamespacesAPI) GetPolicies(ctx context.Context, tenant, namespace string) (*NamespacePolicy, error) {
	path := fmt.Sprintf("/admin/v2/namespaces/%s/%s", tenant, namespace)
	resp, err := n.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var policy NamespacePolicy
	if err := n.client.parseResponse(resp, &policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

// Create 创建命名空间
func (n *NamespacesAPI) Create(ctx context.Context, tenant, namespace string, policy *NamespacePolicy) error {
	path := fmt.Sprintf("/admin/v2/namespaces/%s/%s", tenant, namespace)
	resp, err := n.client.doRequest(ctx, http.MethodPut, path, policy)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Delete 删除命名空间
func (n *NamespacesAPI) Delete(ctx context.Context, tenant, namespace string) error {
	path := fmt.Sprintf("/admin/v2/namespaces/%s/%s", tenant, namespace)
	resp, err := n.client.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GrantPermission 授予权限
func (n *NamespacesAPI) GrantPermission(ctx context.Context, tenant, namespace, role string, actions PermissionActions) error {
	path := fmt.Sprintf("/admin/v2/namespaces/%s/%s/permissions/%s", tenant, namespace, role)
	
	requestBody := map[string]interface{}{
		"actions": actions,
	}
	
	resp, err := n.client.doRequest(ctx, http.MethodPost, path, requestBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// RevokePermission 撤销权限
func (n *NamespacesAPI) RevokePermission(ctx context.Context, tenant, namespace, role string) error {
	path := fmt.Sprintf("/admin/v2/namespaces/%s/%s/permissions/%s", tenant, namespace, role)
	resp, err := n.client.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetPermissions 获取权限
func (n *NamespacesAPI) GetPermissions(ctx context.Context, tenant, namespace string) (map[string]PermissionActions, error) {
	path := fmt.Sprintf("/admin/v2/namespaces/%s/%s/permissions", tenant, namespace)
	resp, err := n.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var permissions map[string]PermissionActions
	if err := n.client.parseResponse(resp, &permissions); err != nil {
		return nil, err
	}

	return permissions, nil
}

// SetRetention 设置保留策略
func (n *NamespacesAPI) SetRetention(ctx context.Context, tenant, namespace string, retention *RetentionPolicies) error {
	path := fmt.Sprintf("/admin/v2/namespaces/%s/%s/retention", tenant, namespace)
	resp, err := n.client.doRequest(ctx, http.MethodPost, path, retention)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetRetention 获取保留策略
func (n *NamespacesAPI) GetRetention(ctx context.Context, tenant, namespace string) (*RetentionPolicies, error) {
	path := fmt.Sprintf("/admin/v2/namespaces/%s/%s/retention", tenant, namespace)
	resp, err := n.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var retention RetentionPolicies
	if err := n.client.parseResponse(resp, &retention); err != nil {
		return nil, err
	}

	return &retention, nil
}

// SetDispatchRate 设置消息分发速率
func (n *NamespacesAPI) SetDispatchRate(ctx context.Context, tenant, namespace string, dispatchRate *DispatchRate) error {
	path := fmt.Sprintf("/admin/v2/namespaces/%s/%s/dispatchRate", tenant, namespace)
	resp, err := n.client.doRequest(ctx, http.MethodPost, path, dispatchRate)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetDispatchRate 获取消息分发速率
func (n *NamespacesAPI) GetDispatchRate(ctx context.Context, tenant, namespace string) (*DispatchRate, error) {
	path := fmt.Sprintf("/admin/v2/namespaces/%s/%s/dispatchRate", tenant, namespace)
	resp, err := n.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var dispatchRate DispatchRate
	if err := n.client.parseResponse(resp, &dispatchRate); err != nil {
		return nil, err
	}

	return &dispatchRate, nil
}