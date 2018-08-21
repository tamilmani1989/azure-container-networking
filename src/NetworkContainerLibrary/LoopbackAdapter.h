/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

	LoopbackAdapterInstaller.cpp

Abstract:

	Implements methods to manipulate loopback adapter.

Author:

	Yongli Chen

--*/

#pragma once

#include "stdafx.h"
#include "Logging.h"
#include "LoopbackAdapterInstaller.h"
#include "Devpkey.h"
#include <iostream>
#include <regex>
#include <sstream>
#include <string>
#include <iterator>
#include <memory>

#define MAX_BUF_LENGTH 2048

#define DEFAULT_ROUTE_PREFIX L"0.0.0.0"

#define DEFAULT_ROUTE_LENGTH 0

using namespace std;

class LoopbackAdapter : public LoopbackAdapterInstaller {

public:
	LoopbackAdapter(
		_In_ wstring Name,
		_In_ IN_ADDR Ip,
		_In_ IN_ADDR Netmask,
		_In_ IN_ADDR Gateway,
		_In_ vector<IN_ADDR> Dns,
		_In_ bool WhsFlag,
		_In_ bool WhrFlag
	);
	~LoopbackAdapter();
	bool isNumber(wstring Name);
	HRESULT FindAdapterByDisplayName(_In_ wstring Name, _Out_ int& InstanceIndex);
	HRESULT FindAdapterByIndex(_In_ wstring Index);
	HRESULT GetNetworkParameters();
	HRESULT FillDevInfo(_In_ int InstanceIndex);
	HRESULT Create();
	HRESULT Delete();
	HRESULT SetIp(_In_ IN_ADDR Ip);
	HRESULT SetDns(_In_ vector<IN_ADDR> DnsVec);
	HRESULT SetWeakHost(_In_ bool WhsFlag, _In_ bool WhrFlag);
	HRESULT SetNetwork(
		_In_ IN_ADDR Ip, 
		_In_ IN_ADDR Netmask = IN_ADDR(),
		_In_ IN_ADDR Gateway = IN_ADDR()
	);
	HRESULT DeleteAllRoutes();
	HRESULT AddCatchAllRoute();

	void SetIndex(_In_ int InterfaceIndex);

private:
	HRESULT Rename();
	wstring displayName;
	IN_ADDR ip, netmask, gateway;
	vector<IN_ADDR> dnsVec;
	bool whsFlag, whrFlag;
	LPWSTR loopbackInfPath = NULL;
	GUID interfaceGuid = { 0 };
	NET_LUID interfaceLuid = { 0 };
	int interfaceIndex = 0;
	ULONG NTEContext = 0;
	ULONG NTEInstance = 0;
	bool catchAllRouteAlreadyExists = false;
};