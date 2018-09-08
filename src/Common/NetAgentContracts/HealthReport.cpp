/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    HealReport.cpp

Abstract:

    Implementation of Health report functions.

Environment:

    User Mode - Win32

--*/

#include "HealthReport.h"
#include "Logging.h"

using namespace std;
using namespace web;
using namespace http;
using namespace json;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

HealthReport::HealthReport()
/*++

Routine Description:

	Default health report constructor.

Arguments:

	None.

Return Value:

None.
--*/
{
	InitializeFieldIdMap();
}

HealthReport::HealthReport(
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
	InitializeFieldIdMap();
    SetMembersFromJson(JsonObject);
}

void
HealthReport::InitializeFieldIdMap()
/*++

Routine Description:

    Initialize the report field FieldName to key map.

Arguments:

    None.
    
Return Value:

    None.

--*/
{
    HealthReportFieldIdMap[GUEST_OS_FAMILY_STR] = HealthEnum::ReportKey::GuestOSFamily;
    HealthReportFieldIdMap[GUEST_OS_VERSION_STR] = HealthEnum::ReportKey::GuestOSVersion;
	HealthReportFieldIdMap[DEBUG_MESSAGE_STR] = HealthEnum::ReportKey::DebugMessage;
	HealthReportFieldIdMap[DRIVER_INSTALL_PRESETUP_SUCCEEDED_STR] = HealthEnum::ReportKey::DriverInstallPreSetupSucceeded;
	HealthReportFieldIdMap[DRIVER_INSTALL_PRESETUP_FAILED_STR] = HealthEnum::ReportKey::DriverInstallPreSetupFailed;
	HealthReportFieldIdMap[DRIVER_INSTALL_PRESETUP_NOT_ATTEMPTED_STR] = HealthEnum::ReportKey::DriverInstallPreSetupNotAttempted;
    HealthReportFieldIdMap[DRIVER_INSTALL_ATTEMPTED_STR] = HealthEnum::ReportKey::DriverInstallAttempted;
    HealthReportFieldIdMap[DRIVER_INSTALL_SUCCEEDED_STR] = HealthEnum::ReportKey::DriverInstallSucceeded;
    HealthReportFieldIdMap[DRIVER_INSTALL_FAILED_STR] = HealthEnum::ReportKey::DriverInstallFailed;
    HealthReportFieldIdMap[DRIVER_INSTALL_NOT_ATTEMPTED_STR] = HealthEnum::ReportKey::DriverInstallNotAttempted;
    HealthReportFieldIdMap[TOTAL_VF_DEVICES_COUNT_STR] = HealthEnum::ReportKey::TotalVFDevices;
    HealthReportFieldIdMap[NON_FUNCTIONAL_VF_DEVICES_COUNT_STR] = HealthEnum::ReportKey::NonFunctionalVFDevices;
    HealthReportFieldIdMap[INSTALLATION_ERROR_CODE] = HealthEnum::ReportKey::InstallationErrorCode;
}

void 
HealthReport::SetFields(
    wstring Name, 
    json::value Value)
