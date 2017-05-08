/*++

Copyright (c) Microsoft Corporation

Module Name:

Network.cpp

Abstract:

This file contains methods that can be used to setup container network in Windows.

Author:

Sushant Sharma (susharm) 30-March-2017

--*/

#ifdef _WIN32
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#include <stdio.h>
#include <iostream>
#include <mutex>
#include <stdlib.h>
#include <string.h>
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>
#include <iphlpapi.h>
#include <windows.h>

#pragma comment(lib, "iphlpapi.lib")

#include "../Common/aquanetworkclient.h"
#include "../Common/nclcore.h"
#include "../Common/NetworkCommon.h"
#include "../Common/ResponseParser.h"
#include "../../Logging/Logging.h"
#include "../../aquanix/Common/aquanixext.h"

extern DllExport unsigned long GetLocalEndpoints(EndpointMapping **endpointMappings, USHORT *mappingsCount);
extern DllExport int AddEndpoint(std::wstring endpointId, std::wstring macAddress, std::wstring ca, std::wstring pa, std::wstring isRemoteEndpoint);
extern DllExport int DeleteEndpoint(std::wstring endpointId);
extern DllExport int AddHostLocalEndpoint(std::wstring hostLocalIp);
extern DllExport unsigned long FreeMemoryForLocalEndpoints(EndpointMapping *endpointMappings, USHORT mappingsCount);

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {

extern NetworkType networkTypeProvided;
std::mutex hostLocalMutex;
bool hostLocalIpProvisionedInHNS = false;
std::mutex initWsaMutex;
int wsaInitialized = 0;

#pragma region private_methods
SOCKET connectToHost(
	char *host, 
	USHORT port)
/*++

Routine Description:

	Creates a socket and connects it to host and port.

Arguments:

	host - the host address to which connection needs to be made
	posrt - the port on which connection needs to be made

Return Value:

	The successfully connected socket.
	INVALID_SOCKET if connection could not be made.
--*/
{
	LOGINFO("Enetring connectToHost.");
	if(host == NULL)
	{
		LOGERR("NULL passed as host.");
		return INVALID_SOCKET;
	}

	struct hostent *hp;
	struct sockaddr_in addr;
	SOCKET sock;

	// Using depricated gethostbyname which is ipv4 only
	if ((hp = gethostbyname(host)) == nullptr)
	{
		LOGERR("gethostbyname failed %d : %s", WSAGetLastError(), host);
		return INVALID_SOCKET;
	}

	memcpy(&addr.sin_addr, hp->h_addr, hp->h_length);
	addr.sin_port = htons(port);
	addr.sin_family = AF_INET;
	sock = socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);

	if (sock == INVALID_SOCKET)
	{
		perror("setsockopt");
		LOGERR("socket call returned INVALID_SOCKET");
		return INVALID_SOCKET;
	}

	if (connect(sock, (struct sockaddr *)&addr, sizeof(struct sockaddr)) == -1)
	{
		int error = WSAGetLastError();
		if (error == WSAENETUNREACH)
		{
			LOGERR("Connection to host failed with WSAENETUNREACH.");
		}
		else
		{
			LOGERR("Connection to host failed with error code %d.", error);
		}
		return INVALID_SOCKET;
	}

	LOGINFO("Connected successfully to host.");
	return sock;
}


int InsertIntoBuffer(
	char **recvBuffer, 
	int byteCount, char c)
/*++

Routine Description:

	Inserts a character into buffer.

Arguments:

	recvBuffer - buffer
	byteCount - required byte count including the new character
	c - the new character to be inserted

Return Value:

	0 on failure, 1 on success;

--*/
{
	char *newBuffer = (char *)realloc((void *)(*recvBuffer), byteCount);
	if (newBuffer == nullptr)
	{
		return 0;
	}
	else
	{
		*recvBuffer = newBuffer;
	}
	(*recvBuffer)[byteCount - 1] = c;
	return 1;
}

int WsaInitialize()
/*++

Routine Description:

Arguments:

Return Value:

--*/
{
	WSADATA wsaData;
	int wsaerr;
	LOGINFO("Entering WsaInitialize.");
	wsaerr = WSAStartup(MAKEWORD(2, 2), &wsaData);

	if (wsaerr != 0)
	{		
		LOGERR("WSAStartup failed with error: %d\n", wsaerr);
	}
	else
	{
		LOGINFO("WSA initialization finished successfully.");
	}
	LOGINFO("Exiting WsaInitialize");
	return wsaerr;
}

char* sendPayloadOnSocket(
	SOCKET sock, 
	char *payload)
