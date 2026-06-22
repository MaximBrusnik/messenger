@echo off
cd /d "%~dp0"

echo Building for Windows...
go env -w GOOS=windows GOARCH=amd64
go build -o messenger.exe cmd/main.go
echo OK: messenger.exe

echo Building for Linux...
go env -w GOOS=linux GOARCH=amd64
go build -o messenger-linux cmd/main.go
echo OK: messenger-linux

:: Reset to Windows
go env -w GOOS=windows GOARCH=amd64

echo Done.
