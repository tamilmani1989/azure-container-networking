/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    HostNetAgent.h

Abstract:

    Header to define the HostNetAgent functions.

Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//
#pragma once

#include <stdio.h>
#include <stdlib.h>
#include <Windows.h>
#ifdef _WIN32
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>
#include <wchar.h>
#include <regex> 
#endif
#ifdef __WFILE__
#undef  __WFILE__
#endif // __WFILE__

#ifdef __WFUNCTION__
#undef  __WFUNCTION__
#endif // __WFUNCTION__

#include <cpprest/http_listener.h>
#include <cpprest/uri.h>
#include <cpprest/asyncrt_utils.h>
#include <cpprest/json.h>
#include <cpprest/filestream.h>
#include <cpprest/containerstream.h>
#include <cpprest/producerconsumerstream.h>
#include "cpprest/filestream.h"
#include "PackagesInfo.h"
#include "GenevaLogger.h"
#include "HealthReport.h"
#include "CNIReport.h"
#include "MonAgent.h"

using namespace std;
using namespace web;
using namespace http;
using namespace web::http::experimental::listener;
using namespace concurrency::streams;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;


#define DIRECTORY_SEPARATOR (L'\\')
#define DEFAULT_LOG_LOCATION  L"C:\\Logs\\HostNetAgent"
#define PACKAGE_FOLDER_NAME L"Package"
#define CONFIG_FILE_CMDLINE_FLAG   L"/ConfigFile"
#define RESOURCE_PATH_CMDLINE_FLAG L"/ResourcePath"
#define LOG_PATH_CMDLINE_FLAG   L"/LogPath"
#define PACKAGE_PATH_CMDLINE_FLAG L"/PackagePath"
#define CONTAINER_SEGMENT L"container"

const wstring POST_CNI_REPORT_KEY = L"cnireport";
const wstring POST_HEALTH_REPORT_KEY = L"healthreport";
const wstring POST_NPM_REPORT_KEY = L"npmreport";
const wstring POST_DNC_REPORT_KEY = L"dncreport";
const wstring GET_VERSION_QUERY_KEY = L"getpackageversion";
const wstring GET_PACKAGE_URIS_QUERY_KEY = L"getpackageuris";
const wstring TYPE_KEY = L"type";
const wstring TYPE_CONTAINER = L"container";

const wstring HOST_NETAGENT_HTTP_LISTEN_PORT = L"8019";
const wstring RELATIVE_URI_PATH = L"hostnetagent";

static
void
GetPackageVersion(
    __inout http_request Request);

static 
void 
ProcessGet(
    __inout http_request Request);

void
StartListener();

bool
SetupLogging(
    __in PCWSTR LogLocation);

DWORD
GetPackageLocation(
    __inout std::wstring& ExecutableDirectory);

const
std::wstring
GetListenerUri();

DWORD
AgentParseCommandLine(
    __in ULONG ArgumentCount,
    __in_ecount(ArgumentCount) PWSTR *ArgumentList,
    __deref_out PWSTR *ConfigFile,
    __deref_out PWSTR *ResourcePath,
    __deref_out PWSTR *LogPath,
    __deref_out PWSTR *PackagePath);