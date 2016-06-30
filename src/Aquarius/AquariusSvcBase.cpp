/****************************** Module Header ******************************\
* Module Name:  AquariusSvcBase.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
* 
* Provides a base class for a service that will exist as part of a service 
* application. AquariusSvcBase must be derived from when creating a new service 
* class.
* 
\***************************************************************************/

#pragma region Includes
#include "AquariusSvcBase.h"
#include <assert.h>
#include <strsafe.h>
#include "../Shared/Utilities.h"
#pragma endregion


#pragma region Static Members

// Initialize the singleton service instance.
AquariusSvcBase *AquariusSvcBase::s_service = nullptr;


//
//   FUNCTION: AquariusSvcBase::Run(AquariusSvcBase &)
//
//   PURPOSE: Register the executable for Aquarius with the Service Control 
//   Manager (SCM). After you call Run(ServiceBase), the SCM issues a Start 
//   command, which results in a call to the OnStart method in the service. 
//   This method blocks until the service has stopped.
//
//   PARAMETERS:
//   * service - the reference to a AquariusSvcBase object. It will become the 
//     singleton service instance of this service application.
//
//   RETURN VALUE: If the function succeeds, the return value is TRUE. If the 
//   function fails, the return value is FALSE. To get extended error 
//   information, call GetLastError.
//
BOOL AquariusSvcBase::Run(AquariusSvcBase &service)
{
    s_service = &service;
    SERVICE_TABLE_ENTRY serviceTable[] = 
    {
        { service.m_name, ServiceMain },
        { nullptr, nullptr }
    };  

    return StartServiceCtrlDispatcher(serviceTable);
}

//
//   FUNCTION: AquariusSvcBase::ServiceMain(DWORD, PWSTR *)
//
//   PURPOSE: Entry point for the service. It registers the handler function 
//   for the service and starts the service.
//
//   PARAMETERS:
//   * dwArgc   - number of command line arguments
//   * lpszArgv - array of command line arguments
//
void WINAPI AquariusSvcBase::ServiceMain(DWORD dwArgc, PWSTR *pszArgv)
{
    assert(s_service != nullptr);
    
    // Register the handler function for the service
    s_service->m_statusHandle = RegisterServiceCtrlHandler(
        s_service->m_name, ServiceCtrlHandler);

    if (s_service->m_statusHandle == nullptr)
    {
        throw GetLastError();
    }

    // Start the service.
    s_service->Start(dwArgc, pszArgv);
}


//
//   FUNCTION: AquariusSvcBase::ServiceCtrlHandler(DWORD)
//
//   PURPOSE: The function is called by the SCM whenever a control code is 
//   sent to the service. 
//
//   PARAMETERS:
//   * dwCtrlCode - the control code. This parameter can be one of the 
//   following values: 
//
//     SERVICE_CONTROL_CONTINUE
//     SERVICE_CONTROL_INTERROGATE
//     SERVICE_CONTROL_NETBINDADD
//     SERVICE_CONTROL_NETBINDDISABLE
//     SERVICE_CONTROL_NETBINDREMOVE
//     SERVICE_CONTROL_PARAMCHANGE
//     SERVICE_CONTROL_PAUSE
//     SERVICE_CONTROL_SHUTDOWN
//     SERVICE_CONTROL_STOP
//
//   This parameter can also be a user-defined control code ranges from 128 
//   to 255.
//
void WINAPI AquariusSvcBase::ServiceCtrlHandler(DWORD dwCtrl)
{
    switch (dwCtrl)
    {
    case SERVICE_CONTROL_STOP: s_service->Stop(); break;
    case SERVICE_CONTROL_PAUSE: s_service->Pause(); break;
    case SERVICE_CONTROL_CONTINUE: s_service->Continue(); break;
    case SERVICE_CONTROL_SHUTDOWN: s_service->Shutdown(); break;
    case SERVICE_CONTROL_INTERROGATE: break;
    default: break;
    }
}

