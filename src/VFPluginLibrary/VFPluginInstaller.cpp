/*++

Copyright (c) Microsoft Corporation

Module Name:

VFPluginInstaller.cpp

Abstract:

This file contains methods to read version from registry and install mellanox driver

Author:

Tamilmani Manoharan (tamanoha) 10-October-2017

--*/

#include "VFPlugin.h"
#include "..\Logging\Logging.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

BOOL 
VFPlugin::GetMlxVFDriverDetails(
        __out wchar_t *InstalledVersion,
        __in size_t InstalledVersionSize
)
{
    DWORD status = ERROR_SUCCESS;
    LPWSTR registryPath = NULL;
    BOOL foundInstalledDriver = FALSE;

    //
    // Check if Device is installed on the system. This includes the case where the VF device was previously
    // installed but currently is not present (disabled or if the VF offload is currently disabled from the host)
    //
    if (VFPlugin::IsDevicePresent(MLX_DEVICE_ID)) {
       LOGINFOW(L"Device %s present\n", MLX_DEVICE_ID); 
	    //
	    // Check if the device is installed on the system. As mentioned above, it may be present
	    // but not installed
	    //

	    status = VFPlugin::GetNicRegistryPath(MLX_DRIVER_DESCRIPTION_VALUE,
	                                         &registryPath);

	    if (status == ERROR_SUCCESS) { 
           LOGINFOW(L"VFPlugin: Device %s registry path %s\n", MLX_DRIVER_DESCRIPTION_VALUE, registryPath);
            
            status = VFPlugin::GetRegistryValueSz(HKEY_LOCAL_MACHINE,
                                                 DRIVER_VERSION_KEY,
                                                 registryPath,
                                                 InstalledVersion,
                                                 InstalledVersionSize);
            
            if (status == ERROR_SUCCESS)  {
                LOGINFOW(L"VFPlugin: Device is installed with a driver version %s", InstalledVersion);
                foundInstalledDriver = TRUE;
                
            } else {
                LOGINFOW(L"VFPlugin: Failed to get version for %s device error 0x%X\n", MLX_DRIVER_DESCRIPTION_VALUE, status);
            } 
        } else {			
	        LOGINFOW(L"VFPlugin: Failed to get NIC registry path for driver %s error 0x%X\n",
	                 MLX_DRIVER_DESCRIPTION_VALUE, status);
	    }
    }   
    
    if (!foundInstalledDriver) {
         LOGINFOW(L"VFPlugin: Could not find the installed device with driver, checking if the driver package is installed\n");  

        //
        // See if the driver package is installed by querying the OEM files in the INF directory
        //

        foundInstalledDriver = FindDriverVersionFromNetmlx(InstalledVersion);

        if(!foundInstalledDriver)
        {
            foundInstalledDriver = FindDriverVersionFromInf(InstalledVersion);  
        }
    }

    if (registryPath) {
        delete [] registryPath;
    }
    return foundInstalledDriver;      
}

BOOL
VFPlugin::IsMlxVFDriverInstallNeeded(
)
/**
Routine description:

    This function checks whether VF driver upgrade is needed

Arguments:

Return Value:

    TRUE if driver upgrade needed, FALSE otherwise

**/
{
    BOOL foundInstalledDriver = TRUE;
    wchar_t installedVersion[MAX_PATH];
 
    if (!IsWindows8Point1OrGreater() || IsWindows10OrGreater()) {
        LOGINFOW(L"Driver upgrade is only supported on Windows Server 2012R2\n");
        goto Cleanup;
    }

    foundInstalledDriver = GetMlxVFDriverDetails(installedVersion, sizeof(installedVersion));

    if (foundInstalledDriver) {
        LOGINFOW(L"VFPlugin: Found the installed driver package with version %s\n", installedVersion);              
    } else {
        LOGINFOW(L"VFPlugin: Failed to find any installed package, install needed\n");
    }
           

    if (foundInstalledDriver) {
        if (Version(MLX_VF_SUPPORTED_VERSION) == Version(installedVersion)) {
            LOGINFOW(L"VFPlugin: VF driver Installed package version %s and supported version are same, not installing\n", installedVersion);
        } else if (Version(installedVersion) > Version(MLX_VF_SUPPORTED_VERSION)) {
            LOGINFOW(L"VFPlugin: VF driver Installed package version %s is higher than supported version %s, not installing \n", installedVersion, MLX_VF_SUPPORTED_VERSION);
        } else {
            LOGINFOW(L"VFPlugin: VF driver Installed package version %s is lower than supported version %s, upgrade not supported\n", installedVersion, MLX_VF_SUPPORTED_VERSION);
        }
     }

Cleanup:
    return !foundInstalledDriver;

}


