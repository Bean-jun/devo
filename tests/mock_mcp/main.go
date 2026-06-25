package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	mode = flag.String("mode", "http", "run mode: 'http' or 'stdio'")
	host = flag.String("host", "127.0.0.1", "host to listen on (http mode only)")
	port = flag.Int("port", 9080, "port to listen on (http mode only)")
)

var mockFiles = map[string]string{
	"README.md":    "# Mock Project\n\nThis is a mock project for testing Devo MCP integration.\n\n## Usage\n\nRun `make test` to execute tests.",
	"main.go":      "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, Devo!\")\n}\n",
	"config.json":  "{\n  \"version\": \"1.0.0\",\n  \"debug\": false,\n  \"database\": {\n    \"host\": \"localhost\",\n    \"port\": 5432\n  }\n}\n",
	"src/utils.go": "package src\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n\nfunc Multiply(a, b int) int {\n\treturn a * b\n}\n",
}

func main() {
	flag.Parse()

	switch *mode {
	case "http":
		runHTTP()
	case "stdio":
		runStdio()
	default:
		log.Fatalf("Unknown mode: %s (use 'http' or 'stdio')", *mode)
	}
}

func runHTTP() {
	addr := fmt.Sprintf("%s:%d", *host, *port)

	server := newServer()

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		httpServer.Shutdown(context.Background())
	}()

	log.Printf("Mock MCP server (HTTP mode) listening on http://%s/mcp", addr)
	log.Println("Available tools: ping, mock_search, mock_fetch, mock_calculate, mock_echo, mock_list_files")
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

func runStdio() {
	server := newServer()

	log.SetOutput(os.Stderr)
	log.Println("Mock MCP server (stdio mode) starting on stdin/stdout...")
	log.Println("Available tools: ping, mock_search, mock_fetch, mock_calculate, mock_echo, mock_list_files")

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Stdio server failed: %v", err)
	}
}

func newServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "devo-mock-mcp",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ping",
		Description: "Simple connectivity test tool. Returns 'pong' with the server timestamp.",
	}, pingHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mock_search",
		Description: "Search through mock data files. Returns matching file names and their content snippets.",
	}, mockSearchHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mock_fetch",
		Description: "Fetch the full content of a mock file by name. Available files: README.md, main.go, config.json, src/utils.go",
	}, mockFetchHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mock_calculate",
		Description: "Perform arithmetic calculations. Supports +, -, *, / operations.",
	}, mockCalculateHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mock_echo",
		Description: "Echo back the input message. Useful for testing request/response roundtrip.",
	}, mockEchoHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mock_list_files",
		Description: "List all available mock files and their sizes.",
	}, mockListFilesHandler)

	return server
}

type PingParams struct {
	Message string `json:"message,omitempty" jsonschema:"Optional message to include in the response"`
}

func pingHandler(ctx context.Context, req *mcp.CallToolRequest, params PingParams) (*mcp.CallToolResult, any, error) {
	msg := params.Message
	if msg == "" {
		msg = "pong"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("pong: %s", msg),
			},
		},
	}, nil, nil
}

type MockSearchParams struct {
	Query string `json:"query" jsonschema:"Search query string to match against file names and content"`
}

func mockSearchHandler(ctx context.Context, req *mcp.CallToolRequest, params MockSearchParams) (*mcp.CallToolResult, any, error) {
	query := strings.ToLower(params.Query)
	if query == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Please provide a search query."},
			},
		}, nil, nil
	}

	var results []string
	for name, content := range mockFiles {
		if strings.Contains(strings.ToLower(name), query) || strings.Contains(strings.ToLower(content), query) {
			preview := content
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			results = append(results, fmt.Sprintf("- %s: %s", name, preview))
		}
	}

	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("No files found matching '%s'", params.Query)},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Found %d file(s) matching '%s':\n%s", len(results), params.Query, strings.Join(results, "\n"))},
		},
	}, nil, nil
}

type MockFetchParams struct {
	Name string `json:"name" jsonschema:"Name of the mock file to fetch (e.g. README.md, main.go, config.json, src/utils.go)"`
}

func mockFetchHandler(ctx context.Context, req *mcp.CallToolRequest, params MockFetchParams) (*mcp.CallToolResult, any, error) {
	content, ok := mockFiles[params.Name]
	if !ok {
		var available []string
		for name := range mockFiles {
			available = append(available, name)
		}
		sort.Strings(available)
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("File '%s' not found. Available files: %s", params.Name, strings.Join(available, ", "))},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Content of %s:\n```\n%s\n```", params.Name, content)},
		},
	}, nil, nil
}

type MockCalculateParams struct {
	Expression string `json:"expression" jsonschema:"Arithmetic expression to evaluate (e.g. '2 + 3 * 4')"`
}

func mockCalculateHandler(ctx context.Context, req *mcp.CallToolRequest, params MockCalculateParams) (*mcp.CallToolResult, any, error) {
	expr := strings.TrimSpace(params.Expression)
	if expr == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Please provide an expression to calculate."},
			},
		}, nil, nil
	}

	result, err := evaluateSimple(expr)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error evaluating '%s': %v", params.Expression, err)},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("%s = %v", params.Expression, result)},
		},
	}, nil, nil
}

func evaluateSimple(expr string) (float64, error) {
	expr = strings.ReplaceAll(expr, " ", "")

	parts := strings.FieldsFunc(expr, func(r rune) bool {
		return r == '+' || r == '-' || r == '*' || r == '/'
	})

	if len(parts) == 0 {
		return 0, fmt.Errorf("no numbers found")
	}

	operators := make([]rune, 0)
	for _, ch := range expr {
		if ch == '+' || ch == '-' || ch == '*' || ch == '/' {
			operators = append(operators, ch)
		}
	}

	if len(operators) == 0 {
		val, err := strconv.ParseFloat(expr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", expr)
		}
		return val, nil
	}

	nums := make([]float64, 0, len(parts))
	for _, p := range parts {
		val, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", p)
		}
		nums = append(nums, val)
	}

	for i := 0; i < len(operators); i++ {
		if operators[i] == '*' || operators[i] == '/' {
			if operators[i] == '*' {
				nums[i] = nums[i] * nums[i+1]
			} else {
				if nums[i+1] == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				nums[i] = nums[i] / nums[i+1]
			}
			nums = append(nums[:i+1], nums[i+2:]...)
			operators = append(operators[:i], operators[i+1:]...)
			i--
		}
	}

	result := nums[0]
	for i := 0; i < len(operators); i++ {
		if operators[i] == '+' {
			result += nums[i+1]
		} else {
			result -= nums[i+1]
		}
	}

	return result, nil
}

type MockEchoParams struct {
	Message string `json:"message" jsonschema:"The message to echo back"`
}

func mockEchoHandler(ctx context.Context, req *mcp.CallToolRequest, params MockEchoParams) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Echo: %s", params.Message),
			},
		},
	}, nil, nil
}

type MockListFilesParams struct{}

func mockListFilesHandler(ctx context.Context, req *mcp.CallToolRequest, params MockListFilesParams) (*mcp.CallToolResult, any, error) {
	var lines []string
	lines = append(lines, "Available mock files:")
	for name, content := range mockFiles {
		lines = append(lines, fmt.Sprintf("  - %s (%d bytes)", name, len(content)))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(lines, "\n")},
		},
	}, nil, nil
}
