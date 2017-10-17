/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    HealReport.h

Abstract:

    Header to define the Health report functions.

Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//
#pragma once

#include <stdio.h>
#include <stdlib.h>
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

#define GUEST_OS_VERSION_STR                                L"GuestOSVersion"
#define GUEST_OS_FAMILY_STR                                 L"GuestOSFamily"
#define DRIVER_INSTALL_NOT_ATTEMPTED_STR                    L"DriverInstallNotAttempted"
#define DRIVER_INSTALL_ATTEMPTED_STR                        L"DriverInstallAttempted"
#define DRIVER_INSTALL_SUCCEEDED_STR                        L"DriverInstallSucceeded"
#define DRIVER_INSTALL_FAILED_STR                           L"DriverInstallFailed"
#define TOTAL_VF_DEVICES_COUNT_STR                          L"TotalVFDevicesCount"
#define NON_FUNCTIONAL_VF_DEVICES_COUNT_STR                 L"NonFunctionalVFDevicesCount"
#define INSTALLATION_ERROR_CODE                             L"InstallationErrorCode"

//
// Enums for fields in the health report.
//

enum ReportKey {
    GuestOSVersion,
    GuestOSFamily,
    DriverInstallNotAttempted,
    DriverInstallAttempted,
    DriverInstallSucceeded,
    DriverInstallFailed,
    TotalVFDevices,
    NonFunctionalVFDevices,
    InstallationErrorCode
};

// Helath report field name to Id map
static  std::map<std::wstring, ReportKey> HealthReportFieldIdMap;
using namespace std;
using namespace web;

class HealthReport {

public:
    // Id of the container.
    wstring ContainerId;

    // Guest OS version.
    wstring GuestOSVersion;

    // Guest OS Family.
    wstring GuestOSFamily;

    // Driver installation not attempted.
    bool DriverInstallNotAttempted;

    // Driver installation not attmepted.
    bool DriverInstallAttempted;

    // Driver installation succeeded.
    bool DriverInstallSucceeded;

    // Driver installation failed.
    bool DriverInstallFailed;

    // Total VF devices count in the VM.
    int TotalVFDevicesCount;

    // Total non functioning VF devices in the VM.
    int NonFunctionalVFDevicesCount;

    // Error Code if installation fails.
    int ErrorCode;

    void
    InitializeHealthReport(
        std::wstring ContainerId,
        json::object &JsonObject);

    static
    void
    InitializeHealthReportFieldIdMap();

    void
    SetHealthReportField(
        std::wstring Name,
        web::json::value Value);

    void SetMembersFromJson(json::object &JsonObject);

    json::value TranslateToJsonValue() const;
};