BOOL
VFPlugin::IsDevicePresent(
    __in PCWSTR DeviceSearchString
)
/**
Routine description:

    This function checks whether device is present on the machine

Arguments:

    DeviceSearchString - Device Search string with no wild char

Return Value:

    True if present, FALSE otherwise

**/
{
    BOOL isDevicePresent = FALSE;
    DWORD status = ERROR_SUCCESS;

    //
    // Get device
    //

    status = VFPlugin::GetDevice(DeviceSearchString);

    if (status == ERROR_SUCCESS) {
        isDevicePresent = TRUE;
    }

    LOGINFOW(L"Device with search string %s present %s\n",
             DeviceSearchString,
             isDevicePresent ? L"True" : L"False");

    return isDevicePresent;
}

VFPlugin::~VFPlugin(
)
/**
Routine description:

    Destructor

Arguments:

Return Value:

**/
{

}

BOOL 
VFPlugin::FindDriverVersionFromNetmlx(
    __out wchar_t* Version)
/*++

Routine Description:
    Enumerates installed oem inf files on the machine and searches for the
    one with the matching DriverDescription

Arguments:

Return Value:

    EXIT_xxxx

--*/
{
    wchar_t winDir[MAX_PATH];
    wchar_t fileName[MAX_PATH];
    wchar_t infBasePath[MAX_PATH];
    DWORD status = ERROR_NO_SUCH_DEVINST;

    fileName[0] = 0;
    if (!GetWindowsDirectory(winDir, ARRAYSIZE(winDir))) {
        LOGINFOW(L"GetWindowsDirectory failed\n");
        goto Cleanup;        
    }

    LOGINFOW(L"GetWindowsDirectory %s\n", winDir);
    PathCombine(fileName, winDir, NETMLX_INF_FILE);
    PathCombine(infBasePath, winDir, L"inf");
    
    status = GetVersionFromInf(fileName,
                                   infBasePath,
                                   MLX_CLASS,
                                   MLX_DRIVER_DESCRIPTION_VALUE,
                                   Version);

    if (status == ERROR_SUCCESS) {
        return TRUE;
    } 

Cleanup:
    return FALSE;
}




BOOL 
VFPlugin::FindDriverVersionFromInf(
    __out wchar_t* Version)
/*++

Routine Description:
    Enumerates installed oem inf files on the machine and searches for the
    one with the matching DriverDescription

Arguments:

Return Value:

    EXIT_xxxx

--*/
{
    wchar_t winDir[MAX_PATH];
    wchar_t findName[MAX_PATH];
    wchar_t infBasePath[MAX_PATH];
    HANDLE hFind = INVALID_HANDLE_VALUE;
    WIN32_FIND_DATA wfd;
    DWORD status = ERROR_NO_SUCH_DEVINST;
    BOOL foundMatchingInf = FALSE;

    findName[0] = 0;
    if (!GetWindowsDirectory(winDir, ARRAYSIZE(winDir))) {
        LOGINFOW(L"GetWindowsDirectory failed\n");
        goto Cleanup;        
    }

    LOGINFOW(L"GetWindowsDirectory %s\n", winDir);

    PathCombine(findName, winDir, MLX_INF_SEARCH_PATH);
    LOGINFOW(L"INF search path %s\n", findName);
    PathCombine(infBasePath, winDir, L"inf");

    hFind = FindFirstFile(findName, &wfd);
    if (hFind == INVALID_HANDLE_VALUE) {
        //
        // No OEM driver packages on this machine.
        //
        LOGINFOW(L"No OEM INFs on this system\n");        
        goto Cleanup;
    }

    do {
        //
        // Open each OEM file and check if it is a Mellanox INF file. 
        //
        LOGINFOW(L"OEM file %s\n", wfd.cFileName);                
        status = GetVersionFromInf(wfd.cFileName,
                                   infBasePath,
                                   MLX_CLASS,
                                   MLX_DRIVER_DESCRIPTION_VALUE,
                                   Version);

        if (status == ERROR_SUCCESS) {
            foundMatchingInf = TRUE;
            break;
        } 
    } while (FindNextFile(hFind, &wfd));

    FindClose(hFind);

Cleanup:

    return foundMatchingInf;
}

