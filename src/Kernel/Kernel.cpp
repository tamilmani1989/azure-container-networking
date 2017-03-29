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

void AquariusKernel::RetrievePackageDeploymentJobs()
{
	
	kernel_objects_mutex.lock();

	// get the real job count
	//int jobcount = 1; 

	//PluginMetadata *jobs = new PluginMetadata[jobcount];
	
	//pluginsMetadata = jobs;

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
{
	kernel_objects_mutex.lock();
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
										
					XmlParser::ParseManifest(manifestFile, &entryPoint);
					if (entryPoint != nullptr)
					{
						wchar_t szMessage[260];
						StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
							L"Kernel received entrypoint %s in %s",
							entryPoint, codeDirectory);
						AquariusUtilities::WriteEventLogEntry(szMessage);
						JobManager::EnsureJobIsRunning(codeDirectory, entryPoint, nullptr);				
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
		kernel_objects_mutex.unlock();
		throw;
	}
	kernel_objects_mutex.unlock();
}
