/****************************** Module Header ******************************\
* Module Name:  Kernel.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#include <iostream>
#include "../Shared/PluginMetadata.h"

class AquariusKernel
{
public:	
	AquariusKernel(wchar_t *packagerStoreLocation);
	~AquariusKernel();
	void UpdateCurrentPackageStoreLocation(wchar_t *packageStoreLocation);
	wchar_t *GetCurrentPackageStoreLocation();
	void LoadPlugins(int pluginCount, PluginMetadata *pluginsToLoad);

protected:

private:
	// mutex for kernel objects		
	std::_Mutex kernel_objects_mutex;

	// location of the packages for all the plugins that Aquarius supports
	wchar_t *packageStoreLocation = nullptr;

	// all jobs that we need to process. These could be CRUD operations
	// on modules in the customer's VM.
	int alreadyLoadedPluginCount = 0;
	PluginMetadata *alreadyLoadedPluginsMetadata = nullptr;

	// whether the kernel is initialized or not
	bool initialized = false;

	void RetrievePackageDeploymentJobs();
};

