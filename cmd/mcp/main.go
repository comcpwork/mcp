package main

import (
	"fmt"
	"mcp/pkg/log"
	"mcp/internal/i18n"
	"mcp/internal/providers"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.1"
	cfgFile string
)

func main() {
	// 设置日志命令（如果有）
	if len(os.Args) > 1 {
		log.SetCommand(os.Args[1])
	}

	// 创建根命令
	rootDesc := i18n.GetRootCommand()
	rootCmd := &cobra.Command{
		Use:     "mcp",
		Short:   rootDesc.Short,
		Long:    rootDesc.Long,
		Version: version,
	}

	// 设置帮助命令为中文
	helpDesc := i18n.GetHelpCommand()
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   helpDesc.Use + " [命令]",
		Short: helpDesc.Short,
		Long:  helpDesc.Long,
		Run: func(c *cobra.Command, args []string) {
			cmd, _, e := c.Root().Find(args)
			if cmd == nil || e != nil {
				c.Printf("%s\n", i18n.GetMessage("unknown_help_topic", args))
				c.Root().Usage()
			} else {
				cmd.Help()
			}
		},
	})

	// 自定义使用模板以支持中文
	rootCmd.SetUsageTemplate(i18n.GetUsageTemplate())
	
	// 移除中文帮助标志的支持，只使用标准的 --help

	// 移除自动补全命令（或者也可以将其翻译为中文）
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// 设置版本模板
	rootCmd.SetVersionTemplate(i18n.GetMessage("version") + "\n")

	// 全局参数
	rootCmd.PersistentFlags().StringVar(&cfgFile, i18n.GetFlag("config"), "", i18n.GetFlagDesc("config"))

	// 注册所有服务器命令
	providers.RegisterCommands(rootCmd)

	// 执行命令
	rootCmd.SilenceUsage = true  // 出错时不显示使用帮助
	rootCmd.SilenceErrors = true // 由我们自己处理错误
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, i18n.FormatError(err))
		os.Exit(1)
	}
}
