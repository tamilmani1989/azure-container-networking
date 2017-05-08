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
#include "../../Logging/Logging.h"
#include "../../aquanix/Common/aquanixext.h"
#include "../Common/ResponseParser.h"

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

SOCKET connectToHost(char *host, USHORT port)
{
	
	size_t msgSize = 2048;
	struct hostent *hp;
	struct sockaddr_in addr;
	SOCKET sock;

	// Using depricated gethostbyname which is ipv4 only
	if ((hp = gethostbyname(host)) == nullptr)
	{
		sprintf_s(logMsg, msgSize,  "gethostbyname failed %d : %s", WSAGetLastError(), host);
		AquaLogging::Log(err, logMsg);
		return INVALID_SOCKET;
	}

	memcpy(&addr.sin_addr, hp->h_addr, hp->h_length);
	addr.sin_port = htons(port);
	addr.sin_family = AF_INET;
	sock = socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);

	if (sock == INVALID_SOCKET)
	{
		perror("setsockopt");
		return INVALID_SOCKET;
	}

	if (connect(sock, (struct sockaddr *)&addr, sizeof(struct sockaddr)) == -1)
	{
		int error = WSAGetLastError();
		if (error == WSAENETUNREACH)
		{
			sprintf_s(logMsg, msgSize,  "Connection to host failed with WSAENETUNREACH.");
			AquaLogging::Log(err, logMsg);
		}
		else
		{
			sprintf_s(logMsg, msgSize,  "Connection to host failed with error code %d.", error);
			AquaLogging::Log(ERROR, logMsg);
		}
		return INVALID_SOCKET;
	}

	sprintf_s(logMsg, msgSize, "Connected successfully to host.\n");
	AquaLogging::Log(info, logMsg);
	return sock;
}


int InsertIntoBuffer(char **recvBuffer, int byteCount, char c)
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
{
	
	size_t msgSize = 2048;
	WSADATA wsaData;
	int wsaerr;

	wsaerr = WSAStartup(MAKEWORD(2, 2), &wsaData);

	if (wsaerr != 0)
	{		
		sprintf_s(logMsg, msgSize, "WSAStartup failed with error: %d\n", wsaerr);
		AquaLogging::Log(info, logMsg);		
	}
	else
	{
		sprintf_s(logMsg, msgSize, "WSA initialization finished successfully. \n");
		AquaLogging::Log(info, logMsg);
	}
	
	return wsaerr;
}

void wsaCleanUp(){}

char* httpSendResponse(SOCKET sock, char *payload)
{

	int bytesRead;
	char *recvBuffer = nullptr;
	int byteCount = 0;
	
	size_t msgSize = 2048;

	int bytesWritten = send(sock, payload, (int)strlen(payload), 0);
	if (bytesWritten == SOCKET_ERROR)
	{
		sprintf_s(logMsg, msgSize,  "Writing to socket failed errno=%d\n", errno);
		AquaLogging::Log(info, logMsg);
	}
	else
	{
		sprintf_s(logMsg, msgSize, "Sent %d bytes on the connection.\nReading response (may take some time).... ", bytesWritten);
		AquaLogging::Log(info, logMsg);
		do
		{
			char c;
			bytesRead = recv(sock, &c, 1, 0);
			if (bytesRead == 1)
			{
				byteCount++;
				if (!InsertIntoBuffer(&recvBuffer, byteCount, c))
				{
					sprintf_s(logMsg, msgSize,  "Failed to insert a char into buffer");
					AquaLogging::Log(err, logMsg);
					return NULL;
				}
			}
		} while (bytesRead > 0);

		if (bytesRead == -1)
		{
			sprintf_s(logMsg, msgSize,  "Reading failed. errno=%d \n", errno);
			AquaLogging::Log(err, logMsg);
		}
		else if (bytesRead == 0)
		{
			sprintf_s(logMsg, msgSize, "Read finished\n");
			if (!InsertIntoBuffer(&recvBuffer, byteCount + 1, '\0'))
			{
				sprintf_s(logMsg, msgSize,  "Failed to insert EOL into buffer");
				AquaLogging::Log(err, logMsg);
				return NULL;
			}
			return recvBuffer;
		}
	}
	return NULL;
}

int retrieveHttpStatus(char *recvBuffer)
{
	int i = 0;
	int k = 0;
	char status[8];
	int ret = 0;
	
	size_t msgSize = 2048;
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
	sprintf_s(logMsg, msgSize, "returning %d as http response\n", ret);
	AquaLogging::Log(info, logMsg);
	return ret;
}

