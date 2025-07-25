package pulsaradmin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client 是 Pulsar Admin API 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
}

// NewClient 创建新的 Pulsar Admin 客户端
func NewClient(baseURL string, options ...ClientOption) *Client {
	client := &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range options {
		opt(client)
	}

	return client
}

// ClientOption 客户端配置选项
type ClientOption func(*Client)

// WithAuth 设置基础认证
func WithAuth(username, password string) ClientOption {
	return func(c *Client) {
		c.username = username
		c.password = password
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 设置基础认证
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	// 检查响应状态码
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求失败: %d %s, 响应: %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	return resp, nil
}

// parseResponse 解析响应
func (c *Client) parseResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()
	
	if result == nil {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体失败: %w", err)
	}

	if len(body) == 0 {
		return nil
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("解析响应体失败: %w, 响应: %s", err, string(body))
	}

	return nil
}

// Tenants 返回租户管理接口
func (c *Client) Tenants() *TenantsAPI {
	return &TenantsAPI{client: c}
}

// Namespaces 返回命名空间管理接口
func (c *Client) Namespaces() *NamespacesAPI {
	return &NamespacesAPI{client: c}
}

// Topics 返回主题管理接口
func (c *Client) Topics() *TopicsAPI {
	return &TopicsAPI{client: c}
}

// Subscriptions 返回订阅管理接口
func (c *Client) Subscriptions() *SubscriptionsAPI {
	return &SubscriptionsAPI{client: c}
}

// Brokers 返回 Broker 管理接口
func (c *Client) Brokers() *BrokersAPI {
	return &BrokersAPI{client: c}
}