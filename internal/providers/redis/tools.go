package redis

import (
	"context"
	"fmt"
	"mcp/internal/common"
	"mcp/pkg/log"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// ensureConnection 获取Redis连接，如果失败返回错误结果
func (s *RedisServer) ensureConnection(ctx context.Context) (*redis.Client, *mcp.CallToolResult) {
	log.Info(ctx, "尝试获取Redis连接",
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "get_connection"))
	client, err := s.getConnection(ctx)
	if err != nil {
		log.Error(ctx, "Redis连接失败",
			log.Err(err),
			log.String(common.FieldProvider, "redis"),
			log.String(common.FieldOperation, "get_connection"),
			log.String(common.FieldStatus, "failed"))
		if errors.Is(err, common.NewNoConfigError("redis")) {
			return nil, mcp.NewToolResultError("没有Redis配置，请先使用 add_redis 工具添加Redis配置")
		}
		return nil, mcp.NewToolResultError(fmt.Sprintf("Redis连接失败: %v", err))
	}
	log.Info(ctx, "Redis连接成功",
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "get_connection"),
		log.String(common.FieldStatus, "success"))
	return client, nil
}

// handleExec 处理Redis命令执行请求，支持使用pipe执行多条命令
func (s *RedisServer) handleExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取参数
	command, err := req.RequireString("command")
	if err != nil {
		return nil, errors.Wrap(err, "缺少command参数")
	}

	log.Info(ctx, "处理 exec 请求",
		log.String(common.FieldTool, "exec"),
		log.String("command", command),
		log.String("active_redis", s.activeRedis),
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "exec"),
	)

	// 获取Redis连接
	client, errResult := s.ensureConnection(ctx)
	if errResult != nil {
		return errResult, nil
	}

	// 检查是否包含多条命令（使用 | 或 ; 分隔）
	commands := s.parseMultipleCommands(command)
	
	if len(commands) == 1 {
		// 单条命令，直接执行
		args, err := s.parseCommand(commands[0])
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("命令解析失败: %v", err)), nil
		}

		if len(args) == 0 {
			return mcp.NewToolResultError("命令不能为空"), nil
		}

		return s.executeRedisCommand(ctx, client, args)
	} else {
		// 多条命令，使用pipeline执行
		return s.executePipelineCommands(ctx, client, commands)
	}
}

// parseMultipleCommands 解析多条命令，支持 | 和 ; 分隔符
func (s *RedisServer) parseMultipleCommands(command string) []string {
	// 先尝试使用 | 分隔
	if strings.Contains(command, "|") {
		parts := strings.Split(command, "|")
		var commands []string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				commands = append(commands, part)
			}
		}
		return commands
	}
	
	// 再尝试使用 ; 分隔
	if strings.Contains(command, ";") {
		parts := strings.Split(command, ";")
		var commands []string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				commands = append(commands, part)
			}
		}
		return commands
	}
	
	// 单条命令
	return []string{strings.TrimSpace(command)}
}

// parseCommand 解析Redis命令字符串
func (s *RedisServer) parseCommand(command string) ([]interface{}, error) {
	// 简单的命令解析，支持引号
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, errors.New("命令为空")
	}

	var args []interface{}
	for _, part := range parts {
		// 去除引号
		if strings.HasPrefix(part, "\"") && strings.HasSuffix(part, "\"") {
			part = strings.Trim(part, "\"")
		} else if strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'") {
			part = strings.Trim(part, "'")
		}
		args = append(args, part)
	}

	return args, nil
}

