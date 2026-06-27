@echo off
echo ============================================
echo   Devo Frontend + Mock Server
echo   LLM Base URL: %DEVO_LLM_BASE_URL%
echo   Model: %DEVO_LLM_MODEL%
echo ============================================
echo.

cd ..\..\web\
npm run dev