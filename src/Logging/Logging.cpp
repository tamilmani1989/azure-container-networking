/*++

Copyright (c) Microsoft Corporation

Module Name:

Logging.cpp

Abstract:

Logging methods.

Author:

Sushant Sharma (susharm) 13-April-2017

--*/
#include "Logging.h"
#include "../NetworkClientLibrary/Common/nclcore.h"
#ifdef _WIN32
#include "../Shared/Utilities.h"
#endif

namespace Microsoft { namespace Azure { namespace Networking {  namespace Aqua { namespace ClientLibrary {
bool AquaLogging::externalLogInitialized = false;
static ExternalLogMethod externalLogMethod;
static bool consoleOutputEnabled = false;

DllExport void AquaLogging::InitLogging(ExternalLogMethod logMethod)
{
	externalLogMethod = logMethod;	
	externalLogInitialized = true;
}
DllExport void AquaLogging::InitLogging(bool conOutputEnabled)
{
	consoleOutputEnabled = conOutputEnabled;
	externalLogInitialized = false;
}

void AquaLogging::LogNarrowString(char *narrow, bool conOutputEnabled)
{
	if (narrow == NULL)
	{
		#ifdef _WIN32
		AquariusUtilities::WriteEventLogEntry(L"LogNarrowString: Cannot log NULL");
		#endif
		return;
	}

#ifdef _WIN32
	wchar_t messageW[msgSize];
	size_t converted;
	DWORD error = mbstowcs_s(&converted, messageW, narrow, _TRUNCATE);
	if (error == 0)
	{
		AquariusUtilities::WriteEventLogEntry(messageW);
	}
	else
	{
		wchar_t szMessage[MAX_PATH];
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Error while converting narrow to wide. Error: %d.",
			AquariusUtilities::allocator);
		AquariusUtilities::WriteEventLogEntry(szMessage);
	}
#endif

	if (conOutputEnabled)
	{
		printf("%s", narrow);
	}

}

DllExport void AquaLogging::Log(int logLevel, char *logMessage)
{
	if (logMessage == NULL)
	{
		#ifdef _WIN32
		AquariusUtilities::WriteEventLogEntry(L"AquaLogging::Log: Cannot log NULL");
		#endif
		return;
	}
	if (externalLogInitialized)
	{
		if (snprintf(logMsg, msgSize, "Aqua: %s", logMessage) <0)
		{
			externalLogMethod(err, "Attach Prefix Error");
			externalLogMethod(logLevel, logMessage);
		}
		else
		{
			externalLogMethod(logLevel, logMsg);
		}
	}
	else
	{
		LogNarrowString(logMessage, consoleOutputEnabled);
	}
}

DWORD AquaLogging::ConvertWcharToChar(
	wchar_t *messageW,
	char *message)
/*++

Routine Description:

	Converts a wchar_t* to char*.

Arguments:

	messageW : wide string
	message: (out parameter) narrow string equivalent of wide string

Return Value:

	0 on success, error code otherwise

--*/
{
	size_t converted;
	size_t wideSize = sizeof(wchar_t) * (wcsnlen(messageW, msgSize) + 1);
	DWORD error = wcstombs_s(&converted, message, wideSize, messageW, _TRUNCATE);
	return error;
}

void LogWideString(
	wchar_t *wide, 
	bool conOutputEnabled)
{
	if (wide == NULL)
	{
#ifdef _WIN32
		AquariusUtilities::WriteEventLogEntry(L"LogWideString: Cannot log NULL");
#endif
		return;
	}

#ifdef _WIN32
	AquariusUtilities::WriteEventLogEntry(wide);
#endif

	if (conOutputEnabled)
	{
		printf("%ws", wide);
	}

}

DllExport void AquaLogging::Logw(int logLevel, wchar_t *logMessagew)
{
	if (logMessagew == NULL)
	{
		AquariusUtilities::WriteEventLogEntry(L"AquaLogging::Logw: Cannot log NULL message.");
		return;
	}

	char log[msgSize];
	if (_snwprintf_s(logMsgw, sizeof(wchar_t)*msgSize, _TRUNCATE, L"Aqua: %ws", logMessagew) == -1)
	{
		externalLogMethod(verbose, "Truncation happenned for log.");
	}

	if (externalLogInitialized)
	{	
		if (ConvertWcharToChar(logMsgw, log) == 0)
		{
			externalLogMethod(logLevel, log);
		}
		else
		{
			AquariusUtilities::WriteEventLogEntry(logMsgw);
			externalLogMethod(verbose, "Unable to convert to narrow string which is required to log. Please see event viewer for the message.");
		}
	}
	else
	{
		LogWideString(logMsgw, consoleOutputEnabled);
	}
}
}}}}}