void printRoutingTable()
{
	PMIB_IPFORWARDTABLE pIpForwardTable = NULL;
	DWORD dwSize = 0;
	BOOL bOrder = FALSE;
	DWORD dwStatus;
	
	size_t msgSize = 2048;
	unsigned int i;

	sprintf_s(logMsg, msgSize, "===========Going to print routing table.============");
	AquaLogging::Log(verbose, logMsg);
	// Find out how big our buffer needs to be.
	dwStatus = GetIpForwardTable(pIpForwardTable, &dwSize, bOrder);
	if (dwStatus == ERROR_INSUFFICIENT_BUFFER) 
	{
		// Allocate the memory for the table
		pIpForwardTable = (PMIB_IPFORWARDTABLE)malloc(dwSize);
		if (pIpForwardTable == NULL)
		{
			sprintf_s(logMsg, msgSize, "Malloc failed. Out of memory, pipForwardTable");
			AquaLogging::Log(warn, logMsg);
			return;
		}
		// Now get the table.
		dwStatus = GetIpForwardTable(pIpForwardTable, &dwSize, bOrder);
	}

	if (dwStatus != ERROR_SUCCESS) 
	{
		sprintf_s(logMsg, msgSize, "getIpForwardTable failed");
		AquaLogging::Log(warn, logMsg);
		if (pIpForwardTable)
			free(pIpForwardTable);
		return;
	}

	for (i = 0; i < pIpForwardTable->dwNumEntries; i++) {

		char mask[INET_ADDRSTRLEN];
		inet_ntop(AF_INET, &pIpForwardTable->table[i].dwForwardMask, mask, INET_ADDRSTRLEN);
		char dest[INET_ADDRSTRLEN];
		inet_ntop(AF_INET, &pIpForwardTable->table[i].dwForwardDest, dest, INET_ADDRSTRLEN);
		char nexthop[INET_ADDRSTRLEN];
		inet_ntop(AF_INET, &pIpForwardTable->table[i].dwForwardNextHop, nexthop, INET_ADDRSTRLEN);
		sprintf_s(logMsg, msgSize, "=> mask=%s fwAge=%d dest=%s if=%d nexthop=%s dwFP=%d dwFP=%d m1=%d m2=%d m3=%d m4=%d m5=%d, nhAS=%d, FT=%d, dwFT=%d, FP=%d\n", 
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
			pIpForwardTable->table[i].ForwardProto
				);

		AquaLogging::Log(verbose, logMsg);
	}

	// Free resources
	if (pIpForwardTable) free(pIpForwardTable);	

	sprintf_s(logMsg, msgSize, "===========Finished printing routing table.============");
	AquaLogging::Log(verbose, logMsg);

}

