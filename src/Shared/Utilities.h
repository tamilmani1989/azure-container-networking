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
#include "PluginMetadata.h"

class AquariusUtilities
{
public:

	const static wchar_t *defaultPackageStoreLocation;
	const static wchar_t *codeDirectoryName;
	const static wchar_t *configFileName;
	const static wchar_t *manifestFileName;
	static int allocator;

	static void WriteEventLogEntry(PWSTR pszMessage);

	// Log a message to the Application event log.
	static void WriteErrorLogEntry(PWSTR pszFunction);

	static void ReleasePluginMetadata(int count, PluginMetadata * pluginsMetadata);

};
