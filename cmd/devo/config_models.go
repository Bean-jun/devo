package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"devo/internal/config"
	"devo/internal/taskexec/llmclient/providers/openai"
	"devo/internal/taskexec/tools"
)

func runConfigModels(args []string) {
	if len(args) == 0 {
		printUsage()
		return
	}

	switch args[0] {
	case "list":
		handleConfigModelsList()
	case "add":
		handleConfigModelsAdd(args[1:])
	case "remove":
		handleConfigModelsRemove(args[1:])
	case "activate":
		handleConfigModelsActivate(args[1:])
	case "test":
		handleConfigModelsTest(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`用法: devo config models <子命令> [参数]

子命令:
  list              列出所有模型
  add               添加模型
  remove --id <id>  删除模型
  activate --id <id> 激活模型
  test --id <id>    测试连接

示例:
  devo config models list
  devo config models add --name "GPT-4o" --api-key "sk-xxx" --model "gpt-4o"
  devo config models remove --id gpt-4o
  devo config models activate --id gpt-4o
  devo config models test --id gpt-4o`)
}

func handleConfigModelsList() {
	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		fmt.Println("暂无模型配置")
		return
	}

	if len(cfg.LLM.Models) == 0 {
		fmt.Println("暂无模型配置")
		return
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
}

func handleConfigModelsAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	name := fs.String("name", "", "模型名称")
	apiKey := fs.String("api-key", "", "API Key")
	baseURL := fs.String("base-url", "", "API Base URL (默认: https://api.openai.com/v1)")
	model := fs.String("model", "", "模型名称 (如: gpt-4o)")
	enableReasoning := fs.Bool("enable-reasoning", false, "启用推理")
	reasoningEffort := fs.String("reasoning-effort", "", "推理强度 (low/medium/high)")
	maxTokens := fs.Int("max-tokens", 0, "最大 Token 数")
	_ = fs.Parse(args)

	if *name == "" || *apiKey == "" || *model == "" {
		fmt.Fprintln(os.Stderr, "错误: --name, --api-key, --model 为必填参数")
		fs.Usage()
		os.Exit(1)
	}

	cfg, err := config.LoadGlobal()
	if err != nil {
		cfg = &config.Config{}
	}

	id := config.Slugify(*name)

	for _, m := range cfg.LLM.Models {
		if m.ID == id {
			fmt.Fprintf(os.Stderr, "错误: 模型 ID '%s' 已存在\n", id)
			os.Exit(1)
		}
	}

	baseURLVal := *baseURL
	if baseURLVal == "" {
		baseURLVal = config.DefaultLLMBaseURL
	}

	newModel := config.ModelConfig{
		ID:              id,
		Name:            *name,
		APIKey:          *apiKey,
		BaseURL:         baseURLVal,
		Model:           *model,
		EnableReasoning: *enableReasoning,
		ReasoningEffort: *reasoningEffort,
		MaxTokens:       *maxTokens,
	}

	cfg.LLM.Models = append(cfg.LLM.Models, newModel)
	if cfg.LLM.ActiveModelID == "" {
		cfg.LLM.ActiveModelID = id
	}

	if err := config.SaveGlobalConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("已添加模型: %s (%s)\n", *name, id)
}

func handleConfigModelsRemove(args []string) {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	id := fs.String("id", "", "模型 ID")
	_ = fs.Parse(args)

	if *id == "" {
		fmt.Fprintln(os.Stderr, "错误: --id 为必填参数")
		os.Exit(1)
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		fmt.Fprintln(os.Stderr, "无配置可删除")
		os.Exit(1)
	}

	found := false
	for i, m := range cfg.LLM.Models {
		if m.ID == *id {
			cfg.LLM.Models = append(cfg.LLM.Models[:i], cfg.LLM.Models[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		fmt.Fprintf(os.Stderr, "错误: 模型 '%s' 不存在\n", *id)
		os.Exit(1)
	}

	if cfg.LLM.ActiveModelID == *id {
		if len(cfg.LLM.Models) > 0 {
			cfg.LLM.ActiveModelID = cfg.LLM.Models[0].ID
		} else {
			cfg.LLM.ActiveModelID = ""
		}
	}

	if err := config.SaveGlobalConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("已删除模型: %s\n", *id)
}

func handleConfigModelsActivate(args []string) {
	fs := flag.NewFlagSet("activate", flag.ExitOnError)
	id := fs.String("id", "", "模型 ID")
	_ = fs.Parse(args)

	if *id == "" {
		fmt.Fprintln(os.Stderr, "错误: --id 为必填参数")
		os.Exit(1)
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		fmt.Fprintln(os.Stderr, "无配置可激活")
		os.Exit(1)
	}

	found := false
	for _, m := range cfg.LLM.Models {
		if m.ID == *id {
			found = true
			break
		}
	}

	if !found {
		fmt.Fprintf(os.Stderr, "错误: 模型 '%s' 不存在\n", *id)
		os.Exit(1)
	}

	cfg.LLM.ActiveModelID = *id

	if err := config.SaveGlobalConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("已激活模型: %s\n", *id)
}

func handleConfigModelsTest(args []string) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	id := fs.String("id", "", "模型 ID")
	_ = fs.Parse(args)

	if *id == "" {
		fmt.Fprintln(os.Stderr, "错误: --id 为必填参数")
		os.Exit(1)
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		fmt.Fprintln(os.Stderr, "加载配置失败")
		os.Exit(1)
	}

	var model *config.ModelConfig
	for i := range cfg.LLM.Models {
		if cfg.LLM.Models[i].ID == *id {
			model = &cfg.LLM.Models[i]
			break
		}
	}

	if model == nil {
		fmt.Fprintf(os.Stderr, "错误: 模型 '%s' 不存在\n", *id)
		os.Exit(1)
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

	fmt.Printf("正在测试模型 '%s' 的连接...\n", *id)
	err = client.TestConnection(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("连接成功!")
}

func runConfigOnboard() {
	fmt.Println("=== Devo LLM 配置引导 ===")
	fmt.Println()

	var name, apiKey, baseURL, model string

	fmt.Print("模型名称 (如: GPT-4o): ")
	fmt.Scanln(&name)
	if name == "" {
		fmt.Fprintln(os.Stderr, "错误: 模型名称不能为空")
		os.Exit(1)
	}

	fmt.Print("API Key: ")
	fmt.Scanln(&apiKey)
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "错误: API Key 不能为空")
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n配置已保存! 模型 '%s' (%s) 已激活。\n", name, id)
	fmt.Println("现在可以启动 devo 开始使用。")
}

func formatJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
