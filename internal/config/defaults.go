package config

import "devo/internal/core/approval"

const (
	DefaultLLMBaseURL                = "https://api.openai.com/v1"
	DefaultLLMModel                  = "gpt-4o"
	DefaultApprovalTimeoutSecs       = 300
	DefaultReasoningEffort           = "medium"
	DefaultMaxTokens                 = 128000
	DefaultToolCallLimit             = 50
	DefaultMaxContextTokens          = 128000
	DefaultKeepRecent                = 30
	DefaultMaxConcurrentToolCalls    = 1
	DefaultMaxConcurrentSubprocesses = 5
)

func DefaultApprovalPolicy() map[approval.OperationType]approval.PolicyLevel {
	return map[approval.OperationType]approval.PolicyLevel{
		approval.OpFileWriteNew:       approval.PolicyAlwaysAsk,
		approval.OpFileWriteOverwrite: approval.PolicyAlwaysAsk,
		approval.OpFileEdit:           approval.PolicyAlwaysAsk,
		approval.OpExecPython:         approval.PolicyAlwaysAsk,
		approval.OpMemoryUpdate:       approval.PolicyAutoApprove,
		approval.OpSolidifySkill:      approval.PolicyAutoApprove,
	}
}

func DefaultApprovalPolicyMap() map[string]string {
	return map[string]string{
		string(approval.OpFileWriteNew):       string(approval.PolicyAlwaysAsk),
		string(approval.OpFileWriteOverwrite): string(approval.PolicyAlwaysAsk),
		string(approval.OpFileEdit):           string(approval.PolicyAlwaysAsk),
		string(approval.OpExecPython):         string(approval.PolicyAlwaysAsk),
		string(approval.OpMemoryUpdate):       string(approval.PolicyAutoApprove),
		string(approval.OpSolidifySkill):      string(approval.PolicyAutoApprove),
	}
}

func DefaultConfig() *Config {
	cfg := &Config{}
	ApplyDefaults(cfg)
	return cfg
}