int GetPrimaryCAIface(char *ca , int *iface)
{
	PIP_ADAPTER_INFO pAdapterInfo;
	PIP_ADAPTER_INFO pAdapter = NULL;
	DWORD dwRetVal = 0;
	UINT i;
	
	size_t msgSize = 2048;
	int retval = -1;

	sprintf_s(logMsg, msgSize, "Going to get primary CA via GetAdaptersInfo");
	AquaLogging::Log(info, logMsg);


	ULONG ulOutBufLen = sizeof(IP_ADAPTER_INFO);
	pAdapterInfo = (IP_ADAPTER_INFO *)malloc(sizeof(IP_ADAPTER_INFO));
	if (pAdapterInfo == NULL) 
	{
		sprintf_s(logMsg, msgSize, "Error allocating memory needed to call GetAdaptersinfo\n");
		AquaLogging::Log(warn, logMsg);
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
			sprintf_s(logMsg, msgSize, "Error allocating memory needed to call GetAdaptersinfo\n");
			AquaLogging::Log(warn, logMsg);
			return retval;
		}
	}

	if ((dwRetVal = GetAdaptersInfo(pAdapterInfo, &ulOutBufLen)) == NO_ERROR) 
	{
		pAdapter = pAdapterInfo;
		while (pAdapter) 
		{
			sprintf_s(logMsg, msgSize, "\tComboIndex: \t%d\n", pAdapter->ComboIndex);
			AquaLogging::Log(verbose, logMsg);
			sprintf_s(logMsg, msgSize, "\tAdapter Name: \t%s\n", pAdapter->AdapterName);
			AquaLogging::Log(verbose, logMsg);
			sprintf_s(logMsg, msgSize, "\tAdapter Desc: \t%s\n", pAdapter->Description);
			AquaLogging::Log(verbose, logMsg);
			sprintf_s(logMsg, msgSize, "\tAdapter Addr: \t");
			AquaLogging::Log(verbose, logMsg);
			for (i = 0; i < pAdapter->AddressLength; i++) 
			{
				if (i == (pAdapter->AddressLength - 1))
				{
					sprintf_s(logMsg, msgSize, "%.2X\n", (int)pAdapter->Address[i]);
					AquaLogging::Log(verbose, logMsg);
				}
				else
				{
					sprintf_s(logMsg, msgSize, "%.2X-", (int)pAdapter->Address[i]);
					AquaLogging::Log(verbose, logMsg);
				}
			}
			sprintf_s(logMsg, msgSize, "\tIndex: \t%d\n", pAdapter->Index);
			AquaLogging::Log(verbose, logMsg);
			sprintf_s(logMsg, msgSize, "\tType: \t");
			AquaLogging::Log(verbose, logMsg);
			switch (pAdapter->Type) {
			case MIB_IF_TYPE_OTHER:
				sprintf_s(logMsg, msgSize, "Other\n");
				AquaLogging::Log(verbose, logMsg);
				break;
			case MIB_IF_TYPE_ETHERNET:
				sprintf_s(logMsg, msgSize, "Ethernet\n");
				AquaLogging::Log(verbose, logMsg);
				break;
			case MIB_IF_TYPE_TOKENRING:
				sprintf_s(logMsg, msgSize, "Token Ring\n");
				AquaLogging::Log(verbose, logMsg);
				break;
			case MIB_IF_TYPE_FDDI:
				sprintf_s(logMsg, msgSize, "FDDI\n");
				AquaLogging::Log(verbose, logMsg);
				break;
			case MIB_IF_TYPE_PPP:
				sprintf_s(logMsg, msgSize, "PPP\n");
				AquaLogging::Log(verbose, logMsg);
				break;
			case MIB_IF_TYPE_LOOPBACK:
				sprintf_s(logMsg, msgSize, "Lookback\n");
				AquaLogging::Log(verbose, logMsg);
				break;
			case MIB_IF_TYPE_SLIP:
				sprintf_s(logMsg, msgSize, "Slip\n");
				AquaLogging::Log(verbose, logMsg);
				break;
			default:
				sprintf_s(logMsg, msgSize, "Unknown type %ld\n", pAdapter->Type);
				AquaLogging::Log(verbose, logMsg);
				break;
			}

			sprintf_s(logMsg, msgSize, "\tIP Address: \t%s\n", pAdapter->IpAddressList.IpAddress.String);
			AquaLogging::Log(verbose, logMsg);
			sprintf_s(logMsg, msgSize, "\tIP Mask: \t%s\n", pAdapter->IpAddressList.IpMask.String);
			AquaLogging::Log(verbose, logMsg);

			sprintf_s(logMsg, msgSize, "\tGateway: \t%s\n", pAdapter->GatewayList.IpAddress.String);
			AquaLogging::Log(verbose, logMsg);
			sprintf_s(logMsg, msgSize, "\t***\n");
			AquaLogging::Log(verbose, logMsg);

			if (pAdapter->DhcpEnabled) 
			{
				sprintf_s(logMsg, msgSize, "\tDHCP Enabled: Yes\n");
				AquaLogging::Log(verbose, logMsg);
				sprintf_s(logMsg, msgSize, "\t  DHCP Server: \t%s\n", pAdapter->DhcpServer.IpAddress.String);
				AquaLogging::Log(verbose, logMsg);
				if(strncmp(pAdapter->DhcpServer.IpAddress.String, "0.0.0.0", 7) != 0)
				{
					(*iface) = pAdapter->ComboIndex;
					strncpy_s(ca, 16, pAdapter->IpAddressList.IpAddress.String, 16);
					sprintf_s(logMsg, msgSize, "Found primary CA: %s iface: %d\n", ca, *iface);
					AquaLogging::Log(info, logMsg);
					retval = 0;
					break;
				}
			}
			else
			{
				sprintf_s(logMsg, msgSize, "\tDHCP Enabled: No\n");
				AquaLogging::Log(verbose, logMsg);
			}

			pAdapter = pAdapter->Next;
		}
	}
	else 
	{
		sprintf_s(logMsg, msgSize, "GetAdaptersInfo failed with error: %d\n", dwRetVal);
		AquaLogging::Log(err, logMsg);

	}
	if (pAdapterInfo) free(pAdapterInfo);

	sprintf_s(logMsg, msgSize, "Finished retrieving primary CA via GetAdaptersInfo. Returning : %d", retval);
	AquaLogging::Log(info, logMsg);
	return retval;
}

