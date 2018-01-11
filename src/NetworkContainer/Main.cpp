#include <stdio.h>
#include <wchar.h>
#include <ws2tcpip.h>
#include <msxml6.h>
#include <windows.h>
#include "RcfEnumerator.h"
#include <iostream>
#include "ConfigReader.h"
#include "NetworkContainer.h"
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
	usage += L"[/dns dns_server_address] ";
	usage += L"[/gateway gateway_address]";

	wcout << usage << endl;
}

HRESULT
ParseCmdArguments(
	_In_ int ArgCount,
	_In_ wchar_t* ArgList[],
	_Out_ wstring& LogPath,
	_Out_ wstring& Name,
	_Out_ wstring& Type,
	_Out_ wstring& Operation,
	_Out_ wstring& IpStr,
	_Out_ wstring& DnsStr,
	_Out_ wstring& GatewayStr
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
	
	if (ArgCount < MIN_ARGCOUNT || 
		ArgCount > MAX_ARGCOUNT)
	{
		PrintUsage();
		hr = E_INVALIDARG;
		return hr;
	}

	for (int i = 1; i < ArgCount - 1; i++)
	{
		wstring arg = ArgList[i];
		std::wcout << arg << std::endl;
		wcout << ArgList[i + 1] << endl;
		if (arg == LOG_PATH_CMDLINE_FLAG)
		{
			i++;
			LogPath = ArgList[i];
		} else if (arg == NAME_CMDLINE_FLAG)
		{
			i++;
			Name = ArgList[i];
		} else if (arg == TYPE_CMDLINE_FLAG)
		{
			i++;
			Type = ArgList[i];
		} else if (arg == OPERATION_CMDLINE_FLAG)
		{
			i++;
			Operation = ArgList[i];
		} else if (arg == IP_CMDLINE_FLAG)
		{
			i++;
			IpStr = ArgList[i];
		} else if (arg == DNS_CMDLINE_FLAG)
		{
			i++;
			DnsStr = ArgList[i];
		} else if (arg == GATEWAY_CMD_LINE_FLAG)
		{
			i++;
			GatewayStr = ArgList[i];
		} else 
		{
			PrintUsage();
			hr = E_INVALIDARG;
			return hr;
		}
	}

	if (Name.empty() || 
		Type.empty() || 
		Operation.empty())
	{
		PrintUsage();
		hr = E_INVALIDARG;
		return hr;
	}

	if (Type == L"Anteres" && 
		Operation == L"CREATE" &&
		IpStr.empty())
	{
		PrintUsage();
		hr = E_INVALIDARG;
		return hr;
	}

	return hr;
}
/*++

Routine Description:

This is the entry point for creating network container.

Arguments:

	argc: Argument count.
	argv: List of command line arguments.

--*/
int wmain(int argc, wchar_t *argv[])
{
	HRESULT hr = S_OK;

	wstring logPath, name, operation, type, ipStr, dnsStr, gatewayStr;
	hr = ParseCmdArguments(argc, 
		argv, 
		logPath,
		name,
		operation,
		type, 
		ipStr, 
		dnsStr, 
		gatewayStr);

	if (logPath.empty()) { logPath = DEFAULT_LOG_LOCATION; }
	SetupLogging(logPath.c_str());
	LOGINFOW(L"Setup logging succeeded.");
	
	wcout << "logPath: " << logPath << endl;
	wcout << "name: " << name << endl;
	wcout << "operation: " << operation << endl;
	wcout << "type: " << type << endl;
	wcout << "ip: " << ipStr << endl;
	wcout << "dns: " << dnsStr << endl;
	wcout << "gateway: " << gatewayStr << endl;

	GUID instanceGuid;
	wchar_t* InterfaceName = L"loopback1";

	NetworkContainer nc;
	hr = nc.CreateLoopbackAdapter(InterfaceName, &instanceGuid);
	//std::wcout << InterfaceName << std::endl;
	
	return hr;
}