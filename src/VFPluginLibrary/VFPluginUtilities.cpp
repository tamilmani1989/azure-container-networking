/*++

Copyright (c) Microsoft Corporation

Module Name:

VFPluginUtilities.cpp

Abstract:

This file contains implementation of utility methods used by VFPlugin.
Author:

Tamilmani Manoharan (tamanoha) 10-October-2017

--*/

#include <cpprest/http_listener.h>
#include <cpprest/http_client.h>
#include <cpprest/uri.h>
#include <cpprest/asyncrt_utils.h>
#include <cpprest/json.h>
#include "..\Logging\Logging.h"
#include "VFPluginUtilities.h"

using namespace std;
using namespace web;
using namespace http;
using namespace web::http::client;
using namespace experimental::listener;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;


void VFPluginUtilities::GetHostNetAgentURL(
    wchar_t *url, 
    size_t urlSize,
    wchar_t *path
)
/**
Routine description:

    This function constructs wireserver agent url based on path.

Arguments:
    url - result od constructed url will be placed in this buffer.
    urlSize - size of url buffer.
    path - path of url.

Return Value:

**/
{
    HKEY returnKey;
    DWORD type = REG_SZ;
    DWORD size;
    wchar_t addr[MAX_PATH];
    bool openFlag = false;
    LONG nError;
    
    size = (DWORD)sizeof(addr);

    LOGINFOW(L"Going to open registry in Iaas %s", IAAS_KEY);
    
    nError = RegOpenKeyEx(HKEY_LOCAL_MACHINE, IAAS_KEY, NULL, KEY_ALL_ACCESS, &returnKey);    
    
    if (nError == ERROR_SUCCESS)
    {
        LOGINFO("Registry open success");
        openFlag =true;
        LONG nError = RegQueryValueEx(returnKey, HOSTNETAGENT_ADDRESS_KEY, nullptr, &type, (LPBYTE)addr, &size);

        if (nError == ERROR_SUCCESS)
        {
            LOGINFOW(L"Retrieved HostNetAgentAddress from registry %s", addr);
            goto Cleanup;
        }
        LOGERRW(L"Retrieved HostNetAgentAddress from IAAS registry failed %d", nError);
    }
    
    if(openFlag)
    {
        RegCloseKey(returnKey);
        openFlag= false;
    }

    LOGINFOW(L"Going to open registry in paas %s", PAAS_KEY);
    
    nError = RegOpenKeyEx(HKEY_LOCAL_MACHINE, PAAS_KEY, NULL, KEY_ALL_ACCESS, &returnKey);
    
    if (nError != ERROR_SUCCESS)
    {
        LOGWARNW(L"Error while retrieving wireserver url registry key");
        goto Cleanup;
    }
    
    openFlag =true;
    
    nError = RegQueryValueEx(returnKey, HOSTNETAGENT_ADDRESS_KEY, nullptr, &type, (LPBYTE)addr, &size);
    
    if (nError != ERROR_SUCCESS)
    {
        LOGINFOW(L"Retrieved HostNetAgentAddress from IAAS registry %d", nError);
        goto Cleanup;
    }

Cleanup:

    if(openFlag)
    {
        RegCloseKey(returnKey);
    }

    if(nError != ERROR_SUCCESS)
    {
        if( swprintf_s(url, urlSize, L"http://%s:80/%s", DEFAULT_HOSTNETAGENT_ADDRESS, path) < 0)
        {
            LOGERRW(L"swprint_s error constructing url path %s", DEFAULT_HOSTNETAGENT_ADDRESS);
        }
    }
    else
    {
        LOGINFOW(L"Address returned from registry : %s", addr);
        if( swprintf_s(url, urlSize, L"http://%s:80/%s", addr, path) < 0)
        {
            LOGERRW(L"swprint_s error constructing url path addr %s", addr);
        }        
    }

    LOGINFOW(L"wireserver url : %s", url);
}



