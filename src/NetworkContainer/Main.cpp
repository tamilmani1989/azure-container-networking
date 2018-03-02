/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

	Main.cpp

Abstract:

	Entry point for NetworkContainer.

Author:

	Yongli Chen

--*/
#include <stdlib.h>  
#include <crtdbg.h> 
#include <stdio.h>
#include <wchar.h>
#include <ws2tcpip.h>
#include <msxml6.h>
#include <windows.h>
#include "RcfEnumerator.h"
#include <iostream>
#include "ConfigReader.h"
#include "NetworkContainer.h"
#include "Logging.h"
#include <sstream>
#include <string>
#include <iomanip>
#include <deque>

#define LOG_FILE_PREFIX L"NetworkContainer"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

template <class T>
void splitWstr(const wstring& Wstr, T& container, wchar_t delim = L',')
{
	wstringstream ss(Wstr);
	wstring token;
	while (getline(ss, token, delim)) { container.push_back(token); }
}

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
	usage += L"/logpath log_path";
	usage += L"/name name_of_device ";
	usage += L"/type type_of_request ";
	usage += L"/operation type_of_operation ";
	usage += L"/ip ip_address_of_device ";
	usage += L"[/netmask netmask_address]";
	usage += L"[/gateway gateway_address]";
	usage += L"[/dns dns_server_address] ";
	wcout << usage << endl;

	wstring create = L"Create example:\n";
	create += L"AzureNetworkContainer /logpath C:\\Logs\\NetworkContainer ";
	create += L"/name funny ";
	create += L"/operation CREATE ";
	create += L"/ip 1.2.3.4 ";
	create += L"/netmask 255.255.255.0 ";
	create += L"/gateway 3.3.3.3 ";
	create += L"/dns 8.8.8.8,8.8.8.7";
	wcout << create << endl;

	wstring update = L"Update example:\n";
	update += L"AzureNetworkContainer /logpath C:\\Logs\\NetworkContainer ";
	update += L"/name funny ";
	update += L"/operation UPDATE ";
	update += L"/ip 1.2.3.4 ";
	update += L"/netmask 255.255.255.0 ";
	update += L"/gateway 3.3.3.3 ";
	update += L"/dns 8.8.8.8,8.8.8.7,8.8.8.6";
	wcout << update << endl;

	wstring del = L"Delete example:\n";
	del += L"AzureNetworkContainer /logpath C:\\Logs\\NetworkContainer ";
	del += L"/name funny ";
	del += L"/operation DELETE ";
	wcout << del << endl;
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
	if (ret)
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
	_Out_ wstring& Operation,
	_Out_ IN_ADDR& Ip,
	_Out_ IN_ADDR& Netmask,
	_Out_ IN_ADDR& Gateway,
	_Out_ vector<IN_ADDR>& DnsVec,
	_Out_ bool& WhsFlag,
	_Out_ bool& WhrFlag
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

	Ip: Required if Type is "Anteres" and Operation is "CREATE".
		   IP adress of the device instance.

	Dns: Optional.
		    DNS server address.

	Gateway: Optional.
			    Gateway address of the device instance.

	WhsFlag: Optional.
			 Flag for weak host send.

	WhrFlag: Optional.
			 Flag for weak host receive.

