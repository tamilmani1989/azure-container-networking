/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

LoopbackAdapterInstaller.cpp

Abstract:

Implements methods to manipulate loopback adapter.

--*/

#pragma once

#include "stdafx.h"
#include "Logging.h"
#include "LoopbackAdapterInstaller.h"
#include "Devpkey.h"
#include <iostream>
#include <string>
#include <iterator>

using namespace std;

class LoopbackAdapter : public LoopbackAdapterInstaller {

public:
	LoopbackAdapter(
		_In_ wstring Name,
		_In_ IN_ADDR Ip,
		_In_ IN_ADDR Netmask,
		_In_ IN_ADDR Dns,
		_In_ IN_ADDR Gateway
	);
	~LoopbackAdapter();
	HRESULT GetNetworkParameters();
	HRESULT Create();
	HRESULT SetIp(_In_ IN_ADDR Ip);
	HRESULT SetNetmask(_In_ IN_ADDR Netmask);
	HRESULT SetGateway(_In_ IN_ADDR Gateway);
	HRESULT SetDns(_In_ IN_ADDR Dns);
	HRESULT Delete();

private:
	wstring displayName;
	IN_ADDR ip, netmask, dns, gateway;
	LPWSTR loopbackInfPath = NULL;
	GUID instanceGuid = { 0 };
	HDEVINFO deviceInfoSet = NULL;
	SP_DEVINFO_DATA deviceInfoData = { 0 };
	int interfaceIndex = 0;
	ULONG NTEContext = 0;
	ULONG NTEInstance = 0;
};