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
#include "PluginMetadata.h"

class AquariusUtilities
{
public:
	static void WriteEventLogEntry(PWSTR pszMessage);

	// Log a message to the Application event log.
	static void WriteErrorLogEntry(PWSTR pszFunction);

	static int allocator;

	static void ReleasePluginMetadata(int count, PluginMetadata * pluginsMetadata);

};
