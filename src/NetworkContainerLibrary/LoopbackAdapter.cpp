/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

	LoopbackAdapterInstaller.cpp

Abstract:

	Implements methods to manipulate loopback adapter.

Author:

	Yongli Chen

--*/
#include "LoopbackAdapter.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

LoopbackAdapter::LoopbackAdapter(
	_In_ wstring Name,
	_In_ IN_ADDR Ip,
	_In_ IN_ADDR Netmask,
	_In_ IN_ADDR Gateway,
	_In_ vector<IN_ADDR> DnsVec,
	_In_ bool WhsFlag,
	_In_ bool WhrFlag
)
/*++

Routine Description:

	Default onstructor for LoopbackAdapter. This constructor will enable default settings.

Arguments:

	Name: Name of the adapter.

	Ip: Ip address to be assigned, in IN_ADDR format.

	Netmask: Netmask associated with Ip, in IN_ADDR format.

	Gateway: Gateway address to be assigned, in IN_ADDR format.

	DnsVec:  A list of DNS address to be assigned, in vector<IN_ADDR> format.

Return Value:

	An instance of LoopbackAdapter object.

--*/
{
	displayName = Name;
	ip = Ip;
	netmask = Netmask;
	dnsVec = DnsVec;
	gateway = Gateway;
	whsFlag = WhsFlag;
	whrFlag = WhrFlag;
}

LoopbackAdapter::~LoopbackAdapter()
/*++

Routine Description:

	Destructor for LoopbackAdapter. This destructor will free memory allocated by network container.

Return Value:

	None.

--*/
{
	STR_FREE(loopbackInfPath);
	if (deviceInfoSet != INVALID_HANDLE_VALUE)
	{
		SetupDiDestroyDeviceInfoList(deviceInfoSet);
	}
}

bool LoopbackAdapter::isNumber(_In_ wstring Name) 
{
	wchar_t* p;
	wcstol(Name.c_str(), &p, 10);
	if (*p)
	{
		return false;
	}
	return true;
}

