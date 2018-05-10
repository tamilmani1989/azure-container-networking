/****************************** Module Header ******************************\
* Module Name:  RegistryOps.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#include <iostream>
#include <windows.h>

#pragma once
class RegistryOps 
{
public: 
    static HKEY CreateOrOpenRegistryKey(HKEY hRootKey, wchar_t* strKey);
    static HKEY OpenRegistryKey(HKEY hRootKey, wchar_t* strKey);
    static LONG CloseRegistryKey(HKEY hRootKey);
    static LONG GetStringFromRegistry(HKEY configKey, wchar_t* key, wchar_t *returnValue, DWORD* size, bool handleRegistryError);
    static void HandleRegistryError(HKEY rootKey, wchar_t* subKey, LONG nError);
    static void PutStringInRegistry(HKEY configKey, wchar_t* key, wchar_t* value);
	static void PutDWORDInRegistry(HKEY configKey, wchar_t* key, DWORD value);
};
