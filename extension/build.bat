@echo off
rem Builds the GoDownload browser extension on Windows, with only what
rem Windows 10 and 11 include (cmd, robocopy, PowerShell). Same output as
rem build.sh:
rem
rem   build.bat            build chrome and firefox
rem   build.bat chrome     build one target
rem
rem Output: dist\<target>\ (unpacked, loadable) and dist\godownload-<target>-<version>.zip
setlocal EnableExtensions

set "ROOT=%~dp0"
if "%ROOT:~-1%"=="\" set "ROOT=%ROOT:~0,-1%"
set "SRC=%ROOT%\src"
set "ICONS=%ROOT%\icons"
set "DIST=%ROOT%\dist"

if not exist "%SRC%\" (
  echo ERROR: source folder "%SRC%" not found. 1>&2
  exit /b 1
)
if not exist "%DIST%\" mkdir "%DIST%"

if "%~1"=="" (
  call :build chrome || exit /b 1
  call :build firefox || exit /b 1
) else (
  for %%T in (%*) do (
    call :build %%T || exit /b 1
  )
)
echo Done.
exit /b 0

:build
set "TARGET=%~1"
set "MANIFEST=%ROOT%\manifests\%TARGET%.json"
set "OUT=%DIST%\%TARGET%"
if not exist "%MANIFEST%" (
  echo ERROR: unknown target "%TARGET%" ^(no %MANIFEST%^) 1>&2
  exit /b 1
)
echo Building %TARGET%...
if exist "%OUT%\" rmdir /s /q "%OUT%"
mkdir "%OUT%"

rem robocopy exit codes below 8 mean success.
robocopy "%SRC%" "%OUT%" /E /NFL /NDL /NJH /NJS /NP /XF .DS_Store *~ *.swp >nul
if errorlevel 8 (
  echo ERROR: copying sources failed. 1>&2
  exit /b 1
)
copy /y "%MANIFEST%" "%OUT%\manifest.json" >nul || exit /b 1

mkdir "%OUT%\icons"
set "MISSING="
for %%S in (16 32 48 128) do (
  if exist "%ICONS%\icon-%%S.png" (
    copy /y "%ICONS%\icon-%%S.png" "%OUT%\icons\" >nul
  ) else (
    call set "MISSING=%%MISSING%% icon-%%S.png"
  )
)
if defined MISSING (
  echo WARNING: missing icons in %ICONS%:%MISSING% 1>&2
  rem Chrome refuses an extension whose manifest names a missing icon, so
  rem drop the icon keys from this build to keep it loadable.
  powershell -NoProfile -ExecutionPolicy Bypass -Command ^
    "$p = $env:OUT + '\manifest.json';" ^
    "$m = Get-Content -Raw -LiteralPath $p | ConvertFrom-Json;" ^
    "$m.PSObject.Properties.Remove('icons');" ^
    "if ($m.action) { $m.action.PSObject.Properties.Remove('default_icon') };" ^
    "[IO.File]::WriteAllText($p, ($m | ConvertTo-Json -Depth 32))" || exit /b 1
  echo WARNING: removed icon references from %TARGET%\manifest.json 1>&2
)

rem Zip with forward-slash entry names: Firefox rejects zips whose paths use
rem backslashes, which older Compress-Archive versions write.
powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$ErrorActionPreference = 'Stop';" ^
  "Add-Type -AssemblyName System.IO.Compression, System.IO.Compression.FileSystem;" ^
  "$v = (Get-Content -Raw -LiteralPath $env:MANIFEST | ConvertFrom-Json).version; if (-not $v) { $v = 'dev' };" ^
  "$zip = Join-Path $env:DIST ('godownload-' + $env:TARGET + '-' + $v + '.zip');" ^
  "if (Test-Path -LiteralPath $zip) { Remove-Item -LiteralPath $zip };" ^
  "$base = (Resolve-Path -LiteralPath $env:OUT).Path.TrimEnd('\') + '\';" ^
  "$a = [IO.Compression.ZipFile]::Open($zip, 'Create');" ^
  "try { Get-ChildItem -LiteralPath $env:OUT -Recurse -File | Sort-Object FullName | ForEach-Object {" ^
  "  $name = $_.FullName.Substring($base.Length).Replace('\', '/');" ^
  "  [void][IO.Compression.ZipFileExtensions]::CreateEntryFromFile($a, $_.FullName, $name, 'Optimal') } }" ^
  "finally { $a.Dispose() };" ^
  "Write-Host ('  packed ' + (Split-Path -Leaf $zip))" || exit /b 1
echo   unpacked: %OUT%
exit /b 0
