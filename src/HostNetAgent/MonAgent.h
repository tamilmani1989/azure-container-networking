#pragma once

#include <Windows.h>
#include <Shlobj_core.h>
#include <WinError.h>
#include <string>

#define MONITORING_GCS_ENVIRONMENT      L"MONITORING_GCS_ENVIRONMENT"
#define MONITORING_GCS_ACCOUNT          L"MONITORING_GCS_ACCOUNT"
#define MONITORING_GCS_NAMESPACE        L"MONITORING_GCS_NAMESPACE"
#define MONITORING_GCS_REGION           L"MONITORING_GCS_REGION"
#define MONITORING_GCS_AUTH_ID          L"MONITORING_GCS_AUTH_ID"
#define MONITORING_GCS_AUTH_ID_TYPE     L"MONITORING_GCS_AUTH_ID_TYPE"
#define MONITORING_DATA_DIRECTORY       L"MONITORING_DATA_DIRECTORY"
#define MONITORING_TENANT               L"MONITORING_TENANT"
#define NODE_ID                         L"NODE_ID"
#define MONITORING_ROLE                 L"MONITORING_ROLE"
#define MONITORING_ROLE_INSTANCE        L"MONITORING_ROLE_INSTANCE"
#define MONITORING_CONFIG_VERSION       L"MONITORING_CONFIG_VERSION"

#define MONITORING_ENABLED_PATH         L"//NetAgentConfiguration/MonitoringConfig/EnableMA"
#define MONITORING_GCS_ENVIRONMENT_PATH L"//NetAgentConfiguration/MonitoringConfig/GcsEnvironment"
#define MONITORING_GCS_ACCOUNT_PATH     L"//NetAgentConfiguration/MonitoringConfig/GcsAccount"
#define MONITORING_GCS_NAMESPACE_PATH   L"//NetAgentConfiguration/MonitoringConfig/GcsNamespace"
#define MONITORING_GCS_REGION_PATH      L"//NetAgentConfiguration/MonitoringConfig/GcsRegion"
#define MONITORING_GCS_AUTH_ID_PATH     L"//NetAgentConfiguration/MonitoringConfig/GcsDsmsUrl"
#define MONITORING_GCS_AUTH_ID_TYPE_VAL L"AuthDsmsUrl"
#define MONITORING_CONFIG_VERSION_VAL   L"999.9"

#define MONITORING_AGENT_STORAGE_PATH   L"C:\\NetAgent\\MonitoringStorage\\"

#define NODE_PROPERTIES_REG_KEY         L"SOFTWARE\\AzureHL\\NodeProperties"

#define MA_LAUNCHER_EXE                 L"MonAgentLauncher.exe"
#define MA_LAUNCHER_USEENV_FLAG         L"-useenv"
#define MA_LAUNCHER_PARENT_FLAG         L"-parent"
#define MA_RELATIVE_PATH                L"MonAgent\\"

const std::wstring role = L"NetAgent";
const std::wstring roleInstance = L"NetAgent";
const ULONG MAX_MA_CMDLINE_LENGTH = 512;

class MonAgent
{
public:
    static MonAgent* GetInstance();
    HRESULT LaunchMA(LPCWSTR SettingsFileName);

private:
    static HANDLE gm_MonitoringJob;
    LPWSTR nodeID;
    LPWSTR clusterName;

    MonAgent() {}
    HRESULT launchMA(LPCWSTR SettingsFileName);
    void getNodeIDFromRegistry();
    void getClusterNameFromRegistry();
};