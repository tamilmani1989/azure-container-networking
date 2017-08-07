/*++

Copyright (c) 2008  Microsoft Corporation

Module Name:

LoopbackAdapterInstaller.cpp

Abstract:

Implements methods to create and remove adapter.

--*/
#include "LoopbackAdapterInstaller.h"
#pragma warning ( disable : 4127 )

HRESULT
LoopbackAdapterInstaller::GetINetCfgWithWriteLock(
	__deref_out INetCfg** NetCfg
)
/*++

Routine Description:

Private helper routine to get a INetCfg instance.

Arguments:

NetCfg - Pointer to instance of INetCfg to hold the retrieved instance.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;
	INetCfg* netCfg;
	bool lockAcquired = false;

	*NetCfg = NULL;

	hr = CoCreateInstance(CLSID_CNetCfg, NULL, CLSCTX_INPROC_SERVER,
		IID_INetCfg, (void**)&netCfg);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to create instance of INetCfg"),
		hr,
		Cleanup);

	hr = AcquireNetCfgWriteLock(netCfg);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to acquire NetCfgWriteLock"),
		hr,
		Cleanup);

	lockAcquired = true;

	hr = netCfg->Initialize(NULL);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to initialize INetCfg instance"),
		hr,
		Cleanup);

	*NetCfg = netCfg;
	netCfg = NULL;

Cleanup:

	if (lockAcquired && !SUCCEEDED(hr))
	{
		ReleaseNetCfgWriteLock(netCfg);
	}

	RELEASE(netCfg);

	return hr;
}


HRESULT
LoopbackAdapterInstaller::AcquireNetCfgWriteLock(
	__in INetCfg *NetCfg
)
/*++

Routine Description:

Private helper routine to acquire write lock on INetCfg instance.

Arguments:

NetCfg - Pointer to instance of INetCfg.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	INetCfgLock *netCfgLock = NULL;
	PWSTR lockedBy = NULL;
	HRESULT hr = S_OK;

	RD_TRACE_STATIC_ENTRY();

	if (NetCfg == NULL) {
		RD_E_WIN32(ERROR_INVALID_PARAMETER, Cleanup);
	}

	hr = NetCfg->QueryInterface(IID_INetCfgLock, (LPVOID *)&netCfgLock);
	RD_E_HR(hr, Cleanup);

	//
	// Attempt to acquire write lock on INetCfg instance.
	//

	hr = netCfgLock->AcquireWriteLock(INFINITE,
		NET_LOOPBACK_LOCK_ID,
		&lockedBy);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to acquire write lock on INetCfg"),
		hr,
		Cleanup);

Cleanup:

	RELEASE(netCfgLock);

	RD_TRACE_STATIC_EXIT(hr);
	return hr;
}

HRESULT
LoopbackAdapterInstaller::ReleaseNetCfgWriteLock(
	__in INetCfg *NetCfg
)
/*++

Routine Description:

Private helper routine to release write lock on INetCfg instance.

Arguments:

NetCfg - Pointer to instance of INetCfg.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	INetCfgLock *netCfgLock = NULL;
	HRESULT hr = S_OK;

	RD_TRACE_STATIC_ENTRY();

	hr = NetCfg->QueryInterface(IID_INetCfgLock, (LPVOID *)&netCfgLock);
	RD_E_HR(hr, Cleanup);

	hr = netCfgLock->ReleaseWriteLock();
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to release write lock on INetCfg instance"),
		hr,
		Cleanup);

Cleanup:

	RELEASE(netCfgLock);

	RD_TRACE_STATIC_EXIT(hr);
	return hr;
}

HRESULT
LoopbackAdapterInstaller::ReleaseINetCfgWithLock(
	__in INetCfg* NetCfg
)
/*++

Routine Description:

Private helper routine to release netcfg instance.

Arguments:

NetCfg - Pointer to instance of INetCfg to hold the retrieved instance.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;
	if (NetCfg == NULL)
	{
		return S_OK;
	}

	hr = NetCfg->Uninitialize();

	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to uninitialize netcfg instance"),
		hr,
		Cleanup);

Cleanup:

	ReleaseNetCfgWriteLock(NetCfg);

	RELEASE(NetCfg);

	return hr;
}

HRESULT
LoopbackAdapterInstaller::UpdateInterfaceDisplayName(
	__in LPWSTR InterfaceName,
	__in LPWSTR InterfaceNewName,
	__out LPGUID InstanceGuid
)
/*++

Routine Description:

This routine updates the display name for a given interface.

Arguments:

InterfaceName - Current display name of the interface.

InterfaceNewName - New display name for the interface.

InstanceGuid - Long Pointer to the GUID. Must free memory using delete call once
you are done with it.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;
	INetCfg* netCfg = NULL;
	INetCfgClass* netCfgClass = NULL;
	IEnumNetCfgComponent* netCfgComponentEnum = NULL;
	INetCfgComponent* netCfgComponent = NULL;
	LPWSTR interfaceDisplayName = NULL;
	GUID instanceGuid = { 0 };
	bool updated = false;
	*InstanceGuid = { 0 };

	hr = GetINetCfgWithWriteLock(&netCfg);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to initialize INetCfg instance. Error[%0x]", hr),
		hr,
		Cleanup);

	hr = netCfg->QueryNetCfgClass(
		&GUID_DEVCLASS_NET,
		IID_INetCfgClass,
		(void**)&netCfgClass);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to query netcfg class Error:[%0x]", hr),
		hr,
		Cleanup);

	hr = netCfgClass->EnumComponents(&netCfgComponentEnum);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to enumerate netcfg components. Error[%0x]", hr),
		hr,
		Cleanup);

	while (S_OK == (hr = netCfgComponentEnum->Next(1, &netCfgComponent, NULL)))
	{
		hr = netCfgComponent->GetDisplayName(&interfaceDisplayName);
		if (FAILED(hr))
		{
			RD_TRACE_LOG((RD_TRACE_INFO, L"Failed to retrieve interface name. Skipping interface"));
			hr = S_OK;
			continue;
		}

		RD_TRACE_LOG((RD_TRACE_INFO, L"Retrieved interface with name [%s]", interfaceDisplayName));
		if (wcsncmp(InterfaceName, interfaceDisplayName, wcslen(InterfaceName)) == 0)
		{
			RD_TRACE_LOG((RD_TRACE_INFO, L"Found matching interface. Name [%s]", interfaceDisplayName));
			if (wcsncmp(InterfaceName, InterfaceNewName, wcslen(InterfaceName)) != 0)
			{
				RD_TRACE_LOG((RD_TRACE_INFO, L"Updating interface name from [%s] to [%s].",
					interfaceDisplayName,
					InterfaceNewName));

				hr = netCfgComponent->SetDisplayName(InterfaceNewName);
				RD_TRACE_E_HR((RD_TRACE_ERROR,
					L"Failed to set display name for interface %s. Error[%0x]", interfaceDisplayName, hr),
					hr,
					Cleanup);
			}
			else
			{
				RD_TRACE_LOG((RD_TRACE_INFO,
					L"Skipping updating interface name as the current and requested names are same. "
					L"Interface Name[%s]", interfaceDisplayName));
			}

			hr = netCfgComponent->GetInstanceGuid(&instanceGuid);
			RD_TRACE_E_HR((RD_TRACE_ERROR,
				L"Failed to retrieve interface Id for interface name %s. Error[%0x]", interfaceDisplayName, hr),
				hr,
				Cleanup);
			memcpy(InstanceGuid, &instanceGuid, sizeof(GUID));

			updated = true;
			break;
		}

		RELEASE(netCfgComponent);
		CO_FREE(interfaceDisplayName);
	}

	if (hr == S_FALSE && !updated)
	{
		hr = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"GetInterfaceByDisplayName: Failed to find interface with name %s", InterfaceName),
			hr,
			Cleanup);
	}

Cleanup:

	RELEASE(netCfgComponent);
	RELEASE(netCfgComponentEnum);
	RELEASE(netCfgClass);
	CO_FREE(interfaceDisplayName);

	ReleaseINetCfgWithLock(netCfg);

	return hr;

}

HRESULT
LoopbackAdapterInstaller::CreateLoopbackAdapter(
	__in LPWSTR InterfaceName,
	__inout LPGUID InstanceGuid
)
/*++

Routine Description:

This routine creates a loopback adapter using API's provided by SetipAPI.

Arguments:

InterfaceName - Current display name of the interface.

InstanceGuid - Out parameter to hold the instance guid associated with the newly created adapter.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;
	wchar_t windowsDirectory[MAX_PATH] = { 0 };
	wchar_t className[MAX_PATH] = { 0 };
	LPWSTR loopbackInfPath = NULL;
	DWORD error = ERROR_SUCCESS;
	HDEVINFO deviceInfoSet = INVALID_HANDLE_VALUE;
	SP_DEVINFO_DATA deviceInfoData;
	LPWSTR classGuidString = NULL;
	GUID classGuid = { 0 };
	GUID instanceGuid = { 0 };
	wchar_t friendlyName[MAX_PATH];
	BOOL setupDiGetDevRegPropResult = false;

	if (!GetWindowsDirectory(windowsDirectory, MAX_PATH))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"Failed to get windows directory. Error[%x0x]", hr),
			hr,
			Cleanup);
	}

	error = AppendStrings(&loopbackInfPath,
		2,
		windowsDirectory,
		NET_LOOPBACK_INF_RELATIVE_FILE_PATH);
	RD_E_WIN32(error, Cleanup);

	RD_TRACE_LOG((RD_TRACE_INFO, L"Using loop back inf %s", loopbackInfPath));

	if (!SetupDiGetINFClass(loopbackInfPath, &classGuid, className, MAX_PATH, NULL))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"SetupDiGetINFClass with inf [%s] failed. Error:[0x%x]",
			loopbackInfPath,
			hr),
			hr,
			Cleanup);
	}

	error = GuidToString(classGuid, &classGuidString);
	if (ERROR_SUCCESS != error)
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"SetupDiGetINFClass GuidToString - failed. Error:[0x%x]",
			error),
			hr,
			Cleanup);
	}

	RD_TRACE_LOG((RD_TRACE_INFO, L"ClassName %s, ClassGuid:[%s]", className, classGuidString));

	deviceInfoSet = SetupDiCreateDeviceInfoList(&classGuid, NULL);
	if (deviceInfoSet == INVALID_HANDLE_VALUE)
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"Failed to SetupDiCreateDeviceInfoList for class '%s'. Error:[0x%x]",
			classGuidString,
			hr),
			hr,
			Cleanup);
	}

	ZeroMemory(&deviceInfoData, sizeof(deviceInfoData));
	deviceInfoData.cbSize = sizeof(deviceInfoData);

	if (!SetupDiCreateDeviceInfo(deviceInfoSet,
		className,
		&classGuid,
		NULL,
		NULL,
		DICD_GENERATE_ID,
		&deviceInfoData))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"Failed to SetupDiCreateDeviceInfo for class '%s'. Error:[0x%x]",
			classGuidString,
			hr),
			hr,
			Cleanup);
	}

	if (!SetupDiSetDeviceRegistryProperty(
		deviceInfoSet,
		&deviceInfoData,
		SPDRP_HARDWAREID,
		(const BYTE *)LOOPBACK_ADAPTER_HARDWARE_ID,
		(DWORD)(wcslen(LOOPBACK_ADAPTER_HARDWARE_ID) + 1 + 1) * sizeof(WCHAR)))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"Failed to SetupDiSetDeviceRegistryProperty for class '%s'. Error:[0x%x]",
			classGuidString,
			hr),
			hr,
			Cleanup);
	}

	if (!SetupDiCallClassInstaller(DIF_REGISTERDEVICE, deviceInfoSet, &deviceInfoData))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"Failed to SetupDiCallClassInstaller for class '%s'. Error:[0x%x]",
			classGuidString,
			hr),
			hr,
			Cleanup);
	}

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
			RD_TRACE_LOG((RD_TRACE_WARNING,
				L"UpdateDriverForPlugAndPlayDevices failed with error:[0x%x]. Ignoring it",
				error));
		}
		else
		{
			hr = HRESULT_FROM_WIN32(error);
			RD_TRACE_E_HR((RD_TRACE_ERROR,
				L"Failed to UpdateDriverForPlugAndPlayDevices for class '%s', Error:[0x%x]",
				classGuidString,
				hr),
				hr,
				Cleanup);
		}
	}

	RD_TRACE_LOG((RD_TRACE_INFO, L"Loopback adapter successfully installed"));

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
		RD_TRACE_LOG((RD_TRACE_WARNING,
			L"SetupDiGetDeviceRegistryProperty failed when using SPDRP_FRIENDLYNAME. "
			L"Will retry using device description. Error:[0x%x]",
			GetLastError()));

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
			RD_TRACE_E_HR((RD_TRACE_ERROR,
				L"SetupDiGetDeviceRegistryProperty failed using SPDRP_DEVICEDESC. Error:[0x%x]", hr),
				hr,
				Cleanup);
		}
	}

	RD_TRACE_LOG((RD_TRACE_INFO, L"Display name for the newly created loopback adapter:%s", friendlyName));

	hr = UpdateInterfaceDisplayName(friendlyName, InterfaceName, &instanceGuid);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"UpdateInterfaceDisplayName failed. Error:[0x%x]", hr),
		hr,
		Cleanup);

	memcpy(InstanceGuid, &instanceGuid, sizeof(GUID));

	RD_TRACE_LOG((RD_TRACE_INFO, L"Disabling the newly created interface with name:[%s]", InterfaceName));
	hr = EnableInterface(false, deviceInfoSet, &deviceInfoData);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to disable interface with name [%s], Error:[0x%x]",
		InterfaceName,
		hr),
		hr,
		Cleanup);

	RD_TRACE_LOG((RD_TRACE_INFO, L"Enabling the newly created interface with name:[%s]", InterfaceName));
	EnableInterface(true, deviceInfoSet, &deviceInfoData);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to enable interface with name [%s], Error:[0x%x]",
		InterfaceName,
		hr),
		hr,
		Cleanup);

Cleanup:

	STR_FREE(loopbackInfPath);
	STR_FREE(classGuidString);

	if (deviceInfoSet != INVALID_HANDLE_VALUE)
	{
		SetupDiDestroyDeviceInfoList(deviceInfoSet);
	}

	return hr;
}

HRESULT
LoopbackAdapterInstaller::GetOrCreateLoopbackAdapter(
	__in LPWSTR InterfaceName,
	__inout int& InterfaceIndex
)
/*++

Routine Description:

This routine creates a loopback adapter using API's provided by SetipAPI.

Arguments:

InterfaceName - Name of the loopback adapter to be queried or created.

InterfaceIndex - Out parameter to hold the interface index associated with the adapter.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;
	GUID classGuid = { 0 };
	WCHAR windowsDirectory[MAX_PATH] = { 0 };
	WCHAR className[MAX_PATH] = { 0 };
	PWSTR loopbackInfPath = NULL;
	DWORD error = ERROR_SUCCESS;
	PWSTR classGuidString = NULL;
	int index;
	wstring interfaceDisplayName;
	GUID instanceGuid;
	hr = UpdateInterfaceDisplayName(InterfaceName, InterfaceName, &instanceGuid);
	if (hr == S_OK)
	{
		error = GuidToString(instanceGuid, &classGuidString);
		RD_TRACE_LOG((RD_TRACE_INFO, L"InstanceGuid %s", classGuidString));
		GetInterfaceIndex(instanceGuid, index);
		InterfaceIndex = index;
		goto Cleanup;
	}

	if ((HRESULT_FROM_WIN32(ERROR_NOT_FOUND) != hr))
	{
		RD_TRACE_LOG((RD_TRACE_INFO, L"Failed to query for the presence of loopback adapter %s. Error[%0x]", InterfaceName, hr));
		goto Cleanup;
	}

	hr = CreateLoopbackAdapter(InterfaceName, &instanceGuid);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to CreateLoopbackAdapter wit name [%s], Error:[0x%x]",
		InterfaceName,
		hr),
		hr,
		Cleanup);

	RD_TRACE_LOG((RD_TRACE_INFO, L"Loopback adapter successfully created. Name:[%s]", InterfaceName));

	error = GuidToString(instanceGuid, &classGuidString);
	if (ERROR_SUCCESS != error)
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"CreateLoopbackAdapter:GuidToString - failed. Error:[0x%x]",
			error),
			hr,
			Cleanup);
	}

	RD_TRACE_LOG((RD_TRACE_INFO, L"InstanceGuid for newly created interface:%s", classGuidString));

	hr = GetInterfaceIndex(instanceGuid, index);
	RD_TRACE_E_HR((RD_TRACE_ERROR,
		L"Failed to retrieve index for interface with name [%s], Error:[0x%x]",
		InterfaceName,
		hr),
		hr,
		Cleanup);

	RD_TRACE_LOG((RD_TRACE_INFO, L"Interface name:[%s], index:[%d]", InterfaceName, index));
	InterfaceIndex = index;

Cleanup:

	STR_FREE(loopbackInfPath);
	STR_FREE(classGuidString);

	return hr;
}

HRESULT
LoopbackAdapterInstaller::GetInterfaceIndex(
	GUID InstanceGuid,
	int& InterfaceIndex
)
/*++

Routine Description:

Helper to retrieve the interface index for a given interface instance id.

Arguments:

InstanceGuid - Instance Id of the adapter.

InterfaceIndex - Out parameter to hold the interface index associated with the isnatnce Id.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	DWORD error;
	DWORD index;
	PWSTR classGuidString = NULL;
	HRESULT hr = S_OK;
	PMIB_IF_TABLE2 ifTable = NULL;
	bool loopbackInterfaceFound = false;

	InterfaceIndex = 0;
	error = GetIfTable2Ex(MibIfTableNormal, &ifTable);
	RD_E_WIN32(error, Cleanup);

	GuidToString(InstanceGuid, &classGuidString);
	RD_TRACE_LOG((RD_TRACE_INFO, L"InstanceGuid:[%s] to search", classGuidString));
	STR_FREE(classGuidString);

	for (index = 0; index < ifTable->NumEntries; index++)
	{
		GuidToString(ifTable->Table[index].InterfaceGuid, &classGuidString);
		RD_TRACE_LOG((RD_TRACE_INFO, L"Retrieved InterfaceInstanceGuid:[%s] Index:[%d]", classGuidString, ifTable->Table[index].InterfaceIndex));
		if (IsEqualGUID(ifTable->Table[index].InterfaceGuid, InstanceGuid))
		{
			RD_TRACE_LOG((RD_TRACE_INFO, L"Matched InterfaceInstanceGuid:%s, index:%d", classGuidString, ifTable->Table[index].InterfaceIndex));
			STR_FREE(classGuidString);
			loopbackInterfaceFound = TRUE;
			InterfaceIndex = ifTable->Table[index].InterfaceIndex;
			break;
		}

		STR_FREE(classGuidString);
	}

	if (loopbackInterfaceFound != TRUE)
	{
		RD_TRACE_E_WIN32_EXIT((RD_TRACE_ERROR,
			L"NetInstallLoopbackAdapter: Can't find IfIndex "
			L"for installed loopback adapter"),
			ERROR_NOT_FOUND,
			Cleanup);
	}

Cleanup:

	if (ifTable != NULL)
	{
		FreeMibTable(ifTable);
	}

	STR_FREE(classGuidString);

	return hr;
}

HRESULT
LoopbackAdapterInstaller::EnableInterface(
	bool Enable,
	HDEVINFO DeviceInfoSet,
	SP_DEVINFO_DATA* DeviceInfoData
)
/*++

Routine Description:

Helper to enable/disable an interface

Arguments:

Enable - True to enable, False otherwise.

DeviceInfoSet - The handle associated with the interface to be enabled/disabled.

DeviceInfoData - The device info data associated with the interface.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	SP_PROPCHANGE_PARAMS propertyChangeParams;
	HRESULT hr = S_OK;
	if (Enable)
	{
		propertyChangeParams.StateChange = DICS_ENABLE;
	}
	else
	{
		propertyChangeParams.StateChange = DICS_DISABLE;
	}

	ZeroMemory(&propertyChangeParams, sizeof(SP_CLASSINSTALL_HEADER));
	propertyChangeParams.ClassInstallHeader.cbSize = sizeof(SP_CLASSINSTALL_HEADER);
	propertyChangeParams.Scope = DICS_FLAG_GLOBAL;
	propertyChangeParams.ClassInstallHeader.InstallFunction = DIF_PROPERTYCHANGE;

	propertyChangeParams.StateChange = DICS_ENABLE;
	if (!SetupDiSetClassInstallParams(DeviceInfoSet, DeviceInfoData, &propertyChangeParams.ClassInstallHeader, sizeof(propertyChangeParams)))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"SetupDiSetClassInstallParams failed %0x", hr),
			hr,
			Cleanup);
	}

	if (!SetupDiCallClassInstaller(DIF_PROPERTYCHANGE, DeviceInfoSet, DeviceInfoData))
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		RD_TRACE_E_HR((RD_TRACE_ERROR,
			L"SetupDiCallClassInstaller failed"),
			hr,
			Cleanup);
	}

Cleanup:

	return hr;
}
