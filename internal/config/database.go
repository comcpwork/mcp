package config

import (
	"time"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host               string        `mapstructure:"host"`
	Port               int           `mapstructure:"port"`
	User               string        `mapstructure:"user"`
	Password           string        `mapstructure:"password"`
	Database           string        `mapstructure:"database"`
	Charset            string        `mapstructure:"charset"`
	MaxConnections     int           `mapstructure:"max_connections"`
	MaxIdleConnections int           `mapstructure:"max_idle_connections"`
	ConnectionTimeout  time.Duration `mapstructure:"connection_timeout"`
}

// MySQLConfig MySQL配置
type MySQLConfig struct {
	Server struct {
		Name    string `mapstructure:"name"`
		Version string `mapstructure:"version"`
	} `mapstructure:"server"`

	// 多数据库配置
	Databases map[string]DatabaseConfig `mapstructure:"databases"`

	// 当前激活的数据库
	ActiveDatabase string `mapstructure:"active_database"`

	// 工具配置
	Tools struct {
		Query struct {
			Timeout time.Duration `mapstructure:"timeout"`
			MaxRows int           `mapstructure:"max_rows"`
		} `mapstructure:"query"`
	} `mapstructure:"tools"`
}
