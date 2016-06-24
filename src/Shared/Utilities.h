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

class AquariusUtilities
{
public:
	static void WriteEventLogEntry(PWSTR pszMessage, WORD wType);

	// Log a message to the Application event log.
	static void WriteErrorLogEntry(PWSTR pszFunction, DWORD dwError);
};
