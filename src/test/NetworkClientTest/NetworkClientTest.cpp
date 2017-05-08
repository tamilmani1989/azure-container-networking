#ifndef UNICODE
#define UNICODE
#endif


#include <windows.h>

#include <stdio.h>
#include <string>
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>


#include "NetworkClientTest.h"
#include "../../NetworkClientLibrary/Common/NclRepositoryManager.h"
#include "../../NetworkClientLibrary/Common/ResponseParser.h"
#include "aquanetworkclient.h"
#include "nclcore.h"
#include "TraceLogging.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

TRACELOGGING_DEFINE_PROVIDER(
    g_hMyComponentProvider,
    "SimpleTraceLoggingProvider",
    (0x3970f9cf, 0x2c0c, 0x4f11, 0xb1, 0xcc, 0xe3, 0xa1, 0xe9, 0x95, 0x88, 0x33));
	
using namespace std;
using namespace WEX::TestExecution;

#ifdef _test
void
NetworkClientTest::TestInitializeNetwork()
/*++

Routine Description:

    This route is to test creation of network adapter for docker

Arguments:

    None.

Return Value:

    None.

--*/
{
	ClusterConfig cfg;
	DWORD ret = 0;
	size_t netLen = strlen("azurenet") + sizeof(char);
	cfg.networkName = (char *) malloc(netLen);
	strncpy_s(cfg.networkName, netLen, "azurenet", 9);
	cfg.networkName[8] = '\0';
	cfg.net = UnderLay;
	cfg.env = Azure;

	ret = InitializeNetwork(&cfg);

    VERIFY_ARE_EQUAL((DWORD)0, ret, L"Azure Underlay Passed!");
	TraceLoggingWrite(g_hMyComponentProvider, 
		"AzureUnderlay",              
		TraceLoggingValue(ret, "InitializeNetwork ret"));

	cfg.net = OverLay;
	cfg.env = Azure;

	ret = InitializeNetwork(&cfg);

	VERIFY_ARE_NOT_EQUAL((DWORD)0, ret, L"Azure Overlay Passed!");
	TraceLoggingWrite(g_hMyComponentProvider,
		"AzureOverlay",
		TraceLoggingValue(ret, "InitializeNetwork ret"));
	free(cfg.networkName);

}
#endif

NclRepositoryManager *nclRepositoryManager = NULL;
#ifdef _WIN32
std::mutex repositoryVariableMutex;
#define LockIpRepositoryVariable repositoryVariableMutex.lock();
#define UnLockIpRepositoryVariable repositoryVariableMutex.unlock();
#endif

NclRepositoryManager *GetIpRepository()
{
	NclRepositoryManager *_nclRepositoryVariable;
	if (nclRepositoryManager)
	{
		_nclRepositoryVariable = nclRepositoryManager;
	}
	else
	{
		LockIpRepositoryVariable
			if (nclRepositoryManager)
			{
				_nclRepositoryVariable = nclRepositoryManager;
			}
			else
			{
				_nclRepositoryVariable = new NclRepositoryManager(NULL);
				nclRepositoryManager = _nclRepositoryVariable;
			}
		UnLockIpRepositoryVariable
	}
	return _nclRepositoryVariable;
}

int getAvailableIpCountOverlay()
{
	return GetIpRepository()->GetAvailableIpsfromRepository(NULL);
}

int getAvailableIpCountUnderlay()
{
	int ipCount = 0;
	NodeNetworkResources nnr = { 0 };
	char *xmlResp = "<Interfaces><Interface MacAddress=\"000D3A72DC45\" IsPrimary=\"true\"><IPSubnet Prefix=\"11.1.0.0/24\"><IPAddress Address=\"11.1.0.74\" IsPrimary=\"true\"/><IPAddress Address=\"11.1.0.75\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.76\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.77\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.78\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.79\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.80\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.81\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.82\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.83\" IsPrimary=\"false\"/></IPSubnet></Interface></Interfaces>";

	ipCount = ParseSecondaryIpAddresses(xmlResp, 1, NULL);

	nnr.ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));

	ipCount = ParseSecondaryIpAddresses(xmlResp, 0, nnr.ips);
	GetIpRepository()->UpdateRepositoryIpList(ipCount, nnr.ips, UnderLay);
	ipCount = GetIpRepository()->GetAvailableIpsfromRepository(nnr.ips);

	free(nnr.ips);
	return ipCount;
}

