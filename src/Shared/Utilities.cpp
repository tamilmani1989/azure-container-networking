/****************************** Module Header ******************************\
* Module Name:  Utilities.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Utilities.
*
\***************************************************************************/

#include "Utilities.h"

// Name of the service
#define SERVICE_NAME             L"AquariusSvc"

int AquariusUtilities::allocator = 0;

const wchar_t* AquariusUtilities::defaultPackageStoreLocation = L"C:\\Aquarius\\Plugins";
const wchar_t * AquariusUtilities::codeDirectoryName = L"code";
const wchar_t * AquariusUtilities::configFileName = L"config.txt";
const wchar_t * AquariusUtilities::manifestFileName = L"manifest.xml";
wchar_t *AquariusUtilities::relativePathFromExe = nullptr;
//
//   FUNCTION: AquariusSvcBase::WriteEventLogEntry(PWSTR, WORD)
//
//   PURPOSE: Log a message to the Application event log.
//
//   PARAMETERS:
//   * pszMessage - string message to be logged.
//   * wType - the type of event to be logged. The parameter can be one of 
//     the following values.
//
//     EVENTLOG_SUCCESS
//     EVENTLOG_AUDIT_FAILURE
//     EVENTLOG_AUDIT_SUCCESS
//     EVENTLOG_ERROR_TYPE
//     EVENTLOG_INFORMATION_TYPE
//     EVENTLOG_WARNING_TYPE
//
void AquariusUtilities::WriteEventLogEntry(PWSTR pszMessage)
{
	WORD wType = EVENTLOG_INFORMATION_TYPE;
	HANDLE hEventSource;
	LPCWSTR lpszStrings[2] = { nullptr, nullptr };
	wchar_t svcName[MAX_PATH];

	if(AquariusUtilities::GetServiceName(svcName, sizeof(svcName)) != 0)
	{
		AquariusUtilities::WriteEventLogEntry(L"NetAgent:: Unable to retrieve service name..");
		wprintf(L"Unable to retrieve service name..");
		return;
	}

	hEventSource = RegisterEventSource(nullptr, svcName);
	if (hEventSource)
	{
		lpszStrings[0] = svcName;
		lpszStrings[1] = pszMessage;

	ReportEvent(hEventSource,  // Event log handle
			wType,                 // Event type
            0x00000001L,                     // Event category
            0x40020100L,                     // Event identifier
			nullptr,                  // No security identifier
			2,                     // Size of lpszStrings array
			0,                     // No binary data
			lpszStrings,           // Array of strings
			nullptr                   // No binary data
		);
		DeregisterEventSource(hEventSource);
	}
}

//
//   FUNCTION: AquariusSvcBase::WriteErrorLogEntry(PWSTR, DWORD)
//
//   PURPOSE: Log an error message to the Application event log.
//
//   PARAMETERS:
//   * pszFunction - the function that gives the error
//   * dwError - the error code
//
void AquariusUtilities::WriteErrorLogEntry(PWSTR pszFunction)
{
	WORD wType = EVENTLOG_ERROR_TYPE;
	wchar_t szMessage[MAX_PATH];
	wchar_t svcName[MAX_PATH];

	StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
		L"%s failed w/err 0x%08lx", pszFunction, wType);

	HANDLE hEventSource;
	LPCWSTR lpszStrings[2] = { nullptr, nullptr };
	
	if(AquariusUtilities::GetServiceName(svcName, sizeof(svcName)) != 0)
	{
		AquariusUtilities::WriteEventLogEntry(L"NetAgent:: Unable to retrieve service name..");
		wprintf(L"Unable to retrieve service name..");
		return;
	}

	hEventSource = RegisterEventSource(nullptr, svcName);
	if (hEventSource)
	{
		lpszStrings[0] = svcName;
		lpszStrings[1] = szMessage;

		ReportEvent(hEventSource,  // Event log handle
			wType,                 // Event type
            0x00000001L,                     // Event category
            0x40020100L,                     // Event identifier
			nullptr,                  // No security identifier
			2,                     // Size of lpszStrings array
			0,                     // No binary data
			lpszStrings,           // Array of strings
			nullptr                   // No binary data
		);

		DeregisterEventSource(hEventSource);
	}
}

