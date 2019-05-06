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
#include<psapi.h>
#include "PluginMetadata.h"

#define SAFE_DELETE(p)     do { if ((p!=nullptr)) { delete[] p; p = nullptr; } } while(0)\

class AquariusUtilities
{
public:

	const static wchar_t *defaultPackageStoreLocation;
	const static wchar_t *codeDirectoryName;
	const static wchar_t *configFileName;
	const static wchar_t *manifestFileName;
	static wchar_t *relativePathFromExe;
	static int allocator;

	static void WriteEventLogEntry(PWSTR pszMessage);

	// Log a message to the Application event log.
	static void WriteErrorLogEntry(PWSTR pszFunction);

	static void ReleasePluginMetadata(int count, PluginMetadata * pluginsMetadata);
	static void PopulatePluginLocation(wchar_t* packageStoreLocation);
	static void PopulateServiceExeLocation(wchar_t* packageStoreLocation);
	static void SetRelativePathFromExe(wchar_t *relPath);
	static int GetServiceName(wchar_t *svcName, size_t size);
	static DWORD ConvertWcharToChar(wchar_t *messageW, char *message, size_t msgSize);
};
