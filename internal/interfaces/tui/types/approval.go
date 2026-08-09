package types

type ApprovalRequest struct {
	ApprovalID     string
	OperationType  string
	RiskLevel      string
	Summary        string
	Diff           string
	CommandPreview string
	Params         map[string]interface{}
}

type ApproveRequest struct {
	ApprovalID string `json:"approval_id,omitempty"`
	Decision   string `json:"decision"`
}

type UpdateConfigRequest struct {
	ToolCallLimit int `json:"tool_call_limit,omitempty"`
}

type FileInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"is_dir"`
}
