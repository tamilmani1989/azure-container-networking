/****************************** Module Header ******************************\
* Module Name:  JobManager.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
*
\***************************************************************************/

#include "JobManager.h"
#include <handleapi.h>
#include "../Shared/Utilities.h"

 JobObject *JobManager::jobObjectCollection = nullptr;
 JobObject *JobManager::head = nullptr;
 JobObject *JobManager::tail = nullptr;

void JobManager::EnqueueJob(wchar_t* directory, wchar_t* entryPoint)
{
}

void JobManager::RemoveJob(wchar_t* directory, wchar_t* entryPoint)
{

}

HANDLE GetOrCreateJobObject(wchar_t *entryPoint)
{
	wchar_t szMessage[260];
	HANDLE handle = CreateJobObject(nullptr, entryPoint);
	if (handle == nullptr)
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"nullptr received while creating jobobject with name %s", entryPoint);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		return nullptr;
	}
	if (GetLastError() == ERROR_ALREADY_EXISTS)
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Jobobject already exists with name %s", entryPoint);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		// Check if the process is running with the required executable, if not, then restart the exe
		JOBOBJECT_BASIC_PROCESS_ID_LIST processIdList;
		bool shouldReturn = TRUE;
		if (!QueryInformationJobObject(handle, JobObjectBasicProcessIdList, &processIdList, sizeof(processIdList), nullptr))
		{
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
				L"Unable to query information for the job object associated with jobName->%s. It should not happen.", entryPoint);
			AquariusUtilities::WriteEventLogEntry(szMessage);
			return nullptr;
		}

		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Retrieved values from job %d %d %d",
			processIdList.NumberOfAssignedProcesses,
			processIdList.NumberOfProcessIdsInList,
			processIdList.ProcessIdList[0]);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		if (processIdList.NumberOfAssignedProcesses == 0)
		{
			// some how the process got killed

			StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
				L"Could not find the process associated with the job object named %s. Going to recreate the process.", entryPoint);
			AquariusUtilities::WriteEventLogEntry(szMessage);
			shouldReturn = FALSE;
		}

		if (shouldReturn)
		{
			return nullptr;
		}
	}
	return handle;
}

void AttachProcessToJobObject(wchar_t* directory, wchar_t* entryPoint, wchar_t **processPath, LPTSTR lpCommandLine, HANDLE handle)
{
	wchar_t szMessage[260];
	size_t len = wcsnlen(directory, MAX_PATH) + wcsnlen(L"\\", 2) + wcsnlen(entryPoint, MAX_PATH) + 1;
	(*processPath) = new wchar_t[len];
	wcscpy_s(*processPath, len, directory);
	wcscat_s(*processPath, len, L"\\");
	wcscat_s(*processPath, len, entryPoint);

	STARTUPINFO si;
	PROCESS_INFORMATION pi;

	ZeroMemory(&si, sizeof(si));
	si.cb = sizeof(si);
	ZeroMemory(&pi, sizeof(pi));
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
		L"Creating process for following exe: %s", *processPath);
	AquariusUtilities::WriteEventLogEntry(szMessage);

	if (!CreateProcess(*processPath, lpCommandLine, nullptr, nullptr, FALSE, 0, nullptr, directory, &si, &pi))
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Create process failed for %s", *processPath);
		throw szMessage;
	}

	StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
		L"Assigning process %s to jobobject: %d", *processPath);
	AquariusUtilities::WriteEventLogEntry(szMessage);
	if (!AssignProcessToJobObject(handle, pi.hProcess))
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Assigning process %s to jobobject failed", *processPath);
		throw szMessage;
	}
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
		L"Successfully assigned process %s to jobobject", *processPath);
	AquariusUtilities::WriteEventLogEntry(szMessage);
}

void ApplyLimitsOnJobObject(HANDLE handle, int cpuPercentageLimit, int memoryLimitinMB)
{	
	// limit cpu usage
	JOBOBJECT_CPU_RATE_CONTROL_INFORMATION cpuLimits =
					{ JOB_OBJECT_CPU_RATE_CONTROL_ENABLE | JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP, cpuPercentageLimit * 100 }; // 20%
	SetInformationJobObject(handle, 
		JobObjectCpuRateControlInformation, 
		&cpuLimits, 
		sizeof(JOBOBJECT_CPU_RATE_CONTROL_INFORMATION));
	
	// limit memory usage
	wchar_t szMessage[260];
	JOBOBJECT_EXTENDED_LIMIT_INFORMATION extendedLimit;

	int val = QueryInformationJobObject(handle, 
		JobObjectExtendedLimitInformation, 
		&extendedLimit, 
		sizeof(JOBOBJECT_EXTENDED_LIMIT_INFORMATION), 
		nullptr);
	StringCchPrintf(
		szMessage, ARRAYSIZE(szMessage), 
		L"return value of QueryInformationJobObject is %d  LastERror=%d", 
		val, GetLastError());	
	AquariusUtilities::WriteEventLogEntry(szMessage);

	extendedLimit.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_JOB_MEMORY;
	extendedLimit.JobMemoryLimit = memoryLimitinMB *1024*1024; // 500 MB
	val = SetInformationJobObject(handle, 
		JobObjectExtendedLimitInformation, 
		&extendedLimit, 
		sizeof(JOBOBJECT_EXTENDED_LIMIT_INFORMATION));
	
	StringCchPrintf(
		szMessage, ARRAYSIZE(szMessage), 
		L"return value of JobObjectExtendedLimitInformation is %d  LastERror=%d", 
		val, GetLastError());
	AquariusUtilities::WriteEventLogEntry(szMessage);
}