void AquariusUtilities::ReleasePluginMetadata(int count, PluginMetadata * pluginsMetadata)
{
	if (pluginsMetadata == nullptr)
	{
		AquariusUtilities::WriteErrorLogEntry((PWSTR)("Received a request to free null pluginMetadata"));
		return;
	}

    wchar_t szMessage[MAX_PATH];

	for (int i = 0; i<count; i++)
	{
		if (pluginsMetadata[i].location != nullptr) 
		{
            StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
                L"Going to Release location %s", pluginsMetadata[i].location);
            WriteEventLogEntry(szMessage);
			delete [] pluginsMetadata[i].location;
			pluginsMetadata[i].location = nullptr;
			AquariusUtilities::allocator--;
		}

		if (pluginsMetadata[i].currentVersion != nullptr) 
		{
            StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
                L"Going to Release version %s", pluginsMetadata[i].currentVersion);
            WriteEventLogEntry(szMessage);
			delete [] pluginsMetadata[i].currentVersion;
			pluginsMetadata[i].currentVersion = nullptr;
			AquariusUtilities::allocator--;
		}

		if (pluginsMetadata[i].entryPoint != nullptr) 
		{
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
                L"Going to Release entrypoint %s", pluginsMetadata[i].entryPoint);
            WriteEventLogEntry(szMessage);
			delete[] pluginsMetadata[i].entryPoint;
			pluginsMetadata[i].entryPoint = nullptr;
		}
	}
}



void AquariusUtilities::PopulateServiceExeLocation(wchar_t* packageStoreLocation)
{
	wchar_t szMessage[MAX_PATH];
	
	if (GetModuleFileName(nullptr, packageStoreLocation, MAX_PATH) == 0)
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
			L"Failed to get location of the plugins w/err 0x%08lx Returning %s\n",
			GetLastError(), AquariusUtilities::defaultPackageStoreLocation);
		AquariusUtilities::WriteEventLogEntry(szMessage);
		wcscpy_s(packageStoreLocation, 
			wcsnlen(defaultPackageStoreLocation, MAX_PATH) + 1, 
			defaultPackageStoreLocation);
		return;
	}

	size_t i = wcsnlen(packageStoreLocation, MAX_PATH);
	wchar_t c = '\0';
	do
	{
		// Review: i - 1 isn't safe in the first iteration of the loop
		// needs error handling for 0 length string: wcsnlen does not include
		// the null terminator
		c = packageStoreLocation[i - 1];
		packageStoreLocation[i - 1] = packageStoreLocation[i];
		i--;
	} while (c != '\\' && i > 0);
}

void AquariusUtilities::PopulatePluginLocation(wchar_t* packageStoreLocation)
{
	PopulateServiceExeLocation(packageStoreLocation);
	wcscat_s(packageStoreLocation, MAX_PATH, relativePathFromExe);
}

void AquariusUtilities::SetRelativePathFromExe(wchar_t *relPath)
{
	size_t len = wcslen(relPath)+1;
	relativePathFromExe = new wchar_t[len];
	wcscpy_s(relativePathFromExe, len, relPath);
}

int AquariusUtilities::GetServiceName(wchar_t *svcName, size_t size)
{
	int i = 0;
	int ret = 1;

	if(GetModuleBaseName(GetCurrentProcess(), GetModuleHandle(NULL), svcName, (DWORD)size) == 0)
	{
		AquariusUtilities::WriteEventLogEntry(L"Error while retrieving process name");     
		wprintf(L"Error while retrieving process name:%d\n", GetLastError());
		return ret;
	}

	while(svcName[i] != '.' && svcName[i] != '\0')
	{
		i++;
	}		

	if(svcName[i] == '\0')
	{
		return ret;
	}

	svcName[i]='\0';

	
	if(wcscat_s(svcName, size, L"Svc") != 0)
	{
		AquariusUtilities::WriteEventLogEntry(L"wcscat_s failed while creating service name");     
		wprintf(L"wcscat_s failed while creating service name:%d\n", GetLastError());
		return ret;
	}

	return 0;
}