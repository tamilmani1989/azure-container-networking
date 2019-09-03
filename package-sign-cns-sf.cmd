set Operation="%1"
set BuildConfig=%2
set FolderLocation=%3
pushd "%~dp0"

if %Operation%=="Sign" (
	robocopy %~dp0src\Cns-SF\pkg\%BuildConfig% %~dp0%FolderLocation% /e
	
    rem Check exit code and return if necessary
    if "%ERRORLEVEL%" neq "0" (
        popd
        echo "Failed to sign CNSApplication."
        exit /b %ERRORLEVEL%
    )
) else if %Operation%=="Package" (
    rem Generate Nuget Package
    nuget pack "AzureCNSSFPackage.nuspec" -Properties branch=%BUILD_SOURCEBRANCHNAME%;drop="%~dp0%FolderLocation%";dir="%~dp0";version=%CDP_MAJOR_NUMBER_ONLY%.%CDP_MINOR_NUMBER_ONLY%.%CDP_VERSION_TAG_ONLY%

    rem Check exit code and return if necessary
    if "%ERRORLEVEL%" neq "0" (
        popd
        echo "Failed to package CNSApplication."
        exit /b %ERRORLEVEL%
    )
) else (
    echo "Non supported operation! Supported operation is Sign or Package"
    exit /b -1
)

rem Restore working directory
popd

rem Exit
if %ERRORLEVEL% LSS 8 (
    exit /b 0
) else (
    exit /b %ERRORLEVEL%
)