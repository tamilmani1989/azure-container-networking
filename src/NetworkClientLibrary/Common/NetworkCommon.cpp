/*++

Copyright (c) Microsoft Corporation

Module Name:

NetworkCommon.c

Abstract:

This file contains methods that can be used by both windows and linux code
to setup container network.

Author:

Sushant Sharma (susharm) 30-March-2017

--*/
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include "../../Logging/Logging.h"
#include "NetworkCommon.h"

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
	
bool NetworkCommon::RetrieveGatewayFromSubnet(
	char*subnet, 
	char **gatewayIp)
/*++

Routine Description:

	Parses the subnet, and retrieves the gateway ip from it.

Arguments:

	subnet - Subnet that needs to be parsed.
	gatewayIp - (output parameter) will be populated with gatewayIp on success.

Return Value:
		
	0 on success.

--*/
{
	LOGINFO("Entering RetrieveGatewayFromSubnet.");
	size_t len = strlen(subnet);

	LOGINFO("Going to parse subnet %s with length:%llu for gateway retrieval.", subnet, len);
	*gatewayIp = static_cast<char *>(malloc(len + 1));
	int dotCount = 0;
	int k;
	for (k = 0; k < len + 1; k++)
	{
		if (subnet[k] == '.') {
			dotCount++;
		}

		(*gatewayIp)[k] = subnet[k];

		if (dotCount == 3)
		{
			break;
		}
	}

	if (dotCount != 3)
	{
		free(*gatewayIp);
		LOGERR("Exiting RetrieveGatewayFromSubnet. Unable to parse subnet.");
		return false;
	}

	(*gatewayIp)[++k] = '1';
	(*gatewayIp)[++k] = '\0';
	LOGINFO("Exiting RetrieveGatewayFromSubnet. Returning Gateway Ip=%s", *gatewayIp);
	return true;
}
	
} } } } }