// executeRedisCommand 执行Redis命令
func (s *RedisServer) executeRedisCommand(ctx context.Context, client *redis.Client, args []interface{}) (*mcp.CallToolResult, error) {
	if len(args) == 0 {
		return mcp.NewToolResultError("命令参数为空"), nil
	}

	cmdName := strings.ToUpper(args[0].(string))
	log.Info(ctx, "执行Redis命令",
		log.String("command", cmdName),
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "execute_command"))

	// 安全检查：基于配置选项验证命令权限
	if err := s.validateRedisCommandSecurity(ctx, cmdName); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 执行命令
	result := client.Do(ctx, args...)
	if result.Err() != nil {
		log.Error(ctx, "Redis命令执行失败",
			log.Err(result.Err()),
			log.String(common.FieldProvider, "redis"),
			log.String(common.FieldOperation, "execute_command"),
			log.String(common.FieldStatus, "failed"))
		return mcp.NewToolResultError(fmt.Sprintf("命令执行失败: %v", result.Err())), nil
	}

	// 格式化结果
	output := s.formatRedisResult(cmdName, result.Val(), strings.Join(convertArgsToStrings(args), " "))
	
	log.Info(ctx, "Redis命令执行成功",
		log.String("command", cmdName),
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "execute_command"),
		log.String(common.FieldStatus, "success"))
	return mcp.NewToolResultText(output), nil
}

// executePipelineCommands 使用pipeline执行多条Redis命令
func (s *RedisServer) executePipelineCommands(ctx context.Context, client *redis.Client, commands []string) (*mcp.CallToolResult, error) {
	log.Info(ctx, "执行Pipeline命令",
		log.Int("command_count", len(commands)),
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "execute_pipeline"))

	// 创建pipeline
	pipe := client.Pipeline()
	
	// 存储命令信息用于后续处果处理
	type CommandInfo struct {
		OriginalCommand string
		ParsedArgs      []interface{}
		CmdName         string
	}
	
	var commandInfos []CommandInfo
	var pipelineResults []redis.Cmder
	
	// 解析并添加命令到pipeline
	for i, command := range commands {
		args, err := s.parseCommand(command)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("命令 %d 解析失败: %v", i+1, err)), nil
		}
		
		if len(args) == 0 {
			return mcp.NewToolResultError(fmt.Sprintf("命令 %d 不能为空", i+1)), nil
		}
		
		cmdName := strings.ToUpper(args[0].(string))
		
		// 安全检查：基于配置选项验证命令权限
		if err := s.validateRedisCommandSecurity(ctx, cmdName); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("命令 %d: %s", i+1, err.Error())), nil
		}
		
		// 添加命令到pipeline
		pipelineCmd := pipe.Do(ctx, args...)
		pipelineResults = append(pipelineResults, pipelineCmd)
		
		commandInfos = append(commandInfos, CommandInfo{
			OriginalCommand: command,
			ParsedArgs:      args,
			CmdName:         cmdName,
		})
	}
	
	// 执行pipeline
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		log.Error(ctx, "Pipeline执行失败",
			log.Err(err),
			log.String(common.FieldProvider, "redis"),
			log.String(common.FieldOperation, "execute_pipeline"),
			log.String(common.FieldStatus, "failed"))
		return mcp.NewToolResultError(fmt.Sprintf("Pipeline执行失败: %v", err)), nil
	}
	
	// 格式化所有结果
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Pipeline Results (%d commands)\n\n", len(commands)))
	
	for i, result := range pipelineResults {
		cmdInfo := commandInfos[i]
		
		output.WriteString(fmt.Sprintf("--- Command %d ---\n", i+1))
		
		// 获取命令结果
		val, cmdErr := result.(*redis.Cmd).Result()
		if cmdErr != nil && cmdErr != redis.Nil {
			output.WriteString(fmt.Sprintf("Command: %s\n", cmdInfo.OriginalCommand))
			output.WriteString(fmt.Sprintf("Error: %v\n\n", cmdErr))
		} else {
			// 格式化成功结果
			formattedResult := s.formatRedisResult(cmdInfo.CmdName, val, cmdInfo.OriginalCommand)
			output.WriteString(formattedResult)
			output.WriteString("\n")
		}
	}
	
	log.Info(ctx, "Pipeline命令执行完成",
		log.Int("command_count", len(commands)),
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "execute_pipeline"),
		log.String(common.FieldStatus, "success"))
	return mcp.NewToolResultText(output.String()), nil
}