void VFPluginUtilities::SetupLogging(wchar_t *logFolder, wchar_t *logPrefix)
/**
Routine description:

    This function sets up logging for VFPlugin.

Arguments:
    logFolder - logFolder path.
    logPrefix -  logprefix for filename.

Return Value:

**/
{
    wchar_t *logDestFolder = NULL;
    
    size_t len = wcslen(logFolder) + wcslen(L"\\") + wcslen(logPrefix) + sizeof(wchar_t);

    logDestFolder = new wchar_t[len];
    if(!logDestFolder)
    {
        wprintf(L"Allocation of logdest folder failed\n");
        return;
    }

    if(swprintf_s(logDestFolder, len, L"%s\\%s", logFolder, logPrefix) < 0)
    {
        wprintf(L"swprintf_s failed constructing log dest folder\n");
        delete[] logDestFolder;
        return;
    }

    if (!CreateDirectory(logDestFolder, NULL))
    {
        DWORD error = GetLastError();
        if (error != ERROR_ALREADY_EXISTS)
        {
            wprintf(L"Failed to create log folder [%s] due to erroro (%d)\n", logDestFolder, error);
            delete[] logDestFolder;
            return;
        }
    }

    LOGSETUP(logDestFolder, logPrefix);
    delete[] logDestFolder;
}

void VFPluginUtilities::SaveInstalledState()
/**
Routine description:

    This function saves state if netagent installs mellanox driver in registry.

Arguments:

Return Value:

**/
{
    HKEY returnKey;
    DWORD type = REG_SZ;
    bool openFlag = false;
    LONG nError;

    LOGINFOW(L"Going to open registry in Iaas for saving install state %s", IAAS_KEY);
    
    nError = RegOpenKeyEx(HKEY_LOCAL_MACHINE, IAAS_KEY, NULL, KEY_ALL_ACCESS, &returnKey);    
    
    if (nError == ERROR_SUCCESS)
    {
        openFlag = true;
        nError = RegSetKeyValue(returnKey, nullptr, INSTALL_KEY, type, INSTALL_VALUE, (DWORD) (wcsnlen(INSTALL_VALUE, MAX_PATH) + 1) * sizeof(wchar_t));
        if (nError == ERROR_SUCCESS)
        {
            LOGINFO("Registry save succeeded");
            goto Cleanup;
        }
    }
   
    if(openFlag)
    {
        RegCloseKey(returnKey);
        openFlag= false;
    }

    LOGINFOW(L"Going to open registry in paas for saving install state %s", PAAS_KEY);
    
    nError = RegOpenKeyEx(HKEY_LOCAL_MACHINE, PAAS_KEY, NULL, KEY_ALL_ACCESS, &returnKey);
    
    if (nError != ERROR_SUCCESS)
    {
        LOGWARNW(L"Error while retrieving Wireserver url registry key");
        goto Cleanup;
    }
    
    openFlag =true;
           
    nError = RegSetKeyValue(returnKey, nullptr, INSTALL_KEY, type, INSTALL_VALUE, (DWORD) (wcsnlen(INSTALL_VALUE, MAX_PATH) + 1) * sizeof(wchar_t));
    
    if (nError != ERROR_SUCCESS)
    {
        LOGINFO("Registry save failed");
        goto Cleanup;
    }
    
    LOGINFO("Registry save succeeded");
Cleanup:

    if(openFlag)
    {
        RegCloseKey(returnKey);
    } 
}


