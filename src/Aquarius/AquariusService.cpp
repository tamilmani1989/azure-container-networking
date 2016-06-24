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
#pragma endregion


AquariusService::AquariusService(PWSTR pszServiceName, 
                               BOOL fCanStop, 
                               BOOL fCanShutdown, 
                               BOOL fCanPauseContinue)
: AquariusSvcBase(pszServiceName, fCanStop, fCanShutdown, fCanPauseContinue)
{
    m_fStopping = FALSE;
	AquariusUtilities::WriteEventLogEntry(L"Going to send a reset event to AquariusSvc", EVENTLOG_INFORMATION_TYPE);
    // Create a manual-reset event that is not signaled at first to indicate 
    // the stopped signal of the service.
    m_hStoppedEvent = CreateEvent(nullptr, TRUE, FALSE, nullptr);
    if (m_hStoppedEvent == nullptr)
    {
		AquariusUtilities::WriteEventLogEntry(L"Got a null event. Going to throw the last error.", EVENTLOG_INFORMATION_TYPE);
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
	AquariusUtilities::WriteEventLogEntry(L"AquariusSvc is going to start.",
        EVENTLOG_INFORMATION_TYPE);
	InitializeServices();
    // Queue the main service function for execution in a worker thread.
    CThreadPool::QueueUserWorkItem(&AquariusService::ServiceWorkerThread, this);
	AquariusUtilities::WriteEventLogEntry(L"Enqueued the worker", EVENTLOG_INFORMATION_TYPE);
}

void AquariusService::InitializeServices(void)
{
	// check if initialization is needed
	// take initialization lock
	wchar_t packageStoreLocation[1024];
	DWORD size = 1024;
	aqPackageManager->GetPackageStoreLocation(packageStoreLocation, &size);

	if (aquariusKernel == nullptr)
	{		
		aquariusKernel = new AquariusKernel(packageStoreLocation);
	}
	else
	{
		aquariusKernel->UpdateCurrentPackageStoreLocation(packageStoreLocation);
	}
	if(aqPackageManager == nullptr)
	{
		aqPackageManager = new AqPackageManager();
	}
}

// The method performs the main function of the service. It runs 
// on a thread pool worker thread.
void AquariusService::ServiceWorkerThread(void)
{
    // Periodically check if the service is stopping.
    while (!m_fStopping)
    {
		if(aquariusKernel != nullptr)
		{
			AquariusUtilities::WriteEventLogEntry(L"Aquarius Kernel is loaded", EVENTLOG_INFORMATION_TYPE);
			wchar_t* pluginDirectory = aquariusKernel->GetCurrentPackageStoreLocation();
			AquariusUtilities::WriteEventLogEntry((PWSTR)(pluginDirectory), EVENTLOG_INFORMATION_TYPE);
			ProcessPlugins(pluginDirectory);
		}
        // Perform main service function here...
		AquariusUtilities::WriteEventLogEntry(L"AquariusSvc doing work", EVENTLOG_INFORMATION_TYPE);

        ::Sleep(2000);  // Simulate some lengthy operations.
    }

    // Signal the stopped event.
    SetEvent(m_hStoppedEvent);
}

void AquariusService::OnStop()
{
    // Log a service stop message to the Application log.
	AquariusUtilities::WriteEventLogEntry(L"AquariusSvc in going to stop: OnStop is called",
        EVENTLOG_INFORMATION_TYPE);

	if (aquariusKernel != nullptr)
	{
		AquariusUtilities::WriteEventLogEntry(L"Going to first unload Aquarius kernel", EVENTLOG_INFORMATION_TYPE);
		delete(aquariusKernel);
		aquariusKernel = nullptr;
	}

	if(aqPackageManager != nullptr)
	{
		AquariusUtilities::WriteEventLogEntry(L"Going to unload Aquarius PackageManager", EVENTLOG_INFORMATION_TYPE);
		delete(aqPackageManager);
		aqPackageManager = nullptr;
	}

    // Indicate that the service is stopping and wait for the finish of the 
    // main service function (ServiceWorkerThread).
    m_fStopping = TRUE;
    if (WaitForSingleObject(m_hStoppedEvent, INFINITE) != WAIT_OBJECT_0)
    {
        throw GetLastError();
    }
}

void AquariusService::ProcessPlugins(wchar_t* pluginDirectory)
{
	//  Get the goal state: 
	// i.e., Get the current plugin versions to be in from Package Manager

	// Forward these to Kernel (retries are okay)
}
