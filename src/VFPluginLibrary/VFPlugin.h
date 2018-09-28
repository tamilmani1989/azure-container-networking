/*++

Copyright (c) Microsoft Corporation

Module Name:

VFPluginUtilities.h

Abstract:

Header contains declaration of methods to install mellanox driver

Author:

Tamilmani Manoharan (tamanoha) 10-October-2017

--*/

#pragma once
#include <Windows.h>
#include <SetupAPI.h>
#include <iostream>
#include <difxapi.h>
#include <time.h>
#include <Shlwapi.h>
#include <winreg.h>
#include <WbemCli.h>
#include <comutil.h>
#include <assert.h>
#include <stdafx.h>
#include <Cfgmgr32.h>


#define LOG_FILE_PREFIX L"VFPlugin"

#define HTTP_PROTO L"http"
#define LOCALHOST L"localhost"
#define WS_PORT L"4433"
#define WS_STATUS L"/PluginStatus"
#define REBOOT_REQD 3010

#define NETMLX_INF_FILE L"INF\\netmlx4eth63.inf"

#define DRIVER_VERSION_KEY                              L"DriverVersion"
#define DRIVER_DESCRIPTION_KEY                          L"DriverDesc"
#define MAX_KEY_NAME_LENGTH                             255

//
// Mlx Driver 
//
#define NIC_DRIVER_REG_SEARCH_PATH                      L"SYSTEM\\CurrentControlSet\\Control\\Class\\{4d36e972-e325-11ce-bfc1-08002be10318}"
#define NIC_DEVICE_REG_SEARCH_PATH                      L"SYSTEM\\CurrentControlSet\\Enum\\MLX4\\ConnectX-3Pro_Eth"

#define MLX_DRIVER_DESCRIPTION_VALUE                    L"Mellanox"
#define MLX_ETHERNET_DRIVER_INF                         L"mlx4eth63.inf"
#define MLX_ETHERNET_DRIVER_SYS                         L"mlx4eth63.sys"
#define MLX_BUS_DRIVER_INF                              L"mlx4_bus.inf"
#define MLX_BUS_DRIVER_SYS                              L"mlx4_bus.sys"
#define MLX_IB_DRIVER_INF                               L"ipoib6x.inf"
#define MLX_IB_DRIVER_SYS                               L"ipoib6x.sys"
#define MLX_DEVICE_ID                                   L"PCI\\VEN_15B3"
#define MLX_CLASS                                       L"Net"
#define MLX_ETHERNET_DRIVER_INF_8Proc                   L"mlx4eth63_8Proc.inf"
#define MLX_ETHERNET_DRIVER_INF_12Proc                  L"mlx4eth63_12Proc.inf"
#define MLX_ETHERNET_DRIVER_INF_16Proc                  L"mlx4eth63_16Proc.inf"

#define MLX_VF_SUPPORTED_VERSION                        L"5.20.12400.0"
#define MLX_INF_SEARCH_PATH                             L"INF\\OEM*.INF"
#define MLX_VF_DRIVER_INSTALL_COMMAND                   L"MLNX_VPI_WinOF-5_20_All_Win2012R2_and_2016_x64.exe"
#define MLX_VF_DRIVER_INSTALL_COMMAND_LINE_ARGS         L"/S /v\"/qn\" /v/norestart"

struct Version {
    Version(PCWSTR versionStr) {
        major = 0;
        minor = 0;
        revision = 0;
        build = 0;
        swscanf_s(versionStr, L"%d.%d.%d.%d", &major, &minor, &revision, &build);
    }

    BOOL operator<(const Version& otherVersion) {
        if (major < otherVersion.major) {
            return TRUE;
        }

        if (major > otherVersion.major) {
            return FALSE;
        }

        if (minor < otherVersion.minor) {
            return TRUE;
        }

        if (minor > otherVersion.minor) {
            return FALSE;
        }

        if (revision < otherVersion.revision) {
            return TRUE;
        }

        if (revision > otherVersion.revision) {
            return FALSE;
        }

        if (build < otherVersion.build) {
            return TRUE;
        } else {
            return FALSE;
        }
    }

    BOOL operator==(const Version& otherVersion) {
        if (major == otherVersion.major &&
                minor == otherVersion.minor &&
                revision == otherVersion.revision &&
                build == otherVersion.build) {
            return TRUE;
        }

        return FALSE;
    }

    BOOL operator>(const Version& otherVersion) {
        if (major > otherVersion.major) {
            return TRUE;
        }

        if (major < otherVersion.major) {
            return FALSE;
        }

        if (minor > otherVersion.minor) {
            return TRUE;
        }

        if (minor < otherVersion.minor) {
            return FALSE;
        }

        if (revision > otherVersion.revision) {
            return TRUE;
        }

        if (revision < otherVersion.revision) {
            return FALSE;
        }

        if (build > otherVersion.build) {
            return TRUE;
        } else {
            return FALSE;
        }
    }

    int major, minor, revision, build;
};

class VFPlugin
{
public:
    ~VFPlugin();

    BOOL
    IsMlxVFDriverInstallNeeded(
    );
    
    BOOL
    IsDevicePresent(
        __in PCWSTR deviceSearchString
    );

    BOOL
    FindDriverVersionFromInf(
        __out wchar_t* Version
    );  

    BOOL
    FindDriverVersionFromNetmlx(
        __out wchar_t* Version
    );
       
    DWORD
    InstallMlxVFDriver(
        __in wchar_t* driverPackagePath
    );

    BOOL
    GetMlxVFDriverDetails(
        __out wchar_t *InstalledVersion,
        __in size_t InstalledVersionSize
    );

    int
    GetNumNonFunctioningDevices(
        __in PCWSTR DeviceSearchString,
        __out int *TotalDevices
    );

private:

    DWORD
    GetVersionFromInf(
        __in wchar_t* InfFileName,
        __in wchar_t* InfFilePath,
        __in wchar_t* ClassName,
        __in wchar_t* DriverDescription,
        __out wchar_t* Version
    );

    DWORD
    GetDevice(
        __in PCWSTR deviceSearchString
    );

    DWORD
    GetNicRegistryPath(
        __in PCWSTR driverDescription,
        __out LPWSTR* registryPath
    );

    DWORD
    GetRegistryValueSz (
        __in HKEY hKey,
        __in LPCWSTR registryName,
        __in LPCWSTR registryPath,
        __out LPWSTR registryValue,
        __in size_t registryValueSize
    );

    DWORD
    ExecuteProcess(
        __in LPWSTR processName,
        __in LPWSTR cmdLine
    );
};