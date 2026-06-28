@echo off
set DEVO_LLM_BASE_URL=http://localhost:8080/v1
set DEVO_LLM_API_KEY=sk-mock-key
set DEVO_LLM_MODEL=mock-model

set DEVO_DB_PATH=./.env/devo_mock.db
set DEVO_LOG_PATH=./.env/devo_mock.log

echo ============================================
echo   Devo + Mock Server
echo   LLM Base URL: %DEVO_LLM_BASE_URL%
echo   Model: %DEVO_LLM_MODEL%
echo ============================================
echo.

cd ..\..\

taskkill /f /im devo.exe
go build -o build\devo.exe ./cmd/devo/
.\build\devo.exe -port=8081