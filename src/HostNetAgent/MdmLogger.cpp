/*++

Copyright (c) 2010  Microsoft Corporation

Module Name:

    MdmLogger.cpp

Abstract:

    Implementation for Mdm logger methods.

Author:

--*/
#include "MdmLogger.h"
#include "HostNetAgent.h"

using namespace std;

//
// Per container MDM guest metric strings.
//

LPCSTR PER_CONTAINER_MDM_GUEST_METRICS[] = {
    MDM_METRIC_GUEST_DRIVER_INSTALL_NOT_ATTEMPTED_STR,
    MDM_METRIC_GUEST_DRIVER_INSTALL_ATTEMPTED_STR,
    MDM_METRIC_GUEST_DRIVER_INSTALL_SUCCEEDED_STR,
    MDM_METRIC_GUEST_DRIVER_INSTALL_FAILED_STR,
    MDM_METRIC_GUEST_TOTAL_VF_STR,
    MDM_METRIC_GUEST_VF_IN_ERROR_STR,
    MDM_METRIC_GUEST_DRIVER_INSTALL_ERROR_CODE_STR,
};

LPCSTR PER_CONTAINER_MDM_CNI_METRICS[] = {
    MDM_METRIC_CNI_RUNNING_STR,
    MDM_METRIC_CNI_ERROR_STR
};

DWORD
ToAnsi(
    __in_z PCWSTR StringW,
    __deref_out PSTR *StringA
    )
/*++

Routine Description:

    Converts a unicode string to ansi equivalent.

Arguments:

    StringW - String to convert to ansi

    StringA - Ansi version of string

Return Value:

    ERROR_SUCCESS on success. On failure appropriate Win32 Error
    Code is returned.

--*/
{
    DWORD dwError = ERROR_SUCCESS;
    int cbMultiByte = 0;
    PSTR pszBuffer = NULL;

    //
    // Find out how much space we need for the converted string
    //
    // Note that we tell the function to disable any best-fit substitutions
    // for characters that don't exactly map. This ensures that the caller
    // always gets back the default substituion character for failed mappings
    // rather than some unexpected character (e.g. Infinity symbol translates
    // into 8 in some code pages).
    //

    cbMultiByte = WideCharToMultiByte(CP_ACP,
                                      WC_NO_BEST_FIT_CHARS,
                                      StringW,
                                      -1,
                                      NULL,
                                      0,
                                      NULL,
                                      NULL);
    if (cbMultiByte == 0) {
        dwError = GetLastError();
        goto Cleanup;
    }

    ASSERT(cbMultiByte > 0);

    //
    // Allocate the buffer required by the conversion function.
    //

    pszBuffer = new (std::nothrow) char[cbMultiByte];
    if (pszBuffer == NULL) {
        dwError = ERROR_NOT_ENOUGH_MEMORY;
        goto Cleanup;
    }

    //
    // Call the conversion routine to perform the actual conversion.
    //

    if (!WideCharToMultiByte(CP_ACP,
                             WC_NO_BEST_FIT_CHARS,
                             StringW,
                             -1,
                             pszBuffer,
                             cbMultiByte,
                             NULL,
                             NULL)) {
        dwError = GetLastError();
        ANSI_STR_FREE(pszBuffer);
        goto Cleanup;
    }

    //
    // Set the output parameter.
    //

    *StringA = pszBuffer;

Cleanup:

    return dwError;
}

HRESULT
MdmLogger::LogPerContainerCNIMetricToMDM(
    LPCSTR ContainerId,
    LPCSTR Name,
    LPCSTR Version,
    LPCSTR OrchestratorName,
    LPCSTR OrchestratorVersion,
    LPCSTR GuestOS,
    LPCSTR GuestOSVersion,
    LPCSTR GuestKernelVersion,
    LPCSTR MetricName,
    ULONG64 Value
)
/*++

Description:

    Log per container counter MDM metric(CNI).

Arguments:

    ContainerId - Id of the container.

    Name - Name of Plugin.

    Version- Version of Plugin.

    OrchestratorName - Orchestrator Name.

    OrchestratorVersion - Orchestrator Version.

    GuestOS - Guest OS.

    GuestOSVersion - OS Version.

    GuestKernelVersion - Kernel Version.

    MetricName - Name of metric.

    Value - Value of the metric.

Return Value:

    S_OK on success, error otherwise.

--*/

