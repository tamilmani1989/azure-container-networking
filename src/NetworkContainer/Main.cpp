/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

	Main.cpp

Abstract:

	Entry point for NetworkContainer.

--*/

#include <stdio.h>
#include <wchar.h>
#include <ws2tcpip.h>
#include <msxml6.h>
#include <windows.h>
#include "RcfEnumerator.h"
#include <iostream>
#include "ConfigReader.h"
#include "..\NetworkContainerLibrary\NetworkContainer.h"
#include "..\VFPluginLibrary\VFPluginUtilities.h"
#include "Logging.h"
#include <sstream>
#include <string>
#include <iomanip>

#define LOG_FILE_PREFIX L"NetworkContainer"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

bool
SetupLogging(
	__in PCWSTR LogLocation
)
/*++

Routine Description:

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

			std::wcout << L"Failed to set up logging directory. Error: [0" << hex << err << " ]" << std::endl;
			return false;
		}
	}

	hr = LOGSETUP(LogLocation, L"NetworkContainer");

	return hr == S_OK ? true : false;
}

void PrintUsage()
/*++

Routine Description:

	This routine prints usage for NetworkContainer.exe

Return Value:

	None.

--*/
{
	wstring usage = L"Usage: NetworkContainer ";
	usage += L"/name name_of_device ";
	usage += L"/type type_of_request ";
	usage += L"/operation type_of_operation ";
	usage += L"/ip ip_address_of_device ";
	usage += L"[/netmask netmask_address]";
	usage += L"[/gateway gateway_address]";
	usage += L"[/dns dns_server_address] ";

	wcout << usage << endl;
}

HRESULT ConvertToINADDR(_In_ wchar_t* Str, IN_ADDR& Retval)
/*++

Routine Description:
	
	This routine convert ip address from string representation to IN_ADDR struct.

Arguments:

	Str: string representation of ip address.

	Retval: IN_ADDR representation of ip address.

Return Value:

	HRESULT - Returns if the conversion succeeds or not.

--*/
{
	DWORD ret;
	HRESULT hr = S_OK;
	PCWSTR temp = NULL;
	ret = RtlIpv4StringToAddress(Str, true, &temp, &Retval);
	if (!ret)
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			wcout << L"NetworkContainer\\Main.cpp:: RtlIpv4StringToAddress function failed with error: " << err << " "
				<< "Address: " << Str << " "
				<< "hr: " << hex << hr << endl;
		}
	}
	return hr;
}

HRESULT
ParseCmdArguments(
	_In_ int ArgCount,
	_In_ wchar_t* ArgList[],
	_Out_ wstring& LogPath,
	_Out_ wstring& Name,
	_Out_ wstring& Type,
	_Out_ wstring& Operation,
	_Out_ IN_ADDR& Ip,
	_Out_ IN_ADDR& Netmask,
	_Out_ IN_ADDR& Gateway,
	_Out_ IN_ADDR& Dns
)
/*++

Routine Description:

	This routine parses command line arguments for NetworkContainer.exe during invocation.

Arguments:

	ArgCount: total number of command line arguments.
	
	ArgList: A pointer to command line arguments.

	LogPath: Optional. 
			 Output string to log path option if found in ArgList.

	Name: Required. 
	      Name of the device instance.

	Type: Required. 
	      Type of request.
		  We only support "Anteres" for now.

	Operation: Required. 
			   Operation to be done on the device instance.
			   For loopback adapter, we support "CREATE"/"UPDATE"/"DELETE" operations.

	IpStr: Required if Type is "Anteres" and Operation is "CREATE".
		   IP adress of the device instance.

	DnsStr: Optional.
		    DNS server address.

	GatewayStr: Optional.
			    Gateway address of the device instance.

--*/
{
	HRESULT hr = S_OK;

	bool ipSet = false;
	for (int i = 1; i < ArgCount - 1; i++)
	{
		wstring arg = ArgList[i];
		if (arg == LOG_PATH_CMDLINE_FLAG)
		{
			LogPath = ArgList[++i];
		} else if (arg == NAME_CMDLINE_FLAG)
		{
			Name = ArgList[++i];
		} else if (arg == TYPE_CMDLINE_FLAG)
		{
			Type = ArgList[++i];
		} else if (arg == OPERATION_CMDLINE_FLAG)
		{
			Operation = ArgList[++i];
		} else if (arg == IP_CMDLINE_FLAG)
		{
			hr = ConvertToINADDR(ArgList[++i], Ip);
			if (hr != S_OK)
			{
				return hr;
			}
			ipSet = true;
		} else if (arg == NETMASK_CMDLINE_FLAG)
		{
			hr = ConvertToINADDR(ArgList[++i], Netmask);
			if (hr != S_OK)
			{
				return hr;
			}
		} else if (arg == GATEWAY_CMD_LINE_FLAG)
		{
			hr = ConvertToINADDR(ArgList[++i], Gateway);
			if (hr != S_OK)
			{
				return hr;
			}
		} else if (arg == DNS_CMDLINE_FLAG)
		{
			hr = ConvertToINADDR(ArgList[++i], Dns);
			if (hr != S_OK)
			{
				return hr;
			}
		} else
		{
			hr = E_INVALIDARG;
			return hr;
		}
	}

	if (Name.empty() || 
		Type.empty() || 
		Operation.empty())
	{
		hr = E_INVALIDARG;
		return hr;
	}

	if (Type == L"Antares" && 
		Operation == L"CREATE" &&
		!ipSet)
	{
		hr = E_INVALIDARG;
		return hr;
	}

	return hr;
}