// formatRedisResult 格式化Redis命令结果为紧凑格式
func (s *RedisServer) formatRedisResult(command string, result interface{}, fullCommand string) string {
	var output strings.Builder
	
	// 简化的命令显示
	if len(fullCommand) > 60 {
		fullCommand = fullCommand[:57] + "..."
	}
	
	output.WriteString(fmt.Sprintf("Command: %s\n", fullCommand))
	
	// 根据结果类型格式化输出
	switch v := result.(type) {
	case nil:
		output.WriteString("Result: (nil)\n")
		
	case string:
		if v == "" {
			output.WriteString("Result: (empty string)\n")
		} else {
			// 不截断字符串，显示完整内容
			output.WriteString(fmt.Sprintf("Result: %s\n", v))
		}
		
	case int64:
		output.WriteString(fmt.Sprintf("Result: %d\n", v))
		
	case []interface{}:
		output.WriteString(fmt.Sprintf("Result: Array (%d items)\n", len(v)))
		if len(v) == 0 {
			output.WriteString("  (empty)\n")
		} else {
			// 显示前10个元素
			maxShow := 10
			if len(v) < maxShow {
				maxShow = len(v)
			}
			
			for i := 0; i < maxShow; i++ {
				item := fmt.Sprintf("%v", v[i])
				output.WriteString(fmt.Sprintf("  [%d] %s\n", i, item))
			}
			
			if len(v) > maxShow {
				output.WriteString(fmt.Sprintf("  ... and %d more items\n", len(v)-maxShow))
			}
		}
		
	case map[string]string:
		output.WriteString(fmt.Sprintf("Result: Hash (%d fields)\n", len(v)))
		if len(v) == 0 {
			output.WriteString("  (empty)\n")
		} else {
			count := 0
			maxShow := 10
			for key, val := range v {
				if count >= maxShow {
					output.WriteString(fmt.Sprintf("  ... and %d more fields\n", len(v)-maxShow))
					break
				}
				
				// 不截断值，显示完整内容
				output.WriteString(fmt.Sprintf("  %s: %s\n", key, val))
				count++
			}
		}
		
	default:
		// 通用格式化
		resultStr := fmt.Sprintf("%v", v)
		output.WriteString(fmt.Sprintf("Result: %s\n", resultStr))
	}
	
	return output.String()
}

// convertArgsToStrings 将参数转换为字符串数组
func convertArgsToStrings(args []interface{}) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		result[i] = fmt.Sprintf("%v", arg)
	}
	return result
}

// handleListRedis 列出所有Redis配置

// parseRedisInfo 解析Redis INFO命令的输出
func (s *RedisServer) parseRedisInfo(info string) map[string]interface{} {
	result := make(map[string]interface{})
	lines := strings.Split(info, "\n")
	
	var currentSection string
	sectionData := make(map[string]interface{})
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			if strings.HasPrefix(line, "# ") {
				// 保存上一个section
				if currentSection != "" && len(sectionData) > 0 {
					result[currentSection] = sectionData
				}
				// 开始新的section
				currentSection = strings.TrimPrefix(line, "# ")
				sectionData = make(map[string]interface{})
			}
			continue
		}
		
		// 解析键值对
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			// 尝试转换为数字
			if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
				sectionData[key] = intVal
			} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				sectionData[key] = floatVal
			} else {
				sectionData[key] = value
			}
		}
	}
	
	// 保存最后一个section
	if currentSection != "" && len(sectionData) > 0 {
		result[currentSection] = sectionData
	}
	
	return result
}

