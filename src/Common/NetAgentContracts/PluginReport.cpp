/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    PluginReport.cpp

Abstract:

    Implementation of Plugin report functions.

Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//
#include "PluginReport.h"
#include "Logging.h"	
using namespace std;
using namespace web;
using namespace http;
using namespace json;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

void
PluginReport::InitializePluginReportFieldIdMap()
/*++

Routine Description:

    Initialize the report field FieldName to key map.

Arguments:

    None.
    
Return Value:

    None.

--*/
{

    PluginReportFieldIdMap[NAME_STR] = PluginReportKey::Name;
    PluginReportFieldIdMap[VERSION_STR] = PluginReportKey::Version;
    PluginReportFieldIdMap[ERROR_MESSAGE_STR] = PluginReportKey::ErrorMessage;
    PluginReportFieldIdMap[CONTEXT_STR] = PluginReportKey::Context;
    PluginReportFieldIdMap[SUBCONTEXT_STR] = PluginReportKey::SubContext;
    PluginReportFieldIdMap[ORCHESTRATOR_DETAILS_STR] = PluginReportKey::OrchestratorDetails;
    PluginReportFieldIdMap[OS_DETAILS_STR] = PluginReportKey::OSDetails;
    PluginReportFieldIdMap[SYSTEM_DETAILS_STR] = PluginReportKey::SystemDetails;
    PluginReportFieldIdMap[ORCHESTRATOR_NAME_STR] = PluginReportKey::OrchestratorName;
    PluginReportFieldIdMap[ORCHESTRATOR_VERSION_STR] = PluginReportKey::OrchestratorVersion;
    PluginReportFieldIdMap[OS_TYPE_STR] = PluginReportKey::OSType;
    PluginReportFieldIdMap[OS_VERSION_STR] = PluginReportKey::OSVersion;
    PluginReportFieldIdMap[KERNEL_VERSION_STR] = PluginReportKey::KernelVersion;
    PluginReportFieldIdMap[OS_DISTRIBUTION_STR] = PluginReportKey::OSDistribution;
    PluginReportFieldIdMap[MEM_VM_TOTAL_STR] = PluginReportKey::MemVMTotal;
    PluginReportFieldIdMap[MEM_VM_FREE_STR] = PluginReportKey::MemVMFree;
    PluginReportFieldIdMap[MEM_USEDBY_PROCESS_STR] = PluginReportKey::MemUsedByProcess;
    PluginReportFieldIdMap[DISK_VM_TOTAL_STR] = PluginReportKey::DiskVMTotal;
    PluginReportFieldIdMap[DISK_VM_FREE_STR] = PluginReportKey::DiskVMFree;
    PluginReportFieldIdMap[CPU_COUNT_STR] = PluginReportKey::CPUCount;
}


void OrchestratorInfo::SetOrchestratorDetailsFields(
    wstring FieldName,
    json::value JsonValue
)
/*++

Routine Description:

    Sets OrchestratorDetails Class fields

Arguments:

    FieldName - Name of the field.

    JsonValue - Json value to be associated with the field.

Return Value:

None.

--*/
{
    switch (PluginReportFieldIdMap[FieldName])
    {
    case PluginReportKey::OrchestratorName:
        OrchestratorName.assign(JsonValue.as_string());
        break;

    case PluginReportKey::OrchestratorVersion:
        OrchestratorVersion.assign(JsonValue.as_string());
        break;

    case PluginReportKey::ErrorMessage:
        ErrorMessage.assign(JsonValue.as_string());
        break;

    default:
        LOGINFOW(L"Orchestrator Details Unknown key:{%s}", FieldName.c_str());
    }
}

void OSInfo::SetOSDetailsFields(
    wstring FieldName,
    json::value JsonValue
)
/*++

Routine Description:

    Sets OSDetails Class fields

Arguments:

    FieldName - Name of the field.

    JsonValue - Json value to be associated with the field.

Return Value:

None.

--*/
{
    switch (PluginReportFieldIdMap[FieldName])
    {
    case PluginReportKey::OSType:
        OSType.assign(JsonValue.as_string());
        break;

    case PluginReportKey::OSVersion:
        OSVersion.assign(JsonValue.as_string());
        break;

    case PluginReportKey::KernelVersion:
        KernelVersion.assign(JsonValue.as_string());
        break;

    case PluginReportKey::OSDistribution:
        OSDistribution.assign(JsonValue.as_string());
        break;

    case PluginReportKey::ErrorMessage:
        ErrorMessage.assign(JsonValue.as_string());
        break;

    default:
        LOGINFOW(L"OSDetails Unknown key:{%s}", FieldName.c_str());
    }
}

