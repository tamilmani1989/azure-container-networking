/****************************** Module Header ******************************\
* Module Name:  JobManager.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#pragma once
#include <minwindef.h>

class JobObject
{
public:
	JobObject *next = nullptr;
	HANDLE value = nullptr;
};

class JobManager
{
private:
	static JobObject *jobObjectCollection;
	static JobObject *head;
	static JobObject *tail;
	
public:	
	static void EnqueueJob(wchar_t* directory, wchar_t* entryPoint);
	static void RemoveJob(wchar_t* directory, wchar_t* entryPoint);
	static void EnsureJobIsRunning(wchar_t* directory, wchar_t* entryPoint, LPTSTR lpCommandLine);
	static void SaveJobHandle(HANDLE handle);
	static void ReleaseAllJobs();
};