{
    HRESULT hr = S_OK;
    string metricName(MetricName);
    map<std::string, MeasureMetric10D*>::iterator metricItr;
    MeasureMetric10D* metric = NULL;

    if (!EnableMDMLogging) {

        goto Cleanup;
    }

    metricItr = PerContainerCNIMetrics.find(metricName);

    if (metricItr != PerContainerCNIMetrics.end()) {

        metric = metricItr->second;
        hr = metric->LogValue(
            Value,
            NodeIdA.c_str(),
            ClusterNameA.c_str(),
            ContainerId,
            GuestOS == NULL ? "" : GuestOS,
            GuestOSVersion == NULL ? "" : GuestOSVersion,
            GuestKernelVersion == NULL? "": GuestKernelVersion,
            Name == NULL? "": Name,
            Version == NULL? "": Version, 
            OrchestratorName == NULL? "": OrchestratorName,
            OrchestratorVersion == NULL ? "" : OrchestratorVersion
        );
    }
    else {

        LOGERR("Failed to find per-Node MDM metric : %hs", MetricName);
    }

Cleanup:

    return hr;
}

HRESULT
MdmLogger::LogPerContainerMetricToMDM(
    LPCSTR ContainerId,
    LPCSTR GuestOS,
    LPCSTR GuestOSVersion,
    LPCSTR MetricName,
    ULONG64 Value
    )
/*++

Description:

    Log per container counter MDM metric.

Arguments:

    ContainerId - Id of the container.

    GuestOS - Guest OS.

    GuestOSVersion - OS Version.

    MetricName - Name of metric.

    Value - Value of the metric.

Return Value:

    S_OK on success, error otherwise.

--*/

{
    HRESULT hr = S_OK;
    string metricName(MetricName);
    map<std::string, MeasureMetric6D*>::iterator metricItr;
    MeasureMetric6D* metric = NULL;

    if (!EnableMDMLogging) {

        goto Cleanup;
    }

    metricItr = PerContainerMetrics.find(metricName);

    if (metricItr != PerContainerMetrics.end()) {

        metric = metricItr->second;
        hr = metric->LogValue(
                 Value,
                 NodeIdA.c_str(),
                 ClusterNameA.c_str(),
                 ContainerId,
                 GuestOS == NULL ? "": GuestOS,
                 GuestOSVersion == NULL ? "": GuestOSVersion,
                 NetAgentVersionA.c_str());
    } else {

        LOGERR("Failed to find per-Node MDM metric : %hs", MetricName);
    }

Cleanup:

    return hr;
}

HRESULT
MdmLogger::LogDownloadMetricToMDM(
    LPCWSTR ContainerId,
    ULONG64 Value
    ) const
/*++

Description:

    Log per container package download counter MDM metric.

Arguments:

    ContainerId - Id of the container.

    Value - value of the metric.

Return Value:

    S_OK on success, error otherwise.

--*/

{
    HRESULT hr = S_OK;
    PSTR containerIdAnsi = NULL;

    if (!EnableMDMLogging) {

        goto Cleanup;
    }

    if (ContainerId != NULL) {

        hr = HRESULT_FROM_WIN32(ToAnsi(ContainerId, &containerIdAnsi));
        if (hr != S_OK) {

            LOGERRW(L"Failed to convert ContainerId parameter %s to ANSI for PerContainerNode metric", 
                    ContainerId);
            goto Cleanup;
        }
    }

    hr = DownloadMetrics->LogValue(Value,
                                   NodeIdA.c_str(),
                                   ClusterNameA.c_str(),
                                   containerIdAnsi,
                                   NetAgentVersionA.c_str());

Cleanup:

    ANSI_STR_FREE(containerIdAnsi);

    return hr;
}

PWSTR
WcsDup(
    __in_z PCWSTR StringW
    )
/*++

Routine Description:

    Creates a duplicate of passed in string.

Arguments:

    StringW - Create duplicate of this string

Return Value:

    Pointer to duplicated string or NULL of failure

--*/
{
    size_t uLen = wcsnlen_s(StringW, MAX_STR_LEN) + 1;

    PWSTR pwszDup = new (std::nothrow) WCHAR[uLen];
    if (pwszDup) {
        wcsncpy_s(pwszDup, uLen, StringW, uLen);
    }

    return pwszDup;
}

HRESULT
MdmLogger::InitializeMdm(
    LPCWSTR SettingsFileName,
    LPCWSTR PackageVersion
    )
