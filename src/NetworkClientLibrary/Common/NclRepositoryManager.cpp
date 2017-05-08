/*++

Copyright (c) Microsoft Corporation

Module Name:

NclRepositoryManager.c

Abstract:

Implementation of methods to manage the state of network client library.

Author:

Sushant Sharma (susharm) 24-March-2017

--*/

#include "NclRepositoryManager.h"

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {

#pragma region private_methods
IpStatus * NclRepositoryManager::getRepositoryReservedIpUnTransacted(
	const char *reservationId)
/*++

Routine Description:

	Retrieves a reserved ip address from the repository.
	This is an untransacted method. 
	It is required to lock the repository before calling this method.

Arguments:

	reservationId - Reservation id for which the ip address needs to be retrieved.

Return Value:

	If the reservation id exists in the repository, then an object containing the
	reservation-id and the ip-address is returned.
	Otherwise, NULL is returned.

--*/
{

	LOGINFO("Entering getRepositoryReservedIpUnTransacted.");
	int i;
	if (repositoryIpAllocations)
	{
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			LOGVERBOSE("Going to obtain ip # %d", i);
			IpStatus *retval = &repositoryIpAllocations->statusForAllIpAddresses[i];
			LOGVERBOSE("Going to check if ip # %d is reserved or not", i);
			if (retval->isReserved)
			{
				LOGVERBOSE("ip # %d is reserved.", i);
				if (strcmp(reservationId, retval->id) == 0)
				{
					LOGINFO("Exiting getRepositoryReservedIpUnTransacted. Reservation id %s for %u matched input %s.", 
						retval->id, retval->ip, reservationId);
					return retval;
				}
				
				LOGVERBOSE("Reservation id %s for %u did not match input %s.", retval->id, retval->ip, reservationId);				
			}
			else
			{
				LOGVERBOSE("ip # %d is not reserved.", i);
			}
		}
	}
	LOGINFO("Entering getRepositoryReservedIpUnTransacted. Returning NULL.");
	return NULL;
}

unsigned long NclRepositoryManager::releaseReservedRepositoryIpUnTransacted(
	const char *reservationId)
/*++

Routine Description:

	Releases a reserved ip address from the repository.
	This is an untransacted method.
	It is required to lock the repository before calling this method.

Arguments:

	reservationId - Reservation id that needs to be released.

Return Value:

	If the reservation id exists in the repository, return value is 1 indicating 
	that that it is successfully released.
	If reservation id was not found, 0 is returned.

--*/
{
	LOGINFO("Entering releaseReservedRepositoryIpUnTransacted.");
	int i;
	unsigned long reservationFound = 0;

	if (repositoryIpAllocations)
	{
		LOGINFO("Going to try to release %s.", reservationId);
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			IpStatus *retval = &repositoryIpAllocations->statusForAllIpAddresses[i];
			LOGINFO("Checking if reservation for %u should be released?", retval->ip);
			if (retval->isReserved == 1)
			{
				if (strcmp(retval->id, reservationId) == 0)
				{
					LOGINFO("Yes, going to release reservation %s %u", reservationId, retval->ip);
					free(repositoryIpAllocations->statusForAllIpAddresses[i].id);
					repositoryIpAllocations->statusForAllIpAddresses[i].id = NULL;
					repositoryIpAllocations->statusForAllIpAddresses[i].isReserved = 0;
					reservationFound = 1;
					break;
				}
				LOGINFO("No, %u is not the target.", retval->ip);
			}
			else
			{
				LOGINFO("No, %u is not reserved.", retval->ip);
			}
		}
	}

	LOGINFO("Exiting releaseReservedRepositoryIpUnTransacted. Returning %lu.", reservationFound);
	return reservationFound;
}

void NclRepositoryManager::updateRepositoryIpListUnTransacted(
	int ipCount, 
	unsigned int* ips, 
	NetworkType networkType)
