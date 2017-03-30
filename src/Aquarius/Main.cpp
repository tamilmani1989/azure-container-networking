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
#include "AquariusInstaller.h"
#include "AquariusSvcBase.h"
#include "AquariusService.h"
#include "../Shared/RegistryOps.h"
#include "../Shared/Utilities.h"
#pragma endregion


// 
// Settings of the service
// 

// Internal name of the service
#define SERVICE_NAME             L"AquariusSvc"

// Displayed name of the service
#define SERVICE_DISPLAY_NAME     L"Azure Aquarius"

// Service start options.
#define SERVICE_START_TYPE       SERVICE_AUTO_START

// List of service dependencies - "dep1\0dep2\0\0"
#define SERVICE_DEPENDENCIES     L""

// The name of the account under which the service should run
#define SERVICE_ACCOUNT          nullptr

// The password to the service account name
#define SERVICE_PASSWORD         nullptr


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
    if ((argc > 1) && ((*argv[1] == L'-' || (*argv[1] == L'/'))))
    {
        if (_wcsicmp(L"install", argv[1] + 1) == 0)
        {
            SetupEventLogging();
            HKEY configKey = RegistryOps::OpenRegistryKey(HKEY_LOCAL_MACHINE, L"SOFTWARE\\Aquarius");
            if (configKey == nullptr)
            {
                AquariusUtilities::WriteEventLogEntry(L"Unable to create config key for Aquarius..");
            }
            // Install the service when the command is 
            // "-install" or "/install".
            InstallService(
                SERVICE_NAME,               // Name of service
                SERVICE_DISPLAY_NAME,       // Name to display
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
            UninstallService(SERVICE_NAME);
        }
    }
    else
    {
        wprintf(L"Parameters:\n");
        wprintf(L" -install  to install the service.\n");
        wprintf(L" -remove   to remove the service.\n");

        AquariusService service(SERVICE_NAME);
        if (!AquariusSvcBase::Run(service))
        {
            wprintf(L"Service failed to run w/err 0x%08lx\n", GetLastError());
        }
    }

    return 0;
}