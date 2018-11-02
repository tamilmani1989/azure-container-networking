#include "MonAgent.h"
#include "XmlParserHelper.h"

HANDLE MonAgent::gm_MonitoringJob = NULL;

MonAgent* MonAgent::GetInstance()
/*++

Routine Description:

    Helper to get static instance of MA Launcher.

Arguments:

    None.

Return Value:

    Instance of MALauncher object.

--*/
{
    static MonAgent* gm_MonAgentInstance = new MonAgent();
    return gm_MonAgentInstance;
}

HRESULT MonAgent::LaunchMA(LPCWSTR SettingsFileName)
/*++

Routine Description:

    Check if our instance of MonAgent running, if not, execute our MA launcher 

Arguments:

    Config file (NetAgentHostPlugin.PluginSpecific.xml)

Return Value:

    S_OK on success.

--*/
{
    return ( gm_MonitoringJob ) ? 
                            S_OK :
                            launchMA(SettingsFileName);
}

HRESULT MonAgent::launchMA(LPCWSTR SettingsFileName)
/*++

Routine Description:

    Configures environment variables based on NetAgentHostPlugin.PluginSpecific.xml
    and executes MonAgentLauncher -useenv

Arguments:

    Config file (NetAgentHostPlugin.PluginSpecific.xml)

Return Value:

    S_OK on success.

--*/
{
    bool comInitialized{};
    std::wstring gcsEnvironment;
    std::wstring gcsAccount;
    std::wstring gcsNamespace;
    std::wstring gcsRegion;
    std::wstring gcsAuthID;
    std::wstring netagentPath;
    std::wstring maPath;
    std::wstring maDataPath;
    std::wstring maEnabled;
    XmlParserHelper xmlHelper;
    JOBOBJECT_EXTENDED_LIMIT_INFORMATION extendedInfo = { 0 };
    STARTUPINFOW startupInfo = { 0 };
    PROCESS_INFORMATION processInformation = { 0 };
    PWCHAR applicationName = NULL;
    PWCHAR commandLine = NULL;
    PWCHAR fileName = NULL;
    HANDLE job = NULL;
    HRESULT hr = S_OK;
    enum { MaxEnvVarLen = 32767 };
    
    // Allocate string arrays to use for building command and application information
    applicationName = (PWCHAR)calloc(1, MAX_MA_CMDLINE_LENGTH*sizeof(WCHAR));
    commandLine = (PWCHAR)calloc(1, MAX_MA_CMDLINE_LENGTH*sizeof(WCHAR));
    fileName = (PWCHAR)calloc(1, MAX_MA_CMDLINE_LENGTH*sizeof(WCHAR));
    if ((applicationName == NULL) ||
        (commandLine == NULL) ||
        (fileName == NULL))
    {
        hr = E_OUTOFMEMORY;
        LOGERRW(L"Failed to allocate string arrays to build MonAgent command info.");
        goto CLEANUP;
    }

    // Initialize/setup startupInfo and processInformation
    ZeroMemory(&startupInfo, sizeof(startupInfo));
    startupInfo.cb = sizeof(startupInfo);
    GetStartupInfoW(&startupInfo);
    ZeroMemory(&processInformation, sizeof(processInformation));

    // We build the application name with full path information
    if (!GetModuleFileNameW(NULL,
                            fileName,
                            MAX_MA_CMDLINE_LENGTH))
    {
        hr = HRESULT_FROM_WIN32(GetLastError());
        LOGERRW(L"Failed to get module file name during MA launch %X", hr);
        goto CLEANUP;
    }

    // Remove the executable name part of fileName to get full path, and then
    // add the relative path constant.
    netagentPath.assign(fileName);
    netagentPath = netagentPath.substr(0, ( (ULONG)netagentPath.find_last_of(L"\\", netagentPath.length()) + 1 ));
    maPath = netagentPath;
    maPath.append(MA_RELATIVE_PATH);
    if ((swprintf_s(
        applicationName,
        MAX_MA_CMDLINE_LENGTH,
        L"%s%s",
        maPath.c_str(),
        MA_LAUNCHER_EXE)) == -1)
    {
        LOGERRW(L"Error creating application name for MA launch");
        hr = E_UNEXPECTED;
        goto CLEANUP;
    }

    // Try to initialize and parse the plugin specific xml for
    // the configuration values required to launch MonAgent.
    hr = CoInitialize(NULL);
    if (hr != S_OK) 
    {
        LOGERRW(L"ComInitialization failed with error [%X].", hr);
        goto CLEANUP;
    }
    comInitialized = true;

    hr = xmlHelper.Initialize(SettingsFileName);
    if (hr != S_OK)
    {
        LOGERRW(L"XML initialization failed when loading file {%s} error [%X].",
                SettingsFileName, hr);
        goto CLEANUP;
    }

    // Return S_OK if MA is disabled.
    hr = xmlHelper.QueryNodeValue(MONITORING_ENABLED_PATH, maEnabled);
    if (hr != S_OK)
    {
        LOGERRW(L"Query node value for path {%s} error [%X].", MONITORING_ENABLED_PATH, hr);
        goto CLEANUP;
    }
    
    transform(maEnabled.begin(), maEnabled.end(), maEnabled.begin(), ::towlower);
    if (maEnabled != L"true")
    {
        goto CLEANUP;
    }

    hr = xmlHelper.QueryNodeValue(MONITORING_GCS_ENVIRONMENT_PATH, gcsEnvironment);
    if (hr != S_OK)
    {
        LOGERRW(L"Query node value for path {%s} error [%X].", MONITORING_GCS_ENVIRONMENT_PATH, hr);
        goto CLEANUP;
    }
    
    hr = xmlHelper.QueryNodeValue(MONITORING_GCS_ACCOUNT_PATH, gcsAccount);
    if (hr != S_OK)
    {
        LOGERRW(L"Query node value for path {%s} error [%X].", MONITORING_GCS_ACCOUNT_PATH, hr);
        goto CLEANUP;
    }

    hr = xmlHelper.QueryNodeValue(MONITORING_GCS_NAMESPACE_PATH, gcsNamespace);
    if (hr != S_OK)
    {
        LOGERRW(L"Query node value for path {%s} error [%X].", MONITORING_GCS_NAMESPACE_PATH, hr);
        goto CLEANUP;
    }

    hr = xmlHelper.QueryNodeValue(MONITORING_GCS_REGION_PATH, gcsRegion);
    if (hr != S_OK)
    {
        LOGERRW(L"Query node value for path {%s} error [%X].", MONITORING_GCS_REGION_PATH, hr);
        goto CLEANUP;
    }

    hr = xmlHelper.QueryNodeValue(MONITORING_GCS_AUTH_ID_PATH, gcsAuthID);
    if (hr != S_OK)
    {
        LOGERRW(L"Query node value for path {%s} error [%X].", MONITORING_GCS_AUTH_ID_PATH, hr);
        goto CLEANUP;
    }

    // Try to setup monitoring agent storage.
    int result = SHCreateDirectoryExW(NULL, MONITORING_AGENT_STORAGE_PATH, NULL);
    if (!(result == ERROR_FILE_EXISTS || result == ERROR_ALREADY_EXISTS || result == ERROR_SUCCESS))
    {
        hr = CO_E_FAILEDTOGETWINDIR; // generic error so that we don't return S_OK
        LOGERRW(L"Failed to create a directory or persistent storage for monitoring agent. {%s} error [%X]", MONITORING_AGENT_STORAGE_PATH, hr);
        goto CLEANUP;
    }

    // Set environment variables for MA
    maDataPath = MONITORING_AGENT_STORAGE_PATH;
    if (!(SetEnvironmentVariableW(MONITORING_DATA_DIRECTORY, maDataPath.c_str()) &&
            SetEnvironmentVariableW(MONITORING_GCS_ENVIRONMENT, gcsEnvironment.c_str()) &&
            SetEnvironmentVariableW(MONITORING_GCS_ACCOUNT, gcsAccount.c_str()) &&
            SetEnvironmentVariableW(MONITORING_GCS_NAMESPACE, gcsNamespace.c_str()) &&
            SetEnvironmentVariableW(MONITORING_GCS_REGION, gcsRegion.c_str()) &&
            SetEnvironmentVariableW(MONITORING_GCS_AUTH_ID, gcsAuthID.c_str()) &&
            SetEnvironmentVariableW(MONITORING_GCS_AUTH_ID_TYPE, MONITORING_GCS_AUTH_ID_TYPE_VAL) &&
            SetEnvironmentVariableW(MONITORING_ROLE, role.c_str()) &&
            SetEnvironmentVariableW(MONITORING_ROLE_INSTANCE, roleInstance.c_str()) &&
            SetEnvironmentVariableW(MONITORING_CONFIG_VERSION, MONITORING_CONFIG_VERSION_VAL)))
    {
        hr = HRESULT_FROM_WIN32(GetLastError());
        LOGERRW(L"Failed to set environment variables for MA due to error %X", hr);
        goto CLEANUP;
    }

    LOGINFOW(L"%s -> %s", MONITORING_DATA_DIRECTORY, maDataPath.c_str());
    LOGINFOW(L"%s -> %s", MONITORING_GCS_ENVIRONMENT, gcsEnvironment.c_str());
    LOGINFOW(L"%s -> %s", MONITORING_GCS_ACCOUNT, gcsAccount.c_str());
    LOGINFOW(L"%s -> %s", MONITORING_GCS_NAMESPACE, gcsNamespace.c_str());
    LOGINFOW(L"%s -> %s", MONITORING_GCS_REGION, gcsRegion.c_str());
    LOGINFOW(L"%s -> %s", MONITORING_GCS_AUTH_ID, gcsAuthID.c_str());
    LOGINFOW(L"%s -> %s", MONITORING_GCS_AUTH_ID_TYPE, MONITORING_GCS_AUTH_ID_TYPE_VAL);

    getNodeIDFromRegistry();
    getClusterNameFromRegistry();

    // In the command line, we include the EXE name, because otherwise
    // the child process will not see argv[0] as the binary. This breaks
    // MA launcher because it will only start looking for arguments at argv[1]
    if ((swprintf_s(
        commandLine,
        MAX_MA_CMDLINE_LENGTH,
        L"%s %s %s %d",
        MA_LAUNCHER_EXE,
        MA_LAUNCHER_USEENV_FLAG,
        MA_LAUNCHER_PARENT_FLAG,
        GetCurrentProcessId())) == -1)
    {
        LOGERRW(L"Error creating command line for MA launch");
        hr = E_UNEXPECTED;
        goto CLEANUP;
    }

    LOGINFOW(L"Executing MA application %s with command line %s",
        applicationName,
        commandLine);

    // Prepare job object &
    // execute process using CreateProcess in a Job that will terminate with parent (current process).
    // This will make sure that when NMAgent gets killed, the MA launcher is also killed.
    job = CreateJobObject(NULL, NULL);
    if (job == NULL)
    {
        hr = HRESULT_FROM_WIN32(GetLastError());
        LOGERRW(L"Failed to create job object with error %X.", hr)
        goto CLEANUP;
    }

    // Query default job limits and add additional limits.
    ZeroMemory(&extendedInfo, sizeof(extendedInfo));
    if (!QueryInformationJobObject(job,
                                JobObjectExtendedLimitInformation,
                                (PVOID)&extendedInfo,
                                sizeof(extendedInfo),
                                NULL))
    {
        hr = HRESULT_FROM_WIN32(GetLastError());
        LOGERRW(L"Failed to query extended limit information from job object with error %X", hr);
        goto CLEANUP;
    }

    // Configures the job to kill processes when the last handle to the job object is closed.
    extendedInfo.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
    if (!SetInformationJobObject(job,
                                JobObjectExtendedLimitInformation,
                                (PVOID)&extendedInfo,
                                sizeof(extendedInfo)))
    {
        hr = HRESULT_FROM_WIN32(GetLastError());
        LOGERRW(L"Set JobObjectExtendedLimitInformation failed on job with error %X", hr);
        goto CLEANUP;
    }

    // Create process.
    if (!(CreateProcessW(applicationName,
                        commandLine,
                        NULL,
                        NULL,
                        FALSE,
                        CREATE_UNICODE_ENVIRONMENT | DETACHED_PROCESS,
                        NULL,
                        maPath.c_str(),
                        &startupInfo,
                        &processInformation)))
    {
        hr = HRESULT_FROM_WIN32(GetLastError());
        LOGERRW(L"Failed to start MA launcher process with error %X", hr);
        goto CLEANUP;
    }

    // Assign process to job
    if (!AssignProcessToJobObject(job, processInformation.hProcess))
    {
        hr = HRESULT_FROM_WIN32(GetLastError());
        LOGERRW(L"AssignProcessToJobObject failed on job with error %X", hr);
        goto CLEANUP;
    }

    // Save the job object handle for restart.
    // The job handle *MUST* remain valid and not released until process exits.
    // Once process exits or get terminated, the OS will close the handle,
    // also killing this logger child processes.
    gm_MonitoringJob = job;
    // Do not close the handle
    job = NULL;

CLEANUP:
    // Free up any allocated strings
    if (job)
    {
        CloseHandle(job);
        job = NULL;
    }

    if (applicationName)
    {
        free(applicationName);
    }

    if (commandLine)
    {
        free(commandLine);
    }

    if (fileName)
    {
        free(fileName);
    }

    if (comInitialized)
    {
        CoUninitialize();
    }

    // Clear environment variables we set
    SetEnvironmentVariableW(MONITORING_DATA_DIRECTORY, NULL);
    SetEnvironmentVariableW(MONITORING_GCS_ENVIRONMENT, NULL);
    SetEnvironmentVariableW(MONITORING_GCS_ACCOUNT, NULL);
    SetEnvironmentVariableW(MONITORING_GCS_NAMESPACE, NULL);
    SetEnvironmentVariableW(MONITORING_GCS_REGION, NULL);
    SetEnvironmentVariableW(MONITORING_GCS_AUTH_ID, NULL);
    SetEnvironmentVariableW(MONITORING_GCS_AUTH_ID_TYPE, NULL);
    SetEnvironmentVariableW(MONITORING_TENANT, NULL);
    SetEnvironmentVariableW(MONITORING_ROLE, NULL);
    SetEnvironmentVariableW(MONITORING_ROLE_INSTANCE, NULL);
    SetEnvironmentVariableW(NODE_ID, NULL);

    return hr;
}

