/****************************** Module Header ******************************\
* Module Name:  RegistryOps.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
*
\***************************************************************************/

#include "RegistryOps.h"
#include "Utilities.h"

HKEY RegistryOps::OpenRegistryKey(HKEY rootKey, wchar_t* subKey)
{
    HKEY returnKey;
    wchar_t szMessage[260];

    LONG nError = RegOpenKeyEx(rootKey, subKey, NULL, KEY_ALL_ACCESS, &returnKey);
    AquariusUtilities::WriteEventLogEntry(L"Open key operation executed.");
    if (nError == ERROR_FILE_NOT_FOUND)
    {
        AquariusUtilities::WriteEventLogEntry(L"Config key does not exist.");
        nError = RegCreateKeyEx(rootKey, subKey, NULL, nullptr, REG_OPTION_NON_VOLATILE,
            KEY_ALL_ACCESS, nullptr, &returnKey, nullptr);
        AquariusUtilities::WriteEventLogEntry(L"Created config key with default location.");
    }

    if (nError != ERROR_SUCCESS)
    {

        StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
            L"JErro happenned during confg key creation %s. Error: 0x%08lx", subKey, nError);
        AquariusUtilities::WriteEventLogEntry(szMessage);
        returnKey = nullptr;
    }

    return returnKey;
}

void RegistryOps::PutStringInRegistry(HKEY configKey, wchar_t* key, wchar_t* value)
{
	DWORD type = REG_SZ;
	LONG nError = RegSetKeyValue(configKey, nullptr, key, type, value, (wcsnlen(value, MAX_PATH) + 1) * sizeof(wchar_t));
	if (nError)
	{
		HandleRegistryError(configKey, key, nError);
	}
}
void RegistryOps::PutDWORDInRegistry(HKEY configKey, wchar_t* key, DWORD value)
{
	DWORD type = REG_DWORD;
	LONG nError = RegSetKeyValue(configKey, nullptr, key, type, (const BYTE *)&value, sizeof(DWORD));
	if (nError)
	{
		HandleRegistryError(configKey, key, nError);
	}
}

void RegistryOps::GetStringFromRegistry(HKEY configKey, wchar_t* key, wchar_t* returnValue, DWORD* size)
{
    DWORD type = REG_SZ;
    LONG nError = RegQueryValueEx(configKey, key, nullptr, &type, (LPBYTE)returnValue, size);

    if (nError)
    {
        HandleRegistryError(configKey, key, nError);
    }
}

void RegistryOps::HandleRegistryError(HKEY rootKey, wchar_t* subKey, LONG nError)
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