/****************************** Module Header ******************************\
* Module Name:  PackMan.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* This is the kernel of Aquarius. 
* It picks up the package deployment/update tasks and executes them.
*
\***************************************************************************/

#include "AqPackageManager.h"
#include <strsafe.h>
#include <iostream>
#include "../Shared/Utilities.h"
#include "../Shared/RegistryOps.h"


void AqPackageManager::GetPackageStoreLocation(wchar_t *packageStoreLocation, DWORD *size)
{
	AquariusUtilities::WriteEventLogEntry(L"Going to get package store location.");
	AquariusUtilities::WriteEventLogEntry(L"Trying to open registry key that contains config.");
	HKEY configKey = nullptr;
	//RegistryOps::OpenRegistryKey(HKEY_LOCAL_MACHINE, L"SOFTWARE\\Aquarius");
    if(configKey != nullptr)
    {
        AquariusUtilities::WriteEventLogEntry(L"Found registry key.");
        RegistryOps::GetStringFromRegistry(configKey, L"PackageStoreLocation", packageStoreLocation, size);
        if(wcsnlen(packageStoreLocation, MAX_PATH)>0)
        {
            return;
        }
    }
	wchar_t szMessage[260];    
	
	AquariusUtilities::WriteEventLogEntry(L"Registry key does not exist. Returning default package location.");

	AquariusUtilities::PopulatePluginLocation(packageStoreLocation);
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Successfully returned default package location: %s", packageStoreLocation);
	AquariusUtilities::WriteEventLogEntry(szMessage);	
}

void AqPackageManager::CopyPluginsMetadata(PluginMetadata **allPluginsMetadata)
{
	(*allPluginsMetadata) = new PluginMetadata[currentPluginCount];
	AquariusUtilities::allocator++;
	for(int i=0; i<currentPluginCount; i++)
	{
		if (packmanCurrentPluginsMetadata[i].location != nullptr) {
			size_t len = wcsnlen(packmanCurrentPluginsMetadata[i].location, MAX_PATH) + 1;
			(*allPluginsMetadata)[i].location = new wchar_t[len];
            AquariusUtilities::allocator++;
			wcscpy_s((*allPluginsMetadata)[i].location, len, packmanCurrentPluginsMetadata[i].location);
		}
		if (packmanCurrentPluginsMetadata[i].currentVersion != nullptr) {
			size_t len = wcsnlen(packmanCurrentPluginsMetadata[i].currentVersion, MAX_PATH) + 1;
			(*allPluginsMetadata)[i].currentVersion = new wchar_t[len];
            AquariusUtilities::allocator++;
			wcscpy_s((*allPluginsMetadata)[i].currentVersion, len, packmanCurrentPluginsMetadata[i].currentVersion);
		}
	}
}

int AqPackageManager::RetrievePluginsMetadata(PluginMetadata **allPluginsMetadata)
{
	AquariusUtilities::WriteEventLogEntry(L"Going to populate metadata for plugins.");
	int pluginCount;
	packman_objects_mutex._Lock();
	try 
	{
		wchar_t packageStoreLocation[MAX_PATH];
		DWORD size = MAX_PATH;
		GetPackageStoreLocation(packageStoreLocation, &size);
		PopulatePluginMetadataFromStore(packageStoreLocation);
		CopyPluginsMetadata(allPluginsMetadata);
		pluginCount = currentPluginCount;
		AquariusUtilities::WriteEventLogEntry(L"Successfully populated plugins metadata");
	}
	catch(...)
	{
		packman_objects_mutex._Unlock(); // can't unlock in destructor
		throw;
	}
	packman_objects_mutex._Unlock();
	return pluginCount;
}


static int GetPluginCount(wchar_t* packageStoreLocation)
{
	HANDLE findHandle = nullptr;
	wchar_t szMessage[260];
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"PackageStore Location in GetPluginCount: %s.",
		packageStoreLocation);
	AquariusUtilities::WriteEventLogEntry(szMessage);

	int pluginCount = 0;
	size_t len = wcsnlen(packageStoreLocation, MAX_PATH) + wcsnlen(L"\\*", 2) + 1;
	wchar_t *pluginDirectory = new wchar_t[len];
    wcscpy_s(pluginDirectory, len, packageStoreLocation);
	wcscat_s(pluginDirectory, len, L"\\*");
	try {
		AquariusUtilities::WriteEventLogEntry((PWSTR)(pluginDirectory));
		WIN32_FIND_DATA findData;
		findHandle = FindFirstFile(pluginDirectory, &findData);
		if (findHandle == INVALID_HANDLE_VALUE)
		{
			throw "INVALID handle retrieved for the pluginDirectory during RetrievePluginsMetadata";
		}

		do
		{
			// only look at directories, we do not expect any files in the plugins directory.
			if (findData.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY)
			{
				if (wcscmp(findData.cFileName, L".") != 0 &&
					wcscmp(findData.cFileName, L"..") != 0)
				{
					StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Obtained plugin directory: %s.", findData.cFileName);
					AquariusUtilities::WriteEventLogEntry(szMessage);
					pluginCount++;
				}
			}

		} while (FindNextFile(findHandle, &findData));
	}
	catch(...)
	{
		if(findHandle != nullptr && findHandle != INVALID_HANDLE_VALUE)
		{
			FindClose(findHandle);
		}
		delete [] pluginDirectory;	
		throw;
	}

	FindClose(findHandle);
	delete[] pluginDirectory;	
	return pluginCount;
}


