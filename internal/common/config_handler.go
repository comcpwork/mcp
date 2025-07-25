package common

import (
	"context"
	"fmt"
	"mcp/pkg/log"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/viper"
)

// ConfigManager 配置管理接口
type ConfigManager interface {
	// GetActiveInstance 获取激活的实例名称
	GetActiveInstance() string
	
	// GetInstanceConfig 获取实例配置
	GetInstanceConfig(name string) (map[string]interface{}, bool)
	
	// GetAllInstances 获取所有实例配置
	GetAllInstances() map[string]interface{}
	
	// UpdateInstanceConfig 更新实例配置
	UpdateInstanceConfig(name string, updates map[string]interface{}) error
	
	// SaveConfig 保存配置到文件
	SaveConfig() error
	
	// GetConfigKey 获取配置键前缀
	GetConfigKey(name string) string
}

// BaseConfigHandler 通用配置处理器
type BaseConfigHandler struct {
	ProviderName string       // 提供者名称 (mysql, redis, pulsar)
	ConfigKey    string       // 配置键前缀 (databases)
	FieldMappings map[string]string // 字段映射
}

// HandleUpdateConfig 通用的配置更新处理
func (h *BaseConfigHandler) HandleUpdateConfig(ctx context.Context, request mcp.CallToolRequest, clearCache func(string)) (*mcp.CallToolResult, error) {
	// 记录操作开始
	ctx = log.WithFields(ctx,
		log.String(FieldProvider, h.ProviderName),
		log.String(FieldTool, "update_config"))
	log.Info(ctx, "处理配置更新请求")
	
	name := request.GetString("name", "")
	if name == "" {
		// 使用当前激活的实例
		name = viper.GetString("active_database")
		if name == "" {
			name = DefaultInstanceName
		}
	}

	// 检查实例是否存在
	configKey := fmt.Sprintf("%s.%s", h.ConfigKey, name)
	if !viper.IsSet(configKey) {
		errMsg := fmt.Sprintf("Error: %s instance '%s' does not exist", h.ProviderName, name)
		log.Error(ctx, "实例不存在",
			log.String(FieldInstance, name),
			log.String(FieldError, errMsg))
		return mcp.NewToolResultError(errMsg), nil
	}

	// 收集需要更新的配置
	updates := make(map[string]interface{})
	var updatedFields []string

	// 处理通用字段
	for field, viperKey := range h.FieldMappings {
		if value := request.GetString(field, ""); value != "" {
			updates[configKey+"."+viperKey] = value
			updatedFields = append(updatedFields, field)
		}
	}

	// 处理端口（特殊处理数字类型）
	if port := request.GetInt("port", 0); port > 0 {
		updates[configKey+".port"] = port
		updatedFields = append(updatedFields, "port")
	}

	// 处理其他特殊字段（由子类覆盖）
	h.handleSpecialFields(request, configKey, updates, &updatedFields)

	if len(updates) == 0 {
		return mcp.NewToolResultError("Error: No configuration properties to update"), nil
	}

	// 应用更新
	for key, value := range updates {
		viper.Set(key, value)
	}

	// 保存配置到文件
	if err := viper.WriteConfig(); err != nil {
		log.Error(ctx, "保存配置失败",
			log.String(FieldError, err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("Error saving configuration: %s", err.Error())), nil
	}

	// 清除缓存
	if clearCache != nil {
		clearCache(name)
	}

	// 格式化输出
	output := h.formatUpdateOutput(name, updatedFields, configKey)
	
	// 记录成功
	log.Info(ctx, "配置更新成功",
		log.String(FieldInstance, name),
		log.String(FieldOperation, "update_config"),
		log.String(FieldFields, strings.Join(updatedFields, ",")),
		log.String(FieldStatus, "success"))
	
	return mcp.NewToolResultText(output), nil
}

// HandleGetConfigDetails 通用的配置详情获取处理
func (h *BaseConfigHandler) HandleGetConfigDetails(ctx context.Context, request mcp.CallToolRequest, formatFunc func(string, bool, bool) string) (*mcp.CallToolResult, error) {
	// 记录操作开始
	ctx = log.WithFields(ctx,
		log.String(FieldProvider, h.ProviderName),
		log.String(FieldTool, "get_config_details"))
	log.Info(ctx, "处理获取配置详情请求")
	
	name := request.GetString("name", "")
	includeSensitive := request.GetBool("include_sensitive", false)

	instances := viper.GetStringMap(h.ConfigKey)
	if len(instances) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No %s instances configured", h.ProviderName)), nil
	}

	activeInstance := viper.GetString("active_database")
	if activeInstance == "" {
		activeInstance = DefaultInstanceName
	}

	var output strings.Builder

	if name == "" {
		name = activeInstance
	}

	if name == "all" {
		// 显示所有实例的配置
		output.WriteString(fmt.Sprintf("📋 All %s Instance Configurations\n", strings.Title(h.ProviderName)))
		output.WriteString(strings.Repeat("=", 35) + "\n\n")

		for instanceName := range instances {
			output.WriteString(formatFunc(instanceName, instanceName == activeInstance, includeSensitive))
			output.WriteString("\n")
		}
	} else {
		// 显示指定实例的配置
		if _, exists := instances[name]; !exists {
			errMsg := fmt.Sprintf("Error: %s instance '%s' does not exist", h.ProviderName, name)
			log.Error(ctx, "实例不存在",
				log.String(FieldInstance, name),
				log.String(FieldError, errMsg))
			return mcp.NewToolResultError(errMsg), nil
		}
		
		output.WriteString(fmt.Sprintf("📋 %s Instance Configuration: %s\n", strings.Title(h.ProviderName), name))
		output.WriteString(strings.Repeat("=", 37) + "\n\n")
		output.WriteString(formatFunc(name, name == activeInstance, includeSensitive))
	}

	log.Info(ctx, "获取配置详情成功", 
		log.String(FieldInstance, name),
		log.String("include_sensitive", fmt.Sprintf("%v", includeSensitive)),
		log.String(FieldStatus, "success"))
		
	return mcp.NewToolResultText(output.String()), nil
}

