/****************************** Module Header ******************************\
* Module Name:  NclRepositoryManager.c
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Manages the state of sdn client library (ip addresses and reservations for now)
* It needs to be integrated with IPAM.
*
\***************************************************************************/

#include <stdlib.h>
#include <string.h>
#include <iostream>
#include "nclcore.h"
#include "../../Logging/Logging.h"

#ifdef _WIN32
#include <windows.h>
#include <mutex>
#include <WinSock2.h>
std::mutex ipAllocationsMutex;
#define LockMutex ipAllocationsMutex.lock();
#define UnlockMutex ipAllocationsMutex.unlock();
#endif

#ifdef __linux__
#include <stdio.h>
#include <arpa/inet.h>
pthread_mutex_t ipAllocationsLock = PTHREAD_MUTEX_INITIALIZER;
#define LockMutex pthread_mutex_lock(&ipAllocationsLock);
#define UnlockMutex pthread_mutex_unlock(&ipAllocationsLock);
#define strcpy_s(dst, count, src) strncpy(dst, src, count)
#endif
#define info 1

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
IpAllocations *repositoryIpAllocations;

void FreeStatusInRepositoryIpAllocations(IpAllocations *myRepositoryIpAllocations)
{
	int i;
	 
	#ifdef _WIN32
	size_t msgSize = 2048;
	#endif
	if (myRepositoryIpAllocations)
	{
		for (i = 0; i<myRepositoryIpAllocations->count; i++)
		{
			IpStatus tempStatus = myRepositoryIpAllocations->statusForAllIpAddresses[i];
			if (tempStatus.id != NULL && strcmp(tempStatus.id, "Sys_Reserved")!=0)
			{
				sprintf_s(logMsg, msgSize, "Going to free %s\n", tempStatus.id);
				AquaLogging::Log(info, logMsg);
				free(tempStatus.id);
				tempStatus.isReserved = false;
				tempStatus.id = NULL;
			}
		}
		if (myRepositoryIpAllocations->statusForAllIpAddresses)
		{
			free(myRepositoryIpAllocations->statusForAllIpAddresses);
		}
	}
}

IpStatus *GetRepositoryReservedIp(const char *reservationId)
{
	int i;
	char logMsg[2048];
#ifdef _WIN32
	size_t msgSize = 2048;
#endif
	if (repositoryIpAllocations)
	{
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			sprintf_s(logMsg, msgSize, "Going to obtain ip # %d", i);
			AquaLogging::Log(verbose, logMsg);
			IpStatus *retval = &repositoryIpAllocations->statusForAllIpAddresses[i];
			sprintf_s(logMsg, msgSize, "Going to check if ip # %d is reserved or not", i);
			AquaLogging::Log(verbose, logMsg);
			if (retval->isReserved)
			{
				sprintf_s(logMsg, msgSize, "ip # %d is reserved.", i);
				AquaLogging::Log(verbose, logMsg);
				if (strcmp(reservationId, retval->id) == 0)
				{
					return retval;
				}
			}
			else
			{
				sprintf_s(logMsg, msgSize, "ip # %d is not reserved.", i);
				AquaLogging::Log(verbose, logMsg);				
			}
		}
	}
	return NULL;
}

