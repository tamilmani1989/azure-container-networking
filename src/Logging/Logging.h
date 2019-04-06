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

#ifndef ASSERT
#define ASSERT( exp ) ((void) 0)
#endif

#include <windows.h>
#include <strsafe.h>
#include <msxml6.h>
#include <RdTrace.h>
#include <RdCommon.h>

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
	static void LogNarrowString(char *narrow, int level, bool conOutputEnabled);
	static void LogWideString(wchar_t *wide, int level, bool conOutputEnabled);
	static DWORD ConvertWcharToChar(wchar_t *messageW, char *message);

	public:			
	static DllExport void InitLogging(ExternalLogMethod logMethod);
	static DllExport void InitLogging(bool consoleOutputEnabled);
	static DllExport void Logw(int, char *msg);
	static DllExport void Log(int, char *);
	static DllExport HRESULT SetupLogging(char* logFolder, char* logPrefix);

	static const size_t msgSize = 2048;
	static bool externalLogInitialized;
};

static char logMsg[AquaLogging::msgSize];
static wchar_t logMsgw[AquaLogging::msgSize];

static char logMsg_l[AquaLogging::msgSize];
static wchar_t logMsgw_l[AquaLogging::msgSize];

enum level
{
	info=1,
	warn,
	err,
	verbose
};

#define LOGSETUP(a, b) AquaLogging::SetupLogging((char *)a, (char*)b);

#define LOGINFO(...); snprintf(logMsg_l, AquaLogging::msgSize,__VA_ARGS__); AquaLogging::Log(info, logMsg_l);
#define LOGWARN(...); snprintf(logMsg_l, AquaLogging::msgSize,__VA_ARGS__); AquaLogging::Log(warn, logMsg_l);
#define LOGERR(...); snprintf(logMsg_l, AquaLogging::msgSize,__VA_ARGS__); AquaLogging::Log(err, logMsg_l);
#define LOGVERBOSE(...); snprintf(logMsg_l, AquaLogging::msgSize,__VA_ARGS__); AquaLogging::Log(verbose, logMsg_l);

#define LOGINFOW(...); _snwprintf_s(logMsgw_l, _countof(logMsgw_l), _TRUNCATE, __VA_ARGS__); AquaLogging::Logw(info, (char *)logMsgw_l);
#define LOGWARNW(...); _snwprintf_s(logMsgw_l, _countof(logMsgw_l), _TRUNCATE,__VA_ARGS__); AquaLogging::Logw(warn, (char *)logMsgw_l);
#define LOGERRW(...); _snwprintf_s(logMsgw_l, _countof(logMsgw_l), _TRUNCATE, __VA_ARGS__); AquaLogging::Logw(err, (char *)logMsgw_l);
#define LOGVERBOSEW(...); _snwprintf_s(logMsgw_l, _countof(logMsgw_l), _TRUNCATE, __VA_ARGS__); AquaLogging::Logw(verbose, (char *)logMsgw_l);


}}}}}