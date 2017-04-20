/****************************** Module Header ******************************\
* Module Name:  Utilities.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Utilities.
*
\***************************************************************************/
#ifdef _WIN32
#include <windows.h>
#include <strsafe.h>
#include <msxml6.h>
#define info 1
#endif
#ifdef __linux__
#include <string.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <stdio.h>
#include <stdlib.h>
#define sprintf_s(arg1, len, ...) sprintf(arg1, __VA_ARGS__)
#define UNREFERENCED_PARAMETER(param) do { (void)(param); } while (0)
#define mbstowcs_s(sz, dest, src, len) mbstowcs(dest, src, len)
#endif
#define warn 2
#define err 3
#define verbose 4

typedef void(*ExternalLogMethod)(int logLevel, char* message);

class AquaLogging
{
	static bool externalLogInitialized;	
	public:			
	static void InitLogging(ExternalLogMethod logMethod);
	static void InitLogging(bool consoleOutputEnabled);
	static void Log(int, char *);	
};
