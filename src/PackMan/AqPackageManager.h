/****************************** Module Header ******************************\
* Module Name:  PackMan.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#include <windows.h>

class AqPackageManager
{
	public:
		void GetPackageStoreLocation(wchar_t *packageStorePath, DWORD *size);

	protected:

	private:
		const static wchar_t *defaultPackageStoreLocation;
		HKEY OpenRegistryKey(HKEY hRootKey, wchar_t* strKey);
		void GetStringFromRegistry(HKEY configKey, wchar_t* key, wchar_t *returnValue, DWORD* size);
		void HandleRegistryError(HKEY rootKey, wchar_t* subKey, LONG nError);
};

