#include "VFPlugin.h"
#include "..\Logging\Logging.h"
#include <memory>
#include "VFPluginUtilities.h"
#include <cstdlib>

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

void wmain(int argc, wchar_t *argv[])
{
    VFPlugin vfplugin;
    DWORD status = ERROR_SUCCESS;
    HRESULT hr = S_OK;
    bool unInstallFlag = false;
    BOOL isInstallRequired = FALSE;
    BOOL isInstalled = FALSE;
    wchar_t *logFolder = NULL;
    wchar_t *statusInfo = NULL;
    wchar_t url[MAX_PATH];
    wchar_t installedVersion[MAX_PATH]; 
    wchar_t versionInfo[MAX_PATH];
    wchar_t winDir[MAX_PATH];
    wchar_t findName[MAX_PATH];
    char netSvcPath[MAX_PATH];
    char netcfg_cmd[1024];
    int numNonFunctioningDevices = 0;
    int sysret = 0;
    size_t retVal;
 
    for (int i = 1; i < argc; i++)
	{
        if(_wcsicmp(L"-remove", argv[i]) == 0)
        {
            unInstallFlag = true;
        }
        else if(_wcsicmp(L"-log", argv[i]) == 0)
        {
            if (i + 1 < argc)
            {
                logFolder = argv[++i];
                if (logFolder == NULL)
                {
                    wprintf(L"LogFolder is null");
                }
            }
        }
    }
    
    VFPluginUtilities::SetupLogging(logFolder, LOG_FILE_PREFIX);

    if(unInstallFlag)
    {
        LOGINFO("VFDriver Remove called");
        return;
    }
    
    if (!GetWindowsDirectory(winDir, ARRAYSIZE(winDir))) {
        LOGINFOW(L"GetWindowsDirectory failed\n");       
    }

    LOGINFOW(L"GetWindowsDirectory %s\n", winDir);

    PathCombine(findName, winDir, NETSVC_INF_PATH);
    LOGINFOW(L"NetSvc Inf path %s\n", findName);
    
    if(wcstombs_s(&retVal, netSvcPath, sizeof(netSvcPath), findName, wcslen(findName) + 1) != 0)
    {
        LOGERRW(L"wcstombs_s failed");
    }

    LOGINFO("NetSvc Inf path after conversion%s\n", netSvcPath);

    if(sprintf_s(netcfg_cmd, sizeof(netcfg_cmd), "%s %s", NETCFG_COMMAND, netSvcPath) < 0)
    {
        LOGERRW(L"sprintf_s failed for rundllcmd");
    }

    sysret = system (RUNDLL_COMMAND);
    LOGINFO("System call return for rundll command %s ret value %d", RUNDLL_COMMAND, sysret);
    
    sysret = system (netcfg_cmd);
    LOGINFO("System call return for rundll command %s ret value %d", netcfg_cmd, sysret);
    

    isInstallRequired = vfplugin.IsMlxVFDriverInstallNeeded();    
    
    do {   
        if(isInstallRequired)
        {
            wchar_t currentPath[MAX_PATH];
            status = ERROR_SUCCESS;
        
            LOGINFO("Going to install Mellanox Driver");
            
            if (GetCurrentDirectory(MAX_PATH, currentPath) == 0) {
                status = GetLastError();
                LOGINFOW(L"VFPlugin: GetCurrentDirectory failed with error %d\n", status);
                break;
            }

            LOGINFOW(L"Current directory path:%s", currentPath);
            status = vfplugin.InstallMlxVFDriver(currentPath);
            
            if (status != ERROR_SUCCESS &&  status != REBOOT_REQD) {
                LOGINFOW(L"VFPlugin: VF Driver Installation Failed from %s\n", currentPath);
                break;            
            } 
    
            LOGINFOW(L"VFPlugin: VF Driver installation succeeded from %s\n", currentPath); 
            VFPluginUtilities::SaveInstalledState();
        }
    }while(0);


    if(swprintf_s(versionInfo, sizeof(versionInfo), L"Windows") < 0)
    {
        LOGERRW(L"swprintf_s rrror while constructing guestos");
    }
    
    while(true)
    {
    
        VFPluginUtilities::GetHostNetAgentURL(url, sizeof(url));

        isInstalled = vfplugin.GetMlxVFDriverDetails(installedVersion, sizeof(installedVersion));

        if(!isInstalled)
        {
            LOGWARNW(L"VFDriver is not installed");
        }

        numNonFunctioningDevices = vfplugin.GetNumNonFunctioningDevices(MLX_DEVICE_ID);
        
        LOGINFOW(L"os version info %s nonfunctioning devs %d isinstalled %d", versionInfo, numNonFunctioningDevices, isInstalled);
        hr = VFPluginUtilities::RetrieveStatusInfoAsXMLStr(isInstalled, installedVersion, versionInfo, numNonFunctioningDevices, &statusInfo);

        if(hr != S_OK)
        {
            LOGERRW(L"Retrieving status info failed. Isinstalled %d version %s nonfnstatus %d osinfo %s", 
                    isInstalled, installedVersion, numNonFunctioningDevices, versionInfo);
            SysFreeString(statusInfo);
            continue;
        }

        LOGINFOW(L"Going to send status to wireserver %s", url);
        VFPluginUtilities::SendStatusToWireServer(url, statusInfo);
        
        SysFreeString(statusInfo);
        statusInfo = NULL;
        Sleep(3600000);
    }
}