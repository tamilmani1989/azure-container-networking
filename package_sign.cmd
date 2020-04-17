set OPERATION="%1"
set DROP=%2
set DIR="%~dp0"
set SERVICE=%3
set GITDROP=%4
cd /d %DIR%

if %OPERATION%=="package" (

    :: if service is not DNC, then use version variable from above
    if %DROP%==drop_dnc (
        nuget pack Azure%SERVICE%Package.nuspec -Properties branch=%BUILD_SOURCEBRANCHNAME%;drop="%~dp0%DROP%";gitdrop="%~dp0%GITDROP%";dir=%DIR%;version=%CDP_MAJOR_NUMBER_ONLY%.%CDP_MINOR_NUMBER_ONLY%.%CDP_VERSION_TAG_ONLY%
    ) else (
        :: call azure-vnet.exe and save version, but then trim the 'v' character from the beginning
        setlocal enabledelayedexpansion
        for /f "tokens=4" %%v in ('%~dp0%DROP%\azure-vnet.exe -v') do (
            set fullversion=%%v
            set version=!fullversion:~1!
            echo !version!
            nuget pack Azure%SERVICE%Package.nuspec -Properties branch=%BUILD_SOURCEBRANCHNAME%;drop="%~dp0%DROP%";gitdrop="%~dp0%GITDROP%";dir=%DIR%;version=!version!
        )
        setlocal disabledelayedexpansion
    )

) else if %OPERATION%=="sign" (
    robocopy %CDP_TEMP_PRIOR_DROP_FOLDER_CONTAINER_PATH%\current\drop\linux_phase\outputs\build\%DROP% %~dp0%DROP% /e
)

if %ERRORLEVEL% LSS 8 (
    exit /b 0
) else (
    exit /b %ERRORLEVEL%
)