/*++

Routine Description:

	Sends a payload on socket and returns the response.

Arguments:
	
	sock - socket on which payload has to be sent.
	payload - payload to be sent

Return Value:

	response received if call is made.
	NULL if send failed or some other erro happenned during execution.

--*/
{

	int bytesRead;
	char *recvBuffer = nullptr;
	int byteCount = 0;
	LOGINFO("Entering sendPayloadOnSocket");
	int bytesWritten = send(sock, payload, (int)strlen(payload), 0);
	if (bytesWritten == SOCKET_ERROR)
	{
		LOGERR("Writing to socket failed errno = %d", errno);
	}
	else
	{
		LOGINFO("Sent %d bytes on the connection. Reading response (may take some time).... ", bytesWritten);
		do
		{
			char c;
			bytesRead = recv(sock, &c, 1, 0);
			if (bytesRead == 1)
			{
				byteCount++;
				if (!InsertIntoBuffer(&recvBuffer, byteCount, c))
				{
					LOGERR("Failed to insert a char into buffer");
					return NULL;
				}
			}
		} while (bytesRead > 0);

		if (bytesRead == -1)
		{
			LOGERR("recv call failed (returned -1). errno = %d", errno);
		}
		else if (bytesRead == 0)
		{
			LOGINFO("Reading response finished. Will insert EOL");
			if (!InsertIntoBuffer(&recvBuffer, byteCount + 1, '\0'))
			{
				LOGERR("Failed to insert EOL into buffer");
				return NULL;
			}
			return recvBuffer;
		}
	}
	LOGINFO("Exiting sendPayloadOnSocket");
	return NULL;
}

int retrieveHttpStatus(
	char *recvBuffer)
/*++

Routine Description:
	
	Retrieves http status from a http response.

Arguments:

	recvBuffer - http response that needs to be parsed

Return Value:

	response retrieved from the 

--*/
{
	int i = 0;
	int k = 0;
	char status[8];
	int ret = 0;
	
	LOGINFO("Entering retrieveHttpStatus");
	while (recvBuffer[i] != ' ')
	{
		i++;
	}

	i++;

	while (recvBuffer[i] != ' ')
	{
		status[k] = recvBuffer[i];
		k++;
		i++;
	}
	status[k] = '\0';
	if (k > 0)
	{
		ret = atoi(status);
	}
	LOGINFO("Returning %d as http response.", ret);
	return ret;
}

void printRoutingTable()
/*++

Routine Description:
	
	Prints routing table.

Arguments:

Return Value:

--*/
{
	PMIB_IPFORWARDTABLE pIpForwardTable = NULL;
	DWORD dwSize = 0;
	BOOL bOrder = FALSE;
	DWORD dwStatus;
	unsigned int i;

	LOGVERBOSE("===========Going to print routing table.============");

	// Find out how big our buffer needs to be.
	dwStatus = GetIpForwardTable(pIpForwardTable, &dwSize, bOrder);
	if (dwStatus == ERROR_INSUFFICIENT_BUFFER) 
	{
		// Allocate the memory for the table
		pIpForwardTable = (PMIB_IPFORWARDTABLE)malloc(dwSize);
		if (pIpForwardTable == NULL)
		{
			LOGWARN("Malloc failed. Out of memory, printRoutingTable.");
			return;
		}
		// Now get the table.
		dwStatus = GetIpForwardTable(pIpForwardTable, &dwSize, bOrder);
	}

	if (dwStatus != ERROR_SUCCESS) 
	{
		LOGWARN("getIpForwardTable failed");
		if (pIpForwardTable)
		{
			free(pIpForwardTable);
		}
		return;
	}

	for (i = 0; i < pIpForwardTable->dwNumEntries; i++) 
	{
		char mask[INET_ADDRSTRLEN];
		inet_ntop(AF_INET, &pIpForwardTable->table[i].dwForwardMask, mask, INET_ADDRSTRLEN);
		char dest[INET_ADDRSTRLEN];
		inet_ntop(AF_INET, &pIpForwardTable->table[i].dwForwardDest, dest, INET_ADDRSTRLEN);
		char nexthop[INET_ADDRSTRLEN];
		inet_ntop(AF_INET, &pIpForwardTable->table[i].dwForwardNextHop, nexthop, INET_ADDRSTRLEN);
		LOGVERBOSE("=> mask=%s fwAge=%d dest=%s if=%d nexthop=%s dwFP=%d dwFP=%d m1=%d m2=%d m3=%d m4=%d m5=%d, nhAS=%d, FT=%d, dwFT=%d, FP=%d\n", 
			mask,
			pIpForwardTable->table[i].dwForwardAge,
			dest,
			pIpForwardTable->table[i].dwForwardIfIndex,
			nexthop,			
			pIpForwardTable->table[i].dwForwardPolicy,
			pIpForwardTable->table[i].dwForwardProto,
			pIpForwardTable->table[i].dwForwardMetric1,
			pIpForwardTable->table[i].dwForwardMetric2,
			pIpForwardTable->table[i].dwForwardMetric3,
			pIpForwardTable->table[i].dwForwardMetric4,
			pIpForwardTable->table[i].dwForwardMetric5,
			pIpForwardTable->table[i].dwForwardNextHopAS,
			pIpForwardTable->table[i].ForwardType,
			pIpForwardTable->table[i].dwForwardType,
			pIpForwardTable->table[i].ForwardProto);
	}

	// Free resources
	if (pIpForwardTable)
	{
		free(pIpForwardTable);
	}

	LOGVERBOSE("===========Finished printing routing table.============");
}

int GetPrimaryCAIface(
	char *ca , 
	int *iface)
