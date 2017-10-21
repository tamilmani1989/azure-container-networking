/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    HealReport.cpp

Abstract:

    Implementation of Health report functions.

Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//
#include "HealthReport.h"

using namespace std;
using namespace web;
using namespace http;
using namespace json;

void
HealthReport::InitializeHealthReport(
    std::wstring ContainerId,
    json::object& JsonObject
    )
/*++

Routine Description:

    Initialize the health report with the Json object received from request.

Arguments:

    ContainerId - Id of the container.

    JsonObject = Json object received as part of the request from the VM.

Return Value:

    None.

--*/
{
    this->ContainerId.assign(ContainerId.c_str());
    SetMembersFromJson(JsonObject);
}

void
HealthReport::InitializeHealthReportFieldIdMap()
/*++

Routine Description:

    Initialize the report field FieldName to key map.

Arguments:

    None.
    
Return Value:

    None.

--*/
{
    HealthReportFieldIdMap[GUEST_OS_FAMILY_STR] = ReportKey::GuestOSFamily;
    HealthReportFieldIdMap[GUEST_OS_VERSION_STR] = ReportKey::GuestOSVersion;
    HealthReportFieldIdMap[DRIVER_INSTALL_ATTEMPTED_STR] = ReportKey::DriverInstallAttempted;
    HealthReportFieldIdMap[DRIVER_INSTALL_SUCCEEDED_STR] = ReportKey::DriverInstallSucceeded;
    HealthReportFieldIdMap[DRIVER_INSTALL_FAILED_STR] = ReportKey::DriverInstallFailed;
    HealthReportFieldIdMap[DRIVER_INSTALL_NOT_ATTEMPTED_STR] = ReportKey::DriverInstallNotAttempted;
    HealthReportFieldIdMap[TOTAL_VF_DEVICES_COUNT_STR] = ReportKey::TotalVFDevices;
    HealthReportFieldIdMap[NON_FUNCTIONAL_VF_DEVICES_COUNT_STR] = ReportKey::NonFunctionalVFDevices;
    HealthReportFieldIdMap[INSTALLATION_ERROR_CODE] = ReportKey::InstallationErrorCode;
}

void 
HealthReport::SetHealthReportField(
    wstring FieldName, 
    json::value JsonValue)
/*++

Routine Description:

    Set JsonValue in report field.

Arguments:

    FieldName - Name of the field.

    JsonValue - Json value to be associated with the field.

Return Value:

    None.

--*/
{
    switch (HealthReportFieldIdMap[FieldName])
    {
    case ReportKey::GuestOSFamily:
        GuestOSFamily.assign(JsonValue.as_string());
        break;

    case ReportKey::GuestOSVersion:
        GuestOSVersion.assign(JsonValue.as_string());
        break;

    case ReportKey::DriverInstallAttempted:
        DriverInstallAttempted = JsonValue.as_bool();
        break;

    case ReportKey::DriverInstallSucceeded:
        DriverInstallSucceeded = JsonValue.as_bool();
        break;

    case ReportKey::DriverInstallFailed:
        DriverInstallFailed = JsonValue.as_bool();
        break;

    case ReportKey::DriverInstallNotAttempted:
        DriverInstallNotAttempted = JsonValue.as_bool();
        break;

    case ReportKey::TotalVFDevices:
        TotalVFDevicesCount = JsonValue.as_integer();
        break;

    case ReportKey::NonFunctionalVFDevices:
        NonFunctionalVFDevicesCount = JsonValue.as_integer();
        break;

    case ReportKey::InstallationErrorCode:
        ErrorCode = JsonValue.as_integer();
        break;

    default:
        wprintf(L"Unknown key:{%s}", FieldName.c_str());

    }
}

void 
HealthReport::SetMembersFromJson(
    json::object & JsonObject)
/*++

Routine Description:

    Set members of the health report with the report received from VM.

Arguments:

    JsonObject - Json health report object.
    
Return Value:

    None.

--*/
{
    for (auto const & e : JsonObject)
    {

        const wstring propertyName = e.first;
        const json::value propertyValue = e.second;

        SetHealthReportField(propertyName, propertyValue);
    }
}

json::value 
HealthReport::TranslateToJsonValue() const
/*++

Routine Description:

    Translate health report in Json object.

Arguments:

    None.

Return Value:

    Json health report object.

--*/
{
    json::value jsonHealthReport;

    jsonHealthReport[GUEST_OS_FAMILY_STR] = value::string(GuestOSFamily);
    jsonHealthReport[GUEST_OS_VERSION_STR] = value::string(GuestOSVersion);
    jsonHealthReport[DRIVER_INSTALL_ATTEMPTED_STR] = value::boolean(DriverInstallAttempted);
    jsonHealthReport[DRIVER_INSTALL_SUCCEEDED_STR] = value::boolean(DriverInstallSucceeded);
    jsonHealthReport[DRIVER_INSTALL_FAILED_STR] = value::boolean(DriverInstallFailed);
    jsonHealthReport[DRIVER_INSTALL_NOT_ATTEMPTED_STR] = value::boolean(DriverInstallNotAttempted);
    jsonHealthReport[TOTAL_VF_DEVICES_COUNT_STR] = value::number(TotalVFDevicesCount);
    jsonHealthReport[NON_FUNCTIONAL_VF_DEVICES_COUNT_STR] = value::number(NonFunctionalVFDevicesCount);
    jsonHealthReport[INSTALLATION_ERROR_CODE] = value::number(ErrorCode);

    return jsonHealthReport;
}