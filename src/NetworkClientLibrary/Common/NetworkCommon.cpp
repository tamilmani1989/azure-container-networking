#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include "../../Logging/Logging.h"

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {

bool RetrieveGatewayFromSubnet(char*subnet, char **gatewayIp)
{
	size_t len = strlen(subnet);
	char logMsg[2048];
	size_t msgSize = 2048;
	sprintf_s(logMsg, msgSize, "retrieved subnet %s with length:%d for gateway retrieval.\n", 
		subnet, static_cast<int>(len));
	AquaLogging::Log(info, logMsg);
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

	if(dotCount != 3)
	{
		free(*gatewayIp);
		return false;
	}
	
	(*gatewayIp)[++k] = '1';
	(*gatewayIp)[++k] = '\0';
	sprintf_s(logMsg, msgSize, "gateway Ip=%s\n", *gatewayIp);
	return true;
}

} } } } }