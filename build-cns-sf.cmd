@echo off
set BuildConfig=%1
pushd "%~dp0"

set Build=false

if %BuildConfig%==Debug set Build=true
if %BuildConfig%==Release set Build=true

if %Build%==true (
    echo ---------- Build CnsWrapperService project ----------
	dotnet build %~dp0\src\Cns-SF\CnsWrapperService.csproj /p:Configuration="%BuildConfig%"

    rem Check exit code and return if necessary
    if "%ERRORLEVEL%" neq "0" (
        popd
        echo "Failed to build CnsWrapperService project."
        exit /b %ERRORLEVEL%
    )

	echo ---------- Prepare CnsWrapperService Application ----------
    nuget restore %~dp0\src\Cns-SF\packages.config -PackagesDirectory %~dp0\src\Cns-SF\packages
	
    robocopy %BUILD_SOURCESDIRECTORY%\artifacts\linux_phase\outputs\build\drop_github %~dp0\src\Cns-SF\bin\%BuildConfig%\net462 azure-cns.exe
    robocopy %BUILD_SOURCESDIRECTORY%\artifacts\linux_phase\outputs\build\drop_github %~dp0\src\Cns-SF\bin\%BuildConfig%\net462 azure-vnet-telemetry.exe
    robocopy %BUILD_SOURCESDIRECTORY%\artifacts\linux_phase\outputs\build\drop_github %~dp0\src\Cns-SF\bin\%BuildConfig%\net462 azure-vnet-telemetry.config
	
	rem Check exit code and return if necessary
    if "%ERRORLEVEL%" neq "0" (
        popd
        echo "Failed to build CnsWrapperApplication SF project."
        exit /b %ERRORLEVEL%
    )
	
    dotnet build %~dp0\src\Cns-SF\CnsWrapperApplication.sfproj /p:Configuration="%BuildConfig%" /t:Package
	
	rem Check exit code and return if necessary
    if "%ERRORLEVEL%" neq "0" (
        popd
        echo "Failed to build CnsWrapperApplication SF project."
        exit /b %ERRORLEVEL%
    )
)

rem Restore working directory
popd

rem Exit

exit /b %ERRORLEVEL%
