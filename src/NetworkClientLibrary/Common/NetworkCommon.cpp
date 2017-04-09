#include <string.h>
#include <stdio.h>
#include <stdlib.h>

bool RetrieveGatewayFromSubnet(char*subnet, char **gatewayIp)
{
	size_t len = strlen(subnet);
	printf("retrieved subnet %s with length:%d for gateway retrieval.\n", 
		subnet, static_cast<int>(len));
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
	printf("gateway Ip=%s\n", *gatewayIp);
	return true;
}