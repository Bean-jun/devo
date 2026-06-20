set DEVO_LLM_BASE_URL=https://api.deepseek.com
@REM set DEVO_LLM_API_KEY=sk-xxx
set DEVO_LLM_MODEL=deepseek-v4-flash

set DEVO_DB_PATH=./.env/devo.db
set DEVO_LOG_PATH=./.env/devo.log


@REM cd web && npm install && npm run build && cd .. && go build -o devo.exe cmd\devo\main.go

@REM devo.exe -tui
.\build\devo.exe -web