HRESULT LoopbackAdapter::FindAdapterByDisplayName(_In_ wstring Name, _Out_ int& InstanceIndex)
/*++

Routine Description:

	This routine finds an adapter by its display name.

Argument:

	Name: Display name of the adapter.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;

	if (isNumber(Name))
	{
		hr = FindAdapterByIndex(Name);
		if (hr == S_OK)
		{
			InstanceIndex = stoi(Name);
		}
		return hr;
	}

	INetCfg* netCfg = NULL;
	INetCfgClass* netCfgClass = NULL;
	IEnumNetCfgComponent* netCfgComponentEnum = NULL;
	INetCfgComponent* netCfgComponent = NULL;
	LPWSTR interfaceDisplayName = NULL;
	GUID instanceGuid = { 0 };
	bool updated = false;

	hr = GetINetCfgWithWriteLock(&netCfg);
	if (hr != S_OK)
	{
		LOGERRW(L"GetINetCfgWithWriteLock failed with error [%0x].", hr);
		goto Cleanup;
	}

	hr = netCfg->QueryNetCfgClass(
		&GUID_DEVCLASS_NET,
		IID_INetCfgClass,
		(void**)&netCfgClass);
	if (hr != S_OK)
	{
		LOGERRW(L"QueryNetCfgClass failed with error [%0x].", hr);
		goto Cleanup;
	}

	hr = netCfgClass->EnumComponents(&netCfgComponentEnum);
	if (hr != S_OK)
	{
		LOGERRW(L"EnumComponents failed with error [%0x].", hr);
		goto Cleanup;
	}

	while (S_OK == (hr = netCfgComponentEnum->Next(1, &netCfgComponent, NULL)))
	{
		hr = netCfgComponent->GetDisplayName(&interfaceDisplayName);
		if (FAILED(hr))
		{
			LOGINFOW(L"Failed to retrieve interface name. Skipping interface");
			hr = S_OK;
			continue;
		}

		if (Name == interfaceDisplayName)
		{
			LOGINFOW(L"Found matching interface. Name [%s]", interfaceDisplayName);
			hr = netCfgComponent->GetInstanceGuid(&instanceGuid);
			if (hr != S_OK)
			{
				LOGERRW(L"GetInstanceGuid for intface name %s failed with error [%0x].", interfaceDisplayName, hr);
				goto Cleanup;
			}

			updated = true;
			break;
		}

		RELEASE(netCfgComponent);
		CO_FREE(interfaceDisplayName);
	}

	if (hr == S_FALSE && !updated)
	{
		hr = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
		LOGERRW(L"GetInterfaceByDisplayName: Failed to find interface with name %s", Name.c_str());
		goto Cleanup;
	}

	hr = GetInterfaceIndex(instanceGuid, InstanceIndex);
	if (hr != S_OK)
	{
		LOGERRW(L"Failed to retrieve index for interface with name [%s], Error:[0x%x]", Name.c_str(), hr);
		goto Cleanup;
	}

	LOGINFOW(L"Interface name:[%s], index:[%d]", Name.c_str(), InstanceIndex);

Cleanup:

	RELEASE(netCfgComponent);
	RELEASE(netCfgComponentEnum);
	RELEASE(netCfgClass);
	CO_FREE(interfaceDisplayName);

	ReleaseINetCfgWithLock(netCfg);
	return hr;
}

HRESULT LoopbackAdapter::FindAdapterByIndex(wstring Index)
/*++

Routine Description:

	This routine finds an adapter by its index.

Argument:

	Index: Index of the adapter.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;

	PIP_ADAPTER_INFO pAdapterInfo;
	PIP_ADAPTER_INFO pAdapter = NULL;
	DWORD dwRetVal = 0;

	ULONG ulOutBufLen = sizeof(IP_ADAPTER_INFO);
	pAdapterInfo = (IP_ADAPTER_INFO *)malloc(sizeof(IP_ADAPTER_INFO));
	if (pAdapterInfo == NULL) 
	{
		LOGINFOW(L"Error allocating memory needed to call GetAdaptersinfo\n");
		hr = E_ABORT;
		return hr;
	}

	if (GetAdaptersInfo(pAdapterInfo, &ulOutBufLen) == ERROR_BUFFER_OVERFLOW) 
	{
		free(pAdapterInfo);
		pAdapterInfo = (IP_ADAPTER_INFO *)malloc(ulOutBufLen);
		if (pAdapterInfo == NULL) 
		{
			LOGINFOW(L"Error allocating memory needed to call GetAdaptersinfo\n");
			hr = E_ABORT;
			return hr;
		}
	}

	if ((dwRetVal = GetAdaptersInfo(pAdapterInfo, &ulOutBufLen)) == NO_ERROR)
	{
		pAdapter = pAdapterInfo;
		while (pAdapter)
		{
			if (pAdapter->Index == (DWORD)stoi(Index))
			{
				LOGINFOW(L"Found matching interface. Index [%d]", pAdapter->Index);
				break;
			}
			pAdapter = pAdapter->Next;
		}
	}

	if (pAdapterInfo)
	{
		free(pAdapterInfo);
	}

	return hr;
}

HRESULT LoopbackAdapter::GetNetworkParameters()
/*++

Routine Description:

	Print network parameters to log file.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

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
	} 
	else 
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
	
	GlobalFree(FixedInfo);
	return hr;
}

HRESULT LoopbackAdapter::FillDevInfo(_In_ int InstanceIndex)
/*++
Routine Description:

	This routine fills the requested device information.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;
	interfaceIndex = InstanceIndex;

	deviceInfoSet = SetupDiGetClassDevs(
		&GUID_DEVCLASS_NET,
		NULL,
		NULL,
		DIGCF_PRESENT);
	if (deviceInfoSet == INVALID_HANDLE_VALUE)
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(L"SetupDiGetClassDevs failed with error [%0x].", hr);
			return hr;
		}
	}

	ZeroMemory(&deviceInfoData, sizeof(SP_DEVINFO_DATA));
	deviceInfoData.cbSize = sizeof(SP_DEVINFO_DATA);

	wchar_t friendlyName[MAX_PATH];
	BOOL setupDiGetDevRegPropResult = false;

	for (int i = 0;
		SetupDiEnumDeviceInfo(deviceInfoSet, i, &deviceInfoData);
		i++)
	{
		setupDiGetDevRegPropResult = SetupDiGetDeviceRegistryProperty(
			deviceInfoSet,
			&deviceInfoData,
			SPDRP_FRIENDLYNAME,
			NULL,
			(PBYTE)friendlyName,
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
		if (wcsncmp(friendlyName, displayName.c_str(), wcslen(friendlyName)) == 0)
		{
			LOGINFOW(L"Filled device information for %s\n", friendlyName);
			return hr;
		}
	}

	hr = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
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
	
	hr = Rename();
	if (hr != S_OK)
	{
		LOGERRW(L"Rename failed. Error:[0x%x]", hr);
	}

	LOGINFOW(L"Interface name:[%s], index:[%d]", displayName.c_str(), interfaceIndex);
	return hr;
}

HRESULT LoopbackAdapter::Rename()
{
	HRESULT hr = S_OK;
	wstring netshCommand = L"netsh interface set interface name=\"" +
		alias +
		L"\" newname=\"" +
		displayName +
		L"\"";

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

	LOGINFOW(L"Netmask set successfully.");
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
			LOGERRW(L"Failed to delete loopback adapter [%ls] with error. Error:[0x%x]", displayName.c_str(), hr);
		}
	}

	// Make sure the adapter is removed.
	while (FindAdapterByDisplayName(displayName, interfaceIndex) != HRESULT_FROM_WIN32(ERROR_NOT_FOUND))
	{
		Sleep(100);
	}

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

	LOGINFOW(L"IP set successfully.");
	return hr;
}

HRESULT LoopbackAdapter::SetDns(_In_ vector<IN_ADDR> DnsVec)
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
	int idx = 0;
	wstring netshCommand;

	for (auto it = DnsVec.begin(); it != DnsVec.end(); it++)
	{
		idx++;
		char dnsCharArray[INET_ADDRSTRLEN];
		inet_ntop(AF_INET, (void *)&(*it), dnsCharArray, sizeof(dnsCharArray));
		wstring dnsStr;
		copy(dnsCharArray, dnsCharArray + strlen(dnsCharArray), std::back_inserter(dnsStr));
		if (idx == 1)
		{
			netshCommand = L"netsh interface ip set dns " +
				to_wstring(interfaceIndex) +
				L" static " +
				dnsStr;
		} else 
		{
			netshCommand = L"netsh interface ip add dnsservers " +
				to_wstring(interfaceIndex) + L" " +
				dnsStr + L" " +
				to_wstring(idx);
		}
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

		LOGINFOW(L"DNS set successfully.");
	}
	return hr;
}

HRESULT LoopbackAdapter::SetWeakHost(bool WhsFlag, bool WhrFlag)
/*++
	
Routine Description:

	Enable/Disable weak host routing on the interface.

Arguments:

	WhsFlag: Flag for weak host send.

	WhrFlag: Flag for weak host receive.

--*/
{
	HRESULT hr = S_OK;
	wstring whs = WhsFlag ? L"enabled" : L"disabled";
	wstring whr = WhrFlag ? L"enabled" : L"disabled";
	wstring netshCommand = L"netsh interface ip set interface " +
		to_wstring(interfaceIndex) +
		L" weakhostsend=" +
		whs +
		L" weakhostreceive=" +
		whr;

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

	LOGINFOW(L"weak host routing set successfully.");
	return hr;
}

