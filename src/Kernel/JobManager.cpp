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
#include "../Logging/Logging.h"

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
	JOBOBJECT_BASIC_PROCESS_ID_LIST processIdList;
	JOBOBJECT_BASIC_PROCESS_ID_LIST *procList=nullptr;
	DWORD actual_len=0;
	
	// Check if the process is running with the required executable, if not, then restart the exe
	int ret = QueryInformationJobObject(handle, JobObjectBasicProcessIdList, &processIdList, sizeof(processIdList), &actual_len);

	if(ret == 0)
	{
		if (GetLastError()!=ERROR_MORE_DATA)
		{
			LOGINFOW(
			L"Unable to query information for the job object Error %d. It should not happen. values of struct %d %d %d ", 
						GetLastError(),
						processIdList.NumberOfAssignedProcesses,
						processIdList.NumberOfProcessIdsInList,
						actual_len);
			return -1;
		}

		LOGINFO("First call:Retrieved values from job numassigned %d numprocinlist %d\n",
			processIdList.NumberOfAssignedProcesses,
			processIdList.NumberOfProcessIdsInList);
			
		if(processIdList.NumberOfProcessIdsInList < processIdList.NumberOfAssignedProcesses)
		{
			DWORD blockSize = sizeof(processIdList)+(processIdList.NumberOfAssignedProcesses)*sizeof(processIdList.ProcessIdList[0]);
		
			LOGINFO("Allocate more space...values of struct NumAssigned %d NumInlist %d actual_len %d allocatedsize %d",
						processIdList.NumberOfAssignedProcesses,
						processIdList.NumberOfProcessIdsInList,
						actual_len,
						blockSize);

			procList = (JOBOBJECT_BASIC_PROCESS_ID_LIST *)malloc(blockSize);

			ret = QueryInformationJobObject(handle, JobObjectBasicProcessIdList, procList, blockSize, &actual_len);
			
			LOGINFO("Second call: return val %d Retrieved values from job num assigned %d num in list %d actual len%d  allocated size :%d\n",
				ret,
				procList->NumberOfAssignedProcesses,
				procList->NumberOfProcessIdsInList,
				actual_len,
				blockSize);

			free(procList);
		}
	}

	
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
		LOGINFOW(L"Creating process for following exe: %s args %s", *processPath, lpCommandLine);
	}
	else
	{
		LOGINFOW(L"Creating process for following exe: %s", *processPath);
	}

	if (!CreateProcess(*processPath, lpCommandLine, nullptr, nullptr, FALSE, 0, nullptr, directory, &si, &pi))
	{
		LOGINFOW(L"Create process failed for %s", *processPath);
		throw;
	}

	LOGINFOW(L"Assigning process %s to jobobject", *processPath);
	
	if (!AssignProcessToJobObject(handle, pi.hProcess))
	{
		LOGINFOW(L"Assigning process %s to jobobject failed", *processPath);
		throw;
	}
	
	LOGINFOW(L"Successfully assigned process %s to jobobject", *processPath);

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
	JOBOBJECT_EXTENDED_LIMIT_INFORMATION extendedLimit;

	int val = QueryInformationJobObject(handle, 
		JobObjectExtendedLimitInformation, 
		&extendedLimit, 
		sizeof(JOBOBJECT_EXTENDED_LIMIT_INFORMATION), 
		nullptr);

	LOGINFOW(L"return value of QueryInformationJobObject is %d  LastERror=%d", val, GetLastError());	

	if(!val) 
	{
		throw; 
	}

	extendedLimit.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_JOB_MEMORY|JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
	extendedLimit.JobMemoryLimit = memoryLimitinMB *1024*1024; // 500 MB
	
	val = SetInformationJobObject(handle, 
		JobObjectExtendedLimitInformation, 
		&extendedLimit, 
		sizeof(JOBOBJECT_EXTENDED_LIMIT_INFORMATION));
	
	LOGINFOW(L"return value of JobObjectExtendedLimitInformation is %d  LastERror=%d", val, GetLastError());

	if(!val) 
	{
		throw;
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
	LOGINFOW(L"Going to open jobobject with name %s", pluginData->entryPoint);

	HANDLE handle = OpenJobObject(JOB_OBJECT_ALL_ACCESS, FALSE, pluginData->entryPoint);
	if(handle)
	{
		int numProcess = QueryJobObject(handle);

		if(numProcess == -1)
		{
			LOGINFOW(L"QueryJobObject failed. return");
			return;
		}
		else if(pluginData->restartOnExit==0 && pluginData->numTimesStarted > 0 && numProcess == 0)
		{
			LOGINFOW(L"Won't restart short running process %s restartOnExit %d numTimesStarted %d",
				pluginData->entryPoint, pluginData->restartOnExit, pluginData->numTimesStarted);

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
			LOGINFOW(L"Plugin %s in running state", pluginData->entryPoint);
			return;
		}
	}
	else
	{
		if(pluginData->restartOnExit==0 && pluginData->numTimesStarted > 0)
		{
			LOGINFOW(L"Won't create new job object for running process %s restartOnExit %d numTimesStarted %d",
				pluginData->entryPoint, pluginData->restartOnExit, pluginData->numTimesStarted);
			return;
		}
		
		LOGINFOW(L"Going to create jobobject with name %s", pluginData->entryPoint);
		handle = CreateJobObject(NULL, pluginData->entryPoint);
	}

	LOGINFOW(L"Job object created with name %s....", pluginData->entryPoint);

	try
	{
		LOGINFOW(L"Going to apply cpu and memory limits on job object with name %s....", pluginData->entryPoint);
		ApplyLimitsOnJobObject(handle, 10, 500);
	}
	catch(...)
	{
		CloseHandle(handle);		
		throw;
	}

	wchar_t *processPath = nullptr;
	
	try
	{
		LOGINFOW(L"Going to attach process to job object with name %s....", pluginData->entryPoint);
		AttachProcessToJobObject(directory, &processPath, lpCommandLine, handle, pluginData);
	}
	catch(...)
	{
		CloseHandle(handle);		
		SAFE_DELETE(processPath);
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
	LOGINFOW(L"Calling plugin cleanup %s", pluginData->entryPoint);

	HANDLE handle = CreateJobObject(NULL, pluginData->entryPoint);
	
	if(!handle)
	{	
		return;
	}
	
	try
	{
		LOGINFOW(L"Going to apply cpu and memory limits on job object with name %s....", pluginData->entryPoint);
		ApplyLimitsOnJobObject(handle, 10, 500);
	}
	catch(wchar_t *e)
	{
		CloseHandle(handle);		
		LOGINFOW(L"ERROR::ApplyLimitsOnJobObject for plugin  %s failed with exception:%s....", pluginData->entryPoint, e);
		throw;
	}

	wchar_t *processPath = nullptr;
	
	try
	{
		LOGINFOW(L"Going to attach process to job object with name %s....", pluginData->entryPoint);
		AttachProcessToJobObject(directory, &processPath, lpCommandLine, handle, pluginData);
	}
	catch(wchar_t *e)
	{
		CloseHandle(handle);		
		SAFE_DELETE(processPath);
		LOGINFOW(L"ERROR::AttachProcessToJobObject for plugin %s failed with exception:%s....", pluginData->entryPoint, e);
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
	JOBOBJECT_BASIC_PROCESS_ID_LIST processIdList;

	if (!QueryInformationJobObject(jobHandle, JobObjectBasicProcessIdList, &processIdList, sizeof(processIdList), nullptr))
	{
		LOGINFOW(L"Unable to query information for the job object associated with job during release. It should not happen.");
		return false;
	}
	
	if (processIdList.NumberOfAssignedProcesses > 0)
	{
		LOGINFOW(L"Going to terminate process ");

		HANDLE pHandle = OpenProcess(PROCESS_ALL_ACCESS, FALSE, (DWORD)processIdList.ProcessIdList[0]);
		if(TerminateProcess(pHandle, 0) == 0)
		{
			LOGINFOW(L"Failed to terminated process. ErrorCode:%d", GetLastError());
		}
		else 
		{
			LOGINFOW(L"Successfully terminated process");
		}
	}
	else
	{	
		LOGINFOW(L"Retrieved zero processes with the job. Nothing to terminate.");
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
