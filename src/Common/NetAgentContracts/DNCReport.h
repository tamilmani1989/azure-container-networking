/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

    DNCReport.h

Abstract:

    Header to define the Azure Delegated Network Controller report functions.

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
#define PARTITION_KEY_STR L"PartitionKey"
#define ALLOCATIONS_STR L"Allocations"
#define CPU_USAGE_STR L"CPUUsage"
#define MEMORY_USAGE_STR L"MemoryUsage"
#define PROCESSES_STR L"Processes"
#define TIMESTAMP_STR L"Timestamp"
#define NUMBER_OF_NODES_STR L"NumberOfNodes"
#define NUMBER_OF_NCS_STR L"NumberOfNCs"
#define ORCHESTRATOR_STR L"Orchestrator"
#define CONTAINER_TYPE_STR L"ContainerType"
#define ERRORCODE_STR L"Errorcode"

//
// Enums for fields in the DNC report.
//
namespace DNCEnum 
{
    enum DNCReportKey 
    {
        IsNewInstance,
        EventMessage,
        PartitionKey,
        Allocations,
        CPUUsage,
        MemoryUsage,
        Processes,
        Timestamp,
        NumberOfNodes,
        NumberOfNCs,
        Orchestrator,
        ContainerType,
        Errorcode,
        Compute
    };
}

static map<std::wstring, DNCEnum::DNCReportKey> DNCReportFieldIdMap;

// DNC Report
class DNCReport : public PluginReport {

public:

    DNCReport(json::object& JsonObject);

    // Flag to identify number of VMs running Azure DNC.
    bool IsNewInstance = false;

    // Event message.
    wstring EventMessage;

    // Partition key.
    wstring PartitionKey;

    // Allocations.
    wstring Allocations;

    // CPU usage.
    wstring CPUUsage;

    // Memeory usage.
    wstring MemoryUsage;

    // Processes.
    wstring Processes;

    // Timestamp
    wstring Timestamp;

    // NumberOfNodes
    int NumberOfNodes = 0;

    // NumberOfNCs
    int NumberOfNCs = 0;

    // Orchestrator
    wstring Orchestrator;

    // ContainerType
    wstring ContainerType;

    // Errorcode
    wstring Errorcode;

    // Metadata for Azure DNC report.
    Metadata Metadata;

    static void InitializeFieldIdMap();

    void SetFields(
        wstring FieldName,
        json::value Value
    );

    void Print();
};