--*/
{
	HRESULT hr = S_OK;
	bool ipSet = false;

	for (int i = 1; i < ArgCount - 1; i++)
	{
		wstring arg = ArgList[i];
		if (arg == HELP_CMDLINE_FLAG)
		{
			PrintUsage();
			return hr;
		}
		if (arg == LOG_PATH_CMDLINE_FLAG)
		{
			LogPath = ArgList[++i];
		} 
		else if (arg == NAME_CMDLINE_FLAG)
		{
			Name = ArgList[++i];
		} 
		else if (arg == OPERATION_CMDLINE_FLAG)
		{
			Operation = ArgList[++i];
		} 
		else if (arg == IP_CMDLINE_FLAG)
		{
			hr = ConvertToINADDR(ArgList[++i], Ip);
			if (hr != S_OK)
			{
				return hr;
			}
			ipSet = true;
		} 
		else if (arg == NETMASK_CMDLINE_FLAG)
		{
			hr = ConvertToINADDR(ArgList[++i], Netmask);
			if (hr != S_OK)
			{
				return hr;
			}
		} 
		else if (arg == GATEWAY_CMD_LINE_FLAG)
		{
			hr = ConvertToINADDR(ArgList[++i], Gateway);
			if (hr != S_OK)
			{
				return hr;
			}
		} 
		else if (arg == DNS_CMDLINE_FLAG)
		{
			wstring dnsStrList(ArgList[++i]);
			deque<wstring> dnsDeq;
			splitWstr(dnsStrList, dnsDeq);
			while (!dnsDeq.empty())
			{
				IN_ADDR dns = { 0 };
				hr = ConvertToINADDR((wchar_t*)dnsDeq.front().c_str(), dns);
				dnsDeq.pop_front();
				if (hr != S_OK)
				{
					return hr;
				}
				DnsVec.push_back(dns);
			}
		} 
		else if (arg == WEAKHOST_SEND_FLAG)
		{

			if (wstring(ArgList[++i]) == L"true") 
			{
				WhsFlag = true;
				continue;
			} 
		} 
		else if (arg == WEAKHOST_RECV_FLAG)
		{
			if (wstring(ArgList[++i]) == L"true")
			{
				WhrFlag = true;
				continue;
			}
		} 
		else 
		{
			hr = E_INVALIDARG;
			return hr;
		}
	}

	if (Name.empty() || Operation.empty())
	{
		hr = E_INVALIDARG;
		return hr;
	}

	if ( Operation == L"CREATE" &&
		!ipSet)
	{
		hr = E_INVALIDARG;
		return hr;
	}

	return hr;
}

HRESULT dispatch(_In_ NetworkContainer& nc,
	_In_ wstring Name,
	_Inout_ wstring Operation,
	_In_ IN_ADDR Ip,
	_In_ IN_ADDR Netmask,
	_In_ IN_ADDR Gateway,
	_In_ vector<IN_ADDR> DnsVec,
	_In_ bool WhsFlag,
	_In_ bool WhrFlag
)
/*++

Routine Description:

	This routine parses command line arguments for NetworkContainer.exe during invocation.

Arguments:

	Name: Name of the device instance.

	Operation: Operation to be done on the device instance.
			   For loopback adapter, we support "CREATE"/"UPDATE"/"DELETE" operations.

	Ip: IP adress of the device instance.

	Dns: DNS server address.

	Gateway: Gateway address of the device instance.

	WhsFlag: Flag for weak host send.

	WhrFlag: Flag for weak host receive.

--*/

{
	HRESULT hr = S_OK;

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

		hr = nc.ConfigLoopbackAdapter(Name, Ip, Netmask, Gateway, DnsVec, WhsFlag, WhrFlag);
		if (hr != S_OK)
		{
			LOGERRW(L"Failed to configure loopback adapter with error: [%0x].", hr);
			return hr;
		}
	} 
	else if (Operation == L"UPDATE")
	{
		hr = nc.ConfigLoopbackAdapter(Name, Ip, Netmask, Gateway, DnsVec, WhsFlag, WhrFlag);
		if (hr != S_OK)
		{
			LOGERRW(L"Failed to created loopback adapter with error [%0x].", hr);
			return hr;
		}
	} 
	else if (Operation == L"DELETE")
	{
		hr = nc.DeleteLoopbackAdapter(Name);
		if (hr != S_OK)
		{
			LOGERRW(L"Failed to delete loopback adapter with error: [%0x].", hr);
			return hr;
		}
	} 
	else 
	{
		hr = E_INVALIDARG;
		PrintUsage();
		return hr;
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
	wstring logPath, name, operation;
	IN_ADDR ip, netmask, gateway = { 0 };
	vector<IN_ADDR> dnsVec;
	bool whsFlag = false, whrFlag = false;

	hr = ParseCmdArguments(argc, 
		argv, 
		logPath,
		name,
		operation,
		ip, 
		netmask,
		gateway,
		dnsVec,
		whsFlag,
		whrFlag
	);

	if (hr != S_OK)
	{
		PrintUsage();
		return hr;
	}

	if (logPath.empty())
	{ 
		logPath = DEFAULT_LOG_LOCATION; 
	}

	SetupLogging(logPath.c_str());
	LOGINFOW(L"Setup logging succeeded.");

	NetworkContainer nc(name, ip, netmask, gateway, dnsVec);

	hr = dispatch(nc, name, operation, ip, netmask, gateway, dnsVec, whsFlag, whrFlag);
	if (hr != S_OK)
	{
		LOGERRW(L"dispatch failed with error [%0x]", hr);
	}

	return hr;
}