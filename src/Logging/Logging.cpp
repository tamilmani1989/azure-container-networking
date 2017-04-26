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
bool AquaLogging::externalLogInitialized = false;
static ExternalLogMethod externalLogMethod;
static bool consoleOutputEnabled = false;

void AquaLogging::InitLogging(ExternalLogMethod logMethod)
{
	externalLogMethod = logMethod;	
	externalLogInitialized = true;
}
void AquaLogging::InitLogging(bool conOutputEnabled)
{
	consoleOutputEnabled = conOutputEnabled;
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

void AquaLogging::Log(int logLevel, char *logMessage)
{
	UNREFERENCED_PARAMETER(logLevel);
	if(externalLogInitialized)
	{
		char logMsg[2048];
		#ifdef _WIN32
		size_t msgSize = 2048;
		#endif
		sprintf_s(logMsg, msgSize, "Aqua: %s", logMessage);
		externalLogMethod(logLevel, logMsg);
	}
	else
	{
		LogNarrowString(logMessage, consoleOutputEnabled);
	}
}