/*++

Routine Description:

	Get primary CA and the interface index for it.

Arguments:

	ca - primary CA to be filled out.
	iface - index of the interface that has primary ca.

Return Value:

--*/
{
	PIP_ADAPTER_INFO pAdapterInfo;
	PIP_ADAPTER_INFO pAdapter;
	DWORD dwRetVal;
	UINT i;
	int retval = -1;

	LOGINFO("Going to get primary CA via GetAdaptersInfo");

	ULONG ulOutBufLen = sizeof(IP_ADAPTER_INFO);
	pAdapterInfo = (IP_ADAPTER_INFO *)malloc(sizeof(IP_ADAPTER_INFO));
	if (pAdapterInfo == NULL) 
	{
		LOGERR("Error allocating memory needed to call GetAdaptersinfo.");
		return retval;
	}

	// Make an initial call to GetAdaptersInfo to get
	// the necessary size into the ulOutBufLen variable
	if (GetAdaptersInfo(pAdapterInfo, &ulOutBufLen) == ERROR_BUFFER_OVERFLOW) 
	{
		free(pAdapterInfo);
		pAdapterInfo = (IP_ADAPTER_INFO *)malloc(ulOutBufLen);
		if (pAdapterInfo == NULL) 
		{
			LOGERR("Error allocating memory needed to call GetAdaptersinfo.");
			return retval;
		}
	}

	if ((dwRetVal = GetAdaptersInfo(pAdapterInfo, &ulOutBufLen)) == NO_ERROR) 
	{
		pAdapter = pAdapterInfo;
		while (pAdapter) 
		{
			LOGVERBOSE("\tComboIndex: \t%d\n", pAdapter->ComboIndex);
			LOGVERBOSE("\tAdapter Name: \t%s\n", pAdapter->AdapterName);
			LOGVERBOSE("\tAdapter Desc: \t%s\n", pAdapter->Description);
			LOGVERBOSE("\tAdapter Addr: \t");
			for (i = 0; i < pAdapter->AddressLength; i++) 
			{
				if (i == (pAdapter->AddressLength - 1))
				{
					LOGVERBOSE("%.2X", (int)pAdapter->Address[i]);
				}
				else
				{
					LOGVERBOSE("%.2X-", (int)pAdapter->Address[i]);
				}
			}
			LOGVERBOSE("\tIndex: \t%d", pAdapter->Index);
			LOGVERBOSE("\tType: \t");
			switch (pAdapter->Type) {
			case MIB_IF_TYPE_OTHER:
				LOGVERBOSE("Other");
				break;
			case MIB_IF_TYPE_ETHERNET:
				LOGVERBOSE("Ethernet");
				break;
			case MIB_IF_TYPE_TOKENRING:
				LOGVERBOSE("Token Ring");
				break;
			case MIB_IF_TYPE_FDDI:
				LOGVERBOSE("FDDI");
				break;
			case MIB_IF_TYPE_PPP:
				LOGVERBOSE("PPP");
				break;
			case MIB_IF_TYPE_LOOPBACK:
				LOGVERBOSE("Lookback");
				break;
			case MIB_IF_TYPE_SLIP:
				LOGVERBOSE("Slip");
				break;
			default:
				LOGVERBOSE("Unknown type %ld", pAdapter->Type);
				break;
			}

			LOGVERBOSE("\tIP Address: \t%s\n", pAdapter->IpAddressList.IpAddress.String);
			LOGVERBOSE("\tIP Mask: \t%s\n", pAdapter->IpAddressList.IpMask.String);
			LOGVERBOSE("\tGateway: \t%s\n", pAdapter->GatewayList.IpAddress.String);
			LOGVERBOSE("\t***\n");

			if (pAdapter->DhcpEnabled) 
			{
				LOGVERBOSE("\tDHCP Enabled: Yes\n");
				LOGVERBOSE("\t  DHCP Server: \t%s\n", pAdapter->DhcpServer.IpAddress.String);
				if(strncmp(pAdapter->DhcpServer.IpAddress.String, "0.0.0.0", 7) != 0)
				{
					(*iface) = pAdapter->ComboIndex;
					strncpy_s(ca, 16, pAdapter->IpAddressList.IpAddress.String, 16);
					LOGINFO("Found primary CA: %s iface: %d\n", ca, *iface);
					retval = 0;
					break;
				}
			}
			else
			{
				LOGVERBOSE("\tDHCP Enabled: No");
			}

			pAdapter = pAdapter->Next;
		}
	}
	else 
	{
		LOGERR("GetAdaptersInfo failed with error: %d", dwRetVal);
	}

	if (pAdapterInfo) free(pAdapterInfo);

	LOGINFO("Finished retrieving primary CA via GetAdaptersInfo. Returning : %d", retval);
	return retval;
}

