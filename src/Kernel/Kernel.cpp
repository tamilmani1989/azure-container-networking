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
#include "Logging.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

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


void AquariusKernel::LoadPlugins(int pluginCount, 
								PluginMetadata *pluginsToLoad, 
								std::set<std::wstring> eligiblePluginSet,
								wchar_t *logFolder)
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
			LOGINFOW(L"Indide load plugins");
			
			for (int i = 0; i < pluginCount; i++)
			{
				wchar_t * codeDirectory = nullptr;
				wchar_t * manifestFile = nullptr;
				wchar_t *lpCommandLine = nullptr;

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
						LOGINFOW(L"Kernel received entrypoint %s in %s", pluginsToLoad[i].entryPoint, codeDirectory);
						std::set<std::wstring>::iterator it = eligiblePluginSet.find(pluginsToLoad[i].name);
						
						if(it == eligiblePluginSet.end())
						{
							LOGINFOW(L"plugin %s not found in eligibe set", pluginsToLoad[i].name);
							SAFE_DELETE(codeDirectory);
							SAFE_DELETE(manifestFile);
							continue;
						}

						if(pluginsToLoad[i].cleanup)
						{
							wchar_t *removeArg =L"-remove";
							size_t len = wcslen(L" ") + wcslen(removeArg) + wcslen(L" ") + wcslen(LOGPARAM) + wcslen(L" ") + wcslen(logFolder) + sizeof(wchar_t);
							
							lpCommandLine = new wchar_t[len];
								
							if(!lpCommandLine)
							{
								LOGERR("Not able to allocate memory for lpCommandLine");
								return;
							}
							
							if(swprintf_s(lpCommandLine, len, L" %s %s %s", removeArg, LOGPARAM, logFolder) < 0)
							{
								LOGINFOW(L"swprintf failed while constructing process arguments");
								return;
							}

							JobManager::PluginCleanup(codeDirectory, lpCommandLine, &pluginsToLoad[i]);		

						}

						else if(pluginsToLoad[i].retryCount < MAX_RETRY_COUNT)
						{		
							if(pluginsToLoad[i].arguments != nullptr)
							{
								size_t len = wcslen(L" ") + wcslen(pluginsToLoad[i].arguments) + wcslen(L" ") 
														+ wcslen(LOGPARAM) + wcslen(L" ") + wcslen(logFolder) + sizeof(wchar_t);
								lpCommandLine = new wchar_t[len];
								
								if(!lpCommandLine)
								{
									LOGERR("Not able to allocate memory for lpCommandLine");
									return;
								}	

								if(swprintf_s(lpCommandLine, len, L" %s %s %s", pluginsToLoad[i].arguments, LOGPARAM, logFolder) < 0)
								{
									LOGINFOW(L"swprintf failed while constructing process arguments");
									return;
								}

							}
							else
							{
								size_t len = wcslen(L" ") + wcslen(LOGPARAM) + wcslen(L" ") + wcslen(logFolder) + sizeof(wchar_t);
								lpCommandLine = new wchar_t[len];
								
								if(!lpCommandLine)
								{
									LOGERR("Not able to allocate memory for lpCommandLine");
									return;
								}	

								if(swprintf_s(lpCommandLine, len, L" %s %s", LOGPARAM, logFolder) < 0)
								{
									LOGINFOW(L"swprintf failed while constructing process arguments");
									return;
								}

							}

							JobManager::EnsureJobIsRunning(codeDirectory, lpCommandLine, &pluginsToLoad[i]);				
						}
						else
						{
							LOGINFOW(L"Retry count %d exceeded for %s", pluginsToLoad[i].retryCount, pluginsToLoad[i].entryPoint);
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
				
				if(lpCommandLine)
				{
					delete lpCommandLine;
					lpCommandLine = nullptr;
				}
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
