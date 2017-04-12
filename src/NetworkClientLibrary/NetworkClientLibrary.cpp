/****************************** Module Header ******************************\
* Module Name:  NetworkClientLibrary.c
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* This is the SDN Library that can be used to setup 
* networking for containers in Azure and OnPrem
*
\***************************************************************************/

#define _WINSOCK_DEPRECATED_NO_WARNINGS
#include "Common/aquanetworkclient.h"
#include <stdlib.h>
#include "Common/nclcore.h"
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>
#include "../aquanix/Common/aquanixext.h"
#include <Mstcpip.h>
#include "../Logging/Logging.h"
#include "Common/aqualoglevels.h"
#ifdef _WIN32
#include <stdio.h>
#include <iostream>
#include <mutex>
#endif
#ifdef __linux__
#include <curl/curl.h>
#include <string.h>
#define UNREFERENCED_PARAMETER(param) do { (void)(param); } while (0)
#endif

extern DWORD setupUnderlayInAzure(ClusterConfig *clusterConfig);
extern DWORD setupOverlay(ClusterConfig *clusterConfig);
extern int getIpsFromNmAgent(unsigned int *ips);
extern int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
extern void UpdateRepositoryIpList(int ipCount, unsigned int *ips);
extern unsigned int ReserveRepositoryIpAddress(char *reservationId);
extern unsigned long ReleaseRepositoryIpAddress(const char *reservationId);
extern int GetAvailableIpsfromRepository(unsigned int *ips);
extern unsigned int GetReservedIpFromRepository(const char *reservationId);
extern void ParseSubnetAndGatewayIp(char *xmlResponse, char **sunbetIp, char **gatewayIp);
extern unsigned long PersistSubnetInRepository(Ipv4Subnet *subnetToPersist);
extern unsigned int PersistedIpCountInRepository();
extern DWORD GetHostIp(unsigned int *ip);
NetworkType networkTypeProvided = UnderLay;
std::mutex hostLocalMutex;
bool hostLocalIpProvisionedInHNS = false;

// Returns the number of ip addresses that were returned
// If nodeNetworkResources is null, only the number of ip addresses is returned
// Otherwise, all the ip addresses (reserved + nonreserved) are populated
int GetAllIpAddresses(NodeNetworkResources *nodeNetworkResources)
{
	int ipCount = getIpsFromNmAgent(nodeNetworkResources->ips);
	
	if (nodeNetworkResources->ips)
	{
		UpdateRepositoryIpList(ipCount, nodeNetworkResources->ips);
	}
	return ipCount;
}

// Returns the number of ip addresses that were returned
// If nodeNetworkResources->ips is null, the available number of ip addresses is returned
// Otherwise, the nonreserved ip addresses are populated
DllExport DWORD GetAvailableIpAddresses(NodeNetworkResources *nodeNetworkResources, int *count)
{
	int ipCount;
	unsigned int *ips;
	
	if(networkTypeProvided == OverLay)
	{
		// overlay ips are persisted during network init
		if(nodeNetworkResources != NULL)
		{
			GetAvailableIpsfromRepository(nodeNetworkResources->ips);
		}
		*count = PersistedIpCountInRepository();
		return 0;
	}

	ipCount = getIpsFromNmAgent(NULL);	
	if (ipCount <= 0)
	{
		return UNEXPECTED_ERROR;
	}

	ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));
	ipCount = getIpsFromNmAgent(ips);
	printf("Retrieved %d ips fron nm agent along with details\n", ipCount);
	
	for(int i = 0; i<ipCount; i++)
	{
		printf("%d\n", ips[i]);
	}
	if(ips)
	{
		UpdateRepositoryIpList(ipCount, ips);
		free(ips);
	}
	if (nodeNetworkResources) 
	{
		nodeNetworkResources->ipCount = ipCount;
		ipCount = GetAvailableIpsfromRepository(nodeNetworkResources->ips);
	}
	
	*count = ipCount;
	return 0;
}

// This method selects an ip address from the available pool 
// of ip addresses and reserves it against the reservationId
// The reserved ip address is returned
// If no ip addresses are available to be returned, 0 is returned
// This method is idempotent.
DllExport DWORD ReserveAvailableIpAddress(char *reservationId, unsigned int *ipAddress)
{
	unsigned int reservedIp = 0;
	int count = 0;
	GetAvailableIpAddresses(NULL, &count);
	if ( count > 0)
	{
		 reservedIp = ReserveRepositoryIpAddress(reservationId);
	}

	if(reservedIp == 0)
	{
		return UNEXPECTED_ERROR;
	}

	*ipAddress = reservedIp;
	return 0;	
}

