/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    Main.cpp

Abstract:

    Entry point into the HostNetAgent service.

Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//

#include "HostNetAgent.h"
#include "RequestHandlers.h"

DWORD
AgentParseCommandLine(
    __in ULONG ArgumentCount,
    __in_ecount(ArgumentCount) PWSTR *ArgumentList,
    __deref_out PWSTR *ConfigFile,
    __deref_out PWSTR *ResourcePath,
    __deref_out PWSTR *LogPath,
    __deref_out PWSTR *PackagePath
)
/*++

Routine Description:

    Parses command line arguments for parameters passed during invocation.

Arguments:

    ArgumentCount - Number of arguments in input string ArgumentList.

    ArgumentList - Command line argument list.

    ConfigFile - Output pointer to configuration file option if found in ArgumentList.

    ResourcePath - Output pointer to resource path option if found in ArgumentList.

    LogPath - Output pointer to log path option if found in ArgumentList.

    PackagePath - Output pointer to package path option if found in ArgumentList.

Return Value:

    S_OK on success. Otherwise appropriate HRESULT is returned.

--*/
{
    ULONG indx = 0;

    if ((NULL == ArgumentList) ||
        (NULL == ConfigFile) ||
        (NULL == ResourcePath) ||
        (NULL == LogPath) ||
        (NULL == ResourcePath)) {

        return ERROR_INVALID_DATA;
    }

    *ConfigFile = NULL;
    *ResourcePath = NULL;
    *LogPath = NULL;
    *PackagePath = NULL;

    //
    // Iterate through command line looking for the expected options
    //

    for (indx = 0; indx < ArgumentCount; indx++) {

        if ((_wcsicmp(ArgumentList[indx], CONFIG_FILE_CMDLINE_FLAG) == 0) &&
                (indx + 1 < ArgumentCount)) {
            indx++;
            *ConfigFile = ArgumentList[indx];

        } else if ((_wcsicmp(ArgumentList[indx], RESOURCE_PATH_CMDLINE_FLAG) == 0) &&
                   (indx + 1 < ArgumentCount)) {

            indx++;
            *ResourcePath = ArgumentList[indx];
        } else if ((_wcsicmp(ArgumentList[indx], LOG_PATH_CMDLINE_FLAG) == 0) &&
                   (indx + 1 < ArgumentCount)) {

            indx++;
            *LogPath = ArgumentList[indx];
        } else if ((_wcsicmp(ArgumentList[indx], PACKAGE_PATH_CMDLINE_FLAG) == 0) &&
                   (indx + 1 < ArgumentCount)) {

            indx++;
            *PackagePath = ArgumentList[indx];
        }
    }

    return ERROR_SUCCESS;
}

void
StartListener()
/*++

Description:

    Starts REST listener.

Arguments:

    None

Return Value:

    None

--*/
{
    std::wstring listenerUri = GetListenerUri();
    LOGINFOW(L"Starting listener Uri:[%s].", listenerUri.c_str());
    uri_builder uriBuilder(listenerUri);

    http_listener listener(uriBuilder.to_uri());

    listener.support(methods::GET, RequestHandlers::ProcessGet);
    listener.support(methods::POST, RequestHandlers::ProcessPost);
    
    pplx::task_status ret = listener.open().then([&listener]() {
        LOGINFOW(L"BEGIN LISTEN");
    }).wait();

    if (ret != pplx::completed) {

        LOGERRW(L"Unable to open the listener. Ret = %d", ret);
        return;
    } else {

        LOGINFOW(L"Successfully listening on port specified");
    }

    while (true) {
        Sleep(5000);
    }
}

bool
SetupLogging(
    __in PCWSTR LogLocation
)
/*++

Description:

    Helper to set up logging.

Arguments:

    LogLocation - Location of the log..

Return Value:

    True if setup succeeded, False otherwise.

--*/
{
    DWORD err;
    HRESULT hr = S_OK;

    if (CreateDirectory(LogLocation, NULL) == 0) {

        err = GetLastError();
        if (err != ERROR_ALREADY_EXISTS) {

            wprintf(L"Failed to set up logging directory. Error:[%0x]", err);
            return false;
        }
    }

    hr = LOGSETUP(LogLocation, L"HostNetAgent");

    return hr == S_OK? true : false;
}