/*++

Routine Description:

    Initialize MDM.

Arguments:

    SettingsFileName - Location of the settings file name.

    PackageVersion - The version of the package.

Return Value:

    S_OK on success, appropriate error code on failure.

--*/
{
    HRESULT hr = S_OK;
    bool comInitialized = false;
    bool xmlParserHelperInitialized = false;
    XmlParserHelper xmlHelper;
    wstring result;
    PSTR mdmAccountName = NULL;
    PSTR netAgentVersion = NULL;
    DWORD err = ERROR_SUCCESS;

    hr = CoInitialize(NULL);
    if (hr != S_OK) {

        LOGERRW(L"ComInitialization failed with error [%0x].", hr);
        goto Cleanup;
    }
    comInitialized = true;

    hr = xmlHelper.Initialize(SettingsFileName);
    if (hr != S_OK) {

        LOGERRW(L"XML initialization failed when loading file {%s} error [%0x].",
                SettingsFileName, hr);
        goto Cleanup;
    }
    xmlParserHelperInitialized = true;

    hr = xmlHelper.QueryNodeValue(ENABLE_MA_NODE_PATH, result);
    if (hr != S_OK) {

        LOGERRW(L"Query node value for path {%s} error [%0x].", 
                ENABLE_MA_NODE_PATH, 
                hr);
        goto Cleanup;
    }
    if(_wcsnicmp(_TRUE_, result.c_str(), sizeof(_TRUE_)) == 0) {

        EnableMDMLogging = true;
    }

    LOGINFOW(L"Result:{%s}", result.c_str());

    hr = xmlHelper.QueryNodeValue(MDM_ACCOUNT_PATH, result);
    if (hr != S_OK) {

        LOGERRW(L"Query node value for path {%s} error [%0x].", MDM_ACCOUNT_PATH, hr);
        goto Cleanup;
    }

    err = ToAnsi(result.c_str(), &mdmAccountName);
    if(err != ERROR_SUCCESS) {

        LOGERRW(L"Failed to convert account name {%s} to ANSI.error [%0x].", 
                result.c_str(), 
                hr);

        hr = HRESULT_FROM_WIN32(err);
        goto Cleanup;
    }
    MdmAccountNameA.assign(mdmAccountName);

    err = ToAnsi(PackageVersion, &netAgentVersion);
    if (err != ERROR_SUCCESS) {

        LOGERRW(L"Failed to convert package version {%s} to ANSI.error [%0x].", 
                PackageVersion, 
                hr);
        hr = HRESULT_FROM_WIN32(err);
        goto Cleanup;
    }

    NetAgentVersionA.assign(netAgentVersion);

    SetNodeInformationFromRegistry();

    // Create per-Container metrics.
    for (int i = 0; i < ARRAYSIZE(PER_CONTAINER_MDM_GUEST_METRICS); i++) {

        LPCSTR metricName = PER_CONTAINER_MDM_GUEST_METRICS[i];
        MeasureMetric6D* metric = MeasureMetric6D::Create(
                                      MdmAccountNameA.c_str(),
                                      MDM_NAMESPACE,
                                      metricName,
                                      MDM_DIMENSION_NAME_NODE_ID_STR,
                                      MDM_DIMENSION_NAME_CLUSTER_NAME_STR,
                                      MDM_DIMENSION_NAME_CONTAINER_ID_STR,
                                      MDM_DIMENSION_NAME_GUEST_OS_STR,
                                      MDM_DIMENSION_NAME_GUEST_OS_VERSION_STR,
                                      MDM_DIMENSION_NAME_NETAGENT_VERSION_STR);

        if (metric != NULL) {

            LOGINFOW(L"Created metric %hs on account %hs.", 
                     metricName, 
                     MdmAccountNameA.c_str());
            PerContainerMetrics.insert(make_pair(string(metricName), metric));
        } else {

            LOGWARNW(L"Failed to create metric %hs on account %hs.", 
                     metricName, 
                     MdmAccountNameA.c_str());
        }
    }
    
    for (int i = 0; i < ARRAYSIZE(PER_CONTAINER_MDM_CNI_METRICS); i++) {
        LPCSTR metricName = PER_CONTAINER_MDM_CNI_METRICS[i];
        MeasureMetric10D* metric = MeasureMetric10D::Create(
            MdmAccountNameA.c_str(),
            MDM_NAMESPACE,
            metricName,
            MDM_DIMENSION_NAME_NODE_ID_STR,
            MDM_DIMENSION_NAME_CLUSTER_NAME_STR,
            MDM_DIMENSION_NAME_CONTAINER_ID_STR,
            MDM_DIMENSION_NAME_GUEST_OS_STR,
            MDM_DIMENSION_NAME_GUEST_OS_VERSION_STR,
            MDM_DIMENSION_NAME_GUEST_KERNEL_VERSION_STR,
            MDM_DIMENSION_NAME_PLUGIN_NAME_STR,
            MDM_DIMENSION_NAME_PLUGIN_VERSION_STR,
            MDM_DIMENSION_NAME_PLUGIN_ORCHESTRATOR_NAME_STR,
            MDM_DIMENSION_NAME_PLUGIN_ORCHESTRATOR_VERSION_STR);

        if (metric != NULL) {

            LOGINFOW(L"Created metric %hs on account %hs.",
                metricName,
                MdmAccountNameA.c_str());
            PerContainerCNIMetrics.insert(make_pair(string(metricName), metric));
        }
        else {

            LOGWARNW(L"Failed to create metric %hs on account %hs.",
                metricName,
                MdmAccountNameA.c_str());
        }
    }
    
    DownloadMetrics = MeasureMetric4D::Create(
                          MdmAccountNameA.c_str(),
                          MDM_NAMESPACE,
                          MDM_METRIC_PACKAGE_DOWNLOAD_REQUEST_STR,
                          MDM_DIMENSION_NAME_NODE_ID_STR,
                          MDM_DIMENSION_NAME_CLUSTER_NAME_STR,
                          MDM_DIMENSION_NAME_CONTAINER_ID_STR,
                          MDM_DIMENSION_NAME_NETAGENT_VERSION_STR);

Cleanup:

    if(xmlParserHelperInitialized)
    {
        xmlHelper.UnInitialize();
    }

    if (comInitialized)
    {
        CoUninitialize();
    }

    ANSI_STR_FREE(mdmAccountName);
    ANSI_STR_FREE(netAgentVersion);

    return hr;
}

