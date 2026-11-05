@echo off
setlocal
title Install openfolder: protocol

rem Double-click wrapper for install-openfolder.ps1.
rem Run as a normal user - NOT "Run as administrator".
rem Pass -Uninstall to remove:  install-openfolder.cmd -Uninstall

set "PS1=%~dp0install-openfolder.ps1"

if not exist "%PS1%" (
    echo.
    echo ERROR: install-openfolder.ps1 not found.
    echo Keep all files of the client\windows folder together and try again.
    echo.
    pause
    exit /b 1
)

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%PS1%" %*
set "RC=%ERRORLEVEL%"
echo.

if /i "%~1"=="-Uninstall" goto :done

reg query "HKCU\Software\Classes\openfolder\shell\open\command" >nul 2>&1
if errorlevel 1 set "RC=1"

if not "%RC%"=="0" (
    echo ============================================================
    echo  ERROR: Installation failed. See the messages above.
    echo ============================================================
) else (
    echo ============================================================
    echo  DONE. openfolder: links now open in Windows Explorer.
    echo  On the first click, allow the link to open the app.
    echo ============================================================
)

:done
echo.
pause
exit /b %RC%
