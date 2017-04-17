/****************************** Module Header ******************************\
* Module Name:  NetworkClientLibrary.c
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* This is the SDN Library that can be used to setup 
* networking for containers in Azure and OnPrem
*
\***************************************************************************/

#include <stdlib.h>
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>
#include <Mstcpip.h>
#include "Common/aquanetworkclient.h"
#include "Common/nclcore.h"
#include "../aquanix/Common/aquanixext.h"
#include "../Logging/Logging.h"

#ifdef _WIN32
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#include <stdio.h>
#include <iostream>
#include <mutex>
#endif

#ifdef __linux__
#include <curl/curl.h>
#include <string.h>
#define UNREFERENCED_PARAMETER(param) do { (void)(param); } while (0)
#endif
#define lInfo 1

extern DllExport unsigned long GetLocalEndpoints(EndpointMapping **endpointMappings, USHORT *mappingsCount);
extern DllExport int AddEndpoint(std::wstring endpointId, std::wstring macAddress, std::wstring ca, std::wstring pa, std::wstring isRemoteEndpoint);
extern DllExport int DeleteEndpoint(std::wstring endpointId);
extern DllExport int AddHostLocalEndpoint(std::wstring hostLocalIp);
extern DllExport unsigned long FreeMemoryForLocalEndpoints(EndpointMapping *endpointMappings, USHORT mappingsCount);

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {

extern DWORD setupUnderlayInAzure(ClusterConfig *clusterConfig);
extern DWORD setupOverlay(ClusterConfig *clusterConfig);
extern int getIpsFromNmAgent(unsigned int *ips);
extern int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
extern void UpdateRepositoryIpList(int ipCount, unsigned int *ips, NetworkType networkType);
extern unsigned int ReserveRepositoryIpAddress(char *reservationId);
extern unsigned long ReleaseRepositoryIpAddress(const char *reservationId);
extern int GetAvailableIpsfromRepository(unsigned int *ips);
extern unsigned int GetReservedIpFromRepository(const char *reservationId);
extern void ParseSubnetAndGatewayIp(char *xmlResponse, char **sunbetIp, char **gatewayIp);
extern unsigned long PersistSubnetInRepository(Ipv4Subnet *subnetToPersist, NetworkType networkType);
extern unsigned int PersistedIpCountInRepository();
extern DWORD GetHostIp(unsigned int *ip);
extern int WsaInitialize();
NetworkType networkTypeProvided = UnderLay;

// Returns the number of ip addresses that were returned
// If nodeNetworkResources is null, only the number of ip addresses is returned
// Otherwise, all the ip addresses (reserved + nonreserved) are populated
int GetAllIpAddresses(NodeNetworkResources *nodeNetworkResources, NetworkType networkType)
{
	int ipCount = getIpsFromNmAgent(nodeNetworkResources->ips);
	
	if (nodeNetworkResources->ips)
	{
		UpdateRepositoryIpList(ipCount, nodeNetworkResources->ips, networkType);
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
	char logMsg[2048];
	#ifdef _WIN32
	size_t msgSize = 2048;
	#endif
	
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
	sprintf_s(logMsg, msgSize, "Retrieved %d ips fron nm agent along with details\n", ipCount);
	AquaLogging::Log(lInfo, logMsg);
	for(int i = 0; i<ipCount; i++)
	{
		sprintf_s(logMsg, msgSize, "%d\n", ips[i]);
		AquaLogging::Log(lInfo, logMsg);
	}
	if(ips)
	{
		UpdateRepositoryIpList(ipCount, ips, networkTypeProvided);
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
	char logMsg[2048];
	#ifdef _WIN32
	size_t msgSize = 2048;
	#endif
	if(clusterConfig->nodeCount <= 0)
	{
		return UNEXPECTED_ERROR;
	}
	if(clusterConfig->nodeConfigs == NULL)
	{
		return UNEXPECTED_ERROR;
	}
	for(int i= 0 ; i<clusterConfig->nodeCount ; i++)
	{
		NodeConfig nc = clusterConfig->nodeConfigs[i];
		sprintf_s(logMsg, msgSize, "Comparing %u %u", nc.nodeIp, clusterConfig->localNodeIp);
		AquaLogging::Log(lInfo, logMsg);
		if(nc.nodeIp == clusterConfig->localNodeIp)
		{
			sprintf_s(logMsg, msgSize, "Found my own node config nodeIp=0x%08X prefix=0x%08X prefix-len%d\n", nc.nodeIp, nc.subnet->ipAddress, nc.subnet->prefixLength);
			AquaLogging::Log(lInfo, logMsg);
			PersistSubnetInRepository(nc.subnet, clusterConfig->net);
			sprintf_s(logMsg, msgSize, "persisted ip space for my own node\n");
			AquaLogging::Log(lInfo, logMsg);
			break;
		}
	}
	return 0;
}

DWORD AddRemoteMapping(const char *endpointId, const char *hostAddress, const char *containerAddress, const char *macAddress)
{
	char logMsg[2048];
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
	char logMsg[2048];
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

void provideConfigToNIX(ClusterConfig *clusterConfig)
{
	char logMsg[2048];
	size_t msgSize = 2048;

	sprintf_s(logMsg, msgSize, "Informing nix about the other nodes in the network");
	AquaLogging::Log(info, logMsg);
	unsigned int localIp;
	GetHostIp(&localIp);
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

DllExport DWORD InitializeNetwork(ClusterConfig *clusterConfig)
{
	unsigned long failed = 0;
	char logMsg[2048];
	#ifdef _WIN32
	size_t msgSize = 2048;
	#endif

	if (clusterConfig == NULL)
	{
		return INVALID_PARAMETER;
	}

	AquaLogging::Log(lInfo, "Going to initialize network ..");

#ifdef _WIN32
	AquaLogging::Log(lInfo, "Going to init wsa.");
	if (WsaInitialize() != 0)
	{
		return UNEXPECTED_ERROR;
	}
	AquaLogging::Log(lInfo, "WSA init successfully done.");
#endif
	switch(clusterConfig->net)
	{
		case OverLay: 
			sprintf_s(logMsg, msgSize, "Going to setup overlay network\n");
			AquaLogging::Log(lInfo, logMsg);
			networkTypeProvided = OverLay; // save the type
			failed = setupOverlay(clusterConfig);
			if(failed == 0)
			{
				sprintf_s(logMsg, msgSize, "Going to persist overlay address space\n");
				AquaLogging::Log(lInfo, logMsg);
				saveOverlayAddressSpace(clusterConfig);
				#ifdef _WIN32
				ProvideConfigToNIX(clusterConfig);
				#endif
				sprintf_s(logMsg, msgSize, "Successfully persisted Overlay Address Space\n");
				AquaLogging::Log(lInfo, logMsg);
			}
			else
			{
				sprintf_s(logMsg, msgSize, "setup Overlay returned non zero value %lu. None of the ips were saved.", failed);
				AquaLogging::Log(err, logMsg);
			}

			sprintf_s(logMsg, msgSize, "Initialize network finished. retunring %lu\n", failed);
			AquaLogging::Log(lInfo, logMsg);
			return failed;

		case UnderLay: 
			sprintf_s(logMsg, msgSize, "Going to setup underlay network\n");
			AquaLogging::Log(lInfo, logMsg);
			networkTypeProvided = UnderLay;
			switch(clusterConfig->env)
			{
			case Azure: 
				return setupUnderlayInAzure(clusterConfig);

			case StandAlone: 		
				sprintf_s(logMsg, msgSize, "Underlay in StandAlone is not a supportedconfig");
				AquaLogging::Log(warn, logMsg);

			default:
				sprintf_s(logMsg, msgSize, "Unsupported environment %d\n", clusterConfig->env);
				AquaLogging::Log(warn, logMsg);
				return UNSUPPORTED_NETWORK;
			}
		default:
			sprintf_s(logMsg, msgSize, "Unknown network type %d\n", clusterConfig->net);
			AquaLogging::Log(err, logMsg);
			return UNSUPPORTED_NETWORK;
	}
}

DllExport DWORD SetupNetworkLogging(void(*log)(int, char*))
{
	AquaLogging::InitLogging(log);
	return 0;
}

DllExport DWORD GetHostLocalIp(unsigned int *hostLocalIp)
{
	unsigned long retval = UNEXPECTED_ERROR;
	#ifdef _WIN32
	char logMsg[2048];
	size_t msgSize = 2048;
	#endif

	if(networkTypeProvided == OverLay)
	{
		#ifdef _WIN32
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
			sprintf_s(logMsg, msgSize, "Reserving Available for host local returned non-zero %d", retval);
			AquaLogging::Log(info, logMsg);
		}
		#endif
	}
	else
	{
		retval = GetHostIp(hostLocalIp);
	}
	return retval;
}

DllExport DWORD SetupNetworkEtwLogging(bool additionConsoleLoggingEnabled)
{
	AquaLogging::InitLogging(additionConsoleLoggingEnabled);
	return 0;
}

} } } } }