/*++

Routine Description:

	This routine adds/removes ip addresses from the repository.
	If an ip is found and reserved, reservation id is preserved.
	The repository ip that is not in the input list, that ip is removed from repository.

Arguments:

	ipCount - Number of ips to be persisted in repository
	ips - List of ips to be preserved.
	networkType - Underlay/Overlay

Return Value:

--*/
	{

	LOGINFO("Entering updateRepositoryIpListUnTransacted. ipCount = %d.", ipCount);

	int i;	
	IpStatus *localIpStatus;

	if (ipCount <= 0)
	{
		LOGERR("Received %d ips to upate in repository. Will not do anything.", ipCount);
		return;
	}

	if(ips == NULL)
	{
		LOGERR("Received NULL as the list of ips. Returning.")
			return;
	}

	localIpStatus = (IpStatus *)malloc(ipCount * sizeof(IpStatus));
	for (i = 0; i<ipCount; i++)
	{
		localIpStatus[i].ip = ips[i];

		// For overlay, first two ip-addresses are blocked 
		if (networkType == OverLay && (i == 0 || i == 1))
		{
			localIpStatus[i].isReserved = 1;
			localIpStatus[i].id = "Sys_Reserved";
		}
		else
		{
			LOGVERBOSE("Going to get status for ip %u.", ips[i]);
			IpStatus *repositoryIpStatus = GetRepositoryIpAllocationStatus(ips[i]);
			LOGVERBOSE("Successfully retrieved status for ip %u.", ips[i]);
			if (repositoryIpStatus)
			{
				LOGVERBOSE("Received ip from repository %u isReserved = %d.", repositoryIpStatus->ip, repositoryIpStatus->isReserved);
				localIpStatus[i].isReserved = repositoryIpStatus->isReserved;
				if (repositoryIpStatus->id != NULL)
				{
					LOGVERBOSE("id = %s isReserved = %d.", repositoryIpStatus->id, repositoryIpStatus->isReserved);
					size_t idLen = strlen(repositoryIpStatus->id) * sizeof(char);
					localIpStatus[i].id = (char *)malloc(idLen + 1);
					if(localIpStatus[i].id == NULL)
					{
						LOGERR("Failed to allocate memory for localIpStatus[i].id.");
						for(int j=0; j<i; j++)
						{
							free(localIpStatus[j].id);
						}
						free(localIpStatus);
						localIpStatus = NULL;
						LOGINFO("Exiting updateRepositoryIpListUnTransacted.");
						return;
					}
					strcpy_s(localIpStatus[i].id, idLen + 1, repositoryIpStatus->id);
				}
				else
				{
					localIpStatus[i].isReserved = 0;
					localIpStatus[i].id = NULL;
				}
			}
			else
			{
				localIpStatus[i].id = NULL;
				localIpStatus[i].isReserved = 0;
			}
		}
	}

	LOGVERBOSE("Going to free current ip status from repository.");
	FreeStatusInRepositoryIpAllocationsUnTransacted();
	LOGVERBOSE("Going to update new status for all ip addresses.");
	repositoryIpAllocations->statusForAllIpAddresses = localIpStatus;
	repositoryIpAllocations->count = ipCount;
}
#pragma endregion private_methods

#pragma region dllexport_methods
DllExport void NclRepositoryManager::InitRepositoryIpAllocations()
/*++

Routine Description:

	Initializes ip allocation if not already initialized.

Arguments:


Return Value:

--*/
{
	LOGINFO("Entering InitRepositoryIpAllocations.");
	LockMutex
	if (!repositoryIpAllocations)
	{
		repositoryIpAllocations = (IpAllocations *)malloc(sizeof(IpAllocations));
		memset(repositoryIpAllocations, 0, sizeof(IpAllocations));
	}
	UnlockMutex
	LOGINFO("Exiting InitRepositoryIpAllocations.");
}	

DllExport unsigned int NclRepositoryManager::GetRepositoryPrimaryIp()
/*++

Routine Description:

	Retrieves the primary IP from repository.

Arguments:


Return Value:
	Returns zero if repository is not yet initialized.
	primaryIp value, otherwise.
--*/
{
	LOGINFO("Entering InitRepositoryIpAllocations.");
	unsigned int retval = 0;
	if(repositoryIpAllocations)
	{
		retval = repositoryIpAllocations->primaryIp;
	}
	LOGINFO("Exiting GetRepositoryPrimaryIp. Returning %u", retval);
	return retval;
}

DllExport void NclRepositoryManager::SetRepositoryPrimaryIp(
	unsigned int ip)
