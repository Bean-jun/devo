package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"devo/internal/cli"
	"devo/internal/config"
	"devo/internal/pkg/logging"
	"devo/internal/taskexec/pathsec"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devo",
	Short: "Devo - AI 驱动的开发助手",
	Long: `Devo 是一个 AI 驱动的开发助手工具，支持终端交互模式（TUI）和 Web 界面模式。

默认启动 Web 服务器，可通过 --tui 切换到终端交互模式。`,
	Version: version,
	RunE:    runServer,
}

var version string

func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().Bool("tui", false, "启动终端交互模式")
	rootCmd.PersistentFlags().Bool("web", false, "启动时自动打开浏览器")
	rootCmd.PersistentFlags().Int("port", 0, "Web 服务器端口（0 = 自动分配）")
	rootCmd.PersistentFlags().String("workspace", "", "默认工作目录（不设置则使用当前目录）")
	rootCmd.PersistentFlags().String("log-level", "info", "日志级别：debug, info, warn, error")

	rootCmd.AddCommand(configCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func runServer(cmd *cobra.Command, args []string) error {
	cli.VerifyDevoDir()

	tui, _ := cmd.Flags().GetBool("tui")
	web, _ := cmd.Flags().GetBool("web")
	port, _ := cmd.Flags().GetInt("port")
	workspace, _ := cmd.Flags().GetString("workspace")
	logLevel, _ := cmd.Flags().GetString("log-level")

	logging.Init(logging.Config{
		Level:   parseLogLevel(logLevel),
		LogPath: config.LogPath(),
	})

	logging.Info(context.Background(), "devo starting",
		"version", version,
		"log_level", logLevel,
	)

	if workspace != "" {
		normalized := pathsec.NormalizePath(workspace)
		if err := os.Chdir(normalized); err != nil {
			logging.Error(context.Background(), "failed to change working directory",
				"path", workspace,
				"error", err,
			)
			return fmt.Errorf("切换工作目录失败: %w", err)
		}
		logging.Info(context.Background(), "working directory set",
			"path", normalized,
		)
	}

	app, err := cli.NewApp(tui, web, port, version)
	if err != nil {
		logging.Error(context.Background(), "failed to create app",
			"error", err,
		)
		return fmt.Errorf("启动应用失败: %w", err)
	}
	defer app.Shutdown()

	app.Run()
	return nil
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