void AqPackageManager::PopulatePluginMetadataFromStore(wchar_t* packageStoreLocation)
{
	wchar_t szMessage[260];
	HANDLE findHandle = nullptr;

	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
		L"PackageStore Location in PopulatePluginMetadataFromStore: %s.",
		packageStoreLocation);
	AquariusUtilities::WriteEventLogEntry(szMessage);
	size_t len = wcsnlen(packageStoreLocation, MAX_PATH) + wcsnlen(L"\\*", 2) + 1;
	wchar_t *pluginDirectory = new wchar_t[len];
    AquariusUtilities::allocator++;
	wcscpy_s(pluginDirectory, len, packageStoreLocation);
	wcscat_s(pluginDirectory, len, L"\\*");
	
	try 
	{
		AquariusUtilities::WriteEventLogEntry((PWSTR)(pluginDirectory));
		int pluginCount = GetPluginCount(packageStoreLocation);
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Received plugin count as: %d",
			pluginCount);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		if (packmanCurrentPluginsMetadata != nullptr) {
			AquariusUtilities::ReleasePluginMetadata(currentPluginCount, packmanCurrentPluginsMetadata);
		}
		packmanCurrentPluginsMetadata = new PluginMetadata[pluginCount];
		currentPluginCount = pluginCount;
		AquariusUtilities::allocator++;
		// plugin count can change after we obtain the count, but thats okay
		WIN32_FIND_DATA findData;
		findHandle = FindFirstFile(pluginDirectory, &findData);
		if (findHandle == INVALID_HANDLE_VALUE)
		{
			throw "INVALID handle retrieved for the pluginDirectory during RetrievePluginsMetadata";
		}

		int processedCount = 0;
		do
		{
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
				L"Going to process directory: %s",
				findData.cFileName);
			AquariusUtilities::WriteEventLogEntry(szMessage);

			// only consider directories, we do not expect any files in the plugins directory.
			if (findData.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY)
			{
				if (wcscmp(findData.cFileName, L".") != 0 &&
					wcscmp(findData.cFileName, L"..") != 0)
				{
					StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
						L"Going to populate information for plugin: %s.", findData.cFileName);
					AquariusUtilities::WriteEventLogEntry(szMessage);
					size_t len = wcsnlen(packageStoreLocation, MAX_PATH) + wcsnlen(L"\\", 2) + wcsnlen(findData.cFileName, MAX_PATH) + 1;
					packmanCurrentPluginsMetadata[processedCount].location = new wchar_t[len];                        
					AquariusUtilities::allocator++;
					wcscpy_s(packmanCurrentPluginsMetadata[processedCount].location, len, packageStoreLocation);
					wcscat_s(packmanCurrentPluginsMetadata[processedCount].location, len, L"\\");
					wcscat_s(packmanCurrentPluginsMetadata[processedCount].location, len, findData.cFileName);
					StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Obtained plugin path: %s", 
						packmanCurrentPluginsMetadata[processedCount].location);
					AquariusUtilities::WriteEventLogEntry(szMessage);			
					processedCount++;
				}
			}

			if (processedCount >= pluginCount)
			{
				// we may find more plugins, but we cannot process mroe in this iteration
				// remaining will be picked up during next iteration.
				break;
			}
			
		} while (FindNextFile(findHandle, &findData));
	}
	catch(...)
	{
		if (findHandle != INVALID_HANDLE_VALUE && findHandle != nullptr) 
		{
			FindClose(findHandle);
		}
		delete[] pluginDirectory;
		AquariusUtilities::allocator--;		
		throw;
	}
	
	FindClose(findHandle);
	delete[] pluginDirectory;
	AquariusUtilities::allocator--;	
}

AqPackageManager::~AqPackageManager()
{
    wchar_t szMessage[260];
	if(packmanCurrentPluginsMetadata != nullptr)
	{
        StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
            L"Going to Release plugin metadata.. plugin count=%d", currentPluginCount);
        AquariusUtilities::WriteEventLogEntry(szMessage);
		AquariusUtilities::ReleasePluginMetadata(currentPluginCount, packmanCurrentPluginsMetadata);
		currentPluginCount = 0;
	}	
}