DWORD
VFPlugin::GetVersionFromInf(
    __in wchar_t* InfFileName,
    __in wchar_t* InfFilePath,
    __in wchar_t* ClassName,
    __in wchar_t* DriverDescription,
    __out wchar_t* Version
)
/**
Routine description:

    This function gets version from INF if the Provider matches

Arguments:

    InfFileName - Inf file name

    InfFilePath - Inf file path

    ClassName - Device class

    DriverDescription - Driver/Provider name

    Version - Driver Version in the INF if found

Return Value:

    DWORD status - Version contains the driver version if successful

**/
{
    DWORD status = ERROR_SUCCESS;
    INFCONTEXT infContext;
    HINF hInf = INVALID_HANDLE_VALUE;
    BOOL result = FALSE;
    wchar_t driverPackageInfPath[MAX_PATH];
    wchar_t provider[MAX_INF_STRING_LENGTH];

    if (InfFileName == NULL || 
        InfFilePath == NULL || 
        ClassName == NULL || 
        DriverDescription == NULL ||
        Version == NULL) {
        status = ERROR_INVALID_PARAMETER;
        goto Cleanup;
    }

    Version[0] = 0;
    provider[0] = 0;;;

    PathCombine(driverPackageInfPath, InfFilePath, InfFileName);
    LOGINFOW(L"Opening INF file %s\n", driverPackageInfPath);

    hInf = SetupOpenInfFileW(
               driverPackageInfPath,
               ClassName,
               INF_STYLE_WIN4,
               NULL);

    if (INVALID_HANDLE_VALUE == hInf) {
        status = GetLastError();
        LOGINFOW(L"Failed to Open INF file %s Error:%d\n", driverPackageInfPath, status);
        goto Cleanup;
    }

    result = SetupFindFirstLineW(
                 hInf,
                 L"Version",
                 L"Provider",
                 &infContext);

    if (!result) {
        status = GetLastError();
        LOGINFOW(L"Failed to find Version/Provider section in inf file %s Error:%d\n", driverPackageInfPath, status);
        goto Cleanup;
    }
    
    result = SetupGetStringFieldW(
                 &infContext,
                 1,
                 provider,
                 MAX_INF_STRING_LENGTH,
                 NULL);

    if (0 == result) {
        status = GetLastError();
        LOGINFOW(L"Failed to find Provider value in inf file %s Error:%d\n", driverPackageInfPath, status);
        goto Cleanup;
    }

    if (wcsstr(provider, DriverDescription) == 0) {
        status = ERROR_NO_SUCH_DEVINST;
        LOGINFOW(L"INF Driver description %s is not %s Driver, ignoring it. Error:%d\n",
                 provider, DriverDescription, status);
        goto Cleanup;
    }

    result = SetupFindFirstLineW(
                 hInf,
                 L"Version",
                 L"DriverVer",
                 &infContext);

    if (!result) {
        status = GetLastError();
        LOGINFOW(L"Failed to find Version/DriverVer section in inf file %s. Error :%d\n", driverPackageInfPath, status);
        goto Cleanup;
    }

    result = SetupGetStringFieldW(
                 &infContext,
                 2,
                 Version,
                 MAX_INF_STRING_LENGTH,
                 NULL);

    if (0 == result) {
        status = GetLastError();
        LOGINFOW(L"Failed to find version value in inf file %s. Error:%d\n", driverPackageInfPath, status);
        goto Cleanup;
    }

    LOGINFOW(L"Driver version from INF %s\n", Version);
    status = ERROR_SUCCESS;
Cleanup:

    if (INVALID_HANDLE_VALUE != hInf) {
        SetupCloseInfFile(hInf);
        hInf = INVALID_HANDLE_VALUE;
    }

    return status;
}

