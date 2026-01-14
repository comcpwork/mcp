package database

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
)

// ParseSSHURI 解析SSH URI字符串为SSHConfig
// 支持两种格式:
//  1. ssh://config-name - 从~/.ssh/config读取
//  2. ssh://[user[:password]@]host[:port][?key=path&passphrase=xxx] - 完整URI
//
// 如果uri为空，返回nil表示不使用SSH
func ParseSSHURI(uri string) (*SSHConfig, error) {
	if uri == "" {
		return nil, nil
	}

	// 检查是否是SSH配置引用（不含@符号且不含?参数）
	if isSSHConfigReference(uri) {
		return ParseSSHConfigByName(uri)
	}

	return parseFullSSHURI(uri)
}

// isSSHConfigReference 判断URI是否是SSH配置引用
// 格式: ssh://config-name (不含@符号)
func isSSHConfigReference(uri string) bool {
	// 移除 ssh:// 前缀
	if !strings.HasPrefix(uri, "ssh://") {
		return false
	}
	rest := strings.TrimPrefix(uri, "ssh://")

	// 如果不含@和?，则认为是配置引用
	return !strings.Contains(rest, "@") && !strings.Contains(rest, "?")
}

// parseFullSSHURI 解析完整的SSH URI
func parseFullSSHURI(uri string) (*SSHConfig, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH URI: %w", err)
	}

	if u.Scheme != "ssh" {
		return nil, fmt.Errorf("invalid scheme: expected 'ssh', got '%s'", u.Scheme)
	}

	config := &SSHConfig{
		Host: u.Hostname(),
		Port: 22, // 默认端口
	}

	// 解析端口
	if portStr := u.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SSH port: %s", portStr)
		}
		config.Port = port
	}

	// 解析用户名和密码
	if u.User != nil {
		config.User = u.User.Username()
		if password, ok := u.User.Password(); ok {
			config.Password = password
		}
	}

	// 解析query参数 (key, passphrase)
	query := u.Query()
	if keyPath := query.Get("key"); keyPath != "" {
		config.KeyPath = keyPath
	}
	if passphrase := query.Get("passphrase"); passphrase != "" {
		config.Passphrase = passphrase
	}

	// 验证: 必须有用户名
	if config.User == "" {
		return nil, fmt.Errorf("SSH user is required")
	}

	// 验证: 必须有密码或私钥
	if config.Password == "" && config.KeyPath == "" {
		return nil, fmt.Errorf("SSH password or key is required")
	}

	return config, nil
}

// ParseSSHConfigByName 从~/.ssh/config解析指定Host的配置
func ParseSSHConfigByName(uri string) (*SSHConfig, error) {
	// 提取配置名称
	hostName := strings.TrimPrefix(uri, "ssh://")
	if hostName == "" {
		return nil, fmt.Errorf("SSH config name is empty")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".ssh", "config")

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("SSH config file not found: %s", configPath)
	}

	// 打开并解析配置文件
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SSH config: %w", err)
	}
	defer f.Close()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH config: %w", err)
	}

	// 获取HostName，如果没有则使用Host本身
	host, err := cfg.Get(hostName, "HostName")
	if err != nil || host == "" {
		host = hostName
	}

	// 检查是否存在该配置
	user, _ := cfg.Get(hostName, "User")
	if user == "" {
		return nil, fmt.Errorf("SSH config host '%s' not found or has no User", hostName)
	}

	config := &SSHConfig{
		Host: host,
		Port: 22,
		User: user,
	}

	// 解析端口
	if portStr, err := cfg.Get(hostName, "Port"); err == nil && portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Port = port
		}
	}

	// 解析私钥路径
	if keyPath, err := cfg.Get(hostName, "IdentityFile"); err == nil && keyPath != "" {
		// 展开 ~ 为home目录
		config.KeyPath = expandPath(keyPath)
	}

	// 如果没有私钥，尝试使用默认私钥
	if config.KeyPath == "" {
		defaultKeys := []string{
			filepath.Join(homeDir, ".ssh", "id_rsa"),
			filepath.Join(homeDir, ".ssh", "id_ed25519"),
			filepath.Join(homeDir, ".ssh", "id_ecdsa"),
		}
		for _, keyPath := range defaultKeys {
			if _, err := os.Stat(keyPath); err == nil {
				config.KeyPath = keyPath
				break
			}
		}
	}

	// 验证: 必须有认证方式
	if config.KeyPath == "" && config.Password == "" {
		return nil, fmt.Errorf("SSH config '%s' has no IdentityFile and no default key found", hostName)
	}

	return config, nil
}

// expandPath 展开路径中的~为home目录
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[1:])
	}
	return path
}

// ExtractMySQLHostPort 从MySQL DSN中提取host:port
// DSN格式: user:pass@tcp(host:port)/dbname
func ExtractMySQLHostPort(dsn string) (host string, port int, err error) {
	// 匹配 tcp(host:port) 模式
	re := regexp.MustCompile(`tcp\(([^:]+):(\d+)\)`)
	matches := re.FindStringSubmatch(dsn)
	if len(matches) != 3 {
		return "", 0, fmt.Errorf("cannot extract host:port from MySQL DSN")
	}

	host = matches[1]
	port, err = strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in MySQL DSN: %s", matches[2])
	}

	return host, port, nil
}

// ReplaceMySQLDSNHostPort 替换MySQL DSN中的host:port为新地址
func ReplaceMySQLDSNHostPort(dsn string, newAddr string) string {
	re := regexp.MustCompile(`tcp\([^)]+\)`)
	return re.ReplaceAllString(dsn, fmt.Sprintf("tcp(%s)", newAddr))
}

// ExtractRedisHostPort 从Redis DSN中提取host:port
// DSN格式: redis://[:password@]host:port/db
func ExtractRedisHostPort(dsn string) (host string, port int, err error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", 0, fmt.Errorf("invalid Redis DSN: %w", err)
	}

	host = u.Hostname()
	if host == "" {
		return "", 0, fmt.Errorf("cannot extract host from Redis DSN")
	}

	portStr := u.Port()
	if portStr == "" {
		port = 6379 // Redis默认端口
	} else {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return "", 0, fmt.Errorf("invalid port in Redis DSN: %s", portStr)
		}
	}

	return host, port, nil
}

// ReplaceRedisDSNHostPort 替换Redis DSN中的host:port为新地址
func ReplaceRedisDSNHostPort(dsn string, newAddr string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}

	u.Host = newAddr
	return u.String()
}

// ExtractClickHouseHostPort 从ClickHouse DSN中提取host:port
// DSN格式: clickhouse://user:pass@host:port/dbname
func ExtractClickHouseHostPort(dsn string) (host string, port int, err error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", 0, fmt.Errorf("invalid ClickHouse DSN: %w", err)
	}

	host = u.Hostname()
	if host == "" {
		return "", 0, fmt.Errorf("cannot extract host from ClickHouse DSN")
	}

	portStr := u.Port()
	if portStr == "" {
		port = 9000 // ClickHouse默认端口
	} else {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return "", 0, fmt.Errorf("invalid port in ClickHouse DSN: %s", portStr)
		}
	}

	return host, port, nil
}

// ReplaceClickHouseDSNHostPort 替换ClickHouse DSN中的host:port为新地址
func ReplaceClickHouseDSNHostPort(dsn string, newAddr string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}

	u.Host = newAddr
	return u.String()
}
