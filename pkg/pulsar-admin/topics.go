package pulsaradmin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// TopicsAPI 主题管理 API
type TopicsAPI struct {
	client *Client
}

// TopicStats 主题统计信息
type TopicStats struct {
	MsgRateIn                   float64                 `json:"msgRateIn"`
	MsgThroughputIn             float64                 `json:"msgThroughputIn"`
	MsgRateOut                  float64                 `json:"msgRateOut"`
	MsgThroughputOut            float64                 `json:"msgThroughputOut"`
	BytesInCounter              int64                   `json:"bytesInCounter"`
	MsgInCounter                int64                   `json:"msgInCounter"`
	BytesOutCounter             int64                   `json:"bytesOutCounter"`
	MsgOutCounter               int64                   `json:"msgOutCounter"`
	AverageMsgSize              float64                 `json:"averageMsgSize"`
	StorageSize                 int64                   `json:"storageSize"`
	BacklogSize                 int64                   `json:"backlogSize"`
	Publishers                  []PublisherStats        `json:"publishers"`
	Subscriptions              map[string]TopicSubscriptionStats `json:"subscriptions"`
	Replication                map[string]interface{}  `json:"replication"`
	DeduplicationStatus        string                  `json:"deduplicationStatus,omitempty"`
}

// PublisherStats 发布者统计信息
type PublisherStats struct {
	MsgRateIn         float64 `json:"msgRateIn"`
	MsgThroughputIn   float64 `json:"msgThroughputIn"`
	AverageMsgSize    float64 `json:"averageMsgSize"`
	ProducerId        int64   `json:"producerId"`
	ProducerName      string  `json:"producerName,omitempty"`
	Address           string  `json:"address,omitempty"`
	ConnectedSince    string  `json:"connectedSince,omitempty"`
}

// TopicSubscriptionStats 主题中的订阅统计信息
type TopicSubscriptionStats struct {
	MsgRateOut              float64         `json:"msgRateOut"`
	MsgThroughputOut        float64         `json:"msgThroughputOut"`
	BytesOutCounter         int64           `json:"bytesOutCounter"`
	MsgOutCounter           int64           `json:"msgOutCounter"`
	MsgRateRedeliver        float64         `json:"msgRateRedeliver"`
	MsgBacklog              int64           `json:"msgBacklog"`
	MsgBacklogNoDelayed     int64           `json:"msgBacklogNoDelayed"`
	BlockedSubscriptionOnUnackedMsgs bool  `json:"blockedSubscriptionOnUnackedMsgs"`
	MsgDelayed              int64           `json:"msgDelayed"`
	UnackedMessages         int64           `json:"unackedMessages"`
	Type                    string          `json:"type,omitempty"`
	MsgRateExpired          float64         `json:"msgRateExpired"`
	LastExpireTimestamp     int64           `json:"lastExpireTimestamp"`
	LastConsumedFlowTimestamp int64         `json:"lastConsumedFlowTimestamp"`
	LastConsumedTimestamp   int64           `json:"lastConsumedTimestamp"`
	LastAckedTimestamp      int64           `json:"lastAckedTimestamp"`
	Consumers               []ConsumerStats `json:"consumers"`
	IsDurable               bool            `json:"isDurable"`
	IsReplicated            bool            `json:"isReplicated"`
}

// ConsumerStats 消费者统计信息
type ConsumerStats struct {
	MsgRateOut              float64 `json:"msgRateOut"`
	MsgThroughputOut        float64 `json:"msgThroughputOut"`
	BytesOutCounter         int64   `json:"bytesOutCounter"`
	MsgOutCounter           int64   `json:"msgOutCounter"`
	MsgRateRedeliver        float64 `json:"msgRateRedeliver"`
	ConsumerName            string  `json:"consumerName"`
	AvailablePermits        int     `json:"availablePermits"`
	UnackedMessages         int64   `json:"unackedMessages"`
	BlockedConsumerOnUnackedMsgs bool `json:"blockedConsumerOnUnackedMsgs"`
	Address                 string  `json:"address,omitempty"`
	ConnectedSince          string  `json:"connectedSince,omitempty"`
}

// PartitionedTopicMetadata 分区主题元数据
type PartitionedTopicMetadata struct {
	Partitions int `json:"partitions"`
}

// List 获取命名空间下的所有主题
func (t *TopicsAPI) List(ctx context.Context, tenant, namespace string, persistent bool) ([]string, error) {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s", schema, tenant, namespace)
	resp, err := t.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var topics []string
	if err := t.client.parseResponse(resp, &topics); err != nil {
		return nil, err
	}

	return topics, nil
}

// CreateNonPartitioned 创建非分区主题
func (t *TopicsAPI) CreateNonPartitioned(ctx context.Context, tenant, namespace, topic string, persistent bool) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s", schema, tenant, namespace, topic)
	resp, err := t.client.doRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// CreatePartitioned 创建分区主题
func (t *TopicsAPI) CreatePartitioned(ctx context.Context, tenant, namespace, topic string, partitions int, persistent bool) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/partitions", schema, tenant, namespace, topic)
	
	// 请求体是分区数的字符串
	resp, err := t.client.doRequest(ctx, http.MethodPut, path, strconv.Itoa(partitions))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// UpdatePartitions 更新分区数
func (t *TopicsAPI) UpdatePartitions(ctx context.Context, tenant, namespace, topic string, partitions int, persistent bool) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/partitions", schema, tenant, namespace, topic)
	
	// 请求体是新分区数的字符串
	resp, err := t.client.doRequest(ctx, http.MethodPost, path, strconv.Itoa(partitions))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetPartitionedMetadata 获取分区主题元数据
func (t *TopicsAPI) GetPartitionedMetadata(ctx context.Context, tenant, namespace, topic string, persistent bool) (*PartitionedTopicMetadata, error) {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/partitions", schema, tenant, namespace, topic)
	resp, err := t.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var metadata PartitionedTopicMetadata
	if err := t.client.parseResponse(resp, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// Delete 删除主题
func (t *TopicsAPI) Delete(ctx context.Context, tenant, namespace, topic string, persistent bool) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s", schema, tenant, namespace, topic)
	resp, err := t.client.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// DeletePartitioned 删除分区主题
func (t *TopicsAPI) DeletePartitioned(ctx context.Context, tenant, namespace, topic string, persistent bool) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/partitions", schema, tenant, namespace, topic)
	resp, err := t.client.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetStats 获取主题统计信息
func (t *TopicsAPI) GetStats(ctx context.Context, tenant, namespace, topic string, persistent bool) (*TopicStats, error) {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/stats", schema, tenant, namespace, topic)
	resp, err := t.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var stats TopicStats
	if err := t.client.parseResponse(resp, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetPartitionedStats 获取分区主题统计信息
func (t *TopicsAPI) GetPartitionedStats(ctx context.Context, tenant, namespace, topic string, persistent bool, perPartition bool) (*TopicStats, error) {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/partitioned-stats", schema, tenant, namespace, topic)
	if perPartition {
		path += "?perPartition=true"
	}
	
	resp, err := t.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var stats TopicStats
	if err := t.client.parseResponse(resp, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// Unload 卸载主题
func (t *TopicsAPI) Unload(ctx context.Context, tenant, namespace, topic string, persistent bool) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/unload", schema, tenant, namespace, topic)
	resp, err := t.client.doRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}