void
MdmLogger::SetNodeInformationFromRegistry()
/*++

Routine Description:

    Retrieve node information and cache it to be used for logging.

Arguments:

    None.

Return Value:

    None.

--*/
{
    LONG error = 0;
    WCHAR buffer[128];
    DWORD bufSize = 128;
    PSTR nodeId = NULL;
    PSTR clusterId = NULL;
    ZeroMemory(reinterpret_cast<LPVOID>(buffer), sizeof(buffer));

    error = RegGetValueW(
                HKEY_LOCAL_MACHINE,
                NODE_PROPERTIES_REG_KEY,
                NODEID_REG_VALUE_NAME,
                RRF_RT_REG_SZ,
                NULL,
                buffer,
                &bufSize);

    if (error == ERROR_SUCCESS) {

        error = ToAnsi(buffer, &nodeId);
        if (error == ERROR_SUCCESS) {

            NodeIdA.assign(nodeId);
            LOGINFO("NodeId: %s", nodeId);
        } else {

            LOGWARNW(L"Failed to duplicate NodeId from buffer");
        }
    } else {

        LOGWARNW(L"Failed to read NodeId from registrey key : %s\\%s", 
                 NODE_PROPERTIES_REG_KEY, 
                 NODEID_REG_VALUE_NAME);
    }

    ZeroMemory(reinterpret_cast<LPVOID>(buffer), sizeof(buffer));
    bufSize = 128;
    error = RegGetValueW(
                HKEY_LOCAL_MACHINE,
                NODE_PROPERTIES_REG_KEY,
                CLUSTER_ID_REG_VALUE_NAME,
                RRF_RT_REG_SZ,
                NULL,
                buffer,
                &bufSize);

    if (error == ERROR_SUCCESS) {
        error = ToAnsi(buffer, &clusterId);
        if (error == ERROR_SUCCESS) {
            ClusterNameA.assign(clusterId);
            LOGINFO("ClusterName: %s", clusterId);
        } else {
            LOGWARNW(L"Failed to duplicate ClusterName from buffer");
        }
    } else {
        LOGWARNW(L"Failed to read ClusterName from registrey key : %s\\%s", 
                 NODE_PROPERTIES_REG_KEY, 
                 CLUSTER_ID_REG_VALUE_NAME);
    }

    ANSI_STR_FREE(nodeId);
    ANSI_STR_FREE(clusterId);
}

void MdmLogger::LogCNIReport(
    wstring ContainerId,
    CNIReport& CNIReport
)
/*++

Routine Description:

    Helper to log CNI report to MDM.

Arguments:

    ContainerId - Container Id.

    CNIReport - The CNI report to be logged.

Return Value:

    None.

--*/