void TestGetLocalHostIp()
{
	DWORD ret = 0;
	unsigned int ip=0;
	ret = GetHostLocalIp(&ip);
	VERIFY_ARE_EQUAL((DWORD)0, ret, L"TestGetLocalHostIp passed");	
	VERIFY_ARE_NOT_EQUAL((unsigned int)0, ip, L"ip not zero");
}

void testReserveIp(char *reservationId, NetworkType type)
{
	int ipCountBefore = -1;
	int ipCountAfter = -1;
	unsigned int ret = 0;

	if (type == OverLay)
		ipCountBefore = getAvailableIpCountOverlay();
	else
		ipCountBefore = getAvailableIpCountUnderlay();

	//printf("IpCount before for %s is %d\n", reservationId,ipCountBefore);

	VERIFY_IS_GREATER_THAN(ipCountBefore,0);

	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReserveIpAddress",
		TraceLoggingValue(reservationId, "ReservationId"),
		TraceLoggingValue(ret, "GetAvailable Ret"),
		TraceLoggingValue(ipCountBefore, "IPCount Before"));

	ret = GetIpRepository()->ReserveRepositoryIpAddress(reservationId);
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReserveIpAddress",
		TraceLoggingValue(reservationId, "ReservationId"),
		TraceLoggingValue(ret, "ReservedAvailable Ret"));
	//printf("Ret value:%d\n", ret);

	VERIFY_ARE_NOT_EQUAL((unsigned int)0, ret, L"ReserveAvailableIpAddress Passed!");


	if (type == OverLay)
		ipCountAfter = getAvailableIpCountOverlay();
	else
		ipCountAfter = getAvailableIpCountUnderlay();

	//printf("IpCount After for %s:%d\n", reservationId, ipCountAfter);

	VERIFY_ARE_EQUAL(ipCountAfter, ipCountBefore - 1, L"Ip Reservation Passed");

	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReserveIpAddress",
		TraceLoggingValue(reservationId, "ReservationId"),
		TraceLoggingValue(ret, "GetAvailable Ret"),
		TraceLoggingValue(ipCountBefore, "IPCount Before"),
		TraceLoggingValue(ipCountAfter, "IPCount After"));

}

void TestGetReservedIpAddress()
{
	unsigned int ip=0;
	DWORD ret;
	char ipInString[INET_ADDRSTRLEN];
	const char *temp = NULL;

	ret = GetReservedIpAddress("ip1", &ip);
	TraceLoggingWrite(
		g_hMyComponentProvider,
		"TestGetReservedIpAddress",
		TraceLoggingValue("ip1", "ReservationId"),
		TraceLoggingValue(ret, "GetReserved Ret"));

	VERIFY_ARE_EQUAL((DWORD)0, ret, L"GetReserveIp fn Passed");	
	VERIFY_ARE_NOT_EQUAL((unsigned int)0, ip, L"GetReserveIp -ipvalue Passed");

	temp = inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN);
	VERIFY_IS_NOT_NULL(temp);
}

void TestReleaseIpReservation()
{
	unsigned int ip;
	DWORD ret;

	ret = ReleaseIpReservation("ip0");
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReleaseIpReservation",
		TraceLoggingValue("ip1", "ReservationId"),
		TraceLoggingValue(ret, "ReleaseIp Ret"));

	VERIFY_ARE_EQUAL((DWORD)0, ret, L"ReleaseIP for ip0 Passed");

	ret = GetReservedIpAddress("ip0", &ip);
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReleaseIpReservation",
		TraceLoggingValue("ip1", "ReservationId"),
		TraceLoggingValue(ret, "GetReservedIp Ret"));

	VERIFY_ARE_NOT_EQUAL((DWORD)0, ret, L"GetReserveAddr for Released Ip0 Passed");

	ret = ReleaseIpReservation("ip1");
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReleaseIpReservation",
		TraceLoggingValue("ip2", "ReservationId"),
		TraceLoggingValue(ret, "ReleaseIp Ret"));

	VERIFY_ARE_EQUAL((DWORD)0, ret, L"ReleaseIP for ip1 Passed");

	ret = GetReservedIpAddress("ip1", &ip);
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReleaseIpReservation",
		TraceLoggingValue("ip2", "ReservationId"),
		TraceLoggingValue(ret, "ReleaseIp Ret"));

	VERIFY_ARE_NOT_EQUAL((DWORD)0, ret, L"GetReserveAddr for Released Ip1 Passed");

}