// handleUpdateConfig 处理更新配置的请求
func (s *RedisServer) handleUpdateConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理update_config请求",
		log.String(common.FieldTool, "update_config"),
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "update_config"))

	name := request.GetString("name", "")
	if name == "" {
		// 使用当前激活的实例
		name = viper.GetString("active_database")
		if name == "" {
			name = "default"
		}
	}

	// 检查实例是否存在
	redisKey := fmt.Sprintf("databases.%s", name)
	if !viper.IsSet(redisKey) {
		return mcp.NewToolResultError(fmt.Sprintf("Error: Redis instance '%s' does not exist", name)), nil
	}

	// 收集需要更新的配置
	updates := make(map[string]interface{})
	var updatedFields []string

	if host := request.GetString("host", ""); host != "" {
		updates[redisKey+".host"] = host
		updatedFields = append(updatedFields, "host")
	}

	if port := request.GetInt("port", 0); port > 0 {
		updates[redisKey+".port"] = port
		updatedFields = append(updatedFields, "port")
	}

	if password := request.GetString("password", ""); password != "" {
		updates[redisKey+".password"] = password
		updatedFields = append(updatedFields, "password")
	}

	if database := request.GetInt("database", -1); database >= 0 && database <= common.MaxRedisDatabase {
		updates[redisKey+".database"] = database
		updatedFields = append(updatedFields, "database")
	}

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
			log.String("error", err.Error()),
			log.String(common.FieldProvider, "redis"),
			log.String(common.FieldOperation, "save_config"),
			log.String(common.FieldStatus, "failed"))
		return mcp.NewToolResultError(fmt.Sprintf("Error saving configuration: %s", err.Error())), nil
	}

	// 清除连接池中的连接，强制重新连接
	if s.redisPool != nil {
		s.redisPool.CloseConnection(name)
		log.Info(ctx, "清除Redis连接池",
			log.String("instance", name),
			log.String(common.FieldProvider, "redis"),
			log.String(common.FieldOperation, "clear_pool"))
	}

	// 格式化输出
	output := fmt.Sprintf("✅ Configuration updated for Redis instance '%s'\n", name)
	output += fmt.Sprintf("Updated fields (%d): %s\n", len(updatedFields), strings.Join(updatedFields, ", "))
	
	// 显示更新后的关键配置
	output += "\nCurrent configuration:\n"
	if host := viper.GetString(redisKey + ".host"); host != "" {
		output += fmt.Sprintf("  Host: %s\n", host)
	}
	if port := viper.GetInt(redisKey + ".port"); port > 0 {
		output += fmt.Sprintf("  Port: %d\n", port)
	}
	if database := viper.GetInt(redisKey + ".database"); database >= 0 {
		output += fmt.Sprintf("  Database: %d\n", database)
	}

	log.Info(ctx, "配置更新成功", 
		log.String("instance", name),
		log.String("fields", strings.Join(updatedFields, ",")),
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "update_config"),
		log.String(common.FieldStatus, "success"))

	return mcp.NewToolResultText(output[:len(output)-1]), nil // 去掉最后的换行符
}

// handleGetConfigDetails 处理获取配置详情的请求
func (s *RedisServer) handleGetConfigDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info(ctx, "处理get_config_details请求",
		log.String(common.FieldTool, "get_config_details"),
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "get_config_details"))

	name := request.GetString("name", "")
	includeSensitive := request.GetBool("include_sensitive", false)

	redisInstances := viper.GetStringMap("databases")
	if len(redisInstances) == 0 {
		return mcp.NewToolResultText("No Redis instances configured"), nil
	}

	activeRedis := viper.GetString("active_database")
	if activeRedis == "" {
		activeRedis = "default"
	}

	var output strings.Builder

	if name == "" {
		name = activeRedis
	}

	if name == "all" {
		// 显示所有实例的配置
		output.WriteString("📋 All Redis Instance Configurations\n")
		output.WriteString("===================================\n\n")

		for instanceName, _ := range redisInstances {
			output.WriteString(s.formatInstanceConfig(instanceName, instanceName == activeRedis, includeSensitive))
			output.WriteString("\n")
		}
	} else {
		// 显示指定实例的配置
		if _, exists := redisInstances[name]; !exists {
			return mcp.NewToolResultError(fmt.Sprintf("Error: Redis instance '%s' does not exist", name)), nil
		}
		
		output.WriteString(fmt.Sprintf("📋 Redis Instance Configuration: %s\n", name))
		output.WriteString("====================================\n\n")
		output.WriteString(s.formatInstanceConfig(name, name == activeRedis, includeSensitive))
	}

	return mcp.NewToolResultText(output.String()), nil
}

