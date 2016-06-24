/****************************** Module Header ******************************\
* Module Name:  Kernel.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#include <iostream>
#include "../Shared/PackageDeploymentJob.h"

class AquariusKernel
{
public:	
	void UpdateCurrentPackageStoreLocation(wchar_t *packageStoreLocation);
	wchar_t *GetCurrentPackageStoreLocation();
	AquariusKernel(wchar_t *packagerStoreLocation);
	~AquariusKernel();

protected:

private:
	// location of the packages for all the plugins that Aquarius supports
	wchar_t *packageStoreLocation = nullptr;

	// all jobs that we need to process. These could be CRUD operations
	// on modules in the customer's VM.
	PackageDeploymentJob *packageDeploymentJobs = nullptr;

	// mutex for kernel objects	
	
	std::_Mutex kernel_objects_mutex;

	// whether the kernel is initialized or not
	bool initialized = false;

	void RetrievePackageDeploymentJobs();
};