/*++

Routine Description:

    Set JsonValue in report field.

Arguments:

    Name - Name of the field.

    Value - Json value to be associated with the field.

Return Value:

    None.

--*/
{
    switch (HealthReportFieldIdMap[Name])
    {
    case HealthEnum::ReportKey::GuestOSFamily:
        GuestOSFamily.assign(Value.as_string());
        break;

    case HealthEnum::ReportKey::GuestOSVersion:
        GuestOSVersion.assign(Value.as_string());
        break;

	case HealthEnum::ReportKey::DebugMessage:
		DebugMessage.assign(Value.as_string());
		break;

	case HealthEnum::ReportKey::DriverInstallPreSetupSucceeded:
		DriverInstallPreSetupSucceeded = Value.as_bool();
		break;

	case HealthEnum::ReportKey::DriverInstallPreSetupFailed:
		DriverInstallPreSetupFailed = Value.as_bool();
		break;
	
	case HealthEnum::ReportKey::DriverInstallPreSetupNotAttempted:
		DriverInstallPreSetupNotAttempted = Value.as_bool();
		break;
    
	case HealthEnum::ReportKey::DriverInstallAttempted:
        DriverInstallAttempted = Value.as_bool();
        break;

    case HealthEnum::ReportKey::DriverInstallSucceeded:
        DriverInstallSucceeded = Value.as_bool();
        break;

    case HealthEnum::ReportKey::DriverInstallFailed:
        DriverInstallFailed = Value.as_bool();
        break;

    case HealthEnum::ReportKey::DriverInstallNotAttempted:
        DriverInstallNotAttempted = Value.as_bool();
        break;

    case HealthEnum::ReportKey::TotalVFDevices:
        TotalVFDevicesCount = Value.as_integer();
        break;

    case HealthEnum::ReportKey::NonFunctionalVFDevices:
        NonFunctionalVFDevicesCount = Value.as_integer();
        break;

    case HealthEnum::ReportKey::InstallationErrorCode:
        ErrorCode = Value.as_integer();
        break;

    default:
        wprintf(L"Unknown key:{%s}", Name.c_str());

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

        SetFields(propertyName, propertyValue);
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
	jsonHealthReport[DEBUG_MESSAGE_STR] = value::string(DebugMessage);
	jsonHealthReport[DRIVER_INSTALL_PRESETUP_SUCCEEDED_STR] = value::boolean(DriverInstallPreSetupSucceeded);
	jsonHealthReport[DRIVER_INSTALL_PRESETUP_FAILED_STR] = value::boolean(DriverInstallPreSetupFailed);
	jsonHealthReport[DRIVER_INSTALL_PRESETUP_NOT_ATTEMPTED_STR] = value::boolean(DriverInstallPreSetupNotAttempted);
    jsonHealthReport[DRIVER_INSTALL_ATTEMPTED_STR] = value::boolean(DriverInstallAttempted);
    jsonHealthReport[DRIVER_INSTALL_SUCCEEDED_STR] = value::boolean(DriverInstallSucceeded);
    jsonHealthReport[DRIVER_INSTALL_FAILED_STR] = value::boolean(DriverInstallFailed);
    jsonHealthReport[DRIVER_INSTALL_NOT_ATTEMPTED_STR] = value::boolean(DriverInstallNotAttempted);
    jsonHealthReport[TOTAL_VF_DEVICES_COUNT_STR] = value::number(TotalVFDevicesCount);
    jsonHealthReport[NON_FUNCTIONAL_VF_DEVICES_COUNT_STR] = value::number(NonFunctionalVFDevicesCount);
    jsonHealthReport[INSTALLATION_ERROR_CODE] = value::number(ErrorCode);

    return jsonHealthReport;
}

void HealthReport::Print()
/*++

Routine Description:

	Print each member of HealthReport.

Arguments:

	None.

Return Value:

	None.

--*/
{
	wstring s;
	s += L"ContainerId: " + ContainerId + L"\n";	
	s += L"GuestOSVersion: " + GuestOSVersion + L"\n";
	s += L"GuestOSFamily: " + GuestOSFamily + L"\n";
	s += L"DebugMessage: " + DebugMessage + L"\n";
	s += L"DriverInstallPreSetupSucceeded: " + ConvertBoolToWstring(DriverInstallPreSetupSucceeded) +L"\n";
	s += L"DriverInstallPreSetupFailed: " + ConvertBoolToWstring(DriverInstallPreSetupFailed) +L"\n";
	s += L"DriverInstallNotAttempted: " + ConvertBoolToWstring(DriverInstallNotAttempted) +L"\n";
	s += L"DriverInstallAttempted: " + ConvertBoolToWstring(DriverInstallAttempted) +L"\n";
	s += L"DriverInstallSucceeded: " + ConvertBoolToWstring(DriverInstallSucceeded) +L"\n";
	s += L"TotalVFDevicesCount: " + to_wstring(TotalVFDevicesCount) +L"\n";
	s += L"NonFunctionalVFDevicesCount: " + to_wstring(NonFunctionalVFDevicesCount) +L"\n";
	s += L"ErrorCode: " + to_wstring(ErrorCode) +L"\n";

	LOGINFOW(L"%s", s.c_str());
}