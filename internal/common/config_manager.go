package common

import (
	"crypto/md5"
	"fmt"
	"time"
)

// ConnectionConfig 连接配置
type ConnectionConfig struct {
	// 基本连接信息
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	User     string `json:"user,omitempty" yaml:"user,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	Database string `json:"database,omitempty" yaml:"database,omitempty"`
	
	// Redis 特有
	DB int `json:"db,omitempty" yaml:"db,omitempty"`
	
	// Pulsar 特有
	AdminURL  string `json:"admin_url,omitempty" yaml:"admin_url,omitempty"`
	Tenant    string `json:"tenant,omitempty" yaml:"tenant,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	
	// 元数据
	LastUsed  time.Time `json:"last_used" yaml:"last_used"`
	UseCount  int       `json:"use_count" yaml:"use_count"`
	Alias     string    `json:"alias,omitempty" yaml:"alias,omitempty"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// GetID 获取配置的唯一标识
func (c *ConnectionConfig) GetID() string {
	// 根据不同类型生成唯一ID
	var key string
	
	// MySQL/Redis 使用 host:port:user:database
	if c.Database != "" || c.DB > 0 {
		key = fmt.Sprintf("%s:%d:%s:%s:%d", c.Host, c.Port, c.User, c.Database, c.DB)
	} else if c.AdminURL != "" {
		// Pulsar 使用 admin_url:tenant:namespace
		key = fmt.Sprintf("%s:%s:%s", c.AdminURL, c.Tenant, c.Namespace)
	} else {
		// 默认使用 host:port:user
		key = fmt.Sprintf("%s:%d:%s", c.Host, c.Port, c.User)
	}
	
	// 生成MD5作为ID，避免特殊字符
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))[:8]
}

// IsSame 判断两个配置是否相同
func (c *ConnectionConfig) IsSame(other *ConnectionConfig) bool {
	if other == nil {
		return false
	}
	
	// 基本连接信息必须相同
	if c.Host != other.Host || c.Port != other.Port {
		return false
	}
	
	// MySQL/通用数据库
	if c.Database != "" || other.Database != "" {
		return c.User == other.User && c.Database == other.Database
	}
	
	// Redis
	if c.DB > 0 || other.DB > 0 {
		return c.User == other.User && c.DB == other.DB
	}
	
	// Pulsar
	if c.AdminURL != "" || other.AdminURL != "" {
		return c.AdminURL == other.AdminURL && 
			c.Tenant == other.Tenant && 
			c.Namespace == other.Namespace
	}
	
	// 默认比较
	return c.User == other.User
}

// Merge 合并配置（更新使用信息）
func (c *ConnectionConfig) Merge(other *ConnectionConfig) {
	if !c.IsSame(other) {
		return
	}
	
	// 更新使用信息
	c.LastUsed = time.Now()
	c.UseCount++
	
	// 如果有新的别名，更新别名
	if other.Alias != "" && c.Alias == "" {
		c.Alias = other.Alias
	}
	
	// 更新密码（如果提供了新密码）
	if other.Password != "" {
		c.Password = other.Password
	}
}

// ConfigStore 配置存储结构
type ConfigStore struct {
	Current map[string]*ConnectionConfig   `yaml:"current"`
	History map[string][]*ConnectionConfig `yaml:"history"`
}

// NewConfigStore 创建新的配置存储
func NewConfigStore() *ConfigStore {
	return &ConfigStore{
		Current: make(map[string]*ConnectionConfig),
		History: make(map[string][]*ConnectionConfig),
	}
}

// SetCurrent 设置当前连接
func (s *ConfigStore) SetCurrent(provider string, config *ConnectionConfig) {
	if config == nil {
		return
	}
	
	// 更新使用信息
	config.LastUsed = time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}
	
	// 设置当前连接
	s.Current[provider] = config
	
	// 添加到历史记录
	s.AddToHistory(provider, config)
}

// GetCurrent 获取当前连接
func (s *ConfigStore) GetCurrent(provider string) *ConnectionConfig {
	return s.Current[provider]
}

// AddToHistory 添加到历史记录（自动去重）
func (s *ConfigStore) AddToHistory(provider string, config *ConnectionConfig) {
	if config == nil {
		return
	}
	
	// 获取该提供者的历史记录
	history := s.History[provider]
	
	// 查找是否已存在相同配置
	for i, h := range history {
		if h.IsSame(config) {
			// 合并配置
			h.Merge(config)
			// 将最近使用的移到前面
			s.History[provider] = append([]*ConnectionConfig{h}, append(history[:i], history[i+1:]...)...)
			return
		}
	}
	
	// 新配置，添加到历史记录前面
	config.UseCount = 1
	s.History[provider] = append([]*ConnectionConfig{config}, history...)
	
	// 限制历史记录数量（最多保留20条）
	if len(s.History[provider]) > 20 {
		s.History[provider] = s.History[provider][:20]
	}
}

// GetHistory 获取历史记录
func (s *ConfigStore) GetHistory(provider string) []*ConnectionConfig {
	return s.History[provider]
}

// FindHistoryByID 根据ID查找历史配置
func (s *ConfigStore) FindHistoryByID(provider, id string) *ConnectionConfig {
	for _, config := range s.History[provider] {
		if config.GetID() == id {
			return config
		}
	}
	return nil
}

// FindHistoryByAlias 根据别名查找历史配置
func (s *ConfigStore) FindHistoryByAlias(provider, alias string) *ConnectionConfig {
	for _, config := range s.History[provider] {
		if config.Alias == alias {
			return config
		}
	}
	return nil
}

// ClearHistory 清理历史记录
func (s *ConfigStore) ClearHistory(provider string) {
	if provider == "" {
		// 清理所有历史
		s.History = make(map[string][]*ConnectionConfig)
	} else {
		// 清理指定提供者的历史
		delete(s.History, provider)
	}
}