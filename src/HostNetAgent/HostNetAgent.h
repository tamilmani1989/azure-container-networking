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
#include "../Shared/Utilities.h"
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

using namespace std;
using namespace web;
using namespace http;
using namespace experimental::listener;
using namespace concurrency::streams;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;


#define DIRECTORY_SEPARATOR (L'\\')
#define DEFAULT_LOG_LOCATION  L"C:\\Logs\\HostNetAgent"
#define PACKAGE_FOLDER_NAME L"Package"

const wstring GET_VERSION_QUERY_KEY = L"getpackageversion";
const wstring GET_PACKAGE_URIS_QUERY_KEY = L"getpackageuris";
const wstring TYPE_KEY = L"type";

const wstring HOST_NETAGENT_HTTP_LISTEN_PORT = L"8019";
const wstring RELATIVE_URI_PATH = L"hostnetagent";

static PackagesInfo* packagesInfo;

STATIC
void
GetPackageVersion(
    __inout http_request Request);

STATIC 
void 
ProcessGet(
    __inout http_request Request);

STATIC
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