package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// RedisPool Redis连接池管理器
type RedisPool struct {
	mu          sync.RWMutex
	connections map[string]*redis.Client // 按Redis实例名称存储的连接池
	serverName  string
}

// NewRedisPool 创建Redis连接池管理器
func NewRedisPool(serverName string) *RedisPool {
	return &RedisPool{
		connections: make(map[string]*redis.Client),
		serverName:  serverName,
	}
}

// GetConnection 获取Redis连接（懒加载）
func (p *RedisPool) GetConnection(ctx context.Context, name string) (*redis.Client, error) {
	// 先尝试读锁获取已存在的连接
	p.mu.RLock()
	if client, exists := p.connections[name]; exists {
		p.mu.RUnlock()
		// 验证连接是否有效
		if err := client.Ping(ctx).Err(); err == nil {
			return client, nil
		}
		// 连接无效，需要重新建立
	} else {
		p.mu.RUnlock()
	}

	// 使用写锁创建新连接
	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查，避免并发创建
	if client, exists := p.connections[name]; exists {
		if err := client.Ping(ctx).Err(); err == nil {
			return client, nil
		}
		// 关闭旧连接
		client.Close()
		delete(p.connections, name)
	}

	// 创建新连接
	client, err := p.createConnection(ctx, name)
	if err != nil {
		return nil, err
	}

	p.connections[name] = client
	return client, nil
}

// GetActiveConnection 获取当前激活的Redis连接
func (p *RedisPool) GetActiveConnection(ctx context.Context) (*redis.Client, error) {
	activeRedis := viper.GetString("active_database")
	if activeRedis == "" {
		activeRedis = "default"
	}
	return p.GetConnection(ctx, activeRedis)
}

// createConnection 创建Redis连接
func (p *RedisPool) createConnection(ctx context.Context, name string) (*redis.Client, error) {
	// 获取Redis配置，适配现有的配置格式
	redisKey := fmt.Sprintf("databases.%s", name)
	if !viper.IsSet(redisKey) {
		return nil, errors.Newf("Redis配置 '%s' 不存在", name)
	}

	// 获取配置参数
	host := viper.GetString(redisKey + ".host")
	if host == "" {
		host = "localhost"
	}
	
	port := viper.GetInt(redisKey + ".port")
	if port == 0 {
		port = 6379
	}
	
	password := viper.GetString(redisKey + ".password")
	database := viper.GetInt(redisKey + ".database")
	
	// 超时配置
	connTimeout := viper.GetDuration(redisKey + ".connection_timeout")
	if connTimeout == 0 {
		connTimeout = 5 * time.Second
	}
	
	readTimeout := viper.GetDuration(redisKey + ".read_timeout")
	if readTimeout == 0 {
		readTimeout = 3 * time.Second
	}
	
	writeTimeout := viper.GetDuration(redisKey + ".write_timeout")
	if writeTimeout == 0 {
		writeTimeout = 3 * time.Second
	}

	// 连接池配置
	maxConns := viper.GetInt(redisKey + ".max_connections")
	if maxConns == 0 {
		maxConns = 10
	}
	
	maxIdleConns := viper.GetInt(redisKey + ".max_idle_connections")
	if maxIdleConns == 0 {
		maxIdleConns = 5
	}
	
	maxIdleTime := viper.GetDuration(redisKey + ".max_idle_time")
	if maxIdleTime == 0 {
		maxIdleTime = 300 * time.Second
	}

	// 创建Redis客户端配置
	options := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           database,
		DialTimeout:  connTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		PoolSize:     maxConns,
		MinIdleConns: maxIdleConns,
		ConnMaxIdleTime: maxIdleTime,
	}

	// 创建Redis客户端
	client := redis.NewClient(options)

	// 测试连接
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	if err := client.Ping(pingCtx).Err(); err != nil {
		client.Close()
		return nil, errors.Wrap(err, "连接Redis失败")
	}

	return client, nil
}

// CloseConnection 关闭指定的Redis连接
func (p *RedisPool) CloseConnection(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, exists := p.connections[name]; exists {
		err := client.Close()
		delete(p.connections, name)
		return err
	}

	return nil
}

// CloseAll 关闭所有Redis连接
func (p *RedisPool) CloseAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for name, client := range p.connections {
		if err := client.Close(); err != nil {
			errs = append(errs, errors.Wrapf(err, "关闭Redis '%s' 失败", name))
		}
	}

	// 清空连接池
	p.connections = make(map[string]*redis.Client)

	if len(errs) > 0 {
		return errors.Newf("关闭Redis连接时发生错误: %v", errs)
	}

	return nil
}

// RefreshConnection 刷新Redis连接
func (p *RedisPool) RefreshConnection(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 关闭旧连接
	if client, exists := p.connections[name]; exists {
		client.Close()
		delete(p.connections, name)
	}

	// 创建新连接
	client, err := p.createConnection(ctx, name)
	if err != nil {
		return err
	}

	p.connections[name] = client
	return nil
}

// HasConnection 检查是否有指定的连接
func (p *RedisPool) HasConnection(name string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	_, exists := p.connections[name]
	return exists
}

// ListConnections 列出所有活动的连接
func (p *RedisPool) ListConnections() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	names := make([]string, 0, len(p.connections))
	for name := range p.connections {
		names = append(names, name)
	}
	return names
}