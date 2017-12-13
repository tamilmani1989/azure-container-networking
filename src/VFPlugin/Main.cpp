/*++

Copyright (c) Microsoft Corporation

Module Name:

Main.cpp

Abstract:

This file is the entry point for installing mellanox driver and report status to NMAgent and HostAgent.

Author:

Tamilmani Manoharan (tamanoha) 10-October-2017

--*/


#include "VFPlugin.h"
#include "..\Logging\Logging.h"
#include <memory>
#include "VFPluginUtilities.h"
#include "VFHealthReport.h"
#include "NMAgentReport.h"
#include <cstdlib>

#pragma comment(lib, "version.lib")

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

DWORD cleanupPnPAndInstallVFPP(
    VFHealthReport& healthReport
)
/**
Routine description:

    This function calls rundll and netcfg methods and report status to hostnetagent.

Arguments:
    healthReport - object that contains information that need to be sent to hostnetagent.

Return Value:
    returns 0 on success else errorcode returned by rundll/netcfg commands.

**/
{
    healthReport.report.DebugMessage.assign(L"Going to run RunDll command");
    healthReport.SendHealthStatusToHostAgent();
    healthReport.ResetHealthObject();

    LOGINFOW(L"Going to run RunDll command");

    DWORD ret = VFPluginUtilities::RunDllCmd();
    if (ret != 0)
    {
        std::wstring msg = L"RunDll command failed with errorcode " + std::to_wstring(ret);
        LOGINFOW(msg.c_str());
        healthReport.report.DebugMessage.assign(msg);
        healthReport.report.DriverInstallPreSetupFailed = true;
        healthReport.SendHealthStatusToHostAgent();
        healthReport.ResetHealthObject();
        goto Cleanup;
    }
    else
    {
        LOGINFOW(L"Rundll cmd succeeded");
        healthReport.report.DebugMessage.assign(L"RunDll command Succeeded. Going to run netcfg command");
        healthReport.SendHealthStatusToHostAgent();
        healthReport.ResetHealthObject();
    }

    LOGINFOW(L"Going to run NetCfg command");

    ret = VFPluginUtilities::RunNetCfgCmd();
    if (ret != 0)
    {
        std::wstring msg = L"NetCfg command failed with errorcode " + std::to_wstring(ret);
        LOGINFOW(msg.c_str());
        healthReport.report.DebugMessage.assign(msg);
        healthReport.report.DriverInstallPreSetupFailed = true;
        healthReport.SendHealthStatusToHostAgent();
        healthReport.ResetHealthObject();
        goto Cleanup;
    }
    else
    {
        LOGINFOW(L"Rundll and Netcfg cmd succeeded");
        healthReport.report.DebugMessage.assign(L"NetCfg command Succeeded");
        healthReport.report.DriverInstallPreSetupSucceeded = true;
        healthReport.SendHealthStatusToHostAgent();
        healthReport.ResetHealthObject();
    }

Cleanup:
    return ret;
}

void cleanUpAndInstallMlxDriver(
    VFHealthReport& healthReport
)

