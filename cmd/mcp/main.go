package main

import (
	"fmt"
	"mcp/database"
	"os"

	"github.com/spf13/cobra"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

var version = "1.0.4"

func main() {
	// 创建根命令
	rootCmd := &cobra.Command{
		Use:     "mcp",
		Short:   "MCP Database Tools",
		Long:    "MCP (Model Context Protocol) Database Tools - Execute MySQL, Redis, ClickHouse and SQLite commands through AI assistants",
		Version: version,
	}

	// database 命令
	databaseCmd := &cobra.Command{
		Use:   "database",
		Short: "Start Database MCP Server (MySQL + Redis + ClickHouse + SQLite)",
		Long:  "Start Database MCP Server that provides MySQL, Redis, ClickHouse and SQLite execution tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := database.NewServer()
			return mcpserver.ServeStdio(server)
		},
	}

	// 添加命令
	rootCmd.AddCommand(databaseCmd)

	// 执行
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}