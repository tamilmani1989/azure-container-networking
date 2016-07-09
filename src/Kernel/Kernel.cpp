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
	int len = wcsnlen(storeLocation, MAX_PATH);
    packageStoreLocation = new wchar_t[len];
	AquariusUtilities::allocator++;
	wcscpy(packageStoreLocation, storeLocation);
	alreadyLoadedPluginsMetadata = nullptr;
	
}

AquariusKernel::~AquariusKernel()
{	
	kernel_objects_mutex._Lock();
	if (packageStoreLocation != nullptr) 
	{
		delete(packageStoreLocation);
	}
	if(alreadyLoadedPluginsMetadata != nullptr)
	{
		delete(alreadyLoadedPluginsMetadata);
	}
	JobManager::ReleaseAllJobs();
	kernel_objects_mutex._Unlock();
	
}

void AquariusKernel::RetrievePackageDeploymentJobs()
{
	
	kernel_objects_mutex._Lock();

	// get the real job count
	//int jobcount = 1; 

	//PluginMetadata *jobs = new PluginMetadata[jobcount];
	
	//pluginsMetadata = jobs;

	kernel_objects_mutex._Unlock();
	
}

void AquariusKernel::UpdateCurrentPackageStoreLocation(wchar_t* storeLocation)
{
	// we need to lock as this got read when we populate package/plugin deployment jobs
	kernel_objects_mutex._Lock();
	int len = wcsnlen(storeLocation, MAX_PATH);
    if(packageStoreLocation != nullptr)
	{
		delete[] packageStoreLocation;
		AquariusUtilities::allocator--;
	}
	packageStoreLocation = new wchar_t[len];
	AquariusUtilities::allocator++;
	wcscpy(packageStoreLocation, storeLocation);
	kernel_objects_mutex._Unlock();	
}

wchar_t* AquariusKernel::GetCurrentPackageStoreLocation()
{	
	return packageStoreLocation;
}

void AquariusKernel::LoadPlugins(int pluginCount, PluginMetadata *pluginsToLoad)
{
	kernel_objects_mutex._Lock();
	try {
		if (alreadyLoadedPluginCount <= 0)
		{
			for (int i = 0; i < pluginCount; i++)
			{
				wchar_t * codeDirectory = nullptr;
				wchar_t * manifestFile = nullptr;
				wchar_t *entryPoint = nullptr;
				try
				{
					int locationLen = wcsnlen(pluginsToLoad[i].location, MAX_PATH);
                    codeDirectory = new wchar_t[locationLen + wcsnlen(L"\\", 2) + wcsnlen(AquariusUtilities::codeDirectoryName, MAX_PATH) + 1];
                    wcscpy(codeDirectory, pluginsToLoad[i].location);
					wcscat(codeDirectory, L"\\");
					wcscat(codeDirectory, AquariusUtilities::codeDirectoryName);

					manifestFile = new wchar_t[locationLen + wcsnlen(L"\\", 2) + wcsnlen(AquariusUtilities::manifestFileName, MAX_PATH)];
                    wcscpy(manifestFile, pluginsToLoad[i].location);
					wcscat(manifestFile, L"\\");
					wcscat(manifestFile, AquariusUtilities::manifestFileName);

					//CreateJobObject(nullptr, )
					
					XmlParser::ParseManifest(manifestFile, &entryPoint);
					if (entryPoint != nullptr)
					{
						wchar_t szMessage[260];
						StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
							L"Kernel received entrypoint %s in %s",
							entryPoint, codeDirectory);
						AquariusUtilities::WriteEventLogEntry(szMessage);
						JobManager::EnsureJobIsRunning(codeDirectory, entryPoint);						
					}
				}
				catch (...)
				{
					SAFE_DELETE(codeDirectory);
					SAFE_DELETE(manifestFile);
					SAFE_DELETE(entryPoint);
					throw;
				}
				SAFE_DELETE(codeDirectory);
				SAFE_DELETE(manifestFile);
				SAFE_DELETE(entryPoint);
			}
		}
		else
		{
			// we may have to upgrade or remove some of the loaded plugins
		}
	}
	catch(...)
	{
		kernel_objects_mutex._Unlock();
		throw;
	}
	kernel_objects_mutex._Unlock();
}