HRESULT dispatch(_In_ NetworkContainer& nc,
	_In_ wstring Name,
	_In_ wstring Type,
	_Inout_ wstring Operation,
	_In_ IN_ADDR Ip,
	_In_ IN_ADDR Netmask,
	_In_ IN_ADDR Dns,
	_In_ IN_ADDR Gateway
)
{
	HRESULT hr = S_OK;

	if (Type == L"Antares")
	{
		if (Operation == L"CREATE")
		{
			hr = nc.CreateLoopbackAdapter(Name);
			if (hr == HRESULT_FROM_WIN32(ERROR_ALIAS_EXISTS))
			{
				LOGERRW(L"Failed to created loopback adapter. Adapter with name %s already exists.", Name.c_str());
				return hr;
			}
			if (hr != S_OK)
			{
				LOGERRW(L"Failed to created loopback adapter with error [%0x].", hr);
				return hr;
			}

			nc.ConfigLoopbackAdapter(Name, Ip, Netmask, Gateway, Dns);
			if (hr != S_OK)
			{
				LOGERRW(L"Failed to configure loopback adapter with error: [%0x].", hr);
				return hr;
			}
		} else if (Operation == L"UPDATE")
		{
			hr = nc.ConfigLoopbackAdapter(Name, Ip, Netmask, Gateway, Dns);
			if (hr != S_OK)
			{
				LOGERRW(L"Failed to created loopback adapter with error [%0x].", hr);
				return hr;
			}
		} else if (Operation == L"DELETE")
		{
			hr = nc.DeleteLoopbackAdapter(Name);
			if (hr != S_OK)
			{
				LOGERRW(L"Failed to delete loopback adapter with error: [%0x].", hr);
				return hr;
			}
		} else 
		{
			hr = E_INVALIDARG;
			return hr;
		}
	}
	return hr;
}

int wmain(_In_ int argc, _In_ wchar_t *argv[])
/*++

Routine Description:

	This is the entry point for creating network container.

Arguments:

	argc: Argument count.
	argv: List of command line arguments.

--*/
{
	HRESULT hr = S_OK;

	wstring logPath, name, operation, type;
	IN_ADDR ip = { 0 };
	IN_ADDR netmask = { 0 };
	IN_ADDR gateway = { 0 };
	IN_ADDR dns = { 0 };

	hr = ParseCmdArguments(argc, 
		argv, 
		logPath,
		name,
		type,
		operation,
		ip, 
		netmask,
		gateway,
		dns);
	if (hr != S_OK)
	{
		PrintUsage();
		return hr;
	}

	if (logPath.empty()) { logPath = DEFAULT_LOG_LOCATION; }
	SetupLogging(logPath.c_str());
	LOGINFOW(L"Setup logging succeeded.");

	// print out arguments.
	wchar_t ipStr[INET_ADDRSTRLEN], netmaskStr[INET_ADDRSTRLEN], dnsStr[INET_ADDRSTRLEN], gatewayStr[INET_ADDRSTRLEN];
	wcout << "logPath: " << logPath << endl;
	wcout << "name: " << name << endl;
	wcout << "type: " << type << endl;
	wcout << "operation: " << operation << endl;
	inet_ntop(AF_INET, (void *)&ip, (char*)ipStr, sizeof(ipStr));
	inet_ntop(AF_INET, (void *)&netmask, (char*)netmaskStr, sizeof(netmaskStr));
	inet_ntop(AF_INET, (void *)&dns, (char*)dnsStr, sizeof(dnsStr));
	inet_ntop(AF_INET, (void *)&gateway, (char*)gatewayStr, sizeof(gatewayStr));
	printf("ip: %s\n", (char*)ipStr);
	printf("netmask: %s\n", (char*)netmaskStr);
	printf("dns: %s\n", (char*)dnsStr);
	printf("gateway: %s\n", (char*)gatewayStr);

	NetworkContainer nc(name, type, operation, ip, netmask, dns, gateway);

	hr = dispatch(nc, name, type, operation, ip, netmask, dns, gateway);
	if (hr != S_OK)
	{
		LOGERRW(L"dispatch failed with error [%0x]", hr);
	}

	return hr;
}