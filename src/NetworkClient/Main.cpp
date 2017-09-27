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
#undef  __WFILE__
#undef __WFUNCTION__

#include "../Logging/Logging.h"
#include "../Server/RestServer.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;
using namespace Microsoft::Azure::Networking::Aqua::Server;

extern DllExport int DeleteLocalEndpointWithIp(wchar_t *hostLocalIp, bool detachIfHostLocalIp);

void assertvalue(unsigned long value)
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

void ExecuteIpScenarios(NodeNetworkResources *nnr)
{
	int i;
	int ipCount;
	char ipInString[22];
	int retval;
	unsigned hostIp;

	nnr->ips = NULL;
	assertvalue(GetAvailableIpAddresses(nnr, &ipCount));
	LOGINFO("Retrieved ipCount = %d\n", ipCount);
	
	nnr->ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));
	memset(nnr->ips, 0, ipCount * sizeof(unsigned int));
	assertvalue(GetAvailableIpAddresses(nnr, &ipCount));
	LOGINFO("Printing available ip addresses count = %d\n", ipCount);

	for (i = 0; i < ipCount; i++)
	{
		LOGERR("%s\n", inet_ntop(AF_INET, &nnr->ips[i], ipInString, INET_ADDRSTRLEN));
	}
	getchar();

	unsigned int ip=0;
	LOGINFO("going to reserve available ip with ip0");
	assertvalue(ReserveAvailableIpAddress((char *)"ip0", &ip));
	LOGINFO("ip0 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));

	assertvalue(GetAvailableIpAddresses(nnr, &ipCount));
	LOGINFO("Printing available ip addresses count = %d\n", ipCount);
	for(i = 0; i < ipCount ; i++)
	{
		LOGINFO("%s\n", inet_ntop(AF_INET, &nnr->ips[i], ipInString, INET_ADDRSTRLEN));
	}
	//getchar();
    
	assertvalue(ReserveAvailableIpAddress((char *)"ip1", &ip));
	LOGINFO("ip1 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

	assertvalue(GetAvailableIpAddresses(nnr, &ipCount));
	LOGINFO("Printing available ip addresses count = %d\n", ipCount);
	for(i = 0; i < ipCount ; i++)
	{
		LOGINFO(" %s\n", inet_ntop(AF_INET, &nnr->ips[i], ipInString, INET_ADDRSTRLEN));
	}
	getchar();

	assertvalue(GetReservedIpAddress("ip1", &ip));
	LOGINFO("ip1 (retrieving reserved ip) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

	retval = GetReservedIpAddress("ip2", &ip);
	LOGINFO("ip2 (without reservation). retval = %d\n", retval);
	getchar();
    
	assertvalue(ReserveAvailableIpAddress((char *)"ip2", &ip));
	LOGINFO("ip2 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

	assertvalue(GetAvailableIpAddresses(nnr, &ipCount));
	LOGINFO("Printing available ip addresses count = %d\n", ipCount);
	for(i = 0; i < ipCount ; i++)
	{
		LOGINFO("%s\n", inet_ntop(AF_INET, &nnr->ips[i], ipInString, INET_ADDRSTRLEN));
	}
	getchar();

	assertvalue(GetReservedIpAddress("ip2", &ip));
	LOGINFO("ip2 (retrieving reserved ip) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();


	assertvalue(ReleaseIpReservation("ip1"));
	retval = GetReservedIpAddress("ip1", &ip);
	LOGINFO("ip1 (without reservation). retval = %d\n", retval);
	getchar();


	assertvalue(ReserveAvailableIpAddress((char *)"ip5", &ip));
	LOGINFO("ip5 (after reservation) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

	assertvalue(GetReservedIpAddress((char*)"ip5", &ip));
	LOGINFO("ip5 (retrieving reserved ip) = %s\n", inet_ntop(AF_INET, &ip, ipInString, INET_ADDRSTRLEN));
	getchar();

	assertvalue(ReleaseIpReservation("ip2"));
	retval = GetReservedIpAddress("ip2", &ip);
	LOGINFO("ip2 (without reservation). retval =  %d\n", retval);
	getchar();

	assertvalue(ReleaseIpReservation("ip0"));
	retval = GetReservedIpAddress("ip0", &ip);
	LOGINFO("ip0 (without reservation). retval =  %d\n", retval);

	getchar();
	GetHostLocalIp(&hostIp);
	LOGINFO("Get local host ip (overlay):%d %s\n",hostIp,inet_ntop(AF_INET, &hostIp, ipInString, INET_ADDRSTRLEN));
	getchar();

	assertvalue(GetAvailableIpAddresses(nnr, &ipCount));
	LOGINFO("Printing available ip addresses count = %d\n", ipCount);
	for (i = 0; i < ipCount; i++)
	{
		LOGINFO("%s\n", inet_ntop(AF_INET, &nnr->ips[i], ipInString, INET_ADDRSTRLEN));
	}
	getchar();
	if(nnr->ips) free(nnr->ips);
}

void ExecuteOverlayScenario()
{
	ClusterConfig cfg;
	NodeNetworkResources nnr;
	char ipInString[INET_ADDRSTRLEN]={0};

	size_t netLen = strlen("azurenet") + sizeof(char);
	cfg.networkName = (char *)malloc(netLen);
	LOGINFO("%llu", netLen);
	strncpy_s(cfg.networkName, netLen, "azurenet", 9);
	cfg.networkName[8] = '\0';
	cfg.net = UnderLay;
	cfg.env = Azure;
	LOGINFO("nw main name:%s\n", cfg.networkName);
	
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

	unsigned int hostLocalIp = 0;
	GetHostLocalIp(&hostLocalIp);
	LOGINFO("Recevied host local as %s\n", inet_ntop(AF_INET, &hostLocalIp, ipInString, INET_ADDRSTRLEN));
	getchar();


	#ifdef _WIN32
	LOGINFO("Going to detach hostlocal directly in HNS\n");
	wchar_t ipInStringw[INET_ADDRSTRLEN];
	size_t converted;
	mbstowcs_s(&converted, ipInStringw, ipInString, INET_ADDRSTRLEN);
	DeleteLocalEndpointWithIp(ipInStringw, true);
	getchar();
	#endif
	
	ExecuteIpScenarios(&nnr);
}

void ExecuteUnderlayScenario()
{	
	ClusterConfig cfg;
	NodeNetworkResources nnr;
	char ipInString[INET_ADDRSTRLEN]={0};

	size_t netLen = strlen("azurenet") + sizeof(char);
	cfg.networkName = (char *)malloc(netLen);
	LOGINFO("%llu", netLen);
	strncpy_s(cfg.networkName, netLen, "azurenet", 9);
	cfg.networkName[8] = '\0';
	cfg.net = UnderLay;
	cfg.env = Azure;
	LOGINFO("nw main name:%s\n", cfg.networkName);

	assertvalue(InitializeNetwork(&cfg));
		
	unsigned int hostLocalIp=0;
	GetHostLocalIp(&hostLocalIp);
	LOGINFO("For Underlay: Recevied host local as %s\n", inet_ntop(AF_INET, &hostLocalIp, ipInString, INET_ADDRSTRLEN));
	getchar();	 

	ExecuteIpScenarios(&nnr);
	if(cfg.networkName) free(cfg.networkName);
}



void ProcessGet(
	http_request message)
	/*++

	Routine Description:

	Arguments:

	Return Value:

	0 if service successful, error code otherwise.

	--*/
{
	LOGINFO("Entering ProcessGet.");
	message.reply(status_codes::OK, "This is Aquarius.");
	LOGINFO("Returned OK as json response to GET from client");
	LOGINFO("Exiting ProcessGet.");
}


static void ProcessPost(
	http_request message)
	/*++

	Routine Description:

	Arguments:

	Return Value:
	N/A

	--*/
{
	LOGINFO("Entering ProcessPost.");
	pplx::task_status status = pplx::not_complete;
	pplx::task<json::value> js = message.extract_json();

	try
	{
		status = js.wait();
	}
	catch (const std::exception &ex)
	{
		LOGERR("Waiting for task (to extract json from POST) encountered exception. %s", ex.what());
	}

	bool failed = false;
	if (status != pplx::completed)
	{
		LOGERR("Task to extract json from GET did not finish. status: %d", status);
		failed = true;
	}
	else
	{
		json::value v = js.get();
		if (!v.is_null())
		{			

			failed = true;
			for (auto iter = v.as_object().cbegin(); iter != v.as_object().cend(); ++iter)
			{
				const utility::string_t &ca = iter->first;
				const json::value &macJSON = iter->second;
				LOGINFOW(L"Received the json on server as %ws %ws.", ca.c_str(), macJSON.as_string().c_str());
				failed = false;
			}
		}
		else
		{
			LOGINFO("Server received NULL/Invalid json. This API require json object as input.");
			failed = true;
		}
	}

	if (failed)
	{
		// did not receive even one ca-pa mapping
		json::value jsonString = web::json::value::string(L"Failed to process JSON.");
		message.reply(status_codes::BadRequest, jsonString);
	}
	else
	{
		json::value jsonString = web::json::value::string(L"Good");
		message.reply(status_codes::OK, jsonString);
	}
	
	LOGINFO("Exiting ProcessPost. Returning.");
}
void TestRestServer()
{
	RestServer rs(L"http", L"localhost", L"8000", L"", ProcessPost, NULL);
	DWORD retval = rs.Listen();
	printf("Server started: %ld\npress any key to exit..", retval);
	getchar();
}

#ifdef _WIN32
int wmain()
#endif
#ifdef __linux__
int main()
#endif
{
	//AquariusUtilities::WriteEventLogEntry(L"Working or not");
	//assertvalue(SetupNetworkEtwLogging(true));
	//assertvalue(SetupNetworkLogging(logstuff));	
	// ExecuteUnderlayScenario();
	// ExecuteOverlayScenario();
	TestRestServer();
	getchar();

	return 0;
}