void
NetworkClientTest::TestInitializeNetworkOverlay()
{
	ClusterConfig cfg;
	Ipv4Subnet localSubnet;
	struct in_addr ipAdd;
	unsigned long ret;

	SetEnvironment(OverLay);

	cfg.nodeCount = 1;
	cfg.nodeConfigs = (NodeConfig *)malloc(cfg.nodeCount * sizeof(NodeConfig));
	inet_pton(AF_INET, "10.10.1.0", &ipAdd);
	localSubnet.ipAddress = ipAdd.s_addr;
	localSubnet.prefixLength = 24;
	inet_pton(AF_INET, "11.1.0.144", &ipAdd);
	cfg.nodeConfigs[0].nodeIp = ipAdd.s_addr;
	cfg.nodeConfigs[0].nodeId = "Id114";
	cfg.nodeConfigs[0].subnet = &localSubnet;
	cfg.localNodeIp = ipAdd.s_addr;
	GetIpRepository()->InitRepositoryIpAllocations();
	ret = SaveOverlayAddressSpace(&cfg);
	VERIFY_ARE_EQUAL((unsigned long)0, ret, L"Overlay initialized passed");
	free(cfg.nodeConfigs);

}
/*
void
NetworkClientTest::TestGetLocalHostIpOverlay()
{
	TestGetLocalHostIp();
}
*/
void
NetworkClientTest::TestGetAvailableAddressOverlay()
{
	NodeNetworkResources nnr = { 0 };
	int ipCount=0;
	nnr.ips = NULL;

	ipCount = GetIpRepository()->PersistedIpCountInRepository();
	VERIFY_ARE_EQUAL(255, ipCount, L"GetAvailableAddress Overlay-Ipcount passed");

	nnr.ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));
	ipCount = GetIpRepository()->GetAvailableIpsfromRepository(nnr.ips);
	VERIFY_ARE_EQUAL(255, ipCount, L"Retrieving ips passed Overlay -Ipcount passed");	

	free(nnr.ips);
}

void
NetworkClientTest::TestReserveIpAddressOverlay()
{

	testReserveIp("ip0", OverLay);
	testReserveIp("ip1", OverLay);
}


void
NetworkClientTest::TestGetReservedIpAddressOverlay()
{
	TestGetReservedIpAddress();
}

void
NetworkClientTest::TestReleaseIpReservationOverlay()
{
	TestReleaseIpReservation();
}

void
NetworkClientTest::TestFreeStatusInRepsoitoryAllocations()
{
	int retval = GetIpRepository()->FreeStatusInRepositoryIpAllocationsUnTransacted();
	VERIFY_ARE_EQUAL(0, retval, L"Cleaning up status in repository allocations passed");	
}

