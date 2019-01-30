/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

    CNSReport.cpp

Abstract:

    Implementation of Azure CNS report functions.

Environment:

    User Mode - Win32

--*/

#include "CNSReport.h"
#include "Logging.h"

using namespace std;
using namespace web;
using namespace http;
using namespace json;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

CNSReport::CNSReport(json::object& JsonObject) : IsNewInstance(false)
/*++

Routine Description:

    Initialize the CNS report with the Json object received from request.

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
CNSReport::InitializeFieldIdMap()
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

    CNSReportFieldIdMap[IS_NEW_INSTANCE_STR] = CNSEnum::CNSReportKey::IsNewInstance;
    CNSReportFieldIdMap[EVENT_MESSAGE_STR] = CNSEnum::CNSReportKey::EventMessage;
    CNSReportFieldIdMap[DNC_PARTITION_KEY_STR] = CNSEnum::CNSReportKey::PartitionKey;
    CNSReportFieldIdMap[CPU_USAGE_STR] = CNSEnum::CNSReportKey::CPUUsage;
    CNSReportFieldIdMap[MEMORY_USAGE_STR] = CNSEnum::CNSReportKey::MemoryUsage;
    CNSReportFieldIdMap[PROCESSES_STR] = CNSEnum::CNSReportKey::Processes;
    CNSReportFieldIdMap[TIMESTAMP_STR] = CNSEnum::CNSReportKey::Timestamp;
    CNSReportFieldIdMap[UUID_STR] = CNSEnum::CNSReportKey::UUID;
    CNSReportFieldIdMap[ERRORCODE_STR] = CNSEnum::CNSReportKey::Errorcode;
    CNSReportFieldIdMap[METADATA_STR] = CNSEnum::CNSReportKey::Compute;
}

void CNSReport::SetFields(wstring Name, json::value Value)
/*++

Routine Description:

    Sets CNSReport fields.

Arguments:

    Name - Name of the field.

    Value - Json value to be associated with the field.

Return Value:

    None.

--*/
{
    // Prevent memory exhaustion attack, since we have a [] operator
    // in the switch statement below.
	if (CNSReportFieldIdMap.find(Name) == CNSReportFieldIdMap.end())
	{
		LOGINFOW(L"Unknown key:{%s}", Name.c_str());
		return;
	}

    switch (CNSReportFieldIdMap[Name])
    {
    case CNSEnum::CNSReportKey::IsNewInstance:
        IsNewInstance = Value.as_bool();
        break;
    case CNSEnum::CNSReportKey::EventMessage:
        EventMessage = Value.as_string();
        break;
    case CNSEnum::CNSReportKey::PartitionKey:
        PartitionKey = Value.as_string();
        break;
    case CNSEnum::CNSReportKey::CPUUsage:
        CPUUsage = Value.as_string();
        break;
    case CNSEnum::CNSReportKey::MemoryUsage:
        MemoryUsage = Value.as_string();
        break;
    case CNSEnum::CNSReportKey::Processes:
        Processes = Value.as_string();
        break;
    case CNSEnum::CNSReportKey::Timestamp:
        Timestamp = Value.as_string();
        break;
    case CNSEnum::CNSReportKey::UUID:
        UUID = Value.as_string();
        break;
    case CNSEnum::CNSReportKey::Errorcode:
        Errorcode = Value.as_string();
        break;
    case CNSEnum::CNSReportKey::Compute:
        Metadata = Metadata::Metadata(Value.as_object());
        break;
    }
}

void CNSReport::Print()
/*++

Routine Description:

    Print each member of CNSReport.

Arguments:

    None.

Return Value:

    None.

--*/
{
    wstring s = L"CNSReport: {";
    s += L"IsNewInstance: " + ConvertBoolToWstring(IsNewInstance) + L"\n";
    s += L"EventMessage: " + EventMessage + L"\n";
    s += L"PartitionKey: " + PartitionKey + L"\n";
    s += L"CPUUsage: " + CPUUsage + L"\n";
    s += L"MemoryUsage: " + MemoryUsage + L"\n";
    s += L"Processes: " + Processes + L"\n";
    s += L"Timestamp: " + Timestamp + L"\n";
    s += L"Errorcode: " + Errorcode + L"\n";
    s += L"UUID: " + UUID + L"\n";
    Metadata.Print();

    LOGINFOW(L"%s}", s.c_str());
}