{
    PSTR containerIdAnsi = NULL;
    PSTR nameAnsi = NULL;
    PSTR versionAnsi = NULL;
    PSTR orchestratorNameAnsi = NULL;
    PSTR orchestratorVersionAnsi = NULL;
    PSTR osTypeAnsi = NULL;
    PSTR osVersionAnsi = NULL;
    PSTR kernelVersionAnsi = NULL;
    PSTR errorMessageAnsi = NULL;
    HRESULT hr = S_OK;

    hr = HRESULT_FROM_WIN32(ToAnsi(ContainerId.c_str(), &containerIdAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert ContainerId parameter %s to ANSI for PerContainerNode metric",
            ContainerId.c_str());
        goto Cleanup;
    }

    hr = HRESULT_FROM_WIN32(ToAnsi(CNIReport.Name.c_str(), &nameAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert CNI Name parameter %s to ANSI for PerContainerNode metric",
            CNIReport.Name.c_str());
    }

    hr = HRESULT_FROM_WIN32(ToAnsi(CNIReport.Version.c_str(), &versionAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert CNI Version parameter %s to ANSI for PerContainerNode metric",
            CNIReport.Version.c_str());
    }

    hr = HRESULT_FROM_WIN32(ToAnsi(CNIReport.OrchestratorDetails.OrchestratorName.c_str(), &orchestratorNameAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert Orchestrator Name parameter %s to ANSI for PerContainerNode metric",
            CNIReport.OrchestratorDetails.OrchestratorName.c_str());
    }


    hr = HRESULT_FROM_WIN32(ToAnsi(CNIReport.OrchestratorDetails.OrchestratorVersion.c_str(), &orchestratorVersionAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert Orchestrator Version parameter %s to ANSI for PerContainerNode metric",
            CNIReport.OrchestratorDetails.OrchestratorVersion.c_str());
    }


    hr = HRESULT_FROM_WIN32(ToAnsi(CNIReport.OSDetails.OSType.c_str(), &osTypeAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert guest OS Type parameter %s to ANSI for PerContainerNode metric",
            CNIReport.OSDetails.OSType.c_str());
    }


    hr = HRESULT_FROM_WIN32(ToAnsi(CNIReport.OSDetails.OSVersion.c_str(), &osVersionAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert guest OS Version parameter %s to ANSI for PerContainerNode metric",
            CNIReport.OSDetails.OSVersion.c_str());
    }

    hr = HRESULT_FROM_WIN32(ToAnsi(CNIReport.OSDetails.KernelVersion.c_str(), &kernelVersionAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert guest OS Kernel Version parameter %s to ANSI for PerContainerNode metric",
            CNIReport.OSDetails.KernelVersion.c_str());
    }

    hr = HRESULT_FROM_WIN32(ToAnsi(CNIReport.ErrorMessage.c_str(), &errorMessageAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert CNI ErrorMessage parameter %s to ANSI for PerContainerNode metric",
            CNIReport.ErrorMessage.c_str());
    }

    if (CNIReport.StartFlag != 0)
    {
        hr = LogPerContainerCNIMetricToMDM(containerIdAnsi,
            nameAnsi,
            versionAnsi,
            orchestratorNameAnsi,
            orchestratorVersionAnsi,
            osTypeAnsi,
            osVersionAnsi,
            kernelVersionAnsi,
            MDM_METRIC_CNI_RUNNING_STR,
            1);
        
        if (hr != ERROR_SUCCESS)
        {
            LOGERRW(L"Uploading CNI metrics to mdm failed with errorcode %d", hr);
        }
    }
    if (!CNIReport.ErrorMessage.empty())
    {
        hr = LogPerContainerCNIMetricToMDM(containerIdAnsi,
            nameAnsi,
            versionAnsi,
            orchestratorNameAnsi,
            orchestratorVersionAnsi,
            osTypeAnsi,
            osVersionAnsi,
            kernelVersionAnsi,
            MDM_METRIC_CNI_ERROR_STR,
            1);

        if (hr != ERROR_SUCCESS)
        {
            LOGERRW(L"Uploading CNI Error metrics to mdm failed with errorcode %d", hr);
        }
    }

Cleanup:

    ANSI_STR_FREE(containerIdAnsi);
    ANSI_STR_FREE(nameAnsi);
    ANSI_STR_FREE(versionAnsi);
    ANSI_STR_FREE(orchestratorNameAnsi);
    ANSI_STR_FREE(orchestratorVersionAnsi);
    ANSI_STR_FREE(osTypeAnsi);
    ANSI_STR_FREE(osVersionAnsi);
    ANSI_STR_FREE(kernelVersionAnsi);
    ANSI_STR_FREE(errorMessageAnsi);

    return;
}



