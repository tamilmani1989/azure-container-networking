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
	size_t len = strlen(narrow);
	wchar_t messageW[2048];
	#ifdef _WIN32
	size_t converted;
	#endif
	mbstowcs_s(&converted, messageW, narrow, len);
        #ifdef _WIN32
	AquariusUtilities::WriteEventLogEntry(messageW);	
        #endif
	if(conOutputEnabled)
	{
		printf("%s", narrow);
	}
}

DllExport void AquaLogging::Log(int logLevel, char *logMessage)
{
	if(externalLogInitialized)
	{
		#ifdef _WIN32
		size_t msgSize = 2048;
		#endif
		snprintf(logMsg, msgSize, "Aqua: %s", logMessage);
		externalLogMethod(logLevel, logMsg);
	}
	else
	{
		LogNarrowString(logMessage, consoleOutputEnabled);
	}
}
}}}}}
