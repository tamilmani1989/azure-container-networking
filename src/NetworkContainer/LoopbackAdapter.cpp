/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

LoopbackAdapterInstaller.cpp

Abstract:

Implements methods to manipulate loopback adapter.

--*/
#include "LoopbackAdapter.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

LoopbackAdapter::LoopbackAdapter(
	_In_ wstring Name,
	_In_ IN_ADDR Ip,
	_In_ IN_ADDR Netmask,
	_In_ IN_ADDR Dns,
	_In_ IN_ADDR Gateway
)
/*++

Routine Description:

Default onstructor for LoopbackAdapter. This constructor will enable default settings.

Return Value:

An instance of LoopbackAdapter object.

--*/
{
	displayName = Name;
	ip = Ip;
	netmask = Netmask;
	dns = Dns;
	gateway = Gateway;
}

LoopbackAdapter::~LoopbackAdapter()
/*++

Routine Description:

Destructor for LoopbackAdapter. This destructor will free memory allocated by network container.

Return Value:

None.

--*/

{
	Delete();
	STR_FREE(loopbackInfPath);
}

HRESULT LoopbackAdapter::GetNetworkParameters()
/*++

Routine Description:

	

--*/
{
	HRESULT hr = S_OK;
	FIXED_INFO * FixedInfo;
	ULONG    ulOutBufLen;
	DWORD    dwRetVal;
	IP_ADDR_STRING * pIPAddr;

	FixedInfo = (FIXED_INFO *)GlobalAlloc(GPTR, sizeof(FIXED_INFO));
	ulOutBufLen = sizeof(FIXED_INFO);

	if (ERROR_BUFFER_OVERFLOW == GetNetworkParams(FixedInfo, &ulOutBufLen)) {
		GlobalFree(FixedInfo);
		FixedInfo = (FIXED_INFO *)GlobalAlloc(GPTR, ulOutBufLen);
	}

	dwRetVal = GetNetworkParams(FixedInfo, &ulOutBufLen);
	if (dwRetVal) 
	{
		LOGINFOW(L"Call to GetNetworkParams failed. Return Value: %08x\n", dwRetVal);
		return hr;
	} else 
	{
		LOGINFOW(L"Host Name: %ls\n", (wchar_t*)FixedInfo->HostName);
		LOGINFOW(L"Domain Name: %ls\n", (wchar_t*)FixedInfo->DomainName);

		LOGINFOW(L"DNS Servers:\n");
		LOGINFOW(L"\t%ls\n", (wchar_t*)FixedInfo->DnsServerList.IpAddress.String);

		pIPAddr = FixedInfo->DnsServerList.Next;
		while (pIPAddr) {
			LOGINFOW(L"\t%ls\n", (wchar_t*)pIPAddr->IpAddress.String);
			pIPAddr = pIPAddr->Next;
		}
	}
	
	return hr;
}