/**
Routine description:
    This function cleans up zombie pnp devices, installs vfpp, and runs mellanox driver installer if needed.

Arguments:
    healthReport - object that contains information that need to be sent to hostnetagent.
	versionInfo - Current version of windows.

Return Value:
    N/A.

**/
{
    VFPlugin vfplugin;
    DWORD status;
	int initializationStatus;
	
    if (IsWindows8Point1OrGreater() && !IsWindows10OrGreater())
	{
        initializationStatus = cleanupPnPAndInstallVFPP(healthReport);
    }
    else
    {        
		LOGINFO("OS Version check IsWindows8Point1OrGreater and !IsWindows10OrGreater does not allow us to perform cleanup.")
        healthReport.report.DebugMessage.assign(L"Windows version is not server 2012 R2. so not calling cleanupPnPAndInstallVFPP");
        healthReport.report.DriverInstallPreSetupNotAttempted = true;
        healthReport.SendHealthStatusToHostAgent();
        healthReport.ResetHealthObject();
        initializationStatus = 0;
    }
	    
    // Check if driver installation is needed by verifying OS version    
    BOOL isInstallRequired = vfplugin.IsMlxVFDriverInstallNeeded();  
	
    do 
    {   
        if(isInstallRequired && initializationStatus == 0)
        {
            wchar_t currentPath[MAX_PATH];
            status = ERROR_SUCCESS;
        
            LOGINFO("Going to install Mellanox Driver");
            
            if (GetCurrentDirectory(MAX_PATH, currentPath) == 0) 
            {
                status = GetLastError();
                LOGINFOW(L"VFPlugin: GetCurrentDirectory failed with error %d\n", status);
                healthReport.report.DebugMessage.assign(L"GetCurrentDirectory failed before driver install attempt");
                healthReport.SendHealthStatusToHostAgent();
                healthReport.ResetHealthObject();
                break;
            }

            healthReport.report.DriverInstallAttempted=true;            
            healthReport.SendHealthStatusToHostAgent();
            healthReport.ResetHealthObject();

            LOGINFOW(L"Current directory path:%s", currentPath);
            status = vfplugin.InstallMlxVFDriver(currentPath);
            healthReport.report.ErrorCode =  status;
            
            if (status != ERROR_SUCCESS &&  status != REBOOT_REQD) 
            {
                LOGINFOW(L"VFPlugin: VF Driver Installation Failed from %s\n", currentPath);                
                healthReport.report.DriverInstallFailed = true;           
                healthReport.SendHealthStatusToHostAgent();
                healthReport.ResetHealthObject();
                break;            
            } 
            
            healthReport.report.DriverInstallSucceeded=true;
            healthReport.SendHealthStatusToHostAgent();
            healthReport.ResetHealthObject();
            
            LOGINFOW(L"VFPlugin: VF Driver installation succeeded from %s\n", currentPath); 
            VFPluginUtilities::SaveInstalledState();
        }
        else
        {
            if(initializationStatus != 0)
            {
                healthReport.report.DebugMessage.assign(L"Driver installation not attempted because pnp cleanup failed.");
            }
            healthReport.report.DriverInstallNotAttempted=true;
            healthReport.SendHealthStatusToHostAgent();
            healthReport.ResetHealthObject();
        }
    }while(false);
}