int createRouteToWireServer()
/*++

Routine Description:

	create a route to wireserver running on azure host.

Arguments:

Return Value:

--*/
{
	PMIB_IPFORWARDROW pRow;
	char primaryCa[16];
	int iFace;
	LOGINFO("Entering createRouteToWireServer. Going to get primary CA");

	if(GetPrimaryCAIface(primaryCa, &iFace) !=0)
	{
		LOGERR("Unable to get primary CA via get adapters");
		return -1;
	}

	LOGINFO("Successfully retrieved primary CA as %s iFace: %d", primaryCa, iFace);

	pRow = (PMIB_IPFORWARDROW)malloc(sizeof(MIB_IPFORWARDROW));
	if (!pRow)
	{
		LOGERR("Unable to allocate memory for pRow");
		return -1;
	}

	pRow->dwForwardDest = inet_addr("169.254.169.254");
	pRow->dwForwardNextHop = inet_addr(primaryCa);
	pRow->dwForwardPolicy = 0; // see http://www.ietf.org/rfc/rfc1354.txt
	pRow->dwForwardProto = MIB_IPPROTO_NETMGMT;
	pRow->dwForwardMask = inet_addr("255.255.255.255");
	pRow->dwForwardIfIndex = iFace;
	pRow->dwForwardMetric1 = 16; //see http://www.ietf.org/rfc/rfc1354.txt
	pRow->dwForwardType = MIB_IPROUTE_TYPE_DIRECT;
	DWORD retval = CreateIpForwardEntry(pRow);
	if(retval == ERROR_OBJECT_ALREADY_EXISTS)
	{
		LOGWARN("Unable to create route for WireServer because it already exists. Retval: %lu ", retval);
	}
	else
	if (retval != 0)
	{
		LOGERR("Unable to create route for WireServer. Error: %lu ", retval);
		if (pRow) free(pRow);
		return -1;
	}

	LOGINFO("Successfully created route for WireSererver.");
	if (pRow) free(pRow);
	LOGINFO("Exiting createRouteToWireServer.");
	return 0;
}

int getSubnetGwIpAndSetPrimary(
	char **subnetIp, 
	char **gwIp, 
	NclRepositoryManager *nclRepositoryManager)
/*++

Routine Description:

	retrieves interface descriptionf rom nm agent, and saves primary CA and gateway.

Arguments:

	subnetIp - subnet ip address
	gwIp - ip address of the gateway
	nclRepositoryManager - pointer to repository, it is where primary CA will be saved

Return Value:

	0 on success, 1 on failure.

--*/
{
	char *recvBuffer = nullptr;
	char *responseLocation = nullptr;
	int retval=0;

	LOGINFO("Entering getSubnetGwIpAndSetPrimary. Trying to connect to nm agent.");
	printRoutingTable();
	createRouteToWireServer();
	printRoutingTable();
	SOCKET sock = connectToHost("169.254.169.254", 80);

	if (sock == INVALID_SOCKET)
	{
		LOGERR("Connect to host failed. Returned INVALID_SOCKET.");
		return 1;
	}

	recvBuffer = sendPayloadOnSocket(sock, HostSuffix);
	if (!recvBuffer) {
		LOGERR("Error while trying to communicate with nmagent.");
		retval=1;
		goto cleanup_subgw;
	}

	int ret = retrieveHttpStatus(recvBuffer);
	LOGINFO("Http status retrieved from call to nmagent: %d", ret);
	if (ret != 200)
	{
		LOGERR(" Http status indicates failure. Invalid Status code: Complete Response: %s", recvBuffer);
		retval=1;
		goto cleanup_subgw;
	}

	responseLocation = strstr(recvBuffer, HttpResponseHeaderSeparator);
	responseLocation += strlen(HttpResponseHeaderSeparator);

	LOGINFO("HttpResponse from NMAgent: %s", responseLocation);

	if (responseLocation != nullptr)
	{
		ParseSubnetGatewayPrimaryIp(responseLocation, subnetIp, gwIp, nclRepositoryManager);
		if (*subnetIp == nullptr || *gwIp == nullptr)
		{
			retval = 1;
		}
	}
	else
	{
		LOGERR("Unable to find portion of http response header that contains valid nm agent response: %s", recvBuffer);
		retval=1;
	}

cleanup_subgw:
	if (recvBuffer)
	{
		free(recvBuffer);
		recvBuffer = nullptr;
	}

	shutdown(sock, SD_BOTH);
	closesocket((int)sock);
	return retval;
}

int networkExists(
	SOCKET sock,
	char *networkName)
/*++

Routine Description:
		
	Talks to docker and see if network exists or not.

Arguments:

	sock - socket that is connected to docker daemon
	networkName - name of the network that is to be verified for existence

Return Value:

	0 if network already exists, 1 if not or in case of error.

--*/
{
	char *payload = nullptr;
	char *recvBuf = nullptr;
	size_t payloadLen;

	char *httpreq = "HTTP/1.1\r\nHost:127.0.0.1\r\nContent-Type:application/json\r\nConnection:close\r\n\r\n";

	LOGINFO("Checking with docker if network already exists.. %s", networkName);

	payloadLen = strlen(DockerNwGet) + strlen(networkName) + strlen(httpreq);

	payload = (char *)malloc(payloadLen + 2);
	sprintf_s(payload, payloadLen + 2, "%s%s %s", DockerNwGet, networkName, httpreq);
	payload[payloadLen + 1] = '\0';
	recvBuf = sendPayloadOnSocket(sock, payload);
	if (!recvBuf)
	{
		LOGERR("Unexpected error while checking for existing network with docker. Network: %s", networkName);
		free(payload);
		return 0;
	}

	LOGINFO("Received response from docker: %s", recvBuf);
	int ret = retrieveHttpStatus(recvBuf);
	if (ret == 200)
	{
		LOGWARN("Docker responded with 200 : Network already exists.");
		if (payload) free(payload);
		free(recvBuf);
		return 0;
	}

	if (payload) free(payload);
	free(recvBuf);
	LOGINFO("Exiting networkExists with 1");
	return 1;
}

#pragma endregion private_methods