void MonAgent::getNodeIDFromRegistry()
/*++

Routine Description:

    Get NodeID from registry sets it for this instance.

Return Value:

    void

--*/
{
    DWORD bufSize = 128;
    WCHAR buffer[128] = {0};
    LPCWSTR nodeIdRegValueName = L"NodeId";

    if (!nodeID)
    {
        if ((RegGetValueW(
                HKEY_LOCAL_MACHINE,
                NODE_PROPERTIES_REG_KEY,
                nodeIdRegValueName,
                RRF_RT_REG_SZ,
                NULL,
                buffer,
                &bufSize)) == ERROR_SUCCESS)
        {
            nodeID = WcsDup(buffer);
            if (nodeID)
            {
                SetEnvironmentVariableW(NODE_ID, nodeID);
            }
            else
            {
                LOGWARNW(L"Failed to duplicate NodeId from buffer");
            }
        }
        else 
        {
            LOGWARNW(L"Failed to read NodeId from registrey key : %s\\%s", 
                    NODE_PROPERTIES_REG_KEY,
                    nodeIdRegValueName);
        }
    }
}

void MonAgent::getClusterNameFromRegistry()
/*++

Routine Description:

    Get Cluster Name from registry sets it for this instance.

Return Value:

    void

--*/
{
    DWORD bufSize = 128;
    WCHAR buffer[128] = {0};
    LPCWSTR clusterIdRegValueName = L"ClusterName";

    if (!clusterName)
    {
        if ((RegGetValueW(
                HKEY_LOCAL_MACHINE,
                NODE_PROPERTIES_REG_KEY,
                clusterIdRegValueName,
                RRF_RT_REG_SZ,
                NULL,
                buffer,
                &bufSize)) == ERROR_SUCCESS)
        {
            clusterName = WcsDup(buffer);
            if (clusterName)
            {
                SetEnvironmentVariableW(MONITORING_TENANT, clusterName);
            }
            else
            {
                LOGWARNW(L"Failed to duplicate ClusterName from buffer");
            }
        }
        else 
        {
            LOGWARNW(L"Failed to read ClusterName from registrey key : %s\\%s", 
                    NODE_PROPERTIES_REG_KEY,
                    clusterIdRegValueName);
        }
    }
}
