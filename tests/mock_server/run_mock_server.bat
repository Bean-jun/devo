@echo off
echo ============================================
echo   Mock LLM Server
echo   Listening on: http://localhost:8080
echo   Requests saved to: requests\
echo ============================================
echo.
taskkill /f /im mock_server.exe
go mod tidy
go build -o mock_server.exe
.\mock_server.exe