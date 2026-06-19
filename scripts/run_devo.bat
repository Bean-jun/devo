set DEVO_LLM_BASE_URL=https://api.deepseek.com
@REM set DEVO_LLM_API_KEY=sk-xxx
set DEVO_LLM_MODEL=deepseek-v4-flash

set DEVO_DB_PATH=./.env/devo.db
set DEVO_LOG_PATH=./.env/devo.log

go build -o devo.exe cmd\devo\main.go

devo.exe --tui
