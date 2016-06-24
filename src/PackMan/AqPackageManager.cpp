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

const wchar_t* AqPackageManager::defaultPackageStoreLocation = L"C:\\ProgramData\\Microsoft\\Aquarius\\Plugins\\";

void AqPackageManager::GetPackageStoreLocation(wchar_t *packageStoreLocation, DWORD *size)
{
	AquariusUtilities::WriteEventLogEntry(L"Going to get package store location.", EVENTLOG_INFORMATION_TYPE);
	AquariusUtilities::WriteEventLogEntry(L"Trying to open registry key that contains config.", EVENTLOG_INFORMATION_TYPE);
	HKEY configKey = OpenRegistryKey(HKEY_LOCAL_MACHINE, L"SOFTWARE\\Microsoft\\Aquarius\\Config");
	if (configKey == nullptr)
	{
		AquariusUtilities::WriteEventLogEntry(L"Registry key does not exist. Returning default package location.",
			EVENTLOG_INFORMATION_TYPE);		
		wcscpy(packageStoreLocation, defaultPackageStoreLocation);
		wchar_t szMessage[260];
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Successfully returned default package location: %s", packageStoreLocation);
		AquariusUtilities::WriteEventLogEntry(szMessage, EVENTLOG_INFORMATION_TYPE);
		return;
	}
	AquariusUtilities::WriteEventLogEntry(L"Found registry key.", EVENTLOG_INFORMATION_TYPE);

	GetStringFromRegistry(configKey, L"PackageStoreLocation", packageStoreLocation, size);
}

HKEY AqPackageManager::OpenRegistryKey(HKEY rootKey, wchar_t* subKey)
{
	HKEY returnKey;

	LONG nError = RegOpenKeyEx(rootKey, subKey, NULL, KEY_ALL_ACCESS, &returnKey);
	AquariusUtilities::WriteEventLogEntry(L"Open key operation executed.", EVENTLOG_INFORMATION_TYPE);
	if (nError == ERROR_FILE_NOT_FOUND)
	{
		AquariusUtilities::WriteEventLogEntry(L"Config key does not exist.", EVENTLOG_INFORMATION_TYPE);
		nError = RegCreateKeyEx(rootKey, subKey, NULL, nullptr, REG_OPTION_NON_VOLATILE,
			KEY_ALL_ACCESS, nullptr, &returnKey, nullptr);
		AquariusUtilities::WriteEventLogEntry(L"Created config key with default location.", EVENTLOG_INFORMATION_TYPE);
	}

	if (nError)
	{
		AquariusUtilities::WriteEventLogEntry(L"Some error happnned during config key creation with default location.",
			EVENTLOG_INFORMATION_TYPE);
		//HandleRegistryError(rootKey, subKey, nError);
		//WriteEventLogEntry(L"Logged error.", EVENTLOG_INFORMATION_TYPE);
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
		AquariusUtilities::WriteEventLogEntry(szMessage, EVENTLOG_ERROR_TYPE);
	}
	catch (const exception &ex)
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"%s", ex.what());
		AquariusUtilities::WriteErrorLogEntry(szMessage, EVENTLOG_ERROR_TYPE);
	}
	catch (...)
	{
		AquariusUtilities::WriteErrorLogEntry(L"Unknown error happenned while crating registry key", EVENTLOG_ERROR_TYPE);
	}
}