#pragma region public_methods
int GetIpsFromNmAgent(
	unsigned int *ips)
/*++

Routine Description:

	Retrieves interface config from nmagent.

Arguments:

	ips - if null, only ip count is returned. otherwise, all secondary ips are populated in it.

Return Value:

	count - number of secondary CAs from nm agent.

--*/
{
	int ipCount = 0;
	char *recvBuffer = nullptr;
	int bytesRead;
	char *responseLocation = nullptr;
	int byteCount = 0;
	
	LOGINFO("Entering GetIpsFromNmAgent.");
	AquaLogging::Log(info, logMsg);	
	printRoutingTable();
	createRouteToWireServer();
	printRoutingTable();
	SOCKET sock = connectToHost("169.254.169.254", 80);

	if (sock == INVALID_SOCKET)
	{
		LOGERR("Connect to 169.254.169.254 failed.");
		return -1;
	}

	LOGINFO("Now going to get ips from nm agent... fd: %llu", sock);

	int bytesWritten = send(sock, HostSuffix, (int)strlen(HostSuffix), 0);
	if (bytesWritten == SOCKET_ERROR)
	{		
		LOGERR("Writing to socket failed with errno = %d", errno);
	}
	else
	{
		LOGINFO("Sent %d bytes on the connection.\nNow reading into buffer..", bytesWritten);
		do
		{
			char c;
			bytesRead = recv(sock, &c, 1, 0);
			if (bytesRead == 1)
			{
				byteCount++;
				if (!InsertIntoBuffer(&recvBuffer, byteCount, c))
				{
					LOGERR("Failed to insert a char into buffer");
					ipCount = -1;
					goto cleanup;
				}
			}
		} while (bytesRead > 0);

		if (bytesRead == -1)
		{
			LOGERR("Reading failed. errno=%d ", errno);
		}
		else if (bytesRead == 0)
		{
			if (!InsertIntoBuffer(&recvBuffer, byteCount + 1, '\0'))
			{
				LOGERR("Failed to insert EOL into buffer");
				ipCount = -1;
				goto cleanup;
			}

			LOGINFO("Successfully finished reading. Printing response below..");
			responseLocation = strstr(recvBuffer, HttpResponseHeaderSeparator);
			responseLocation += strlen(HttpResponseHeaderSeparator);
			LOGINFO("Response: %s", responseLocation);
		}
		if (responseLocation != nullptr)
		{
			if (ips == NULL)
			{
				ipCount = ParseSecondaryIpAddresses(responseLocation, 1, ips);
				LOGINFO("Received only ip count as %d", ipCount);
			}
			else
			{
				// TODO: Check if ips have sufficient space
				ipCount = ParseSecondaryIpAddresses(responseLocation, 0, ips);
				LOGINFO("Received ips with ip count as %d\n", ipCount);
			}
		}
		else
		{
			LOGERR("Unable to find portion of http response header that contains valid nm agent response response %s", recvBuffer);
		}
		free(recvBuffer);
		recvBuffer = nullptr;
	}

cleanup:
	if (recvBuffer)
	{
		free(recvBuffer);
		recvBuffer = nullptr;
	}

	shutdown(sock, SD_BOTH);

	if (closesocket(sock) != 0)
	{
		LOGERR("+++++++++ERROR While closing Socket!+++++++++++++++++");
	}

	LOGINFO("Exiting GetIpsFromNmAgent.");
	return ipCount;
}

DWORD GetHostIp(
	unsigned int *ip, 
	NclRepositoryManager *nclRepositoryManager)
/*++

Routine Description:

	For overlay: It creates an endpoint in the host namespace, and attaches it to the Vnet (if not already done).
	For underlay, it returns the primary CA of primary nic
	
Arguments:

	ip - to be populated with the HostIp

Return Value:

	0 on success, error code otherwise

--*/
{	
	unsigned long retval = UNEXPECTED_ERROR;
	LOGINFO("Entering GetHostIp.");
	if(networkTypeProvided == UnderLay)
	{
		*ip = nclRepositoryManager->GetRepositoryPrimaryIp();
		LOGINFO("Selected Ip: %d ",*ip);
		if(*ip!=0)
			retval=0;
		else	
			retval=1;	
	}
	else if(networkTypeProvided == OverLay)
	{
		retval = ReserveAvailableIpAddress("HostLocalIp", ip);
		if (retval == 0)
		{
			if (!hostLocalIpProvisionedInHNS)
			{
				hostLocalMutex.lock();
				if (!hostLocalIpProvisionedInHNS)
				{
					char ipInString[INET_ADDRSTRLEN];
					wchar_t ipInStringw[INET_ADDRSTRLEN];
					inet_ntop(AF_INET, ip, ipInString, INET_ADDRSTRLEN);
					size_t converted;
					mbstowcs_s(&converted, ipInStringw, ipInString, INET_ADDRSTRLEN);
					AddHostLocalEndpoint(ipInStringw);
					hostLocalIpProvisionedInHNS = true;
				}
				hostLocalMutex.unlock();
			}
		}
		else
		{
			LOGERR("Reserving Available for host local returned non-zero %d", retval);
		}
	}
	return retval;
}


DWORD SetupUnderlayInAzure(
	ClusterConfig *clusterConfig, 
	NclRepositoryManager *nclRepositoryManager)
