#include <stdio.h>
#include <stdlib.h>
#ifdef _WIN32
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>
#endif
#ifdef __linux__
#include <arpa/inet.h>
#include <string.h>
#include <cstdlib>
#define strncpy_s(dst, count, src, cp) strncpy(dst, src, count)
#endif
#include "../NetworkClientLibrary/Common/aquanetworkclient.h"
#include "../Shared/Utilities.h"

extern DllExport int DeleteLocalEndpointWithIp(wchar_t *hostLocalIp, bool detachIfHostLocalIp);

void assert(unsigned long value)
{
	if (value == 0) return;
	printf("assert failed");
	getchar();
	throw "assert failed";
}

void logstuff(int level, char *text)
{
	printf("%d: %s", level, text);
}

#ifdef _WIN32
int wmain()
#endif
#ifdef __linux__
int main()
#endif
{
	AquariusUtilities::WriteEventLogEntry(L"Working or not");

	ClusterConfig cfg;
    NodeNetworkResources nnr;
    int ipCount = -1;
    int i;
    char ipInString[INET_ADDRSTRLEN];
	
	assert(SetupNetworkLogging(logstuff));

	size_t netLen = strlen("azurenet") + sizeof(char);
	cfg.networkName = (char *) malloc(netLen);
	printf("%llu", netLen);
	strncpy_s(cfg.networkName, netLen, "azurenet", 9);
	cfg.networkName[8] = '\0';	
	cfg.net = UnderLay;
	cfg.env = Azure;
	printf("nw main name:%s\n",cfg.networkName);
    int retval = 0;
	#ifdef __linux__
	retval = InitializeNetwork(&cfg);
	#endif

	#ifdef _WIN32
	//assert(InitializeNetwork(&cfg));
	#endif

	printf("Successfully initialized Azure/Underlay. retval = %d\n", retval);
	getchar();

	cfg.net = OverLay;
	cfg.env = Azure;
	cfg.nodeCount = 1;
	Ipv4Subnet ovSubnet;
	struct in_addr ipAdd;
	inet_pton(AF_INET, "10.10.0.0", &ipAdd);
	ovSubnet.ipAddress = ipAdd.s_addr;

	printf("converted ip prefix %s\n", inet_ntop(AF_INET, &ovSubnet.ipAddress, ipInString, INET_ADDRSTRLEN));
	getchar();
	ovSubnet.prefixLength = 24;
	cfg.overlaySubnet = &ovSubnet;
	cfg.nodeConfigs = (NodeConfig *)malloc(sizeof(NodeConfig));

	inet_pton(AF_INET, "11.1.0.114", &ipAdd);
	cfg.nodeConfigs[0].nodeIp = ipAdd.s_addr;
	cfg.nodeConfigs[0].nodeId = "Id";
	cfg.nodeConfigs[0].subnet = &ovSubnet;
	printf("Initialization result for StandAlone/Overlay: %ld \n", InitializeNetwork(&cfg));
	free(cfg.networkName);
	getchar();

	unsigned int hostLocalIp;
	GetHostLocalIp(&hostLocalIp);
	printf("Recevied host local as %s\n", inet_ntop(AF_INET, &hostLocalIp, ipInString, INET_ADDRSTRLEN));
	getchar();

	printf("Going ot detach hostlocal directly in HNS\n");
	wchar_t ipInStringw[INET_ADDRSTRLEN];
	size_t converted;
	mbstowcs_s(&converted, ipInStringw, ipInString, INET_ADDRSTRLEN);
	DeleteLocalEndpointWithIp(ipInStringw, true);
	getchar();
	unsigned int ip;
	printf("going to reserve available ip with ip0");
	assert(ReserveAvailableIpAddress("ip0", &ip));
	printf("ip0 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

    nnr.ips = NULL;
    assert(GetAvailableIpAddresses(&nnr, &ipCount));
	printf("Retrieved ipCount = %d\n", ipCount);
	getchar();

    nnr.ips = (unsigned int *) malloc(ipCount * sizeof(unsigned int));
    assert(GetAvailableIpAddresses(&nnr, &ipCount));
    printf("Printing available ip addresses count = %d\n", ipCount);
    for(i = 0; i < ipCount ; i++)
    {
        printf("%s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
    }
	getchar();
    
	assert(ReserveAvailableIpAddress("ip1", &ip));
    printf("ip1 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

    assert(GetAvailableIpAddresses(&nnr, &ipCount));
    printf("Printing available ip addresses count = %d\n", ipCount);
    for(i = 0; i < ipCount ; i++)
    {
        printf(" %s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
    }
	getchar();

    assert(GetReservedIpAddress("ip1", &ip));
    printf("ip1 (retrieving reserved ip) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

	retval = GetReservedIpAddress("ip2", &ip);
    printf("ip2 (without reservation). retval = %d\n", retval);
	getchar();
    
    assert(ReserveAvailableIpAddress("ip2", &ip));
    printf("ip2 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

    assert(GetAvailableIpAddresses(&nnr, &ipCount));
    printf("Printing available ip addresses count = %d\n", ipCount);
    for(i = 0; i < ipCount ; i++)
    {
        printf("%s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
    }
	getchar();

	assert(GetReservedIpAddress("ip2", &ip));
    printf("ip2 (retrieving reserved ip) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();


	assert(ReleaseIpReservation("ip1"));
	retval = GetReservedIpAddress("ip1", &ip);
	printf("ip1 (without reservation). retval = %d\n", retval);
	getchar();

	assert(ReleaseIpReservation("ip2"));
	retval = GetReservedIpAddress("ip2", &ip);
	printf("ip2 (without reservation). retval =  %d\n", retval);
	getchar();
    free(nnr.ips);

	return 0;
}
