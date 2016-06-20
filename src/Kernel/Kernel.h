/****************************** Module Header ******************************\
* Module Name:  Kernel.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#include <iostream>

class PackageDeploymentJob
{
public:

};

class AquariusKernel
{
public:	
	void UpdatePackageStoreLocation(wchar_t *packageStoreLocation);
	wchar_t *GetPackageStoreLocation();
	AquariusKernel(wchar_t *packagerStoreLocation);
	~AquariusKernel();

protected:

private:
	// location of the packages for all teh plugins that Aquarius supports
	wchar_t *packageStoreLocation = nullptr;

	// mutex for kernel objects	
	
	std::_Mutex kernel_objects_mutex;

	// all jobs that we need to process. These could be CRUD operations
	// on modules in the customer's VM.
	PackageDeploymentJob *packageDeploymentJobs = nullptr;

	// whether the kernel is initialized or not
	bool initialized = false;

	void RetrievePackageDeploymentJobs();
};

