/****************************** Module Header ******************************\
* Module Name:  Utilities.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Utilities.
*
\***************************************************************************/
#pragma once 
#ifdef _WIN32
#include <windows.h>
#include <strsafe.h>
#include <msxml6.h>
#define DllExport   __declspec( dllexport )  
#endif
#ifdef __linux__
#include <string.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <stdio.h>
#include <stdlib.h>
#define snprintf(arg1, len, ...) sprintf(arg1, __VA_ARGS__)
#define UNREFERENCED_PARAMETER(param) do { (void)(param); } while (0)
#define mbstowcs_s(sz, dest, src, len) mbstowcs(dest, src, len)
#define DllExport 
#endif

namespace Microsoft { namespace Azure { namespace Networking {  namespace Aqua { namespace ClientLibrary { 

typedef void(*ExternalLogMethod)(int logLevel, char* message);

class AquaLogging
{
	public:			
	static DllExport void InitLogging(ExternalLogMethod logMethod);
	static DllExport void InitLogging(bool consoleOutputEnabled);
	static DllExport void Log(int, char *);
	static const size_t msgSize = 2048;
	static bool externalLogInitialized;
};

static char logMsg[2048];
enum level
{
	info=1,
	warn,
	err,
	verbose
};

#define LOGINFO(...); snprintf(logMsg, AquaLogging::msgSize,__VA_ARGS__); AquaLogging::Log(info, logMsg);
#define LOGWARN(...); snprintf(logMsg, AquaLogging::msgSize,__VA_ARGS__); AquaLogging::Log(warn, logMsg);
#define LOGERR(...); snprintf(logMsg, AquaLogging::msgSize,__VA_ARGS__); AquaLogging::Log(err, logMsg);
#define LOGVERBOSE(...); snprintf(logMsg, AquaLogging::msgSize,__VA_ARGS__); AquaLogging::Log(verbose, logMsg);

}}}}}