#include <cppunit/TestCase.h>
#include <cppunit/TestFixture.h>
#include <cppunit/ui/text/TextTestRunner.h>
#include <cppunit/extensions/HelperMacros.h>
#include <cppunit/extensions/TestFactoryRegistry.h>
#include <cppunit/TestResult.h>
#include <cppunit/TestResultCollector.h>
#include <cppunit/TestRunner.h>
#include <cppunit/BriefTestProgressListener.h>
#include <cppunit/CompilerOutputter.h>
#include <cppunit/XmlOutputter.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include<stdlib.h>
#include "../../NetworkClientLibrary/Common/nclcore.h"
#include "../../NetworkClientLibrary/Common/aquanetworkclient.h"

using namespace CppUnit;
using namespace std;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;
//-----------------------------------------------------------------------------

class NetworkClientTest : public CppUnit::TestFixture
{
    CPPUNIT_TEST_SUITE(NetworkClientTest);
	CPPUNIT_TEST(TestInitializeNetworkOverlay);
	CPPUNIT_TEST(TestGetAvailableAddressOverlay);
    CPPUNIT_TEST(TestGetLocalHostIpOverlay);
    CPPUNIT_TEST(TestReserveIpAddressOverlay);
    CPPUNIT_TEST(TestGetReservedIpAddressOverlay);
    CPPUNIT_TEST(TestReleaseIpReservationOverlay);
	CPPUNIT_TEST(TestFreeStatusInRepsoitoryAllocations);
	
    CPPUNIT_TEST(TestInitializeNetworkUnderlay);
    CPPUNIT_TEST(TestGetLocalHostIpUnderlay);
    CPPUNIT_TEST(TestReserveIpAddressUnderlay);
    CPPUNIT_TEST(TestGetReservedIpAddressUnderlay);
    CPPUNIT_TEST(TestReleaseIpReservationUnderlay);
	CPPUNIT_TEST(TestMemLeak);
    CPPUNIT_TEST_SUITE_END();

public:
    void setUp(void);
    void tearDown(void);

protected:
	void TestInitializeNetworkOverlay();
	void TestGetAvailableAddressOverlay();
    void TestGetLocalHostIpOverlay();
    void TestReserveIpAddressOverlay();
    void TestGetReservedIpAddressOverlay();	
    void TestReleaseIpReservationOverlay();
	void TestFreeStatusInRepsoitoryAllocations();
	
    void TestInitializeNetworkUnderlay();
    void TestGetLocalHostIpUnderlay();
    void TestReserveIpAddressUnderlay();
    void TestGetReservedIpAddressUnderlay();	
    void TestReleaseIpReservationUnderlay();

	void TestMemLeak();

private:

};
int getAvailableIpCountOverlay()
{
	return GetAvailableIpsfromRepository(NULL);
}

int getAvailableIpCountUnderlay()
{
	int ipCount = 0;
	NodeNetworkResources nnr = { 0 };
	char *xmlResp = (char *)"<Interfaces><Interface MacAddress=\"000D3A72DC45\" IsPrimary=\"true\"><IPSubnet Prefix=\"11.1.0.0/24\"><IPAddress Address=\"11.1.0.74\" IsPrimary=\"true\"/><IPAddress Address=\"11.1.0.75\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.76\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.77\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.78\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.79\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.80\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.81\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.82\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.83\" IsPrimary=\"false\"/></IPSubnet></Interface></Interfaces>";

	ipCount = ParseSecondaryIpAddresses(xmlResp, 1, NULL);

	nnr.ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));

	ipCount = ParseSecondaryIpAddresses(xmlResp, 0, nnr.ips);
	UpdateRepositoryIpList(ipCount, nnr.ips, UnderLay);
	ipCount = GetAvailableIpsfromRepository(nnr.ips);

	free(nnr.ips);
	return ipCount;
}