int VFPluginUtilities::GetWindowsVersion(
	wchar_t *versionInfo, 
	size_t versionInfoSize
)
/**
Routine description:

This function retrieves windows version.

Arguments:
	versionInfo - buffer to store windows version.
	versionInfoSize - size of buffer.

Return Value:

	return 0 on success else 1.

**/
{
	static const wchar_t kernel32[] = L"\\kernel32.dll";
	wchar_t *path = NULL;
	void *ver = NULL, *block;
	UINT n;
	BOOL r;
	DWORD versz, blocksz;
	VS_FIXEDFILEINFO *vinfo;
	int status = 1;

	path = new wchar_t[MAX_PATH];
	if (!path)
	{
		LOGINFOW(L"GetWindowsVersion::Allocation failed for path");
		abort();
	}

	n = GetSystemDirectory(path, MAX_PATH);
	if (n >= MAX_PATH || n == 0 ||
		n > MAX_PATH - sizeof(kernel32) / sizeof(*kernel32))
	{
		LOGINFOW(L"GetWindowsVersion::GetSystemDirectory failed");
		abort();
	}

	memcpy(path + n, kernel32, sizeof(kernel32));
	LOGINFOW(L"Filename to check for version: %s", path);

	versz = GetFileVersionInfoSize(path, NULL);
	if (versz == 0)
	{
		LOGINFOW(L"GetFileVersionInfoSize failed for path %s", path);
		goto Cleanup;
	}

	ver = malloc(versz);
	if (!ver)
	{
		LOGINFOW(L"GetWindowsVersion::Allocation failed for version");
		abort();
	}

	r = GetFileVersionInfo(path, 0, versz, ver);
	if (!r)
	{
		LOGINFOW(L"GetWindowsVersion::GetFileVersionInfo failed for path %s", path);
		goto Cleanup;
	}

	r = VerQueryValue(ver, L"\\", &block, (PUINT)&blocksz);
	if (!r || blocksz < sizeof(VS_FIXEDFILEINFO))
	{
		LOGINFOW(L"GetWindowsVersion::VerQueryValue failed for path %s", path);
		goto Cleanup;
	}

	vinfo = (VS_FIXEDFILEINFO *)block;

	if (swprintf_s(versionInfo, versionInfoSize,
		L"%d.%d.%d",
		(int)HIWORD(vinfo->dwProductVersionMS),
		(int)LOWORD(vinfo->dwProductVersionMS),
		(int)HIWORD(vinfo->dwProductVersionLS)) <= 0)
	{
		LOGINFOW(L"GetWindowsVersion::swprintf_s failed for path %s", path);
		goto Cleanup;
	}

	status = 0;

Cleanup:
	if (path)
	{
		delete[] path;
	}
	if (ver)
	{
		free(ver);
	}

	return status;
}

DWORD VFPluginUtilities::RunDllCmd(
)
/**
Routine description:

This function runs rundll command to cleanup zombie devices.

Arguments:

Return Value:
	returns 0 on success else returns errorcode.

**/
{
	STARTUPINFO sinfo;
	PROCESS_INFORMATION pinfo;
	DWORD dwExitCode = 1;
	wchar_t rundll_cmd[MAX_PATH];
	wchar_t winDir[MAX_PATH];

	ZeroMemory(&sinfo, sizeof(sinfo));
	sinfo.cb = sizeof(sinfo);
	ZeroMemory(&pinfo, sizeof(pinfo));
	

	if (!GetSystemDirectory(winDir, ARRAYSIZE(winDir))) {
		LOGINFOW(L"RunDllCmd::GetWindowsDirectory failed\n");
		goto Cleanup;
	}

	LOGINFOW(L"GetSystemDirectory %s\n", winDir);

	if (swprintf_s(rundll_cmd, sizeof(rundll_cmd), L"%s\\%s", winDir, RUNDLL_COMMAND) < 0)
	{
		LOGERRW(L"RunDllCmd::swprintf_s failed for netcfgcmd");
		goto Cleanup;
	}

	LOGINFOW(L"Create process for rundll cmd %s args %s", rundll_cmd, RUNDLL_ARGS);

	if (CreateProcess(rundll_cmd, RUNDLL_ARGS, NULL, NULL, FALSE, NORMAL_PRIORITY_CLASS | CREATE_NO_WINDOW, NULL, NULL, &sinfo, &pinfo))
	{
		LOGINFO("RunDllCmd::Wait for rundll process for 60 seconds");
		
		DWORD ret = WaitForSingleObject(pinfo.hProcess, 60000);
		if (ret != WAIT_OBJECT_0)
		{
			LOGERRW(L"RunDllCmd::WaitForSingleObject failed with errorcode %d", ret);
			dwExitCode = ret;

			if (!TerminateProcess(pinfo.hProcess, WAIT_TIMEOUT))
			{
				LOGERRW(L"RunDllCmd::TerminateProcess failed %d", GetLastError());
			}
			else
			{
				ret = WaitForSingleObject(pinfo.hProcess, 30000);
				if (ret != WAIT_OBJECT_0)
				{
					LOGERRW(L"RunDllCmd::TerminateProcess not terminating process for more than 30 seconds %d", ret);
				}
			}
		}
		else
		{
			if (GetExitCodeProcess(pinfo.hProcess, &dwExitCode) == FALSE)
			{
				LOGERRW(L"RunNetCfgCmd::GetExitCodeProcess failed %d", GetLastError());
			}
		}

		CloseHandle(pinfo.hProcess);
		CloseHandle(pinfo.hThread);
	}
	else
	{
		LOGERRW(L"Create process failed %d", GetLastError());
	}

Cleanup:
	return dwExitCode;
}

