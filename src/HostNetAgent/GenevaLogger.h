/*++

Copyright (c) 2010  Microsoft Corporation

Module Name:

    GenevaLogger.h

Abstract:

    Class declaration and information for Mdm logger.

Author:

--*/

#pragma once

#include <Windows.h>
#include <IfxMetrics.h>
#include "XmlParserHelper.h"
#include <map>
#include <locale>
#include <codecvt>
#include "HealthReport.h"
#include "CNIReport.h"
#include "NPMReport.h"
#include "DNCReport.h"

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

#define MDM_DIMENSION_NAME_NPM_VERSION_STR              "NpmVersion"
#define MDM_DIMENSION_NAME_KUBERNETES_VERSION_STR       "KubernetesVersion"
#define MDM_DIMENSION_NAME_SUBSCRIPTION_ID_STR          "SubscriptionId"
#define MDM_DIMENSION_NAME_RESOURCE_GROUP_NAME_STR      "ResourceGroupName"
#define MDM_DIMENSION_NAME_CLUSTER_ID_STR               "ClusterId"
#define MDM_DIMENSION_NAME_NODE_NAME_STR                "NodeName"

#define MDM_DIMENSION_NAME_PARTITION_KEY_STR            "PartitionKey"
#define MDM_DIMENSION_NAME_LOCATION_STR                 "Location"
#define MDM_DIMENSION_NAME_CPU_USAGE_STR                "CPUUsage"
#define MDM_DIMENSION_NAME_MEMORY_USAGE_STR             "MemoryUsage"
#define MDM_DIMENSION_NAME_PROCESSES_STR                "Processes"

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

// MDM metric names for NPM metrics.
#define MDM_METRIC_NPM_RUNNING_STR      "NPMRunning"
#define MDM_METRIC_NPM_ERROR_STR        "NPMError"

// MDM metric names for DNC metrics.
#define MDM_METRIC_DNC_RUNNING_STR      "DNCRunning"
#define MDM_METRIC_DNC_HEARTBEAT_STR    "DNCHeartbeat"
#define MDM_METRIC_DNC_ERROR_STR        "DNCError"

#define MAX_STR_LEN (size_t) ((INT_MAX/2) -1)

#define ENABLE_MA_NODE_PATH               L"//NetAgentConfiguration/MonitoringConfig/EnableMA"
#define MDM_ACCOUNT_PATH                  L"//NetAgentConfiguration/MonitoringConfig/MDMAccount"

#define  NODEID_REG_VALUE_NAME                                    L"NodeId"
#define  CLUSTER_ID_REG_VALUE_NAME                                L"ClusterName"

#define _TRUE_                                                    L"TRUE"

// Get array size in terms of elements
#define ARRAY_SIZE(_Array_) (sizeof(_Array_)/sizeof(_Array_[0]))

class GenevaLogger
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
        LPCSTR Location,
        LPCSTR GuestOSVersion,
        LPCSTR MetricName,
        ULONG64 Value = 1
    );

    HRESULT LogPerContainerCNIMetricToMDM(
        LPCSTR Location,
        LPCSTR ContainerId,
        LPCSTR Name,
        LPCSTR Version,
        LPCSTR OrchestratorName,
        LPCSTR OrchestratorVersion,
        LPCSTR GuestOSVersion,
        LPCSTR KernelVersion,
        LPCSTR MetricName,
        ULONG64 Value = 1
    );

    HRESULT LogPerNPMMetricToMDM(
        string Location,
        string NpmVersion,
        string KubernetesVersion,
        string SubscriptionId,
        string ResourceGroupName,
        string ClusterId,
        string NodeName,
        string NodeId,
        string GuestOSVersion,
        string KernelVersion,
        string MetricName,
        ULONG64 Value = 1
    );

    HRESULT LogPerDNCMetricToMDM(
        string Location,
        string SubscriptionId,
        string ResourceGroupName,
        string PartitionKey,
        string CPUUsage,
        string MemoryUsage,
        string Processes,
        string MetricName,
        ULONG64 Value = 1
    );

    HRESULT LogDownloadMetricToMDM(LPCWSTR ContainerId, ULONG64 Value = 1) const;

    HRESULT InitializeMdm(LPCWSTR SettingsFileName, LPCWSTR PackageVersion);

    HRESULT InitializeMds();

    void SetNodeInformationFromRegistry();

    void LogHealthReport(HealthReport& HealthReport);

    void LogCNIReport(wstring ContainerId, CNIReport& CNIReport);

    void LogNPMReport(NPMReport& NPMReport);

    void LogDNCReport(DNCReport& DNCReport);

    static GenevaLogger* GetInstance();

    static wstring s2ws(const string& s);

    static string ws2s(const wstring& ws);

private:

    GenevaLogger() {}

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

    // map from metric name to Per-NPM MDM metric
    map<string, MeasureMetric10D*> PerNPMMetrics;

    // map from metric name to Per-DNC MDM metric
    map<string, MeasureMetric7D*> PerDNCMetrics;
};


