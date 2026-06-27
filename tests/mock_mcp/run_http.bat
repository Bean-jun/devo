@echo off
echo Starting Mock MCP Server (HTTP mode) on http://127.0.0.1:8090/mcp
echo.
echo Available tools: ping, mock_search, mock_fetch, mock_calculate, mock_echo, mock_list_files
echo.
go build -o mock_mcp_server_http.exe main.go
.\mock_mcp_server_http.exe --mode http --host 127.0.0.1 --port 8090