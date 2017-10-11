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

STATIC 
void 
GetPackageVersion(
    __inout http_request Request)
/*++

Description:

    Function to process get package version query.

Arguments:

    Request - The http request.

Return Value:

    None

--*/
{
    std::wstring packageVersion = packagesInfo->GetPackageVersion();
    Request.reply(status_codes::OK, packageVersion);
    LOGINFOW(L"Version response request:{%s}", Request.to_string().c_str());
}

STATIC 
void 
GetPackageUris(
    __inout http_request Request)
/*++

Description:

    Function to process get package Uri's query.

Arguments:

    Request - The http request.

Return Value:

    None

--*/
{
    json::value packageMap;
    std::vector<std::wstring> packageNames = packagesInfo->GetPackageNames();
    for (std::vector<std::wstring>::iterator it = packageNames.begin(); it != packageNames.end(); ++it)
    {
        packageMap[it->c_str()] = json::value::string(L"");
    }
    Request.reply(status_codes::OK, packageMap);
    LOGINFOW(L"Response Request:{%s}", Request.to_string().c_str());
}

STATIC 
void 
ProcessGet(
    __inout http_request Request)
/*++

Description:

    Handler to process get request.

Arguments:

    Request - The http request.

Return Value:

    None

--*/
{
    LOGINFOW(L"Processing get request:[%s]", Request.to_string().c_str());

    // Parse the query associated with the request.
    auto query = http::uri::split_query(http::uri::decode(Request.relative_uri().query()));

    LOGINFOW(L"Query Count:%I64u", query.size());
    for (
        std::map<utility::string_t, 
        utility::string_t>::iterator it = query.begin(); it != query.end(); 
        ++it)
    {
        LOGINFOW(L"Query Key:[%s], Value:[%s]", it->first.c_str(),it->second.c_str());
    }

    // Respond with bad request if the query is missing type.
    auto found_name = query.find(TYPE_KEY);
    if (found_name == end(query))
    {
        LOGERRW(L"Unsupported Query");
        Request.reply(status_codes::BadRequest);
        return;
    }

    if (_wcsnicmp(
        found_name->second.c_str(), 
        GET_VERSION_QUERY_KEY.c_str(), 
        GET_VERSION_QUERY_KEY.length()) == 0)
    {
        return GetPackageVersion(Request);
    }

    if (_wcsnicmp(
        found_name->second.c_str(), 
        GET_PACKAGE_URIS_QUERY_KEY.c_str(), 
        GET_PACKAGE_URIS_QUERY_KEY.length()) == 0)
    {
        return GetPackageUris(Request);
    }
    
    // Assume this is a request for an explicit resource by name.
    // Check the package cache to check if the package exist.
    std::vector<std::wstring> packageNames = packagesInfo->GetPackageNames();
    auto fileFound = find(
        packageNames.begin(), 
        packageNames.end(), 
        found_name->second.c_str());
    if (fileFound == packageNames.end())
    {
        LOGERRW(L"Requested resource {%s} not found", found_name->second.c_str());
        Request.reply(status_codes::NotFound);
        return;
    }

    LOGERRW(L"Resource {%s} is found in cache.", found_name->second.c_str());
    std::ifstream ifs;
    std::wstring fileName = packagesInfo->GetPackageLocation() + L"\\" + found_name->second.c_str();

    // Check if the resource actually exist by trying to open an handle to the package.
    ifs.open(fileName, std::ifstream::in);
    if (ifs.good())
    {
        pplx::task<void> requestTask = file_stream<unsigned char>::open_istream(fileName.c_str(), std::ios_base::binary)
            .then([=](pplx::task<concurrency::streams::basic_istream<unsigned char>> previousTask)
        {
            auto fileStream = previousTask.get();
            auto length = fileStream.seek(0, std::ios_base::end);
            fileStream.seek(0);
            Request.reply(status_codes::OK, fileStream, length);
        });
        ifs.close();
    }
    else
    {
        LOGERRW(L"Requested resource {%s} is not found on the disk", found_name->second.c_str());
        Request.reply(status_codes::NotFound);
    }
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
    listener.support(methods::GET, ProcessGet);
    pplx::task_status ret = listener.open().then([&listener]() {LOGINFOW(L"BEGIN LISTEN"); }).wait();

    if (ret != pplx::completed)
    {
        LOGERRW(L"Unable to open the listener. Ret = %d", ret);
        return;
    }
    else
    {
        LOGINFOW(L"Successfully listening on port specified");
    }

    while (true) 
    { 
        Sleep(5000); 
    }
}

bool 
SetupLogging(
    __in PCWSTR LogLocation)
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
    if (CreateDirectory(LogLocation, NULL) == 0)
    {
        err = GetLastError();
        if (err != ERROR_ALREADY_EXISTS)
        {
            wprintf(L"Failed to set up logging directory. Error:[%0x]", err);
            return false;
        }
    }

    LOGSETUP(LogLocation, L"HostNetAgent");
    return true;
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

    if (FALSE == len)
    {
        return GetLastError();
    }

    executableFullPathFileName = std::wstring(fileName, len);

    index = executableFullPathFileName.rfind(DIRECTORY_SEPARATOR);

    if (index == std::wstring::npos)
    {
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
    wchar_t* Argv[])
/*++

Description:

    Enrty point.

Arguments:

    Argc - Argument count.
    
    Argv - Array of arguments.

Return Value:

    Non zero on failure.

--*/
{
    wstring logLocation;
    wstring packageLocation;
    DWORD err = ERROR_SUCCESS;

    if (Argc == 3)
    {
        logLocation.assign(Argv[1]);
        packageLocation.assign(Argv[2]);
    }
    else if (Argc == 1)
    {
        logLocation.assign(DEFAULT_LOG_LOCATION);
        err = GetPackageLocation(packageLocation);
        if (err != ERROR_SUCCESS)
        {
            wprintf(L"Failed to retrieve package location. Error[%0x]", err);
            return err;
        }
    }
    else
    {
        wprintf(L"Invalid argument count. Argument count should be none or two. (Log location and package location)");
        return -1;
    }

    wprintf(L"Using package location:[%s] Log location [%s]\n", packageLocation.c_str(), logLocation.c_str());

    if(!SetupLogging(logLocation.c_str()))
    {
        wprintf(L"Failed to setup logging.");
        return - 1;
    }

    packagesInfo = new PackagesInfo();
    err = packagesInfo->Initialize(packageLocation);
    if (err != ERROR_SUCCESS)
    {
        return err;
    }

    const wstring listenerUri = GetListenerUri();
    std::thread listenerThread(StartListener);
    listenerThread.join();
}