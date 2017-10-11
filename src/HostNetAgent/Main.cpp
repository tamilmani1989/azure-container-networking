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
    __in wchar_t* LogLocation)
/*++

Description:

    Helper to set up logging.

Arguments:

    LogLocation - Location of the log..

Return Value:

    True if setup succeeded, False otherwise.

--*/
{
    if (CreateDirectory(LogLocation, NULL) == 0 &&
        ERROR_ALREADY_EXISTS != GetLastError())
    {
        return false;
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
    if ((Argc != 3))
    {
        return -1;
    }

    wchar_t* logLocation = Argv[1];
    wchar_t* packageLocation = Argv[2];
    DWORD err;
    std::wstring packageLoc(packageLocation);
    
    if(!SetupLogging(logLocation))
    {
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