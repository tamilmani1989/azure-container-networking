/*++

Copyright (c) 2010  Microsoft Corporation

Module Name:

    MdmLogger.h

Abstract:

    Class declaration and information for Mdm logger.

Author:

--*/

#pragma once

#include <Windows.h>
#include <IfxMetrics.h>
#include "XmlParserHelper.h"
#include <map>
#include "HealthReport.h"
#include "CNIReport.h"

// Constant string for vnet mdm account
#define MDM_ACCOUNT L"MdmAccount"

#define NODE_PROPERTIES_REG_KEY     L"SOFTWARE\\AzureHL\\NodeProperties"

#define MDM_NAMESPACE                      "NetAgent"

#define MDM_DIMENSION_NAME_CONTAINER_ID_STR             "ContainerId"
#define MDM_DIMENSION_NAME_NODE_ID_STR                  "NodeId"
#define MDM_DIMENSION_NAME_CLUSTER_NAME_STR             "Cluster"
#define MDM_DIMENSION_NAME_GUEST_OS_STR                 "GuestOS"
#define MDM_DIMENSION_NAME_GUEST_OS_VERSION_STR         "GuestOSVersion"
#define	MDM_DIMENSION_NAME_GUEST_KERNEL_VERSION_STR     "KernelVersion"
#define MDM_DIMENSION_NAME_NETAGENT_VERSION_STR         "NetAgentVersion"

#define MDM_DIMENSION_NAME_PLUGIN_ORCHESTRATOR_NAME_STR         "OrchestratorName"
#define	MDM_DIMENSION_NAME_PLUGIN_ORCHESTRATOR_VERSION_STR      "OrchestratorVersion"
#define	MDM_DIMENSION_NAME_PLUGIN_NAME_STR                      "PluginName"
#define	MDM_DIMENSION_NAME_PLUGIN_VERSION_STR                   "PluginVersion"


// MDM metric names for per-container metrics
#define MDM_METRIC_PACKAGE_DOWNLOAD_REQUEST_STR                 "PackageDownloadAttempted"
#define MDM_METRIC_GUEST_DRIVER_INSTALL_NOT_ATTEMPTED_STR       "DriverInstallNotAttempted"
#define MDM_METRIC_GUEST_DRIVER_INSTALL_ATTEMPTED_STR           "DriverInstallAttempted"
#define MDM_METRIC_GUEST_DRIVER_INSTALL_SUCCEEDED_STR           "DriverInstallSucceeded"
#define MDM_METRIC_GUEST_DRIVER_INSTALL_FAILED_STR              "DriverInstallFailed"
#define MDM_METRIC_GUEST_TOTAL_VF_STR                           "VFCount"
#define MDM_METRIC_GUEST_VF_IN_ERROR_STR                        "VFInError"
#define MDM_METRIC_GUEST_DRIVER_INSTALL_ERROR_CODE_STR          "DriverInstallErrorCode"
#define MDM_METRIC_CNI_RUNNING_STR                              "CNIRunning"
#define MDM_METRIC_CNI_SUCCEEDED_STR                            "CNISucceeded"
#define MDM_METRIC_CNI_ERROR_STR                                "CNIError"

#define MAX_STR_LEN (size_t) ((INT_MAX/2) -1)

#define ENABLE_MA_NODE_PATH               L"//NetAgentConfiguration/MonitoringConfig/EnableMA"
#define MDM_ACCOUNT_PATH                  L"//NetAgentConfiguration/MonitoringConfig/MDMAccount"

#define  NODEID_REG_VALUE_NAME                                    L"NodeId"
#define  CLUSTER_ID_REG_VALUE_NAME                                L"ClusterName"

#define _TRUE_                                                    L"TRUE"

// Get array size in terms of elements
#define ARRAY_SIZE(_Array_) (sizeof(_Array_)/sizeof(_Array_[0]))

class MdmLogger
{
public:

    bool IsMDMLoggingEnabled()
    {
        return EnableMDMLogging;
    };

    string ClusterNameA;

    string NodeIdA;

    string MdmAccountNameA;

    string NetAgentVersionA;

    HRESULT LogPerContainerMetricToMDM(
        LPCSTR ContainerId,
        LPCSTR GuestOS,
        LPCSTR GuestOSVersion,
        LPCSTR MetricName,
        ULONG64 Value);

    HRESULT LogPerContainerCNIMetricToMDM(
        LPCSTR ContainerId,
        LPCSTR Name,
        LPCSTR Version,
        LPCSTR OrchestratorName,
        LPCSTR OrchestratorVersion,
        LPCSTR GuestOS,
        LPCSTR GuestOSVersion,
        LPCSTR KernelVersion,
        LPCSTR MetricName,
        ULONG64 Value);

    HRESULT LogDownloadMetricToMDM(LPCWSTR ContainerId, ULONG64 Value) const;

    HRESULT InitializeMdm(LPCWSTR SettingsFileName, LPCWSTR PackageVersion);

    void SetNodeInformationFromRegistry();

    void LogHealthReport(HealthReport& HealthReport);

    void LogCNIReport(wstring ContainerId, CNIReport& CNIReport);

    static MdmLogger* GetInstance();

private:

    MeasureMetric6D* gm_DriverInstallationAttemptedCountMetric;

    MeasureMetric6D* gm_DriverInstallationFailedCountMetric;

    MeasureMetric6D* gm_DriverInstallationSuccededCountMetric;

    MeasureMetric6D* gm_VFInErrorCountMetric;

    // Flag for enable/disable MDM logging
    bool EnableMDMLogging;

    // Container download metrics
    MeasureMetric4D* DownloadMetrics;

    // map from metric name to Per-container MDM metric
    map<string, MeasureMetric6D*> PerContainerMetrics;

    // map from metric name to Per-container CNI MDM metric
    map<string, MeasureMetric10D*> PerContainerCNIMetrics;
};