/*++

Routine Description:

	Sets the primary IP in repository.

Arguments:

	Value of ip address to set as primary ip.

Return Value:
	
--*/
{
	LOGINFO("Entering SetRepositoryPrimaryIp. ip = %u", ip);
	LockMutex
	if (repositoryIpAllocations)
	{
		repositoryIpAllocations->primaryIp = ip;
	}
	UnlockMutex
	LOGINFO("Entering SetRepositoryPrimaryIp. ip = %u", ip);
}

DllExport int NclRepositoryManager::FreeStatusInRepositoryIpAllocationsUnTransacted()
/*++

Routine Description:

	Cleans up the repository. Free all the ip addresses from repository.

Arguments:

Return Value:

	0 if repository was initialized, 1 otherwise.

--*/
{
	int i;
	 
	if (repositoryIpAllocations)
	{
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			IpStatus tempStatus = repositoryIpAllocations->statusForAllIpAddresses[i];
			if (tempStatus.id != NULL && strcmp(tempStatus.id, "Sys_Reserved")!=0)
			{
				LOGINFO("Going to free %s (%u) from repository.", tempStatus.id, tempStatus.ip);
				free(tempStatus.id);
				tempStatus.isReserved = false;
				tempStatus.id = NULL;
			}
		}
		if (repositoryIpAllocations->statusForAllIpAddresses)
		{
			free(repositoryIpAllocations->statusForAllIpAddresses);
			repositoryIpAllocations->statusForAllIpAddresses=NULL;
		}
		repositoryIpAllocations->count=0;
		return 0;
	}
	return 1;
}

DllExport void NclRepositoryManager::UpdateRepositoryIpList(
	int ipCount, 
	unsigned int *ips, 
	NetworkType networkType)
/*++

Routine Description:
	This routine adds/removes ip addresses from the repository.
	If an ip is found and reserved, reservation id is preserved.
	The repository ip that is not in the input list, that ip is removed from repository.

Arguments:

	ipCount - Number of ips to be persisted in repository
	ips - List of ips to be preserved.
	networkType - Underlay/Overlay

Return Value:

--*/
{
	LOGINFO("Entering UpdateRepositoryIpList.");
	LockMutex
		updateRepositoryIpListUnTransacted(ipCount, ips, networkType);
	UnlockMutex
	LOGINFO("Exiting UpdateRepositoryIpList.");
}

DllExport bool NclRepositoryManager::IpRepositoryIsEmpty()
/*++

Routine Description:
	This routine checks if repository is empty or not.

Arguments:

Return Value:
	true, if repository is empty. false, otherwise.
--*/
{
	bool retval = false;
	LOGINFO("Entering IpRepositoryIsEmpty");

	if (repositoryIpAllocations == NULL)
	{
		LOGVERBOSE("repository ip allocations is NULL");
		retval = true;
	}

	LOGINFO("Exiting IpRepositoryIsEmpty. Retrning %d", retval);
	return retval;
}

DllExport unsigned int NclRepositoryManager::ReserveRepositoryIpAddress(
	char *reservationId)
/*++

Routine Description:
	Reserves an ip address that is available.

Arguments:
	Reservation id against which ip address needs to be reserved.

Return Value:
	The ip address that is reserved, 0 otherwise.

--*/
{
	unsigned int returnIp = 0;
	LOGINFO("Enetering ReserveRepositoryIpAddress.");
	LockMutex;

	if (repositoryIpAllocations)
	{
		IpStatus *repositoryReservedIpStatus = getRepositoryReservedIpUnTransacted(reservationId);
		if (repositoryReservedIpStatus != NULL)
		{
			LOGVERBOSE("Retrieved an existing ip %u for %s.", repositoryReservedIpStatus->ip, reservationId);
			returnIp = repositoryReservedIpStatus->ip;
		}
		else
		{
			LOGVERBOSE("Could not retrieve an existing ip for %s. Will reserve one.", reservationId);
			AquaLogging::Log(verbose, logMsg);
			repositoryReservedIpStatus = ReserveRepositoryIp(reservationId);
			if (repositoryReservedIpStatus)
			{
				returnIp = repositoryReservedIpStatus->ip;
			}
		}
	}
	UnlockMutex;
	LOGINFO("Exiting ReserveRepositoryIpAddress. Returning %u.", returnIp);
	return returnIp;
}

