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

AquariusKernel::AquariusKernel(wchar_t* storeLocation): packageStoreLocation(nullptr) 
{	
	int len = wcslen(storeLocation);
	packageStoreLocation = new wchar_t[len];
	AquariusUtilities::allocator++;
	wcscpy(packageStoreLocation, storeLocation);
	pluginsMetadata = nullptr;
	
}

AquariusKernel::~AquariusKernel()
{
	
	kernel_objects_mutex._Lock();
	if (packageStoreLocation != nullptr) 
	{
		delete(packageStoreLocation);
	}
	if(pluginsMetadata != nullptr)
	{
		delete(pluginsMetadata);
	}
	kernel_objects_mutex._Unlock();
	
}

void AquariusKernel::RetrievePackageDeploymentJobs()
{
	
	kernel_objects_mutex._Lock();

	// get the real job count
	int jobcount = 1; 

	//PluginMetadata *jobs = new PluginMetadata[jobcount];
	
	//pluginsMetadata = jobs;

	kernel_objects_mutex._Unlock();
	
}

void AquariusKernel::UpdateCurrentPackageStoreLocation(wchar_t* storeLocation)
{
	// we need to lock as this got read when we populate package/plugin deployment jobs
	kernel_objects_mutex._Lock();
	int len = wcslen(storeLocation);
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
	
}