HRESULT LoopbackAdapter::Create()
/*++

Routine Description:

	This routine creates a minimal working loopback adapter using API's provided by SetupAPI.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/

{
	HRESULT hr = S_OK;
	wchar_t windowsDirectory[MAX_PATH] = { 0 };
	if (!GetWindowsDirectory(windowsDirectory, MAX_PATH))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(L"GetWindowsDirectory failed with error [%0x].", hr);
			return hr;
		}
	}

	DWORD error = ERROR_SUCCESS;
	//loopbackInfPath: C:\windows\INF\NetLoop.INF
	error = AppendStrings(&loopbackInfPath,
		2,
		windowsDirectory,
		NET_LOOPBACK_INF_RELATIVE_FILE_PATH);
	if (error != ERROR_SUCCESS)
	{
		LOGERRW(L"CreateLoopbackAdapter AppendStrings failed errorcode %d", error);
		hr = HRESULT_FROM_WIN32(error);
		return hr;
	}
	LOGINFOW(L"Using loop back inf %s", loopbackInfPath);

	GUID classGuid = { 0 };
	wchar_t className[MAX_PATH] = { 0 };
	// Retrieve class of the Netloop.INF file.
	if (!SetupDiGetINFClass(loopbackInfPath, &classGuid, className, MAX_PATH, NULL))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(L"SetupDiGetINFClass with inf [%s] failed. Error:[0x%x].", loopbackInfPath, hr);
			return hr;
		}
	}

	LPWSTR classGuidString = NULL;
	error = GuidToString(classGuid, &classGuidString);
	if (error != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(L"SetupDiGetINFClass GuidToString failed with error [%0x].", hr);
		}
	}
	LOGINFOW(L"ClassName: [%s], ClassGuid:[%s]", className, classGuidString);

	// Fill device information set.
	deviceInfoSet = SetupDiCreateDeviceInfoList(&classGuid, NULL);
	if (deviceInfoSet == INVALID_HANDLE_VALUE)
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(L"Failed to SetupDiCreateDeviceInfoList for class '%s'. Error:[0x%x]", classGuidString, hr);
			return hr;
		}
	}

	ZeroMemory(&deviceInfoData, sizeof(deviceInfoData));
	deviceInfoData.cbSize = sizeof(deviceInfoData);
	// Fill device information element and adds it as a new member to deviceInfoSet.
	if (!SetupDiCreateDeviceInfo(deviceInfoSet,
		className,
		&classGuid,
		NULL,
		NULL,
		DICD_GENERATE_ID,
		&deviceInfoData))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(L"Failed to SetupDiCreateDeviceInfo for class '%s'. Error:[0x%x]", classGuidString, hr);
			return hr;
		}
	}

	// Sets a PnP(plug and play) device property for the device.
	if (!SetupDiSetDeviceRegistryProperty(
		deviceInfoSet,
		&deviceInfoData,
		SPDRP_HARDWAREID,
		(const BYTE *)LOOPBACK_ADAPTER_HARDWARE_ID,
		(DWORD)(wcslen(LOOPBACK_ADAPTER_HARDWARE_ID) + 1 + 1) * sizeof(WCHAR)))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(L"Failed to SetupDiSetDeviceRegistryProperty for class '%s'. Error:[0x%x]", classGuidString, hr);
			return hr;
		}
	}

	// Registers the newly created device instance with the PnP manager.
	if (!SetupDiCallClassInstaller(DIF_REGISTERDEVICE, deviceInfoSet, &deviceInfoData))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(L"Failed to SetupDiSetDeviceRegistryProperty for class '%s'. Error:[0x%x]", classGuidString, hr);
			return hr;
		}
	}

	// Installs the driver for the loopback adapter.
	if (!UpdateDriverForPlugAndPlayDevices(
		NULL,
		LOOPBACK_ADAPTER_HARDWARE_ID,
		loopbackInfPath,
		NULL,
		false))
	{
		error = GetLastError();
		if (error == ERROR_NO_MORE_ITEMS)
		{
			LOGINFOW(
				L"UpdateDriverForPlugAndPlayDevices failed with error:[0x%x]. Ignoring it",
				error);
		}
		else
		{
			hr = HRESULT_FROM_WIN32(error);
			if (hr != S_OK)
			{
				LOGERRW(L"Failed to UpdateDriverForPlugAndPlayDevices for class '%s'. Error:[0x%x]", classGuidString, hr);
				return hr;
			}
		}
	}
	LOGINFOW(L"Loopback adapter successfully installed");

	BOOL setupDiGetDevRegPropResult = false;
	// Retrieve the PnP property of the device.
	wchar_t friendlyName[MAX_PATH];
	setupDiGetDevRegPropResult = SetupDiGetDeviceRegistryProperty(
		deviceInfoSet,
		&deviceInfoData,
		SPDRP_FRIENDLYNAME,
		NULL,
		(PBYTE)friendlyName, //PropertyBuffer
		sizeof(friendlyName),
		NULL);
	if (!setupDiGetDevRegPropResult)
	{
		LOGINFOW(
			L"SetupDiGetDeviceRegistryProperty failed when using SPDRP_FRIENDLYNAME. "
			L"Will retry using device description. Error:[0x%x]",
			GetLastError());

		if (!SetupDiGetDeviceRegistryProperty(
			deviceInfoSet,
			&deviceInfoData,
			SPDRP_DEVICEDESC,
			NULL,
			(PBYTE)friendlyName,
			sizeof(friendlyName),
			NULL))
		{
			hr = HRESULT_FROM_WIN32(GetLastError());
			if (hr != S_OK)
			{
				LOGERRW(L"SetupDiGetDeviceRegistryProperty failed using SPDRP_DEVICEDESC. Error:[0x%x]", hr);
				return hr;
			}
		}
	}

	LOGINFOW(L"Display name for the newly created loopback adapter:%s", friendlyName);
	hr = UpdateInterfaceDisplayName(friendlyName, (LPWSTR)displayName.c_str(), &instanceGuid);
	if (hr != S_OK)
	{
		LOGERRW(L"UpdateInterfaceDisplayName failed. Error:[0x%x]", hr);
		return hr;
	}

	memcpy(&instanceGuid, &instanceGuid, sizeof(GUID));
	LOGINFOW(L"Disabling the newly created interface with name:[%s]", displayName.c_str());
	hr = EnableInterface(false, deviceInfoSet, &deviceInfoData);
	if (hr != S_OK)
	{
		LOGERRW(L"Failed to disable interface with name [%s], Error:[0x%x]", displayName.c_str(), hr);
		return hr;
	}

	LOGINFOW(L"Enabling the newly created interface with name:[%s]", displayName.c_str());
	hr = EnableInterface(true, deviceInfoSet, &deviceInfoData);
	if (hr != S_OK)
	{
		LOGERRW(L"Failed to enable interface with name [%s], Error:[0x%x]", displayName.c_str(), hr);
		return hr;
	}

	hr = GetInterfaceIndex(instanceGuid, interfaceIndex);
	if (hr != S_OK)
	{
		LOGERRW(L"Failed to retrieve index for interface with name [%s], Error:[0x%x]", displayName.c_str(), hr);
		return hr;
	}

	LOGINFOW(L"Interface name:[%s], index:[%d]", displayName.c_str(), interfaceIndex);
	return hr;
}

HRESULT LoopbackAdapter::SetIp(_In_ IN_ADDR Ip)
/*++

Routine Description:

	This routine sets the ip of the loopback adapter.

Arguments:

	IP address in IN_ADDR format.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;

	UINT iaIPAddress = Ip.S_un.S_addr;
	UINT iaIPMask = netmask.S_un.S_addr;

	if (iaIPAddress == INADDR_NONE)
	{
		LOGERRW(L"Invalid ip address.");
		hr = E_INVALIDARG;
		return hr;
	}
	if (iaIPMask == INADDR_NONE)
	{
		LOGERRW(L"Invalid ip mask.");
		hr = E_INVALIDARG;
		return hr;
	}

	GetInterfaceIndex(instanceGuid, interfaceIndex);

	if (AddIPAddress(iaIPAddress,
		iaIPMask,
		interfaceIndex,
		&NTEContext,
		&NTEInstance) != NO_ERROR)
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(L"AddIPAddress failed with Error:[0x%x]", hr);
			return hr;
		}
	}

	LOGINFOW(L"Loopback adapter ip set.");
	return hr;
}

HRESULT LoopbackAdapter::SetNetmask(_In_ IN_ADDR Netmask)
/*++

Routine Description:

	This routine sets the netmask for the adapter.

Arguments:

	Netmask in IN_ADDR format.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;

	char ipCharArray[INET_ADDRSTRLEN], nmCharArray[INET_ADDRSTRLEN];
	inet_ntop(AF_INET, (void*)&ip, ipCharArray, sizeof(ipCharArray));
	inet_ntop(AF_INET, (void*)&Netmask, nmCharArray, sizeof(nmCharArray));
	wstring ipStr, gwStr;
	copy(ipCharArray, ipCharArray + strlen(ipCharArray), back_inserter(ipStr));
	copy(nmCharArray, nmCharArray + strlen(nmCharArray), back_inserter(gwStr));
	wstring netshCommand = L"netsh interface ip set address name=" +
		to_wstring(interfaceIndex) +
		L" static addr=" +
		ipStr +
		L" mask=" +
		gwStr;

	LOGINFOW(L"Running cmd %ls", netshCommand.c_str());
	DWORD exitCode = 0;
	hr = InvokeSystemCommand(netshCommand.c_str(), &exitCode, INFINITE, false);
	if (hr != S_OK)
	{
		LOGERRW(L"Command failed ('%ls')", netshCommand.c_str());
		return hr;
	}

	if (exitCode != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(exitCode);
		LOGERRW(L"Command returned non zero exit code %u ('%ls').", exitCode, netshCommand.c_str());
		return hr;
	}

	return hr;
}

HRESULT LoopbackAdapter::SetGateway(_In_ IN_ADDR Gateway)
/*++

Routine Description:

	This routine sets the gateway for the adapter.

Arguments:

	DNS address in IN_ADDR format.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;

	char ipCharArray[INET_ADDRSTRLEN], gwCharArray[INET_ADDRSTRLEN];
	inet_ntop(AF_INET, (void*)&ip, ipCharArray, sizeof(ipCharArray));
	inet_ntop(AF_INET, (void*)&Gateway, gwCharArray, sizeof(gwCharArray));
	wstring ipStr, gwStr;
	copy(ipCharArray, ipCharArray + strlen(ipCharArray), back_inserter(ipStr));
	copy(gwCharArray, gwCharArray + strlen(gwCharArray), back_inserter(gwStr));
	wstring netshCommand = L"netsh interface ip set address name=" +
		to_wstring(interfaceIndex) +
		L" static addr=" +
		ipStr +
		L" gateway=" +
		gwStr;

	LOGINFOW(L"Running cmd %ls", netshCommand.c_str());
	DWORD exitCode = 0;
	hr = InvokeSystemCommand(netshCommand.c_str(), &exitCode, INFINITE, false);
	if (hr != S_OK)
	{
		LOGERRW(L"Command failed ('%ls')", netshCommand.c_str());
		return hr;
	}

	if (exitCode != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(exitCode);
		LOGERRW(L"Command returned non zero exit code %u ('%ls').", exitCode, netshCommand.c_str());
		return hr;
	}

	return hr;
}

HRESULT LoopbackAdapter::SetDns(_In_ IN_ADDR Dns)
/*++

Routine Description:

	This routine sets the dns for the adapter.

Arguments:

	DNS address in IN_ADDR format.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;

	char dnsCharArray[INET_ADDRSTRLEN];
	inet_ntop(AF_INET, (void *)&Dns, dnsCharArray, sizeof(dnsCharArray));
	wstring dnsStr;
	copy(dnsCharArray, dnsCharArray + strlen(dnsCharArray), std::back_inserter(dnsStr));
	wstring netshCommand = L"netsh interface ip set dns " +
		to_wstring(interfaceIndex) +
		L" static " +
		dnsStr;
	
	LOGINFOW(L"Running cmd %ls", netshCommand.c_str());
	DWORD exitCode = 0;
	hr = InvokeSystemCommand(netshCommand.c_str(), &exitCode, INFINITE, false);
	if (hr != S_OK)
	{
		LOGERRW(L"Command failed ('%ls')", netshCommand.c_str());
		return hr;
	}

	if (exitCode != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(exitCode);
		LOGERRW(L"Command returned non zero exit code %u ('%ls').", exitCode, netshCommand.c_str());
		return hr;
	}

	return hr;
}

HRESULT LoopbackAdapter::Delete()
{
/*++

Routine Description:

	This routine deletes the loopback adapter using API's provided by SetupAPI.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/

	HRESULT hr = S_OK;

	if (!SetupDiCallClassInstaller(DIF_REMOVE, deviceInfoSet, &deviceInfoData))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(L"Failed to SetupDiSetDeviceRegistryProperty during loopback adapter [%ls] deletion. Error:[0x%x]", displayName.c_str(), hr);
		}
	}

	return hr;
}