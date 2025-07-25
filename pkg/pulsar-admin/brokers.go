package pulsaradmin

import (
	"context"
	"fmt"
	"net/http"
)

// BrokersAPI Broker管理 API
type BrokersAPI struct {
	client *Client
}

// BrokerInfo Broker信息
type BrokerInfo struct {
	ServiceUrl    string `json:"serviceUrl"`
	ServiceUrlTls string `json:"serviceUrlTls,omitempty"`
	BrokerUrl     string `json:"brokerUrl"`
	PulsarVersion string `json:"pulsarVersion,omitempty"`
	BuildVersion  string `json:"version,omitempty"`
}

// BrokerLoadData Broker负载数据
type BrokerLoadData struct {
	Name                     string                 `json:"name"`
	BrokerHostUsage          map[string]interface{} `json:"brokerHostUsage"`
	LocalBrokerData          LocalBrokerData        `json:"localBrokerData"`
	PreAllocatedBundleData   map[string]interface{} `json:"preAllocatedBundleData"`
	TimeAverageMessageData   map[string]interface{} `json:"timeAverageMessageData"`
}

// LocalBrokerData 本地Broker数据
type LocalBrokerData struct {
	WebServiceUrl          string  `json:"webServiceUrl"`
	WebServiceUrlTls       string  `json:"webServiceUrlTls,omitempty"`
	PulsarServiceUrl       string  `json:"pulsarServiceUrl"`
	PulsarServiceUrlTls    string  `json:"pulsarServiceUrlTls,omitempty"`
	CPU                    ResourceUsage `json:"cpu"`
	Memory                 ResourceUsage `json:"memory"`
	DirectMemory           ResourceUsage `json:"directMemory"`
	BandwidthIn            ResourceUsage `json:"bandwidthIn"`
	BandwidthOut           ResourceUsage `json:"bandwidthOut"`
	LastUpdate             int64   `json:"lastUpdate"`
	LastStats              map[string]interface{} `json:"lastStats"`
	NumTopics              int     `json:"numTopics"`
	NumBundles             int     `json:"numBundles"`
	NumConsumers           int     `json:"numConsumers"`
	NumProducers           int     `json:"numProducers"`
	Protocols              map[string]interface{} `json:"protocols"`
	PersistentTopicsEnabled bool   `json:"persistentTopicsEnabled"`
	NonPersistentTopicsEnabled bool `json:"nonPersistentTopicsEnabled"`
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	Usage float64 `json:"usage"`
	Limit float64 `json:"limit"`
}

// NamespaceOwnership 命名空间归属信息
type NamespaceOwnership struct {
	BrokerAssignment string `json:"brokerAssignment"`
	IsControlled     bool   `json:"isControlled"`
}

// List 获取所有活跃的Broker列表
func (b *BrokersAPI) List(ctx context.Context, cluster string) ([]string, error) {
	path := fmt.Sprintf("/admin/v2/brokers/%s", cluster)
	resp, err := b.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var brokers []string
	if err := b.client.parseResponse(resp, &brokers); err != nil {
		return nil, err
	}

	return brokers, nil
}

// GetLeaderBroker 获取Leader Broker
func (b *BrokersAPI) GetLeaderBroker(ctx context.Context) (*BrokerInfo, error) {
	resp, err := b.client.doRequest(ctx, http.MethodGet, "/admin/v2/brokers/leaderBroker", nil)
	if err != nil {
		return nil, err
	}

	var broker BrokerInfo
	if err := b.client.parseResponse(resp, &broker); err != nil {
		return nil, err
	}

	return &broker, nil
}

// GetDynamicConfig 获取Broker动态配置
func (b *BrokersAPI) GetDynamicConfig(ctx context.Context) (map[string]string, error) {
	resp, err := b.client.doRequest(ctx, http.MethodGet, "/admin/v2/brokers/configuration", nil)
	if err != nil {
		return nil, err
	}

	var config map[string]string
	if err := b.client.parseResponse(resp, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// UpdateDynamicConfig 更新Broker动态配置
func (b *BrokersAPI) UpdateDynamicConfig(ctx context.Context, configName, configValue string) error {
	path := fmt.Sprintf("/admin/v2/brokers/configuration/%s/%s", configName, configValue)
	resp, err := b.client.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetAllDynamicConfig 获取所有可用的动态配置
func (b *BrokersAPI) GetAllDynamicConfig(ctx context.Context) (map[string]string, error) {
	resp, err := b.client.doRequest(ctx, http.MethodGet, "/admin/v2/brokers/configuration/values", nil)
	if err != nil {
		return nil, err
	}

	var configs map[string]string
	if err := b.client.parseResponse(resp, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}

// GetLoadData 获取Broker负载数据
func (b *BrokersAPI) GetLoadData(ctx context.Context) (map[string]BrokerLoadData, error) {
	resp, err := b.client.doRequest(ctx, http.MethodGet, "/admin/v2/broker-stats/load-report", nil)
	if err != nil {
		return nil, err
	}

	var loadData map[string]BrokerLoadData
	if err := b.client.parseResponse(resp, &loadData); err != nil {
		return nil, err
	}

	return loadData, nil
}

// GetTopics 获取Broker上的所有主题
func (b *BrokersAPI) GetTopics(ctx context.Context, cluster, brokerWebServiceUrl string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/admin/v2/brokers/%s/%s/ownedNamespaces", cluster, brokerWebServiceUrl)
	resp, err := b.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var topics map[string]interface{}
	if err := b.client.parseResponse(resp, &topics); err != nil {
		return nil, err
	}

	return topics, nil
}

// Healthcheck 检查Broker健康状态
func (b *BrokersAPI) Healthcheck(ctx context.Context) error {
	resp, err := b.client.doRequest(ctx, http.MethodGet, "/admin/v2/brokers/health", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 如果状态码是200，则表示健康
	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("broker健康检查失败: %d", resp.StatusCode)
}

// GetVersion 获取Broker版本信息
func (b *BrokersAPI) GetVersion(ctx context.Context) (map[string]string, error) {
	resp, err := b.client.doRequest(ctx, http.MethodGet, "/admin/v2/brokers/version", nil)
	if err != nil {
		return nil, err
	}

	var version map[string]string
	if err := b.client.parseResponse(resp, &version); err != nil {
		return nil, err
	}

	return version, nil
}

// GetNamespaceOwnership 获取命名空间的归属信息
func (b *BrokersAPI) GetNamespaceOwnership(ctx context.Context, tenant, namespace string) (*NamespaceOwnership, error) {
	path := fmt.Sprintf("/admin/v2/brokers/ownership/%s/%s", tenant, namespace)
	resp, err := b.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var ownership NamespaceOwnership
	if err := b.client.parseResponse(resp, &ownership); err != nil {
		return nil, err
	}

	return &ownership, nil
}

// ShutdownBroker 关闭Broker（仅限管理员）
func (b *BrokersAPI) ShutdownBroker(ctx context.Context) error {
	resp, err := b.client.doRequest(ctx, http.MethodPost, "/admin/v2/brokers/shutdown", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}