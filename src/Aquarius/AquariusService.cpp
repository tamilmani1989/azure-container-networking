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

    // Queue the main service function for execution in a worker thread.
    CThreadPool::QueueUserWorkItem(&AquariusService::ServiceWorkerThread, this);
    WriteEventLogEntry(L"Enqueued the worker", EVENTLOG_INFORMATION_TYPE);
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
        // Perform main service function here...
        WriteEventLogEntry(L"AquariusSvc doing work", 
        EVENTLOG_INFORMATION_TYPE);
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