void TestGetLocalHostIp()
{
	DWORD ret = 0;
	unsigned int ip=0;
	ret = GetHostLocalIp(&ip);
	CPPUNIT_ASSERT(ret==0);
	CPPUNIT_ASSERT(ip!=0);
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

	CPPUNIT_ASSERT(ipCountBefore>0);
	

	ret = ReserveRepositoryIpAddress(reservationId);

	CPPUNIT_ASSERT(ret!=0);

	if (type == OverLay)
		ipCountAfter = getAvailableIpCountOverlay();
	else
		ipCountAfter = getAvailableIpCountUnderlay();

	CPPUNIT_ASSERT(ipCountAfter == ipCountBefore-1);
}

void TestGetReservedIpAddress()
{
	unsigned int ip=0;
	DWORD ret;
	char ipInString[INET_ADDRSTRLEN];
	const char *temp = NULL;

	ret = GetReservedIpAddress("ip1", &ip);

	CPPUNIT_ASSERT(ret==0);		
	CPPUNIT_ASSERT(ip!=0);

	temp = inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN);
	CPPUNIT_ASSERT(temp!=NULL);
}

void TestReleaseIpReservation()
{
	unsigned int ip;
	DWORD ret;

	ret = ReleaseIpReservation("ip0");
	CPPUNIT_ASSERT(ret==0);
	
	ret = GetReservedIpAddress("ip0", &ip);
	CPPUNIT_ASSERT(ret!=0);

	ret = ReleaseIpReservation("ip1");
	CPPUNIT_ASSERT(ret==0);
	
	ret = GetReservedIpAddress("ip0", &ip);
	CPPUNIT_ASSERT(ret!=0);

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
	cfg.nodeConfigs[0].nodeId = (char *)"Id114";
	cfg.nodeConfigs[0].subnet = &localSubnet;
	cfg.localNodeIp = ipAdd.s_addr;
	cfg.net=OverLay;
	InitRepositoryIpAllocations();
	SetRepositoryPrimaryIp(ipAdd.s_addr);
	ret = SaveOverlayAddressSpace(&cfg);
	CPPUNIT_ASSERT(ret==0);
	free(cfg.nodeConfigs);

}

void
NetworkClientTest::TestGetLocalHostIpOverlay()
{
	TestGetLocalHostIp();
}

void
NetworkClientTest::TestGetAvailableAddressOverlay()
{
	NodeNetworkResources nnr = { 0 };
	int ipCount=0;
	nnr.ips = NULL;

	ipCount = PersistedIpCountInRepository();
	CPPUNIT_ASSERT(ipCount==255);

	nnr.ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));
	ipCount = GetAvailableIpsfromRepository(nnr.ips);
	free(nnr.ips);
	CPPUNIT_ASSERT(ipCount==253);
}

void
NetworkClientTest::TestReserveIpAddressOverlay()
{

	testReserveIp((char *)"ip0", OverLay);
	testReserveIp((char *)"ip1", OverLay);
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
	int retval = FreeStatusInRepositoryIpAllocations();
	CPPUNIT_ASSERT(retval==0);	
}

//-----------------------------------------------------------------------------
/*
void
NetworkClientTest::testAddition(void)
{
    CPPUNIT_ASSERT(5 == Addition(2,3));
}
*/
void
NetworkClientTest::TestInitializeNetworkUnderlay()
{
	NodeNetworkResources nnr = { 0 };


	int ipCount = 0;
	nnr.ips = NULL;
 	const char *xmlResp = (char *)"<Interfaces><Interface MacAddress=\"000D3A72DC45\" IsPrimary=\"true\"><IPSubnet Prefix=\"11.1.0.0/24\"><IPAddress Address=\"11.1.0.74\" IsPrimary=\"true\"/><IPAddress Address=\"11.1.0.75\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.76\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.77\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.78\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.79\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.80\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.81\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.82\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.83\" IsPrimary=\"false\"/></IPSubnet></Interface></Interfaces>";
	
	SetEnvironment(UnderLay);
	InitRepositoryIpAllocations();

	ipCount = ParseSecondaryIpAddresses((char *)xmlResp, 1, nnr.ips);
	CPPUNIT_ASSERT(9 == ipCount);

	nnr.ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));
	ipCount = ParseSecondaryIpAddresses((char *)xmlResp, 0, nnr.ips);
	CPPUNIT_ASSERT(9 == ipCount);
	UpdateRepositoryIpList(ipCount, nnr.ips, UnderLay);
	ipCount = GetAvailableIpsfromRepository(nnr.ips);
	CPPUNIT_ASSERT(9 == ipCount);
	free(nnr.ips);

}

