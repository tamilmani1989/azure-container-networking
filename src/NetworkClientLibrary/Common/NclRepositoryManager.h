/*++

Copyright (c) Microsoft Corporation

Module Name:

NclRepositoryManager.h

Abstract:

Headers to manage the state of network client library.

Author:

Sushant Sharma (susharm) 6-May-2017

--*/

#pragma once
#include "nclcore.h"
#include <string.h>
#include <iostream>
#include "../../Logging/Logging.h"

#ifdef _WIN32
#include <windows.h>
#include <mutex>
#include <WinSock2.h>
#define LockMutex ipAllocationsMutex.lock();
#define UnlockMutex ipAllocationsMutex.unlock();
#define bytesallocated _msize
#endif

#ifdef __linux__
#include <stdio.h>
#include <malloc.h>
#include <arpa/inet.h>
#define LockMutex pthread_mutex_lock(&ipAllocationsLock);
#define UnlockMutex pthread_mutex_unlock(&ipAllocationsLock);
#define strcpy_s(dst, count, src) strncpy(dst, src, count)
#define bytesallocated malloc_usable_size
#endif

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {

	class NclRepositoryManager {
#ifdef _WIN32
	std::mutex ipAllocationsMutex;
#endif
#ifdef __linux__
	pthread_mutex_t ipAllocationsLock = PTHREAD_MUTEX_INITIALIZER;
#endif
	public:
		NclRepositoryManager(IpAllocations* repository_ip_allocations)			
		{
			if(repositoryIpAllocations)
			{
				return;
			}
			LockMutex
			if(repositoryIpAllocations)
			{
				return;
			}		
			// NULL is fine
			repositoryIpAllocations = repository_ip_allocations;
			UnlockMutex
		}
		~NclRepositoryManager()
		{
			if (repositoryIpAllocations)
			{
				free(repositoryIpAllocations);
				repositoryIpAllocations = NULL;
			}
		}

	private:
		DWORD updateRepositoryIpListUnTransacted(int ipCount, unsigned* ips, NetworkType networkType);
		IpStatus *getRepositoryReservedIpUnTransacted(const char *reservationId);
		unsigned long releaseReservedRepositoryIpUnTransacted(const char *reservationId);
		IpAllocations *repositoryIpAllocations=NULL;

	public:	
		DllExport unsigned long InitRepositoryIpAllocations();	
		DllExport unsigned int GetRepositoryPrimaryIp();
		DllExport void SetRepositoryPrimaryIp(unsigned int ip);
		DllExport int FreeStatusInRepositoryIpAllocationsUnTransacted();
		DllExport void UpdateRepositoryIpList(int ipCount, unsigned int *ips, NetworkType networkType);
		DllExport bool IpRepositoryIsEmpty();
		DllExport unsigned int ReserveRepositoryIpAddress(char *reservationId);
		DllExport unsigned int PersistedIpCountInRepository();
		DllExport int GetAvailableIpsfromRepository(unsigned int *ips);
		unsigned int GetReservedIpFromRepository(const char *reservationId);
		IpStatus *ReserveRepositoryIp(char *reservationId);
		IpStatus *GetRepositoryIpAllocationStatus(unsigned int ip);
		unsigned long ReleaseRepositoryIpAddress(const char *reservationId);
		unsigned long PersistSubnetInRepository(Ipv4Subnet *subnetToPersist, NetworkType networkType);
	};
} } } } }