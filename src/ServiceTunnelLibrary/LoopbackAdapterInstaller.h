/*++

Copyright (c) Microsoft Corporation

Module Name:

LoopbackAdapterInstaller.h

Abstract:

Functions related to installing loop back adapter.

Environment:

User-mode

--*/

#pragma once

#include "stdafx.h"

using namespace std;

#define NET_LOOPBACK_INF_RELATIVE_FILE_PATH L"\\INF\\NetLoop.INF"

#define LOOPBACK_ADAPTER_HARDWARE_ID L"*MSLOOP"

#define NET_LOOPBACK_LOCK_ID L"4fb66e02-302d-4a83-9bba-a5964be4b315"

class LoopbackAdapterInstaller
{

public:

	HRESULT GetOrCreateLoopbackAdapter(LPWSTR InterfaceName, int& InterfaceIndex);

private:
	HRESULT GetINetCfgWithWriteLock(__deref_out INetCfg** NetCfg);
	HRESULT ReleaseINetCfgWithLock(__in INetCfg* pnc);
	HRESULT UpdateInterfaceDisplayName(__in LPWSTR InterfaceName, __in LPWSTR InterfaceNewName, __out LPGUID InstanceGuid);
	HRESULT AcquireNetCfgWriteLock(__in INetCfg *NetCfg);
	HRESULT ReleaseNetCfgWriteLock(__in INetCfg *NetCfg);
	HRESULT EnableInterface(__in bool bEnable, __in HDEVINFO DeviceInfoSet, __in SP_DEVINFO_DATA* DeviceInfoData);
	HRESULT CreateLoopbackAdapter(__in LPWSTR interfaceName, __inout LPGUID InstanceId);
	HRESULT GetInterfaceIndex(__in GUID InstanceGuid, __inout int& InterfaceIndex);
};