int createRouteToWireServer()
{
	PMIB_IPFORWARDROW pRow;
	char primaryCa[16];
	int iFace;
	
	size_t msgSize = 2048;

	sprintf_s(logMsg, msgSize, "Going to get primary CA");
	AquaLogging::Log(info, logMsg);

	if(GetPrimaryCAIface(primaryCa, &iFace) !=0)
	{
		sprintf_s(logMsg, msgSize, "Unable to get primary CA via get adapters");
		AquaLogging::Log(err, logMsg);
		return -1;
	}

	sprintf_s(logMsg, msgSize, "Successfully retrieved primary CA as %s iFace: %d", primaryCa, iFace);
	AquaLogging::Log(info, logMsg);


	pRow = (PMIB_IPFORWARDROW)malloc(sizeof(MIB_IPFORWARDROW));
	if (!pRow)
	{
		sprintf_s(logMsg, msgSize, "Unable to allocate memory for pRow");
		AquaLogging::Log(err, logMsg);
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
		sprintf_s(logMsg, msgSize, "Unable to create route for WireServer because it already exists. Retval: %lu ", retval);
		AquaLogging::Log(warn, logMsg);	
	}
	else
	if (retval != 0)
	{
		sprintf_s(logMsg, msgSize, "Unable to create route for WireServer. Error: %lu ", retval);
		AquaLogging::Log(warn, logMsg);
		if (pRow) free(pRow);
		return -1;
	}

	sprintf_s(logMsg, msgSize, "Successfully created route for WireSererver.");
	AquaLogging::Log(info, logMsg);
	if (pRow) free(pRow);
	return 0;
}
int getSubnetGwIpAndSetPrimary(char **subnetIp, char **gwIp, NclRepositoryManager *nclRepositoryManager)
{
	char *recvBuffer = nullptr;
	char *responseLocation = nullptr;
	
	size_t msgSize = 2048;
	int retval=0;

	sprintf_s(logMsg, msgSize, "Trying to connect to nm agent.\n");
	AquaLogging::Log(info, logMsg);
	printRoutingTable();
	createRouteToWireServer();
	printRoutingTable();
	SOCKET sock = connectToHost("169.254.169.254", 80);

	if (sock == INVALID_SOCKET)
	{
		sprintf_s(logMsg, msgSize,  "Connect to host failed.\n");
		AquaLogging::Log(err, logMsg);
		wsaCleanUp();
		return 1;
	}

	recvBuffer = httpSendResponse(sock, HostSuffix);
	if (!recvBuffer) {
		sprintf_s(logMsg, msgSize, "Error while retrieving xml file from NMAgent");
		AquaLogging::Log(err, logMsg);
		retval=1;
		goto cleanup_subgw;
	}
	//sprintf_s(logMsg, msgSize, "xml recv buf:%s\n", recvBuffer);

	int ret = retrieveHttpStatus(recvBuffer);
	sprintf_s(logMsg, msgSize, "status:%d\n", ret);
	AquaLogging::Log(info, logMsg);
	if (ret != 200)
	{
		sprintf_s(logMsg, msgSize, "Error while retrieving xml file from NMAgent. Invalid Status code:%s\n", recvBuffer);
		AquaLogging::Log(err, logMsg);
		retval=1;
		goto cleanup_subgw;
	}

	responseLocation = strstr(recvBuffer, HttpResponseHeaderSeparator);
	responseLocation += strlen(HttpResponseHeaderSeparator);

	sprintf_s(logMsg, msgSize, "HttpResponse from NMAgent: %s", responseLocation);
	AquaLogging::Log(info, logMsg);

	if (responseLocation != nullptr)
	{
		ParseSubnetGatewayPrimaryIp(responseLocation, subnetIp, gwIp, nclRepositoryManager);
		if(*subnetIp==nullptr || *gwIp==nullptr)
			retval=1;
	}
	else
	{
		sprintf_s(logMsg, msgSize,  "Unable to find portion of http response header that contains valid nm agent response response %s", recvBuffer);
		AquaLogging::Log(err, logMsg);
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
	wsaCleanUp();

	return retval;
}

int GetIpsFromNmAgent(unsigned int *ips)
{
	int ipCount = 0;
	char *recvBuffer = nullptr;
	int bytesRead;
	char *responseLocation = nullptr;
	int byteCount = 0;
	
	size_t msgSize = 2048;
	
	sprintf_s(logMsg, msgSize, "Trying to connect to nm agent.\n");
	AquaLogging::Log(info, logMsg);	
	printRoutingTable();
	createRouteToWireServer();
	printRoutingTable();
	SOCKET sock = connectToHost("169.254.169.254", 80);

	if (sock == INVALID_SOCKET)
	{
		sprintf_s(logMsg, msgSize,  "Connect to host failed.\n");
		AquaLogging::Log(err, logMsg);
		wsaCleanUp();
		return -1;
	}
	sprintf_s(logMsg, msgSize, "Now going to get ips from nm agent... fd: %llu\n", sock);
	AquaLogging::Log(info, logMsg);
	int bytesWritten = send(sock, HostSuffix, (int)strlen(HostSuffix), 0);
	if (bytesWritten == SOCKET_ERROR)
	{
		sprintf_s(logMsg, msgSize,  "Writing to socket failed errno=%d\n", errno);
		AquaLogging::Log(err, logMsg);
	}
	else
	{
		sprintf_s(logMsg, msgSize, "Sent %d bytes on the connection.\nNow reading into buffer..", bytesWritten);
		AquaLogging::Log(info, logMsg);
		do
		{
			char c;
			bytesRead = recv(sock, &c, 1, 0);
			if (bytesRead == 1)
			{
				byteCount++;
				if (!InsertIntoBuffer(&recvBuffer, byteCount, c))
				{
					sprintf_s(logMsg, msgSize,  "Failed to insert a char into buffer");
					AquaLogging::Log(err, logMsg);
					ipCount = -1;
					goto cleanup;
				}
			}
		} while (bytesRead > 0);

		if (bytesRead == -1)
		{
			sprintf_s(logMsg, msgSize,  "Reading failed. errno=%d \n", errno);
			AquaLogging::Log(err, logMsg);
		}
		else if (bytesRead == 0)
		{
			if (!InsertIntoBuffer(&recvBuffer, byteCount + 1, '\0'))
			{
				sprintf_s(logMsg, msgSize,  "Failed to insert EOL into buffer");
				AquaLogging::Log(err, logMsg);
				ipCount = -1;
				goto cleanup;
			}

			sprintf_s(logMsg, msgSize, "\nSuccessfully finished reading. Printing response below..");
			AquaLogging::Log(info, logMsg);
			responseLocation = strstr(recvBuffer, HttpResponseHeaderSeparator);
			responseLocation += strlen(HttpResponseHeaderSeparator);
			sprintf_s(logMsg, msgSize, "Response: %s", responseLocation);
			AquaLogging::Log(info, logMsg);
		}
		if (responseLocation != nullptr)
		{
			if (ips == NULL)
			{
				ipCount = ParseSecondaryIpAddresses(responseLocation, 1, ips);
				sprintf_s(logMsg, msgSize, "Received only ip count as %d\n", ipCount);
				AquaLogging::Log(info, logMsg);
			}
			else
			{
				ipCount = ParseSecondaryIpAddresses(responseLocation, 0, ips);
				sprintf_s(logMsg, msgSize, "Received ips with ip count as %d\n", ipCount);
				AquaLogging::Log(info, logMsg);
			}
		}
		else
		{
			sprintf_s(logMsg, msgSize,  "Unable to find portion of http response header that contains valid nm agent response response %s", recvBuffer);
			AquaLogging::Log(err, logMsg);
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
		sprintf_s(logMsg, msgSize, "+++++++++ERROR While closing Socket!+++++++++++++++++\n");
		AquaLogging::Log(err, logMsg);
	}

	wsaCleanUp();
	return ipCount;
}

DWORD GetHostIp(unsigned int *ip, NclRepositoryManager *nclRepositoryManager)
{
	
	size_t msgSize = 2048;
	unsigned long retval = UNEXPECTED_ERROR;
	
	if(networkTypeProvided == UnderLay)
	{
		*ip = nclRepositoryManager->GetRepositoryPrimaryIp();
		LOGINFO("Ip:%d\n",*ip);
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
			sprintf_s(logMsg, msgSize, "Reserving Available for host local returned non-zero %d", retval);
			AquaLogging::Log(info, logMsg);
		}
	}
	return retval;
}

int networkExists(SOCKET sock, char *networkName)
{
	char *payload = nullptr;
	char *recvBuf = nullptr;
	size_t payloadLen;
	
	size_t msgSize = 2048;
	char *httpreq = "HTTP/1.1\r\nHost:127.0.0.1\r\nContent-Type:application/json\r\nConnection:close\r\n\r\n";

	sprintf_s(logMsg, msgSize, "Checking if network already exists.. %s\n", networkName);
	AquaLogging::Log(info, logMsg);

	payloadLen = strlen(DockerNwGet) + strlen(networkName) + strlen(httpreq);

	payload = (char *)malloc(payloadLen + 2);
	sprintf_s(payload, payloadLen + 2, "%s%s %s", DockerNwGet, networkName, httpreq);
	payload[payloadLen + 1] = '\0';
	recvBuf = httpSendResponse(sock, payload);
	if (!recvBuf)
	{
		sprintf_s(logMsg, msgSize, "Unexpected error while check for existing network:%s\n", networkName);
		AquaLogging::Log(err, logMsg);
		free(payload);
		return 0;
	}

	sprintf_s(logMsg, msgSize, "receive buf:%s\n", recvBuf);
	AquaLogging::Log(info, logMsg);
	int ret = retrieveHttpStatus(recvBuf);
	if (ret == 200)
	{
		sprintf_s(logMsg, msgSize, "Network Initialized already\n");
		AquaLogging::Log(warn, logMsg);
		if (payload) free(payload);
		free(recvBuf);
		return 0;
	}

	if (payload) free(payload);
	free(recvBuf);
	return 1;
}

DWORD SetupUnderlayInAzure(ClusterConfig *clusterConfig, NclRepositoryManager *nclRepositoryManager)
{
	size_t conLength = 0, ipamLength = 0;
	char *nwCreateReq = nullptr;
	char *recvBuf = nullptr;
	size_t totalLength = 0;
	char *subnetIp = nullptr;
	char *gwIp = nullptr;
	char ipamConfig[2048];
	unsigned long retval = 0;
	
	size_t msgSize = 2048;

	sprintf_s(logMsg, msgSize, "wsa initialization finished\n");
	AquaLogging::Log(info, logMsg);
	SOCKET sock = connectToHost("localhost", 2375);
	if (sock == INVALID_SOCKET)
	{
		sprintf_s(logMsg, msgSize,  "Connect to host failed.\n");
		AquaLogging::Log(err, logMsg);
		wsaCleanUp();
		return UNREACHABLE_HOST;
	}

	if(getSubnetGwIpAndSetPrimary(&subnetIp, &gwIp, nclRepositoryManager)!=0)
	{
		sprintf_s(logMsg, msgSize,  "Retrieving subnet and gateway ip failed.\n");
		AquaLogging::Log(err, logMsg);
		goto cleanup2;
	}

	if (networkExists(sock, clusterConfig->networkName))
	{
		sprintf_s(logMsg, msgSize, "Network %s does not exist, going to create it\n", clusterConfig->networkName);
		AquaLogging::Log(info, logMsg);

		shutdown(sock, SD_BOTH);
		closesocket(sock);
		wsaCleanUp();
		sprintf_s(logMsg, msgSize, "wsa cleanup finished\n");
		AquaLogging::Log(info, logMsg);

		sprintf_s(logMsg, msgSize, "retrieved subnet and gateway %s %s\n", subnetIp, gwIp);
		AquaLogging::Log(info, logMsg);

		sprintf_s(ipamConfig, sizeof(ipamConfig), "\"IPAM\":{\"Config\":[{\"Subnet\":\"%s\",\"Gateway\":\"%s\"}]}", subnetIp, gwIp);
		AquaLogging::Log(info, logMsg);

		sprintf_s(logMsg, msgSize, "ipamconfig:%s\n", ipamConfig);
		AquaLogging::Log(info, logMsg);
		
		sock = connectToHost("localhost", 2375);

		ipamLength = strlen(ipamConfig);
		conLength = strlen(clusterConfig->networkName) + 9 + 19 + 4 + ipamLength;
		totalLength = strlen(DockerNwCreatePost) + conLength + ipamLength + 64;
		nwCreateReq = (char *)malloc(totalLength + 1);

		sprintf_s(nwCreateReq, totalLength + 1, 
			"%sContent-Length:%d\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s}", 
				DockerNwCreatePost, (int)conLength, clusterConfig->networkName, "l2tunnel", ipamConfig);	
		AquaLogging::Log(info, logMsg);

		sprintf_s(logMsg, msgSize, "Now going to get create network driver... cmd:%s:%d:%llu:%d\n", 
			nwCreateReq, (int)totalLength, strlen(nwCreateReq), (int)conLength);
		AquaLogging::Log(info, logMsg);
		recvBuf = httpSendResponse(sock, nwCreateReq);
		if (!recvBuf)
		{
			sprintf_s(logMsg, msgSize, "Error while creating docker network");
			AquaLogging::Log(err, logMsg);
			retval = UNEXPECTED_ERROR;
			goto cleanup2;
		}

		int ret = retrieveHttpStatus(recvBuf);
		sprintf_s(logMsg, msgSize, "status:%d\n", ret);
		if (ret == 201)
		{
			sprintf_s(logMsg, msgSize, "docker network create succeeded\n");
			AquaLogging::Log(info, logMsg);
		}
		else
		{
			sprintf_s(logMsg, msgSize, "docker network create failed:%s\n", recvBuf);
			AquaLogging::Log(err, logMsg);
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
	wsaCleanUp();
	sprintf_s(logMsg, msgSize, "setup underlay returning %d", retval);
	AquaLogging::Log(info, logMsg);
	return retval;
}

DWORD installHNSOverlayPlugin()
{
	int retval;
	
	system("start powershell.exe Set-ExecutionPolicy RemoteSigned \n");
	retval = system("start powershell.exe InstallPlugin.ps1");
	
	return retval;
}

unsigned long CreateDockerNetwork(char *networkName, char ipamConfig[2048])
{
	char *recvBuf = NULL;
	unsigned long retval = 0;
	
	size_t msgSize = 2048;

	SOCKET sock = connectToHost("localhost", 2375);
	if(sock == INVALID_SOCKET)
	{
		return UNREACHABLE_DOCKER_DAEMON;
	}

	size_t ipamLength = strlen(ipamConfig);
	size_t conLength = strlen(networkName) + 9 + 19 + 4 + ipamLength;
	size_t totalLength = strlen(DockerNwCreatePost) + conLength + ipamLength + 64;
	char *nwCreateReq = (char *)malloc(totalLength + 1);

	sprintf_s(nwCreateReq, totalLength + 1, 
	          "%sContent-Length:%d\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s}", 
	          DockerNwCreatePost, (int)conLength, networkName, "l2tunnel", ipamConfig);
	AquaLogging::Log(info, logMsg);

	sprintf_s(logMsg, msgSize, "Now going to get create network driver... cmd:%s:%d:%llu:%d\n",
		nwCreateReq, (int)totalLength, strlen(nwCreateReq), (int)conLength);
	AquaLogging::Log(info, logMsg);

	recvBuf = httpSendResponse(sock, nwCreateReq);
	if (!recvBuf)
	{
		sprintf_s(logMsg, msgSize, "Error while creating docker network");
		AquaLogging::Log(err, logMsg);
		retval = UNEXPECTED_ERROR;
	}
	else
	{
		int ret = retrieveHttpStatus(recvBuf);
		sprintf_s(logMsg, msgSize, "status:%d\n", ret);
		AquaLogging::Log(info, logMsg);
		if (ret == 201)
		{
		AquaLogging::Log(err, logMsg);
			sprintf_s(logMsg, msgSize, "docker network create succeeded\n");
			AquaLogging::Log(info, logMsg);
		}
		else
		{
			sprintf_s(logMsg, msgSize, "docker network create failed:%s\n", recvBuf);
			AquaLogging::Log(err, logMsg);
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

	sprintf_s(logMsg, msgSize, "create docker network returning %d\n", retval);
	AquaLogging::Log(info, logMsg);
	return retval;
}

DWORD SetupOverlay(ClusterConfig *clusterConfig)
{
	char *recvBuf = nullptr;
	char *gwIp = nullptr;
	char ipamConfig[2048];
	unsigned long retval = 0;
	
	size_t msgSize = 2048;

	if (clusterConfig->networkName == NULL)
	{
		return UNSPECIFIED_NETWORK_NAME;
	}

	sprintf_s(logMsg, msgSize, "wsa initialization finished to setup overlay\n");
	AquaLogging::Log(info, logMsg);	

	SOCKET sock = connectToHost("localhost", 2375);
	if (sock == INVALID_SOCKET)
	{
		sprintf_s(logMsg, msgSize,  "Connect to host failed (overlay).\n");
		AquaLogging::Log(err, logMsg);
		wsaCleanUp();
		return UNREACHABLE_DOCKER_DAEMON;
	}

	if (networkExists(sock, clusterConfig->networkName))
	{
		sprintf_s(logMsg, msgSize, "Overlay Network %s does not exist, going to create it\n", clusterConfig->networkName);
		AquaLogging::Log(info, logMsg);

		shutdown(sock, SD_BOTH);
		closesocket(sock);
		wsaCleanUp();
		sprintf_s(logMsg, msgSize, "wsa cleanup finished\n");
		AquaLogging::Log(info, logMsg);

		if(clusterConfig->overlaySubnet == NULL)
		{
			return OVERLAY_SUBNET_NOT_SPECIFIED;
		}

		if(clusterConfig->overlaySubnet->prefixLength > 30)
		{
			return SUBNET_SIZE_TOO_SMALL;
		}

		sprintf_s(logMsg, msgSize, "Going to get gateway ip from %d\n", clusterConfig->overlaySubnet->ipAddress);
		AquaLogging::Log(info, logMsg);
		char gatewayIp[INET_ADDRSTRLEN];
		if(inet_ntop(AF_INET, &clusterConfig->overlaySubnet->ipAddress, gatewayIp, INET_ADDRSTRLEN)==NULL)
		{
			return MALFORMED_SUBNET_IP;
		}

		if(NetworkCommon::RetrieveGatewayFromSubnet(gatewayIp, &gwIp) == false)
		{
			return MALFORMED_SUBNET_IP;
		}

		sprintf_s(logMsg, msgSize, "retrieved subnet and gateway %s/%d %s\n", gatewayIp, clusterConfig->overlaySubnet->prefixLength, gwIp);
		AquaLogging::Log(info, logMsg);

		sprintf_s(ipamConfig, sizeof(ipamConfig), 
			"\"IPAM\":{\"Config\":[{\"Subnet\":\"%s/%d\",\"Gateway\":\"%s\"}]}", 
			gatewayIp, clusterConfig->overlaySubnet->prefixLength, gwIp);
		AquaLogging::Log(info, logMsg);

		sprintf_s(logMsg, msgSize, "ipamconfig:%s\n", ipamConfig);
		AquaLogging::Log(info, logMsg);	

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

	wsaCleanUp();
	sprintf_s(logMsg, msgSize, "SetupOverlay returning %d\n", retval);
	AquaLogging::Log(info, logMsg);
	return retval;
}

DWORD AddRemoteMapping(const char *endpointId, const char *hostAddress, const char *containerAddress, const char *macAddress)
{
	 
	size_t msgSize = 2048;
	sprintf_s(logMsg, msgSize, "Call back to add endpoint received... success\n");
	AquaLogging::Log(info, logMsg);
	wchar_t endPointIdW[150]; //GUID cannot exceed 38
	size_t converted;
	mbstowcs_s(&converted, endPointIdW, endpointId, 150);
	
	wchar_t hostAddressW[INET_ADDRSTRLEN];
	mbstowcs_s(&converted, hostAddressW, hostAddress, INET_ADDRSTRLEN);

	wchar_t containerAddressW[INET_ADDRSTRLEN]; //GUID cannot exceed 38
	mbstowcs_s(&converted, containerAddressW, containerAddress, INET_ADDRSTRLEN);

	wchar_t macAddressW[50]; //MAC cannot exceed 24
	mbstowcs_s(&converted, macAddressW, macAddress, 50);

	sprintf_s(logMsg, msgSize, "%s %s %s %s\n", endpointId, macAddress, containerAddress, macAddress);
	AquaLogging::Log(info, logMsg);
	return AddEndpoint(endPointIdW, macAddressW, containerAddressW, hostAddressW, L"true");
}

DWORD DeleteRemoteMapping (const char *endpointId)
{
	wchar_t endpointIdW[150]; //GUID cannot exceed 38
	size_t converted;
	 
	size_t msgSize = 2048;
	sprintf_s(logMsg, msgSize, "Call back to delete endpoint received...\n");
	AquaLogging::Log(info, logMsg);
	mbstowcs_s(&converted, endpointIdW, endpointId, 150);
	unsigned long retval =  DeleteEndpoint(endpointIdW);
	sprintf_s(logMsg, msgSize, "Call back to delete endpoint finished with %u...\n", retval);
	AquaLogging::Log(info, logMsg);
	return retval;
}


DWORD GetLocalMappings(EndpointMapping **endpointMappings, USHORT *mappingsCount)
{
	AquaLogging::Log(info, "Enter GetLocalMappings");
	GetLocalEndpoints(endpointMappings, mappingsCount);
	AquaLogging::Log(info, "Exit GetLocalMappings");
	return 0;
}

DWORD FreeEndpointMappings(EndpointMapping *endpointMappings, USHORT mappingsCount)
{
	AquaLogging::Log(info, "Enter FreeEndpointMappings");
	FreeMemoryForLocalEndpoints(endpointMappings, mappingsCount);
	AquaLogging::Log(info, "Enter FreeEndpointMappings");
	return 0;
}

void ProvideConfigToNIX(ClusterConfig *clusterConfig)
{
	 
	size_t msgSize = 2048;

	sprintf_s(logMsg, msgSize, "Informing nix about the other nodes in the network");
	AquaLogging::Log(info, logMsg);
	unsigned int localIp;
	localIp = clusterConfig->localNodeIp;
	char ipInString[INET_ADDRSTRLEN];
	inet_ntop(AF_INET, &localIp, ipInString, INET_ADDRSTRLEN);
	char**remoteHostAddresses = new char *[clusterConfig->nodeCount - 1]; //-1 for local address
	USHORT cntr = 0;
	for (int i = 0; i < clusterConfig->nodeCount; i++)
	{
		
		NodeConfig cnc = clusterConfig->nodeConfigs[i];
		char remoteIpInString[INET_ADDRSTRLEN];
		inet_ntop(AF_INET, &cnc.nodeIp, remoteIpInString, INET_ADDRSTRLEN);

		sprintf_s(logMsg, msgSize, "--------from cluster config remote ip = %s %s %u---------\n", remoteIpInString, cnc.nodeId, cnc.subnet->ipAddress);
		AquaLogging::Log(info, logMsg);
		if (cnc.nodeIp != clusterConfig->localNodeIp)
		{
			char remoteIpInString2[INET_ADDRSTRLEN];
			inet_ntop(AF_INET, &cnc.nodeIp, remoteIpInString2, INET_ADDRSTRLEN);
			remoteHostAddresses[cntr] = (char *) malloc(sizeof(char)* INET_ADDRSTRLEN);
			strcpy_s(remoteHostAddresses[cntr], INET_ADDRSTRLEN, remoteIpInString2);
			sprintf_s(logMsg, msgSize, "saved remote ip = %s\n", remoteHostAddresses[cntr]);
			AquaLogging::Log(info, logMsg);
			cntr++;
		}
	}

	sprintf_s(logMsg, msgSize, "Local Host provided to NIX: %s", ipInString);
	AquaLogging::Log(info, logMsg);
	for(int j=0; j<cntr; j++)
	{
		sprintf_s(logMsg, msgSize, "Remote Host %d provided to NIX: %s", j, remoteHostAddresses[j]);
		AquaLogging::Log(info, logMsg);
	}

	InitializeAquaNix(ipInString, remoteHostAddresses, cntr, "127.0.0.1", AddRemoteMapping, DeleteRemoteMapping, 
		GetLocalMappings, FreeEndpointMappings, 10987, 10987);
	RequestRemoteMappings();
	for(int k = 0; k<cntr; k++)
	{
		free(remoteHostAddresses[k]);
	}
	delete []remoteHostAddresses;
}

} } } } }
#endif