// formatInstanceConfig 格式化实例配置信息
func (s *RedisServer) formatInstanceConfig(name string, isActive bool, includeSensitive bool) string {
	redisKey := fmt.Sprintf("databases.%s", name)
	
	var output strings.Builder
	
	// 实例名称和状态
	status := "inactive"
	if isActive {
		status = "🟢 ACTIVE"
	} else {
		status = "⚪ inactive"
	}
	output.WriteString(fmt.Sprintf("Instance: %s (%s)\n", name, status))
	
	// 基本配置
	if host := viper.GetString(redisKey + ".host"); host != "" {
		output.WriteString(fmt.Sprintf("  Host: %s\n", host))
	}
	
	if port := viper.GetInt(redisKey + ".port"); port > 0 {
		output.WriteString(fmt.Sprintf("  Port: %d\n", port))
	}
	
	if password := viper.GetString(redisKey + ".password"); password != "" {
		if includeSensitive {
			output.WriteString(fmt.Sprintf("  Password: %s\n", password))
		} else {
			output.WriteString("  Password: *** (hidden)\n")
		}
	}
	
	if database := viper.GetInt(redisKey + ".database"); database >= 0 {
		output.WriteString(fmt.Sprintf("  Database: %d\n", database))
	}
	
	// 连接状态检查（仅对激活的实例）
	if isActive {
		if client, err := s.getConnection(context.Background()); err == nil && client != nil {
			output.WriteString("  Connection: ✅ Available\n")
			// 尝试获取Redis版本信息
			if result := client.Info(context.Background(), "server"); result.Err() == nil {
				info := result.Val()
				// 解析版本信息
				lines := strings.Split(info, "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "redis_version:") {
						version := strings.TrimPrefix(line, "redis_version:")
						version = strings.TrimSpace(version)
						output.WriteString(fmt.Sprintf("  Server Version: %s\n", version))
						break
					}
				}
			}
		} else {
			output.WriteString(fmt.Sprintf("  Connection: ❌ Failed (%s)\n", err.Error()))
		}
	}
	
	return output.String()
}

// validateRedisCommandSecurity 验证Redis命令的安全性
func (s *RedisServer) validateRedisCommandSecurity(ctx context.Context, cmdName string) error {
	cmdName = strings.ToUpper(cmdName)

	// 检查危险的数据操作命令
	deleteCommands := []string{"DEL", "UNLINK", "FLUSHDB", "FLUSHALL"}
	for _, cmd := range deleteCommands {
		if cmdName == cmd && s.disableDelete {
			return errors.New(fmt.Sprintf("%s操作已被禁用", cmdName))
		}
	}

	// 检查配置修改命令
	configCommands := []string{"CONFIG"}
	for _, cmd := range configCommands {
		if cmdName == cmd && s.disableUpdate {
			return errors.New(fmt.Sprintf("%s操作已被禁用", cmdName))
		}
	}

	// 检查危险的管理命令（始终禁用）
	adminCommands := []string{"SHUTDOWN", "DEBUG", "MONITOR", "CLIENT"}
	for _, cmd := range adminCommands {
		if cmdName == cmd {
			return errors.New(fmt.Sprintf("管理命令 %s 已被禁用", cmdName))
		}
	}

	// 检查脚本执行命令（根据用户配置）
	scriptCommands := []string{"EVAL", "EVALSHA", "SCRIPT"}
	for _, cmd := range scriptCommands {
		if cmdName == cmd && s.disableUpdate {
			return errors.New(fmt.Sprintf("脚本命令 %s 已被禁用", cmdName))
		}
	}

	log.Debug(ctx, "Redis命令安全检查通过",
		log.String("command", cmdName),
		log.String(common.FieldProvider, "redis"),
		log.String(common.FieldOperation, "validate_security"))

	return nil
}