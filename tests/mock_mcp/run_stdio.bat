@echo off
echo Starting Mock MCP Server (stdio mode) on stdin/stdout...
echo.
echo Available tools: ping, mock_search, mock_fetch, mock_calculate, mock_echo, mock_list_files
echo.
echo This mode is meant to be invoked by an MCP client (e.g., Devo) as a subprocess.
echo To test manually, pipe JSON-RPC requests to stdin and read responses from stdout.
echo.
go run main.go --mode stdio