#pragma endregion


#pragma region Service Constructor and Destructor

//
//   FUNCTION: AquariusSvcBase::AquariusSvcBase(PWSTR, BOOL, BOOL, BOOL)
//
//   PURPOSE: The constructor of AquariusSvcBase. It initializes a new instance 
//   of the AquariusSvcBase class. The optional parameters (fCanStop, 
///  fCanShutdown and fCanPauseContinue) allow you to specify whether the 
//   service can be stopped, paused and continued, or be notified when system 
//   shutdown occurs.
//
//   PARAMETERS:
//   * pszServiceName - the name of the service
//   * fCanStop - the service can be stopped
//   * fCanShutdown - the service is notified when system shutdown occurs
//   * fCanPauseContinue - the service can be paused and continued
//
AquariusSvcBase::AquariusSvcBase(PWSTR pszServiceName, 
                           BOOL fCanStop, 
                           BOOL fCanShutdown, 
                           BOOL fCanPauseContinue)
{
    // Service name must be a valid string and cannot be nullptr.
    m_name = (pszServiceName == nullptr) ? L"" : pszServiceName;

    m_statusHandle = nullptr;

    // The service runs in its own process.
    m_status.dwServiceType = SERVICE_WIN32_OWN_PROCESS;

    // The service is starting.
    m_status.dwCurrentState = SERVICE_START_PENDING;

    // The accepted commands of the service.
    DWORD dwControlsAccepted = 0;
    if (fCanStop) 
        dwControlsAccepted |= SERVICE_ACCEPT_STOP;
    if (fCanShutdown) 
        dwControlsAccepted |= SERVICE_ACCEPT_SHUTDOWN;
    if (fCanPauseContinue) 
        dwControlsAccepted |= SERVICE_ACCEPT_PAUSE_CONTINUE;
    m_status.dwControlsAccepted = dwControlsAccepted;

    m_status.dwWin32ExitCode = NO_ERROR;
    m_status.dwServiceSpecificExitCode = 0;
    m_status.dwCheckPoint = 0;
    m_status.dwWaitHint = 0;
}


//
//   FUNCTION: AquariusSvcBase::~AquariusSvcBase()
//
//   PURPOSE: The virtual destructor of AquariusSvcBase. 
//
AquariusSvcBase::~AquariusSvcBase(void)
{
}

#pragma endregion


#pragma region Service Start, Stop, Pause, Continue, and Shutdown

//
//   FUNCTION: AquariusSvcBase::Start(DWORD, PWSTR *)
//
//   PURPOSE: The function starts the service. It calls the OnStart virtual 
//   function in which you can specify the actions to take when the service 
//   starts. If an error occurs during the startup, the error will be logged 
//   in the Application event log, and the service will be stopped.
//
//   PARAMETERS:
//   * dwArgc   - number of command line arguments
//   * lpszArgv - array of command line arguments
//
void AquariusSvcBase::Start(DWORD dwArgc, PWSTR *pszArgv)
{
    try
    {
        
		AquariusUtilities::WriteEventLogEntry(L"Going to set status as pending");

        // Tell SCM that the service is starting.
        SetServiceStatus(SERVICE_START_PENDING);

		AquariusUtilities::WriteEventLogEntry(L"Going to call on start");
        // Perform service-specific initialization.
        OnStart(dwArgc, pszArgv);

		AquariusUtilities::WriteEventLogEntry(L"Going to mark service as running");
        // Tell SCM that the service is started.
        SetServiceStatus(SERVICE_RUNNING);

		AquariusUtilities::WriteEventLogEntry(L"Successfully marked as running.");
    }
    catch (DWORD dwError)
    {
        // Log the error.
		AquariusUtilities::WriteErrorLogEntry(L"Error during Service Start");

        // Set the service status to be stopped.
        SetServiceStatus(SERVICE_STOPPED, dwError);
    }
    catch (...)
    {
        // Log the error.
		AquariusUtilities::WriteEventLogEntry(L"Service failed to start.");

        // Set the service status to be stopped.
        SetServiceStatus(SERVICE_STOPPED);
    }
}


