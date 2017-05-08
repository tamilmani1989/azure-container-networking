/*++

Copyright (c) Microsoft Corporation

Module Name:

NetworkClientLibrary.cpp

Abstract:

This contains the implementation of external interface that can be used by clients to setup
networking for containers in Azure and OnPrem.

Author:

Sushant Sharma (susharm) 24-March-2017

--*/

#include "NetworkClientLibrary.h"
#include "Common/NclRepositoryManager.h"
#include "Common/Network.h"

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {

extern void ProvideConfigToNIX(ClusterConfig *clusterConfig);
extern int WsaInitialize();

NclRepositoryManager *GetIpRepository()
/*++

Routine Description:
	
	Retrives (or creates if does not exist) the object for ip repository. (Singleton)

Arguments:

Return Value:

	handle to ip repository.

--*/
{
	LOGINFO("Entering GetIpRepository.");
	NclRepositoryManager *_nclRepositoryVariable;
	if(nclRepositoryManager)
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
	LOGINFO("Exiting GetIpRepository.");
	return _nclRepositoryVariable;
}

int GetAllIpAddresses(
	NodeNetworkResources *nodeNetworkResources, 
	NetworkType networkType)
/*++

Routine Description:
	
	Retrieves all the ip addresses in the repository.
	If nodeNetworkResources->ips is null, only count is returned.

Arguments:
	
	nodeNetworkResources - to be populated with ip addresses
	networkType - Overlay/Underlay

Return Value:
	
	Returns the number of ip addresses in the repository. 

--*/
{
	LOGINFO("Entering GetAllIpAddresses.");
	if(nodeNetworkResources == NULL)
	{
		LOGERR("NULL value passed for nodeNetworkResources.");
		return UNEXPECTED_ERROR;
	}
	int ipCount = GetIpsFromNmAgent(nodeNetworkResources->ips);
	
	if (nodeNetworkResources->ips)
	{
		GetIpRepository()->UpdateRepositoryIpList(ipCount, nodeNetworkResources->ips, networkType);
	}
	LOGINFO("Exiting GetAllIpAddresses. Returning: %d.", ipCount);
	return ipCount;
}

DllExport DWORD GetAvailableIpAddresses(
	NodeNetworkResources *nodeNetworkResources, 
	int *count)
/*++

Routine Description:
	
	Retrieves ip adresses from repository that are not reserved.
	If nodeNetworkResources->ips is null, only count is returned.

Arguments:

	nodeNetworkResources - to be populated with ip addresses.
	count - will be populated with teh count of ip addresses populated.

Return Value:
	
	0 on success, error code otherwise.

--*/
{
	int ipCount;
	unsigned int *ips;
	LOGINFO("Entering GetAvailableIpAddresses");
	if(networkTypeProvided == OverLay)
	{
		LOGINFO("Overlay network specified");
		// overlay ips are persisted during network init
		if(nodeNetworkResources != NULL)
		{
			GetIpRepository()->GetAvailableIpsfromRepository(nodeNetworkResources->ips);
		}
		*count = GetIpRepository()->PersistedIpCountInRepository();
		LOGINFO("Exiting GetAvailableIpAddresses for overlay. Returning %d as ipCount.", *count);
		return 0;
	}

	ipCount = GetIpsFromNmAgent(NULL);	
	if (ipCount <= 0)
	{
		LOGERR("Retrieved non positive ip count from nm agent. ipCount: %d", ipCount);
		return UNEXPECTED_ERROR;
	}

	ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));
	ipCount = GetIpsFromNmAgent(ips);
	LOGINFO("Retrieved %d ips fron nm agent along with details.", ipCount);
	if (ipCount <= 0)
	{
		LOGERR("Retrieved non positive ip count from nm agent. ipCount: %d", ipCount);
		free(ips);
		return UNEXPECTED_ERROR;
	}
	for(int i = 0; i<ipCount; i++)
	{
		LOGVERBOSE("%d -> %d", i, ips[i]);
	}
	if(ips)
	{
		GetIpRepository()->UpdateRepositoryIpList(ipCount, ips, networkTypeProvided);
		free(ips);
	}
	if (nodeNetworkResources) 
	{
		nodeNetworkResources->ipCount = ipCount;
		ipCount = GetIpRepository()->GetAvailableIpsfromRepository(nodeNetworkResources->ips);
	}
	
	*count = ipCount;
	LOGINFO("Exiting GetAvailableIpAddresses.");
	return 0;
}


DllExport DWORD ReserveAvailableIpAddress(
	char *reservationId, 
	unsigned int *ipAddress)
