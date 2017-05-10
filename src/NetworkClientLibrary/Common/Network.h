/*++

Copyright (c) Microsoft Corporation

Module Name:

Network.h

Abstract:

This file contains headers that can be used to setup container network.
This header is common for both Windows and Linux.

Author:

Sushant Sharma (susharm) 30-March-2017

--*/
#pragma once
#include "NclRepositoryManager.h"

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {

	DWORD SetupUnderlayInAzure(ClusterConfig *clusterConfig, NclRepositoryManager *nclRepositoryManager);
	DWORD SetupOverlay(ClusterConfig *clusterConfig);
	DWORD GetHostIp(unsigned int *ip);
	unsigned long CreateDockerNetwork(char *networkName, char ipamConfig[2048]);
	int GetIpsFromNmAgent(unsigned int *ips);
	#define IPAM_CONFIG_LEN 2048
	#define MAX_DOCKER_CONFIG_LEN 16384
} } } } }