// handleSpecialFields 处理特殊字段（由子类覆盖）
func (h *BaseConfigHandler) handleSpecialFields(request mcp.CallToolRequest, configKey string, updates map[string]interface{}, updatedFields *[]string) {
	// 基类不处理特殊字段
}

// formatUpdateOutput 格式化更新输出
func (h *BaseConfigHandler) formatUpdateOutput(name string, updatedFields []string, configKey string) string {
	output := fmt.Sprintf("✅ Configuration updated for %s instance '%s'\n", h.ProviderName, name)
	output += fmt.Sprintf("Updated fields (%d): %s\n", len(updatedFields), strings.Join(updatedFields, ", "))
	
	// 显示更新后的关键配置
	output += "\nCurrent configuration:\n"
	
	// 通用字段
	if host := viper.GetString(configKey + ".host"); host != "" {
		output += fmt.Sprintf("  Host: %s\n", host)
	}
	if port := viper.GetInt(configKey + ".port"); port > 0 {
		output += fmt.Sprintf("  Port: %d\n", port)
	}
	
	// 移除最后的换行符
	return strings.TrimSuffix(output, "\n")
}

// FormatInstanceConfig 通用的实例配置格式化
func FormatInstanceConfig(providerName, instanceName, configKey string, isActive, includeSensitive bool) string {
	var output strings.Builder
	
	// 实例名称和状态
	status := "⚪ inactive"
	if isActive {
		status = "🟢 ACTIVE"
	}
	output.WriteString(fmt.Sprintf("Instance: %s (%s)\n", instanceName, status))
	
	// 基本配置
	if host := viper.GetString(configKey + ".host"); host != "" {
		output.WriteString(fmt.Sprintf("  Host: %s\n", host))
	}
	
	if port := viper.GetInt(configKey + ".port"); port > 0 {
		output.WriteString(fmt.Sprintf("  Port: %d\n", port))
	}
	
	// 认证信息
	if user := viper.GetString(configKey + ".user"); user != "" {
		output.WriteString(fmt.Sprintf("  User: %s\n", user))
	}
	
	if password := viper.GetString(configKey + ".password"); password != "" {
		if includeSensitive {
			output.WriteString(fmt.Sprintf("  Password: %s\n", password))
		} else {
			output.WriteString("  Password: *** (hidden)\n")
		}
	}
	
	return output.String()
}