@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

echo ============================================
echo   go_tui - Cross-Platform Build Script
echo ============================================
echo.

set "OUTPUT_DIR=build"

if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

echo [1/2] Building Windows 64-bit (go_tui-windows-amd64.exe)...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o "%OUTPUT_DIR%\go_tui-windows-amd64.exe" .
if !errorlevel! neq 0 (
    echo [FAIL] Windows build failed!
    exit /b 1
)
echo [ OK ] Windows build successful.

echo.

echo [2/2] Building Linux 64-bit (go_tui-linux-amd64)...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o "%OUTPUT_DIR%\go_tui-linux-amd64" .
if !errorlevel! neq 0 (
    echo [FAIL] Linux build failed!
    exit /b 1
)
echo [ OK ] Linux build successful.

echo.
echo ============================================
echo   Build Complete!
echo   Output: %OUTPUT_DIR%\
echo     go_tui-windows-amd64.exe
echo     go_tui-linux-amd64
echo ============================================

endlocal