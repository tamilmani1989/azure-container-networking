/****************************** Module Header ******************************\
* Module Name:  Logging.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Utilities.
*
\*********/

#include "Logging.h"
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

void LogNarrowString(char *narrow, bool conOutputEnabled)
{
	if(narrow == NULL)
	{
		AquariusUtilities::WriteEventLogEntry(L"LogNarrowString: Cannot log NULL");
		return;
	}
	
	#ifdef _WIN32
	wchar_t messageW[2048];
	size_t converted;	
	DWORD error = mbstowcs_s(&converted, messageW, narrow, _TRUNCATE);	
	if (error != 0)
	{
		AquariusUtilities::WriteEventLogEntry(messageW);
	}
	else
	{
		AquariusUtilities::WriteEventLogEntry(L"Error while converting narrow to wide.");
	}
	#endif

	if (conOutputEnabled)
	{
		printf("%s", narrow);
	}
	
}

DllExport void AquaLogging::Log(int logLevel, char *logMessage)
{
	if(logMessage == NULL)
	{
		AquariusUtilities::WriteEventLogEntry(L"AquaLogging::Log: Cannot log NULL");
		return;
	}
	if(externalLogInitialized)
	{
		#ifdef _WIN32
		size_t msgSize = 2048;
		#endif
		if(snprintf(logMsg, msgSize, "Aqua: %s", logMessage) <0)
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
}}}}}
