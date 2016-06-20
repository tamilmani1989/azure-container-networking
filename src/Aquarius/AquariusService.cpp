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
#pragma endregion


AquariusService::AquariusService(PWSTR pszServiceName, 
                               BOOL fCanStop, 
                               BOOL fCanShutdown, 
                               BOOL fCanPauseContinue)
: AquariusSvcBase(pszServiceName, fCanStop, fCanShutdown, fCanPauseContinue)
{
    m_fStopping = FALSE;
    WriteEventLogEntry(L"Going to send a reset event to AquariusSvc", EVENTLOG_INFORMATION_TYPE);
    // Create a manual-reset event that is not signaled at first to indicate 
    // the stopped signal of the service.
    m_hStoppedEvent = CreateEvent(nullptr, TRUE, FALSE, nullptr);
    if (m_hStoppedEvent == nullptr)
    {
        WriteEventLogEntry(L"Got a null event. Going to throw the last error.", EVENTLOG_INFORMATION_TYPE);
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
	if (aquariusKernel != nullptr)
	{
		delete(aquariusKernel);
		aquariusKernel = nullptr;
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
    WriteEventLogEntry(L"AquariusSvc is going to start.", 
        EVENTLOG_INFORMATION_TYPE);
	InitializeServices();
    // Queue the main service function for execution in a worker thread.
    CThreadPool::QueueUserWorkItem(&AquariusService::ServiceWorkerThread, this);
    WriteEventLogEntry(L"Enqueued the worker", EVENTLOG_INFORMATION_TYPE);
}

void AquariusService::InitializeServices(void)
{
	// check if initialization is needed
	// take initialization lock
	wchar_t packageStoreLocation[1024];
	DWORD size = 1024;
	GetPackageStoreLocation(packageStoreLocation, &size);

	if (aquariusKernel == nullptr)
	{		
		aquariusKernel = new AquariusKernel(packageStoreLocation);
	}
	else
	{
		aquariusKernel->UpdatePackageStoreLocation(packageStoreLocation);
	}
}

//
//   FUNCTION: AquariusService::ServiceWorkerThread(void)
//
//   PURPOSE: The method performs the main function of the service. It runs 
//   on a thread pool worker thread.
//
void AquariusService::ServiceWorkerThread(void)
{
    // Periodically check if the service is stopping.
    while (!m_fStopping)
    {
		if(aquariusKernel != nullptr)
		{
			WriteEventLogEntry(L"Aquarius Kernel is loaded", EVENTLOG_INFORMATION_TYPE);
			WriteEventLogEntry((PWSTR)(aquariusKernel->GetPackageStoreLocation()), EVENTLOG_INFORMATION_TYPE);
		}
        // Perform main service function here...
        WriteEventLogEntry(L"AquariusSvc doing work", EVENTLOG_INFORMATION_TYPE);

        ::Sleep(2000);  // Simulate some lengthy operations.
    }

    // Signal the stopped event.
    SetEvent(m_hStoppedEvent);
}

//
//   FUNCTION: AquariusService::OnStop(void)
//
//   PURPOSE: The function is executed when a Stop command is sent to the 
//   service by SCM. It specifies actions to take when a service stops 
//   running. In this code sample, OnStop logs a service-stop message to the 
//   Application log, and waits for the finish of the main service function.
//
//   COMMENTS:
//   Be sure to periodically call ReportServiceStatus() with 
//   SERVICE_STOP_PENDING if the procedure is going to take long time. 
//
void AquariusService::OnStop()
{
    // Log a service stop message to the Application log.
    WriteEventLogEntry(L"AquariusSvc in OnStop", 
        EVENTLOG_INFORMATION_TYPE);

    // Indicate that the service is stopping and wait for the finish of the 
    // main service function (ServiceWorkerThread).
    m_fStopping = TRUE;
    if (WaitForSingleObject(m_hStoppedEvent, INFINITE) != WAIT_OBJECT_0)
    {
        throw GetLastError();
    }
}

void AquariusService::GetPackageStoreLocation(wchar_t *packageStoreLocation, DWORD *size)
{
	WriteEventLogEntry(L"Going to get package store location.", EVENTLOG_INFORMATION_TYPE);
	WriteEventLogEntry(L"Trying to open registry key that contains config.", EVENTLOG_INFORMATION_TYPE);
	HKEY configKey = OpenRegistryKey(HKEY_LOCAL_MACHINE, L"SOFTWARE\\Microsoft\\Aquarius\\Config");	
	if (configKey == nullptr)
	{
		WriteEventLogEntry(L"Registry key does not exist. Returning defauilt package location.", 
				EVENTLOG_INFORMATION_TYPE);
		wcscpy(packageStoreLocation, defaultPackageStoreLocation);
		wchar_t szMessage[260];
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Successfully returned default package location: %s", packageStoreLocation);
		WriteEventLogEntry(szMessage, EVENTLOG_INFORMATION_TYPE);
		return;
	}
	WriteEventLogEntry(L"Found registry key.", EVENTLOG_INFORMATION_TYPE);

	GetStringFromRegistry(configKey, L"PackageStoreLocation", packageStoreLocation, size);
}

HKEY AquariusService::OpenRegistryKey(HKEY rootKey, wchar_t* subKey)
{
	HKEY returnKey;
	
	LONG nError = RegOpenKeyEx(rootKey, subKey, NULL, KEY_ALL_ACCESS, &returnKey);
	WriteEventLogEntry(L"Open key operation executed.", EVENTLOG_INFORMATION_TYPE);
	if (nError == ERROR_FILE_NOT_FOUND)
	{		
		WriteEventLogEntry(L"Config key does not exist.", EVENTLOG_INFORMATION_TYPE);
		nError = RegCreateKeyEx(rootKey, subKey, NULL, nullptr, REG_OPTION_NON_VOLATILE,
			KEY_ALL_ACCESS, nullptr, &returnKey, nullptr);
		WriteEventLogEntry(L"Created config key with default location.", EVENTLOG_INFORMATION_TYPE);
	}

	if (nError)
	{
		WriteEventLogEntry(L"Some error happnned during config key creation with default location.", 
			EVENTLOG_INFORMATION_TYPE);
		//HandleRegistryError(rootKey, subKey, nError);
		//WriteEventLogEntry(L"Logged error.", EVENTLOG_INFORMATION_TYPE);
		returnKey = nullptr;
	}

	return returnKey;
}

void AquariusService::GetStringFromRegistry(HKEY configKey, wchar_t* key, wchar_t* returnValue, DWORD* size)
{
	DWORD type = REG_SZ;
	LONG nError = RegQueryValueEx(configKey, key, nullptr, &type, (LPBYTE)returnValue, size);

	if (nError)
	{
		HandleRegistryError(configKey, key, nError);
	}
}

void AquariusService::HandleRegistryError(HKEY rootKey, wchar_t* subKey, LONG nError)
{
	wchar_t szMessage[260];
	try
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Reg Open of %s %s failed w/err 0x%08lx", rootKey, subKey, nError);
		WriteEventLogEntry(szMessage, EVENTLOG_ERROR_TYPE);
	}
	catch (const exception &ex)
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"%s", ex.what());
		WriteErrorLogEntry(szMessage, EVENTLOG_ERROR_TYPE);
	}
	catch (...)
	{
		WriteErrorLogEntry(L"Unknown error happenned while crating registry key", EVENTLOG_ERROR_TYPE);
	}
}