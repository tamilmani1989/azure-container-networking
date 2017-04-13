/****************************** Module Header ******************************\
* Module Name:  Logging.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Utilities.
*
\*********/

#include "Logging.h"
#include "../Shared/Utilities.h"

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
	size_t converted;
	mbstowcs_s(&converted, messageW, narrow, len);
	AquariusUtilities::WriteEventLogEntry(messageW);	
	if(conOutputEnabled)
	{
		printf(narrow);
	}
}

void AquaLogging::Log(int logLevel, char *logMessage)
{
	UNREFERENCED_PARAMETER(logLevel);
	if(externalLogInitialized)
	{
		char logMsg[2048];
		size_t msgSize = 2048;
		sprintf_s(logMsg, msgSize, "Aqua: %s", logMessage);
		externalLogMethod(logLevel, logMsg);
	}
	else
	{
		LogNarrowString(logMessage, consoleOutputEnabled);
	}
}

