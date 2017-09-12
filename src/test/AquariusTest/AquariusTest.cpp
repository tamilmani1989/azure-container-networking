/*++

Copyright (c) Microsoft Corporation

Module Name:

AquariusTest.cpp

Abstract:

This contains test methods for testing the functionalities of aquarius service.

Author:

Tamilmani Manoharan (tamanoha) 08-Aug-2017

--*/

#ifndef UNICODE
#define UNICODE
#endif

#define _WINSOCK_DEPRECATED_NO_WARNINGS

#include <windows.h>
#include <stdio.h>
#include <string>
#include "AquariusTest.h"
#include "AquariusService.h"
#include "AquariusInstaller.h"	
#include "RegistryOps.h"
#include "Utilities.h"

using namespace std;
using namespace WEX::TestExecution;

AquariusService *service;
PluginMetadata *allPluginsMetadata = nullptr;
int pluginCount=0;

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
	
	GetCurrentDirectory(MAX_PATH, aquariusLocation);
	wcsncat_s(aquariusLocation, MAX_PATH, L"test\\AquariusTest\\AqEventMessage.dll", MAX_PATH);
		
	RegistryOps::PutStringInRegistry(configKey, L"CategoryMessageFile", aquariusLocation);
	RegistryOps::PutStringInRegistry(configKey, L"EventMessageFile", aquariusLocation);
	RegistryOps::PutStringInRegistry(configKey, L"ParameterMessageFile", aquariusLocation);
	RegistryOps::PutDWORDInRegistry(configKey, L"TypesSupported", 0x7);
	RegistryOps::PutDWORDInRegistry(configKey, L"CategoryCount", 0x1);

	delete[] aquariusLocation;
}

void
AquariusTest::TestInstallUninstall()
/*++

Routine Description:
	This function tests installing and uninstalling aquarius service

Arguments:
    None

Return Value:
    None

--*/
{
	AquariusInstaller installer;

	SetupEventLogging();
	
	HKEY configKey = RegistryOps::OpenRegistryKey(HKEY_LOCAL_MACHINE, L"SOFTWARE\\Aquarius");
	VERIFY_IS_NOT_NULL(configKey);
	
	installer.InstallService(
		SERVICE_NAME,               // Name of service
		SERVICE_DISPLAY_NAME,       // Name to display
		SERVICE_START_TYPE,         // Service start type
		SERVICE_DEPENDENCIES,       // Dependencies
		SERVICE_ACCOUNT,            // Service running account
		SERVICE_PASSWORD            // Password of the account
	);   

	installer.UninstallService(SERVICE_NAME);
}

void
AquariusTest::TestInitializeService()
/*++

Routine Description:
	This function tests initializing aquarius service

Arguments:
    None

Return Value:
    None

--*/
{
	wchar_t packageStoreLocation[MAX_PATH]={0};
	DWORD size = 1024;
	service = new AquariusService(SERVICE_NAME);

	if(service->aqPackageManager == nullptr)
	{
		service->aqPackageManager = new AqPackageManager();
		AquariusUtilities::allocator++;
	}

	AquariusUtilities::SetRelativePathFromExe(L"\\..\\AquariusTest\\Plugins");
	service->aqPackageManager->GetPackageStoreLocation(packageStoreLocation, &size);

	if (service->aquariusKernel == nullptr)
	{		
		GetCurrentDirectory(MAX_PATH, packageStoreLocation);
		wcsncat_s(packageStoreLocation, MAX_PATH, L"test\\AquariusTest\\Plugins", MAX_PATH);
		service->aquariusKernel = new AquariusKernel(packageStoreLocation);
		AquariusUtilities::allocator++;
	}

	VERIFY_IS_NOT_NULL(service->aqPackageManager);
	VERIFY_IS_NOT_NULL(service->aquariusKernel);
}

void
AquariusTest::TestGetPluginMetadata()
/*++

Routine Description:
	This function tests populating plugin metadata

Arguments:
    None

Return Value:
    None

--*/
{
	try 
	{
		pluginCount = service->aqPackageManager->RetrievePluginsMetadata(&allPluginsMetadata);
		VERIFY_ARE_EQUAL(pluginCount, 2, L"GetPluginMetaData Passed");
		VERIFY_IS_NOT_NULL(allPluginsMetadata);
	}
	catch(...)
	{
		VERIFY_IS_TRUE(false);
	}
}

void
AquariusTest::TestProcessConfig()
/*++

Routine Description:
	This function tests reading plugins to start from configuration file 

Arguments:
    None

Return Value:
    None

--*/
{
	wchar_t curDir[MAX_PATH];
	
	GetCurrentDirectory(MAX_PATH, curDir);
	wcscat_s(curDir, L"\\test\\AquariusTest\\Plugins");
	
	int ret = service->ProcessConfig(curDir);
	VERIFY_ARE_EQUAL(ret, 0, L"ProcessConfig return success");

	std::set<std::wstring>::iterator it = service->pluginsToStart.find(L"azureworld");
	VERIFY_ARE_NOT_EQUAL(it, service->pluginsToStart.end(), L"ProcessConfig reading plugins Passed");

	it = service->pluginsToStart.find(L"helloworld");
	VERIFY_ARE_EQUAL(it, service->pluginsToStart.end(), L"ProcessConfig reading plugins Passed");
}

void
AquariusTest::TestLoadPlugins()
/*++

Routine Description:
	This function tests loading and starting plugins

Arguments:
    None

Return Value:
    None

--*/
{
	try 
	{
		service->aquariusKernel->LoadPlugins(pluginCount, allPluginsMetadata, service->pluginsToStart);
	}
	catch(...)
	{
		VERIFY_IS_TRUE(false);		
	}
}
bool AquariusTest::TestSetup()
/*++

Routine Description:

    TAEF Setup. This API is called before invoking test APIs. This API sets up test adapter which is consider as primary nic.

Arguments:

    None

Return Value:

    None

--*/
{
    //
    // Initialize tracing.
    //
    return true;
}

bool
AquariusTest::TestCleanup()
/*++

Routine Description:

    TAEF Cleanup. This API is called after invoking test APIs

Arguments:

    None

Return Value:

    None

--*/
{
	return true;
}

AquariusTest::AquariusTest()
/*++

Routine Description:

    This constructor initializes data members of the class as appropriate.

Arguments:

    None.

Return Value:

    None.

--*/
{
}


AquariusTest::~AquariusTest()
/*++

Routine Description:

    Destructor for NetworkClientTest

Arguments:

    None.

Return Value:

    None.

--*/
{

}

int wmain(int argc, wchar_t *argv[])
{
    UNREFERENCED_PARAMETER(argc);
    UNREFERENCED_PARAMETER(argv);
    return 0;
}
