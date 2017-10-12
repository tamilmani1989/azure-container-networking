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
#include "../Logging/Logging.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

void AqPackageManager::GetPackageStoreLocation(wchar_t *packageStoreLocation, DWORD *size)
{
	LOGINFOW(L"Going to get package store location.");
	LOGINFOW(L"Trying to open registry key that contains config.");
	HKEY configKey = nullptr;
	//RegistryOps::OpenRegistryKey(HKEY_LOCAL_MACHINE, L"SOFTWARE\\Aquarius");
    if(configKey != nullptr)
    {
        LOGINFOW(L"Found registry key.");
        RegistryOps::GetStringFromRegistry(configKey, L"PackageStoreLocation", packageStoreLocation, size);
        if(wcsnlen(packageStoreLocation, MAX_PATH)>0)
        {
            return;
        }
    }  
	
	LOGINFOW(L"Registry key does not exist. Returning default package location.");

	AquariusUtilities::PopulatePluginLocation(packageStoreLocation);
	LOGINFOW(L"Successfully returned default package location: %s", packageStoreLocation);
}

void AqPackageManager::CopyPluginsMetadata(PluginMetadata **allPluginsMetadata)
/*++

Routine Description:
	This function initializes and populates allPluginsMetadata object

Arguments:

Return Value:
	N/A

--*/
{
	*allPluginsMetadata = GetPluginInstances(currentPluginCount);
	
	if (!*allPluginsMetadata)
	{
		LOGINFOW(L"Getting plugin metadata failed");
		return;
	}

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
	LOGINFOW(L"Going to populate metadata for plugins.");
	int pluginCount;
	packman_objects_mutex.lock();
	try 
	{
		wchar_t packageStoreLocation[MAX_PATH];
		DWORD size = MAX_PATH;
		GetPackageStoreLocation(packageStoreLocation, &size);
		PopulatePluginMetadataFromStore(packageStoreLocation);
		CopyPluginsMetadata(allPluginsMetadata);
		pluginCount = currentPluginCount;
		LOGINFOW(L"Successfully populated plugins metadata");
	}
	catch(...)
	{
		packman_objects_mutex.unlock(); // can't unlock in destructor
		throw;
	}
	packman_objects_mutex.unlock();
	return pluginCount;
}
int AqPackageManager::GetPluginCount()
{
	return currentPluginCount;
}

static int getPluginCount(wchar_t* packageStoreLocation)
{
	HANDLE findHandle = nullptr;
	
	LOGINFOW(L"PackageStore Location in GetPluginCount: %s.",
		packageStoreLocation);

	int pluginCount = 0;
	size_t len = wcsnlen(packageStoreLocation, MAX_PATH) + wcsnlen(L"\\*", 2) + 1;
	wchar_t *pluginDirectory = new wchar_t[len];
    wcscpy_s(pluginDirectory, len, packageStoreLocation);
	wcscat_s(pluginDirectory, len, L"\\*");
	try {
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
					LOGINFOW(L"Obtained plugin directory: %s.", findData.cFileName);
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
	HANDLE findHandle = nullptr;

	LOGINFOW(L"PackageStore Location in PopulatePluginMetadataFromStore: %s.",
		packageStoreLocation);

	size_t len = wcsnlen(packageStoreLocation, MAX_PATH) + wcsnlen(L"\\*", 2) + 1;
	wchar_t *pluginDirectory = new wchar_t[len];
    AquariusUtilities::allocator++;
	wcscpy_s(pluginDirectory, len, packageStoreLocation);
	wcscat_s(pluginDirectory, len, L"\\*");
	
	try 
	{
		int pluginCount = getPluginCount(packageStoreLocation);
		LOGINFOW(L"Received plugin count as: %d",
			pluginCount);

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
			LOGINFOW(L"Going to process directory: %s",
				findData.cFileName);

			// only consider directories, we do not expect any files in the plugins directory.
			if (findData.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY)
			{
				if (wcscmp(findData.cFileName, L".") != 0 &&
					wcscmp(findData.cFileName, L"..") != 0)
				{
					LOGINFOW(L"Going to populate information for plugin: %s.", findData.cFileName);
					size_t len = wcsnlen(packageStoreLocation, MAX_PATH) + wcsnlen(L"\\", 2) + wcsnlen(findData.cFileName, MAX_PATH) + 1;
					packmanCurrentPluginsMetadata[processedCount].location = new wchar_t[len];                        
					AquariusUtilities::allocator++;
					wcscpy_s(packmanCurrentPluginsMetadata[processedCount].location, len, packageStoreLocation);
					wcscat_s(packmanCurrentPluginsMetadata[processedCount].location, len, L"\\");
					wcscat_s(packmanCurrentPluginsMetadata[processedCount].location, len, findData.cFileName);
					LOGINFOW(L"Obtained plugin path: %s", 
						packmanCurrentPluginsMetadata[processedCount].location);
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
	if(packmanCurrentPluginsMetadata != nullptr)
	{
        LOGINFOW(L"Going to Release plugin metadata.. plugin count=%d", currentPluginCount);
		AquariusUtilities::ReleasePluginMetadata(currentPluginCount, packmanCurrentPluginsMetadata);
		currentPluginCount = 0;
	}	
}