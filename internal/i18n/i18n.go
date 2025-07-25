package i18n

import (
	_ "embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed messages.yaml
var messagesYAML string

// Messages 存储所有消息
type Messages struct {
	Root   CommandDesc `yaml:"root"`
	Help   CommandDesc `yaml:"help"`
	Server struct {
		MySQL   CommandDesc `yaml:"mysql"`
		Redis   CommandDesc `yaml:"redis"`
		Default CommandDesc `yaml:"default"`
	} `yaml:"server"`
	Flags             map[string]string `yaml:"flags"`
	Messages          map[string]string `yaml:"messages"`
	UsageTemplate     string            `yaml:"usage_template"`
	ServerUsageTemplate string          `yaml:"server_usage_template"`
}

// CommandDesc 命令描述
type CommandDesc struct {
	Use   string `yaml:"use"`
	Short string `yaml:"short"`
	Long  string `yaml:"long"`
}

var messages Messages

func init() {
	if err := yaml.Unmarshal([]byte(messagesYAML), &messages); err != nil {
		panic(fmt.Sprintf("加载消息文件失败: %v", err))
	}
}

// GetRootCommand 获取根命令描述
func GetRootCommand() CommandDesc {
	return messages.Root
}

// GetHelpCommand 获取帮助命令描述
func GetHelpCommand() CommandDesc {
	return messages.Help
}

// GetServerCommand 获取服务器命令描述
func GetServerCommand(serverName string) CommandDesc {
	var desc CommandDesc
	switch serverName {
	case "mysql":
		desc = messages.Server.MySQL
	case "redis":
		desc = messages.Server.Redis
	default:
		// 使用默认模板
		desc = CommandDesc{
			Short: fmt.Sprintf(messages.Server.Default.Short, serverName),
			Long:  fmt.Sprintf(messages.Server.Default.Long, serverName),
		}
	}
	// 确保 Use 字段始终是服务器名称（英文）
	desc.Use = serverName
	return desc
}


// GetFlag 获取标志名称
func GetFlag(key string) string {
	if val, ok := messages.Flags[key]; ok {
		return val
	}
	return key
}

// GetFlagDesc 获取标志描述
func GetFlagDesc(key string) string {
	descKey := key + "_desc"
	if val, ok := messages.Flags[descKey]; ok {
		return val
	}
	return ""
}

// GetMessage 获取消息
func GetMessage(key string, args ...interface{}) string {
	if val, ok := messages.Messages[key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(val, args...)
		}
		return val
	}
	return key
}

// GetUsageTemplate 获取使用模板
func GetUsageTemplate() string {
	return strings.TrimSpace(messages.UsageTemplate)
}

// GetServerUsageTemplate 获取服务器使用模板
func GetServerUsageTemplate() string {
	return strings.TrimSpace(messages.ServerUsageTemplate)
}

// FormatError 格式化错误
func FormatError(err error) string {
	return fmt.Sprintf(GetMessage("error"), err)
}