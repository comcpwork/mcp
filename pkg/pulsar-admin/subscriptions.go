package pulsaradmin

import (
	"context"
	"fmt"
	"net/http"
)

// SubscriptionsAPI 订阅管理 API
type SubscriptionsAPI struct {
	client *Client
}

// SubscriptionStats 订阅统计信息
type SubscriptionStats struct {
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

// SubscriptionType 订阅类型
type SubscriptionType string

const (
	SubscriptionTypeExclusive SubscriptionType = "Exclusive"
	SubscriptionTypeShared    SubscriptionType = "Shared"
	SubscriptionTypeFailover  SubscriptionType = "Failover"
	SubscriptionTypeKeyShared SubscriptionType = "Key_Shared"
)

// SubscriptionPosition 订阅位置
type SubscriptionPosition struct {
	LedgerId       int64 `json:"ledgerId"`
	EntryId        int64 `json:"entryId"`
	PartitionIndex int   `json:"partitionIndex,omitempty"`
}

// List 获取主题的所有订阅
func (s *SubscriptionsAPI) List(ctx context.Context, tenant, namespace, topic string, persistent bool) ([]string, error) {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscriptions", schema, tenant, namespace, topic)
	resp, err := s.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var subscriptions []string
	if err := s.client.parseResponse(resp, &subscriptions); err != nil {
		return nil, err
	}

	return subscriptions, nil
}

// Create 创建订阅
func (s *SubscriptionsAPI) Create(ctx context.Context, tenant, namespace, topic, subscription string, persistent bool) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscription/%s", schema, tenant, namespace, topic, subscription)
	resp, err := s.client.doRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Delete 删除订阅
func (s *SubscriptionsAPI) Delete(ctx context.Context, tenant, namespace, topic, subscription string, persistent bool) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscription/%s", schema, tenant, namespace, topic, subscription)
	resp, err := s.client.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetStats 获取订阅统计信息
func (s *SubscriptionsAPI) GetStats(ctx context.Context, tenant, namespace, topic, subscription string, persistent bool) (*SubscriptionStats, error) {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscription/%s/stats", schema, tenant, namespace, topic, subscription)
	resp, err := s.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var stats SubscriptionStats
	if err := s.client.parseResponse(resp, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// Reset 重置订阅位置
func (s *SubscriptionsAPI) Reset(ctx context.Context, tenant, namespace, topic, subscription string, persistent bool, position *SubscriptionPosition) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscription/%s/resetcursor", schema, tenant, namespace, topic, subscription)
	resp, err := s.client.doRequest(ctx, http.MethodPost, path, position)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ResetToTimestamp 重置订阅到指定时间戳
func (s *SubscriptionsAPI) ResetToTimestamp(ctx context.Context, tenant, namespace, topic, subscription string, persistent bool, timestamp int64) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscription/%s/resetcursor/%d", schema, tenant, namespace, topic, subscription, timestamp)
	resp, err := s.client.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Skip 跳过消息
func (s *SubscriptionsAPI) Skip(ctx context.Context, tenant, namespace, topic, subscription string, persistent bool, numMessages int64) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscription/%s/skip/%d", schema, tenant, namespace, topic, subscription, numMessages)
	resp, err := s.client.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// SkipAll 跳过所有消息
func (s *SubscriptionsAPI) SkipAll(ctx context.Context, tenant, namespace, topic, subscription string, persistent bool) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscription/%s/skip_all", schema, tenant, namespace, topic, subscription)
	resp, err := s.client.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ExpireMessages 使消息过期
func (s *SubscriptionsAPI) ExpireMessages(ctx context.Context, tenant, namespace, topic, subscription string, persistent bool, expireTimeInSeconds int) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscription/%s/expireMessages/%d", schema, tenant, namespace, topic, subscription, expireTimeInSeconds)
	resp, err := s.client.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ExpireAllMessages 使所有消息过期
func (s *SubscriptionsAPI) ExpireAllMessages(ctx context.Context, tenant, namespace, topic, subscription string, persistent bool) error {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscription/%s/expireMessages", schema, tenant, namespace, topic, subscription)
	resp, err := s.client.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetPosition 获取订阅位置
func (s *SubscriptionsAPI) GetPosition(ctx context.Context, tenant, namespace, topic, subscription string, persistent bool) (*SubscriptionPosition, error) {
	schema := "persistent"
	if !persistent {
		schema = "non-persistent"
	}
	
	path := fmt.Sprintf("/admin/v2/%s/%s/%s/%s/subscription/%s/position", schema, tenant, namespace, topic, subscription)
	resp, err := s.client.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var position SubscriptionPosition
	if err := s.client.parseResponse(resp, &position); err != nil {
		return nil, err
	}

	return &position, nil
}