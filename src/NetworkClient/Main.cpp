#include <stdio.h>
#include <stdlib.h>
#ifdef _WIN32
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>
#include "../Shared/Utilities.h"
#endif
#ifdef __linux__
#include <arpa/inet.h>
#include <string.h>
#include <cstdlib>
#define strncpy_s(dst, count, src, cp) strncpy(dst, src, count)
#endif
#include "../NetworkClientLibrary/Common/aquanetworkclient.h"
#include "../Logging/Logging.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

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
	printf("%d: %s\n", level, text);
}

extern DllExport DWORD SetupNetworkEtwLogging(bool additionConsoleLoggingEnabled);

#ifdef _WIN32
int wmain()
#endif
#ifdef __linux__
int main()
#endif
{
	//AquariusUtilities::WriteEventLogEntry(L"Working or not");

	ClusterConfig cfg;
	NodeNetworkResources nnr;
	int ipCount = -1;
	int i;
	char ipInString[INET_ADDRSTRLEN]={0};

	//assert(SetupNetworkEtwLogging(true));
	assert(SetupNetworkLogging(logstuff));

	size_t netLen = strlen("azurenet") + sizeof(char);
	cfg.networkName = (char *)malloc(netLen);
	printf("%llu", netLen);
	strncpy_s(cfg.networkName, netLen, "azurenet", 9);
	cfg.networkName[8] = '\0';
	cfg.net = UnderLay;
	cfg.env = Azure;
	printf("nw main name:%s\n", cfg.networkName);
	int retval = 0;
#ifdef __linux__
//	retval = InitializeNetwork(&cfg);
#endif

#ifdef _WIN32
	//assert(InitializeNetwork(&cfg));
#endif

	printf("Successfully initialized Azure/Underlay. retval = %d\n", retval);
	getchar();

	cfg.net = OverLay;
	cfg.env = Azure;
	cfg.nodeCount = 3;


	Ipv4Subnet ovSubnet;
	struct in_addr ipAdd;
	inet_pton(AF_INET, "10.10.0.0", &ipAdd);
	ovSubnet.ipAddress = ipAdd.s_addr;

	printf("converted ip prefix %s\n", inet_ntop(AF_INET, &ovSubnet.ipAddress, ipInString, INET_ADDRSTRLEN));
	getchar();
	ovSubnet.prefixLength = 16;
	cfg.overlaySubnet = &ovSubnet;

	cfg.nodeConfigs = (NodeConfig *)malloc(cfg.nodeCount * sizeof(NodeConfig));
	
	Ipv4Subnet localSubnet;

	inet_pton(AF_INET, "10.10.1.0", &ipAdd);
	localSubnet.ipAddress = ipAdd.s_addr;
	localSubnet.prefixLength = 24;
	inet_pton(AF_INET, "11.1.0.144", &ipAdd);
	cfg.nodeConfigs[0].nodeIp = ipAdd.s_addr;
	cfg.nodeConfigs[0].nodeId = "Id114";
	cfg.nodeConfigs[0].subnet = &localSubnet;
	cfg.localNodeIp = ipAdd.s_addr;
	

	Ipv4Subnet localSubnet2;
	inet_pton(AF_INET, "10.10.2.0", &ipAdd);
	localSubnet2.ipAddress = ipAdd.s_addr;
	localSubnet2.prefixLength = 24;
	inet_pton(AF_INET, "11.1.0.154", &ipAdd);
	cfg.nodeConfigs[1].nodeIp = ipAdd.s_addr;
	cfg.nodeConfigs[1].nodeId = "Id124";
	cfg.nodeConfigs[1].subnet = &localSubnet2;
	//cfg.localNodeIp = ipAdd.s_addr;


	Ipv4Subnet localSubnet3;
	inet_pton(AF_INET, "10.10.3.0", &ipAdd);
	localSubnet3.ipAddress = ipAdd.s_addr;
	localSubnet3.prefixLength = 24;
	inet_pton(AF_INET, "11.1.0.74", &ipAdd);
	cfg.nodeConfigs[2].nodeIp = ipAdd.s_addr;
	cfg.nodeConfigs[2].nodeId = "Id134";
	cfg.nodeConfigs[2].subnet = &localSubnet3;
	//cfg.localNodeIp = ipAdd.s_addr;

	printf("Initialization result for StandAlone/Overlay: %ld \n", InitializeNetwork(&cfg));
	free(cfg.networkName);
	getchar();

	unsigned int hostLocalIp=0;
	GetHostLocalIp(&hostLocalIp);
	printf("Recevied host local as %s\n", inet_ntop(AF_INET, &hostLocalIp, ipInString, INET_ADDRSTRLEN));
	getchar();

	printf("Going ot detach hostlocal directly in HNS\n");
	wchar_t ipInStringw[INET_ADDRSTRLEN];
	size_t converted;
	mbstowcs_s(&converted, ipInStringw, ipInString, INET_ADDRSTRLEN);
#ifdef _WIN32
	DeleteLocalEndpointWithIp(ipInStringw, true);
#endif
	getchar();
	unsigned int ip=0;
	printf("going to reserve available ip with ip0");
	assert(ReserveAvailableIpAddress((char *)"ip0", &ip));
	printf("ip0 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

    nnr.ips = NULL;
    assert(GetAvailableIpAddresses(&nnr, &ipCount));
	printf("Retrieved ipCount = %d\n", ipCount);
	getchar();

    nnr.ips = (unsigned int *) malloc(ipCount * sizeof(unsigned int));
    memset(nnr.ips, 0, ipCount * sizeof(unsigned int));
    assert(GetAvailableIpAddresses(&nnr, &ipCount));
    printf("Printing available ip addresses count = %d\n", ipCount);
    for(i = 0; i < ipCount ; i++)
    {
        printf("%s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
    }
	getchar();
    
	assert(ReserveAvailableIpAddress((char *)"ip1", &ip));
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
    
    assert(ReserveAvailableIpAddress((char *)"ip2", &ip));
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

	assert(ReleaseIpReservation("ip0"));
	retval = GetReservedIpAddress("ip0", &ip);
	printf("ip0 (without reservation). retval =  %d\n", retval);
    free(nnr.ips);

	return 0;
}