DllExport unsigned int NclRepositoryManager::PersistedIpCountInRepository()
/*++

Routine Description:

	Retrieve the ip count saved in repository.

Arguments:

Return Value:
	number of ip addresses saved if repository is initialized, 0 otherwise

--*/
{
	int count = 0;
	LOGINFO("Entering PersistedIpCountInRepository");
	LockMutex
	if (repositoryIpAllocations != NULL)
	{
		count = repositoryIpAllocations->count;
	}
	UnlockMutex
	LOGINFO("Exiting PersistedIpCountInRepository. Returning %u", count);
	return count;

}

DllExport int NclRepositoryManager::GetAvailableIpsfromRepository(
	unsigned int *ips)
/*++

Routine Description:

	retrieves the values of ip addresses that are not reserved and are available.

Arguments:

	ips - out parameter where ip addresses should be saved.

Return Value:

	number of available ip addresses, 0 if repository is not initialized.

--*/
{
	int i, ipCount = 0;
	LOGINFO("Entering GetAvailableIpsfromRepository.");
	LockMutex;
	if (repositoryIpAllocations)
	{
		size_t passedCount;
		if (ips)
		{
#ifdef _WIN32
			passedCount = _msize(ips) / sizeof(unsigned int);
#endif
#ifdef __linux__
			passedCount = malloc_usable_size(ips) / sizeof(unsigned int);
#endif 
		}
		else
		{
			passedCount = repositoryIpAllocations->count;
		}
		if(passedCount < repositoryIpAllocations->count)
		{
			LOGERR("Insufficient memory passed in ips. Passed Count = %llu. Count in repo = %d. ", 
				passedCount, repositoryIpAllocations->count);
			ipCount = -1;
		}
		else
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			if (repositoryIpAllocations->statusForAllIpAddresses[i].isReserved == false)
			{
				if (ips)
				{
					// may be we should check if passed memory is sufficient or not
					ips[ipCount] = repositoryIpAllocations->statusForAllIpAddresses[i].ip;
				}
				ipCount++;
			}
		}
	}
	UnlockMutex;
	LOGINFO("Exiting GetAvailableIpsfromRepository. Returning %d.", ipCount);
	return ipCount;
}
#pragma endregion dllexport_methods

#pragma region public_methods
IpStatus * NclRepositoryManager::ReserveRepositoryIp(
	char *reservationId)
/*++

Routine Description:

	Reserve an available ip address in repository.

Arguments:

	reservationId - The id against which ip address needs to be reserved.

Return Value:

	The reservation id and the ip address reserved.
	NULL if repository is not initialized, or an ip address is not available.
--*/
{
	int i;
	
	if (repositoryIpAllocations)
	{
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			LOGVERBOSE("Going to check if ip # %d is reserved or not", i);
			if (repositoryIpAllocations->statusForAllIpAddresses[i].isReserved == false)
			{
				LOGVERBOSE("ip # %d is not reserved.", i);
				size_t resSize = strlen(reservationId) * sizeof(char);
				repositoryIpAllocations->statusForAllIpAddresses[i].id = (char *)malloc(resSize + 1);
				strcpy_s(repositoryIpAllocations->statusForAllIpAddresses[i].id, resSize + 1, reservationId);
				repositoryIpAllocations->statusForAllIpAddresses[i].isReserved = true;
				LOGVERBOSE("returning ip # %d (%u) for %s", i, repositoryIpAllocations->statusForAllIpAddresses[i].ip, reservationId);
				return & (repositoryIpAllocations->statusForAllIpAddresses[i]);
			}
			LOGVERBOSE("ip # %d is reserved.", i);
		}
	}
	return NULL;
}

IpStatus * NclRepositoryManager::GetRepositoryIpAllocationStatus(
	unsigned int ip)
/*++

Routine Description:

	Retrieves whether an ip address is reserved or not.

Arguments:
	
	ip - ip address to be retrieved fromr repository.

Return Value:

	ip address and reservation-id if found. NULL, otherwise.

--*/
{
	LOGINFO("Enetering GetRepositoryIpAllocationStatus");
	int i;
	if (repositoryIpAllocations)
	{
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			IpStatus *retval = &repositoryIpAllocations->statusForAllIpAddresses[i];
			if (retval->ip == ip)
			{
				LOGINFO("Exiting GetRepositoryIpAllocationStatus. Returning %s %u.", retval->id, retval->ip);
				return retval;
			}
		}
	}
	LOGINFO("Exiting GetRepositoryIpAllocationStatus. Returning NULL.");
	return NULL;
}

