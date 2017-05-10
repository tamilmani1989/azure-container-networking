/*++

Copyright (c) Microsoft Corporation

Module Name:

NetworkClientLibrary.h

Abstract:

This cotains the private headers used by external interface for clients to setup
networking for containers.
This header is common for both Windows and Linux.

Author:

Sushant Sharma (susharm) 6-May-2017

--*/

#pragma once
#include <stdlib.h>
#include "Common/aquanetworkclient.h"
#include "Common/nclcore.h"
#include "Common/NclRepositoryManager.h"

#ifdef _WIN32
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#include <stdio.h>
#include <iostream>
#include <mutex>
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>
#include <Mstcpip.h>
#include "Windows/VfpOperations.h"
#include "../aquanix/Common/aquanixext.h"
std::mutex repositoryVariableMutex;
#define LockIpRepositoryVariable repositoryVariableMutex.lock();
#define UnLockIpRepositoryVariable repositoryVariableMutex.unlock();
#endif

#ifdef __linux__
#include <curl/curl.h>
#include <string.h>
#include "../Logging/Logging.h"
#define UNREFERENCED_PARAMETER(param) do { (void)(param); } while (0)
pthread_mutex_t repositoryVariableMutex = PTHREAD_MUTEX_INITIALIZER;
#define LockIpRepositoryVariable pthread_mutex_lock(&repositoryVariableMutex);
#define UnLockIpRepositoryVariable pthread_mutex_unlock(&repositoryVariableMutex);
#endif

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
	
extern DWORD SetupUnderlayInAzure(ClusterConfig *clusterConfig, NclRepositoryManager *nclRepositoryManager);
extern DWORD SetupOverlay(ClusterConfig *clusterConfig);
extern int GetIpsFromNmAgent(unsigned int *ips);
extern int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
extern DWORD ProvideConfigToNIX(ClusterConfig *clusterConfig);
extern DWORD GetHostIp(unsigned int *ip, NclRepositoryManager *nclRepositoryManager);
extern int WsaInitialize();
NetworkType networkTypeProvided = UnderLay;
NclRepositoryManager *nclRepositoryManager = NULL;
DllExport NclRepositoryManager *GetIpRepository();

} } } } }
