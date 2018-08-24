// stdafx.h : include file for standard system include files,
// or project specific include files that are used frequently, but
// are changed infrequently
//

#pragma once

#ifndef ASSERT
#define ASSERT( exp ) ((void) 0)
#endif

#define _CRTDBG_MAP_ALLOC

#include "targetver.h"
#include <ws2tcpip.h>
#include <windows.h> 
#include <tchar.h>
#include <stdlib.h>
#include <stdio.h>
#include <setupapi.h>
#include <regstr.h>
#include <infstr.h>
#include <cfgmgr32.h>
#include <string.h>
#include <malloc.h>
#include <newdev.h>
#include <objbase.h>
#include <strsafe.h>
#include <RdCommon.h>
#include <RdTrace.h>
#include <vector>
#include <Mstcpip.h>
#include <malloc.h>
#include <ws2def.h>
#include <ws2ipdef.h>
#include <Netioapi.h>
#include <Netcfgx.h>
#include <devguid.h>
#include <winsock2.h>
#include <windef.h>
#include <ws2ipdef.h>
#include <iphlpapi.h>
#include <assert.h>
#include <crtdbg.h>

#pragma comment(lib, "IPHLPAPI.lib")