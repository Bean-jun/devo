@echo off
echo Starting Mock MCP Server (HTTP mode) on http://127.0.0.1:9080/mcp
echo.
echo Available tools: ping, mock_search, mock_fetch, mock_calculate, mock_echo, mock_list_files
echo.
go run main.go --mode http --host 127.0.0.1 --port 9080