IpStatus *ReserveRepositoryIp(char *reservationId)
{
	int i;
	char logMsg[2048];
#ifdef _WIN32
	size_t msgSize = 2048;
#endif
	if (repositoryIpAllocations)
	{
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			sprintf_s(logMsg, msgSize, "Going to obtain ip # %d", i);
			AquaLogging::Log(verbose, logMsg);
			sprintf_s(logMsg, msgSize, "Going to check if ip # %d is reserved or not", i);
			AquaLogging::Log(verbose, logMsg);
			if (repositoryIpAllocations->statusForAllIpAddresses[i].isReserved == false)
			{
				sprintf_s(logMsg, msgSize, "ip # %d is not reserved.", i);
				AquaLogging::Log(verbose, logMsg);
				size_t resSize = strlen(reservationId) * sizeof(char);
				repositoryIpAllocations->statusForAllIpAddresses[i].id = (char *)malloc(resSize + 1);
				strcpy_s(repositoryIpAllocations->statusForAllIpAddresses[i].id, resSize + 1, reservationId);
				repositoryIpAllocations->statusForAllIpAddresses[i].isReserved = true;
				sprintf_s(logMsg, msgSize, "returning ip # %d (%u) for %s", i, repositoryIpAllocations->statusForAllIpAddresses[i].ip, reservationId);
				AquaLogging::Log(verbose, logMsg);
				return & (repositoryIpAllocations->statusForAllIpAddresses[i]);
			}
			sprintf_s(logMsg, msgSize, "ip # %d is reserved.", i);
			AquaLogging::Log(verbose, logMsg);			
		}
	}
	return NULL;
}

unsigned long ReleaseReservedRepositoryIp(const char *reservationId)
{
	int i;
	unsigned long reservationFound = 0;
	 
	#ifdef _WIN32
	size_t msgSize = 2048;
	#endif
	if (repositoryIpAllocations)
	{
		sprintf_s(logMsg, msgSize, "Going to try to release %s\n", reservationId);
		AquaLogging::Log(info, logMsg);
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			IpStatus *retval = &repositoryIpAllocations->statusForAllIpAddresses[i];
			sprintf_s(logMsg, msgSize, "Checking if reservation for %u should be released\n", retval->ip);
			AquaLogging::Log(info, logMsg);
			if (retval->isReserved == 1)
			{
				if (strcmp(retval->id, reservationId) == 0) 
				{
					sprintf_s(logMsg, msgSize, "yes, going to release reservation %s %u", reservationId, retval->ip);
					AquaLogging::Log(info, logMsg);
					free(repositoryIpAllocations->statusForAllIpAddresses[i].id);
					repositoryIpAllocations->statusForAllIpAddresses[i].id = NULL;
					repositoryIpAllocations->statusForAllIpAddresses[i].isReserved = 0;
					reservationFound = 1;
					break;
				}
				sprintf_s(logMsg, msgSize, "no, %u not the target", retval->ip);
				AquaLogging::Log(info, logMsg);
			}
			else
			{
				sprintf_s(logMsg, msgSize, "no, %u not reserved)\n", retval->ip);
				AquaLogging::Log(info, logMsg);
			}
		}
	}
	return reservationFound;
}

IpStatus *GetRepositoryIpAllocationStatus(unsigned int ip)
{
	int i;
	if (repositoryIpAllocations)
	{
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			IpStatus *retval = &repositoryIpAllocations->statusForAllIpAddresses[i];
			if (retval->ip == ip)
			{
				return retval;
			}
		}
	}
	return NULL;
}