void JobManager::EnsureJobIsRunning(wchar_t* directory, wchar_t* entryPoint, LPTSTR lpCommandLine)
{		
	wchar_t szMessage[260];
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Going to create a jobobject with name %s", entryPoint);
	AquariusUtilities::WriteEventLogEntry(szMessage);

	HANDLE handle = GetOrCreateJobObject(entryPoint);
	if(handle == nullptr)
	{
		return;
	}	

	StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
		L"Job object created with name %s....", entryPoint);
	AquariusUtilities::WriteEventLogEntry(szMessage);

	wchar_t *processPath = nullptr;

	try
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Going to apply cpu and memory limits on job object with name %s....", entryPoint);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		ApplyLimitsOnJobObject(handle, 10, 500);
				
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Going to attach process to job object with name %s....", entryPoint);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		AttachProcessToJobObject(directory, entryPoint, &processPath, lpCommandLine, handle);
	}
	catch(...)
	{
		CloseHandle(handle);		
		SAFE_DELETE(processPath);
		throw;
	}
	SAFE_DELETE(processPath);
	SaveJobHandle(handle);	
}

static void InsertAtTail(JobObject** tail, HANDLE handle)
{
	JobObject *jo = new JobObject;
	jo->value = handle;
	jo->next = nullptr;
	(*tail)->next = jo;
	(*tail) = jo;
}

void JobManager::SaveJobHandle(HANDLE handle)
{
	if (jobObjectCollection == nullptr)
	{
		jobObjectCollection = new JobObject;
		jobObjectCollection->value = handle;
		jobObjectCollection->next = nullptr;
		head = jobObjectCollection;
		tail = jobObjectCollection;
	}
	else 
	{
		InsertAtTail(&tail, handle);
	}
}


bool TerminateProcessInJob(HANDLE jobHandle)
{
	wchar_t szMessage[260];
	JOBOBJECT_BASIC_PROCESS_ID_LIST processIdList;
	if (!QueryInformationJobObject(jobHandle, JobObjectBasicProcessIdList, &processIdList, sizeof(processIdList), nullptr))
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Unable to query information for the job object associated with job during release. It should not happen.");
		AquariusUtilities::WriteEventLogEntry(szMessage);
		return false;
	}
	
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
		L"Retrieved process Id %d from the job.",
		processIdList.ProcessIdList[0]);
	AquariusUtilities::WriteEventLogEntry(szMessage);

	if (processIdList.NumberOfAssignedProcesses > 0)
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Going to terminate process Id %d.",
			processIdList.ProcessIdList[0]);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		HANDLE pHandle = OpenProcess(PROCESS_ALL_ACCESS, FALSE, (DWORD)processIdList.ProcessIdList[0]);
		if(TerminateProcess(pHandle, 0) == 0)
		{
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
				L"Failed to terminated process Id %d. ErrorCode:%d",
				processIdList.ProcessIdList[0], GetLastError());
			AquariusUtilities::WriteEventLogEntry(szMessage);				
		}
		else 
		{
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
				L"Successfully terminated process Id %d.",
				processIdList.ProcessIdList[0]);
			AquariusUtilities::WriteEventLogEntry(szMessage);
		}
	}
	else
	{	
		AquariusUtilities::WriteEventLogEntry(L"Retrieved zero processes with the job. Nothing to terminate.");
	}
    return true;
}

static void Delete(JobObject *list)
{
	if (list != nullptr) {
		Delete(list->next);
        if (TerminateProcessInJob(list->value)) 
        {
            CloseHandle(list->value);
        }
		delete list;
	}
}

// Call this method within the lock
void JobManager::ReleaseAllJobs()
{
	if(jobObjectCollection!=nullptr)
	{
		Delete(jobObjectCollection);
		jobObjectCollection = nullptr;
	}
}
