/*++

Copyright (c) Microsoft Corporation

Module Name:

VFPluginUtilities.h

Abstract:

Header contains declaration of utility methods used by VFPlugin.
Author:

Tamilmani Manoharan (tamanoha) 10-October-2017

--*/

#pragma once
#include <Windows.h>
#include <Shlwapi.h>

#define WIRESERVER_URL L"http://127.0.0.1:5555/PluginStatus"
#define IAAS_KEY L"SOFTWARE\\Microsoft\\NetAgent"
#define PAAS_KEY L"SOFTWARE\\Microsoft\\GuestAgent"
#define HOSTNETAGENT_ADDRESS_KEY L"HostNetAgentAddress"
#define DEFAULT_HOSTNETAGENT_ADDRESS L"168.63.129.16"

#define INSTALL_KEY L"VF_INSTALL"
#define INSTALL_VALUE L"True"

#define RUNDLL_COMMAND L"RUNDLL32.exe"
#define RUNDLL_ARGS L" pnpclean1.dll,RunDLL_PnpClean /devices /maxclean"
#define NETCFG_COMMAND L"netcfg.exe" 
#define NETCFG_ARGS L" -c p -i netvsc_vfpp -l"
#define NETSVC_INF_PATH L"INF\\wnetvsc_vfpp.inf"

class VFPluginUtilities
{
public:

	static void GetHostNetAgentURL(    
        wchar_t *url, 
        size_t urlSize,
        wchar_t *path
    );
    
    static void SetupLogging(wchar_t *logFolder, wchar_t *logPrefix);

    static void SaveInstalledState();

	static int GetWindowsVersion(
		wchar_t *versionInfo,
		size_t versionInfoSize
	);

	static DWORD RunDllCmd(
	);

	static DWORD RunNetCfgCmd(
	);

};

