set Operation="%1"
set BuildConfig=%2
set FolderLocation=%3
pushd "%~dp0"

if %Operation%=="Sign" (
    robocopy %~dp0src\Dnc-SF\DncSFProj\pkg\%BuildConfig% %~dp0%FolderLocation% /e

    rem Check exit code and return if necessary
    if "%ERRORLEVEL%" neq "0" (
        popd
        echo "Failed to sign DNCApplication."
        exit /b %ERRORLEVEL%
    )
) else if %Operation%=="Package" (
    rem Generate Nuget Package
    nuget pack "AzureDNCSFPackage.nuspec" -Properties branch=%BUILD_SOURCEBRANCHNAME%;drop="%~dp0%FolderLocation%";dir="%~dp0";version=%CDP_MAJOR_NUMBER_ONLY%.%CDP_MINOR_NUMBER_ONLY%.%CDP_VERSION_TAG_ONLY%

    rem Check exit code and return if necessary
    if "%ERRORLEVEL%" neq "0" (
        popd
        echo "Failed to package DNCApplication."
        exit /b %ERRORLEVEL%
    )
)

rem Restore working directory
popd

rem Exit
if %ERRORLEVEL% LSS 8 (
    exit /b 0
) else (
    exit /b %ERRORLEVEL%
)