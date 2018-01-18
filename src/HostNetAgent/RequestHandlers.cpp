/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    RequestHandlers.cpp

Abstract:

    Implementation to Rest handler functions and helpers.


Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//


#include "RequestHandlers.h"
#include "HostNetAgent.h"

using namespace http;


void
RequestHandlers::GetPackageVersion(
    __inout http_request Request
)
/*++

Description:

    Function to process get package version query.

Arguments:

    Request - The http request.

Return Value:

    None

--*/
{
    wstring packageVersion;
    PackagesInfo* packagesInfoInstance = PackagesInfo::GetInstance();
    if (packagesInfoInstance == NULL)
    {
        LOGERRW(L"Failed to allocate memory for packagesInfo.");
        Request.reply(status_codes::InternalError);
        return;
    }

    packageVersion = packagesInfoInstance->GetPackageVersion();
    Request.reply(status_codes::OK, packageVersion);
    LOGINFOW(L"Version response request:{%s}. Version:{%s}", 
             Request.to_string().c_str(), 
             packageVersion.c_str());
}

void
RequestHandlers::GetPackageUris(
    __inout http_request Request
)
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
    PackagesInfo* packagesInfoInstance = PackagesInfo::GetInstance();
    if (packagesInfoInstance == NULL)
    {
        LOGERRW(L"Failed to allocate memory for packagesInfo.");
        Request.reply(status_codes::InternalError);
        return;
    }

    vector<wstring> packageNames = packagesInfoInstance->GetPackageNames();
    for (vector<wstring>::iterator it = packageNames.begin(); it != packageNames.end(); ++it) {

        packageMap[it->c_str()] = json::value::string(L"");
        LOGINFOW(L"Adding packageName:{%s} to the response", it->c_str());
    }

    Request.reply(status_codes::OK, packageMap);
    LOGINFOW(L"Response Request:{%s}", Request.to_string().c_str());
}

HRESULT
RequestHandlers::ExtractContainerIdAndRequestType(
    uri Uri,
    wstring& ContainerId,
    wstring& RequestType
)
/*++

Description:

    Function to extract the container Id and the request type from the URI.

Arguments:

    Uri - URI associated with the request.

<<<<<<< HEAD
    ContainerId = Place holder for the container Id extracted from the URI.

    RequestType - Plcae holder for the requesttype key extracted from the URI 
=======
    ContainerId = Place holder for the container Id extracted from the URI.

    RequestType - Plcae holder for the requesttype key extracted from the URI 
>>>>>>> VSO:Formatting changes

Return Value:

    None

--*/
{
    HRESULT hr = S_OK;
    bool validUri = false;
    bool foundContainerSegment = false;
    vector<wstring> pathFragments = uri::split_path(Uri.path());

    for (vector<utility::string_t>::iterator it = pathFragments.begin();
         it != pathFragments.end();
         ++it) {

        if (_wcsicmp(CONTAINER_SEGMENT, it->c_str()) == 0) {

            ++it;
            if (it == pathFragments.end()) {

                HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
                goto Cleanup;
            }

            ContainerId.assign(it->c_str());
            foundContainerSegment = true;
        }

        if (!foundContainerSegment) {

            continue;
        }

        ++it;
        if (it == pathFragments.end()) {

            hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
            goto Cleanup;
        }

        RequestType.assign(it->c_str());

        ++it;
        if (it != pathFragments.end()) {

            hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
            goto Cleanup;
        }

        validUri = true;
        break;
    }

    if (!validUri) {

        hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
    }

Cleanup:

    if (hr != S_OK) {

        ContainerId.clear();
        RequestType.clear();
    }

    return hr;
}

void
RequestHandlers::ProcessPost(
    __inout http_request Request
)
/*++

Description:

    Handler to process post request.

Arguments:

    Request - The http request.

Return Value:

    None

--*/
{
    wstring containerId;
    wstring requestType;
    HRESULT hr = S_OK;
    status_code statusCode = status_codes::OK;
    json::value jsonBody;
    MdmLogger* mdmLogger;

    LOGINFOW(L"Received request:{%s}", Request.to_string().c_str());

    mdmLogger = MdmLogger::GetInstance();
    if (mdmLogger == NULL) {

        LOGERRW(L"Failed to allocate memory for mdm Logger");
        goto Cleanup;
    }

    hr = ExtractContainerIdAndRequestType(Request.absolute_uri(),
                                          containerId,
                                          requestType);
    if (hr != S_OK) {

        LOGERRW(L"Invalid URI received in the request");
        statusCode = status_codes::BadRequest;
        goto Cleanup;
    }

    LOGINFOW(L"Retrieved ContainerId:{%s}", containerId.c_str());
    LOGINFOW(L"retrieved RequestType:{%s}", requestType.c_str());



    jsonBody = Request.extract_json().get();
    if (jsonBody.is_null() || jsonBody.size() == 0 || !jsonBody.is_object()) {

        LOGERRW(L"Invalid Json body");
        statusCode = status_codes::BadRequest;
        goto Cleanup;
    }

    if (_wcsicmp(POST_HEALTH_REPORT_KEY.c_str(), requestType.c_str()) == 0) {
        try {

            HealthReport healthReport;
            healthReport.InitializeHealthReport(containerId, jsonBody.as_object());
            mdmLogger->LogHealthReport(healthReport);
        }
        catch (const exception& e) {

            LOGERR("Failed to process health report with exception :{%s}", e.what());
            statusCode = status_codes::BadRequest;
            goto Cleanup;
        }
    }
    else if (_wcsicmp(POST_CNI_REPORT_KEY.c_str(), requestType.c_str()) == 0) {
        try {

            CNIReport cniReport;
            cniReport.InitializeCNIReport(jsonBody.as_object());
            cniReport.PrintCNIReport();
            mdmLogger->LogCNIReport(containerId, cniReport);
        }
        catch (const exception& e) {

            LOGERR("Failed to process CNI report with exception :{%s}", e.what());
            statusCode = status_codes::BadRequest;
            goto Cleanup;
        }
    }
    else
    {
        LOGERRW(L"Received request type {%s} which is invalid", requestType.c_str());
        statusCode = status_codes::BadRequest;
        goto Cleanup;
    }


Cleanup:

    Request.reply(statusCode, json::value());
    LOGINFOW(L"Completed Processing of the request with status code [%d]", statusCode);
}

