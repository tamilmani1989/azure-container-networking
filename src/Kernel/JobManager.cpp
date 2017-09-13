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
#include "Logging.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

 JobObject *JobManager::jobObjectCollection = nullptr;
 JobObject *JobManager::head = nullptr;
 JobObject *JobManager::tail = nullptr;

void JobManager::EnqueueJob(wchar_t* directory, wchar_t* entryPoint)
{
	UNREFERENCED_PARAMETER(directory);
	UNREFERENCED_PARAMETER(entryPoint);
}

void JobManager::RemoveJob(wchar_t* directory, wchar_t* entryPoint)
{
		UNREFERENCED_PARAMETER(directory);
		UNREFERENCED_PARAMETER(entryPoint);
}


int QueryJobObject(HANDLE handle)
/*++

Routine Description:
	This function returns the job object if it exists otherwise creates a new job object

Arguments:
	pluginData - Contains information about plugin

Return Value:
	returns job handle on successful case else returns NULL

--*/
{

	wchar_t szMessage[MAX_PATH];
	JOBOBJECT_BASIC_PROCESS_ID_LIST processIdList;
	JOBOBJECT_BASIC_PROCESS_ID_LIST *procList=nullptr;
	DWORD actual_len=0;
	
	// Check if the process is running with the required executable, if not, then restart the exe
	int ret = QueryInformationJobObject(handle, JobObjectBasicProcessIdList, &processIdList, sizeof(processIdList), &actual_len);
	
	while(ret == 0)
	{
		if (GetLastError()!=ERROR_MORE_DATA)
		{
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Unable to query information for the job object Error %d. It should not happen. values of struct %d %d %d %d ", 
						GetLastError(),
						processIdList.NumberOfAssignedProcesses,
						processIdList.NumberOfProcessIdsInList,
						processIdList.ProcessIdList[0],
						actual_len);
			AquariusUtilities::WriteEventLogEntry(szMessage);
			return 1;
		}
		
		DWORD blockSize = sizeof(processIdList)+(processIdList.NumberOfAssignedProcesses)*sizeof(ULONG);
		
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Allocate more space...values of struct %d %d %d %d %d %d",
						GetLastError(),
						processIdList.NumberOfAssignedProcesses,
						processIdList.NumberOfProcessIdsInList,
						processIdList.ProcessIdList[0],
						actual_len,
						blockSize);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		if(!procList)
		{
			free(procList);
			procList = (JOBOBJECT_BASIC_PROCESS_ID_LIST *)malloc(blockSize);
		}

		processIdList.NumberOfProcessIdsInList=processIdList.NumberOfAssignedProcesses+1;
		ret = QueryInformationJobObject(handle, JobObjectBasicProcessIdList, procList, blockSize, &actual_len);
	}

	LOGINFO("Retrieved values from job %d %d\n",
			processIdList.NumberOfAssignedProcesses,
			processIdList.NumberOfProcessIdsInList);


	return processIdList.NumberOfAssignedProcesses;
}


void AttachProcessToJobObject(wchar_t* directory, wchar_t **processPath, LPTSTR lpCommandLine, HANDLE handle, PluginMetadata *pluginData)
/*++

Routine Description:
	This function attaches process to job object

Arguments:
	directory - directory of exe
	processPath - exe file location
	lpCommandLine - command line to be executed
	handle - job object handle
	pluginData - object contains info about plugin

Return Value:

--*/
{
	wchar_t szMessage[MAX_PATH];
	STARTUPINFO si;
	PROCESS_INFORMATION pi;
	size_t len = 0;
	
	len = wcsnlen(directory, MAX_PATH) + wcsnlen(L"\\", 2) + wcsnlen(pluginData->entryPoint, MAX_PATH) + 2*wcslen(L"\"") +1;
	(*processPath) = new wchar_t[len];

	if(swprintf_s(*processPath, len, L"%s\\%s", directory, pluginData->entryPoint) < 0)
	{
		LOGINFO("swprintf failed while constructing process path");
		return;
	}

	ZeroMemory(&si, sizeof(si));
	si.cb = sizeof(si);
	ZeroMemory(&pi, sizeof(pi));

	if(pluginData->arguments)
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Creating process for following exe: %s args %s", *processPath, lpCommandLine);
	}
	else
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Creating process for following exe: %s", *processPath);
	}

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

	pluginData->processHandle =  pi.hProcess;
}

void ApplyLimitsOnJobObject(HANDLE handle, int cpuPercentageLimit, int memoryLimitinMB)
/*++

Routine Description:
	This function applies CPU and memory limits on job object

Arguments:
	handle - job object handle
	cpuPercentageLimit  -  CPU limit
	memoryLimitinMB  -  memory limit in MB

Return Value:

--*/
{	
	// limit cpu usage
	JOBOBJECT_CPU_RATE_CONTROL_INFORMATION cpuLimits =
					{ JOB_OBJECT_CPU_RATE_CONTROL_ENABLE | JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP, DWORD(cpuPercentageLimit) * 100 }; // 20%

	SetInformationJobObject(handle, 
		JobObjectCpuRateControlInformation, 
		&cpuLimits, 
		sizeof(JOBOBJECT_CPU_RATE_CONTROL_INFORMATION));
	
	// limit memory usage
	wchar_t szMessage[MAX_PATH];
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

	if(!val) 
	{
		throw szMessage; 
	}

	extendedLimit.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_JOB_MEMORY|JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
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

	if(!val) 
	{
		throw szMessage;
	}

}

