/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

	NetworkContainer.h

Abstract:

	Implements functions related to network container.

Author:

	Yongli Chen

	--*/

#pragma once

#include "stdafx.h"
#include "Logging.h"
#include "LoopbackAdapterInstaller.h"
#include "LoopbackAdapter.h"
#include "Devpkey.h"
#include <iostream>
#include <map>
#include <memory>

using namespace std;

#define NET_LOOPBACK_INF_RELATIVE_FILE_PATH L"\\INF\\NetLoop.INF"

#define LOOPBACK_ADAPTER_HARDWARE_ID L"*MSLOOP"

#define NET_LOOPBACK_LOCK_ID L"4fb66e02-302d-4a83-9bba-a5964be4b315"

#define LOG_PATH_CMDLINE_FLAG L"/logpath"

#define DEFAULT_LOG_LOCATION  L"C:\\Logs\\NetworkContainer"

#define HELP_CMDLINE_FLAG L"/help"

#define NAME_CMDLINE_FLAG L"/name"

#define OPERATION_CMDLINE_FLAG L"/operation"

#define TYPE_CMDLINE_FLAG L"/type"

#define IP_CMDLINE_FLAG L"/ip"

#define NETMASK_CMDLINE_FLAG L"/netmask"

#define DNS_CMDLINE_FLAG L"/dns"

#define GATEWAY_CMD_LINE_FLAG L"/gateway"

class NetworkContainer: public LoopbackAdapterInstaller
{

public:
	NetworkContainer();
	~NetworkContainer();
	NetworkContainer(
		_In_ wstring Name,
		_In_ IN_ADDR Ip,
		_In_ IN_ADDR Netmask,
		_In_ IN_ADDR Gateway,
		_In_ vector<IN_ADDR> DnsVec
	);

	HRESULT FindAdapterByDisplayName(_In_ wstring Name);
	HRESULT CreateLoopbackAdapter(_In_ wstring Name);
	HRESULT ConfigLoopbackAdapter(
		_In_ wstring Name,
		_In_ IN_ADDR Ip,
		_In_ IN_ADDR Netmask,
		_In_ IN_ADDR Gateway,
		_In_ vector<IN_ADDR> DnsVec);
	HRESULT DeleteLoopbackAdapter(_In_ wstring Name);
	const map<wstring, std::unique_ptr<LoopbackAdapter>>& GetAdapters();

private:
	map<wstring, std::unique_ptr<LoopbackAdapter>> adapters;
	wstring name, type;
	IN_ADDR ip, netmask, gateway;
	vector<IN_ADDR> dnsVec;
	int lastIndex = 0;
};