DWORD
VFPlugin::GetDevice(
    __in PCWSTR DeviceSearchString
)
/**
Routine description:

    This function checks whether device is present on the machine
    This can be installed, but may not be present (disabled or the device
    is not plugged in)
    
    Caller should call in Cleanup

    if (*devInfo != INVALID_HANDLE_VALUE) {
        SetupDiDestroyDeviceInfoList(*devInfo);
    }

Arguments:

    DeviceSearchString - Device Search string with no wild char

Return Value:

    ERROR_SUCCESS S_OK If found;
    ERROR_NO_SUCH_DEVINST If not found

**/
{
    HDEVINFO hdevinfo = INVALID_HANDLE_VALUE;
    SP_DEVINFO_DATA devinfo;
    wchar_t instanceId[4096];
    DWORD count = 0;
    DWORD status = ERROR_SUCCESS;
	bool deviceFound = false;

    if (DeviceSearchString == NULL || wcslen(DeviceSearchString) == 0) {
        status = ERROR_INVALID_PARAMETER;
        goto Cleanup;
    }

    //
    // Open Device manager
    // Here, we are not checking for only present devices (DIGCF_PRESENT).
    // This is because a VF device could be installed, but not
    // present if the VF offload is disabled from the host or the VF device is disabled
    // in the VM. So the device driver for this device could still be installed
    // This will also be true for the case where the VF offload has been enabled so the
    // VF device exists in the VM but no driver has been installed. 
    // 
    hdevinfo = SetupDiGetClassDevs(NULL,
                                   0,
                                   NULL,
                                   DIGCF_ALLCLASSES);

    if (hdevinfo == INVALID_HANDLE_VALUE) {
        status = GetLastError();
        LOGINFOW(L"GetDevice :: SetupDiGetClassDevs failed with error 0x%X\n", status);
        goto Cleanup;
    }

    for (count = 0; ; count++) {
        devinfo.cbSize = sizeof(devinfo);

        //
        // Get Device Info
        //

        if (!SetupDiEnumDeviceInfo(hdevinfo,
                                   count,
                                   &devinfo)) {
            if (GetLastError() != ERROR_NO_MORE_ITEMS) {
                status = GetLastError();
                LOGINFOW(L"GetDevice :: SetupDiEnumDeviceInfo failed with error  0x%X\n", status);
                goto Cleanup;
            }

            break;
        }

        //
        // Get device Id
        //

        if (!SetupDiGetDeviceInstanceId(hdevinfo,
                                        &devinfo,
                                        instanceId,
                                        _countof(instanceId), NULL)) {
            status = GetLastError();
            LOGINFOW(L"SetupDiGetDeviceInstanceId failed with error  0x%X\n", status);
            goto Cleanup;
        } 
        
        if (wcsstr(instanceId, DeviceSearchString) != 0) {
            LOGINFOW(L"GetDevice :: Device found, DevicePath: %s\n", instanceId);
			deviceFound = true;
            break;
        }
    }

Cleanup:

    LOGINFOW(L"GetDevice :: Device with search string %s present %s\n",
             DeviceSearchString,
             deviceFound? L"true" : L"false");

    if (!deviceFound) {
        status = ERROR_NO_SUCH_DEVINST;
    }

    if(hdevinfo != INVALID_HANDLE_VALUE)
    {
        SetupDiDestroyDeviceInfoList(hdevinfo);
    }

    return status;
}