const
std::wstring
GetListenerUri()
/*++

Description:

    Helper to get the listener Uri.

Arguments:

    None

Return Value:

    None

--*/
{
    return L"http://localhost:" + HOST_NETAGENT_HTTP_LISTEN_PORT + L"/" + RELATIVE_URI_PATH;
}

DWORD
GetPackageLocation(
    __inout std::wstring& PackageLocation
)
/*++

Description:

    Helper to get the package location.

Arguments:

    PackageLocation - Location to hold the retrieved package location.

Return Value:

    ERROR_SUCCESS on success, appropriate error on failures.

--*/
{
    std::wstring executableFullPathFileName;
    DWORD len = 0;
    WCHAR fileName[MAX_PATH] = { 0 };
    size_t index;

    len = GetModuleFileName(
              NULL,
              fileName,
              MAX_PATH
          );

    if (FALSE == len) {

        return GetLastError();
    }

    executableFullPathFileName = std::wstring(fileName, len);

    index = executableFullPathFileName.rfind(DIRECTORY_SEPARATOR);

    if (index == std::wstring::npos) {

        return ERROR_NO_MATCH;
    }

    PackageLocation.assign(executableFullPathFileName.substr(0, index));
    PackageLocation.append(L"\\");
    PackageLocation.append(PACKAGE_FOLDER_NAME);

    return ERROR_SUCCESS;
}

int
wmain(
    int Argc,
    PWSTR* Argv
)
/*++

Description:

    Entry point.

Arguments:

    Argc - Argument count.

    Argv - Array of arguments.

Return Value:

    Non zero on failure.

--*/
{
    PWSTR configFile;
    PWSTR resourcePath;
    PWSTR logPath;
    PWSTR packagePath;

    wstring logLocation;
    wstring packageLocation;
    DWORD err = ERROR_SUCCESS;
    MdmLogger* mdmLogger;
    HRESULT hr = S_OK;
	PackagesInfo* packagesInfoInstance;

    err = AgentParseCommandLine(Argc,
                                Argv, 
                                &configFile, 
                                &resourcePath, 
                                &logPath, 
                                &packagePath);
    if (err != ERROR_SUCCESS) {
        wprintf(L"Failed to parse command line arguments. Error[%0x]", err);
        return err;
    }

    logPath == NULL ? logLocation.assign(DEFAULT_LOG_LOCATION) : logLocation.assign(logPath);

    if (packagePath == NULL) {
        err = GetPackageLocation(packageLocation);
        if (err != ERROR_SUCCESS) {
            wprintf(L"Failed to generate package location. Error[%0x]", err);
            return err;
        }
    } else {
        packageLocation.assign(packagePath);
    }

    wprintf(L"Using package location:[%s] Log location [%s]\n", 
            packageLocation.c_str(), 
            logLocation.c_str());

    if (!SetupLogging(logLocation.c_str())) {
        wprintf(L"Failed to setup logging.");
        return -1;
    }

	packagesInfoInstance = PackagesInfo::GetInstance();
    if (packagesInfoInstance == NULL)
    {
        LOGERRW(L"Failed to allocate memory for packagesInfo.");
        return ERROR_OUTOFMEMORY;
    }

    err = packagesInfoInstance->Initialize(packageLocation);
    if (err != ERROR_SUCCESS) {

        LOGERRW(L"Failed to initialize packages info.");
        return err;
    }

    mdmLogger = MdmLogger::GetInstance();
    if(mdmLogger == NULL)
    {
        LOGERRW(L"Failed to allocate memory for mdm Logger");
        return ERROR_OUTOFMEMORY;
    }

    hr = mdmLogger->InitializeMdm(
        configFile,
		packagesInfoInstance->GetPackageVersion().c_str());
    if( hr != S_OK)
    {
        LOGERRW(L"Failed to set up Mdm using settings file {%s}", configFile);
        return hr;
    }

    HealthReport::InitializeHealthReportFieldIdMap();
	CNIReport::InitializeCNIReportFieldIdMap();

    std::thread listenerThread(StartListener);
    listenerThread.join();
}