void
RequestHandlers::ProcessGet(
    __inout http_request Request
)
/*++

Description:

    Handler to process get request.

Arguments:

    Request - The http request.

Return Value:

    None

--*/
{
    MdmLogger* mdmLogger = MdmLogger::GetInstance();
    PackagesInfo* packagesInfoInstance;

    mdmLogger = MdmLogger::GetInstance();
    if (mdmLogger == NULL) {

        LOGERRW(L"Failed to allocate memory for mdm Logger");
        return;
    }

    packagesInfoInstance = PackagesInfo::GetInstance();
    if (packagesInfoInstance == NULL)
    {
        LOGERRW(L"Failed to allocate memory for packagesInfo.");
        return;
    }

    LOGINFOW(L"Processing get request:[%s]", Request.to_string().c_str());
    wstring containerId;
    // Parse the query associated with the request.
    auto query = http::uri::split_query(http::uri::decode(Request.relative_uri().query()));

    LOGINFOW(L"Query Count:%I64u", query.size());
    for (map<utility::string_t,
         utility::string_t>::iterator it = query.begin(); it != query.end();
         ++it) {

        LOGINFOW(L"Query Key:[%s], Value:[%s]", it->first.c_str(), it->second.c_str());
    }

    // Respond with bad request if the query is missing type.
    auto found_name = query.find(TYPE_KEY);
    if (found_name == end(query)) {

        LOGERRW(L"Unsupported Query");
        Request.reply(status_codes::BadRequest);
        return;
    }

    // Respond with bad request if the query is missing type.
    auto containerIdIter = query.find(TYPE_CONTAINER);
    if (containerIdIter == end(query)) {

        LOGERRW(L"Container Id missing in Query");
        Request.reply(status_codes::BadRequest);
        return;
    }
    containerId = containerIdIter->second;

    if (_wcsnicmp(found_name->second.c_str(),
                  GET_VERSION_QUERY_KEY.c_str(),
                  GET_VERSION_QUERY_KEY.length()) == 0) {

        return GetPackageVersion(Request);
    }

    if (_wcsnicmp(found_name->second.c_str(),
                  GET_PACKAGE_URIS_QUERY_KEY.c_str(),
                  GET_PACKAGE_URIS_QUERY_KEY.length()) == 0) {

        return GetPackageUris(Request);
    }

    mdmLogger->LogDownloadMetricToMDM(
        containerId.c_str(),
        1);

    // Assume this is a request for an explicit resource by name.
    // Check the package cache to check if the package exist.
    vector<wstring> packageNames = packagesInfoInstance->GetPackageNames();
    auto fileFound = find(packageNames.begin(),
                          packageNames.end(),
                          found_name->second.c_str());
    if (fileFound == packageNames.end()) {

        LOGERRW(L"Requested resource {%s} not found", found_name->second.c_str());
        Request.reply(status_codes::NotFound);
        return;
    }

    LOGERRW(L"Resource {%s} is found in cache.", found_name->second.c_str());
    ifstream ifs;
    wstring fileName = packagesInfoInstance->GetPackageLocation() + 
                       L"\\" + 
                       found_name->second.c_str();

    // Check if the resource actually exist by trying to open an handle to the package.
    ifs.open(fileName, ifstream::in);
    if (ifs.good()) {

        pplx::task<void> requestTask = file_stream<unsigned char>::open_istream(fileName.c_str(), ios_base::binary)
        .then([=](pplx::task<concurrency::streams::basic_istream<unsigned char>> previousTask) {
            auto fileStream = previousTask.get();
            auto length = fileStream.seek(0, ios_base::end);
            fileStream.seek(0);
            Request.reply(status_codes::OK, fileStream, length);
        });
        ifs.close();

        LOGINFOW(L"Requested resource {%s} successfully transmited", found_name->second.c_str());
    } else {

        LOGERRW(L"Requested resource {%s} is not found on the disk", found_name->second.c_str());
        Request.reply(status_codes::NotFound);
    }
}