void MdmLogger::LogHealthReport(
    HealthReport& HealthReport
    )
/*++

Routine Description:

    Helper to log report to MDM.

Arguments:

    HealthReport - The health report to be logged.

Return Value:

    None.

--*/

{
    PSTR containerIdAnsi = NULL;
    PSTR guestOSAnsi = NULL;
    PSTR guestOSVersionAnsi = NULL;
    HRESULT hr = S_OK;

    hr = HRESULT_FROM_WIN32(ToAnsi(HealthReport.ContainerId.c_str(), &containerIdAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert ContainerId parameter %s to ANSI for PerContainerNode metric",
                HealthReport.ContainerId.c_str());
        goto Cleanup;
    }

    hr = HRESULT_FROM_WIN32(ToAnsi(HealthReport.GuestOSFamily.c_str(), &guestOSAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert guest OS parameter %s to ANSI for PerContainerNode metric",
                HealthReport.GuestOSFamily.c_str());
    }

    hr = HRESULT_FROM_WIN32(ToAnsi(HealthReport.GuestOSVersion.c_str(), &guestOSVersionAnsi));
    if (hr != S_OK) {

        LOGERRW(L"Failed to convert guest OSVersion parameter %s to ANSI for PerContainerNode metric",
                HealthReport.GuestOSVersion.c_str());
    }

    if(HealthReport.DriverInstallAttempted != 0) {

        LogPerContainerMetricToMDM(containerIdAnsi,
                                   guestOSAnsi,
                                   guestOSVersionAnsi,
                                   MDM_METRIC_GUEST_DRIVER_INSTALL_ATTEMPTED_STR,
                                   1);
    }

    if (HealthReport.DriverInstallNotAttempted != 0) {

        LogPerContainerMetricToMDM(containerIdAnsi,
                                   guestOSAnsi,
                                   guestOSVersionAnsi,
                                   MDM_METRIC_GUEST_DRIVER_INSTALL_NOT_ATTEMPTED_STR,
                                   1);
    }

    if (HealthReport.DriverInstallSucceeded != 0) {

        LogPerContainerMetricToMDM(containerIdAnsi,
                                   guestOSAnsi,
                                   guestOSVersionAnsi,
                                   MDM_METRIC_GUEST_DRIVER_INSTALL_SUCCEEDED_STR,
                                   1);
    }

    if (HealthReport.DriverInstallFailed != 0) {

        LogPerContainerMetricToMDM(containerIdAnsi,
                                   guestOSAnsi,
                                   guestOSVersionAnsi,
                                   MDM_METRIC_GUEST_DRIVER_INSTALL_FAILED_STR,
                                   1);
    }

    if (HealthReport.TotalVFDevicesCount != 0) {

        LogPerContainerMetricToMDM(containerIdAnsi,
                                   guestOSAnsi,
                                   guestOSVersionAnsi,
                                   MDM_METRIC_GUEST_TOTAL_VF_STR,
                                   HealthReport.TotalVFDevicesCount);
    }

    if (HealthReport.NonFunctionalVFDevicesCount != 0) {

        LogPerContainerMetricToMDM(containerIdAnsi,
                                   guestOSAnsi,
                                   guestOSVersionAnsi,
                                   MDM_METRIC_GUEST_VF_IN_ERROR_STR,
                                   HealthReport.NonFunctionalVFDevicesCount);
    }

    if (HealthReport.ErrorCode != 0) {

        LogPerContainerMetricToMDM(containerIdAnsi,
                                   guestOSAnsi,
                                   guestOSVersionAnsi,
                                   MDM_METRIC_GUEST_DRIVER_INSTALL_ERROR_CODE_STR,
                                   HealthReport.ErrorCode);
    }

Cleanup:

    ANSI_STR_FREE(containerIdAnsi);
    ANSI_STR_FREE(guestOSAnsi);
    ANSI_STR_FREE(guestOSVersionAnsi);

    return;
}

MdmLogger* MdmLogger::GetInstance()
/*++

Routine Description:

    Helper to get static instance of Mdm logger.

Arguments:

    None.

Return Value:

    Instance of MdmLogger object.

--*/
{
    static MdmLogger* gm_MdmLoggerInstance = new MdmLogger();
    return gm_MdmLoggerInstance;
}
