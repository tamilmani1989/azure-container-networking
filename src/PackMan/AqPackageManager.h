/****************************** Module Header ******************************\
* Module Name:  PackMan.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#include <iostream>
#include <windows.h>
#include "../Shared/PluginMetadata.h"

class AqPackageManager
{
	public:
		void GetPackageStoreLocation(wchar_t *packageStorePath, DWORD *size);
		~AqPackageManager();
		int RetrievePluginsMetadata(PluginMetadata **allPluginsMetadata);
	protected:

	private:
		const static wchar_t *defaultPackageStoreLocation;
		const static wchar_t *codeDirectoryName;
		const static wchar_t *configFileName;
		const static wchar_t *manifestFileName;

		// mutex for package manager objects		
		std::_Mutex packman_objects_mutex;

		int currentPluginCount = 0;
		PluginMetadata *packmanCurrentPluginsMetadata = nullptr;

		int installedPluginCount = 0;
		PluginMetadata *packmanInstalledPluginsMetadata = nullptr;

		static HKEY OpenRegistryKey(HKEY hRootKey, wchar_t* strKey);
		static void GetStringFromRegistry(HKEY configKey, wchar_t* key, wchar_t *returnValue, DWORD* size);
		static void HandleRegistryError(HKEY rootKey, wchar_t* subKey, LONG nError);
		void CopyPluginsMetadata(PluginMetadata** allPluginsMetadata);
		void PopulatePluginMetadataFromStore(wchar_t* packageStoreLocation);
};