/*++

Routine Description:
	
	Reserves an available ip address.

Arguments:
	
	reservationId - id against which ip address is returned.
	ipAddress - (out parameter) the value of the reserved ip address.

Return Value:
	
	0 on success, error code otherwise.

--*/
{
	unsigned int reservedIp = 0;
	int count = 0;
	
	if(reservationId == NULL || ipAddress == NULL)
	{
		LOGERR("Either reservationId or ipAddress is NULL");
		return UNEXPECTED_ERROR;
	}

	LOGINFO("Received request to reserve an ip against %s", reservationId);

	// only get from nm agent if repository is empty
	if (GetIpRepository()->IpRepositoryIsEmpty()) 
	{
		GetAvailableIpAddresses(NULL, &count);
		if (count <= 0)		
		{
			LOGWARN( "GetAvailableIpAddresses returned non positive ipcount %d", count);
		}
	}

	// no harm in retrying once more if repository is empty
	reservedIp = GetIpRepository()->ReserveRepositoryIpAddress(reservationId);
	LOGINFO("Allocated ip %d for reservationid %s", reservedIp, reservationId);
	
	if(reservedIp == 0)
	{
		LOGERR("Unable to reserve an ip address.");
		return UNEXPECTED_ERROR;
	}

	*ipAddress = reservedIp;
	LOGINFO("Exiting ReserveAvailableIpAddress. Reserved ip %d against %s.", *ipAddress, reservationId);
	return 0;	
}

DllExport  DWORD GetIpAddressForContainer(
	char *containerId, 
	unsigned int ipAddress)
/*++

Routine Description:

Arguments:

Return Value:

--*/
{
	UNREFERENCED_PARAMETER(containerId);
	UNREFERENCED_PARAMETER(ipAddress);
	return 0;
}

DllExport DWORD GetReservedIpAddress(
	const char *reservationId, 
	unsigned int *ipAddress)
/*++

Routine Description:
	
	Retrieves ip address that is already reserved against the reservation id.

Arguments:
	
	reservationId - 
	ipAddress - 

Return Value:
	0 on success, error code if reservation id does not exist.

--*/
{
	LOGINFO("Entering GetReservedIpAddress");
	unsigned int reservedIp = GetIpRepository()->GetReservedIpFromRepository(reservationId);
	if(reservedIp == 0)
	{
		return UNEXPECTED_ERROR;
	}

	*ipAddress = reservedIp;
	LOGINFO("Exiting GetReservedIpAddress. Returning %d for %s.", *ipAddress, reservationId);
	return 0;
}

// Releases reservation of an ip address if exists
DllExport DWORD ReleaseIpReservation(
	const char *reservationId)
/*++

Routine Description:
	
	Releases an ip reservation.

Arguments:
	
	reservationId - The id that needs to be released.

Return Value:

	0 if reservation exists, RESERVATION_NOT_FOUND otherwise.

--*/
{
	LOGINFO("Entering ReleaseIpReservation.");
	DWORD retval = 0;
	if(!GetIpRepository()->ReleaseRepositoryIpAddress(reservationId))
	{
		retval = RESERVATION_NOT_FOUND;
	}
	LOGINFO("Exiting ReleaseIpReservation. Returning %u", retval);
	return retval;
}

bool IsLocalNodeIp(
	unsigned int ip)
/*++

Routine Description:

	Checks if ip equals hostip or not.

Arguments:

	ip - ip to be checked

Return Value:
	
	true if yes, false otherwise.

--*/
{
	LOGINFO("Entering IsLocalNodeIp.");
	unsigned int hostIp;
	bool retval = false;
	if(GetHostIp(&hostIp, GetIpRepository()) != 0)
	{
		retval = false;
	}
	else
	if(hostIp == ip)
	{
		retval = true;
	}
	LOGINFO("Exiting IsLocalNodeIp. Returning %u", retval);
	return retval;
}


DllExport unsigned long SaveOverlayAddressSpace(
	ClusterConfig *clusterConfig)
/*++

Routine Description:
	
	Saves overlay address space in the ip repository.

Arguments:
	
	clusterConfig - contains the overlay address space that needs to be saved.

Return Value:

	0 if successfully saved, error code otherwise.

--*/
{
	LOGINFO("Entering SaveOverlayAddressSpace.");
	if(clusterConfig == NULL)
	{
		LOGERR("Clusterconfig is NULL");
		return UNEXPECTED_ERROR;
	}
	if(clusterConfig->nodeCount <= 0)
	{
		LOGERR("Overlay node count is non-positive. %d", clusterConfig->nodeCount);
		return UNEXPECTED_ERROR;
	}
	if(clusterConfig->nodeConfigs == NULL)
	{
		LOGERR("nodeConfigs are NULL");
		return UNEXPECTED_ERROR;
	}
	for(int i= 0 ; i<clusterConfig->nodeCount ; i++)
	{
		NodeConfig nc = clusterConfig->nodeConfigs[i];
		LOGINFO( "Comparing %u %u", nc.nodeIp, clusterConfig->localNodeIp);
		if(nc.nodeIp == clusterConfig->localNodeIp)
		{
			LOGINFO("Found my own node config nodeIp=0x%08X prefix=0x%08X prefix-len%d\n", nc.nodeIp, nc.subnet->ipAddress, nc.subnet->prefixLength);
			GetIpRepository()->PersistSubnetInRepository(nc.subnet, clusterConfig->net);
			LOGINFO("Persisted ip space for my own node.");
			break;
		}
	}
	LOGINFO("Exiting SaveOverlayAddressSpace. Returning 0.");
	return 0;
}