void
NetworkClientTest::TestGetLocalHostIpUnderlay()
{
	//TestGetLocalHostIp();
	char *xmlResp = (char *)"<Interfaces><Interface MacAddress=\"000D3A72DC45\" IsPrimary=\"true\"><IPSubnet Prefix=\"11.1.0.0/24\"><IPAddress Address=\"11.1.0.74\" IsPrimary=\"true\"/><IPAddress Address=\"11.1.0.75\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.76\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.77\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.78\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.79\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.80\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.81\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.82\" IsPrimary=\"false\"/><IPAddress Address=\"11.1.0.83\" IsPrimary=\"false\"/></IPSubnet></Interface></Interfaces>";
	unsigned int ip=0;
	char *subnet=NULL, *gateway=NULL;
	ParseSubnetGatewayPrimaryIp(xmlResp, &subnet, &gateway);
	CPPUNIT_ASSERT(subnet!=NULL);
	CPPUNIT_ASSERT(gateway!=NULL);
	free(subnet);
	free(gateway);
	ip = GetRepositoryPrimaryIp();
	//printf("Ip:%d\n",ip);
	CPPUNIT_ASSERT(ip!=0);
}

void
NetworkClientTest::TestReserveIpAddressUnderlay()
{

	testReserveIp((char*)"ip0", UnderLay);
	testReserveIp((char *)"ip1", UnderLay);
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

void
NetworkClientTest::TestMemLeak()
{
	int i;
	for(i=0;i<5;i++)
	{
		TestInitializeNetworkOverlay();
		TestGetAvailableAddressOverlay();
		TestGetLocalHostIpOverlay();
		TestReserveIpAddressOverlay();
		TestGetReservedIpAddressOverlay();	
		TestReleaseIpReservationOverlay();
		TestFreeStatusInRepsoitoryAllocations();
		TestInitializeNetworkUnderlay();
		TestGetLocalHostIpUnderlay();
		TestReserveIpAddressUnderlay();
		TestGetReservedIpAddressUnderlay();	
		TestReleaseIpReservationUnderlay();
	}
}

void logstuff(int level, char *text)
{
	if(level<4)
		printf("%d: %s\n", level, text);
}

void NetworkClientTest::setUp(void)
{
//    mTestObj = new CBasicMath();
	SetupNetworkLogging(logstuff);

}

void NetworkClientTest::tearDown(void)
{
  //  delete mTestObj;
}

//-----------------------------------------------------------------------------

CPPUNIT_TEST_SUITE_REGISTRATION( NetworkClientTest );

int main(int argc, char* argv[])
{
    // informs test-listener about testresults
    CPPUNIT_NS::TestResult testresult;

    // register listener for collecting the test-results
    CPPUNIT_NS::TestResultCollector collectedresults;
    testresult.addListener (&collectedresults);

    // register listener for per-test progress output
    CPPUNIT_NS::BriefTestProgressListener progress;
    testresult.addListener (&progress);

    // insert test-suite at test-runner by registry
    CPPUNIT_NS::TestRunner testrunner;
    testrunner.addTest (CPPUNIT_NS::TestFactoryRegistry::getRegistry().makeTest ());
    testrunner.run(testresult);

    // output results in compiler-format
    CPPUNIT_NS::CompilerOutputter compileroutputter(&collectedresults, std::cerr);
    compileroutputter.write ();

    // Output XML for Jenkins CPPunit plugin
    ofstream xmlFileOut("NetworkClientTest.xml");
    XmlOutputter xmlOut(&collectedresults, xmlFileOut);
    xmlOut.write();

    // return 0 if tests were successful
    return collectedresults.wasSuccessful() ? 0 : 1;
}
