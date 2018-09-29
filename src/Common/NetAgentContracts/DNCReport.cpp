/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

    DNCReport.cpp

Abstract:

    Implementation of Azure DNC report functions.

Environment:

    User Mode - Win32

--*/

#include "DNCReport.h"
#include "Logging.h"

using namespace std;
using namespace web;
using namespace http;
using namespace json;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

DNCReport::DNCReport(json::object& JsonObject)
/*++

Routine Description:

    Initialize the DNC report with the Json object received from request.

Arguments:

    JsonObject = Json object received as part of the request from the VM.

Return Value:

    None.

--*/
{
    InitializeFieldIdMap();
    SetMembersFromJson(JsonObject);
}

void
DNCReport::InitializeFieldIdMap()
/*++

Routine Description:

    Initialize the report field FieldName to key map.

Arguments:

    None.

Return Value:

    None.

--*/
{
    PluginReport::InitializeFieldIdMap();
    Metadata::InitializeFieldIdMap();
    
    DNCReportFieldIdMap[IS_NEW_INSTANCE_STR] = DNCEnum::DNCReportKey::IsNewInstance;
    DNCReportFieldIdMap[EVENT_MESSAGE_STR] = DNCEnum::DNCReportKey::EventMessage;
    DNCReportFieldIdMap[PARTITION_KEY_STR] = DNCEnum::DNCReportKey::PartitionKey;
    DNCReportFieldIdMap[ALLOCATIONS_STR] = DNCEnum::DNCReportKey::Allocations;
    DNCReportFieldIdMap[CPU_USAGE_STR] = DNCEnum::DNCReportKey::CPUUsage;
    DNCReportFieldIdMap[MEMORY_USAGE_STR] = DNCEnum::DNCReportKey::MemoryUsage;
    DNCReportFieldIdMap[PROCESSES_STR] = DNCEnum::DNCReportKey::Processes;
    DNCReportFieldIdMap[METADATA_STR] = DNCEnum::DNCReportKey::Compute;
}

void DNCReport::SetFields(wstring Name, json::value Value)
/*++

Routine Description:

    Sets DNCReport fields.

Arguments:

    Name - Name of the field.

    Value - Json value to be associated with the field.

Return Value:

    None.

--*/
{
    switch (DNCReportFieldIdMap[Name])
    {
    case DNCEnum::DNCReportKey::IsNewInstance:
        IsNewInstance = Value.as_bool();
        break;
    case DNCEnum::DNCReportKey::EventMessage:
        EventMessage = Value.as_string();
        break;
    case DNCEnum::DNCReportKey::PartitionKey:
        PartitionKey = Value.as_string();
        break;
    case DNCEnum::DNCReportKey::Allocations:
        Allocations = Value.as_string();
        break;
    case DNCEnum::DNCReportKey::CPUUsage:
        CPUUsage = Value.as_string();
        break;
    case DNCEnum::DNCReportKey::MemoryUsage:
        MemoryUsage = Value.as_string();
        break;
    case DNCEnum::DNCReportKey::Processes:
        Processes = Value.as_string();
        break;
    case DNCEnum::DNCReportKey::Compute:
        Metadata = Metadata::Metadata(Value.as_object());
        break;
    }
}

void DNCReport::Print()
/*++

Routine Description:

    Print each member of DNCReport.

Arguments:

    None.

Return Value:

    None.

--*/
{
    wstring s = L"DNCReport: {";
    s += L"IsNewInstance: " + ConvertBoolToWstring(IsNewInstance) +L"\n";
    s += L"EventMessage: " + EventMessage + L"\n";
    s += L"PartitionKey: " + PartitionKey + L"\n";
    s += L"Allocations: " + Allocations + L"\n";
    s += L"CPUUsage: " + CPUUsage + L"\n";
    s += L"MemoryUsage: " + MemoryUsage + L"\n";
    s += L"Processes: " + Processes + L"\n";
    Metadata.Print();

    LOGINFOW(L"%s}", s.c_str());
}