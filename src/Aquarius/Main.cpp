/****************************** Module Header ******************************\
* Module Name:  Main.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
* 
* The file defines the entry point of the application. According to the 
* arguments in the command line, the function installs or uninstalls or 
* starts the service by calling into different routines.
* 
\***************************************************************************/
#define CRTDBG_MAP_ALLOC
#include<stdlib.h>
#include<crtdbg.h>
#pragma region Includes
#include <stdio.h>
#include <windows.h>
#include "../AquariusLibrary/AquariusInstaller.h"
#include "../AquariusLibrary/AquariusSvcBase.h"
#include "../AquariusLibrary/AquariusService.h"
#include "../Shared/RegistryOps.h"
#include "../Shared/Utilities.h"
#pragma endregion

// 
// Settings of the service
// 

void SetupEventLogging()
{
    HKEY configKey = RegistryOps::OpenRegistryKey(HKEY_LOCAL_MACHINE, 
        L"SYSTEM\\CurrentControlSet\\Services\\EventLog\\Application\\AquariusSvc");

    if (configKey == nullptr)
    {
        AquariusUtilities::WriteEventLogEntry(L"Unable to create registry entry so that events from Aquarius can be written properly..");
        return;
    }

	wchar_t *aquariusLocation = new wchar_t[MAX_PATH];
	
    AquariusUtilities::PopulateServiceExeLocation(aquariusLocation);
	wcsncat_s(aquariusLocation, MAX_PATH, L"\\AqEventMessage.dll", MAX_PATH);
	RegistryOps::PutStringInRegistry(configKey, L"CategoryMessageFile", aquariusLocation);
	RegistryOps::PutStringInRegistry(configKey, L"EventMessageFile", aquariusLocation);
	RegistryOps::PutStringInRegistry(configKey, L"ParameterMessageFile", aquariusLocation);
	RegistryOps::PutDWORDInRegistry(configKey, L"TypesSupported", 0x7);
	RegistryOps::PutDWORDInRegistry(configKey, L"CategoryCount", 0x1);
	delete[] aquariusLocation;
}

//
//  FUNCTION: wmain(int, wchar_t *[])
//
//  PURPOSE: entrypoint for the application.
// 
//  PARAMETERS:
//    argc - number of command line arguments
//    argv - array of command line arguments
//
//  RETURN VALUE:
//    none
//
//  COMMENTS:
//    wmain() either performs the command line task, or run the service.
//

int wmain(int argc, wchar_t *argv[])
{
    AquariusInstaller installer;
	wchar_t svcDisplayName[MAX_PATH];

    int ret=1;

    if ((argc > 1) && ((*argv[1] == L'-' || (*argv[1] == L'/'))))
    {
        wchar_t svcName[MAX_PATH];
        
        SetupEventLogging();
        
        if(AquariusUtilities::GetServiceName(svcName, sizeof(svcName)) != 0)
        {
            AquariusUtilities::WriteEventLogEntry(L"NetAgent:: Unable to retrieve service name..");
            wprintf(L"Unable to retrieve service name..");
            return ret;
        }

        wprintf(L"Service Name :%s", svcName);

		if (wcscmp(svcName, AQUARIUSSVC) == 0)
		{
			if (wcscpy_s(svcDisplayName, AQUARIUS_DISPLAY_NAME) != 0)
			{
				AquariusUtilities::WriteEventLogEntry(L"NetAgent:: wcscpy_s failed copying display name");
				wprintf(L"wcscpy_s failed copying display name..");
			}
		}
		else
		{
			if (wcscpy_s(svcDisplayName, SERVICE_DISPLAY_NAME) != 0)
			{
				AquariusUtilities::WriteEventLogEntry(L"NetAgent:: wcscpy_s failed copying display name");
				wprintf(L"wcscpy_s failed copying display name..");
			}
		}

        if (_wcsicmp(L"install", argv[1] + 1) == 0)
        {
            HKEY configKey = RegistryOps::OpenRegistryKey(HKEY_LOCAL_MACHINE, L"SOFTWARE\\Aquarius");
            if (configKey == nullptr)
            {
                AquariusUtilities::WriteEventLogEntry(L"Unable to create config key for Aquarius..");
            }

            // Install the service when the command is 
            // "-install" or "/install".
            ret = installer.InstallService(
                svcName,               // Name of service
				svcDisplayName,       // Name to display
                SERVICE_START_TYPE,         // Service start type
                SERVICE_DEPENDENCIES,       // Dependencies
                SERVICE_ACCOUNT,            // Service running account
                SERVICE_PASSWORD            // Password of the account
                );       
        }
        else if (_wcsicmp(L"remove", argv[1] + 1) == 0)
        {
            // Uninstall the service when the command is 
            // "-remove" or "/remove".
            
            ret = installer.UninstallService(svcName);
        }
    }
    else
    {
        wprintf(L"Parameters:\n");
        wprintf(L" -install  to install the service.\n");
        wprintf(L" -remove   to remove the service.\n");

        AquariusService *service=nullptr;
        service = GetAquariusService();

        if (!AquariusSvcBase::Run(service))
        {
            wprintf(L"Service failed to run w/err 0x%08lx\n", GetLastError());
        }
    }

    return ret;
}