DWORD
VFPlugin::GetNicRegistryPath(
    __in PCWSTR DriverDescription,
    __out LPWSTR* RegistryPath
)
/**
Routine description:

    This function Gets the registry subkey from the software key matching the driver description.

Arguments:

    DriverDescription - Name of driver e.g. Mellanox
    RegistryPath - Registry subkey path if the device with this driver is located.

Return Value:

    ERROR_SUCCESS if success else return error code

**/
{
    HKEY networkSubkey = NULL;
    HKEY interfaceSubkey = NULL;
    DWORD status = ERROR_SUCCESS;
    LPWSTR pwszKeyName = NULL;
    LPWSTR pwszDriverDescription = NULL;
    BOOLEAN ifMatch = TRUE;
    DWORD currKeyIndex = 0;
    DWORD dwLen = 0;
    DWORD driverDescriptionLength = 0;
    ULONG numTotalKeys = 0;
    ULONG numMaxSubkeyLen = 0;

    if (DriverDescription == NULL || wcslen(DriverDescription) == 0) {
        status = ERROR_INVALID_PARAMETER;
        goto Cleanup;
    }

    //
    // Open the registry hive for software key
    //

    status =  RegOpenKeyEx(HKEY_LOCAL_MACHINE,
                           NIC_DRIVER_REG_SEARCH_PATH,
                           0,
                           KEY_ALL_ACCESS,
                           &networkSubkey);

    if (status != ERROR_SUCCESS) {
        networkSubkey = NULL;
        LOGINFOW(L"Possible registry corruption. Cannot open %s\n",
                 NIC_DRIVER_REG_SEARCH_PATH);
        goto Cleanup;
    }

    //
    // Get maximum name length for Keys
    //
    status = RegQueryInfoKey(networkSubkey,
                             NULL,
                             NULL,
                             NULL,
                             &numTotalKeys,
                             &numMaxSubkeyLen,
                             NULL,
                             NULL,
                             NULL,
                             NULL,
                             NULL,
                             NULL);

    if (status != ERROR_SUCCESS) {
        LOGINFOW(L"Failed to enumerate first key path %s\n",
                 NIC_DRIVER_REG_SEARCH_PATH);
        goto Cleanup;
    }

    LOGINFOW(L"Enumerating the subkeys under %s, nNumber of subkeys %d \n",
        NIC_DRIVER_REG_SEARCH_PATH, numTotalKeys);

    //
    // Account for NULL terminator
    //

    numMaxSubkeyLen++;

    //
    // Loop through the subkeys under the Net devices software key 
    //

    while (currKeyIndex < numTotalKeys) {
        //
        // Clear up any allocations from the previous loop
        //
        if (pwszKeyName) {
            delete [] pwszKeyName;
            pwszKeyName = NULL;
        }
        
        if (pwszDriverDescription) {
            delete [] pwszDriverDescription;
            pwszDriverDescription = NULL;
        }

        if (interfaceSubkey) {
            RegCloseKey(interfaceSubkey);
            interfaceSubkey = NULL;
        }        
        
        dwLen = MAX_KEY_NAME_LENGTH + 1;
        driverDescriptionLength = 0;
        pwszKeyName = new wchar_t[dwLen];

        LOGINFOW(L"Enumerating at index %u\n",
                 currKeyIndex);

        status = RegEnumKeyExW(networkSubkey,
                               currKeyIndex++,
                               pwszKeyName,
                               &dwLen,
                               NULL,
                               NULL,
                               NULL,
                               NULL);

        if (status != ERROR_SUCCESS) {
            status = ERROR_SUCCESS;
            LOGINFOW(L"Failed to enumerate next key path %s at index %u, ignoring status\n",
                     NIC_DRIVER_REG_SEARCH_PATH, currKeyIndex - 1);
            continue;
        }

        ifMatch = TRUE;

        //
        // Check to make sure subket are 00XX format
        // Ignore this entry if it is not in this format
        //

        for (size_t i = 0; i < wcslen(pwszKeyName); i++) {
            if (!isdigit(pwszKeyName[i])) {
                ifMatch = FALSE;
                break;
            }
        }

        if (!ifMatch) {
            LOGINFOW(L"Subkey %s under key path %s at index %u is not in 00XX format, ignoring\n",
                     pwszKeyName, NIC_DRIVER_REG_SEARCH_PATH, currKeyIndex - 1);
            
            continue;
        }

        status =  RegOpenKeyEx(networkSubkey,
                               pwszKeyName,
                               0,
                               KEY_ALL_ACCESS,
                               &interfaceSubkey);

        if (status != ERROR_SUCCESS) {
            status = ERROR_SUCCESS;
            LOGINFOW(L"Possible registry corruption. Cannot open subkey %s under key path %s at index %u, ignoring this subkey\n",
                     pwszKeyName, NIC_DRIVER_REG_SEARCH_PATH, currKeyIndex - 1);
            interfaceSubkey = NULL;
            continue;
        }

        status = RegQueryValueExW(interfaceSubkey,
                                  DRIVER_DESCRIPTION_KEY,
                                  NULL,
                                  NULL,
                                  NULL,
                                  &driverDescriptionLength);

        if (status != ERROR_SUCCESS) {
            status = ERROR_SUCCESS;
            LOGINFOW(L"Failed to get DriverDesc value in subkey %s under key path %s at index %u, ignoring this subkey\n",
                     pwszKeyName, NIC_DRIVER_REG_SEARCH_PATH, currKeyIndex - 1);
            continue;
        }

        pwszDriverDescription = new wchar_t[driverDescriptionLength / sizeof(wchar_t)];

        if (pwszDriverDescription == NULL) {
            status = ERROR_OUTOFMEMORY;
            LOGINFOW(L"Failed to allocate DriverDescription buffer for subkey %s under key path %s at index %u\n",
                     pwszKeyName, NIC_DRIVER_REG_SEARCH_PATH, currKeyIndex - 1);
            
            goto Cleanup;
        }

        status = RegQueryValueExW(interfaceSubkey,
                                  DRIVER_DESCRIPTION_KEY,
                                  NULL,
                                  NULL,
                                  (PBYTE) pwszDriverDescription,
                                  &driverDescriptionLength);

        if (status != ERROR_SUCCESS) {
            status = ERROR_SUCCESS;
            LOGINFOW(L"Failed to get driverDesc value for subkey %s under key path %s at index %u, ignoring error\n",
                     pwszKeyName, NIC_DRIVER_REG_SEARCH_PATH, currKeyIndex - 1);
            continue;
        }

        if (wcsstr(pwszDriverDescription, DriverDescription) == 0) {
             LOGINFOW( L"Interface %s is not %s interface ignoring it\n",
                     pwszDriverDescription, DriverDescription);
            continue;
        }

        *RegistryPath = new wchar_t[MAX_KEY_NAME_LENGTH];

        if (RegistryPath == NULL) {
            LOGERRW(L"Failed to allocate registryPath buffer for subkey %s path %s\n",
                     pwszKeyName, NIC_DRIVER_REG_SEARCH_PATH);
            
            status = ERROR_OUTOFMEMORY;
            goto Cleanup;
        }

        swprintf_s(*RegistryPath,
                   MAX_KEY_NAME_LENGTH,
                   L"%s\\%s",
                   NIC_DRIVER_REG_SEARCH_PATH,
                   pwszKeyName);

         LOGINFOW(L"Found key %s for DriverDescription %s\n",
                 *RegistryPath, DriverDescription);

        break;
    }

Cleanup:

    if (pwszKeyName) {
        delete [] pwszKeyName;
        pwszKeyName = NULL;
    }

    if (pwszDriverDescription) {
        delete [] pwszDriverDescription;
        pwszDriverDescription = NULL;
    }

    if (interfaceSubkey) {
        RegCloseKey(interfaceSubkey);
        interfaceSubkey = NULL;
    }

    if (networkSubkey) {
        RegCloseKey(networkSubkey);
        networkSubkey = NULL;
    }

    if ((status == ERROR_SUCCESS) &&
        *RegistryPath == NULL) {
         LOGINFOW(L"Could not find %s\n",
                 DriverDescription);
        
        status = ERROR_NOT_FOUND;
    }
    return status;
}

