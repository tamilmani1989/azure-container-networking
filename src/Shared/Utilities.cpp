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

	hEventSource = RegisterEventSource(nullptr, SERVICE_NAME);
	if (hEventSource)
	{
		lpszStrings[0] = SERVICE_NAME;
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
	wchar_t szMessage[260];
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
		L"%s failed w/err 0x%08lx", pszFunction, wType);

	HANDLE hEventSource;
	LPCWSTR lpszStrings[2] = { nullptr, nullptr };

	hEventSource = RegisterEventSource(nullptr, SERVICE_NAME);
	if (hEventSource)
	{
		lpszStrings[0] = SERVICE_NAME;
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

    wchar_t szMessage[260];

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
	}
	delete[] pluginsMetadata;
	pluginsMetadata = nullptr;
	AquariusUtilities::allocator--;
}



void AquariusUtilities::PopulateServiceExeLocation(wchar_t* packageStoreLocation)
{
	wchar_t szMessage[260];
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

	int i = wcsnlen(packageStoreLocation, MAX_PATH);
	wchar_t c = '\0';
	do
	{
		c = packageStoreLocation[i - 1];
		packageStoreLocation[i - 1] = packageStoreLocation[i];
		i--;
	} while (c != '\\' && i > 0);
}

void AquariusUtilities::PopulatePluginLocation(wchar_t* packageStoreLocation)
{
	PopulateServiceExeLocation(packageStoreLocation);
	wcscat_s(packageStoreLocation, MAX_PATH, L"\\Plugins");
}