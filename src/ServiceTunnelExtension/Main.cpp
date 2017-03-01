/*++

    Copyright (c) 2008  Microsoft Corporation

Module Name:

    Main.cpp

Abstract:

    Entry point for the service tunnel extension 

    Performs the following

        1. Reads the latest RCF file and checks if service tunnel is enabled for any endpoint and 
           retrieves the IPV6 address to be used for tunnelling. If true performs steps 2-5

        2. Checks for an existense of loopback adapter with the name ServiceTunnel adapter and creates
           if one does not exist.

        3. Adds the IPV6 address retrieved from RCF file.

        4. Enables weak host routing

        5. Adds fake gateway and fake gateway macs.

--*/

//
// ------------------------------------------------------------------- Includes
//

#include "ServiceTunnelExtension.h"


HRESULT SetupLogging(
    )
/*++

Routine Description:

    Helper to set up logging.

Arguments:

    None.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    DWORD error;
    HRESULT hr;
    wchar_t executablePath[MAX_PATH];

    LPWSTR executableFolder = NULL;
    LPWSTR logFolder = NULL;
    LPWSTR logFilePathWithPrefix = NULL;
    if (!GetModuleFileName(NULL, executablePath, MAX_PATH))
    {
        error = GetLastError();
        wprintf(L"Failed to retrieve process working directory (%d)\n", error);
        RD_E_WIN32(error, Cleanup);
    }

    hr = GetDirectoryFromPath(executablePath, &executableFolder);

    wprintf(L"Retrieved executable folder[%s] [%0x]", executableFolder, hr);

    RD_E_HR(hr, Cleanup);

    error = AppendStrings(&logFolder,
        2,
        executableFolder,
        L"\\Logs\\");


    wprintf(L"Creating log folder[%s] [%d]", logFolder, error);

    RD_E_WIN32(error, Cleanup);
    
    if (!CreateDirectory(logFolder, NULL))
    {
        error = GetLastError();
        if (error != ERROR_ALREADY_EXISTS)
        {
            wprintf(L"Failed to create log folder [%s] due to erroro (%d)\n", logFolder, error);
            RD_E_WIN32(error, Cleanup);
        }
    }

    error = AppendStrings(&logFilePathWithPrefix,
        2,
        logFolder,
        LOG_FILE_PREFIX);
    RD_E_WIN32(error, Cleanup);

    hr = RD_TRACE_INITIALIZE_CUSTOM_SIZE(RD_TRACE_OUTPUT_FILE,
        RD_TRACE_ALL,
        logFilePathWithPrefix,
        NULL,
        0,
        NULL,
        RD_TRACE_ALL,
        10,
        RD_TRACE_LOG_LIMIT);

Cleanup:
    
    STR_FREE(executableFolder);
    STR_FREE(logFolder);
    STR_FREE(logFilePathWithPrefix);

    return hr;
}

int wmain(
    int argc, 
    wchar_t* argv[]
)
/*++

Routine Description:

    Entry point for the service tunnel extension 

Arguments:

    None.

Return Value:

    0 on success and non zero value of failure.

--*/
{
    HRESULT hr = S_OK;
    LPWSTR rcfFile = NULL;
    RcfEnumerator rcfEnumerator;
    XmlParserHelper xmlParserHelper;
    std::set<std::wstring> attributeValues;
    bool comInitialized = false;
    bool traceInitialized = false;
    std::set<wstring>::iterator it;
    ServiceTunnelConfigurator serviceTunnelConfigurator;

    hr = SetupLogging();
    if (FAILED(hr))
    {
        wprintf(L"Failed to setup logging due to error [%0x]", hr);
        goto Cleanup;
    }

    traceInitialized = true;

    hr = ::CoInitialize(NULL);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"ComInitialization failed with error [%0x].",
        hr),
        hr,
        Cleanup);

    comInitialized = true;

    hr = rcfEnumerator.FindLatestRcf(&rcfFile);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"FindLatestRcf failed with error [%0x].",
        hr),
        hr,
        Cleanup);

    hr = xmlParserHelper.Initialize(rcfFile);

    xmlParserHelper.QueryNodeAttributeValues(ENDPOINT_NODE_NAME, SERVICE_TUNNEL_IPV6_ATTRIBUTE_NAME, attributeValues);
    wprintf(L"Count:%d\n", attributeValues.size());


    for (it = attributeValues.begin(); it != attributeValues.end(); ++it)
    {
        wprintf(L"\tValue:%s\n", (*it).c_str());
    }

    serviceTunnelConfigurator.SetupLoopbackAdapter(attributeValues);

Cleanup:

    xmlParserHelper.UnInitialize();

    STR_FREE(rcfFile);

    if (comInitialized)
    {
        CoUninitialize();
    }

    if (traceInitialized)
    {
        RD_TRACE_TERMINATE();
    }

    STR_FREE(rcfFile);
}