DWORD
VFPlugin::GetRegistryValueSz (
    __in HKEY hKey,
    __in LPCWSTR registryName,
    __in LPCWSTR registryPath,
    __out LPWSTR registryValue,
    __in size_t  registryValueSize
)
/**
Routine description:

    This function Gets string registry value

Arguments:

    hKey                    - Root key HKLM/HKCU etc.

    registryName            - Name of the key e.g. *JumpPacketSize

    registryValue           - Value of the key

    registryPath            - Path to Network interface, function will look for all interfaces


Return Value:

    ERROR_SUCCESS if successful

**/
{
    DWORD status = ERROR_SUCCESS;
    DWORD size = 0;
    PWSTR buffer = NULL;
    ULONG arraySize = 0;

    if (registryName == NULL || registryPath == NULL || registryValue == NULL) {
        status = ERROR_INVALID_PARAMETER;
        goto Cleanup;
    }

    registryValue[0] = 0;

    status = RegGetValue(
                 hKey,
                 registryPath,
                 registryName,
                 RRF_RT_REG_SZ,
                 NULL,
                 NULL,
                 &size);

    if (status != ERROR_MORE_DATA && status != ERROR_SUCCESS) {
         LOGINFOW(L"RegGetValue path %s key %s failed with status 0x%X\n",
                 registryPath, registryName, status);
        goto Cleanup;
    }

    size += sizeof(WCHAR);
    arraySize = (size / sizeof(WCHAR));

    buffer = new WCHAR[arraySize];

    if (buffer == NULL) {
        status = ERROR_OUTOFMEMORY;
        goto Cleanup;
    }

    status = RegGetValue(
                 hKey,
                 registryPath,
                 registryName,
                 RRF_RT_REG_SZ,
                 NULL,
                 (PVOID)buffer,
                 &size);

    if (status != ERROR_SUCCESS) {
        LOGINFOW(L"RegGetValue path %s key %s failed with status 0x%X\n",
                 registryPath, registryName, status);
        goto Cleanup;
    }

    if(wcsncpy_s(registryValue, registryValueSize, buffer, size)!=0)
    {
        LOGINFOW(L"wcsncpy_s returned error copying registry value:%s", buffer);
    }

Cleanup:

    if (buffer != NULL) {
        delete [] buffer;
    }

    return status;
}

