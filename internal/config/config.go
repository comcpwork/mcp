package config

import (
	"context"
	"fmt"
	"mcp/pkg/log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const ConfigDir = ".co-mcp"

// Config 配置管理器
type Config struct {
	serverName string
	configPath string
}

// NewConfig 创建配置管理器
func NewConfig(serverName string) *Config {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ConfigDir)

	return &Config{
		serverName: serverName,
		configPath: filepath.Join(configDir, serverName+".yaml"),
	}
}

// Init 初始化配置（如果不存在则创建默认配置）
func (c *Config) Init(ctx context.Context) error {
	// 确保配置目录存在
	configDir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 如果配置文件不存在，写入默认配置
	if _, err := os.Stat(c.configPath); os.IsNotExist(err) {
		// 获取默认配置
		defaultConfig := c.getDefaultConfig()
		if err := os.WriteFile(c.configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("写入默认配置失败: %w", err)
		}
		log.Info(ctx, "创建配置文件", log.String("path", c.configPath))
	}

	// 加载配置
	viper.SetConfigFile(c.configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	log.Info(ctx, "加载配置文件", log.String("path", c.configPath))
	return nil
}

// GetConfigPath 获取配置文件路径
func (c *Config) GetConfigPath() string {
	return c.configPath
}

// getDefaultConfig 获取默认配置
func (c *Config) getDefaultConfig() string {
	// 这里可以根据不同的服务器类型返回不同的默认配置
	// 实际项目中，每个服务器应该提供自己的默认配置
	return `# MCP 服务器配置
server:
  name: "` + c.serverName + ` MCP Server"
  version: "1.0.0"

# 日志配置
logging:
  level: "info"

# 工具配置
tools:
  enabled: true

# 资源配置  
resources:
  enabled: true
`
}
