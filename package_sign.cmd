set DROP="%2"
set DIR="%~dp0"
cd /d %DIR%

if "%1"=="package" (
    nuget pack "AzureDNCPackage.nuspec" -Properties branch=%BUILD_SOURCEBRANCHNAME%;drop=%~dp0\%DROP%;dir=%DIR%;version=%CDP_MAJOR_NUMBER_ONLY%.%CDP_MINOR_NUMBER_ONLY%.%CDP_BUILD_NUMBER%
) else if "%1"=="sign" (
    robocopy %CDP_TEMP_PRIOR_DROP_FOLDER_CONTAINER_PATH%\current\drop\linux_phase\outputs\build\drop %~dp0\%DROP% /e
)

if %ERRORLEVEL% LSS 8 (
    exit /b 0
) else (
    exit /b %ERRORLEVEL%
)