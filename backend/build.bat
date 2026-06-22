@echo off
cd /d "%~dp0"

:: 1. Build frontend
echo [1/4] Building frontend...
cd /d "%~dp0..\frontend\frontend"
call npx vite build
if %errorlevel% neq 0 (
    echo Frontend build failed!
    exit /b %errorlevel%
)
echo OK

:: 2. Copy dist to backend
echo [2/4] Copying frontend dist...
cd /d "%~dp0"
if exist "dist\assets" rmdir /s /q "dist\assets"
mkdir dist\assets >nul 2>&1
copy /y "..\frontend\frontend\dist\assets\*" "dist\assets\" >nul
copy /y "..\frontend\frontend\dist\index.html" "dist\" >nul
copy /y "..\frontend\frontend\dist\favicon.svg" "dist\" >nul
copy /y "..\frontend\frontend\dist\icons.svg" "dist\" >nul
echo OK

:: 3. Build Windows binary
echo [3/4] Building messenger.exe...
go env -w GOOS=windows GOARCH=amd64
go build -o messenger.exe cmd/main.go
echo OK

:: 4. Build Linux binary
echo [4/4] Building messenger-linux...
go env -w GOOS=linux GOARCH=amd64
go build -o messenger-linux cmd/main.go
echo OK

:: Reset to Windows
go env -w GOOS=windows GOARCH=amd64

echo Done.
