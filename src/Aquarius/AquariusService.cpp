/****************************** Module Header ******************************\
* Module Name:  AquariusService.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
* 
* Provides a sample service class that derives from the service base class - 
* AquariusSvcBase. The sample service logs the service start and stop 
* information to the Application event log, and shows how to run the main 
* function of the service in a thread pool worker thread.
* 
\***************************************************************************/

#pragma region Includes
#include "AquariusService.h"
#include "ThreadPool.h"
#include <strsafe.h>
#include "../Shared/Utilities.h"
#include "../Kernel/JobManager.h"
#pragma endregion


AquariusService::AquariusService(PWSTR pszServiceName, 
                               BOOL fCanStop, 
                               BOOL fCanShutdown, 
                               BOOL fCanPauseContinue)
: AquariusSvcBase(pszServiceName, fCanStop, fCanShutdown, fCanPauseContinue)
{
    m_fStopping = FALSE;
	AquariusUtilities::WriteEventLogEntry(L"Going to send a reset event to AquariusSvc");
    // Create a manual-reset event that is not signaled at first to indicate 
    // the stopped signal of the service.
    m_hStoppedEvent = CreateEvent(nullptr, TRUE, FALSE, nullptr);
    if (m_hStoppedEvent == nullptr)
    {
		AquariusUtilities::WriteEventLogEntry(L"Got a null event. Going to throw the last error.");
        throw GetLastError();
    }
}


AquariusService::~AquariusService(void)
{
    if (m_hStoppedEvent)
    {
        CloseHandle(m_hStoppedEvent);
        m_hStoppedEvent = nullptr;
    }
}

//
//   FUNCTION: AquariusService::OnStart(DWORD, LPWSTR *)
//
//   PURPOSE: The function is executed when a Start command is sent to the 
//   service by the SCM or when the operating system starts (for a service 
//   that starts automatically). It specifies actions to take when the 
//   service starts.
//
void AquariusService::OnStart(DWORD dwArgc, LPWSTR *lpszArgv)
{
    // Log a service start message to the Application log.
	AquariusUtilities::WriteEventLogEntry(L"AquariusSvc is going to start.");
	InitializeServices();
    // Queue the main service function for execution in a worker thread.
    CThreadPool::QueueUserWorkItem(&AquariusService::ServiceWorkerThread, this);
	AquariusUtilities::WriteEventLogEntry(L"Enqueued the worker");
}

void AquariusService::InitializeServices(void)
{
	// check if initialization is needed
	// take initialization lock
	wchar_t packageStoreLocation[MAX_PATH];
	DWORD size = 1024;
	aqPackageManager->GetPackageStoreLocation(packageStoreLocation, &size);

	if (aquariusKernel == nullptr)
	{		
		aquariusKernel = new AquariusKernel(packageStoreLocation);
		AquariusUtilities::allocator++;
	}
	else
	{
		//aquariusKernel->UpdateCurrentPackageStoreLocation(packageStoreLocation);
	}
	if(aqPackageManager == nullptr)
	{
		aqPackageManager = new AqPackageManager();
		AquariusUtilities::allocator++;
	}
}

// The method performs the main function of the service. It runs 
// on a thread pool worker thread.
void AquariusService::ServiceWorkerThread(void)
{
    // Periodically check if the service is stopping.
	wchar_t szMessage[260];

    while (!m_fStopping)
    {
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"# of allocations that are not deleted yet: %d.",
			AquariusUtilities::allocator);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		try 
		{
			if (aquariusKernel != nullptr)
			{
				AquariusUtilities::WriteEventLogEntry(L"Aquarius Kernel is loaded");
				wchar_t* pluginDirectory = nullptr; // aquariusKernel->GetCurrentPackageStoreLocation();
				//AquariusUtilities::WriteEventLogEntry((PWSTR)(pluginDirectory));
				ProcessPlugins(pluginDirectory);
			}
			// Perform main service function here...
			AquariusUtilities::WriteEventLogEntry(L"AquariusSvc doing work");
		}
		catch(const char *err)
		{
			AquariusUtilities::WriteErrorLogEntry((PWSTR)(err));
		}
        ::Sleep(5000);  // Simulate some lengthy operations.
    }

    // Signal the stopped event.
    SetEvent(m_hStoppedEvent);
}

void AquariusService::OnStop()
{
    // Log a service stop message to the Application log.
	AquariusUtilities::WriteEventLogEntry(L"AquariusSvc in going to stop: OnStop is called");
	service_objects_mutex._Lock();
	try
	{
		if (aquariusKernel != nullptr)
		{
			AquariusUtilities::WriteEventLogEntry(L"Going to first unload Aquarius kernel");
			delete(aquariusKernel);
			aquariusKernel = nullptr;
			AquariusUtilities::allocator--;
		}

		if (aqPackageManager != nullptr)
		{
			AquariusUtilities::WriteEventLogEntry(L"Going to unload Aquarius PackageManager");
			delete(aqPackageManager);
			AquariusUtilities::allocator--;
			aqPackageManager = nullptr;
		}
	}
	catch(...)
	{
		service_objects_mutex._Unlock();
		throw;
	}
	service_objects_mutex._Unlock();

    // Indicate that the service is stopping and wait for the finish of the 
    // main service function (ServiceWorkerThread).
    m_fStopping = TRUE;

	wchar_t szMessage[260];
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"# of allocations that were nto deleted: %d.",
		AquariusUtilities::allocator);
	AquariusUtilities::WriteEventLogEntry(szMessage);

    if (WaitForSingleObject(m_hStoppedEvent, INFINITE) != WAIT_OBJECT_0)
    {
        DWORD error = GetLastError();
        StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Going to throw error  0x%08lx", error);
        AquariusUtilities::WriteEventLogEntry(szMessage);
        throw GetLastError();
    }
}

void AquariusService::DisplayPluginsMetadata(PluginMetadata* allPluginsMetadata, int pluginCount)
{
	wchar_t szMessage[260];
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Received %d plugins to process", pluginCount);
	AquariusUtilities::WriteEventLogEntry(szMessage);
	for(int i=0; i<pluginCount; i++)
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Plugin to process. Location: %s", allPluginsMetadata[i].location);
		AquariusUtilities::WriteEventLogEntry(szMessage);
	}
}

void AquariusService::ProcessPlugins(wchar_t* pluginDirectory)
{
	//  Get the goal state: 
	// i.e., Get the plugin versions that should be current from Package Manager
	PluginMetadata *allPluginsMetadata = nullptr;
	service_objects_mutex._Lock();
	try 
	{
		if (aqPackageManager != nullptr)
		{
			int pluginCount = aqPackageManager->RetrievePluginsMetadata(&allPluginsMetadata);
			//DisplayPluginsMetadata(allPluginsMetadata, pluginCount);
			aquariusKernel->LoadPlugins(pluginCount, allPluginsMetadata);
			AquariusUtilities::ReleasePluginMetadata(pluginCount, allPluginsMetadata);
			// Forward these to Kernel (retries are okay)
		}
	}
	catch(...)
	{
		service_objects_mutex._Unlock();
		throw;
	}
	service_objects_mutex._Unlock();
}
