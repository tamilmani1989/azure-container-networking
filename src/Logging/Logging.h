/****************************** Module Header ******************************\
* Module Name:  Utilities.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Utilities.
*
\***************************************************************************/

#include <windows.h>
#include <strsafe.h>
#include <msxml6.h>

typedef void(*ExternalLogMethod)(int logLevel, char* message);

class AquaLogging
{
	static bool externalLogInitialized;	
	public:			
	static void InitLogging(ExternalLogMethod logMethod);
	static void InitLogging(bool consoleOutputEnabled);
	static void Log(int, char *);	
};


#define info 1
#define warn 2
#define err 3
#define verbose 4
