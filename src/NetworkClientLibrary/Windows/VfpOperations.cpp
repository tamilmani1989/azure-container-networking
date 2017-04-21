/****************************** Module Header ******************************\
* Module Name:  VfpOperations.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Setup vfp in windows
*
\***************************************************************************/

#ifdef _WIN32
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#include "VfpOperations.h"

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
	
	wchar_t* VfpOperations::vfpProcessName = L"vfpext";

	static DWORD WaitToStop(SC_HANDLE schSCManager, SC_HANDLE schService, SERVICE_STATUS_PROCESS ssStatus)
	{
		DWORD dwOldCheckPoint;
		DWORD dwStartTickCount;
		DWORD dwWaitTime;
		DWORD retval = 0;
		DWORD dwBytesNeeded;
		char logMsg[2048];
		size_t msgSize = 2048;

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
			else if (dwWaitTime > 10000)
			{
				dwWaitTime = 10000;
			}

			Sleep(dwWaitTime);

			// Check the status until the service is no longer stop pending. 

			if (!QueryServiceStatusEx(schService, SC_STATUS_PROCESS_INFO, (LPBYTE)&ssStatus, sizeof(SERVICE_STATUS_PROCESS), &dwBytesNeeded))
			{
				sprintf_s(logMsg, msgSize, "QueryServiceStatusEx failed (%d)\n", GetLastError());
				AquaLogging::Log(err, logMsg);
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
					sprintf_s(logMsg, msgSize, "Timeout waiting for service to stop\n");
					AquaLogging::Log(err, logMsg);
					CloseServiceHandle(schService);
					CloseServiceHandle(schSCManager);
					retval = ERROR_STARTING_VFP;
					break;
				}
			}
		}
		return retval;
	}

	static DWORD WaitToStart(SC_HANDLE schSCManager, SC_HANDLE schService, SERVICE_STATUS_PROCESS ssStatus)
	{
		DWORD dwOldCheckPoint;
		DWORD dwStartTickCount;
		DWORD dwWaitTime;
		DWORD retval = 0;
		DWORD dwBytesNeeded;
		char logMsg[2048];
		size_t msgSize = 2048;

		if (!QueryServiceStatusEx(schService, SC_STATUS_PROCESS_INFO, (LPBYTE)&ssStatus, sizeof(SERVICE_STATUS_PROCESS), &dwBytesNeeded))
		{
			sprintf_s(logMsg, msgSize, "QueryServiceStatusEx failed (%d)\n", GetLastError());
			AquaLogging::Log(err, logMsg);
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
				else if (dwWaitTime > 10000)
				{
					dwWaitTime = 10000;
				}

				Sleep(dwWaitTime);

				// Check the status again. 

				if (!QueryServiceStatusEx(schService, SC_STATUS_PROCESS_INFO, (LPBYTE)&ssStatus, sizeof(SERVICE_STATUS_PROCESS), &dwBytesNeeded))
				{
					sprintf_s(logMsg, msgSize, "QueryServiceStatusEx failed (%d)", GetLastError());
					AquaLogging::Log(err, logMsg);
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
						sprintf_s(logMsg, msgSize, "Timeout waiting for service to start");
						AquaLogging::Log(err, logMsg);
						CloseServiceHandle(schService);
						CloseServiceHandle(schSCManager);
						retval = ERROR_STARTING_VFP;
						break;
					}
				}
			}
		}
		return retval;
	}

	DWORD VfpOperations::StartVfpService()
	{
		return StartServiceByName(vfpProcessName);
	}

	DWORD VfpOperations::StartServiceByName(wchar_t * vfpProcName)
	{
		DWORD retval = 0;
		SERVICE_STATUS_PROCESS ssStatus;
		char logMsg[2048];
		size_t msgSize = 2048;

		// Get a handle to the SCM database. 

		SC_HANDLE schSCManager = OpenSCManager(NULL, NULL, SC_MANAGER_ALL_ACCESS);  // full access rights 

		if (schSCManager == NULL)
		{
			sprintf_s(logMsg, msgSize, "OpenSCManager failed (%d)\n", GetLastError());
			AquaLogging::Log(err, logMsg);
			retval = ERROR_STARTING_VFP;
		}
		else
		{
			SC_HANDLE  schService = OpenService(schSCManager, vfpProcName, SERVICE_ALL_ACCESS);  // full access 

			if (schService == NULL)
			{
				sprintf_s(logMsg, msgSize, "OpenService %ws failed (%d)\n", vfpProcName, GetLastError());
				AquaLogging::Log(err, logMsg);
				CloseServiceHandle(schSCManager);
				retval = ERROR_STARTING_VFP;
			}
			else
			{
				DWORD dwBytesNeeded;
				if (!QueryServiceStatusEx(schService, SC_STATUS_PROCESS_INFO, (LPBYTE)&ssStatus, sizeof(SERVICE_STATUS_PROCESS), &dwBytesNeeded))
				{
					sprintf_s(logMsg, msgSize, "QueryServiceStatusEx failed (%d)\n", GetLastError());
					AquaLogging::Log(err, logMsg);
					CloseServiceHandle(schService);
					CloseServiceHandle(schSCManager);
					retval = ERROR_STARTING_VFP;
				}
				else
				{
					if (ssStatus.dwCurrentState != SERVICE_STOPPED && ssStatus.dwCurrentState != SERVICE_STOP_PENDING)
					{
						sprintf_s(logMsg, msgSize, "%ws is already running. no need to start", vfpProcName);
						AquaLogging::Log(info, logMsg);
						CloseServiceHandle(schService);
						CloseServiceHandle(schSCManager);
					}
					else
					{
						retval = WaitToStop(schSCManager, schService, ssStatus);

						if(retval != ERROR_STARTING_VFP)
						{
							if (!StartService(schService, 0, NULL))
							{
								sprintf_s(logMsg, msgSize, "StartService failed (%d)\n", GetLastError());
								AquaLogging::Log(err, logMsg);
								CloseServiceHandle(schService);
								CloseServiceHandle(schSCManager);
								retval = ERROR_STARTING_VFP;
							}
							else
							{
								retval = WaitToStart(schSCManager, schService, ssStatus);


								if (retval != ERROR_STARTING_VFP)
								{
									if (!QueryServiceStatusEx(schService, SC_STATUS_PROCESS_INFO, (LPBYTE)&ssStatus, sizeof(SERVICE_STATUS_PROCESS), &dwBytesNeeded))
									{
										sprintf_s(logMsg, msgSize, "QueryServiceStatusEx failed (%d)\n", GetLastError());
										AquaLogging::Log(err, logMsg);
										CloseServiceHandle(schService);
										CloseServiceHandle(schSCManager);
										retval = ERROR_STARTING_VFP;
									}
									else
									if (ssStatus.dwCurrentState == SERVICE_RUNNING)
									{
										sprintf_s(logMsg, msgSize, "Service started successfully.\n");
										AquaLogging::Log(err, logMsg);
									}
									else
									{
										sprintf_s(logMsg, msgSize, "Service not started. \n");
										AquaLogging::Log(err, logMsg);
										sprintf_s(logMsg, msgSize, "  Current State: %d\n", ssStatus.dwCurrentState);
										AquaLogging::Log(err, logMsg);
										sprintf_s(logMsg, msgSize, "  Exit Code: %d\n", ssStatus.dwWin32ExitCode);
										AquaLogging::Log(err, logMsg);
										sprintf_s(logMsg, msgSize, "  Check Point: %d\n", ssStatus.dwCheckPoint);
										AquaLogging::Log(err, logMsg);
										sprintf_s(logMsg, msgSize, "  Wait Hint: %d\n", ssStatus.dwWaitHint);
										AquaLogging::Log(err, logMsg);
										retval = ERROR_STARTING_VFP;
									}

									CloseServiceHandle(schService);
									CloseServiceHandle(schSCManager);
								}
							}
						}
					}
				}
			}
		}
		return retval;
	}


} } } } }
#endif