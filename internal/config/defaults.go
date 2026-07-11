package config

import "devo/internal/core/approval"

const (
	DefaultLLMBaseURL           = "https://api.openai.com/v1"
	DefaultLLMModel             = "gpt-4o"
	DefaultToolCallLimit        = 50
	DefaultKeepRecent           = 20
	DefaultMaxContextTokens     = 128000
	DefaultContextCompressRatio = 0.8
	DefaultApprovalTimeoutSecs  = 300
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