void JobManager::EnsureJobIsRunning(wchar_t* directory, LPTSTR lpCommandLine, PluginMetadata *pluginData)
/*++

Routine Description:
	This function gets job object and attach process to job object. This function also makes sure that long running process restarted
	even if it exits

Arguments:
	directory  - Directory of Plugin exe
	lpCommandLine - Command line to be executed
	pluginData - Contains information about plugin

Return Value:

--*/
{		
	wchar_t szMessage[MAX_PATH];
	
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Going to create a jobobject with name %s", pluginData->entryPoint);
	AquariusUtilities::WriteEventLogEntry(szMessage);
	
	HANDLE handle = OpenJobObject(JOB_OBJECT_ALL_ACCESS, FALSE, pluginData->entryPoint);
	if(handle)
	{
		int numProcess = QueryJobObject(handle);

		if(pluginData->restartOnExit==0 && pluginData->numTimesStarted > 0 && numProcess == 0)
		{
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage),L"Won't restart short running process %s restartOnExit %d numTimesStarted %d",
				pluginData->entryPoint, pluginData->restartOnExit, pluginData->numTimesStarted);
			AquariusUtilities::WriteEventLogEntry(szMessage);

			if(pluginData->processHandle!=NULL)
			{
				DWORD dwExitCode = NULL;
				GetExitCodeProcess(pluginData->processHandle, &dwExitCode);
				pluginData->exitStatus = (int)dwExitCode;
				pluginData->processState = STOPPED;
			}

			TerminateJobObject(handle, 0);
			CloseHandle(handle);
			return;
		}
		else if (numProcess != 0)
		{
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage),L"Plugin %s in running state", pluginData->entryPoint);
			AquariusUtilities::WriteEventLogEntry(szMessage);
			return;
		}
	}
	else
	{
		if(pluginData->restartOnExit==0 && pluginData->numTimesStarted > 0)
		{
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage),L"Won't create new job object for running process %s restartOnExit %d numTimesStarted %d",
				pluginData->entryPoint, pluginData->restartOnExit, pluginData->numTimesStarted);
			AquariusUtilities::WriteEventLogEntry(szMessage);
			return;
		}
		
		handle = CreateJobObject(NULL, pluginData->entryPoint);
	}

	StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
		L"Job object created with name %s....", pluginData->entryPoint);
	AquariusUtilities::WriteEventLogEntry(szMessage);

	try
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Going to apply cpu and memory limits on job object with name %s....", pluginData->entryPoint);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		ApplyLimitsOnJobObject(handle, 10, 500);
	}
	catch(wchar_t *e)
	{
		CloseHandle(handle);		
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"ERROR::ApplyLimitsOnJobObject for plugin  %s failed with exception:%s....", pluginData->entryPoint, e);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		throw;
	}

	wchar_t *processPath = nullptr;
	
	try
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Going to attach process to job object with name %s....", pluginData->entryPoint);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		AttachProcessToJobObject(directory, &processPath, lpCommandLine, handle, pluginData);
	}
	catch(wchar_t *e)
	{
		CloseHandle(handle);		
		SAFE_DELETE(processPath);
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"ERROR::AttachProcessToJobObject for plugin %s failed with exception:%s....", pluginData->entryPoint, e);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		throw;
	}
	
	SAFE_DELETE(processPath);
	SaveJobHandle(handle);	
	pluginData->processState = RUNNING;
	pluginData->numTimesStarted++;
	pluginData->exitStatus=0;
}

void JobManager::PluginCleanup(wchar_t* directory, LPTSTR lpCommandLine, PluginMetadata *pluginData)
{
	wchar_t szMessage[MAX_PATH];

	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Calling plugin cleanup %s", pluginData->entryPoint);
	AquariusUtilities::WriteEventLogEntry(szMessage);

	HANDLE handle = CreateJobObject(NULL, pluginData->entryPoint);
	
	if(!handle)
	{	
		return;
	}
	
	try
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Going to apply cpu and memory limits on job object with name %s....", pluginData->entryPoint);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		ApplyLimitsOnJobObject(handle, 10, 500);
	}
	catch(wchar_t *e)
	{
		CloseHandle(handle);		
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"ERROR::ApplyLimitsOnJobObject for plugin  %s failed with exception:%s....", pluginData->entryPoint, e);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		throw;
	}

	wchar_t *processPath = nullptr;
	
	try
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Going to attach process to job object with name %s....", pluginData->entryPoint);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		AttachProcessToJobObject(directory, &processPath, lpCommandLine, handle, pluginData);
	}
	catch(wchar_t *e)
	{
		CloseHandle(handle);		
		SAFE_DELETE(processPath);
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"ERROR::AttachProcessToJobObject for plugin %s failed with exception:%s....", pluginData->entryPoint, e);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		throw;
	}
	
	pluginData->cleanup = false;
	SAFE_DELETE(processPath);
	
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
	wchar_t szMessage[MAX_PATH];
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