DWORD VFPluginUtilities::RunNetCfgCmd(
)
/**
Routine description:

This function runs netcfg command to install Windows protocol driver.

Arguments:

Return Value:
	returns 0 on success else returns errorcode.

**/
{
	wchar_t winDir[MAX_PATH];
	wchar_t netSvcPath[MAX_PATH];
	wchar_t netcfg_cmd[MAX_PATH];
	wchar_t netcfg_args[MAX_PATH];
	STARTUPINFO sinfo;
	PROCESS_INFORMATION pinfo;
	DWORD dwExitCode = 1;


	if (!GetSystemDirectory(winDir, ARRAYSIZE(winDir))) {
		LOGINFOW(L"RunNetCfgCmd::GetWindowsDirectory failed\n");
		goto Cleanup;
	}

	LOGINFOW(L"GetSystemDirectory %s\n", winDir);

	if (swprintf_s(netcfg_cmd, sizeof(netcfg_cmd), L"%s\\%s", winDir, NETCFG_COMMAND) < 0)
	{
		LOGERRW(L"RunDllCmd::swprintf_s failed for netcfgcmd");
		goto Cleanup;
	}

	PathCombine(netSvcPath, winDir, NETSVC_INF_PATH);
	LOGINFOW(L"NetSvc Inf path %s\n", netSvcPath);
	
	if (swprintf_s(netcfg_args, sizeof(netcfg_args), L"%s %s", NETCFG_ARGS, netSvcPath) < 0)
	{
		LOGERRW(L"RunNetCfgCmd::swprintf_s failed for netcfgcmd");
		goto Cleanup;
	}


	ZeroMemory(&sinfo, sizeof(sinfo));
	sinfo.cb = sizeof(sinfo);
	ZeroMemory(&pinfo, sizeof(pinfo));

	LOGINFOW(L"Create process for netcfg cmd %s args %s", netcfg_cmd, netcfg_args);
	if (CreateProcess(netcfg_cmd, netcfg_args, NULL, NULL, FALSE, NORMAL_PRIORITY_CLASS | CREATE_NO_WINDOW, NULL, NULL, &sinfo, &pinfo))
	{
		DWORD ret = WaitForSingleObject(pinfo.hProcess, 60000);
		if (ret != WAIT_OBJECT_0)
		{
			LOGERRW(L"RunNetCfgCmd::WaitForSingleObject failed with errorcode %d", ret);
			dwExitCode = ret;

			if (!TerminateProcess(pinfo.hProcess, WAIT_TIMEOUT))
			{
				LOGERRW(L"RunNetCfgCmd::TerminateProcess failed %d", GetLastError());
			}
			else
			{
				ret = WaitForSingleObject(pinfo.hProcess, 30000);
				if (ret != WAIT_OBJECT_0)
				{
					LOGERRW(L"RunNetCfgCmd::TerminateProcess not terminating process for more than 30 seconds %d", ret);
				}
			}
		}
		else
		{
			if (GetExitCodeProcess(pinfo.hProcess, &dwExitCode) == FALSE)
			{
				LOGERRW(L"RunNetCfgCmd::GetExitCodeProcess failed %d", GetLastError());
			}
		}

		CloseHandle(pinfo.hProcess);
		CloseHandle(pinfo.hThread);
	}
	else
	{
		LOGERRW(L"RunNetCfgCmd::Create proceess failed %d", GetLastError());
	}

Cleanup:

	return dwExitCode;
}
