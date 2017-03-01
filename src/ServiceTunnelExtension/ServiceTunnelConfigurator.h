/*++

Copyright (c) Microsoft Corporation

Module Name:

    ServiceTunnelConfigurator.h

Abstract:

    Implements functions related to configuration of service tunnel on loopback adapter.

Environment:

    User-mode

--*/
#pragma once

#include <stdio.h>
#include <wchar.h>
#include <ws2tcpip.h>
#include <windows.h>
#include "LoopbackAdapterInstaller.h"
#include <regex>
#include <iostream>
#include <set>
using namespace std;

#define SERVICE_TUNNEL_SOURCE_PREFIX L"fde4:8dba::/32"

#define SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS L"fe80::944:63c6:6ed9:2336"

#define SERVICE_TUNNEL_ENDPOINT_FAKE_GW_MAC_ADDRESS L"12-34-56-78-9a-bc"

#define SERVICE_TUNNEL_INTERFACE_NAME L"ServiceTunnelAdapter"

#define NET_UNICAST_IP_ADDRESS_CHANGE_TIMEOUT 10 * 1000

typedef struct _NET_UNICAST_IPv6_ADDRESS_CHANGE_CONTEXT
{
    HANDLE Event;
    IN6_ADDR Ipv6Address;
} NET_UNICAST_IPv6_ADDRESS_CHANGE_CONTEXT,
*PNET_UNICAST_IPv6_ADDRESS_CHANGE_CONTEXT;

class ServiceTunnelConfigurator
{
    private:

        LoopbackAdapterInstaller installer;

    public:

        HRESULT SetupLoopbackAdapter(__in std::set<std::wstring> Ipv6AddressList);

        HRESULT NetIfEnableWeakHostRouting(__in ULONG IfIndex);

        HRESULT NetIfSetIpv6Address(__in ULONG IfIndex, __in PCWSTR Ipv6Address);

        HRESULT NetIfAddDeleteIPv6Route(__in ULONG IfIndex, __in PCWSTR Ipv6AddressPrefix, __in PCWSTR Ipv6AddressNextHop, __in BOOL IsAdd);

        HRESULT NetIfAddDeleteNeighbor(__in ULONG IfIndex, __in PCWSTR NeighborAddress, __in PCWSTR NeighborLinkLayerAddress, __in BOOL IsAdd);
};