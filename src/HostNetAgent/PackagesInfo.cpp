/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    PackagesInfo.cpp

Abstract:

    Defines function to track packages information.

Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//
#include "packagesInfo.h"

using namespace std;

DWORD 
PackagesInfo::Initialize(
    __in std::wstring PackageLocation
    )
/*++

Description:

    Function to initialize the cache with the package information.

Arguments:

    PackageLocation - The package location

Return Value:

    ERROR_SUCCESS on sucess, aprropriate error code on failure.

--*/
{
    DWORD result;
    this->packageLocation = PackageLocation;
    result = SetPackageVersion();
    if (result != ERROR_SUCCESS)
    {
        return result;
    }
    result = FindPackages();
    return result;
}

const 
std::vector<std::wstring>& 
PackagesInfo::GetPackageNames()
/*++

Description:

    Function to retrieve the package names from cache.

Arguments:

    None.

Return Value:

    Collection of package names.

--*/
{
  return packageNames;
}

const 
std::wstring& 
PackagesInfo::GetPackageVersion()
/*++

Description:

    Function to get the apckage version from cache.

Arguments:

    None

Return Value:

    Location of the package.

--*/
{
    return packageVersion;
}

const 
std::wstring& 
PackagesInfo::GetPackageLocation()
/*++

Description:

    Function to retrieve the package location from cache.

Arguments:

    None

Return Value:

    Location of the package.

--*/
{
    return packageLocation;
}

DWORD 
PackagesInfo::SetPackageVersion()
/*++

Description:

    Function to set the verison in cache extracted from version.txt file.

Arguments:

    None.

Return Value:

    ERROR_SUCCESS on sucess, aprropriate error code on failure.

--*/
{	
    std::wstring versionFileName = packageLocation + L"\\" + VERSION_FILE_NAME;
    LOGINFOW(L"Version File:[%s]", versionFileName.c_str());
    std::wifstream versionFileStream(versionFileName.c_str());
    if (!versionFileStream.good())
    {
        LOGERRW(L"Version File:[%s] not found.", versionFileName.c_str());
        return ERROR_NOT_FOUND;
    }

    getline(versionFileStream, packageVersion);
    versionFileStream.close();
    if (packageVersion.empty())
    {
        LOGERRW(L"Version File:[%s] contents are empty", versionFileName.c_str());
        return ERROR_INVALID_DATA;
    }
    LOGINFOW(L"Retrieved version [%s] from Version File:[%s].", packageVersion.c_str(), versionFileName.c_str());
    return ERROR_SUCCESS;
}
DWORD 
PackagesInfo::FindPackages()
/*++

Description:

    Function to enumerate the packages with extension zip and add to the cache.

Arguments:

    None.

Return Value:

    None.

--*/
{
    std::vector<WIN32_FIND_DATA> fdata;
    WIN32_FIND_DATA fd;
    std::wstring fileName;
    std::wstring fileFilterString = packageLocation + L"\\*."+ZIP_EXTENSION;

    HANDLE hFind = ::FindFirstFile(fileFilterString.c_str(), &fd);
    while (INVALID_HANDLE_VALUE != hFind) {
        fileName.assign(fd.cFileName, wcslen(fd.cFileName));
        packageNames.push_back(fileName);
        if (!::FindNextFile(hFind, &fd)) {
            ::FindClose(hFind);
            break;
        }
    }

    LOGINFOW(L"Enumerate File Count:[%I64u]", packageNames.size());
    if (packageNames.size() == 0)
    {
        LOGERRW(L"No plugin files matching filter [%s]", fileFilterString.c_str());
        return ERROR_NOT_FOUND;
    }

    return ERROR_SUCCESS;
}