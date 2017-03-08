/*++

Copyright (c) Microsoft Corporation

Module Name:

    ServiceTunnelExtension.h

Abstract:

    Headers and definitions for service tunnel extension 

Environment:

    User-mode

--*/
#pragma once


#include <stdio.h>
#include <wchar.h>
#include <ws2tcpip.h>
#include <msxml6.h>
#include <windows.h>
#include "RcfEnumerator.h"
#include "ServiceTunnelConfigurator.h"
#include <iostream>
#include "XmlParserHelper.h"
#include <sstream>
#include <string>


#define AGENT_TRACE_FLAGS (RD_TRACE_ALL & ~RD_TRACE_PAINFUL)
#define LOG_FILE_PREFIX L"ServiceTunnelExtension"

#define ENDPOINT_NODE_NAME L"Endpoint"
#define SERVICE_TUNNEL_IPV6_ATTRIBUTE_NAME L"serviceTunnelIPv6Address"
#define INCARNATION_NODE_NAME L"Incarnation"
#define INSTANCE_ATTRIBUTE_NAME L"instance"
#define INSTANCE_NODE_NAME L"Instance"
#define ID_ATRRIBUTE_NAME L"id"
#define ADDRESS_ATTRIBUTE_NAME L"address"

HRESULT SetupLogging();

HRESULT GetInstanceId(
    __in XmlParserHelper* XmlParserHelper, 
    __inout std::wstring& InstanceName);

HRESULT GetInstanceAddress(
    __in XmlParserHelper* XmlParserHelper,
    __in std::wstring InstanceName, 
    __inout std::wstring& InstanceAddress);

HRESULT GetServiceTunnelIpv6Addresses(
    __in XmlParserHelper* XmlParserHelper,
    __in std::wstring InstanceName,
    __inout std::set<std::wstring>& ServiceTunnelIpv6Addresses);

