/*++

Copyright (c) Microsoft Corporation

Module Name:

VfpOperations.cpp

Abstract:

This cotains the implementaion of methods that perform operations specific to Vfp.

Author:

Sushant Sharma (susharm) 20-April-2017

--*/

#ifdef _WIN32
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#include "VfpOperations.h"

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
	
	wchar_t* VfpOperations::vfpProcessName = L"vfpext";

	static DWORD WaitToStop(
		SC_HANDLE schSCManager, 
		SC_HANDLE schService, 
		SERVICE_STATUS_PROCESS ssStatus)
	/*++

	Routine Description:

		Waits for a service to stop.
	
	Arguments:

		schSCManager - handle to SC Manager.
		schService - handle to service.
		ssStatus - process status info for the service

	Return Value:

		0 if service successfully stoped, error code otherwise.
		
	--*/
	{
		LOGINFO("Entering WaitToStop.");
		DWORD dwOldCheckPoint;
		DWORD dwStartTickCount;
		DWORD dwWaitTime;
		DWORD retval = 0;
		DWORD dwBytesNeeded;

		dwStartTickCount = GetTickCount();
		dwOldCheckPoint = ssStatus.dwCheckPoint;

		// Wait for the service to stop before attempting to start it.

		while (ssStatus.dwCurrentState != SERVICE_STOPPED)
		{
			// Do not wait longer than the wait hint/10.

			dwWaitTime = ssStatus.dwWaitHint / 10;

			if (dwWaitTime < 5000)
			{
				dwWaitTime = 5000;
			}
			else if (dwWaitTime > 30000)
			{
				dwWaitTime = 30000;
			}

			Sleep(dwWaitTime);

			// Check the status until the service is no longer stop pending. 

			if (!QueryServiceStatusEx(schService, SC_STATUS_PROCESS_INFO, (LPBYTE)&ssStatus, sizeof(SERVICE_STATUS_PROCESS), &dwBytesNeeded))
			{
				LOGERR("QueryServiceStatusEx failed (%d)\n", GetLastError());
				CloseServiceHandle(schService);
				CloseServiceHandle(schSCManager);
				retval = ERROR_STARTING_VFP;
				break;
			}

			if (ssStatus.dwCheckPoint > dwOldCheckPoint)
			{
				// Continue to wait and check.
				dwStartTickCount = GetTickCount();
				dwOldCheckPoint = ssStatus.dwCheckPoint;
			}
			else
			{
				if (GetTickCount() - dwStartTickCount > ssStatus.dwWaitHint)
				{
					LOGERR("Timeout waiting for service to stop\n");
					CloseServiceHandle(schService);
					CloseServiceHandle(schSCManager);
					retval = ERROR_STARTING_VFP;
					break;
				}
			}
		}

		LOGINFO("Exiting WaitToStop. Returning %lu", retval);
		return retval;
	}

	static DWORD WaitToStart(
		SC_HANDLE schSCManager, 
		SC_HANDLE schService, 
		SERVICE_STATUS_PROCESS ssStatus)
	/*++

	Routine Description:

		Waits for a service to start.
	
	Arguments:

		schSCManager - handle to SC Manager.
		schService - handle to service.
		ssStatus - process status info for the service

	Return Value:

		0 if service successfully started, error code otherwise.
		
	--*/
	{
		LOGINFO("Entering WaitToStart.");
		DWORD dwOldCheckPoint;
		DWORD dwStartTickCount;
		DWORD dwWaitTime;
		DWORD retval = 0;
		DWORD dwBytesNeeded;

		if (!QueryServiceStatusEx(schService, SC_STATUS_PROCESS_INFO, (LPBYTE)&ssStatus, sizeof(SERVICE_STATUS_PROCESS), &dwBytesNeeded))
		{
			LOGERR("QueryServiceStatusEx failed (%d)\n", GetLastError());
			CloseServiceHandle(schService);
			CloseServiceHandle(schSCManager);
			retval = ERROR_STARTING_VFP;
		}
		else
		{
			// Save the tick count and initial checkpoint.

			dwStartTickCount = GetTickCount();
			dwOldCheckPoint = ssStatus.dwCheckPoint;

			while (ssStatus.dwCurrentState != SERVICE_RUNNING)
			{
				// Do not wait longer than the wait hint/10.

				dwWaitTime = ssStatus.dwWaitHint / 10;

				if (dwWaitTime < 5000)
				{
					dwWaitTime = 5000;
				}
				else if (dwWaitTime > 30000)
				{
					dwWaitTime = 30000;
				}

				Sleep(dwWaitTime);

				// Check the status again. 

				if (!QueryServiceStatusEx(schService, SC_STATUS_PROCESS_INFO, (LPBYTE)&ssStatus, sizeof(SERVICE_STATUS_PROCESS), &dwBytesNeeded))
				{
					LOGERR("QueryServiceStatusEx failed (%d)", GetLastError());
					CloseServiceHandle(schService);
					CloseServiceHandle(schSCManager);
					retval = ERROR_STARTING_VFP;
					break;
				}

				if (ssStatus.dwCheckPoint > dwOldCheckPoint)
				{
					// Continue to wait and check.
					dwStartTickCount = GetTickCount();
					dwOldCheckPoint = ssStatus.dwCheckPoint;
				}
				else
				{
					if (GetTickCount() - dwStartTickCount > ssStatus.dwWaitHint)
					{
						// No progress made within the wait hint.
						LOGERR("Timeout waiting for service to start");
						CloseServiceHandle(schService);
						CloseServiceHandle(schSCManager);
						retval = ERROR_STARTING_VFP;
						break;
					}
				}
			}
		}
		LOGINFO("Exiting WaitToStart. Returning %lu", retval);
		return retval;
	}

	DWORD VfpOperations::StartVfpService()
	{
		return StartServiceByName(vfpProcessName);
	}

	DWORD VfpOperations::CheckIfServiceStartedSuccessfully(
		SERVICE_STATUS_PROCESS ssStatus, 
		SC_HANDLE schSCManager, 
		SC_HANDLE schService)
	/*++

	Routine Description:

		Checks if a service started succesfully or not.

	Arguments:

		schSCManager - handle to SC Manager.
		schService - handle to service.
		ssStatus - process status info for the service

	Return Value:

	0 if successfully started. error code, otherwise.

	--*/
	{	
		LOGINFO("Entering CheckIfServiceStartedSuccessfully.");
		DWORD dwBytesNeeded;
		DWORD retval = 0;
		if (!QueryServiceStatusEx(schService, SC_STATUS_PROCESS_INFO, (LPBYTE)&ssStatus, sizeof(SERVICE_STATUS_PROCESS), &dwBytesNeeded))
		{
			LOGERR("QueryServiceStatusEx failed (%d)\n", GetLastError());
			CloseServiceHandle(schService);
			CloseServiceHandle(schSCManager);
			retval = ERROR_STARTING_VFP;
		}
		else
		if (ssStatus.dwCurrentState == SERVICE_RUNNING)
		{
			LOGINFO("Service started successfully.");
		}
		else
		{
			LOGERR("Service not started. \n");
			LOGERR("  Current State: %d\n", ssStatus.dwCurrentState);
			LOGERR("  Exit Code: %d\n", ssStatus.dwWin32ExitCode);
			LOGERR("  Check Point: %d\n", ssStatus.dwCheckPoint);
			LOGERR("  Wait Hint: %d\n", ssStatus.dwWaitHint);
			retval = ERROR_STARTING_VFP;
		}

		CloseServiceHandle(schService);
		CloseServiceHandle(schSCManager);	
		LOGINFO("Exiting CheckIfServiceStartedSuccessfully. Returning %lu.", retval);
		return retval;
	}

	DWORD VfpOperations::StartServiceAndWaitToStart( 
		SERVICE_STATUS_PROCESS ssStatus, 
		SC_HANDLE schSCManager, 
		SC_HANDLE schService)
	/*++

	Routine Description:

		Starts a service and wait for it to start.

	Arguments:

		schSCManager - handle to SC Manager.
		schService - handle to service.
		ssStatus - process status info for the service

	Return Value:

	0 if successfully started. error code, otherwise.

	--*/
	{
		LOGINFO("EnteringStartServiceAndWaitToStart. ");
		DWORD retval = 0;
		if (!StartService(schService, 0, NULL))
		{
			LOGERR("StartService failed (%d)", GetLastError());
			CloseServiceHandle(schService);
			CloseServiceHandle(schSCManager);
			retval = ERROR_STARTING_VFP;
		}
		else
		{
			retval = WaitToStart(schSCManager, schService, ssStatus);
			if (retval != ERROR_STARTING_VFP)
			{
				retval = CheckIfServiceStartedSuccessfully(ssStatus, schSCManager, schService);
			}
		}
		LOGINFO("Exiting StartServiceAndWaitToStart. Returning %lu", retval);
		return retval;
	}

	DWORD VfpOperations::StartServiceByName(
		wchar_t * vfpProcName)
	/*++

	Routine Description:

		Starts a windows service.

	Arguments:

	vfpProcName - name of the service to start.

	Return Value:

	0 if successfully started. error code, otherwise.

	--*/
	{
		LOGINFO("Entering StartServiceByName.");
		DWORD retval = 0;
		SERVICE_STATUS_PROCESS ssStatus;

		// Get a handle to the SCM database. 
		SC_HANDLE schSCManager = OpenSCManager(NULL, NULL, SC_MANAGER_ALL_ACCESS);  // full access rights 

		if (schSCManager == NULL)
		{
			LOGERR("OpenSCManager failed (%d)\n", GetLastError());
			retval = ERROR_STARTING_VFP;
		}
		else
		{
			SC_HANDLE  schService = OpenService(schSCManager, vfpProcName, SERVICE_ALL_ACCESS);  // full access 

			if (schService == NULL)
			{
				LOGERR("OpenService %ws failed (%d)\n", vfpProcName, GetLastError());
				CloseServiceHandle(schSCManager);
				retval = ERROR_STARTING_VFP;
			}
			else
			{
				DWORD dwBytesNeeded;
				if (!QueryServiceStatusEx(schService, SC_STATUS_PROCESS_INFO, (LPBYTE)&ssStatus, sizeof(SERVICE_STATUS_PROCESS), &dwBytesNeeded))
				{
					LOGERR("QueryServiceStatusEx failed (%d)\n", GetLastError());
					CloseServiceHandle(schService);
					CloseServiceHandle(schSCManager);
					retval = ERROR_STARTING_VFP;
				}
				else
				{
					if(ssStatus.dwCurrentState == SERVICE_RUNNING)
					{
						LOGINFO("%ws is already running. no need to start", vfpProcName);
						CloseServiceHandle(schService);
						CloseServiceHandle(schSCManager);
					}
					else
					if (ssStatus.dwCurrentState != SERVICE_STOPPED && 
						ssStatus.dwCurrentState != SERVICE_STOP_PENDING)
					{
						retval = StartServiceAndWaitToStart(ssStatus, schSCManager, schService);
					}
					else
					{
						retval = WaitToStop(schSCManager, schService, ssStatus);

						if(retval != ERROR_STARTING_VFP)
						{
							retval = StartServiceAndWaitToStart(ssStatus, schSCManager, schService);
						}
					}
				}
			}
		}
		LOGINFO("Exiting StartServiceByName. Returning %lu", retval);
		return retval;
	}


} } } } }
#endif