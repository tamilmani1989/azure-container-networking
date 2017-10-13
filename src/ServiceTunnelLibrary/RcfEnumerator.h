/*++

Copyright (c) Microsoft Corporation

Module Name:

RcfEnumerator.h

Abstract:

Implements functions related to enumerating rcf files.

Environment:

User-mode

--*/
#pragma once

#ifndef ASSERT
#define ASSERT( exp ) ((void) 0)
#endif

#include <stdio.h>
#include <wchar.h>
#include <regex>
#include <iostream>
#include <windows.h>
#include <regex>
#include <iostream>
#include "RdTrace.h"
#include "RegKey.h"

using namespace std;

#define WINDOWS_AZURE_VM_TYPE_KEY L"SOFTWARE\\Microsoft\\Windows Azure\\"
#define VM_TYPE_VALUE L"VMType"
#define PAAS_RCF_LOCATION L"C:\\Config"
#define IAAS_RCF_LOCATION L"%SystemDrive%\\WindowsAzure\\Config"
#define IAAS_STRING L"IAAS"


class RcfEnumerator
{
public:
	HRESULT GetRcfLocation(__out LPWSTR* RcfLocation);
	HRESULT FindLatestRcf(__out LPWSTR* latestRcfFile, __in LPWSTR RcfLocation);
};