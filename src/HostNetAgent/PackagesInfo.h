/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    PackagesInfo.h

Abstract:

    Defines function headers to track packages information.

Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//
#pragma once

#include <stdio.h>
#include <wchar.h>
#include <iostream>
#include <windows.h>
#include <iostream>
#include <vector>
#include <fstream>
#include <string>
#include "../Logging/Logging.h"

using namespace std;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

#define VERSION_FILE_NAME L"version.txt";
#define ZIP_EXTENSION L"zip";

class PackagesInfo
{

public:
    DWORD 
    Initialize(
        __in std::wstring PackageLocation
    );

    const 
    std::vector<std::wstring>& 
    GetPackageNames();

    const
    std::wstring& 
    GetPackageVersion();

    const 
    std::wstring& 
    GetPackageLocation() const;

    static PackagesInfo* GetInstance();

private:

    DWORD 
    FindPackages();

    DWORD 
    SetPackageVersion();

    // Cache to store the package names.
    std::vector<std::wstring> packageNames;

    // Package folder
    std::wstring packageLocation;

    // Cached package version
    std::wstring packageVersion;
};