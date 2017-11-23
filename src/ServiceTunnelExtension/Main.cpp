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
#include "Logging.h"
#include <sstream>
#include <string>
#include <iomanip>

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

#define LOG_FILE_PREFIX L"ServiceTunnelExtension"


void SetupLogging();


void SetupLogging(
    wchar_t *logFolder
    )
/*++

Routine Description:

    Helper to set up logging.

Arguments:

    logFolder - Path of log directory.

Return Value:


--*/
{
    DWORD error;
    LPWSTR logDestFolder = NULL;


    error = AppendStrings(&logDestFolder,
        2,
        logFolder,
        L"\\ServiceTunnelExtension\\");


    wprintf(L"Creating log folder[%s] [%d]", logDestFolder, error);

    
    if (!CreateDirectory(logDestFolder, NULL))
    {
        error = GetLastError();
        if (error != ERROR_ALREADY_EXISTS)
        {
            wprintf(L"Failed to create log folder [%s] due to erroro (%d)\n", logDestFolder, error);
			goto Cleanup;
        }
    }

	LOGSETUP(logDestFolder, LOG_FILE_PREFIX);

Cleanup:
    
    STR_FREE(logDestFolder);
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
	bool xmlInitialized = false;
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
					wprintf(L"Failed retrieving retry count");
					goto Cleanup;
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
					wprintf(L"Failed retrieving waitTimeStart");
					goto Cleanup;
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
					wprintf(L"Failed retrieving totalWaitTime");
					goto Cleanup;
				}
			}
		}
        else if(_wcsicmp(L"-remove", argv[i]) == 0)
        {
            unInstallFlag = true;
            wprintf(L"Uninstall argument is set");
        }
        else if(_wcsicmp(L"-log", argv[i]) == 0)
        {
            if (i + 1 < argc)
            {
                logFolder = argv[++i];
                if (logFolder == NULL)
                {
					hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
					wprintf(L"Failed retrieving logFolder");
					goto Cleanup;
                }
           }
        }
        else
        {
            wprintf(L"Invalid Argument %s",argv[i]);
            return 1;
        }
	}
    
    SetupLogging(logFolder);

    LOGINFOW(L"RetryCount %d startwaittime %d totalwaittime %d", retryCount, waitTimeStart, totalWaitTime);
        
    traceInitialized = true;

    hr = ::CoInitialize(NULL);
	if (hr != S_OK)
	{
		LOGERRW(L"ComInitialization failed with error [%0x].", hr);
		goto Cleanup;
	}
 
    comInitialized = true;

	hr = rcfEnumerator.GetRcfLocation(&rcfLocation);
	if (hr != S_OK)
	{
		LOGERRW(L"GetRcfLocation failed with error [%0x].", hr);
		goto Cleanup;
	}

    hr = rcfEnumerator.FindLatestRcf(&rcfFile, rcfLocation);
	if (hr != S_OK)
	{
		LOGERRW(L"FindLatestRcf failed with error [%0x].", hr);
		goto Cleanup;
	}

    hr = xmlParserHelper.Initialize(rcfFile);
	if (hr != S_OK)
	{
		LOGERRW(L"XML Initialize failed with error [%0x].", hr);
		goto Cleanup;
	}

	xmlInitialized = true;

    hr = configReader.GetInstanceId(&xmlParserHelper, instanceName);
	if (hr != S_OK)
	{
		LOGERRW(L"GetInstanceId failed with error [%0x].", hr);
		goto Cleanup;
	}

    LOGINFOW(L"Instance Name:%s.",instanceName.c_str());

    hr = configReader.GetInstanceAddress(&xmlParserHelper, instanceName, instanceAddress);
	if (hr != S_OK)
	{
		LOGERRW(L"GetInstanceAddress failed with error [%0x].", hr);
		goto Cleanup;
	}

	LOGINFOW(L"Instance Address:%s.", instanceAddress.c_str());

    hr = configReader.GetServiceTunnelIpv6Addresses(&xmlParserHelper, instanceName, serviceTunnelIpv6Addresses);
	if (hr != S_OK)
	{
		LOGERRW(L"GetServiceTunnelIpv6Addresses failed with error [%0x].", hr);
	}

    for (serviceTunnelIpv6AddressesIter = serviceTunnelIpv6Addresses.begin(); serviceTunnelIpv6AddressesIter != serviceTunnelIpv6Addresses.end(); ++serviceTunnelIpv6AddressesIter)
    {
		LOGINFOW(L"ServiceTunnelIpv6Address:%s", serviceTunnelIpv6AddressesIter->c_str());
    }

    if (serviceTunnelIpv6Addresses.size() == 0)
    {
		LOGINFOW(L"Configuring service tunnel adapter with IPV6 address derived implicitly from Instance Ipv4 address");
        instanceDerviedIpv6Address = serviceTunnelConfigurator.GetSTIpv6AddrFromInstanceIpv4Addr(instanceAddress);
        serviceTunnelIpv6Addresses.insert(instanceDerviedIpv6Address);
    }

    if(unInstallFlag)
    {
        hr = serviceTunnelConfigurator.CleanupLoopbackAdapter(instanceAddress);
		if (hr != S_OK)
		{
			LOGERRW(L"CleanupLoopbackAdapter failed with error [%0x].", hr);
			goto Cleanup;
		}
    }
    else
    {
        hr = serviceTunnelConfigurator.SetupLoopbackAdapter(instanceAddress, serviceTunnelIpv6Addresses, retryCount, waitTimeStart, totalWaitTime);
		if (hr != S_OK)
		{
			LOGERRW(L"SetupLoopbackAdapter failed with error [%0x].", hr);
			goto Cleanup;
		}
    }

Cleanup:

	STR_FREE(rcfFile);

	if (xmlInitialized)
	{
		xmlParserHelper.UnInitialize();
	}

    if (comInitialized)
    {
        CoUninitialize();
    }

	LOGINFOW(L"Return Code from main %x",hr);

    return hr;
}