//
//   FUNCTION: AquariusSvcBase::OnStart(DWORD, PWSTR *)
//
//   PURPOSE: When implemented in a derived class, executes when a Start 
//   command is sent to the service by the SCM or when the operating system 
//   starts (for a service that starts automatically). Specifies actions to 
//   take when the service starts. Be sure to periodically call 
//   AquariusSvcBase::SetServiceStatus() with SERVICE_START_PENDING if the 
//   procedure is going to take long time. You may also consider spawning a 
//   new thread in OnStart to perform time-consuming initialization tasks.
//
//   PARAMETERS:
//   * dwArgc   - number of command line arguments
//   * lpszArgv - array of command line arguments
//
void AquariusSvcBase::OnStart(DWORD dwArgc, PWSTR *pszArgv)
{
}


//
//   FUNCTION: AquariusSvcBase::Stop()
//
//   PURPOSE: The function stops the service. It calls the OnStop virtual 
//   function in which you can specify the actions to take when the service 
//   stops. If an error occurs, the error will be logged in the Application 
//   event log, and the service will be restored to the original state.
//
void AquariusSvcBase::Stop()
{
    DWORD dwOriginalState = m_status.dwCurrentState;
    try
    {
        // Tell SCM that the service is stopping.
        SetServiceStatus(SERVICE_STOP_PENDING);

        // Perform service-specific stop operations.
        OnStop();

        // Tell SCM that the service is stopped.
        SetServiceStatus(SERVICE_STOPPED);
    }
    catch (DWORD dwError)
    {
        // Log the error.
		AquariusUtilities::WriteErrorLogEntry(L"Error during Service Stop");

        // Set the orginal service status.
        SetServiceStatus(dwOriginalState);
    }
    catch (...)
    {
        // Log the error.
		AquariusUtilities::WriteEventLogEntry(L"Service failed to stop.");

        // Set the orginal service status.
        SetServiceStatus(dwOriginalState);
    }
}


//
//   FUNCTION: AquariusSvcBase::OnStop()
//
//   PURPOSE: When implemented in a derived class, executes when a Stop 
//   command is sent to the service by the SCM. Specifies actions to take 
//   when a service stops running. Be sure to periodically call 
//   AquariusSvcBase::SetServiceStatus() with SERVICE_STOP_PENDING if the 
//   procedure is going to take long time. 
//
void AquariusSvcBase::OnStop()
{
}


//
//   FUNCTION: AquariusSvcBase::Pause()
//
//   PURPOSE: The function pauses the service if the service supports pause 
//   and continue. It calls the OnPause virtual function in which you can 
//   specify the actions to take when the service pauses. If an error occurs, 
//   the error will be logged in the Application event log, and the service 
//   will become running.
//
void AquariusSvcBase::Pause()
{
    try
    {
        // Tell SCM that the service is pausing.
        SetServiceStatus(SERVICE_PAUSE_PENDING);

        // Perform service-specific pause operations.
        OnPause();

        // Tell SCM that the service is paused.
        SetServiceStatus(SERVICE_PAUSED);
    }
    catch (DWORD dwError)
    {
        // Log the error.
		AquariusUtilities::WriteErrorLogEntry(L"Service Pause");

        // Tell SCM that the service is still running.
        SetServiceStatus(SERVICE_RUNNING);
    }
    catch (...)
    {
        // Log the error.
		AquariusUtilities::WriteEventLogEntry(L"Service failed to pause.");

        // Tell SCM that the service is still running.
        SetServiceStatus(SERVICE_RUNNING);
    }
}


//
//   FUNCTION: AquariusSvcBase::OnPause()
//
//   PURPOSE: When implemented in a derived class, executes when a Pause 
//   command is sent to the service by the SCM. Specifies actions to take 
//   when a service pauses.
//
void AquariusSvcBase::OnPause()
{
}


