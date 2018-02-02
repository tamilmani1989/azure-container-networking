/****************************** Module Header ******************************\
* Module Name:  Kernel.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#include <iostream>
#include "../Shared/PluginMetadata.h"
#include<mutex>
#include<set>
#define LOGPARAM L"-log"

class AquariusKernel
{
public:	
    AquariusKernel(wchar_t *packagerStoreLocation);
    ~AquariusKernel();
    void UpdateCurrentPackageStoreLocation(wchar_t *packageStoreLocation);
    wchar_t *GetCurrentPackageStoreLocation();
    void LoadPlugins(int pluginCount, PluginMetadata *pluginsToLoad);
    void MonitorPlugins(int pluginCount, PluginMetadata *pluginsToLoad, std::set<std::wstring> eligiblePluginSet, wchar_t *logFolder);

protected:

private:
    // mutex for kernel objects		
    std::mutex kernel_objects_mutex;

    // location of the packages for all the plugins that Aquarius supports
    wchar_t *packageStoreLocation = nullptr;

    // whether the kernel is initialized or not
    bool initialized = false;
};

