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
	LOGERR("assert failed");
	getchar();
	throw "assert failed";
}

void logstuff(int level, char *text)
{
	printf("%d: %s\n", level, text);
}

extern DllExport DWORD SetupNetworkEtwLogging(bool additionConsoleLoggingEnabled);

void testM()
{
	
	ClusterConfig cfg;
	NodeNetworkResources nnr;
	int ipCount = -1;
	int i;
	char ipInString[INET_ADDRSTRLEN]={0};

	//assert(SetupNetworkEtwLogging(true));
	assert(SetupNetworkLogging(logstuff));

	size_t netLen = strlen("azurenet") + sizeof(char);
	cfg.networkName = (char *)malloc(netLen);
	LOGINFO("%llu", netLen);
	strncpy_s(cfg.networkName, netLen, "azurenet", 9);
	cfg.networkName[8] = '\0';
	cfg.net = UnderLay;
	cfg.env = Azure;
	LOGINFO("nw main name:%s\n", cfg.networkName);
	int retval = 0;
	unsigned int hostIp=0;

#ifdef __linux__
	retval = InitializeNetwork(&cfg);
#endif

#ifdef _WIN32
	//assert(InitializeNetwork(&cfg));
#endif

	/*LOGINFO("Successfully initialized Azure/Underlay. retval = %d\n", retval);
	getchar();

	GetHostLocalIp(&hostIp);
	LOGINFO("Get local host ip (underlay):%d %s\n",hostIp,inet_ntop(AF_INET, &hostIp, ipInString, INET_ADDRSTRLEN));
	hostIp=0;
	getchar();
	 */
	cfg.net = OverLay;
	cfg.env = Azure;
	cfg.nodeCount = 3;


	Ipv4Subnet ovSubnet;
	struct in_addr ipAdd;
	inet_pton(AF_INET, "10.0.0.0", &ipAdd);
	ovSubnet.ipAddress = ipAdd.s_addr;

	LOGINFO("converted ip prefix %s\n", inet_ntop(AF_INET, &ovSubnet.ipAddress, ipInString, INET_ADDRSTRLEN));
	getchar();
	ovSubnet.prefixLength = 8;
	cfg.overlaySubnet = &ovSubnet;

	cfg.nodeConfigs = (NodeConfig *)malloc(cfg.nodeCount * sizeof(NodeConfig));
	
	Ipv4Subnet localSubnet;

	inet_pton(AF_INET, "10.10.0.0", &ipAdd);
	localSubnet.ipAddress = ipAdd.s_addr;
	localSubnet.prefixLength = 21;
	inet_pton(AF_INET, "11.1.0.164", &ipAdd);
	cfg.nodeConfigs[0].nodeIp = ipAdd.s_addr;
	cfg.nodeConfigs[0].nodeId = "Id114";
	cfg.nodeConfigs[0].subnet = &localSubnet;
	//cfg.localNodeIp = ipAdd.s_addr;
	

	Ipv4Subnet localSubnet2;
	inet_pton(AF_INET, "10.11.0.0", &ipAdd);
	localSubnet2.ipAddress = ipAdd.s_addr;
	localSubnet2.prefixLength = 21;
	inet_pton(AF_INET, "11.1.0.124", &ipAdd);
	cfg.nodeConfigs[1].nodeIp = ipAdd.s_addr;
	cfg.nodeConfigs[1].nodeId = "Id124";
	cfg.nodeConfigs[1].subnet = &localSubnet2;
	cfg.localNodeIp = ipAdd.s_addr;


	Ipv4Subnet localSubnet3;
	inet_pton(AF_INET, "10.12.0.0", &ipAdd);
	localSubnet3.ipAddress = ipAdd.s_addr;
	localSubnet3.prefixLength = 21;
	inet_pton(AF_INET, "11.1.0.134", &ipAdd);
	cfg.nodeConfigs[2].nodeIp = ipAdd.s_addr;
	cfg.nodeConfigs[2].nodeId = "Id134";
	cfg.nodeConfigs[2].subnet = &localSubnet3;
	//cfg.localNodeIp = ipAdd.s_addr;

	LOGINFO("Initialization result for StandAlone/Overlay: %ld \n", InitializeNetwork(&cfg));
	free(cfg.networkName);
	free(cfg.nodeConfigs);
	getchar();

	unsigned int hostLocalIp=0;
	GetHostLocalIp(&hostLocalIp);
	LOGINFO("Recevied host local as %s\n", inet_ntop(AF_INET, &hostLocalIp, ipInString, INET_ADDRSTRLEN));
	getchar();

	LOGINFO("Going to detach hostlocal directly in HNS\n");
	wchar_t ipInStringw[INET_ADDRSTRLEN];
	size_t converted;
	mbstowcs_s(&converted, ipInStringw, ipInString, INET_ADDRSTRLEN);
#ifdef _WIN32
	DeleteLocalEndpointWithIp(ipInStringw, true);
#endif
	getchar();

	//unsigned int ip;
	 

	nnr.ips = NULL;
	assert(GetAvailableIpAddresses(&nnr, &ipCount));
	LOGINFO("Retrieved ipCount = %d\n", ipCount);
	
	nnr.ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));
	memset(nnr.ips, 0, ipCount * sizeof(unsigned int));
	assert(GetAvailableIpAddresses(&nnr, &ipCount));
	LOGINFO("Printing available ip addresses count = %d\n", ipCount);

	for (i = 0; i < ipCount; i++)
	{
		LOGERR("%s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
	}
	getchar();

	unsigned int ip=0;
	LOGINFO("going to reserve available ip with ip0");
	assert(ReserveAvailableIpAddress((char *)"ip0", &ip));
	LOGINFO("ip0 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	//getchar();

	//getchar();

    assert(GetAvailableIpAddresses(&nnr, &ipCount));
    LOGINFO("Printing available ip addresses count = %d\n", ipCount);
    for(i = 0; i < ipCount ; i++)
    {
        LOGINFO("%s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
    }
	//getchar();
    
	assert(ReserveAvailableIpAddress((char *)"ip1", &ip));
    LOGINFO("ip1 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

    assert(GetAvailableIpAddresses(&nnr, &ipCount));
    LOGINFO("Printing available ip addresses count = %d\n", ipCount);
    for(i = 0; i < ipCount ; i++)
    {
        LOGINFO(" %s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
    }
	getchar();

    assert(GetReservedIpAddress("ip1", &ip));
    LOGINFO("ip1 (retrieving reserved ip) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

	retval = GetReservedIpAddress("ip2", &ip);
    LOGINFO("ip2 (without reservation). retval = %d\n", retval);
	getchar();
    
    assert(ReserveAvailableIpAddress((char *)"ip2", &ip));
    LOGINFO("ip2 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

    assert(GetAvailableIpAddresses(&nnr, &ipCount));
    LOGINFO("Printing available ip addresses count = %d\n", ipCount);
    for(i = 0; i < ipCount ; i++)
    {
        LOGINFO("%s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
    }
	getchar();

	assert(GetReservedIpAddress("ip2", &ip));
    LOGINFO("ip2 (retrieving reserved ip) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();


	assert(ReleaseIpReservation("ip1"));
	retval = GetReservedIpAddress("ip1", &ip);
	LOGINFO("ip1 (without reservation). retval = %d\n", retval);
	getchar();


	assert(ReserveAvailableIpAddress((char *)"ip5", &ip));
	LOGINFO("ip5 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

	assert(GetReservedIpAddress((char*)"ip5", &ip));
	LOGINFO("ip5 (retrieving reserved ip) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

	assert(ReleaseIpReservation("ip2"));
	retval = GetReservedIpAddress("ip2", &ip);
	LOGINFO("ip2 (without reservation). retval =  %d\n", retval);
	getchar();

	assert(ReleaseIpReservation("ip0"));
	retval = GetReservedIpAddress("ip0", &ip);
	LOGINFO("ip0 (without reservation). retval =  %d\n", retval);

	getchar();
	GetHostLocalIp(&hostIp);
	LOGINFO("Get local host ip (overlay):%d %s\n",hostIp,inet_ntop(AF_INET, &hostIp, ipInString, INET_ADDRSTRLEN));
	getchar();

	assert(GetAvailableIpAddresses(&nnr, &ipCount));
	LOGINFO("Printing available ip addresses count = %d\n", ipCount);
	for (i = 0; i < ipCount; i++)
	{
		LOGINFO("%s\n", inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN));
	}
	getchar();
   	free(nnr.ips);

}

#ifdef _WIN32
int wmain()
#endif
#ifdef __linux__
int main()
#endif
{
	//AquariusUtilities::WriteEventLogEntry(L"Working or not");

	testM();

	return 0;
}