DWORD
VFPlugin::ExecuteProcess(
    __in LPWSTR processName,
    __in LPWSTR cmdLine
)
/**
Routine description:

    This function executes command line

Arguments:

    processName   - Process to execute

    cmdLine       - Command line arguments

Return Value:

    DWORD exit code

**/
{
    PROCESS_INFORMATION processInformation = {0};
    STARTUPINFO startupInfo = {0};
    startupInfo.cb = sizeof(startupInfo);
    DWORD error = 0;

     LOGINFOW(L"VFPlugin:ExecuteProcess ProcessName %s Command=%s\n", processName, cmdLine);

    //
    // Create the process
    //

    BOOL result = CreateProcess(processName, cmdLine,
                                NULL, NULL, FALSE,
                                NORMAL_PRIORITY_CLASS | CREATE_NO_WINDOW,
                                NULL, NULL, &startupInfo, &processInformation);

    if (!result) {
        error = GetLastError();

         LOGINFOW(L"ExecuteProcess failed at execute ProcessName %s Command=%s Error=%u\n", processName, cmdLine, error);

        goto Cleanup;
    } else {

        //
        // Successfully created the process.  Wait for it to finish.
        //

        WaitForSingleObject(processInformation.hProcess, INFINITE);

        //
        // Get the exit code.
        //

        result = GetExitCodeProcess(processInformation.hProcess, &error);

        //
        // Close the handles.
        //

        CloseHandle(processInformation.hProcess);
        CloseHandle(processInformation.hThread);

        if (!result) {

            //
            // Could not get exit code.
            //

             LOGINFOW(L"ExecuteProcess ProcessName %s Command=%s failed with error Error=%u\n", processName, cmdLine, error);
             goto Cleanup;
        }

         LOGINFOW(L"ExecuteProcess ProcessName %s Command=%s succeeded\n", processName, cmdLine);

    }

Cleanup:
    return error;
}


DWORD
VFPlugin::InstallMlxVFDriver(
    __in wchar_t* driverPackagePath
)
/**
Routine description:

    This function install MLX VF driver

Arguments:

    driverPackagePath -  path to the driver package
    
Return Value:

    DWORD status

**/
{
    DWORD status = ERROR_SUCCESS;
    wchar_t driverPackageFullPath[MAX_PATH];
//    wchar_t vfDriverInstallCommand[MAX_PATH];
//    wchar_t installedVersion[MAX_PATH];
//    LPWSTR registryPath = NULL;

    if (driverPackagePath == NULL) {
        status = ERROR_INVALID_PARAMETER;
        goto Cleanup;
    }

    //
    // This works for 2012 R2 and 2016. 
    // TODO Can there be non-x64 VMs? what about other versions of OS? 
    //
    PathCombine(driverPackageFullPath, driverPackagePath, MLX_VF_DRIVER_INSTALL_COMMAND);

    //
    // Install VF driver
    //
    LOGINFOW(L"Executing Command=%s\n", driverPackageFullPath);

    status = ExecuteProcess(driverPackageFullPath,
                            MLX_VF_DRIVER_INSTALL_COMMAND_LINE_ARGS);

    if (status != ERROR_SUCCESS && status != REBOOT_REQD) {
        LOGINFOW(L"VF Driver installation command failed with error %d.\n", status);
        goto Cleanup;
    }


Cleanup:
    return status;
}


