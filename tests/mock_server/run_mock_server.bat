@echo off
echo ============================================
echo   Mock LLM Server
echo   Listening on: http://localhost:8080
echo   Requests saved to: requests\
echo ============================================
echo.
go mod tidy
go build -o mock_server.exe
.\mock_server.exe