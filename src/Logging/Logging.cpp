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

DWORD convertcharToWchar(
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
	size_t wideSize = sizeof(wchar_t) * strlen(message) + 1;
	DWORD error = mbstowcs_s(&converted, messageW, wideSize, message, _TRUNCATE);
	return error;
}


DllExport HRESULT AquaLogging::SetupLogging(
	char* logFolder, 
	char* logPrefix
)
/*++

Routine Description:

    Helper to set up logging.

Arguments:

    None.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    HRESULT hr = S_OK;
	wchar_t *wLogPrefix = (wchar_t *)logPrefix;
	wchar_t *wLogFolder = (wchar_t *)logFolder;
    wchar_t *logFilePathWithPrefix = NULL;

	wprintf(L"Logfolder: %s LogPrefix: %s\n", wLogFolder, wLogPrefix);
	size_t finalLen = wcslen(wLogFolder) + wcslen(L"\\") + wcslen(wLogPrefix) + 1;
	
	logFilePathWithPrefix = new wchar_t[finalLen];
	if(!logFilePathWithPrefix)
	{
		wprintf(L"Not able to allocate memory for logfilewithprefix.\n");
		hr = HRESULT_FROM_WIN32(ERROR_INSUFFICIENT_BUFFER);
		goto Cleanup;
	}

	if(swprintf_s(logFilePathWithPrefix, finalLen, L"%s\\%s", wLogFolder, wLogPrefix) < 0)
	{
		wprintf(L"swprintf_s failed inside setup logging\n");
		hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
	    goto Cleanup;
	}

    hr = RD_TRACE_INITIALIZE_CUSTOM_SIZE(RD_TRACE_OUTPUT_FILE,
        RD_TRACE_ALL,
        logFilePathWithPrefix,
        NULL,
        0,
        NULL,
        RD_TRACE_ALL,
        10,
        RD_TRACE_LOG_LIMIT);

Cleanup:
	
	if(logFilePathWithPrefix)
	{
		delete[] logFilePathWithPrefix;
	}
	
    return hr;
}

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

void AquaLogging::LogNarrowString(char *narrow, int logLevel, bool conOutputEnabled)
{
	if (narrow == NULL)
	{
		#ifdef _WIN32
		wprintf(L"LogNarrowString: Cannot log NULL");
		#endif
		return;
	}

#ifdef _WIN32

	int level = RD_TRACE_INFO;

	switch(logLevel)
	{
		case warn:
		{
			level = RD_TRACE_WARNING;
		}

		case err:
		{
			level = RD_TRACE_ERROR;
		}

		case verbose:
		{
			level = RD_TRACE_VERBOSE;
		}
	}
	
	RD_TRACE_LOG((level,L"%hs", narrow));
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
		wprintf(L"AquaLogging::Log: Cannot log NULL");
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
		LogNarrowString(logMessage, logLevel, consoleOutputEnabled);
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

void AquaLogging::LogWideString(
	wchar_t *wide, 
	int level,
	bool conOutputEnabled)
{
	if (wide == NULL)
	{
		#ifdef _WIN32
		wprintf(L"LogWideString: Cannot log NULL");
		#endif
		return;	
	}
#ifdef _WIN32
	int rd_level = RD_TRACE_INFO;

	switch(level)
	{
		case warn:
		{
			rd_level = RD_TRACE_WARNING;
		}

		case err:
		{
			rd_level = RD_TRACE_ERROR;
		}

		case verbose:
		{
			rd_level = RD_TRACE_VERBOSE;
		}
	}

	RD_TRACE_LOG((rd_level, L"%s", wide));
#endif

	if (conOutputEnabled)
	{
		printf("%ws", wide);
	}

}

DllExport void AquaLogging::Logw(int logLevel, char *logMessage)
{
	wchar_t *logMessagew = (wchar_t *)logMessage;

	if (logMessagew == NULL)
	{
		wprintf(L"AquaLogging::Logw: Cannot log NULL message.");
		return;
	}

	char log[msgSize];
	if (_snwprintf_s(logMsgw, _countof(logMsgw), _TRUNCATE, L"%ws", logMessagew) == -1)
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
			externalLogMethod(verbose, "Unable to convert to narrow string which is required to log. Please see event viewer for the message.");
		}
	}
	else
	{
		LogWideString(logMsgw, logLevel, consoleOutputEnabled);
	}
}
}}}}}