// Returns a single ip address
DllExport  DWORD GetIpAddressForContainer(char *containerId, unsigned int ipAddress)
{
	UNREFERENCED_PARAMETER(containerId);
	UNREFERENCED_PARAMETER(ipAddress);
	return 0;
}

// Returns an ip address reserved correspoiding to the reservation id
// returns -1 if reservation id does not exist
DllExport DWORD GetReservedIpAddress(const char *reservationId, unsigned int *ipAddress)
{
	unsigned int reservedIp =  GetReservedIpFromRepository(reservationId);
	if(reservedIp == 0)
	{
		return UNEXPECTED_ERROR;
	}

	*ipAddress = reservedIp;
	return 0;
}

// Releases reservation of an ip address if exists
DllExport DWORD ReleaseIpReservation(const char *reservationId)
{
	if(!ReleaseRepositoryIpAddress(reservationId))
	{
		return RESERVATION_NOT_FOUND;
	}

	return 0;
}

bool IsLocalNodeIp(unsigned int ip)
{
	UNREFERENCED_PARAMETER(ip);

	unsigned int hostIp;
	if(GetHostIp(&hostIp) != 0)
	{
		return false;
	}
	if(hostIp == ip)
	{
		return true;
	}
	return false;
}


unsigned long saveOverlayAddressSpace(ClusterConfig *clusterConfig)
{
	if(clusterConfig->nodeConfigs == NULL)
	{
		return UNEXPECTED_ERROR;
	}
	for(int i= 0 ; i<clusterConfig->nodeCount ; i++)
	{
		NodeConfig nc = clusterConfig->nodeConfigs[i];
		printf("Comparing %d %d", nc.nodeIp, clusterConfig->localNodeIp);
		if(nc.nodeIp == clusterConfig->localNodeIp)
		{
			printf("Found my own node config nodeIp=0x%08X prefix=0x%08X prefix-len%d\n", nc.nodeIp, nc.subnet->ipAddress, nc.subnet->prefixLength);
			PersistSubnetInRepository(nc.subnet);
			printf("persisted ip space for my own node\n");
			break;
		}
	}
	return 0;
}

extern DllExport int AddEndpoint(std::wstring endpointId, std::wstring macAddress, std::wstring ca, std::wstring pa, std::wstring isRemoteEndpoint);
DWORD AddRemoteMapping(const char *endpointId, const char *hostAddress, const char *containerAddress, const char *macAddress)
{
	printf("Call back to add endpoint received... success\n");
	wchar_t endPointIdW[150]; //GUID cannot exceed 38
	size_t converted;
	mbstowcs_s(&converted, endPointIdW, endpointId, 150);
	
	wchar_t hostAddressW[INET_ADDRSTRLEN];
	mbstowcs_s(&converted, hostAddressW, hostAddress, INET_ADDRSTRLEN);

	wchar_t containerAddressW[INET_ADDRSTRLEN]; //GUID cannot exceed 38
	mbstowcs_s(&converted, containerAddressW, containerAddress, INET_ADDRSTRLEN);

	wchar_t macAddressW[50]; //MAC cannot exceed 24
	mbstowcs_s(&converted, macAddressW, macAddress, 50);

	printf("%s %s %s %s\n", endpointId, macAddress, containerAddress, macAddress);
	return AddEndpoint(endPointIdW, macAddressW, containerAddressW, hostAddressW, L"true");
}

