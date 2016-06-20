/****************************** Module Header ******************************\
* Module Name:  Kernel.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* This is the kernel of Aquarius. 
* It picks up the package deployment/update tasks and executes them.
*
\***************************************************************************/


#include <stdio.h>
#include <wchar.h>
#include "Kernel.h"

void fun(){}
AquariusKernel::AquariusKernel(wchar_t* storeLocation): packageStoreLocation(nullptr) 
{	
	int len = wcslen(storeLocation);
	packageStoreLocation = new wchar_t[len];
	wcscpy(packageStoreLocation, storeLocation);
	packageDeploymentJobs = nullptr;
	
}

AquariusKernel::~AquariusKernel()
{
	
	kernel_objects_mutex._Lock();
	if (packageStoreLocation != nullptr) 
	{
		delete(packageStoreLocation);
	}
	if(packageDeploymentJobs != nullptr)
	{
		delete(packageDeploymentJobs);
	}
	kernel_objects_mutex._Unlock();
	
}

void AquariusKernel::RetrievePackageDeploymentJobs()
{
	
	kernel_objects_mutex._Lock();

	// get the real job count
	int jobcount = 1; 

	PackageDeploymentJob *jobs = new PackageDeploymentJob[jobcount];	
	
	packageDeploymentJobs = jobs;

	kernel_objects_mutex._Unlock();
	
}

void AquariusKernel::UpdatePackageStoreLocation(wchar_t* storeLocation)
{
	// we need to lock as this got read when we populate package/plugin deployment jobs
	kernel_objects_mutex._Lock();
	int len = wcslen(storeLocation);
	if(packageStoreLocation != nullptr)
	{
		delete(packageStoreLocation);
	}
	packageStoreLocation = new wchar_t[len];
	wcscpy(packageStoreLocation, storeLocation);
	kernel_objects_mutex._Unlock();
	
}

wchar_t* AquariusKernel::GetPackageStoreLocation()
{	
	return packageStoreLocation;
}
