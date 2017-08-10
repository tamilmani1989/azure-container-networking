/****************************** Module Header ******************************\
* Module Name:  Kernel.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* This is the kernel of Aquarius. 
* It picks up the package deployment/update tasks and executes them.
*
\***************************************************************************/

#include "Kernel.h"
#include "../Shared/Utilities.h"
#include "../Shared/XmlParser.h"
#include "JobManager.h"

AquariusKernel::AquariusKernel(wchar_t* storeLocation): packageStoreLocation(nullptr) 
{	
	size_t len = wcsnlen(storeLocation, MAX_PATH) + 1;
    packageStoreLocation = new wchar_t[len];
	AquariusUtilities::allocator++;
	wcscpy_s(packageStoreLocation, len, storeLocation);
	alreadyLoadedPluginsMetadata = nullptr;
}

AquariusKernel::~AquariusKernel()
{	
	kernel_objects_mutex.lock();
	if (packageStoreLocation != nullptr) 
	{
		delete(packageStoreLocation);
	}
	if(alreadyLoadedPluginsMetadata != nullptr)
	{
		delete(alreadyLoadedPluginsMetadata);
	}
	JobManager::ReleaseAllJobs();
	kernel_objects_mutex.unlock();
	
}

void AquariusKernel::UpdateCurrentPackageStoreLocation(wchar_t* storeLocation)
{
	// we need to lock as this got read when we populate package/plugin deployment jobs
	kernel_objects_mutex.lock();
	size_t len = wcsnlen(storeLocation, MAX_PATH) + 1;
    if(packageStoreLocation != nullptr)
	{
		delete[] packageStoreLocation;
		AquariusUtilities::allocator--;
	}
	packageStoreLocation = new wchar_t[len];
	AquariusUtilities::allocator++;
	wcscpy_s(packageStoreLocation, len, storeLocation);
	kernel_objects_mutex.unlock();	
}

wchar_t* AquariusKernel::GetCurrentPackageStoreLocation()
{	
	return packageStoreLocation;
}


void AquariusKernel::LoadPlugins(int pluginCount, PluginMetadata *pluginsToLoad)
/*++

Routine Description:
	This function parses the manifest file and load all plugins.

Arguments:
	pluginCount - number of plugins to load
	pluginsToLoad - object contains info about plugin

Return Value:

--*/
{
	kernel_objects_mutex.lock();
	
	try {
		if (alreadyLoadedPluginCount <= 0)
		{
			AquariusUtilities::WriteEventLogEntry(L"Indide load plugins");
			
			for (int i = 0; i < pluginCount; i++)
			{
				wchar_t * codeDirectory = nullptr;
				wchar_t * manifestFile = nullptr;

				try
				{
					size_t locationLen = wcsnlen(pluginsToLoad[i].location, MAX_PATH);
					size_t len = locationLen + wcsnlen(L"\\", 2) + wcsnlen(AquariusUtilities::codeDirectoryName, MAX_PATH) + 1;
                    codeDirectory = new wchar_t[len];
                    wcscpy_s(codeDirectory, len, pluginsToLoad[i].location);
					wcscat_s(codeDirectory, len, L"\\");
					wcscat_s(codeDirectory, len, AquariusUtilities::codeDirectoryName);

					len = locationLen + wcsnlen(L"\\", 2) + wcsnlen(AquariusUtilities::manifestFileName, MAX_PATH) + 1;
					manifestFile = new wchar_t[len];
                    wcscpy_s(manifestFile, len, pluginsToLoad[i].location);
					wcscat_s(manifestFile, len, L"\\");
					wcscat_s(manifestFile, len, AquariusUtilities::manifestFileName);
										
					XmlParser::ParseManifest(manifestFile,&pluginsToLoad[i]);
					
					if (pluginsToLoad[i].entryPoint != nullptr)
					{
						wchar_t szMessage[MAX_PATH];
						StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
							L"Kernel received entrypoint %s in %s",
							pluginsToLoad[i].entryPoint, codeDirectory);
						AquariusUtilities::WriteEventLogEntry(szMessage);
			
						if(pluginsToLoad[i].retryCount < MAX_RETRY_COUNT)
						{
							JobManager::EnsureJobIsRunning(codeDirectory, nullptr, &pluginsToLoad[i]);				
						}
						else
						{
							StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
								L"Retry count exceeded for %s",pluginsToLoad[i].entryPoint);
							AquariusUtilities::WriteEventLogEntry(szMessage);
							throw;
						}
					}
				}
				catch (...)
				{
					SAFE_DELETE(codeDirectory);
					SAFE_DELETE(manifestFile);
					pluginsToLoad[i].exitStatus=1;
					pluginsToLoad[i].retryCount++;					
					throw;
				}

				pluginsToLoad[i].retryCount=0;
				SAFE_DELETE(codeDirectory);
				SAFE_DELETE(manifestFile);
			}
		}
		else
		{
			// we may have to upgrade or remove some of the loaded plugins
		}
	}
	catch(...)
	{
		kernel_objects_mutex.unlock();
		throw;
	}
	
	kernel_objects_mutex.unlock();
}