HRESULT LoopbackAdapter::SetNetwork(
	_In_ IN_ADDR Ip,
	_In_ IN_ADDR Netmask,
	_In_ IN_ADDR Gateway
)
/*++

Routine Description:

	Configure ip, netmask and gateway on the interface.

Arguments:

	Ip: IP address of the interface.

	Netmask: Netmask of the interface.

	Gateway: Gateway of the interface.

--*/
{
	HRESULT hr = S_OK;
	char ipCharArray[INET_ADDRSTRLEN], nmCharArray[INET_ADDRSTRLEN], gwCharArray[INET_ADDRSTRLEN];
	inet_ntop(AF_INET, (void*)&Ip, ipCharArray, sizeof(ipCharArray));
	inet_ntop(AF_INET, (void*)&Netmask, nmCharArray, sizeof(nmCharArray));
	inet_ntop(AF_INET, (void*)&Gateway, gwCharArray, sizeof(gwCharArray));
	wstring ipStr, nmStr, gwStr;
	copy(ipCharArray, ipCharArray + strlen(ipCharArray), back_inserter(ipStr));
	copy(nmCharArray, nmCharArray + strlen(nmCharArray), back_inserter(nmStr));
	copy(gwCharArray, gwCharArray + strlen(gwCharArray), back_inserter(gwStr));
	wstring netshCommand = L"netsh interface ip set address name=" +
		to_wstring(interfaceIndex) +
		L" static addr=" +
		ipStr;

	if (Netmask.S_un.S_addr) 
	{
		netshCommand += L" mask=";
		netshCommand += nmStr;
	}

	if (Gateway.S_un.S_addr)
	{
		netshCommand += L" gateway=";
		netshCommand += gwStr;
	}
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

	LOGINFOW(L"network parameters set successfully.");

	return hr;
}

void LoopbackAdapter::SetIndex(_In_ int InterfaceIndex)
/*++

Routine Description:

	This routine sets interface index for the loop adapter.

Argument:

	InterfaceIndex: Unique index of the interface.

--*/
{
	interfaceIndex = InterfaceIndex;
}
