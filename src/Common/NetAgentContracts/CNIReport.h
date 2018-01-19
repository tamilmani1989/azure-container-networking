/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    CNIReport.h

Abstract:

    Header to define the CNI report functions.

Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//
#pragma once

#include <stdio.h>
#include <stdlib.h>
#include<stdint.h>
#include <map>
#include <cpprest/json.h>
#include <cpprest/http_listener.h>
#include <cpprest/uri.h>
#include <cpprest/asyncrt_utils.h>
#include <cpprest/json.h>
#include <cpprest/filestream.h>
#include <cpprest/containerstream.h>
#include <cpprest/producerconsumerstream.h>
#include "cpprest/filestream.h"
#include "PluginReport.h"

using namespace std;
using namespace web;

#define UINT64 uint64_t

#define STARTFLAG_STR               L"StartFlag"
#define VNET_ADDRESS_SPACE_STR      L"VnetAddressSpace"
#define INTERFACE_DETAILS_STR       L"InterfaceDetails"
#define INTERFACE_TYPE_STR          L"InterfaceType"
#define SUBNET_STR                  L"Subnet"
#define PRIMARY_CA_STR              L"PrimaryCA"
#define MAC_STR                     L"MAC"
#define CA_TOTAL_COUNT_STR          L"SecondaryCATotalCount"
#define CA_USED_COUNT_STR           L"SecondaryCAUsedCount"
#define BRIDGE_DETAILS_STR          L"BridgeDetails"

//
// Enums for fields in the CNI report.
//

enum CNIReportKey {
    StartFlag = PluginReportKey::LastEnum,
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
    BridgeDetails
};

// CNI report field name to Id map
static  std::map<std::wstring, CNIReportKey> CNIReportFieldIdMap;

// Interface Details
class InterfaceInfo {
public:

    wstring InterfaceType;
    wstring Subnet;
    wstring PrimaryCA;
    wstring MAC;
    wstring Name;
    int SecondaryCATotalCount;
    int SecondaryCAUsedCount;
    wstring ErrorMessage;

    void
    SetInterfaceDetailsFields(
            wstring FieldName,
            json::value JsonValue
    );
};

// Bridge Details
class BridgeInfo{
    wstring NetworkMode;
    wstring BridgeName;
    wstring ErrorMessage;
};

// CNI Report
class CNIReport : public PluginReport {

public:
    // Flag to identify Number of VMs running this Plugin
    bool StartFlag;

    // VnetAddressSpaces in a VNET
    vector <wstring> VnetAddressSpace;

    // Interface Info
    InterfaceInfo InterfaceDetails;

    // Bridge Info
    BridgeInfo BridgeDetails;

    void
        InitializeCNIReport(
        json::object &JsonObject);

    static
    void
    InitializeCNIReportFieldIdMap();

    void
    SetCNIReportField(
        std::wstring Name,
        web::json::value Value);


    void SetMembersFromJson(json::object &JsonObject);

    void PrintCNIReport();
};