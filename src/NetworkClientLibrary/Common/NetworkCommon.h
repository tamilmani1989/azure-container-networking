/*++

Copyright (c) Microsoft Corporation

Module Name:

NetworkCommon.h

Abstract:

This file contains declaration for methods that can be used by both windows and linux code
to setup container network.

Author:

Sushant Sharma (susharm) 5-May-2017

--*/

#pragma once
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include "../../Logging/Logging.h"

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
	class NetworkCommon
	{
	public:
		static bool RetrieveGatewayFromSubnet(char*subnet, char **gatewayIp);
	};
} } } } }