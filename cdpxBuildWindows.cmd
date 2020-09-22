@echo off
pushd "%~dp0"
echo "Running build powershell script"
powershell.exe -ExecutionPolicy Unrestricted -NoProfile -WindowStyle Hidden -File "%~dp0\cdpxBuildWindows.ps1" %1

if "%ERRORLEVEL%" neq "0" (
        popd
        echo "Build powershell script with parameters %1 failed."
        exit /b %ERRORLEVEL%
)

rem Restore working directory
popd

rem Exit
exit /B %ERRORLEVEL%