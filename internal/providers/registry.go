package providers

import (
	"context"
	"fmt"
	"mcp/internal/config"
	"mcp/internal/i18n"
	"mcp/internal/server"
	"mcp/pkg/log"
	"mcp/internal/providers/mysql"
	"mcp/internal/providers/redis"
	"mcp/internal/providers/pulsar"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// ServerFactory 服务器工厂函数
type ServerFactory func() server.MCPServer

// GetRegistry 获取服务器注册表
func GetRegistry() map[string]ServerFactory {
	return map[string]ServerFactory{
		"mysql":  mysql.NewServer,
		"redis":  redis.NewServer,
		"pulsar": pulsar.NewServer,
	}
}

// RegisterCommands 注册所有服务器命令
func RegisterCommands(root *cobra.Command) {
	registry := GetRegistry()
	for name, factory := range registry {
		cmd := createServerCommand(name, factory)
		root.AddCommand(cmd)
	}
}

// createServerCommand 创建服务器命令
func createServerCommand(name string, factory ServerFactory) *cobra.Command {
	var transport string
	var port int
	
	// 细分的安全选项
	var disableCreate bool
	var disableDrop bool
	var disableAlter bool
	var disableTruncate bool
	var disableUpdate bool
	var disableDelete bool

	desc := i18n.GetServerCommand(name)
	cmd := &cobra.Command{
		Use:   desc.Use,
		Short: desc.Short,
		Long:  desc.Long,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// 添加日志字段
			ctx = log.WithFields(ctx,
				log.String("server", name),
				log.String("transport", transport),
			)

			log.Info(ctx, i18n.GetMessage("starting_server"))

			// 创建服务器实例
			srv := factory()

			// 为所有服务器设置安全选项
			switch s := srv.(type) {
			case *mysql.MySQLServer:
				s.SetSecurityOptions(disableCreate, disableDrop, disableAlter, disableTruncate, disableUpdate, disableDelete)
			case *redis.RedisServer:
				s.SetSecurityOptions(disableCreate, disableDrop, disableAlter, disableTruncate, disableUpdate, disableDelete)
			case *pulsar.PulsarServer:
				s.SetSecurityOptions(disableCreate, disableDrop, disableAlter, disableTruncate, disableUpdate, disableDelete)
			}

			// 初始化配置
			cfg := config.NewConfig(name)
			if err := cfg.Init(ctx); err != nil {
				return fmt.Errorf(i18n.GetMessage("init_config_failed"), err)
			}

			// 初始化服务器
			if err := srv.Init(ctx); err != nil {
				return fmt.Errorf(i18n.GetMessage("init_server_failed"), err)
			}

			// 注册工具
			if err := srv.RegisterTools(); err != nil {
				return fmt.Errorf(i18n.GetMessage("register_tools_failed"), err)
			}

			// 注册资源
			if err := srv.RegisterResources(); err != nil {
				return fmt.Errorf(i18n.GetMessage("register_resources_failed"), err)
			}

			// 启动服务器
			return startServer(ctx, srv, transport, port)
		},
	}

	// 添加命令参数
	cmd.Flags().StringVarP(&transport, i18n.GetFlag("transport"), "t", "stdio", i18n.GetFlagDesc("transport"))
	cmd.Flags().IntVarP(&port, i18n.GetFlag("port"), "p", 8080, i18n.GetFlagDesc("port"))

	// 为所有 provider 添加细分的安全选项
	cmd.Flags().BoolVar(&disableCreate, "disable-create", false, "禁用CREATE操作（CREATE TABLE、CREATE DATABASE等）")
	cmd.Flags().BoolVar(&disableDrop, "disable-drop", false, "禁用DROP操作（DROP TABLE、DROP DATABASE等）")
	cmd.Flags().BoolVar(&disableAlter, "disable-alter", false, "禁用ALTER操作（ALTER TABLE等）")
	cmd.Flags().BoolVar(&disableTruncate, "disable-truncate", false, "禁用TRUNCATE操作")
	cmd.Flags().BoolVar(&disableUpdate, "disable-update", false, "禁用UPDATE操作")
	cmd.Flags().BoolVar(&disableDelete, "disable-delete", false, "禁用DELETE操作")

	// 设置中文使用模板
	cmd.SetUsageTemplate(i18n.GetServerUsageTemplate())

	return cmd
}

// startServer 根据传输方式启动服务器
func startServer(ctx context.Context, srv server.MCPServer, transport string, port int) error {
	mcpServer := srv.GetServer()

	switch transport {
	case "stdio":
		log.Info(ctx, i18n.GetMessage("using_stdio"))
		return mcpserver.ServeStdio(mcpServer)

	case "sse":
		addr := fmt.Sprintf(":%d", port)
		log.Info(ctx, i18n.GetMessage("using_sse"), log.String("addr", addr))
		sseServer := mcpserver.NewSSEServer(mcpServer)
		return sseServer.Start(addr)

	case "http":
		addr := fmt.Sprintf(":%d", port)
		log.Info(ctx, i18n.GetMessage("using_http"), log.String("addr", addr))
		httpServer := mcpserver.NewStreamableHTTPServer(mcpServer)
		return httpServer.Start(addr)

	default:
		return fmt.Errorf(i18n.GetMessage("unsupported_transport"), transport)
	}
}
