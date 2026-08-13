package commands

import (
	"fmt"

	"devo/internal/config"

	"github.com/spf13/cobra"
)

var configOnboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "交互式 LLM 配置引导",
	Long:  "通过交互式问答的方式引导你完成 LLM 模型的初始配置，适合首次使用 devo 时快速上手。",
	RunE:  runConfigOnboard,
}

func runConfigOnboard(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Devo LLM 配置引导 ===")
	fmt.Println()

	var name, apiKey, baseURL, model string

	fmt.Print("模型名称 (如: GPT-4o): ")
	fmt.Scanln(&name)
	if name == "" {
		return fmt.Errorf("错误: 模型名称不能为空")
	}

	fmt.Print("API Key: ")
	fmt.Scanln(&apiKey)
	if apiKey == "" {
		return fmt.Errorf("错误: API Key 不能为空")
	}

	fmt.Print("API Base URL (默认: https://api.openai.com/v1): ")
	fmt.Scanln(&baseURL)
	if baseURL == "" {
		baseURL = config.DefaultLLMBaseURL
	}

	fmt.Print("模型名称 (默认: gpt-4o): ")
	fmt.Scanln(&model)
	if model == "" {
		model = config.DefaultLLMModel
	}

	fmt.Print("最大输出 Token 数 (默认: 128000): ")
	maxTokens, _ := cmd.Flags().GetInt("max-tokens")
	if maxTokens == 0 {
		maxTokens = config.DefaultMaxTokens
	}

	cfg, err := config.LoadGlobal()
	if err != nil {
		cfg = &config.Config{}
	}

	id := config.Slugify(name)

	newModel := config.ModelConfig{
		ID:      id,
		Name:    name,
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}

	cfg.LLM.Models = append(cfg.LLM.Models, newModel)
	cfg.LLM.ActiveModelID = id

	if err := config.SaveGlobalConfig(cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Printf("\n配置已保存! 模型 '%s' (%s) 已激活。\n", name, id)
	fmt.Println("现在可以启动 devo 开始使用。")
	return nil
}