DllExport void SetEnvironment(
	NetworkType networkType)
/*++

Routine Description:

	Sets the network type

Arguments:

Return Value:

--*/
{
	networkTypeProvided = networkType;
}

DllExport DWORD InitializeNetwork(
	ClusterConfig *clusterConfig)
/*++

Routine Description:

	Initializes the network. Creates the network if needed.

Arguments:
	
	clusterConfig - configuration for the whole cluster (for both overlay/underlay)

Return Value:

	0 on successful initialization, error code otherwise.

--*/
{
	unsigned long failed = 0;
	DWORD retval;
	LOGINFO("Entering InitializeNetwork.");
	if (clusterConfig == NULL)
	{
		LOGERR("ClusterConfig is NULL.");
		return INVALID_PARAMETER;
	}

	AquaLogging::Log(info, "Going to initialize network ..");

#ifdef _WIN32
	AquaLogging::Log(info, "Going to init wsa.");
	if (WsaInitialize() != 0)
	{
		LOGERR("WSA Initialization failed.");
		return UNEXPECTED_ERROR;
	}
	LOGINFO("WSA init successfully done.");

	DWORD VfpError = VfpOperations::StartVfpService();
	if(VfpError != 0)
	{
		AquaLogging::Log(err, "Vfp Initializtion failed.. cannot continue");
		return VfpError;
	}
#endif
	GetIpRepository()->InitRepositoryIpAllocations();
	switch(clusterConfig->net)
	{
		case OverLay: 
			LOGINFO("Going to setup overlay network.");
			SetEnvironment(OverLay); // save the type
			failed = SetupOverlay(clusterConfig);
			if(failed == 0)
			{
				LOGINFO("Going to persist overlay address space\n");
				SaveOverlayAddressSpace(clusterConfig);
				#ifdef _WIN32
				LOGINFO("Going to provide config to NIX.");
				ProvideConfigToNIX(clusterConfig);
				LOGINFO("Successfully provided config to NIX.");
				#endif
				LOGINFO("Successfully persisted Overlay Address Space.");
			}
			else
			{
				LOGERR("Setup Overlay returned non zero value %lu. None of the ips were persisted.", failed);
			}

			LOGINFO("Initialize network finished. retunring %lu", failed);
			return failed;

		case UnderLay: 
			LOGINFO("Going to setup underlay network.");
			SetEnvironment(UnderLay);
			switch(clusterConfig->env)
			{
			case Azure: 
				LOGINFO("Will setup underlay in Azure");
				retval = SetupUnderlayInAzure(clusterConfig, GetIpRepository());
				LOGINFO("Exiting InitializeNetwork (aure/underlay. Returning %u.", retval);
				return retval;
				break;

			case StandAlone: 		
				LOGERR("Underlay in StandAlone is not a supportedconfig");
				return UNSUPPORTED_NETWORK;
				break;

			default:
				LOGERR("Unsupported environment setup requested: %d", clusterConfig->env);
				return UNSUPPORTED_NETWORK;
				break;
			}
		default:
			LOGERR("Unknown network type %d", clusterConfig->net);
			return UNSUPPORTED_NETWORK;
	}
}

DllExport DWORD SetupNetworkLogging(
	void(*log)(int, char*))
/*++

Routine Description:

	Accepts an external logger.

Arguments:

	log - pointer to the log method.

Return Value:

	0 upon success.

--*/
{
	LOGINFO("Entering SetupNetworkLogging.");
	AquaLogging::InitLogging(log);
	LOGINFO("Exiting SetupNetworkLogging.");
	return 0;
}

DllExport DWORD GetHostLocalIp(
	unsigned int *hostLocalIp)
/*++

Routine Description:

		To retrieve the ip address on host that containers can reach.

Arguments:

	hostLocalIp - (output parameter) will be populated with the host lcoal ip.

Return Value:

	0 on success.

--*/
{
	return GetHostIp(hostLocalIp, GetIpRepository());
}

DllExport DWORD SetupNetworkEtwLogging(
	bool additionConsoleLoggingEnabled)
/*++

Routine Description:

	This can be used to setup console output when external logger is not set.

Arguments:

	additionConsoleLoggingEnabled - boolean to indicate wheter console printing should be enabled or not.

Return Value:

	0 on success.

--*/
{
	AquaLogging::InitLogging(additionConsoleLoggingEnabled);
	return 0;
}

} } } } }
