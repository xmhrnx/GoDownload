@echo off
rem Builds GoDownload for Windows into dist\windows\.
rem
rem With the giode CLI installed (recommended: it embeds the icon, registers
rem magnet: links and .torrent files, and can make an installer) this runs
rem "giode build windows". Without it, a plain go build with the same size
rem settings as giode.toml.
setlocal EnableExtensions
cd /d "%~dp0"

where giode >nul 2>nul
if not errorlevel 1 (
  giode build windows
  exit /b %errorlevel%
)

where go >nul 2>nul
if errorlevel 1 (
  echo ERROR: Go isn't installed. Get it from https://go.dev/dl/ 1>&2
  exit /b 1
)
echo giode CLI not found; building with go build.
echo For the app icon and an installer: go install github.com/xmhrnx/giode-cli/cmd/giode@latest
if not exist dist\windows mkdir dist\windows
rem nosqlite: the torrent library would compile in SQLite, which GoDownload
rem doesn't use. -s -w drop debug symbols. -H windowsgui: no console window.
go build -trimpath -tags nosqlite -ldflags "-s -w -H windowsgui" -o dist\windows\GoDownload.exe .
if errorlevel 1 exit /b 1
echo Built dist\windows\GoDownload.exe
