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
#include "CNSReport.h"
#include "Payload.h"

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
#define MDM_METRIC_DNC_RUNNING_STR                                 "DNCRunning"
#define MDM_METRIC_DNC_HEARTBEAT_STR                               "DNCHeartbeat"
#define MDM_METRIC_DNC_ERROR_STR                                   "DNCError"
#define MDM_METRIC_DNC_PRIMARY_NOT_AVAILABLE                       "PrimaryNotAvailable"
#define MDM_METRIC_DNC_NETWORK_NOT_FOUND                            "NetworkNotFound"
#define MDM_METRIC_DNC_SUBNET_NOT_FOUND                             "SubnetNotFound"
#define MDM_METRIC_DNC_RESERVATION_NOT_FOUND                        "ReservationNotFound"
#define MDM_METRIC_DNC_JOIN_DENIED                                  "JoinDenied"
#define MDM_METRIC_DNC_NODE_NOT_FOUND                               "NodeNotFound"
#define MDM_METRIC_DNC_ALLOCATION_PROCESS_FAILED_INTERNAL_ERROR     "AllocationProcessFailedInternalError"
#define MDM_METRIC_DNC_ALLOCATION_FAILED                            "AllocationFailed"
#define MDM_METRIC_DNC_NODE_PROCESSING_FAILED_DUE_TO_INTERNAL_ERROR "NodeProcessingFailedDueToInternalError"
#define MDM_METRIC_DNC_STATUS_INTERNAL_SERVER_ERROR                 "StatusInternalServerError"
#define MDM_METRIC_DNC_NETWORK_STATUS_INVALID                       "NetworkStatusInvalid"
#define MDM_METRIC_DNC_NETWORK_STATE_INVALID                        "NetworkStateInvalid"
#define MDM_METRIC_DNC_SUBNET_CONFIGURATION_NOT_FOUND               "SubnetConfigurationNotFound"
#define MDM_METRIC_DNC_SUBNET_IS_NOT_DELEGATED                      "SubnetIsNotDelegated"
#define MDM_METRIC_DNC_SUBNET_IS_NOT_JOINED                         "SubnetIsNotJoined"
#define MDM_METRIC_DNC_NETWORK_IN_USE                               "NetworkInUse"
#define MDM_METRIC_DNC_RESERVATION_IN_USE                           "ReservationInUse"
#define MDM_METRIC_DNC_INVALID_IP_SUBNET                            "InvalidIPSubnet"
#define MDM_METRIC_DNC_INVALID_IP_CONSTRAINT                        "InvalidIPConstraint"
#define MDM_METRIC_DNC_IP_CONSTRAINT_IS_OUTSIDE_SUBNET_RANGE        "IPConstraintIsOutsideSubnetRange"
#define MDM_METRIC_DNC_INVALID_AUTHENTICATION_TOKEN                 "InvalidAuthenticationToken"
#define MDM_METRIC_DNC_INVALID_SUBNET_NAME                          "InvalidSubnetName"
#define MDM_METRIC_DNC_INVALID_NODE_ID                              "InvalidNodeID"
#define MDM_METRIC_DNC_INVALID_RESERVATION_ID                       "InvalidReservationID"
#define MDM_METRIC_DNC_INVALID_NETWORK_ID                           "InvalidNetworkID"
#define MDM_METRIC_DNC_INVALID_NETWORK_CONTAINER_ID                 "InvalidNetworkContainerID"
#define MDM_METRIC_DNC_INVALID_ORCHESTRATOR_TYPE                    "InvalidOrchestratorType"
#define MDM_METRIC_DNC_INVALID_CONTAINER_TYPE                       "InvalidContainerType"
#define MDM_METRIC_DNC_NODE_NOT_REGISTERED                          "NodeNotRegistered"
#define MDM_METRIC_DNC_UNSUPPROTED_NETWORK_TYPE                     "UnSupprotedNetworkType"
#define MDM_METRIC_DNC_OUT_OF_VLAN_IDS                              "OutOfVlanIDs"
#define MDM_METRIC_DNC_INVALID_NODE_IP_ADDRESS                      "InvalidNodeIPAddress"
#define MDM_METRIC_DNC_JSON_BODY_UN_MARSHAL_FAILED                  "JsonBodyUnMarshalFailed"
#define MDM_METRIC_DNC_JSON_MARSHALLING_FAILED                      "JsonMarshallingFailed"
#define MDM_METRIC_DNC_JSON_UN_MARSHALLING_FAILED                   "JsonUnMarshallingFailed"
#define MDM_METRIC_DNC_NODE_IN_USE                                  "NodeInUse"
#define MDM_METRIC_DNC_NETWORK_CONTAINER_NOT_FOUND                  "NetworkContainerNotFound"
#define MDM_METRIC_DNC_API_VERSION_MISSING                          "APIVersionMissing"
#define MDM_METRIC_DNC_API_VERSION_NOT_SUPPORTED                    "APIVersionNotSupported"
#define MDM_METRIC_DNC_RESERVATION_STATE_INVALID                    "ReservationStateInvalid"
#define MDM_METRIC_DNC_NODE_CONSTRAINT_MISMATCH                     "NodeConstraintMismatch"
#define MDM_METRIC_DNC_NETWORK_JOIN_FAILED                          "NetworkJoinFailed"
#define MDM_METRIC_DNC_DEOCODE_RESPONSE_FAILED                      "DeocodeResponseFailed"
#define MDM_METRIC_DNC_PARSE_RESPONSE_CODE_FAILED                   "ParseResponseCodeFailed"
#define MDM_METRIC_DNC_PUBLISHING_FAILED                            "PublishingFailed"
#define MDM_METRIC_DNC_UNPUBLISHING_FAILED                          "UnpublishingFailed"
#define MDM_METRIC_DNC_PROCESSING_PENDING                           "ProcessingPending"
#define MDM_METRIC_DNC_ASSOCIATION_FAILED                           "AssociationFailed"
#define MDM_METRIC_DNC_DISASSOCIATION_FAILED                        "DisAssociationFailed"
#define MDM_METRIC_DNC_JOIN_FAILED                                  "JoinFailed"
#define MDM_METRIC_DNC_JOIN_PENDING                                 "JoinPending"
#define MDM_METRIC_DNC_INVALID_NETWORK_CONTAINER_REQUEST            "InvalidNetworkContainerRequest"
#define MDM_METRIC_DNC_NETWORK_CONFIGURATION_RETRIEVAL_FAILED       "NetworkConfigurationRetrievalFailed"
#define MDM_METRIC_DNC_GET_TAG_DEFINITION_FAILED                    "GetTagDefinitionFailed"
#define MDM_METRIC_DNC_AGENT_TAG_DEFINITION_RETRIEVAL_FAILED        "AgentTagDefinitionRetrievalFailed"
#define MDM_METRIC_DNC_REPOSITORY_TAG_RETRIEVAL_FAILED              "RepositoryTagRetrievalFailed"
#define MDM_METRIC_DNC_TAG_RESOLUTION_PENDING                       "TagResolutionPending"
#define MDM_METRIC_DNC_INVALID_NETWORK_STATE_FOR_DELETE             "InvalidNetworkStateForDelete"
#define MDM_METRIC_DNC_CNS_NC_VERSION_RETRIEVAL_FAILED              "CnsNCVersionRetrievalFailed"
#define MDM_METRIC_DNC_INVALID_AZURE_STORAGE_INFO                   "InvalidAzureStorageInfo"
#define MDM_METRIC_DNC_STORAGE_NOT_INITIALIZED                      "StorageNotInitialized"
#define MDM_METRIC_DNC_NODE_CONTAINER_CAPACITY_EXHAUSTED            "NodeContainerCapacityExhausted"

