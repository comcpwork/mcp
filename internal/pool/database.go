package pool

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
)

// DatabasePool 数据库连接池管理器
type DatabasePool struct {
	mu          sync.RWMutex
	connections map[string]*sql.DB // 按数据库名称存储的连接池
	serverName  string
}

// NewDatabasePool 创建数据库连接池管理器
func NewDatabasePool(serverName string) *DatabasePool {
	return &DatabasePool{
		connections: make(map[string]*sql.DB),
		serverName:  serverName,
	}
}

// GetConnection 获取数据库连接（懒加载）
func (p *DatabasePool) GetConnection(name string) (*sql.DB, error) {
	// 先尝试读锁获取已存在的连接
	p.mu.RLock()
	if db, exists := p.connections[name]; exists {
		p.mu.RUnlock()
		// 验证连接是否有效
		if err := db.Ping(); err == nil {
			return db, nil
		}
		// 连接无效，需要重新建立
	} else {
		p.mu.RUnlock()
	}

	// 使用写锁创建新连接
	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查，避免并发创建
	if db, exists := p.connections[name]; exists {
		if err := db.Ping(); err == nil {
			return db, nil
		}
		// 关闭旧连接
		db.Close()
		delete(p.connections, name)
	}

	// 创建新连接
	db, err := p.createConnection(name)
	if err != nil {
		return nil, err
	}

	p.connections[name] = db
	return db, nil
}

// GetActiveConnection 获取当前激活的数据库连接
func (p *DatabasePool) GetActiveConnection() (*sql.DB, error) {
	activeDB := viper.GetString("active_database")
	if activeDB == "" {
		activeDB = "default"
	}
	return p.GetConnection(activeDB)
}

// createConnection 创建数据库连接
func (p *DatabasePool) createConnection(name string) (*sql.DB, error) {
	// 获取数据库配置
	dbKey := fmt.Sprintf("databases.%s", name)
	if !viper.IsSet(dbKey) {
		return nil, errors.Newf("数据库配置 '%s' 不存在", name)
	}

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true",
		viper.GetString(dbKey+".user"),
		viper.GetString(dbKey+".password"),
		viper.GetString(dbKey+".host"),
		viper.GetInt(dbKey+".port"),
		viper.GetString(dbKey+".database"),
		viper.GetString(dbKey+".charset"),
	)

	// 打开数据库连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "打开数据库连接失败")
	}

	// 设置连接池参数
	db.SetMaxOpenConns(viper.GetInt(dbKey + ".max_connections"))
	db.SetMaxIdleConns(viper.GetInt(dbKey + ".max_idle_connections"))

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, errors.Wrap(err, "连接数据库失败")
	}

	return db, nil
}

// CloseConnection 关闭指定的数据库连接
func (p *DatabasePool) CloseConnection(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if db, exists := p.connections[name]; exists {
		err := db.Close()
		delete(p.connections, name)
		return err
	}

	return nil
}

// CloseAll 关闭所有数据库连接
func (p *DatabasePool) CloseAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for name, db := range p.connections {
		if err := db.Close(); err != nil {
			errs = append(errs, errors.Wrapf(err, "关闭数据库 '%s' 失败", name))
		}
	}

	// 清空连接池
	p.connections = make(map[string]*sql.DB)

	if len(errs) > 0 {
		return errors.Newf("关闭数据库连接时发生错误: %v", errs)
	}

	return nil
}

// RefreshConnection 刷新数据库连接
func (p *DatabasePool) RefreshConnection(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 关闭旧连接
	if db, exists := p.connections[name]; exists {
		db.Close()
		delete(p.connections, name)
	}

	// 创建新连接
	db, err := p.createConnection(name)
	if err != nil {
		return err
	}

	p.connections[name] = db
	return nil
}

// HasConnection 检查是否有指定的连接
func (p *DatabasePool) HasConnection(name string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	_, exists := p.connections[name]
	return exists
}

// ListConnections 列出所有活动的连接
func (p *DatabasePool) ListConnections() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	names := make([]string, 0, len(p.connections))
	for name := range p.connections {
		names = append(names, name)
	}
	return names
}