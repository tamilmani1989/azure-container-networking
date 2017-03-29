#include <stdio.h>
#include <stdlib.h>
#ifdef _WIN32
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>
#endif
#ifdef __linux__
#include <arpa/inet.h>
#endif
#include "../NetworkClientLibrary/Common/aquanetworkclient.h"

#ifdef _WIN32
int wmain(int argc, wchar_t *argv[])
#endif
#ifdef __linux__
int main()
#endif
{

	ClusterConfig cfg;
    NodeNetworkResources nnr;
    int ipCount = -1;
    int i;
    struct in_addr ipaddr;
    char ipInString[INET_ADDRSTRLEN];

	cfg.networkName = (char *) malloc(strlen("azurenet") +1);
	strncpy(cfg.networkName, "azurenet", 8);
	cfg.networkName[8] = '\0';
	
	cfg.net = UnderLay;
	cfg.env = Azure;
	printf("Initialization result for Azure/Underlay: %d\n",InitializeNetwork(&cfg));
	free(cfg.networkName);
	getchar();

	cfg.net = OverLay;
	cfg.env = StandAlone;
	printf("Initialization result for StandAlone/Overlay: %d\n", InitializeNetwork(&cfg));
	getchar();

	unsigned int ip = ReserveAvailableIpAddress("ip0");
	printf("ip0 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

    nnr.ips = NULL;
    ipCount = GetAvailableIpAddresses(&nnr);
	printf("Retrieved ipCount = %d\n", ipCount);
	getchar();

    nnr.ips = (unsigned int *) malloc(ipCount * sizeof(unsigned int));
    ipCount = GetAvailableIpAddresses(&nnr);
    printf("Printing available ip addresses count = %d\n", ipCount);
    for(i = 0; i < ipCount ; i++)
    {
        printf("%s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
    }
	getchar();
    
	ip = ReserveAvailableIpAddress("ip1");
    printf("ip1 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

    ipCount = GetAvailableIpAddresses(&nnr);
    printf("Printing available ip addresses count = %d\n", ipCount);
    for(i = 0; i < ipCount ; i++)
    {
        printf(" %s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
    }
	getchar();

    ip = GetReservedIpAddress("ip1");
    printf("ip1 (retrieving reserved ip) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

    ip = GetReservedIpAddress("ip2");
    printf("ip2 (without reservation) = %d\n", ip);
	getchar();
    
    ip = ReserveAvailableIpAddress("ip2");
    printf("ip2 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

    ipCount = GetAvailableIpAddresses(&nnr);
    printf("Printing available ip addresses count = %d\n", ipCount);
    for(i = 0; i < ipCount ; i++)
    {
        printf("%s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
    }
	getchar();

    ip = GetReservedIpAddress("ip2");
    printf("ip2 (retrieving reserved ip) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

	ReleaseIpReservation("ip2");
	ip = GetReservedIpAddress("ip2");
	printf("ip2 (without reservation) = %d\n", ip);
	getchar();
    
    free(nnr.ips);

	return 0;
}
