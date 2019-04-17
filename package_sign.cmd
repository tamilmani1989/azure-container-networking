set OPERATION="%1"
set DROP=%2
set DIR="%~dp0"
set SERVICE=%3
set GITDROP=%4
cd /d %DIR%

if %OPERATION%=="package" (
    nuget pack Azure%SERVICE%Package.nuspec -Properties branch=%BUILD_SOURCEBRANCHNAME%;drop="%~dp0%DROP%";gitdrop="%~dp0%GITDROP%";dir=%DIR%;version=%CDP_MAJOR_NUMBER_ONLY%.%CDP_MINOR_NUMBER_ONLY%.%CDP_VERSION_TAG_ONLY%
) else if %OPERATION%=="sign" (
    robocopy %CDP_TEMP_PRIOR_DROP_FOLDER_CONTAINER_PATH%\current\drop\linux_phase\outputs\build\%DROP% %~dp0%DROP% /e
)

if %ERRORLEVEL% LSS 8 (
    exit /b 0
) else (
    exit /b %ERRORLEVEL%
)