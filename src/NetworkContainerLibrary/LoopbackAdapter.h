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

#define MAX_BUF_LENGTH 2048

using namespace std;

class LoopbackAdapter : public LoopbackAdapterInstaller {

public:
	LoopbackAdapter(
		_In_ wstring Name,
		_In_ IN_ADDR Ip,
		_In_ IN_ADDR Netmask,
		_In_ IN_ADDR Gateway,
		_In_ vector<IN_ADDR> Dns
	);
	~LoopbackAdapter();
	HRESULT FindAdapterByDisplayName(_In_ wstring Name, _Out_ int& InstanceIndex);
	HRESULT GetNetworkParameters();
	HRESULT FillDevInfo(_In_ int InstanceIndex);
	HRESULT Create();
	HRESULT Delete();
	HRESULT SetIp(_In_ IN_ADDR Ip);
	HRESULT SetNetmask(_In_ IN_ADDR Netmask);
	HRESULT SetGateway(_In_ IN_ADDR Gateway);
	HRESULT SetDns(_In_ vector<IN_ADDR> DnsVec);
	void SetIndex(_In_ int InterfaceIndex);

private:
	HRESULT Rename();
	wstring displayName;
	IN_ADDR ip, netmask, gateway;
	vector<IN_ADDR> dnsVec;
	LPWSTR loopbackInfPath = NULL;
	GUID instanceGuid = { 0 };
	int interfaceIndex = 0;
	ULONG NTEContext = 0;
	ULONG NTEInstance = 0;
};