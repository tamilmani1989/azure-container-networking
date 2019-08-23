@echo off
set BuildConfig="%1"
pushd "%~dp0"

set Build=false

if %BuildConfig%=="Debug" set Build=true
if %BuildConfig%=="Release" set Build=true

if %Build%==true (
    echo ---------- Build DNCService project ----------
	dotnet build %~dp0\src\Dnc-SF\DNCService\DNCService.csproj /p:Configuration=%BuildConfig%

    rem Check exit code and return if necessary
    if "%ERRORLEVEL%" neq "0" (
        popd
        echo "Failed to build DNCService project."
        exit /b %ERRORLEVEL%
    )
	
	echo ---------- Prepare DNC SF Application ----------
    nuget restore %~dp0\src\Dnc-SF\DncSFProj\packages.config -PackagesDirectory %~dp0\src\Dnc-SF\DncSFProj\packages
    dotnet build %~dp0\src\Dnc-SF\DncSFProj\DncApplication.sfproj /p:Configuration=%BuildConfig% /t:Package
	
	rem Check exit code and return if necessary
    if "%ERRORLEVEL%" neq "0" (
        popd
        echo "Failed to build DncApplication SF project."
        exit /b %ERRORLEVEL%
    )
)

rem Restore working directory
popd

rem Exit

exit /b %ERRORLEVEL%
