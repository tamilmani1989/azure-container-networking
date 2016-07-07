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

void AqPackageManager::GetPackageStoreLocation(wchar_t *packageStoreLocation, DWORD *size)
{
	AquariusUtilities::WriteEventLogEntry(L"Going to get package store location.");
	AquariusUtilities::WriteEventLogEntry(L"Trying to open registry key that contains config.");
	HKEY configKey = OpenRegistryKey(HKEY_LOCAL_MACHINE, L"SOFTWARE\\Microsoft\\Aquarius\\Config");
	wchar_t szMessage[260];
	if (configKey == nullptr)
	{
		AquariusUtilities::WriteEventLogEntry(L"Registry key does not exist. Returning default package location.");		

		if (GetModuleFileName(nullptr, packageStoreLocation, MAX_PATH) == 0)
		{			
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
				L"Failed to get location of the plugins w/err 0x%08lx Returning %s\n", 
				GetLastError(), AquariusUtilities::defaultPackageStoreLocation);
			AquariusUtilities::WriteEventLogEntry(szMessage);			
			wcscpy(packageStoreLocation, AquariusUtilities::defaultPackageStoreLocation);
			return;
		}
		
		int i = wcslen(packageStoreLocation);
		wchar_t c = '\0';
		do
		{
			c = packageStoreLocation[i - 1];
			packageStoreLocation[i - 1] = packageStoreLocation[i];
			i--;
		} while (c != '\\' && i > 0);
		wcscat(packageStoreLocation, L"\\Plugins");				
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Successfully returned default package location: %s", packageStoreLocation);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		return;
	}
	AquariusUtilities::WriteEventLogEntry(L"Found registry key.");

	GetStringFromRegistry(configKey, L"PackageStoreLocation", packageStoreLocation, size);
}

void AqPackageManager::CopyPluginsMetadata(PluginMetadata **allPluginsMetadata)
{
	(*allPluginsMetadata) = new PluginMetadata[currentPluginCount];
	AquariusUtilities::allocator++;
	for(int i=0; i<currentPluginCount; i++)
	{
		if (packmanCurrentPluginsMetadata[i].location != nullptr) {
			(*allPluginsMetadata)[i].location = new wchar_t[wcslen(packmanCurrentPluginsMetadata[i].location)+1];
			AquariusUtilities::allocator++;
			wcscpy((*allPluginsMetadata)[i].location, packmanCurrentPluginsMetadata[i].location);
		}
		if (packmanCurrentPluginsMetadata[i].currentVersion != nullptr) {
			(*allPluginsMetadata)[i].currentVersion = new wchar_t[wcslen(packmanCurrentPluginsMetadata[i].currentVersion) + 1];
			AquariusUtilities::allocator++;
			wcscpy((*allPluginsMetadata)[i].currentVersion, packmanCurrentPluginsMetadata[i].currentVersion);
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

HKEY AqPackageManager::OpenRegistryKey(HKEY rootKey, wchar_t* subKey)
{
	HKEY returnKey;

	LONG nError = RegOpenKeyEx(rootKey, subKey, NULL, KEY_ALL_ACCESS, &returnKey);
	AquariusUtilities::WriteEventLogEntry(L"Open key operation executed.");
	if (nError == ERROR_FILE_NOT_FOUND)
	{
		AquariusUtilities::WriteEventLogEntry(L"Config key does not exist.");
		nError = RegCreateKeyEx(rootKey, subKey, NULL, nullptr, REG_OPTION_NON_VOLATILE,
			KEY_ALL_ACCESS, nullptr, &returnKey, nullptr);
		AquariusUtilities::WriteEventLogEntry(L"Created config key with default location.");
	}

	if (nError)
	{
		AquariusUtilities::WriteEventLogEntry(L"Some error happnned during config key creation with default location.");		
		returnKey = nullptr;
	}

	return returnKey;
}

void AqPackageManager::GetStringFromRegistry(HKEY configKey, wchar_t* key, wchar_t* returnValue, DWORD* size)
{
	DWORD type = REG_SZ;
	LONG nError = RegQueryValueEx(configKey, key, nullptr, &type, (LPBYTE)returnValue, size);

	if (nError)
	{
		HandleRegistryError(configKey, key, nError);
	}
}

void AqPackageManager::HandleRegistryError(HKEY rootKey, wchar_t* subKey, LONG nError)
{
	wchar_t szMessage[260];
	try
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Reg Open of %s %s failed w/err 0x%08lx", rootKey, subKey, nError);
		AquariusUtilities::WriteErrorLogEntry(szMessage);
	}
	catch (const exception &ex)
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"%s", ex.what());
		AquariusUtilities::WriteErrorLogEntry(szMessage);
	}
	catch (...)
	{
		AquariusUtilities::WriteErrorLogEntry(L"Unknown error happenned while creating registry key");
	}
}

static int GetPluginCount(wchar_t* packageStoreLocation)
{
	HANDLE findHandle = nullptr;
	wchar_t szMessage[260];
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"PackageStore Location in GetPluginCount: %s.",
		packageStoreLocation);
	AquariusUtilities::WriteEventLogEntry(szMessage);

	int pluginCount = 0;
	wchar_t *pluginDirectory = new wchar_t[wcslen(packageStoreLocation) + wcslen(L"\\*") + 1];
	wcscpy(pluginDirectory, packageStoreLocation);
	wcscat(pluginDirectory, L"\\*");
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
	wchar_t *pluginDirectory = new wchar_t[wcslen(packageStoreLocation) + wcslen(L"\\*")];
	AquariusUtilities::allocator++;
	wcscpy(pluginDirectory, packageStoreLocation);
	wcscat(pluginDirectory, L"\\*");
	
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
					packmanCurrentPluginsMetadata[processedCount].location = new wchar_t[
						wcslen(packageStoreLocation) + wcslen(L"\\") + wcslen(findData.cFileName)];
					AquariusUtilities::allocator++;
					wcscpy(packmanCurrentPluginsMetadata[processedCount].location, packageStoreLocation);
					wcscat(packmanCurrentPluginsMetadata[processedCount].location, L"\\");
					wcscat(packmanCurrentPluginsMetadata[processedCount].location, findData.cFileName);
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
	if(packmanCurrentPluginsMetadata != nullptr)
	{
		AquariusUtilities::ReleasePluginMetadata(currentPluginCount, packmanCurrentPluginsMetadata);
		currentPluginCount = 0;
	}	
}