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


#define AGENT_TRACE_FLAGS (RD_TRACE_ALL & ~RD_TRACE_PAINFUL)
#define LOG_FILE_PREFIX L"ServiceTunnelExtension"

#define ENDPOINT_NODE_NAME L"Endpoint"
#define SERVICE_TUNNEL_IPV6_ATTRIBUTE_NAME L"serviceTunnelIPv6Address"

HRESULT SetupLogging();

