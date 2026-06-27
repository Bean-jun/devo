taskkill /f /im node.exe
taskkill /f /im mock_server.exe
taskkill /f /im devo.exe

start cmd /c run_mock_server.bat
start cmd /c run_devo_mock.bat
timeout /t 5
start cmd /c run_devo_frontend.bat