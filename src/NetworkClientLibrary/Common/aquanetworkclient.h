/****************************** Module Header ******************************\
* Module Name:  aquanetworkclient.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Header to be consumed by partners.
* Contains external contract.
*
\***************************************************************************/

#include <stdio.h>
#include <stdlib.h>
#ifdef __linux__
#include <pthread.h>
#define DllExport  
#endif
#ifdef _WIN32
#define DllExport   __declspec( dllexport )   
#endif

typedef struct NodeNetworkResources
{
    int ipCount; // number of ips
    unsigned int *ips;
}NodeNetworkResources;
 
typedef struct Ipv4Subnet
{
    unsigned int ipAddress;
    int prefixLength;
}Ipv4Subnet;
 
typedef struct NodeConfig
{
    unsigned int nodeIp;
    char *nodeId;
    Ipv4Subnet *subnet; // node's ip address space for overlay
}NodeConfig;
 
enum Environment
{
    Azure = 0,
    StandAlone = 1
};
 
enum NetworkType
{
    OverLay = 0,
    UnderLay = 1
};
 
typedef struct ClusterConfig
{
    enum Environment env;
    enum NetworkType net;
    int nodeCount;
    NodeConfig *nodeConfigs;       
}ClusterConfig;

// Returns the number of ip addresses that were returned
// If nodeNetworkResources is null, only the number of ip addresses is returned
// Otherwise, all the ip addresses (reserved + nonreserved) are populated
// int GetAllIpAddresses(NodeNetworkResources *nodeNetworkResources);

// Returns the number of ip addresses that were returned
// If nodeNetworkResources is null, the available number of ip addresses is returned
// Otherwise, the nonreserved ip addresses are populated
DllExport int GetAvailableIpAddresses(NodeNetworkResources *nodeNetworkResources);

// This method selects an ip address from the available pool 
// of ip addresses and reserves it against the reservationId
// The reserved ip address is returned
// If no ip addresses are available to be returned, -1 is returned
// This method is idempotent.
DllExport unsigned int ReserveAvailableIpAddress(char *reservationId);

// Returns a single ip address
DllExport unsigned int GetIpAddressForContainer(char *containerId);

// To initialize the ip address space on the node for overlay network
DllExport void InitializeOverlayAddressSpace(ClusterConfig *clusterConfig);

// Returns an ip address reserved correspoiding to the reservation id
// returns -1 if reservation id does not exist
DllExport unsigned int GetReservedIpAddress(char *reservationId);

// Releases reservation of an ip address if exists
DllExport void ReleaseIpReservation(char *reservationId);