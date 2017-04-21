/****************************** Module Header ******************************\
* Module Name:  VfpOperations.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Setup vfp
*
\***************************************************************************/

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
	};
} } } } }
#endif