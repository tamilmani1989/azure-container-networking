/*++

Copyright (c) 2017 Microsoft Corporation

Module Name:

    PluginReport.h

Abstract:

    Header to define the Plugin report functions.

Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//
#pragma once

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
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
#include "json.h"

using namespace std;
using namespace web;

#define UINT64 uint64_t

#define NAME_STR                    L"Name"
#define VERSION_STR                 L"Version"
#define ERROR_MESSAGE_STR           L"ErrorMessage"
#define EVENT_MESSAGE_STR           L"EventMessage"
#define OPTYPE_STR                  L"OperationType"
#define OPDURATION_STR              L"OperationDuration"
#define CONTEXT_STR                 L"Context"
#define SUBCONTEXT_STR              L"SubContext"
#define VM_UPTIME_STR               L"VMUptime"
#define TIMESTAMP_STR               L"Timestamp"
#define ORCHESTRATOR_DETAILS_STR    L"OrchestratorDetails"
#define ORCHESTRATOR_NAME_STR       L"OrchestratorName"
#define ORCHESTRATOR_VERSION_STR    L"OrchestratorVersion"
#define OS_DETAILS_STR              L"OSDetails"
#define OS_TYPE_STR                 L"OSType"
#define OS_VERSION_STR              L"OSVersion"
#define KERNEL_VERSION_STR          L"KernelVersion"
#define OS_DISTRIBUTION_STR         L"OSDistribution"
#define SYSTEM_DETAILS_STR          L"SystemDetails"
#define MEM_VM_TOTAL_STR            L"MemVMTotal"
#define MEM_VM_FREE_STR             L"MemVMFree"
#define MEM_USEDBY_PROCESS_STR      L"MemUsedByProcess"
#define DISK_VM_TOTAL_STR           L"DiskVMTotal"
#define DISK_VM_FREE_STR            L"DiskVMFree"
#define CPU_COUNT_STR               L"CPUCount"

//
// Enums for fields in the Plugin report.
//

enum PluginReportKey {
    Name=1,
    Version,
    ErrorMessage,
    EventMessage,
    OperationType,
    OperationDuration,
    Context,
    SubContext,
    VMUptime,
    Timestamp,
    OrchestratorDetails,
    OrchestratorName,
    OrchestratorVersion,
    OSDetails,
    OSType,
    OSVersion,
    KernelVersion,
    OSDistribution,
    SystemDetails,
    MemVMTotal,
    MemVMFree,
    MemUsedByProcess,
    DiskVMTotal,
    DiskVMFree,
    CPUCount,
    LastEnum
};

static std::map<std::wstring, PluginReportKey> PluginReportFieldIdMap;

// Orchestrator Details
class OrchestratorInfo {
public:
    wstring OrchestratorName;
    wstring OrchestratorVersion;
    wstring ErrorMessage;

    void
    SetOrchestratorDetailsFields(
        wstring FieldName,
        json::value JsonValue
    );
};

// OS Details
class OSInfo {
public:
    wstring OSType;
    wstring OSVersion;
    wstring KernelVersion;
    wstring OSDistribution;
    wstring ErrorMessage;

    void
    SetOSDetailsFields(
            wstring FieldName,
            json::value JsonValue
    );
};

// System Details
class SystemInfo {
public:
    UINT64 MemVMTotal = 0;
    UINT64 MemVMFree = 0;
    UINT64 MemUsedByProcess = 0;
    UINT64 DiskVMTotal = 0;
    UINT64 DiskVMFree = 0;
    int CPUCount = 0;
    wstring ErrorMessage;

    void
    SetSystemDetailsFields(
            wstring FieldName,
            json::value JsonValue
    );
};

// Generic Plugin Report
class PluginReport: public Microsoft::Azure::Networking::Aqua::ClientLibrary::Json<PluginReport>{

public:
    // Name of Plugin
    wstring Name;

    // Version of Plugin
    wstring Version;

    // Error Message
    wstring ErrorMessage;

    // Event Message
    wstring EventMessage;

    // Operation Type
    wstring OperationType;

    // Operation Duration
    int OperationDuration;

    // Context of event
    wstring Context;

    // SubContext of event
    wstring SubContext;

    // VM Uptime
    wstring VMUptime;

    // Timestamp
    wstring Timestamp;

    // Orchestrator Info
    OrchestratorInfo OrchestratorDetails;

    // OS Info
    OSInfo OSDetails;

    // System Info
    SystemInfo SystemDetails;

    static void
    InitializeFieldIdMap();

    bool
    FindFieldFromBaseClass(
        wstring FieldName,
        json::value JsonValue
    );

    void
    Print();
};