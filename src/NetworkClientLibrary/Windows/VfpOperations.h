/*++

Copyright (c) Microsoft Corporation

Module Name:

VfpOperations.h

Abstract:

This cotains the declaration of methods that can be used to perform operations specific to Vfp in Windows.

Author:

Sushant Sharma (susharm) 20-April-2017

--*/

#ifdef _WIN32
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#include "../../Logging/Logging.h"
#include "../Common/nclcore.h"

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {

	class VfpOperations
	{
		static wchar_t *vfpProcessName;
		static DWORD StartServiceByName(wchar_t *vfpProcName);
	public:		
		static DWORD StartVfpService();
		static DWORD CheckIfServiceStartedSuccessfully(SERVICE_STATUS_PROCESS ssStatus, SC_HANDLE schSCManager, SC_HANDLE schService);
		static DWORD StartServiceAndWaitToStart(SERVICE_STATUS_PROCESS ssStatus, SC_HANDLE schSCManager, SC_HANDLE schService);
	};
} } } } }
#endif