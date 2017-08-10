/****************************** Module Header ******************************\
* Module Name:  PackMan.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#include <iostream>
#include <windows.h>
#include <mutex>
#include "../Shared/PluginMetadata.h"

class AqPackageManager
{
	public:
		void GetPackageStoreLocation(wchar_t *packageStorePath, DWORD *size);
		~AqPackageManager();
		int RetrievePluginsMetadata(PluginMetadata **allPluginsMetadata);
		int GetPluginCount();
	protected:

	private:

		// mutex for package manager objects		
		std::mutex packman_objects_mutex;

		int currentPluginCount = 0;
		PluginMetadata *packmanCurrentPluginsMetadata = nullptr;

		int installedPluginCount = 0;
		PluginMetadata *packmanInstalledPluginsMetadata = nullptr;
		
		void CopyPluginsMetadata(PluginMetadata** allPluginsMetadata);
		void PopulatePluginMetadataFromStore(wchar_t* packageStoreLocation);
};