void SystemInfo::SetSystemDetailsFields(
    wstring FieldName,
    json::value JsonValue
)
/*++

Routine Description:

    Sets SystemDetails Class fields

Arguments:

    FieldName - Name of the field.

    JsonValue - Json value to be associated with the field.

Return Value:

None.

--*/
{
    switch (PluginReportFieldIdMap[FieldName])
    {
    case PluginReportKey::MemVMTotal:
        MemVMTotal = JsonValue.as_integer();
        break;

    case PluginReportKey::MemVMFree:
        MemVMFree = JsonValue.as_integer();
        break;

    case PluginReportKey::MemUsedByProcess:
        MemUsedByProcess = JsonValue.as_integer();
        break;

    case PluginReportKey::DiskVMTotal:
        DiskVMTotal = JsonValue.as_integer();
        break;

    case PluginReportKey::DiskVMFree:
        DiskVMFree = JsonValue.as_integer();
        break;

    case PluginReportKey::CPUCount:
        CPUCount = JsonValue.as_integer();
        break;

    case PluginReportKey::ErrorMessage:
        ErrorMessage.assign(JsonValue.as_string());
        break;

    default:
        LOGINFOW(L"SystemDetails Unknown key:{%s}", FieldName.c_str());
    }
}

bool
PluginReport::FindFieldFromBaseClass(
    wstring FieldName,
    json::value JsonValue
)
/*++

Routine Description:

    Find if field present in PluginReportFieldIdMap

Arguments:

    FieldName - Name of the field.

    JsonValue - Json value to be associated with the field.

Return Value:
    
    returns true if field found else false

--*/
{
    switch (PluginReportFieldIdMap[FieldName])
    {
    case PluginReportKey::Name:
        Name.assign(JsonValue.as_string());
        break;

    case PluginReportKey::Version:
        Version.assign(JsonValue.as_string());
        break;

    case PluginReportKey::ErrorMessage:
        ErrorMessage.assign(JsonValue.as_string());
        break;

    case PluginReportKey::Context:
        Context.assign(JsonValue.as_string());
        break;

    case PluginReportKey::SubContext:
        SubContext.assign(JsonValue.as_string());
        break;

    case PluginReportKey::OrchestratorDetails:
    {
        json::object JsonObject = JsonValue.as_object();

        for (auto const & e : JsonObject)
        {
            const wstring propertyName = e.first;
            const json::value propertyValue = e.second;

            OrchestratorDetails.SetOrchestratorDetailsFields(propertyName, propertyValue);
        }

        break;
    }

    case PluginReportKey::OSDetails:
    {
        json::object JsonObject = JsonValue.as_object();

        for (auto const & e : JsonObject)
        {
            const wstring propertyName = e.first;
            const json::value propertyValue = e.second;

            OSDetails.SetOSDetailsFields(propertyName, propertyValue);
        }
        break;
    }

    case PluginReportKey::SystemDetails:
    {
        json::object JsonObject = JsonValue.as_object();

        for (auto const & e : JsonObject)
        {
            const wstring propertyName = e.first;
            const json::value propertyValue = e.second;

            SystemDetails.SetSystemDetailsFields(propertyName, propertyValue);
        }
        break;
    }

    default:
        return false;
    }

    return true;
}

void
PluginReport::PrintPluginReport()
/*++

Routine Description:

    Print each member of Plugin Report

Arguments:

    None.

Return Value:

    None.

--*/
{
    LOGINFOW(L"Name %s Version %s ErrorMessage %s Context %s SubContext %s", 
        Name.c_str(), Version.c_str(), ErrorMessage.c_str(), Context.c_str(), SubContext.c_str());

    LOGINFOW(L"OrchestratorDetails{Name %s Version %s ErrMsg %s}", 
        OrchestratorDetails.OrchestratorName.c_str(), OrchestratorDetails.OrchestratorVersion.c_str(), OrchestratorDetails.ErrorMessage.c_str());

    LOGINFOW(L"OSDetails{OSType %s OSVersion %s KernelVersion %s OSDistribution %s ErrMsg %s}",
        OSDetails.OSType.c_str(), OSDetails.OSVersion.c_str(), OSDetails.KernelVersion.c_str(), OSDetails.OSDistribution.c_str(), OSDetails.ErrorMessage.c_str());

    LOGINFOW(L"SystemDetails{MemTotal %lld MemFree %lld MemProcessUsed %lld DiskTotal %lld DiskFree %lld CPUCount %d ErrorMessage %s}",
        SystemDetails.MemVMTotal, SystemDetails.MemVMFree, SystemDetails.MemUsedByProcess, SystemDetails.DiskVMTotal,
        SystemDetails.DiskVMFree, SystemDetails.CPUCount, SystemDetails.ErrorMessage.c_str());
}

