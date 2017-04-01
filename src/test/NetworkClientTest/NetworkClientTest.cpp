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
#include "aquanetworkclient.h"

#include "TraceLogging.h"

TRACELOGGING_DEFINE_PROVIDER(
    g_hMyComponentProvider,
    "SimpleTraceLoggingProvider",
    (0x3970f9cf, 0x2c0c, 0x4f11, 0xb1, 0xcc, 0xe3, 0xa1, 0xe9, 0x95, 0x88, 0x33));
	
using namespace std;
//using namespace Mock10;
using namespace WEX::TestExecution;




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
void testReserveIp(char *reservationId)
{
	unsigned int ip;
	NodeNetworkResources nnr;
	int ipCountBefore = -1;
	int ipCountAfter = -1;
	DWORD ret = 0;
	char ipInString[INET_ADDRSTRLEN] = { 0 };
	char availableIpString[INET_ADDRSTRLEN * 100] = { 0 };

	nnr.ips = NULL;

	ret = GetAvailableIpAddresses(&nnr, &ipCountBefore);
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReserveIpAddress",
		TraceLoggingValue(reservationId, "ReservationId"),
		TraceLoggingValue(ret, "GetAvailable Ret"),
		TraceLoggingValue(ipCountBefore, "IPCount Before"));

	VERIFY_ARE_EQUAL((DWORD)0, ret, L"GetAvailableAddress Passed");


	ret = ReserveAvailableIpAddress(reservationId, &ip);
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReserveIpAddress",
		TraceLoggingValue(reservationId, "ReservationId"),
		TraceLoggingValue(ret, "ReservedAvailable Ret")); 

	VERIFY_ARE_EQUAL((DWORD)0, ret, L"ReserveAvailableIpAddress for Ip0 Passed!");


	ret = GetAvailableIpAddresses(&nnr, &ipCountAfter);
	VERIFY_ARE_EQUAL((DWORD)0, ret, L"GetAvailableAddress Passed");
	VERIFY_ARE_EQUAL(ipCountAfter, ipCountBefore - 1, L"Ip Reservation Passed");

	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReserveIpAddress",
		TraceLoggingValue(reservationId, "ReservationId"),
		TraceLoggingValue(ret, "GetAvailable Ret"),
		TraceLoggingValue(ipCountBefore, "IPCount Before"),
		TraceLoggingValue(ipCountAfter, "IPCount After"));


	nnr.ips = (unsigned int *)malloc(ipCountAfter * sizeof(unsigned int));
	GetAvailableIpAddresses(&nnr, &ipCountAfter);
	VERIFY_IS_NOT_NULL(nnr.ips);
	

	for (int i = 0; i < ipCountAfter; i++)
	{
		inet_ntop(AF_INET, &nnr.ips[i], ipInString, INET_ADDRSTRLEN);
		strcat_s(availableIpString, sizeof(availableIpString), ipInString);
		strcat_s(availableIpString, sizeof(availableIpString),  ",");
	}

	TraceLoggingWrite(
		g_hMyComponentProvider,
		"TestReserveIpAddress",
		TraceLoggingValue(reservationId, "ReservationId"),
		TraceLoggingString(availableIpString, "AvailableIpList"));

}

void
NetworkClientTest::TestReserveIpAddress()
{

	testReserveIp("ip0");
	testReserveIp("ip1");
}

void 
NetworkClientTest::TestGetReservedIpAddress()
{
	unsigned int ip;
	DWORD ret;
	char ipInString[INET_ADDRSTRLEN];

	ret = GetReservedIpAddress("ip1", &ip);
	TraceLoggingWrite(
		g_hMyComponentProvider,
		"TestGetReservedIpAddress",
		TraceLoggingValue("ip1", "ReservationId"),
		TraceLoggingValue(ret, "GetReserved Ret"));

	VERIFY_ARE_EQUAL((DWORD)0, ret, L"GetReserveIp Passed");
	
	inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN);
	TraceLoggingWrite(
		g_hMyComponentProvider,
		"TestGetReservedIpAddress",
		TraceLoggingValue("ip1", "ReservationId"),
		TraceLoggingValue(ret, "GetReserved Ret"),
		TraceLoggingString(ipInString, "ReservedIpAddr"));


	ret = GetReservedIpAddress("ip2", &ip);
	TraceLoggingWrite(
		g_hMyComponentProvider,
		"TestGetReservedIpAddress",
		TraceLoggingValue("ip2", "ReservationId"),
		TraceLoggingValue(ret, "GetReserved Ret"));

	VERIFY_ARE_NOT_EQUAL((DWORD)0, ret, L"GetReserveIp Passed for not reserved IP");

	testReserveIp("ip2");
	
	ret = GetReservedIpAddress("ip2", &ip);
	TraceLoggingWrite(
		g_hMyComponentProvider,
		"TestGetReservedIpAddress",
		TraceLoggingValue("ip2", "ReservationId"),
		TraceLoggingValue(ret, "GetReserved Ret"));


	VERIFY_ARE_EQUAL((DWORD)0, ret, L"GetReserveIp Passed");
	inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN);
	TraceLoggingWrite(
		g_hMyComponentProvider,
		"TestGetReservedIpAddress",
		TraceLoggingValue("ip2", "ReservationId"),
		TraceLoggingValue(ret, "GetReserved Ret"),
		TraceLoggingString(ipInString, "ReservedIpAddr"));

}

void
NetworkClientTest::TestReleaseIpReservation()
{
	unsigned int ip;
	DWORD ret;

	ret = ReleaseIpReservation("ip1");
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReleaseIpReservation",
		TraceLoggingValue("ip1", "ReservationId"),
		TraceLoggingValue(ret, "ReleaseIp Ret"));

	VERIFY_ARE_EQUAL((DWORD)0, ret, L"ReleaseIP for ip1 Passed");

	ret = GetReservedIpAddress("ip1", &ip);
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReleaseIpReservation",
		TraceLoggingValue("ip1", "ReservationId"),
		TraceLoggingValue(ret, "GetReservedIp Ret"));

	VERIFY_ARE_NOT_EQUAL((DWORD)0, ret, L"GetReserveAddr for Released Ip1 Passed");

	ret = ReleaseIpReservation("ip2");
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReleaseIpReservation",
		TraceLoggingValue("ip2", "ReservationId"),
		TraceLoggingValue(ret, "ReleaseIp Ret"));

	VERIFY_ARE_EQUAL((DWORD)0, ret, L"ReleaseIP for ip2 Passed");

	ret = GetReservedIpAddress("ip2", &ip);
	TraceLoggingWrite(g_hMyComponentProvider,
		"TestReleaseIpReservation",
		TraceLoggingValue("ip2", "ReservationId"),
		TraceLoggingValue(ret, "ReleaseIp Ret"));

	VERIFY_ARE_NOT_EQUAL((DWORD)0, ret, L"GetReserveAddr for Released Ip2 Passed");

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

    NetworkClientTest* clientTest = new NetworkClientTest();
    clientTest->TestSetup();
    clientTest->TestInitializeNetwork();
	clientTest->TestReserveIpAddress();
	clientTest->TestGetReservedIpAddress();
	clientTest->TestReleaseIpReservation();
    clientTest->TestCleanup();

    return 0;
}