void wmain(int argc, wchar_t *argv[])
{
    VFPlugin vfplugin;
    HRESULT hr = S_OK;
    bool unInstallFlag = false;
    bool comInitialized = true;
    wchar_t *logFolder = NULL;
    BSTR statusInfo = NULL; 
    int totalDevices=0;
    int timeout = 3600000;
    VFHealthReport healthReport;
    NMAgentReport nmAgentReport;
    bool cleanup = false;
	wchar_t *cleanupStr;
    
    if(argc == 1)
    {
        wprintf(L"usage: VFPlugin.exe -log <logfolderpath> [-timeout <timeoutinseconds>] [-pnpcleanup <true/false>] ");
        return;
    }
	
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
        else if (_wcsicmp(L"-timeout", argv[i]) == 0)
        {
            if (i + 1 < argc)
            {
                timeout = _wtoi(argv[++i]);
                if (timeout == 0)
                {
                    wprintf(L"Timeout cannot be 0");
                    timeout = 3600;
                }
                timeout = timeout * 1000;
            }
        }
        else if (_wcsicmp(L"-installmellanox", argv[i]) == 0)
        {
            if (i + 1 < argc)
            {
                cleanupStr = argv[++i];				
                if (wcscmp (cleanupStr, L"true") == 0)
                {
                    wprintf(L"pnp Cleanup and driver installation requested. Will do that if needed.");
                    cleanup = true;
                }
                else
                {
                    wprintf(L"installmellanox is not true. Will not perform pnp cleanup and/or install mellanox driver.");
                }
            }
        }
    }
	
	VFPluginUtilities::SetupLogging(logFolder, LOG_FILE_PREFIX);

	LOGINFOW(L"NMAgent url %s", nmAgentReport.nmAgentUrl);
    LOGINFOW(L"HostNetAgent url %s", healthReport.healthUrl);
    LOGINFO("Status send interval in milliseconds:%d", timeout);

	healthReport.ResetHealthObject();

	healthReport.report.GuestOSVersion.assign(L"");
    healthReport.report.GuestOSFamily.assign(L"Windows");
    healthReport.report.DebugMessage.assign(L"VFPlugin Started");
    healthReport.SendHealthStatusToHostAgent();
    healthReport.ResetHealthObject();
	
	if (unInstallFlag)
    {
        LOGINFO("VFDriver Remove called");
        healthReport.report.DebugMessage.assign(L"VFPlugin Remove Called");
        healthReport.SendHealthStatusToHostAgent();
        healthReport.ResetHealthObject();
        return;
    }
	
	hr = CoInitialize(NULL);
    if (hr != S_OK)
    {
        LOGERR("CoInitialize failed with errorcode %x", hr);
        std::wstring msg = L"CoInitialize failed with errorcode " + to_wstring(hr);
        healthReport.report.DebugMessage.assign(msg);
        healthReport.SendHealthStatusToHostAgent();
        healthReport.ResetHealthObject();
        comInitialized = false;
    }

	wchar_t versionInfo[MAX_PATH];
	if (VFPluginUtilities::GetWindowsVersion(versionInfo, sizeof(versionInfo)) != 0)
	{
		LOGERRW(L"Windows version is %s. Not calling cleanupPnPAndInstallVFPP", versionInfo);
		healthReport.report.DebugMessage.assign(L"Getting windows version failed");
		healthReport.SendHealthStatusToHostAgent();
		healthReport.ResetHealthObject();
	}
	else
	{
		LOGINFOW(L"Windows version is %s.", versionInfo);
		healthReport.report.GuestOSVersion.assign(versionInfo);
	}

    if (cleanup == true)
    {
        LOGINFO("Going to perform pnp cleanup.")
        cleanUpAndInstallMlxDriver(healthReport);
    }
    else
    {
        LOGINFOW(L"Going to skip pnp cleanup");
		healthReport.report.DebugMessage.assign(L"Not calling cleanUpAndInstallMlxDriver. Was not explicitly requested.");
		healthReport.report.DriverInstallPreSetupNotAttempted = true;
		healthReport.report.DriverInstallNotAttempted = true;
		healthReport.SendHealthStatusToHostAgent();
		healthReport.ResetHealthObject();
    }
	
    /*
      Report driver installation status to NMAgent Periodically
    */	
    while(true)
    {
    
        nmAgentReport.NumNonFunctioningDevices = vfplugin.GetNumNonFunctioningDevices(MLX_DEVICE_ID, &totalDevices);
    
        if (IsWindows8Point1OrGreater() && comInitialized)
        {
            nmAgentReport.IsInstalled = vfplugin.GetMlxVFDriverDetails(nmAgentReport.DriverVersion, sizeof(nmAgentReport.DriverVersion));

            if (!nmAgentReport.IsInstalled)
            {
                LOGWARNW(L"VFDriver is not installed");
            }

            LOGINFOW(L"OS version info %s nonfunctioning devs %d isinstalled %d totaldevs %d nonfunctioning %d",
                healthReport.report.GuestOSVersion.c_str(), nmAgentReport.NumNonFunctioningDevices, 
                nmAgentReport.IsInstalled, totalDevices, nmAgentReport.NumNonFunctioningDevices);

            hr = nmAgentReport.RetrieveStatusInfoAsXMLStr(&statusInfo);
            if (hr != S_OK)
            {
                LOGERRW(L"Retrieving status info failed %x", hr);
                std::wstring msg = L"RetrieveStatusInfoAsXMLStr function failed with errorcode " + to_wstring(hr);
                healthReport.report.DebugMessage.assign(msg);
                healthReport.SendHealthStatusToHostAgent();
                healthReport.ResetHealthObject();
            }
            else
            {
                LOGINFOW(L"Going to send status to NMAgent - %s", nmAgentReport.nmAgentUrl);
                nmAgentReport.SendStatusToNMAgent(statusInfo);
            }
        }

        healthReport.report.NonFunctionalVFDevicesCount = nmAgentReport.NumNonFunctioningDevices;
        healthReport.report.TotalVFDevicesCount = totalDevices;
        healthReport.SendHealthStatusToHostAgent();
        healthReport.ResetHealthObject();

        if (statusInfo)
        {
            SysFreeString(statusInfo);
            statusInfo = NULL;
        }
    
        LOGINFO("Sleep for %d milliseconds.", timeout);
        Sleep(timeout);
    }
}
