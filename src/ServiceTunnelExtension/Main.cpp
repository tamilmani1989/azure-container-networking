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

HRESULT GetInstanceId(
    __in XmlParserHelper* XmlParserHelper,
    std::wstring& InstanceName)
/*++

Routine Description:

    Helper to retrieve the instance name from the RCF.

Arguments:

    XmlParserHelper - XmlParserHelepr object which has the RCF file loaded.

    InstanceName - Out paramter to hold the instance name retrieved from RCF.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    HRESULT hr = S_OK;
    std::set<std::wstring> instanceValues;
    std::wstring incarnationNodePath;
    incarnationNodePath.append(XPATH_ROOT_START_TAG);
    incarnationNodePath.append(INCARNATION_NODE_NAME);

    hr = xmlParserHelper->QueryNodeAttributeValues(
        (LPWSTR)incarnationNodePath.c_str(),
        INSTANCE_ATTRIBUTE_NAME,
        instanceValues);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"GetInstanceId failed with error [%0x]. NodeQuery:[%s], Attribute[%s]", hr, incarnationNodePath.c_str(), INSTANCE_ATTRIBUTE_NAME),
        hr,
        Cleanup);

    RD_TRACE_LOG((RD_TRACE_INFO,
        L"NodeQuery:%s, Attribute:%s, Count:%d.",
        incarnationNodePath.c_str(),
        INSTANCE_ATTRIBUTE_NAME,
        instanceValues.size()));

    if (instanceValues.size() == 0)
    {
        hr = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"GetInstanceId: Failed to find attribute NodeQuery:[ %s] Attribue:[%s]", incarnationNodePath.c_str(), INSTANCE_ATTRIBUTE_NAME),
            hr,
            Cleanup);
    }

    if (instanceValues.size() > 1)
    {
        hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"GetInstanceId: Failed more than one attribute value. NodeQuery:[ %s] Attribue:[%s]", incarnationNodePath.c_str(), INSTANCE_ATTRIBUTE_NAME),
            hr,
            Cleanup);
    }

    InstanceName.clear();
    InstanceName.assign(*instanceValues.begin());

Cleanup:

    return hr;
}

HRESULT GetInstanceAddress(
    __in XmlParserHelper* xmlParserHelper,
    __in std::wstring InstanceName,
    __inout std::wstring& InstanceAddress)
/*++

Routine Description:

    Helper to retrieve the instance address from the RCF.

Arguments:

    XmlParserHelper - XmlParserHelepr object which has the RCF file loaded.

    InstanceName - Instance name whose address to be retrieved from RCF.

    InstanceAddress - Out parameter to hold the instance address.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    HRESULT hr = S_OK;
    std::set<std::wstring> instanceValues;
    std::wstring instanceAddressNodePath;
    instanceAddressNodePath.append(XPATH_ROOT_START_TAG);
    instanceAddressNodePath.append(INSTANCE_NODE_NAME);
    instanceAddressNodePath.append(L"[@");
    instanceAddressNodePath.append(ID_ATRRIBUTE_NAME);
    instanceAddressNodePath.append(L"='");
    instanceAddressNodePath.append(InstanceName);
    instanceAddressNodePath.append(L"']");

    hr = xmlParserHelper->QueryNodeAttributeValues(
        (LPWSTR)instanceAddressNodePath.c_str(),
        ADDRESS_ATTRIBUTE_NAME,
        instanceValues);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"GetInstanceAddress failed with error [%0x]. NodeQuery:[%s], Attribute[%s]", hr, instanceAddressNodePath.c_str(), ADDRESS_ATTRIBUTE_NAME),
        hr,
        Cleanup);

    RD_TRACE_LOG((RD_TRACE_INFO,
        L"NodeQuery:%s, Attribute:%s, Count:%d.",
        instanceAddressNodePath.c_str(),
        ADDRESS_ATTRIBUTE_NAME,
        instanceValues.size()));

    if (instanceValues.size() == 0)
    {
        hr = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"GetInstanceAddress: Failed to find attribute NodeQuery:[ %s] Attribue:[%s]", instanceAddressNodePath.c_str(), ADDRESS_ATTRIBUTE_NAME),
            hr,
            Cleanup);
    }

    if (instanceValues.size() > 1)
    {
        hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"GetInstanceAddress: Failed more than one attribute value. NodeQuery:[ %s] Attribue:[%s]", instanceAddressNodePath.c_str(), ADDRESS_ATTRIBUTE_NAME),
            hr,
            Cleanup);
    }

    InstanceAddress.clear();
    InstanceAddress.assign(*instanceValues.begin());

Cleanup:

    return hr;
}

HRESULT GetServiceTunnelIpv6Addresses(
    __in XmlParserHelper* xmlParserHelper,
    __in std::wstring InstanceName,
    __inout std::set<std::wstring>& ServiceTunnelIpv6Addresses)
/*++

Routine Description:

    Helper to retrieve the service tunnel Ipv6 addresses from the RCF.

Arguments:

    XmlParserHelper - XmlParserHelepr object which has the RCF file loaded.

    InstanceName - Instance name whose address to be retrieved from RCF.

    InstanceAddress - Out parameter to hold the service tunnel Ipv6 addresses.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    HRESULT hr = S_OK;
    std::set<std::wstring> instanceValues;
    std::wstring serviceTunnelAddressNodePath;
    std::set<std::wstring>::iterator instanceValuesIter;

    serviceTunnelAddressNodePath.append(XPATH_ROOT_START_TAG);
    serviceTunnelAddressNodePath.append(INSTANCE_NODE_NAME);
    serviceTunnelAddressNodePath.append(xmlParserHelper->PrepareAttributeQuery(ID_ATRRIBUTE_NAME, InstanceName));
    serviceTunnelAddressNodePath.append(XPATH_PATH_SEPARATOR_TAG);
    serviceTunnelAddressNodePath.append(L"InputEndpoints");
    serviceTunnelAddressNodePath.append(XPATH_PATH_SEPARATOR_TAG);
    serviceTunnelAddressNodePath.append(ENDPOINT_NODE_NAME);

    hr = xmlParserHelper->QueryNodeAttributeValues(
        serviceTunnelAddressNodePath.c_str(),
        SERVICE_TUNNEL_IPV6_ATTRIBUTE_NAME,
        instanceValues);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"GetInstanceAddress failed with error [%0x]. NodeQuery:[%s], Attribute[%s]", hr, serviceTunnelAddressNodePath.c_str(), ADDRESS_ATTRIBUTE_NAME),
        hr,
        Cleanup);

    RD_TRACE_LOG((RD_TRACE_INFO,
        L"NodeQuery:%s, Attribute:%s, Count:%d.",
        serviceTunnelAddressNodePath.c_str(),
        ADDRESS_ATTRIBUTE_NAME,
        instanceValues.size()));

    ServiceTunnelIpv6Addresses.clear();
    for (instanceValuesIter = instanceValues.begin(); instanceValuesIter != instanceValues.end(); ++instanceValuesIter)
    {
        ServiceTunnelIpv6Addresses.insert(*instanceValuesIter);
    }

Cleanup:

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
    std::set<std::wstring> serviceTunnelIpv6Addresses;
    std::set<std::wstring>::iterator serviceTunnelIpv6AddressesIter;
    bool comInitialized = false;
    bool traceInitialized = false;
    std::wstring instanceName;
    std::wstring instanceAddress;
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
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"Initialize failed with error [%0x]. RcfFile:[%s]", hr, rcfFile),
        hr,
        Cleanup);

    hr = GetInstanceId(&xmlParserHelper, instanceName);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"GetInstanceId failed with error [%0x]. ", hr),
        hr,
        Cleanup);

    RD_TRACE_LOG((RD_TRACE_INFO, L"Instance Name:%s.",instanceName.c_str()));

    hr = GetInstanceAddress(&xmlParserHelper, instanceName, instanceAddress);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"GetInstanceAddress failed with error [%0x]. ", hr),
        hr,
        Cleanup);

    RD_TRACE_LOG((RD_TRACE_INFO, L"Instance Address:%s.", instanceAddress.c_str()));

    hr = GetServiceTunnelIpv6Addresses(&xmlParserHelper, instanceName, serviceTunnelIpv6Addresses);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"GetServiceTunnelIpv6Addresses failed with error [%0x]. ", hr),
        hr,
        Cleanup);

    for (serviceTunnelIpv6AddressesIter = serviceTunnelIpv6Addresses.begin(); serviceTunnelIpv6AddressesIter != serviceTunnelIpv6Addresses.end(); ++serviceTunnelIpv6AddressesIter)
    {
        RD_TRACE_LOG((RD_TRACE_INFO, L"ServiceTunnelIpv6Address:%s", serviceTunnelIpv6AddressesIter->c_str()));
    }

    if (serviceTunnelIpv6Addresses.size() > 0)
    {
        serviceTunnelConfigurator.SetupLoopbackAdapter(serviceTunnelIpv6Addresses);
    }
    else
    {
        RD_TRACE_LOG((RD_TRACE_INFO, L"Skipped configuring service tunnel adapter as there was no endpoint enabled for service tunnelling"));
    }

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

    return hr;
}