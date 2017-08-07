#pragma once
/*++

Copyright (c) Microsoft Corporation

Module Name:

ConfigReader.h

Abstract:

Implements functions to retrieve values from XML file.

Environment:

User-mode

--*/


#ifndef ASSERT
#define ASSERT( exp ) ((void) 0)
#endif

#include <stdio.h>
#include <wchar.h>
#include <iostream>
#include <windows.h>
#include <iostream>
#include "RdTrace.h"
#include "XmlParserHelper.h"

using namespace std;

#define ENDPOINT_NODE_NAME L"Endpoint"
#define SERVICE_TUNNEL_IPV6_ATTRIBUTE_NAME L"serviceTunnelIPv6Address"
#define INCARNATION_NODE_NAME L"Incarnation"
#define INSTANCE_ATTRIBUTE_NAME L"instance"
#define INSTANCE_NODE_NAME L"Instance"
#define ID_ATRRIBUTE_NAME L"id"
#define ADDRESS_ATTRIBUTE_NAME L"address"


class ConfigReader
{

public:

	HRESULT GetInstanceId(
		__in XmlParserHelper* XmlParserHelper, 
		std::wstring& InstanceName);

	HRESULT GetInstanceAddress(
		__in XmlParserHelper* XmlParserHelper,
		__in std::wstring InstanceName,
		__inout std::wstring& InstanceAddress);

	HRESULT GetServiceTunnelIpv6Addresses(
		__in XmlParserHelper* XmlParserHelper,
		__in std::wstring InstanceName,
		__inout std::set<std::wstring>& ServiceTunnelIpv6Addresses);
	
	std::wstring
		GetSTIpv6AddrFromInstanceIpv4Addr(
			__in std::wstring InstanceIpv4Addr
		);
};