int
VFPlugin::GetNumNonFunctioningDevices(
        __in PCWSTR DeviceSearchString,
        __out int *TotalDevices
)
/**
Routine description:

    This function checks whether device is present on the machine
    This can be installed, but may not be present (disabled or the device
    is not plugged in)
    
    Caller should call in Cleanup

    if (*devInfo != INVALID_HANDLE_VALUE) {
        SetupDiDestroyDeviceInfoList(*devInfo);
    }

Arguments:

    DeviceSearchString - Device Search string with no wild char

Return Value:

    ERROR_SUCCESS S_OK If found;
    ERROR_NO_SUCH_DEVINST If not found

**/
{
    HDEVINFO hdevinfo = INVALID_HANDLE_VALUE;
    SP_DEVINFO_DATA devinfo;
    wchar_t instanceId[4096];
    DWORD count = 0;
    DWORD status = 1;
    ULONG dnStatus = 0;
    ULONG pbmNumber = 0;
    int numNonFunctioning = 0;

    //
    // Open Device manager
    // Here, we are not checking for only present devices (DIGCF_PRESENT).
    // This is because a VF device could be installed, but not
    // present if the VF offload is disabled from the host or the VF device is disabled
    // in the VM. So the device driver for this device could still be installed
    // This will also be true for the case where the VF offload has been enabled so the
    // VF device exists in the VM but no driver has been installed. 
    // 
    hdevinfo = SetupDiGetClassDevs(NULL,
                                   0,
                                   NULL,
                                   DIGCF_ALLCLASSES);

    if (hdevinfo == INVALID_HANDLE_VALUE) {
        status = GetLastError();
        LOGINFOW(L"NonFunctioningDevs :: SetupDiGetClassDevs failed with error 0x%X\n", status);
        numNonFunctioning = -1;
        goto Cleanup;
    }

    for (count = 0; ; count++) {
        devinfo.cbSize = sizeof(devinfo);

        //
        // Get Device Info
        //

        if (!SetupDiEnumDeviceInfo(hdevinfo,
                                   count,
                                   &devinfo)) {
            if (GetLastError() != ERROR_NO_MORE_ITEMS) {
                status = GetLastError();
                LOGERRW(L"NonFunctioningDevs :: SetupDiEnumDeviceInfo failed with error  0x%X\n", status);
                numNonFunctioning = -1;
                goto Cleanup;
            }

            break;
        }

        if (!SetupDiGetDeviceInstanceId(hdevinfo,
                                        &devinfo,
                                        instanceId,
                                        _countof(instanceId), NULL)) {
            status = GetLastError();
            LOGINFOW(L"NonFunctioningDevs :: SetupDiGetDeviceInstanceId failed with error  0x%X\n", status);
            numNonFunctioning = -1;
            goto Cleanup;
        } 
        
        if (wcsstr(instanceId, DeviceSearchString) == NULL) {
             continue;
        }

        LOGINFOW(L"NonFunctioningDevs :: Device found. DevicePath: %s\n", instanceId);
        (*TotalDevices)++;

        status = CM_Get_DevNode_Status(&dnStatus, &pbmNumber, devinfo.DevInst, 0);
        LOGINFOW(L"CM_Get_DevNode_Status returned %d", status);

        if(status != CR_SUCCESS && status != CR_NO_SUCH_DEVNODE && status != CR_NO_SUCH_DEVINST)
        {
            LOGERRW(L"CM_Get_DevNode_Status failed with error %d",status);
            numNonFunctioning++;
            continue;
        }

        if(dnStatus == DN_HAS_PROBLEM || dnStatus == DN_PRIVATE_PROBLEM)
        {
            LOGERRW(L"CM_Get_DevNode_Status failed with DN_HAS_PROBLEM or DN_PRIVATE_PROBLEM pbmnumber %d",pbmNumber);
            numNonFunctioning++;       
        }
    }

Cleanup:
    if (hdevinfo != INVALID_HANDLE_VALUE) {
        SetupDiDestroyDeviceInfoList(hdevinfo);
    }

    return numNonFunctioning;
}


