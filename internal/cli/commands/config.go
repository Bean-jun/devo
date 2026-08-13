package commands

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理 devo 配置",
	Long:  `管理 devo 的全局配置，包括 LLM 模型配置（models）和交互式配置引导（onboard）。`,
}

func init() {
	configCmd.AddCommand(configModelsCmd)
	configCmd.AddCommand(configOnboardCmd)
}