extern DllExport int DeleteEndpoint(std::wstring endpointId);
DWORD DeleteRemoteMapping (const char *endpointId)
{
	wchar_t endpointIdW[150]; //GUID cannot exceed 38
	size_t converted;
	mbstowcs_s(&converted, endpointIdW, endpointId, 150);
	return DeleteEndpoint(endpointIdW);
}
void provideConfigToNIX(ClusterConfig *clusterConfig)
{

	printf("Informing nix about the other nodes in the network");
	unsigned int localIp;
	GetHostIp(&localIp);
	localIp = clusterConfig->localNodeIp;
	char ipInString[INET_ADDRSTRLEN];
	inet_ntop(AF_INET, &localIp, ipInString, INET_ADDRSTRLEN);
	char**remoteHostAddresses = new char *[clusterConfig->nodeCount - 1]; //-1 for local address
	int cntr = 0;
	for (int i = 0; i < clusterConfig->nodeCount; i++)
	{
		
		NodeConfig cnc = clusterConfig->nodeConfigs[i];
		char remoteIpInString[INET_ADDRSTRLEN];
		inet_ntop(AF_INET, &cnc.nodeIp, remoteIpInString, INET_ADDRSTRLEN);

		printf("--------from cluster config remote ip = %s %s %u---------\n", remoteIpInString, cnc.nodeId, cnc.subnet->ipAddress);
		if (cnc.nodeIp != clusterConfig->localNodeIp)
		{
			char remoteIpInString2[INET_ADDRSTRLEN];
			inet_ntop(AF_INET, &cnc.nodeIp, remoteIpInString2, INET_ADDRSTRLEN);
			remoteHostAddresses[cntr] = remoteIpInString2;
			printf("saved remote ip = %s\n", remoteIpInString2);
			cntr++;
		}
	}

	printf("giving remote ip as %s local=%s", remoteHostAddresses[0], ipInString);
	InitializeAquaNix(ipInString, remoteHostAddresses, AddRemoteMapping, DeleteRemoteMapping, 10050, 10050);
	delete []remoteHostAddresses;
}

DllExport DWORD InitializeNetwork(ClusterConfig *clusterConfig)
{
	unsigned long failed = 0;
	if (!clusterConfig)
	{
		return INVALID_PARAMETER;
	}

	AquaLogging::Log(info, "Going to initialize network ..\n");

	switch(clusterConfig->net)
	{
		case OverLay: 
			printf("Going to setup overlay network\n");
			networkTypeProvided = OverLay; // save the type
			failed = setupOverlay(clusterConfig);
			if(failed == 0)
			{
				printf("Going to persist overlay address space\n");
				saveOverlayAddressSpace(clusterConfig);
				provideConfigToNIX(clusterConfig);
				printf("Successfully persisted Overlay Address Space\n");
			}
			else
			{
				printf("setup Overlay returned non zero value %d. None of the ips were saved.", failed);
			}

			printf("Initialize network finished. retunring %d\n", failed);
			return failed;

		case UnderLay: 
			printf("Going to setup underlay network\n");
			networkTypeProvided = UnderLay;
			switch(clusterConfig->env)
			{
			case Azure: 
				return setupUnderlayInAzure(clusterConfig);

			case StandAlone: 		
				printf("Underlay in StandAlone is not a supportedconfig");

			default:
				printf("Unsupported environment %d\n", clusterConfig->env);
				return UNSUPPORTED_NETWORK;
			}
		default:
			printf("Unknown network type %d\n", clusterConfig->net);
			return UNSUPPORTED_NETWORK;
	}
}

extern DllExport int AddHostLocalEndpoint(std::wstring hostLocalIp);
DllExport DWORD GetHostLocalIp(unsigned int *hostLocalIp)
{
	unsigned long retval = 0;
	if(networkTypeProvided == OverLay)
	{
		retval = ReserveAvailableIpAddress("HostLocalIp", hostLocalIp);
		if (retval == 0)
		{
			if (!hostLocalIpProvisionedInHNS)
			{
				hostLocalMutex.lock();
				if (!hostLocalIpProvisionedInHNS)
				{
					char ipInString[INET_ADDRSTRLEN];
					wchar_t ipInStringw[INET_ADDRSTRLEN];
					inet_ntop(AF_INET, hostLocalIp, ipInString, INET_ADDRSTRLEN);
					size_t converted;
					mbstowcs_s(&converted, ipInStringw, ipInString, INET_ADDRSTRLEN);
					AddHostLocalEndpoint(ipInStringw);
					hostLocalIpProvisionedInHNS = true;
				}
			}
		}
		else
		{
			printf("Reserve Available for host local returned non-zero %d", retval);
		}
	}
	else
	{
		retval = GetHostIp(hostLocalIp);
	}
	return retval;
}

DllExport DWORD SetupNetworkLogging(void(*log)(int, char*))
{
	AquaLogging::InitLogging(log);
	return 0;
}

DllExport DWORD SetupNetworkEtwLogging(bool additionConsoleLoggingEnabled)
{
	AquaLogging::InitLogging(additionConsoleLoggingEnabled);
	return 0;
}