void UpdateRepositoryIpListWithoutLock(int ipCount, unsigned* ips, NetworkType networkType)
{
	 
	#ifdef _WIN32
	size_t msgSize = 2048;
	#endif
	sprintf_s(logMsg, msgSize, "Inside repository lock ipCount=%d\n", ipCount);
	AquaLogging::Log(verbose, logMsg);
	int i;
	// non performant but okay for now, since this will move to ipam anyways
	IpStatus *localIpStatus;

	if (ipCount < 0)
	{
		sprintf_s(logMsg, msgSize, "Error: Received %d ips to upate in repository. Will not do anything.", ipCount);
		AquaLogging::Log(err, logMsg);
		return;
	}
	localIpStatus = (IpStatus *)malloc(ipCount * sizeof(IpStatus));
	for (i = 0; i<ipCount; i++)
	{
		localIpStatus[i].ip = ips[i];
		
		if( networkType == OverLay && (i==0 || i==1))
		{
			localIpStatus[i].isReserved = 1;
			localIpStatus[i].id = "Sys_Reserved";
		}
		else
		{
			sprintf_s(logMsg, msgSize, "Going to get status for ip %u\n", ips[i]);
			AquaLogging::Log(verbose, logMsg);
			IpStatus *repositoryIpStatus = GetRepositoryIpAllocationStatus(ips[i]);
			sprintf_s(logMsg, msgSize, "Successfully retrieved status for ip %u\n", ips[i]);
			AquaLogging::Log(verbose, logMsg);
			if (repositoryIpStatus)
			{
				sprintf_s(logMsg, msgSize, "Received ip from repository %u isReserved=%d\n",
					   repositoryIpStatus->ip, repositoryIpStatus->isReserved);
				AquaLogging::Log(verbose, logMsg);
				localIpStatus[i].isReserved = repositoryIpStatus->isReserved;
				if (repositoryIpStatus->id != NULL)
				{
					sprintf_s(logMsg, msgSize, "id= %s res:%d\n", repositoryIpStatus->id,repositoryIpStatus->isReserved);
					AquaLogging::Log(verbose, logMsg);
					size_t idLen = strlen(repositoryIpStatus->id) * sizeof(char);
					localIpStatus[i].id = (char *)malloc(idLen+1);
					strcpy_s(localIpStatus[i].id, idLen+1, repositoryIpStatus->id);
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

	sprintf_s(logMsg, msgSize, "Going to free current ip status from repository\n");
	AquaLogging::Log(verbose, logMsg);
	FreeStatusInRepositoryIpAllocations(repositoryIpAllocations);
	sprintf_s(logMsg, msgSize, "Going to update new status for all ip addresses\n");
	AquaLogging::Log(verbose, logMsg);
	if (!repositoryIpAllocations)
	{
		repositoryIpAllocations = (IpAllocations *)malloc(sizeof(IpAllocations));
	}
	repositoryIpAllocations->statusForAllIpAddresses = localIpStatus;
	repositoryIpAllocations->count = ipCount;
}

DllExport void UpdateRepositoryIpList(int ipCount, unsigned int *ips, NetworkType networkType)
{
	LockMutex
	UpdateRepositoryIpListWithoutLock(ipCount, ips, networkType);
	UnlockMutex
}

DllExport bool IpRepositoryIsEmpty()
{
	char logMsg[2048];
#ifdef _WIN32
	size_t msgSize = 2048;
#endif
	sprintf_s(logMsg, msgSize, "Going to check if IpRepositoryIsEmpty or not.");
	AquaLogging::Log(verbose, logMsg);
	if(repositoryIpAllocations == NULL)
	{
		sprintf_s(logMsg, msgSize, "IpRepositoryIsEmpty returning true");
		AquaLogging::Log(verbose, logMsg);
		return true;
	}

	sprintf_s(logMsg, msgSize, "IpRepositoryIsEmpty returning false");
	AquaLogging::Log(verbose, logMsg);
	return false;
}

DllExport unsigned int ReserveRepositoryIpAddress(char *reservationId)
{
	unsigned int returnIp = 0;
	char logMsg[2048];
#ifdef _WIN32
	size_t msgSize = 2048;
#endif
	LockMutex;

	if (repositoryIpAllocations)
	{
		IpStatus *repositoryReservedIpStatus = GetRepositoryReservedIp(reservationId);
		if (repositoryReservedIpStatus != NULL)
		{
			sprintf_s(logMsg, msgSize, "Retrieved an existing ip %u for %s.", repositoryReservedIpStatus->ip, reservationId);
			AquaLogging::Log(verbose, logMsg);
			returnIp = repositoryReservedIpStatus->ip;
		}
		else
		{
			sprintf_s(logMsg, msgSize, "Could not retrieve an existing ip for %s. Will reserve one.", reservationId);
			AquaLogging::Log(verbose, logMsg);
			repositoryReservedIpStatus = ReserveRepositoryIp(reservationId);
			if (repositoryReservedIpStatus)
			{
				returnIp = repositoryReservedIpStatus->ip;
			}
		}
	}
	UnlockMutex;
	return returnIp;
}

unsigned long ReleaseRepositoryIpAddress(const char *reservationId)
{
	unsigned long reservationStatus;
	LockMutex;
	reservationStatus = ReleaseReservedRepositoryIp(reservationId);		
	UnlockMutex;
	return reservationStatus;
}

int GetAvailableIpsfromRepository(unsigned int *ips)
{
	int i, ipCount = 0;

	LockMutex;
	if (repositoryIpAllocations)
	{
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

	return ipCount;
}

unsigned int GetReservedIpFromRepository(const char *reservationId)
{
	unsigned int returnIp = 0;
	char logMsg[2048];
#ifdef _WIN32
	size_t msgSize = 2048;
#endif
	LockMutex;

	if (repositoryIpAllocations)
	{
		sprintf_s(logMsg, msgSize, "Going to get an ip from repository for %s", reservationId );
		AquaLogging::Log(info, logMsg);
		IpStatus *repositoryReservedIpStatus = GetRepositoryReservedIp(reservationId);
		if (repositoryReservedIpStatus)
		{
			sprintf_s(logMsg, msgSize, "Successfully received an ip from repository for %s. IP = %u", reservationId, repositoryReservedIpStatus->ip);
			AquaLogging::Log(info, logMsg);
			returnIp = repositoryReservedIpStatus->ip;
		}
		else
		{
			sprintf_s(logMsg, msgSize, "Was unsucessful in obtaining an existing ip from repository for %s.", reservationId);
			AquaLogging::Log(info, logMsg);
		}
	}
	UnlockMutex;
	return returnIp;
}

unsigned long PersistSubnetInRepository(Ipv4Subnet *subnetToPersist, NetworkType networkType)
{
	LockMutex
	 
	#ifdef _WIN32
	size_t msgSize = 2048;
	#endif
	unsigned int mask = 0xFFFFFFFF;
	int leftShiftCount = 32 - subnetToPersist->prefixLength;
	mask = mask << leftShiftCount;
	sprintf_s(logMsg, msgSize, "mask=0x%08X\n", mask);
	AquaLogging::Log(info, logMsg);
	sprintf_s(logMsg, msgSize, "ipAddress=0x%08X\n", subnetToPersist->ipAddress);
	AquaLogging::Log(info, logMsg);
	
	unsigned int startingIp = (mask) & ntohl(subnetToPersist->ipAddress);

	sprintf_s(logMsg, msgSize, "starting ip=0x%08X\n", startingIp);
	unsigned int ipCount = 1;
	for( int i = 0; i<leftShiftCount-1 ;i++)
	{
		ipCount = (ipCount << 1);
		ipCount = ipCount | 0x1;
	}
	sprintf_s(logMsg, msgSize, "ipCount=%d\n", ipCount);
	unsigned int *ips = (unsigned int*)malloc(ipCount *(sizeof(unsigned int)));
	unsigned int i = 0;
	while(i < ipCount)
	{
		ips[i] = htonl(startingIp);
		sprintf_s(logMsg, msgSize, "Saving 0x%08X\n", ips[i]);
		AquaLogging::Log(verbose, logMsg);
		startingIp++;
		i++;
	}
	UpdateRepositoryIpListWithoutLock(ipCount, ips, networkType);
	free(ips);
	sprintf_s(logMsg, msgSize, "Updated ip repository with %d ips", ipCount);
	AquaLogging::Log(info, logMsg);
	UnlockMutex
	return 0;
}

DllExport unsigned int PersistedIpCountInRepository()
{
	int count = 0;
	LockMutex
	if (repositoryIpAllocations != NULL)
	{
		count = repositoryIpAllocations->count;
	}
	UnlockMutex
		return count;

}
} } } } }