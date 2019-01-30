/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

    CNSReport.h

Abstract:

    Header to define the Azure Container Networking Service report functions.

Environment:

    User Mode - Win32

--*/

#pragma once

#include <string>
#include <map>
#include <cpprest/json.h>
#include "PluginReport.h"
#include "Metadata.h"

using namespace std;
using namespace web;

#define IS_NEW_INSTANCE_STR L"IsNewInstance"
#define EVENT_MESSAGE_STR L"EventMessage"
#define DNC_PARTITION_KEY_STR L"DncPartitionKey"
#define CPU_USAGE_STR L"CPUUsage"
#define MEMORY_USAGE_STR L"MemoryUsage"
#define PROCESSES_STR L"Processes"
#define TIMESTAMP_STR L"Timestamp"
#define UUID_STR L"UUID"
#define ERRORCODE_STR L"Errorcode"

//
// Enums for fields in the CNS report.
//
namespace CNSEnum
{
    enum CNSReportKey
    {
        IsNewInstance,
        EventMessage,
        PartitionKey,
        CPUUsage,
        MemoryUsage,
        Processes,
        Timestamp,
        UUID,
        Errorcode,
        Compute
    };
}

static map<std::wstring, CNSEnum::CNSReportKey> CNSReportFieldIdMap;

// CNS Report
class CNSReport : public PluginReport {

public:

    CNSReport(json::object& JsonObject);

    // Flag to identify number of VMs running Azure CNS.
    bool IsNewInstance;

    // Event message.
    wstring EventMessage;

    // Partition key.
    wstring PartitionKey;

    // CPU usage.
    wstring CPUUsage;

    // Memeory usage.
    wstring MemoryUsage;

    // Processes.
    wstring Processes;

    // Timestamp
    wstring Timestamp;

    // UUID
    wstring UUID;

    // Errorcode
    wstring Errorcode;

    // Metadata for Azure CNS report.
    Metadata Metadata;

    static void InitializeFieldIdMap();

    void SetFields(
        wstring FieldName,
        json::value Value
    );

    void Print();
};