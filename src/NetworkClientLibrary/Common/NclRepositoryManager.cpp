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
#include <WinSock2.h>

#ifdef _WIN32
#include <windows.h>
#include <mutex>
std::mutex ipAllocationsMutex;
#define LockMutex ipAllocationsMutex.lock();
#define UnlockMutex ipAllocationsMutex.unlock();
#endif

#ifdef __linux__
#include <stdio.h>
pthread_mutex_t ipAllocationsLock = PTHREAD_MUTEX_INITIALIZER;
#define LockMutex pthread_mutex_lock(&ipAllocationsLock);
#define UnlockMutex pthread_mutex_unlock(&ipAllocationsLock);
#define strcpy_s(dst, count, src) strncpy(dst, src, count)
#endif

IpAllocations *repositoryIpAllocations;

void FreeStatusInRepositoryIpAllocations(IpAllocations *myRepositoryIpAllocations)
{
	int i;
	if (myRepositoryIpAllocations)
	{
		for (i = 0; i<myRepositoryIpAllocations->count; i++)
		{
			IpStatus tempStatus = myRepositoryIpAllocations->statusForAllIpAddresses[i];
			free(tempStatus.id);
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
	if (repositoryIpAllocations)
	{
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			IpStatus *retval = &repositoryIpAllocations->statusForAllIpAddresses[i];
			if (retval->id)
			{
				if (strcmp(reservationId, retval->id) == 0)
				{
					return retval;
				}
			}
		}
	}
	return NULL;
}

IpStatus *ReserveRepositoryIp(char *reservationId)
{
	int i;
	if (repositoryIpAllocations)
	{
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			IpStatus *retval = &repositoryIpAllocations->statusForAllIpAddresses[i];
			if (!retval->id)
			{
				size_t resSize = strlen(reservationId) * sizeof(char);
				retval->id = (char *)malloc(resSize + 1);
				strcpy_s(retval->id, resSize + 1, reservationId);
				retval->isReserved = 1;
				return retval;
			}
		}
	}
	return NULL;
}

unsigned long ReleaseReservedRepositoryIp(const char *reservationId)
{
	int i;
	unsigned long reservationFound = 0;
	if (repositoryIpAllocations)
	{
		printf("Going to try to release %s\n", reservationId);
		for (i = 0; i<repositoryIpAllocations->count; i++)
		{
			IpStatus *retval = &repositoryIpAllocations->statusForAllIpAddresses[i];
			printf("Checking if reservation for %u should be released\n", retval->ip);
			if (retval->isReserved == 1)
			{
				if (strcmp(retval->id, reservationId) == 0) 
				{
					printf("yes, going to release reservation\n");
					free(repositoryIpAllocations->statusForAllIpAddresses[i].id);
					repositoryIpAllocations->statusForAllIpAddresses[i].id = NULL;
					repositoryIpAllocations->statusForAllIpAddresses[i].isReserved = 0;
					reservationFound = 1;
					break;
				}
				printf("no, not the target\n");
			}
			else
			{
				printf("no, not reserved)\n");
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

void UpdateRepositoryIpListWithoutLock(int ipCount, unsigned* ips)
{
	fprintf(stdout, "Inside repository lock ipCount=%d\n", ipCount);
	int i;
	// non performant but okay for now, since this will move to ipam anyways
	IpStatus *localIpStatus;

	if (ipCount < 0)
	{
		fprintf(stderr, "Error: Received %d ips to upate in repository. Will not do anything.", ipCount);
		return;
	}
	localIpStatus = (IpStatus *)malloc(ipCount * sizeof(IpStatus));
	fprintf(stdout, "Allocated memory for ip status \n");
	for (i = 0; i<ipCount; i++)
	{
		localIpStatus[i].ip = ips[i];
		printf("Going to get status for ip %u\n", ips[i]);
		IpStatus *repositoryIpStatus = GetRepositoryIpAllocationStatus(ips[i]);
		printf("Successfully retrieved status for ip %u\n", ips[i]);
		if (repositoryIpStatus)
		{
			printf("Received ip from repository %u isReserved=%d\n",
			       repositoryIpStatus->ip, repositoryIpStatus->isReserved);
			localIpStatus[i].isReserved = repositoryIpStatus->isReserved;
			if (repositoryIpStatus->id != NULL)
			{
				printf("id= %s\n", repositoryIpStatus->id);
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
		if(i==0 || i==1)
		{
			localIpStatus[i].isReserved = 1;
			localIpStatus[i].id = "Sys_Reserved";
		}
	}

	printf("Going to free current ip status from repository\n");
	FreeStatusInRepositoryIpAllocations(repositoryIpAllocations);
	printf("Going to update new status for all ip addresses\n");
	if (!repositoryIpAllocations)
	{
		repositoryIpAllocations = (IpAllocations *)malloc(sizeof(IpAllocations));
	}
	repositoryIpAllocations->statusForAllIpAddresses = localIpStatus;
	repositoryIpAllocations->count = ipCount;
}

void UpdateRepositoryIpList(int ipCount, unsigned int *ips)
{
	LockMutex
	UpdateRepositoryIpListWithoutLock(ipCount, ips);
	UnlockMutex
}

unsigned int ReserveRepositoryIpAddress(char *reservationId)
{
	unsigned int returnIp = 0;
	LockMutex;

	if (repositoryIpAllocations)
	{
		IpStatus *repositoryReservedIpStatus = GetRepositoryReservedIp(reservationId);
		if (repositoryReservedIpStatus)
		{
			returnIp = repositoryReservedIpStatus->ip;
		}
		else
		{
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
			if (!repositoryIpAllocations->statusForAllIpAddresses[i].id)
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
	LockMutex;

	if (repositoryIpAllocations)
	{
		IpStatus *repositoryReservedIpStatus = GetRepositoryReservedIp(reservationId);
		if (repositoryReservedIpStatus)
		{
			returnIp = repositoryReservedIpStatus->ip;
		}
	}
	UnlockMutex;
	return returnIp;
}

unsigned long PersistSubnetInRepository(Ipv4Subnet *subnetToPersist)
{
	LockMutex
	unsigned int mask = 0xFFFFFFFF;
	int leftShiftCount = 32 - subnetToPersist->prefixLength;
	mask = mask << leftShiftCount;
	printf("mask=0x%08X\n", mask);
	printf("ipAddress=0x%08X\n", subnetToPersist->ipAddress);
	
	unsigned int startingIp = (mask) & ntohl(subnetToPersist->ipAddress);

	printf("starting ip=0x%08X\n", startingIp);
	unsigned int ipCount = 1;
	for( int i = 0; i<leftShiftCount ;i++)
	{
		ipCount = (ipCount << 1);
		ipCount = ipCount | 0x1;
	}
	printf("ipCount=%d\n", ipCount);
	unsigned int *ips = (unsigned int*)malloc(ipCount *(sizeof(unsigned int)));
	unsigned int i = 0;
	while(i < ipCount)
	{
		ips[i] = htonl(startingIp);
		printf("Saving 0x%08X\n", ips[i]);
		startingIp++;
		i++;
	}
	UpdateRepositoryIpListWithoutLock(ipCount, ips);
	free(ips);
	printf("Updated ip repository with %d ips", ipCount);
	UnlockMutex
	return 0;
}

unsigned int PersistedIpCountInRepository()
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