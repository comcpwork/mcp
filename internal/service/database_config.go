package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// DatabaseConfigService 数据库配置服务
type DatabaseConfigService struct {
	serverName string
	configPath string
}

// NewDatabaseConfigService 创建数据库配置服务
func NewDatabaseConfigService(serverName string) *DatabaseConfigService {
	configPath := filepath.Join(os.Getenv("HOME"), ".co-mcp", serverName+".yaml")
	return &DatabaseConfigService{
		serverName: serverName,
		configPath: configPath,
	}
}

// DatabaseInfo 数据库配置信息
type DatabaseInfo struct {
	Name               string `json:"name"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	User               string `json:"user"`
	Database           string `json:"database"`
	Charset            string `json:"charset"`
	MaxConnections     int    `json:"max_connections"`
	MaxIdleConnections int    `json:"max_idle_connections"`
	ConnectionTimeout  string `json:"connection_timeout"`
	IsActive           bool   `json:"is_active"`
}

// ListDatabases 列出所有数据库配置
func (s *DatabaseConfigService) ListDatabases() ([]DatabaseInfo, error) {
	// 加载配置
	if err := s.loadConfig(); err != nil {
		if os.IsNotExist(err) {
			return []DatabaseInfo{}, nil
		}
		return nil, errors.Wrap(err, "加载配置失败")
	}

	// 获取所有数据库配置
	databases := viper.GetStringMap("databases")
	activeDB := viper.GetString("active_database")

	var result []DatabaseInfo
	for name := range databases {
		dbKey := fmt.Sprintf("databases.%s", name)
		info := DatabaseInfo{
			Name:               name,
			Host:               viper.GetString(dbKey + ".host"),
			Port:               viper.GetInt(dbKey + ".port"),
			User:               viper.GetString(dbKey + ".user"),
			Database:           viper.GetString(dbKey + ".database"),
			Charset:            viper.GetString(dbKey + ".charset"),
			MaxConnections:     viper.GetInt(dbKey + ".max_connections"),
			MaxIdleConnections: viper.GetInt(dbKey + ".max_idle_connections"),
			ConnectionTimeout:  viper.GetString(dbKey + ".connection_timeout"),
			IsActive:           name == activeDB,
		}
		result = append(result, info)
	}

	return result, nil
}

// AddDatabase 添加数据库配置
func (s *DatabaseConfigService) AddDatabase(name string, config map[string]interface{}, setActive bool) error {
	// 如果没有指定名称，使用 default
	if name == "" {
		name = "default"
	}

	// 加载配置，如果不存在则先确保基础配置存在
	if err := s.loadConfig(); err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，创建基础配置
			if err := s.ensureBaseConfig(); err != nil {
				return errors.Wrap(err, "创建基础配置失败")
			}
		} else {
			return errors.Wrap(err, "加载配置失败")
		}
	}
	
	// 确保工具配置存在
	s.ensureToolsConfig()

	// 检查是否已存在
	dbKey := fmt.Sprintf("databases.%s", name)
	if viper.IsSet(dbKey) {
		return errors.Newf("数据库配置 '%s' 已存在", name)
	}

	// 设置默认值
	if config["charset"] == nil {
		config["charset"] = "utf8mb4"
	}
	if config["max_connections"] == nil {
		config["max_connections"] = 10
	}
	if config["max_idle_connections"] == nil {
		config["max_idle_connections"] = 5
	}
	if config["connection_timeout"] == nil {
		config["connection_timeout"] = "30s"
	}

	// 设置新的数据库配置
	for key, value := range config {
		viper.Set(dbKey+"."+key, value)
	}

	// 如果设置为激活或当前没有激活的数据库
	if setActive || viper.GetString("active_database") == "" {
		viper.Set("active_database", name)
	}

	// 保存配置
	return s.saveConfig()
}

// SetActiveDatabase 设置激活的数据库
func (s *DatabaseConfigService) SetActiveDatabase(name string) error {
	// 加载配置
	if err := s.loadConfig(); err != nil {
		return errors.Wrap(err, "加载配置失败")
	}

	// 检查是否存在
	dbKey := fmt.Sprintf("databases.%s", name)
	if !viper.IsSet(dbKey) {
		return errors.Newf("数据库配置 '%s' 不存在", name)
	}

	// 设置激活数据库
	viper.Set("active_database", name)

	// 保存配置
	return s.saveConfig()
}

// RemoveDatabase 删除数据库配置
func (s *DatabaseConfigService) RemoveDatabase(name string) error {
	// 加载配置
	if err := s.loadConfig(); err != nil {
		return errors.Wrap(err, "加载配置失败")
	}

	// 检查是否存在
	dbKey := fmt.Sprintf("databases.%s", name)
	if !viper.IsSet(dbKey) {
		return errors.Newf("数据库配置 '%s' 不存在", name)
	}

	// 不允许删除激活的数据库
	if viper.GetString("active_database") == name {
		return errors.New("不能删除激活的数据库配置，请先切换到其他数据库")
	}

	// 删除配置
	databases := viper.GetStringMap("databases")
	delete(databases, name)
	viper.Set("databases", databases)

	// 保存配置
	return s.saveConfig()
}

// GetDatabase 获取单个数据库配置
func (s *DatabaseConfigService) GetDatabase(name string) (*DatabaseInfo, error) {
	// 加载配置
	if err := s.loadConfig(); err != nil {
		return nil, errors.Wrap(err, "加载配置失败")
	}

	// 检查是否存在
	dbKey := fmt.Sprintf("databases.%s", name)
	if !viper.IsSet(dbKey) {
		return nil, errors.Newf("数据库配置 '%s' 不存在", name)
	}

	activeDB := viper.GetString("active_database")
	info := &DatabaseInfo{
		Name:               name,
		Host:               viper.GetString(dbKey + ".host"),
		Port:               viper.GetInt(dbKey + ".port"),
		User:               viper.GetString(dbKey + ".user"),
		Database:           viper.GetString(dbKey + ".database"),
		Charset:            viper.GetString(dbKey + ".charset"),
		MaxConnections:     viper.GetInt(dbKey + ".max_connections"),
		MaxIdleConnections: viper.GetInt(dbKey + ".max_idle_connections"),
		ConnectionTimeout:  viper.GetString(dbKey + ".connection_timeout"),
		IsActive:           name == activeDB,
	}

	return info, nil
}

// GetActiveDatabase 获取当前激活的数据库配置
func (s *DatabaseConfigService) GetActiveDatabase() (*DatabaseInfo, error) {
	// 加载配置
	if err := s.loadConfig(); err != nil {
		return nil, errors.Wrap(err, "加载配置失败")
	}

	activeDB := viper.GetString("active_database")
	if activeDB == "" {
		activeDB = "default"
	}

	return s.GetDatabase(activeDB)
}

// loadConfig 加载配置文件
func (s *DatabaseConfigService) loadConfig() error {
	viper.SetConfigFile(s.configPath)
	viper.SetConfigType("yaml")
	return viper.ReadInConfig()
}

// saveConfig 保存配置文件
func (s *DatabaseConfigService) saveConfig() error {
	// 确保目录存在
	dir := filepath.Dir(s.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "创建配置目录失败")
	}

	// 保存配置
	return viper.WriteConfig()
}

// ExportConfig 导出配置（用于显示）
func (s *DatabaseConfigService) ExportConfig(name string) (string, error) {
	// 读取配置文件内容
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", errors.Wrap(err, "读取配置文件失败")
	}

	// 如果指定了名称，只返回该数据库配置
	if name != "" {
		// 解析YAML
		var config map[string]interface{}
		if err := yaml.Unmarshal(data, &config); err != nil {
			return "", errors.Wrap(err, "解析配置文件失败")
		}

		databases, ok := config["databases"].(map[string]interface{})
		if !ok {
			return "", nil
		}

		dbConfig, exists := databases[name]
		if !exists {
			return "", errors.Newf("数据库配置 '%s' 不存在", name)
		}

		// 构建只包含指定数据库的配置
		filteredConfig := map[string]interface{}{
			"databases": map[string]interface{}{
				name: dbConfig,
			},
		}

		if config["active_database"] == name {
			filteredConfig["active_database"] = name
		}

		// 输出YAML
		output, err := yaml.Marshal(filteredConfig)
		if err != nil {
			return "", errors.Wrap(err, "格式化配置失败")
		}
		return string(output), nil
	}

	// 返回完整配置
	return string(data), nil
}

// ensureBaseConfig 确保基础配置存在
func (s *DatabaseConfigService) ensureBaseConfig() error {
	// 设置基础配置
	viper.Set("server.name", "MySQL MCP Server")
	viper.Set("server.version", "1.0.0")
	viper.Set("logging.level", "info")
	viper.Set("resources.enabled", true)
	viper.Set("tools.enabled", true)
	
	// 确保工具配置
	s.ensureToolsConfig()
	
	// 保存配置
	return s.saveConfig()
}

// ensureToolsConfig 确保工具配置存在
func (s *DatabaseConfigService) ensureToolsConfig() {
	// 检查 tools.query.max_rows 是否存在，如果不存在则设置默认值
	if !viper.IsSet("tools.query.max_rows") {
		viper.Set("tools.query.max_rows", 1000)
	}
	
	// 检查 tools.query.timeout 是否存在，如果不存在则设置默认值
	if !viper.IsSet("tools.query.timeout") {
		viper.Set("tools.query.timeout", "30s")
	}
}