unsigned long NclRepositoryManager::ReleaseRepositoryIpAddress(
	const char *reservationId)
/*++

Routine Description:

	Releases a reserved ip address from the repository.
	This is an untransacted method.
	It is required to lock the repository before calling this method.

Arguments:

	reservationId - Reservation id that needs to be released.

Return Value:

	If the reservation id exists in the repository, return value is 1 indicating 
	that that it is successfully released.
	If reservation id was not found, 0 is returned.

--*/
{
	unsigned long reservationStatus;
	LOGINFO("Enetering ReleaseRepositoryIpAddress.");
	LockMutex;
	reservationStatus = releaseReservedRepositoryIpUnTransacted(reservationId);		
	UnlockMutex;
	LOGINFO("Exiting ReleaseRepositoryIpAddress.");
	return reservationStatus;
}

unsigned int NclRepositoryManager::GetReservedIpFromRepository(
	const char *reservationId)
/*++

Routine Description:

	Retrieves an ip address from repository.

Arguments:

	reservationId - The id whose corresponding ip address needs to be retrieved.

Return Value:

	The ip address. 0 if reservation id is not found in repository.
--*/
{
	unsigned int returnIp = 0;
	
	LockMutex;
	if (repositoryIpAllocations)
	{
		IpStatus *repositoryReservedIpStatus = getRepositoryReservedIpUnTransacted(reservationId);
		LOGINFO("Going to get an ip from repository for %s", reservationId );
		if (repositoryReservedIpStatus)
		{
			LOGINFO("Successfully received an ip from repository for %s. IP = %u", reservationId, repositoryReservedIpStatus->ip);
			returnIp = repositoryReservedIpStatus->ip;
		}
		else
		{
			LOGINFO("Was unsucessful in obtaining an existing ip from repository for %s.", reservationId);
		}
	}
	UnlockMutex;
	return returnIp;
}

unsigned long NclRepositoryManager::PersistSubnetInRepository(
	Ipv4Subnet *subnetToPersist, 
	NetworkType networkType)
/*++

Routine Description:
	
	Persist ip addresses in the subnet to repository.

Arguments:

	subnetToPersist - Subnet
	networkType - Overlay/Underlay

Return Value:

	0 indicating success

--*/
{
	LOGINFO("Entering PersistSubnetInRepository.");
	if (subnetToPersist == NULL)
	{
		LOGERR("ERROR: NULL passed in subnetToPersist.");
		return UNEXPECTED_ERROR;
	}
	LockMutex	
	unsigned int mask = 0xFFFFFFFF;
	int leftShiftCount = 32 - subnetToPersist->prefixLength;
	mask = mask << leftShiftCount;
	LOGINFO("Subnet mask = 0x%08X", mask);
	LOGINFO("IpAddress = 0x%08X", subnetToPersist->ipAddress);
	
	unsigned int startingIp = (mask) & ntohl(subnetToPersist->ipAddress);

	LOGINFO("Starting IpAddress = 0x%08X", startingIp);

	unsigned int ipCount = 1;
	for( int i = 0; i<leftShiftCount-1 ;i++)
	{
		ipCount = (ipCount << 1);
		ipCount = ipCount | 0x1;
	}
	LOGINFO("Ip Count = %d", ipCount);
	unsigned int *ips = (unsigned int*)malloc(ipCount *(sizeof(unsigned int)));
	unsigned int i = 0;
	while(i < ipCount)
	{
		ips[i] = htonl(startingIp);
		LOGVERBOSE("Saving 0x%08X", ips[i]);
		startingIp++;
		i++;
	}
	updateRepositoryIpListUnTransacted(ipCount, ips, networkType);
	free(ips);
	LOGINFO("Updated ip repository with %d ips", ipCount);
	UnlockMutex
	LOGINFO("Exiting PersistSubnetInRepository.");
	return 0;
}
#pragma endregion public_methods
} } } } }