/*++

Routine Description:
	
	Sets up the underlay network in Azure.

Arguments:
	
	clusterConfig - contains name of the network to be created.
	nclRepositoryManager - repository where metadata about ntework will be persisted.

Return Value:
	0 on success, error code otherwise.

--*/
{
	size_t conLength = 0, ipamLength = 0;
	char *nwCreateReq = nullptr;
	char *recvBuf = nullptr;
	size_t totalLength = 0;
	char *subnetIp = nullptr;
	char *gwIp = nullptr;
	char ipamConfig[2048];
	unsigned long retval = 0;
	
	LOGINFO("Entering SetupUnderlayInAzure");

	SOCKET sock = connectToHost("localhost", 2375);
	if (sock == INVALID_SOCKET)
	{
		LOGERR("Connect to host failed for SetupUnderlayInAzure.");
		return UNREACHABLE_HOST;
	}

	if(getSubnetGwIpAndSetPrimary(&subnetIp, &gwIp, nclRepositoryManager)!=0)
	{
		LOGERR("Retrieving subnet and gateway ip failed during SetupUnderlayInAzure.");
		goto cleanup2;
	}

	if (networkExists(sock, clusterConfig->networkName))
	{
		LOGINFO("Network %s does not exist, going to create it.", clusterConfig->networkName);

		shutdown(sock, SD_BOTH);
		closesocket(sock);

		LOGINFO("Retrieved subnet and gateway %s %s", subnetIp, gwIp);

		sprintf_s(ipamConfig, sizeof(ipamConfig), "\"IPAM\":{\"Config\":[{\"Subnet\":\"%s\",\"Gateway\":\"%s\"}]}", subnetIp, gwIp);
		LOGINFO("ipamconfig : %s", ipamConfig);
		
		sock = connectToHost("localhost", 2375);
		ipamLength = strlen(ipamConfig);
		conLength = strlen(clusterConfig->networkName) + 9 + 19 + 4 + ipamLength;
		totalLength = strlen(DockerNwCreatePost) + conLength + ipamLength + 64;
		nwCreateReq = (char *)malloc(totalLength + 1);

		sprintf_s(nwCreateReq, totalLength + 1, 
			"%sContent-Length:%d\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s}", 
				DockerNwCreatePost, (int)conLength, clusterConfig->networkName, "l2tunnel", ipamConfig);

		LOGINFO("Now going to get create network driver... cmd:%s:%d:%llu:%d", 
			nwCreateReq, (int)totalLength, strlen(nwCreateReq), (int)conLength);
		recvBuf = sendPayloadOnSocket(sock, nwCreateReq);
		if (!recvBuf)
		{
			LOGERR("Error while creating docker network");
			retval = UNEXPECTED_ERROR;
			goto cleanup2;
		}

		int ret = retrieveHttpStatus(recvBuf);
		LOGINFO("Http status in response to docker network create: %d", ret);
		if (ret == 201)
		{
			LOGINFO("docker network create succeeded");
		}
		else
		{
			LOGERR("docker network create failed for netwprk: %s", recvBuf);
			retval = UNEXPECTED_ERROR;
		}
	}

cleanup2:
	if (nwCreateReq)
		free(nwCreateReq);
	if (recvBuf)
		free(recvBuf);
	if (subnetIp)
		free(subnetIp);
	if (gwIp)
		free(gwIp);

	shutdown(sock, SD_BOTH);
	closesocket(sock);
	LOGINFO("Exiting SetupUnderlayInAzure. Returning %d", retval);
	return retval;
}

unsigned long CreateDockerNetwork(
	char *networkName, 
	char ipamConfig[2048])
/*++

Routine Description:

	Creates a network with docker. The network driver is alwas l2tunnel and is not input.

Arguments:

	networkName - Name of the network to be created
	ipamConfig - Creation of a network requires subnet to be provided to ipam.

Return Value:

	0 on success, error code otherwise.

--*/
{
	LOGINFO("Entering CreateDockerNetwork");
	char *recvBuf = NULL;
	unsigned long retval = 0;

	SOCKET sock = connectToHost("localhost", 2375);
	if(sock == INVALID_SOCKET)
	{
		LOGERR("Docker daemon is unreachanble");
		return UNREACHABLE_DOCKER_DAEMON;
	}
	LOGINFO("Successfully connected to docker daemon.");

	size_t ipamLength = strlen(ipamConfig);
	size_t conLength = strlen(networkName) + 9 + 19 + 4 + ipamLength;
	size_t totalLength = strlen(DockerNwCreatePost) + conLength + ipamLength + 64;
	char *nwCreateReq = (char *)malloc(totalLength + 1);

	sprintf_s(nwCreateReq, totalLength + 1, 
	          "%sContent-Length:%d\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s}", 
	          DockerNwCreatePost, (int)conLength, networkName, "l2tunnel", ipamConfig);

	LOGINFO("Going to ask Docker to create network: %s:%d:%llu:%d",	nwCreateReq, (int)totalLength, strlen(nwCreateReq), (int)conLength);

	recvBuf = sendPayloadOnSocket(sock, nwCreateReq);
	if (!recvBuf)
	{
		LOGERR("Error while sending network create request to Docker.");
		retval = UNEXPECTED_ERROR;
	}
	else
	{
		int ret = retrieveHttpStatus(recvBuf);
		LOGINFO("Http status of the response received from docker : %d", ret);
		if (ret == 201)
		{
			LOGINFO("Docker network create succeeded.");
		}
		else
		{
			LOGERR("Docker network create did not succeded. Reponse: %s.", recvBuf);
			retval = UNEXPECTED_ERROR;
		}
	}

	if (recvBuf)
	{
		free(recvBuf);
	}
	if (nwCreateReq)
	{
		free(nwCreateReq);
	}

	shutdown(sock, SD_BOTH);
	closesocket(sock);

	LOGINFO("Exiting CreateDockerNetwork. Returning: %u", retval);
	return retval;
}

