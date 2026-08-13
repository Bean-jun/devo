package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"devo/internal/config"
	"devo/internal/taskexec/llmclient/providers/openai"
	"devo/internal/taskexec/tools"

	"github.com/spf13/cobra"
)

var configModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "管理 LLM 模型配置",
	Long:  `管理 devo 使用的 LLM 模型配置，支持添加、删除、列出、激活和测试模型连接。`,
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有已配置的模型",
	RunE:  runModelsList,
}

var modelsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "添加新的 LLM 模型配置",
	Long:  "添加一个新的 LLM 模型配置到全局配置中。添加后自动设为当前模型（如果此前没有激活模型）。",
	Example: `  devo config models add --name "GPT-4o" --api-key "sk-xxx" --model "gpt-4o"
  devo config models add --name "GPT-4o" --api-key "sk-xxx" --model "gpt-4o" --base-url "https://api.openai.com/v1"`,
	RunE: runModelsAdd,
}

var modelsRemoveCmd = &cobra.Command{
	Use:     "remove",
	Short:   "删除模型配置",
	Example: `  devo config models remove --id gpt-4o`,
	RunE:    runModelsRemove,
}

var modelsActivateCmd = &cobra.Command{
	Use:     "activate",
	Short:   "激活指定的模型",
	Example: `  devo config models activate --id gpt-4o`,
	RunE:    runModelsActivate,
}

var modelsTestCmd = &cobra.Command{
	Use:     "test",
	Short:   "测试模型连接",
	Example: `  devo config models test --id gpt-4o`,
	RunE:    runModelsTest,
}

func init() {
	modelsAddCmd.Flags().String("name", "", "模型名称（必填）")
	modelsAddCmd.Flags().String("api-key", "", "API Key（必填）")
	modelsAddCmd.Flags().String("model", "", "模型名称，如 gpt-4o（必填）")
	modelsAddCmd.Flags().String("base-url", "", "API Base URL（默认: https://api.openai.com/v1）")
	modelsAddCmd.Flags().Bool("enable-reasoning", false, "启用推理功能")
	modelsAddCmd.Flags().String("reasoning-effort", "", "推理强度（low/medium/high）")
	modelsAddCmd.Flags().Int("max-tokens", 128000, "最大 Token 数")

	modelsRemoveCmd.Flags().String("id", "", "模型 ID（必填）")
	modelsActivateCmd.Flags().String("id", "", "模型 ID（必填）")
	modelsTestCmd.Flags().String("id", "", "模型 ID（必填）")

	configModelsCmd.AddCommand(modelsListCmd)
	configModelsCmd.AddCommand(modelsAddCmd)
	configModelsCmd.AddCommand(modelsRemoveCmd)
	configModelsCmd.AddCommand(modelsActivateCmd)
	configModelsCmd.AddCommand(modelsTestCmd)
}

func runModelsList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		fmt.Println("暂无模型配置")
		return nil
	}

	if len(cfg.LLM.Models) == 0 {
		fmt.Println("暂无模型配置")
		return nil
	}

	fmt.Printf("%-20s %-20s %-20s %s\n", "ID", "名称", "模型", "状态")
	fmt.Println(strings.Repeat("-", 80))
	for _, m := range cfg.LLM.Models {
		status := ""
		if m.ID == cfg.LLM.ActiveModelID {
			status = " ✓ 当前"
		}
		fmt.Printf("%-20s %-20s %-20s%s\n", m.ID, m.Name, m.Model, status)
	}
	return nil
}

func runModelsAdd(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	apiKey, _ := cmd.Flags().GetString("api-key")
	model, _ := cmd.Flags().GetString("model")
	baseURL, _ := cmd.Flags().GetString("base-url")
	enableReasoning, _ := cmd.Flags().GetBool("enable-reasoning")
	reasoningEffort, _ := cmd.Flags().GetString("reasoning-effort")
	maxTokens, _ := cmd.Flags().GetInt("max-tokens")

	if name == "" || apiKey == "" || model == "" {
		return fmt.Errorf("错误: --name, --api-key, --model 为必填参数")
	}

	cfg, err := config.LoadGlobal()
	if err != nil {
		cfg = &config.Config{}
	}

	id := config.Slugify(name)

	for _, m := range cfg.LLM.Models {
		if m.ID == id {
			return fmt.Errorf("错误: 模型 ID '%s' 已存在", id)
		}
	}

	baseURLVal := baseURL
	if baseURLVal == "" {
		baseURLVal = config.DefaultLLMBaseURL
	}

	newModel := config.ModelConfig{
		ID:              id,
		Name:            name,
		APIKey:          apiKey,
		BaseURL:         baseURLVal,
		Model:           model,
		EnableReasoning: enableReasoning,
		ReasoningEffort: reasoningEffort,
		MaxTokens:       maxTokens,
	}

	cfg.LLM.Models = append(cfg.LLM.Models, newModel)
	if cfg.LLM.ActiveModelID == "" {
		cfg.LLM.ActiveModelID = id
	}

	if err := config.SaveGlobalConfig(cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Printf("已添加模型: %s (%s)\n", name, id)
	return nil
}

func runModelsRemove(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")

	if id == "" {
		return fmt.Errorf("错误: --id 为必填参数")
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		return fmt.Errorf("无配置可删除")
	}

	found := false
	for i, m := range cfg.LLM.Models {
		if m.ID == id {
			cfg.LLM.Models = append(cfg.LLM.Models[:i], cfg.LLM.Models[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("错误: 模型 '%s' 不存在", id)
	}

	if cfg.LLM.ActiveModelID == id {
		if len(cfg.LLM.Models) > 0 {
			cfg.LLM.ActiveModelID = cfg.LLM.Models[0].ID
		} else {
			cfg.LLM.ActiveModelID = ""
		}
	}

	if err := config.SaveGlobalConfig(cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Printf("已删除模型: %s\n", id)
	return nil
}

func runModelsActivate(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")

	if id == "" {
		return fmt.Errorf("错误: --id 为必填参数")
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		return fmt.Errorf("无配置可激活")
	}

	found := false
	for _, m := range cfg.LLM.Models {
		if m.ID == id {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("错误: 模型 '%s' 不存在", id)
	}

	cfg.LLM.ActiveModelID = id

	if err := config.SaveGlobalConfig(cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Printf("已激活模型: %s\n", id)
	return nil
}

func runModelsTest(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")

	if id == "" {
		return fmt.Errorf("错误: --id 为必填参数")
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		return fmt.Errorf("加载配置失败")
	}

	var model *config.ModelConfig
	for i := range cfg.LLM.Models {
		if cfg.LLM.Models[i].ID == id {
			model = &cfg.LLM.Models[i]
			break
		}
	}

	if model == nil {
		return fmt.Errorf("错误: 模型 '%s' 不存在", id)
	}

	llmCfg := &config.LLMConfig{
		APIKey:          model.APIKey,
		BaseURL:         model.BaseURL,
		Model:           model.Model,
		ExtraHeaders:    model.ExtraHeaders,
		EnableReasoning: model.EnableReasoning,
		ReasoningEffort: model.ReasoningEffort,
		MaxTokens:       model.MaxTokens,
	}

	client := openai.New(openai.Config{
		LLMConfig: llmCfg,
	}, tools.NewRegistry())

	fmt.Printf("正在测试模型 '%s' 的连接...\n", id)
	err = client.TestConnection(context.Background())
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}

	fmt.Println("连接成功!")
	return nil
}

func formatJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