//
//   FUNCTION: AquariusSvcBase::Continue()
//
//   PURPOSE: The function resumes normal functioning after being paused if
//   the service supports pause and continue. It calls the OnContinue virtual 
//   function in which you can specify the actions to take when the service 
//   continues. If an error occurs, the error will be logged in the 
//   Application event log, and the service will still be paused.
//
void AquariusSvcBase::Continue()
{
    try
    {
        // Tell SCM that the service is resuming.
        SetServiceStatus(SERVICE_CONTINUE_PENDING);

        // Perform service-specific continue operations.
        OnContinue();

        // Tell SCM that the service is running.
        SetServiceStatus(SERVICE_RUNNING);
    }
    catch (DWORD dwError)
    {
        // Log the error.
		AquariusUtilities::WriteErrorLogEntry(L"Service Continue");

        // Tell SCM that the service is still paused.
        SetServiceStatus(SERVICE_PAUSED);
    }
    catch (...)
    {
        // Log the error.
		AquariusUtilities::WriteEventLogEntry(L"Service failed to resume.");

        // Tell SCM that the service is still paused.
        SetServiceStatus(SERVICE_PAUSED);
    }
}


//
//   FUNCTION: AquariusSvcBase::OnContinue()
//
//   PURPOSE: When implemented in a derived class, OnContinue runs when a 
//   Continue command is sent to the service by the SCM. Specifies actions to 
//   take when a service resumes normal functioning after being paused.
//
void AquariusSvcBase::OnContinue()
{
}


//
//   FUNCTION: AquariusSvcBase::Shutdown()
//
//   PURPOSE: The function executes when the system is shutting down. It 
//   calls the OnShutdown virtual function in which you can specify what 
//   should occur immediately prior to the system shutting down. If an error 
//   occurs, the error will be logged in the Application event log.
//
void AquariusSvcBase::Shutdown()
{
    try
    {
        // Perform service-specific shutdown operations.
        OnShutdown();

        // Tell SCM that the service is stopped.
        SetServiceStatus(SERVICE_STOPPED);
    }
    catch (DWORD dwError)
    {
        // Log the error.
		AquariusUtilities::WriteErrorLogEntry(L"Error during Service Shutdown");
    }
    catch (...)
    {
        // Log the error.
		AquariusUtilities::WriteErrorLogEntry(L"Service failed to shut down.");
    }
}


//
//   FUNCTION: AquariusSvcBase::OnShutdown()
//
//   PURPOSE: When implemented in a derived class, executes when the system 
//   is shutting down. Specifies what should occur immediately prior to the 
//   system shutting down.
//
void AquariusSvcBase::OnShutdown()
{
}

#pragma endregion


#pragma region Helper Functions

//
//   FUNCTION: AquariusSvcBase::SetServiceStatus(DWORD, DWORD, DWORD)
//
//   PURPOSE: The function sets the service status and reports the status to 
//   the SCM.
//
//   PARAMETERS:
//   * dwCurrentState - the state of the service
//   * dwWin32ExitCode - error code to report
//   * dwWaitHint - estimated time for pending operation, in milliseconds
//
void AquariusSvcBase::SetServiceStatus(DWORD dwCurrentState, 
                                    DWORD dwWin32ExitCode, 
                                    DWORD dwWaitHint)
{
    static DWORD dwCheckPoint = 1;

    // Fill in the SERVICE_STATUS structure of the service.

    m_status.dwCurrentState = dwCurrentState;
    m_status.dwWin32ExitCode = dwWin32ExitCode;
    m_status.dwWaitHint = dwWaitHint;

    m_status.dwCheckPoint = 
        ((dwCurrentState == SERVICE_RUNNING) ||
        (dwCurrentState == SERVICE_STOPPED)) ? 
        0 : dwCheckPoint++;

    // Report the status of the service to the SCM.
    ::SetServiceStatus(m_statusHandle, &m_status);
}




#pragma endregion