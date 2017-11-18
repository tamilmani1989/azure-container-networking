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

DWORD initialize(
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

void wmain(int argc, wchar_t *argv[])
{
    VFPlugin vfplugin;
    DWORD status = ERROR_SUCCESS;
    HRESULT hr = S_OK;
    bool unInstallFlag = false;
	bool comInitialized = true;
    BOOL isInstallRequired = FALSE;
    wchar_t *logFolder = NULL;
    BSTR statusInfo = NULL; 
    wchar_t versionInfo[MAX_PATH];
    int totalDevices=0;
	int timeout = 3600000;
	int initializationStatus = 0;
    VFHealthReport healthReport;
	NMAgentReport nmAgentReport;

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


	if(VFPluginUtilities::GetWindowsVersion(versionInfo, sizeof(versionInfo)) != 0)
	{
		healthReport.report.DebugMessage.assign(L"Getting windows version failed");
		healthReport.SendHealthStatusToHostAgent();
		healthReport.ResetHealthObject();
	}
	else
	{
		healthReport.report.GuestOSVersion.assign(versionInfo);
	}

	if (IsWindows8Point1OrGreater() && !IsWindows10OrGreater())
	{
		initializationStatus = initialize(healthReport);
	}
	else
	{
		LOGINFOW(L"Windows version is %s. Not calling initialize", versionInfo);
		healthReport.report.DebugMessage.assign(L"Windows version is not server 2012 R2. so not calling initialize");
		healthReport.report.DriverInstallPreSetupNotAttempted = true;
		healthReport.SendHealthStatusToHostAgent();
		healthReport.ResetHealthObject();
	}
    /*
      Check if driver installation is needed by verifying OS version
    */

    isInstallRequired = vfplugin.IsMlxVFDriverInstallNeeded();    
    
    do {   
        if(isInstallRequired && initializationStatus == 0)
        {
            wchar_t currentPath[MAX_PATH];
            status = ERROR_SUCCESS;
        
            LOGINFO("Going to install Mellanox Driver");
            
            if (GetCurrentDirectory(MAX_PATH, currentPath) == 0) {
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
            
            if (status != ERROR_SUCCESS &&  status != REBOOT_REQD) {
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
            healthReport.report.DriverInstallNotAttempted=true;
			healthReport.SendHealthStatusToHostAgent();
			healthReport.ResetHealthObject();
        }
    }while(0);


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
				healthReport.report.GuestOSVersion.c_str(), nmAgentReport.NumNonFunctioningDevices, nmAgentReport.IsInstalled, totalDevices, nmAgentReport.NumNonFunctioningDevices);

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