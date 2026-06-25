package mcp

type McpServerConfig struct {
	ServerID  string `json:"server_id"`
	Source    string `json:"source"`
	Endpoint  string `json:"endpoint"`
	Transport string `json:"transport"`
}

type McpTool struct {
	ToolName    string      `json:"tool_name"`
	ServerID    string      `json:"server_id"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type ServerStatus string

const (
	StatusConnected    ServerStatus = "connected"
	StatusDisconnected ServerStatus = "disconnected"
	StatusError        ServerStatus = "error"
)

type ServerInfo struct {
	Config   McpServerConfig `json:"config"`
	Status   ServerStatus    `json:"status"`
	Tools    []McpTool       `json:"tools"`
	ErrorMsg string          `json:"error_msg,omitempty"`
}
