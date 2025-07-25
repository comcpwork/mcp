package pool

import (
	"context"
	"fmt"
	"mcp/internal/common"
	"mcp/pkg/log"
	pulsaradmin "mcp/pkg/pulsar-admin"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// PulsarPool Pulsar连接池
type PulsarPool struct {
	provider    string
	connections map[string]*pulsaradmin.Client
	mu          sync.RWMutex
}

// NewPulsarPool 创建新的Pulsar连接池
func NewPulsarPool(provider string) *PulsarPool {
	return &PulsarPool{
		provider:    provider,
		connections: make(map[string]*pulsaradmin.Client),
	}
}

// GetConnection 获取Pulsar连接
func (p *PulsarPool) GetConnection(ctx context.Context, name string) (*pulsaradmin.Client, error) {
	p.mu.RLock()
	if client, exists := p.connections[name]; exists {
		p.mu.RUnlock()
		return client, nil
	}
	p.mu.RUnlock()

	// 需要创建新连接
	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查
	if client, exists := p.connections[name]; exists {
		return client, nil
	}

	// 获取配置
	configKey := fmt.Sprintf("databases.%s", name)
	if !viper.IsSet(configKey) {
		return nil, common.NewInvalidConfigError(p.provider, fmt.Sprintf("configuration '%s' not found", name))
	}

	// 创建Pulsar Admin客户端
	adminURL := viper.GetString(configKey + ".admin_url")
	if adminURL == "" {
		return nil, common.NewInvalidConfigError(p.provider, "Pulsar Admin URL cannot be empty")
	}

	var options []pulsaradmin.ClientOption

	// 设置认证
	username := viper.GetString(configKey + ".username")
	password := viper.GetString(configKey + ".password")
	if username != "" && password != "" {
		options = append(options, pulsaradmin.WithAuth(username, password))
	}

	// 设置超时
	if timeout := viper.GetDuration(configKey + ".timeout"); timeout > 0 {
		options = append(options, pulsaradmin.WithTimeout(timeout))
	} else {
		// 使用默认超时
		defaultTimeout, _ := time.ParseDuration("30s")
		options = append(options, pulsaradmin.WithTimeout(defaultTimeout))
	}

	client := pulsaradmin.NewClient(adminURL, options...)

	// 缓存连接
	p.connections[name] = client

	log.Info(ctx, "Pulsar连接创建成功",
		log.String(common.FieldProvider, p.provider),
		log.String(common.FieldInstance, name),
		log.String("admin_url", adminURL),
		log.String(common.FieldOperation, "connect"),
		log.String(common.FieldStatus, "success"))

	return client, nil
}

// CloseConnection 关闭指定连接
func (p *PulsarPool) CloseConnection(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, exists := p.connections[name]; exists {
		// Pulsar Admin客户端没有Close方法，直接删除引用
		delete(p.connections, name)
		_ = client // 避免未使用警告
	}

	return nil
}

// CloseAllConnections 关闭所有连接
func (p *PulsarPool) CloseAllConnections() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 清空连接映射
	p.connections = make(map[string]*pulsaradmin.Client)

	return nil
}