// MDM metric names for CNS metrics
#define MDM_METRIC_CNS_RUNNING_STR                      "CNSRunning"
#define MDM_METRIC_CNS_HEARTBEAT_STR                    "CNSHeartbeat"
#define MDM_METRIC_CNS_ERROR_STR                        "CNSError"
#define MDM_METRIC_CNS_UNSUPPORTED_NETWORK_TYPE         "UnsupportedNetworkType"
#define MDM_METRIC_CNS_INVALID_PARAMETER                "InvalidParameter"
#define MDM_METRIC_CNS_UNREACHABLE_HOST                 "UnreachableHost"
#define MDM_METRIC_CNS_RESERVATION_NOT_FOUND            "ReservationNotFound"
#define MDM_METRIC_CNS_MALFORMED_SUBNET                 "MalformedSubnet"
#define MDM_METRIC_CNS_UNREACHABLE_DOCKER_DAEMON        "UnreachableDockerDaemon"
#define MDM_METRIC_CNS_UNSPECIFIED_NETWORK_NAME         "UnspecifiedNetworkName"
#define MDM_METRIC_CNS_NOT_FOUND                        "NotFound"
#define MDM_METRIC_CNS_ADDRESS_UNAVAILABLE              "AddressUnavailable"
#define MDM_METRIC_CNS_NETWORK_CONTAINER_NOT_SPECIFIED  "NetworkContainerNotSpecified"
#define MDM_METRIC_CNS_CALL_TO_HOST_FAILED              "CallToHostFailed"
#define MDM_METRIC_CNS_UNKNOWN_CONTAINER_ID             "UnknownContainerID"
#define MDM_METRIC_CNS_UNSUPPORTED_ORCHESTRATOR_TYPE    "UnsupportedOrchestratorType"
#define MDM_METRIC_CNS_UNEXPECTED_ERROR                 "UnexpectedError"
#define MDM_METRIC_CNS_UNKNOWN_ERROR                    "UnknownError"

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
        string MetricName,
        ULONG64 Value = 1
    );

    HRESULT LogPerCNSMetricToMDM(
        string Location,
        string SubscriptionId,
        string ResourceGroupName,
        string PartitionKey,
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

    void LogCNSReport(CNSReport& CNSReport);

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
    map<string, MeasureMetric4D*> PerDNCMetrics;

    // map from metric name to Per-CNS MDM metric
    map<string, MeasureMetric4D*> PerCNSMetrics;
};