DWORD SetupOverlay(
	ClusterConfig *clusterConfig)
/*++

Routine Description:

	Sets up the overlay network.
	Asks docker to create a network using the overlay driver that is expected to be already installed.

Arguments:

	clusterConfig - The configuration required to setup the overlay.

Return Value:

	0 on success, error code otherwise.

--*/
{
	char *recvBuf = nullptr;
	char *gwIp = nullptr;
	char ipamConfig[2048];
	unsigned long retval = 0;
	
	LOGINFO("Entering SetupOverlay.");

	if (clusterConfig->networkName == NULL)
	{
		LOGERR("Network name not specified as input to SetupOverlay.");
		return UNSPECIFIED_NETWORK_NAME;
	}
	
	SOCKET sock = connectToHost("localhost", 2375);
	if (sock == INVALID_SOCKET)
	{
		LOGERR("Connect to docker daemon failed for SetupOverlay. Returned INVALID_SOCKET.");
		return UNREACHABLE_DOCKER_DAEMON;
	}

	if (networkExists(sock, clusterConfig->networkName))
	{
		LOGINFO("Overlay Network with name %s does not exist. Going to create it.", clusterConfig->networkName);

		shutdown(sock, SD_BOTH);
		closesocket(sock);

		if(clusterConfig->overlaySubnet == NULL)
		{
			LOGERR("Subnet not specified while creating overlay network.");
			return OVERLAY_SUBNET_NOT_SPECIFIED;
		}

		if(clusterConfig->overlaySubnet->prefixLength > 30)
		{
			LOGERR("Subnet size is too small. Prefix specified: %d.", clusterConfig->overlaySubnet->prefixLength);
			return SUBNET_SIZE_TOO_SMALL;
		}

		LOGINFO("Going to retrieve gateway ip from subnet: %u", clusterConfig->overlaySubnet->ipAddress);
		char gatewayIp[INET_ADDRSTRLEN];
		if(inet_ntop(AF_INET, &clusterConfig->overlaySubnet->ipAddress, gatewayIp, INET_ADDRSTRLEN)==NULL)
		{
			LOGERR("Subnet ip could not be converted from ntop. Subnet ip specified: %u", clusterConfig->overlaySubnet->ipAddress);
			return MALFORMED_SUBNET_IP;
		}

		if(NetworkCommon::RetrieveGatewayFromSubnet(gatewayIp, &gwIp) == false)
		{
			LOGERR("Gateway ip could not be retrieved from Subnet ip specified: %u", clusterConfig->overlaySubnet->ipAddress);
			return MALFORMED_SUBNET_IP;
		}

		LOGINFO("Successfully retrieved Subnet and Gateway %s/%d %s", gatewayIp, clusterConfig->overlaySubnet->prefixLength, gwIp);

		sprintf_s(ipamConfig, sizeof(ipamConfig), 
			"\"IPAM\":{\"Config\":[{\"Subnet\":\"%s/%d\",\"Gateway\":\"%s\"}]}", 
			gatewayIp, clusterConfig->overlaySubnet->prefixLength, gwIp);

		LOGINFO("Constructed input for ipam (to be given as input to docker network create): %s", ipamConfig);

		retval = CreateDockerNetwork(clusterConfig->networkName, ipamConfig);
	}
	else 
	{
		if (recvBuf)
		{
			free(recvBuf);
			recvBuf = NULL;
		}
		shutdown(sock, SD_BOTH);
		closesocket(sock);
	}
	if (gwIp)
	{
		free(gwIp);
		gwIp = NULL;
	}

	LOGINFO("Exiting SetupOverlay. Returning %u.", retval);
	return retval;
}
#pragma endregion public_methods

#pragma region overlay_methods
DWORD installHNSOverlayPlugin()
/*++

Routine Description:

	Execute powershell script to install HNSOverlayPlugin

Arguments:

Return Value:

--*/
{
	int retval;

	system("start powershell.exe Set-ExecutionPolicy RemoteSigned \n");
	retval = system("start powershell.exe InstallPlugin.ps1");

	return retval;
}

DWORD AddRemoteMapping(
	const char *endpointId, 
	const char *hostAddress, 
	const char *containerAddress, 
	const char *macAddress)
