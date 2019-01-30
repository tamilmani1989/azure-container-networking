/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    CNIReport.h

Abstract:

    Header to define the CNI report functions.

Environment:

    User Mode - Win32

--*/

#pragma once

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <map>
#include <cpprest/json.h>
#include <cpprest/http_listener.h>
#include <cpprest/uri.h>
#include <cpprest/asyncrt_utils.h>
#include <cpprest/filestream.h>
#include <cpprest/containerstream.h>
#include <cpprest/producerconsumerstream.h>
#include "cpprest/filestream.h"
#include "PluginReport.h"
#include "Metadata.h"

using namespace std;
using namespace web;

#define UINT64 uint64_t

// TODO: Remove STARTFLAG_STR in future versions.
// This is here for backward compatibility for now.
#define STARTFLAG_STR               L"StartFlag"
#define IS_NEW_INSTANCE_STR         L"IsNewInstance"
#define CNI_SUCCEEDED_STR           L"CniSucceeded"
#define CONTAINERNAME_STR           L"ContainerName"
#define INFRA_VNETID_STR            L"InfraVnetID"
#define VNET_ADDRESS_SPACE_STR      L"VnetAddressSpace"
#define INTERFACE_DETAILS_STR       L"InterfaceDetails"
#define INTERFACE_TYPE_STR          L"InterfaceType"
#define SUBNET_STR                  L"Subnet"
#define PRIMARY_CA_STR              L"PrimaryCA"
#define MAC_STR                     L"MAC"
#define CA_TOTAL_COUNT_STR          L"SecondaryCATotalCount"
#define CA_USED_COUNT_STR           L"SecondaryCAUsedCount"
#define BRIDGE_DETAILS_STR          L"BridgeDetails"
#define NETWORKMODE_STR             L"NetworkMode"
//
// Enums for fields in the CNI report.
//

namespace CNIEnum {
    enum CNIReportKey
    {
        IsNewInstance = PluginReportKey::LastEnum,
        ContainerName,
        InfraVnetID,
        VnetAddressSpace,
        InterfaceDetails,
        InterfaceType,
        IfName,
        Subnet,
        PrimaryCA,
        MAC,
        SecondaryCATotalCount,
        SecondaryCAUsedCount,
        IfErrorMessage,
        BridgeDetails,
        NetworkMode,
        CniSucceeded,
        Metadata
    };
}

// CNI report field name to Id map
static  std::map<std::wstring, CNIEnum::CNIReportKey> CNIReportFieldIdMap;

// Interface Details
class InterfaceInfo
{
public:

    wstring InterfaceType;
    wstring Subnet;
    wstring PrimaryCA;
    wstring MAC;
    wstring Name;
    int SecondaryCATotalCount = 0;
    int SecondaryCAUsedCount = 0;
    wstring ErrorMessage;

    void
    SetInterfaceDetailsFields(
            wstring FieldName,
            json::value JsonValue
    );
};

// Bridge Details
class BridgeInfo
{
public:

    wstring NetworkMode;
    wstring BridgeName;
    wstring ErrorMessage;

    void
    SetBridgeDetailsFields(
            wstring FieldName,
            json::value JsonValue
    );
};

// CNI Report
class CNIReport : public PluginReport
{

public:
    CNIReport(json::object &JsonObject);

    // Flag to identify Number of VMs running this Plugin
    bool IsNewInstance = false;

    // Flag to indicate whether CNI ran successfully
    bool CniSucceeded = false;

    // Pod name and namespace
    wstring ContainerName;

    // InfraVnetID
    wstring InfraVnetID;

    // VnetAddressSpaces in a VNET
    vector <wstring> VnetAddressSpace;

    // Interface Info
    InterfaceInfo InterfaceDetails;

    // Bridge Info
    BridgeInfo BridgeDetails;

    // Metadata for Azure NPM report.
    Metadata Metadata;

    static
    void
    InitializeFieldIdMap();

    void
    SetFields(
        std::wstring Name,
        web::json::value Value);


    void SetMembersFromJson(json::object &JsonObject);

    void Print();
};