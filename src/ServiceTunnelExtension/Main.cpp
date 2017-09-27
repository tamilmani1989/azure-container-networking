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
#include <stdio.h>
#include <wchar.h>
#include <ws2tcpip.h>
#include <msxml6.h>
#include <windows.h>
#include "RcfEnumerator.h"
#include <iostream>
#include "ConfigReader.h"
#include "ServiceTunnelLibrary.h"
#include <sstream>
#include <string>
#include <iomanip>

#define AGENT_TRACE_FLAGS (RD_TRACE_ALL & ~RD_TRACE_PAINFUL)
#define LOG_FILE_PREFIX L"ServiceTunnelExtension"


HRESULT SetupLogging();


HRESULT SetupLogging(
    wchar_t *logFolder
    )
/*++

Routine Description:

    Helper to set up logging.

Arguments:

    logFolder - Path of log directory.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    DWORD error;
    HRESULT hr;
    LPWSTR logDestFolder = NULL;
    LPWSTR logFilePathWithPrefix = NULL;


    error = AppendStrings(&logDestFolder,
        2,
        logFolder,
        L"\\ServiceTunnelExtension\\");


    wprintf(L"Creating log folder[%s] [%d]", logDestFolder, error);

    RD_E_WIN32(error, Cleanup);
    
    if (!CreateDirectory(logDestFolder, NULL))
    {
        error = GetLastError();
        if (error != ERROR_ALREADY_EXISTS)
        {
            wprintf(L"Failed to create log folder [%s] due to erroro (%d)\n", logDestFolder, error);
            RD_E_WIN32(error, Cleanup);
        }
    }

    error = AppendStrings(&logFilePathWithPrefix,
        2,
        logDestFolder,
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
    
    STR_FREE(logDestFolder);
    STR_FREE(logFilePathWithPrefix);

    return hr;
}


int wmain(
	int argc,
	wchar_t *argv[]
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
	LPWSTR rcfLocation = NULL;
	RcfEnumerator rcfEnumerator;
	XmlParserHelper xmlParserHelper;
	ConfigReader configReader;
	std::set<std::wstring> serviceTunnelIpv6Addresses;
	std::set<std::wstring>::iterator serviceTunnelIpv6AddressesIter;
	bool comInitialized = false;
	bool traceInitialized = false;
    bool unInstallFlag = false;
	std::wstring instanceName;
	std::wstring instanceAddress;
	std::wstring instanceDerviedIpv6Address;
	int retryCount = 30;
	int waitTimeStart = 5;
	int totalWaitTime = 180;
    wchar_t *logFolder = NULL;
	ServiceTunnelConfigurator serviceTunnelConfigurator;
	

	for (int i = 1; i < argc; i++)
	{
		if (_wcsicmp(L"-retrycount", argv[i]) == 0)
		{
			if (i + 1 < argc)
			{
				retryCount = _wtoi(argv[++i]);
				if (retryCount == 0)
				{
					hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
					RD_TRACE_E_HR((RD_TRACE_ERROR,
						L"_wtoi failed retrieving retry count"),
						hr,
						Cleanup);
				}
			}
		}
		else if (_wcsicmp(L"-waittimestart", argv[i]) == 0)
		{
			if (i + 1 < argc)
			{
				waitTimeStart = _wtoi(argv[++i]);
				if (waitTimeStart == 0)
				{
					hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
					RD_TRACE_E_HR((RD_TRACE_ERROR,
						L"_wtoi failed retrieving waittimestart"),
						hr,
						Cleanup);
				}
			}
		}
		else if(_wcsicmp(L"-totalwaittime", argv[i]) == 0)
		{
			if (i + 1 < argc)
			{
				totalWaitTime = _wtoi(argv[++i]);
				if (totalWaitTime == 0)
				{
					hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
					RD_TRACE_E_HR((RD_TRACE_ERROR,
						L"Failed retrieving totalwaittime"),
						hr,
						Cleanup);
				}
			}
		}
        else if(_wcsicmp(L"-remove", argv[i]) == 0)
        {
            unInstallFlag = true;
            RD_TRACE_LOG((RD_TRACE_INFO, L"Uninstall argument is set"));
        }
        else if(_wcsicmp(L"-log", argv[i]) == 0)
        {
            if (i + 1 < argc)
            {
                logFolder = argv[++i];
                if (logFolder == NULL)
                {
                    hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
                    RD_TRACE_E_HR((RD_TRACE_ERROR,
                        L"Failed retrieving logFolder"),
                        hr,
                        Cleanup);
                }
           }
        }
        else
        {
            RD_TRACE_LOG((RD_TRACE_INFO, L"Invalid Argument %s",argv[i]));
            return 1;
        }
	}
    
    hr = SetupLogging(logFolder);
	if (FAILED(hr))
	{
		wprintf(L"Failed to setup logging due to error [%0x]", hr);
		goto Cleanup;
	}

    RD_TRACE_LOG((RD_TRACE_INFO, L"RetryCount %d startwaittime %d totalwaittime %d", retryCount, waitTimeStart, totalWaitTime));
        
    traceInitialized = true;

    hr = ::CoInitialize(NULL);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"ComInitialization failed with error [%0x].",
        hr),
        hr,
        Cleanup);

    comInitialized = true;

	hr = rcfEnumerator.GetRcfLocation(&rcfLocation);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"GetRcfLocation failed with error [%0x].",
		hr),
		hr,
		Cleanup);

    hr = rcfEnumerator.FindLatestRcf(&rcfFile, rcfLocation);
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

    hr = configReader.GetInstanceId(&xmlParserHelper, instanceName);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"GetInstanceId failed with error [%0x]. ", hr),
        hr,
        Cleanup);

    RD_TRACE_LOG((RD_TRACE_INFO, L"Instance Name:%s.",instanceName.c_str()));

    hr = configReader.GetInstanceAddress(&xmlParserHelper, instanceName, instanceAddress);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"GetInstanceAddress failed with error [%0x]. ", hr),
        hr,
        Cleanup);

    RD_TRACE_LOG((RD_TRACE_INFO, L"Instance Address:%s.", instanceAddress.c_str()));

    hr = configReader.GetServiceTunnelIpv6Addresses(&xmlParserHelper, instanceName, serviceTunnelIpv6Addresses);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"GetServiceTunnelIpv6Addresses failed with error [%0x]. ", hr),
        hr,
        Cleanup);

    for (serviceTunnelIpv6AddressesIter = serviceTunnelIpv6Addresses.begin(); serviceTunnelIpv6AddressesIter != serviceTunnelIpv6Addresses.end(); ++serviceTunnelIpv6AddressesIter)
    {
        RD_TRACE_LOG((RD_TRACE_INFO, L"ServiceTunnelIpv6Address:%s", serviceTunnelIpv6AddressesIter->c_str()));
    }

    if (serviceTunnelIpv6Addresses.size() == 0)
    {
        RD_TRACE_LOG((RD_TRACE_INFO, L"Configuring service tunnel adapter with IPV6 address derived implicitly from Instance Ipv4 address"));
        instanceDerviedIpv6Address = serviceTunnelConfigurator.GetSTIpv6AddrFromInstanceIpv4Addr(instanceAddress);
        serviceTunnelIpv6Addresses.insert(instanceDerviedIpv6Address);
    }

    if(unInstallFlag)
    {
        hr = serviceTunnelConfigurator.CleanupLoopbackAdapter(instanceAddress);
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"CleanupLoopbackAdapter failed with error [%0x]. ", hr),
            hr,
            Cleanup);
    }
    else
    {
        hr = serviceTunnelConfigurator.SetupLoopbackAdapter(instanceAddress, serviceTunnelIpv6Addresses, retryCount, waitTimeStart, totalWaitTime);
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"SetupLoopbackAdapter failed with error [%0x]. ", hr),
            hr,
            Cleanup);
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

	//hr= HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
    RD_TRACE_LOG((RD_TRACE_INFO, L"Return Code from main %x",hr));
    return hr;
}