/*++

Routine Description:

		Adds a remote CA-PA mapping.

Arguments:

Return Value:

	0 on success.

--*/
{	 
	LOGINFO("Entering AddRemoteMapping.");
	LOGINFO("%s %s %s %s\n", endpointId, hostAddress, containerAddress, macAddress);

	wchar_t endPointIdW[150]; //GUID cannot exceed 38
	size_t converted;
	mbstowcs_s(&converted, endPointIdW, endpointId, 150);
	
	wchar_t hostAddressW[INET_ADDRSTRLEN];
	mbstowcs_s(&converted, hostAddressW, hostAddress, INET_ADDRSTRLEN);

	wchar_t containerAddressW[INET_ADDRSTRLEN]; //GUID cannot exceed 38
	mbstowcs_s(&converted, containerAddressW, containerAddress, INET_ADDRSTRLEN);

	wchar_t macAddressW[50]; //MAC cannot exceed 24
	mbstowcs_s(&converted, macAddressW, macAddress, 50);

	LOGINFO("%s %s %s %s\n", endpointId, hostAddress, containerAddress, macAddress);
	DWORD retval = AddEndpoint(endPointIdW, macAddressW, containerAddressW, hostAddressW, L"true");
	LOGINFO("Exiting AddRemoteMapping. Returning %u.", retval);
	return retval;
}

DWORD DeleteRemoteMapping (
	const char *endpointId)
/*++

Routine Description:
	
	Deletes a remote CA-PA mapping.

Arguments:

	endpointId - mapping Id that needs to be removed.

Return Value:

	0 on success, error code otherwise.

--*/
{
	wchar_t endpointIdW[150]; //GUID cannot exceed 38
	size_t converted;	
	LOGINFO("Entering DeleteRemoteMapping.");
	mbstowcs_s(&converted, endpointIdW, endpointId, 150);
	DWORD retval =  DeleteEndpoint(endpointIdW);
	LOGINFO("Exiting DeleteRemoteMapping. Returning; %u", retval);
	return retval;
}


DWORD GetLocalMappings(
	EndpointMapping **endpointMappings, 
	USHORT *mappingsCount)
/*++

Routine Description:
	
	Retrive the local endpoint mappings.

Arguments:
	
	endpointMappings - out parameter
	mappingsCount - out parameter

Return Value:

--*/
{
	AquaLogging::Log(info, "Enter GetLocalMappings");
	GetLocalEndpoints(endpointMappings, mappingsCount);
	AquaLogging::Log(info, "Exit GetLocalMappings");
	return 0;
}

DWORD FreeEndpointMappings(
	EndpointMapping *endpointMappings, 
	USHORT mappingsCount)
/*++

Routine Description:
	
	Free the memory allocated in the object

Arguments:
	
	endpointMappings - pointer pointing ot the memry to be freed
	mappingsCount - how many mappings are to be freed

Return Value:
	
	0 on success

--*/
{
	AquaLogging::Log(info, "Enter FreeEndpointMappings");
	FreeMemoryForLocalEndpoints(endpointMappings, mappingsCount);
	AquaLogging::Log(info, "Enter FreeEndpointMappings");
	return 0;
}

void ProvideConfigToNIX(
	ClusterConfig *clusterConfig)
/*++

Routine Description:

	Provides cluster config to NIX to setup overlay.

Arguments:
	
	clusterConfig - configuration of the cluster containing CAs and PAs

Return Value:

--*/
{
	LOGINFO("Entering ProvideConfigToNIX");
	unsigned int localIp = clusterConfig->localNodeIp;
	char ipInString[INET_ADDRSTRLEN];
	inet_ntop(AF_INET, &localIp, ipInString, INET_ADDRSTRLEN);
	char**remoteHostAddresses = new char *[clusterConfig->nodeCount - 1]; //-1 for local address
	USHORT cntr = 0;
	for (int i = 0; i < clusterConfig->nodeCount; i++)
	{
		
		NodeConfig cnc = clusterConfig->nodeConfigs[i];
		char remoteIpInString[INET_ADDRSTRLEN];
		inet_ntop(AF_INET, &cnc.nodeIp, remoteIpInString, INET_ADDRSTRLEN);

		LOGINFO("Cluster config remote ip = %s %s %u---------", remoteIpInString, cnc.nodeId, cnc.subnet->ipAddress);
		if (cnc.nodeIp != clusterConfig->localNodeIp)
		{
			char remoteIpInString2[INET_ADDRSTRLEN];
			inet_ntop(AF_INET, &cnc.nodeIp, remoteIpInString2, INET_ADDRSTRLEN);
			remoteHostAddresses[cntr] = (char *) malloc(sizeof(char)* INET_ADDRSTRLEN);
			strcpy_s(remoteHostAddresses[cntr], INET_ADDRSTRLEN, remoteIpInString2);
			LOGINFO("Saved remote ip = %s", remoteHostAddresses[cntr]);
			cntr++;
		}
	}

	LOGINFO("Local Host provided to NIX: %s", ipInString);
	for(int j=0; j<cntr; j++)
	{
		LOGINFO("Remote Host %d provided to NIX: %s", j, remoteHostAddresses[j]);
	}

	InitializeAquaNix(ipInString, remoteHostAddresses, cntr, "127.0.0.1", AddRemoteMapping, DeleteRemoteMapping, 
		GetLocalMappings, FreeEndpointMappings, 10987, 10987);
	RequestRemoteMappings();
	for(int k = 0; k<cntr; k++)
	{
		free(remoteHostAddresses[k]);
	}
	delete []remoteHostAddresses;
	LOGINFO("Exiting ProvideConfigToNIX.");
}
#pragma endregion overlay_methods
} } } } }
#endif