void
NetworkClientTest::TestInitializeNetworkUnderlay()
{
	NodeNetworkResources nnr = { 0 };


	int ipCount = 0;
	nnr.ips = NULL;
	char *xmlResp = "<Interfaces><Interface MacAddress=\"000D3A72DC45\" IsPrimary=\"true\"><IPSubnet Prefix=\"11.1.0.0/24\"><IPAddress Address=\"11.1.0.74\" IsPrimary=\"true\"/><IPAddress Address=\"11.1.0.75\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.76\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.77\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.78\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.79\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.80\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.81\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.82\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.83\" IsPrimary=\"false\"/></IPSubnet></Interface></Interfaces>";
	
	SetEnvironment(UnderLay);
	GetIpRepository()->InitRepositoryIpAllocations();

	ipCount = ParseSecondaryIpAddresses(xmlResp, 1, nnr.ips);
	VERIFY_ARE_EQUAL(9, ipCount, L"Retrieving ipcount passed underlay");

	nnr.ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));
	ipCount = ParseSecondaryIpAddresses(xmlResp, 0, nnr.ips);
	VERIFY_ARE_EQUAL(9, ipCount, L"Retrieving ipcount with ips passed underlay");

	GetIpRepository()->UpdateRepositoryIpList(ipCount, nnr.ips, UnderLay);
	ipCount = GetIpRepository()->GetAvailableIpsfromRepository(nnr.ips);
	VERIFY_ARE_EQUAL(9, ipCount, L"initialization underlay passed");

	free(nnr.ips);

}

void
NetworkClientTest::TestGetLocalHostIpUnderlay()
{
	//TestGetLocalHostIp();
	char *xmlResp = "<Interfaces><Interface MacAddress=\"000D3A72DC45\" IsPrimary=\"true\"><IPSubnet Prefix=\"11.1.0.0/24\"><IPAddress Address=\"11.1.0.74\" IsPrimary=\"true\"/><IPAddress Address=\"11.1.0.75\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.76\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.77\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.78\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.79\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.80\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.81\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.82\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.83\" IsPrimary=\"false\"/></IPSubnet></Interface></Interfaces>";
	unsigned int ip=0;
	char *subnet=nullptr, *gateway=nullptr;
	ParseSubnetGatewayPrimaryIp(xmlResp, &subnet, &gateway, GetIpRepository());
	VERIFY_IS_NOT_NULL(subnet);
	VERIFY_IS_NOT_NULL(gateway);
	ip = GetIpRepository()->GetRepositoryPrimaryIp();
	//printf("Ip:%d\n",ip);
	VERIFY_ARE_NOT_EQUAL((unsigned int)0, ip,L"TestGetLocalHostIpUnderlay passed");
}

void
NetworkClientTest::TestReserveIpAddressUnderlay()
{

	testReserveIp("ip0", UnderLay);
	testReserveIp("ip1", UnderLay);
}


void
NetworkClientTest::TestGetReservedIpAddressUnderlay()
{
	TestGetReservedIpAddress();
}


void
NetworkClientTest::TestReleaseIpReservationUnderlay()
{
	TestReleaseIpReservation();
}


bool NetworkClientTest::TestSetup()
/*++

Routine Description:

    TAEF Setup. This API is called before invoking test APIs

Arguments:

    None

Return Value:

    None

--*/
{
    //
    // Initialize tracing.
    //

    TraceLoggingRegister(g_hMyComponentProvider);

    return true;
}

bool
NetworkClientTest::TestCleanup()
/*++

Routine Description:

    TAEF Cleanup. This API is called after invoking test APIs

Arguments:

    None

Return Value:

    None

--*/
{
   // RD_TRACE_TERMINATE();
	TraceLoggingUnregister(g_hMyComponentProvider);

    return true;

}

NetworkClientTest::NetworkClientTest()
/*++

Routine Description:

    This constructor initializes data members of the class as appropriate.

Arguments:

    None.

Return Value:

    None.

--*/
{
}


NetworkClientTest::~NetworkClientTest()
/*++

Routine Description:

    Destructor for NetworkClientTest

Arguments:

    None.

Return Value:

    None.

--*/
{

}

int wmain(int argc, wchar_t *argv[])
{
    UNREFERENCED_PARAMETER(argc);
    UNREFERENCED_PARAMETER(argv);
/*
    NetworkClientTest* clientTest = new NetworkClientTest();
    clientTest->TestSetup();
	clientTest->TestInitializeNetworkOverlay();
	clientTest->TestGetAvailableAddressOverlay();

	
    clientTest->TestInitializeNetwork();
	clientTest->TestReserveIpAddress();
	clientTest->TestGetReservedIpAddress();
	clientTest->TestReleaseIpReservation();
    clientTest->TestCleanup();*/

    return 0;
}