/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

    GetInterfaceForContainerResponse.h

Abstract:

    Header to define the Reports GetInterfaceForContainerResponse functions.

Environment:

    User Mode - Win32

--*/

#pragma once

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <set>
#include <cpprest/json.h>
#include <cpprest/http_listener.h>
#include <cpprest/uri.h>
#include <cpprest/asyncrt_utils.h>
#include <cpprest/filestream.h>
#include <cpprest/containerstream.h>
#include <cpprest/producerconsumerstream.h>
#include "cpprest/filestream.h"
#include "json.h"

using namespace std;
using namespace web;

#define RESPONSE                  L"Response"
#define NETWORK_INTERFACE         L"NetworkInterface"
#define CNET_ADDRESS_SPACE        L"CnetAddressSpace"
#define DNS_SERVERS               L"DNSServers"
#define NETWORK_CONTAINER_VERSION L"NetworkContainerVersion"

#define RETURN_CODE               L"ReturnCode"
#define MESSAGE                   L"Message"

#define NAME                      L"Name"
#define IP_ADDRESS                L"IPAddress"
#define PREFIX_LENGTH             L"PrefixLength"

// Enums for fields in GetInterfaceForContainerResponse
namespace GetInterfaceForContainerResponseEnum
{
    enum ResponseKey
    {
        Response,
        NetworkInterface,
        CnetAddressSpace,
        DNSServers,
        NetworkContainerVersion,
        ReturnCode,
        Message,
        Name,
        IPAddressKey,
        PrefixLength
    };
}

static map<wstring, GetInterfaceForContainerResponseEnum::ResponseKey> GetInterfaceForContainerResponseFieldIdMap;

// NetworkInterface
class NetworkInterface : public Microsoft::Azure::Networking::Aqua::ClientLibrary::Json<NetworkInterface>{

public:

    wstring Name;
    wstring Address;

    NetworkInterface();
    NetworkInterface(json::object& JsonObject);
    static void InitializeFieldIdMap();
    void SetFields(wstring FieldName, json::value JsonValue);
    void Print()
    {
    }
};

// IPSubnet
class IPSubnet : public Microsoft::Azure::Networking::Aqua::ClientLibrary::Json<IPSubnet>{

public:

    wstring Address;
    uint8_t Cidr;

    IPSubnet(json::object& JsonObject);
    static void InitializeFieldIdMap();
    void SetFields(wstring FieldName, json::value JsonValue);
    void Print()
    {
    }

    bool operator< (const IPSubnet &right) const
    {
        return Cidr < right.Cidr;
    }
};

// Response
class Response : public Microsoft::Azure::Networking::Aqua::ClientLibrary::Json<Response>{

public:

    int ReturnCode;
    wstring Message;

    Response();
    Response(json::object& JsonObject);
    static void InitializeFieldIdMap();
    void SetFields(wstring FieldName, json::value JsonValue);
    void Print()
    {
    }
};

// GetInterfaceForContainerResponse
class GetInterfaceForContainerResponse : public Microsoft::Azure::Networking::Aqua::ClientLibrary::Json<GetInterfaceForContainerResponse>{

public:

    wstring NetworkContainerVersion;
    NetworkInterface NetworkInterface;
    vector<IPSubnet> CnetAddressSpace;
    vector<wstring> DNSServers;
    Response Response;

    GetInterfaceForContainerResponse(json::object& JsonObject);
    static void InitializeFieldIdMap();
    void SetFields(wstring FieldName, json::value JsonValue);
    void Print()
    {
    }
};
