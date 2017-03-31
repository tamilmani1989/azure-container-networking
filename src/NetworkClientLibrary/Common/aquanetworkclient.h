/****************************** Module Header ******************************\
* Module Name:  aquanetworkclient.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Header to be consumed by partners.
* Contains external contract.
*
\***************************************************************************/

#pragma once

#ifdef _WIN32
#include <windows.h>
#define DllExport   __declspec( dllexport )  
#endif

#ifdef __linux__
#include <pthread.h>
#define DllExport   
#define DWORD unsigned long
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
	// Target Environment
    enum Environment env;

	// Target Network Type
    enum NetworkType net;

	// Name of the network. Container will join network with this name.
	char *networkName;

	// Number of container hosts
    int nodeCount;

	// The overall subnet space from where smaller subnets for individual nodes are carved out
	// This is applicable to overlay network type only
	Ipv4Subnet *overlaySubnet; 

	//Config for individual nodes
    NodeConfig *nodeConfigs;       
}ClusterConfig;

// Returns the number of ip addresses that were returned
// If nodeNetworkResources is null, only the number of ip addresses is returned
// Otherwise, all the ip addresses (reserved + nonreserved) are populated
// int GetAllIpAddresses(NodeNetworkResources *nodeNetworkResources);

// Returns the number of ip addresses that were returned
// If nodeNetworkResources is null, the available number of ip addresses is returned
// Otherwise, the nonreserved ip addresses are populated
DllExport DWORD GetAvailableIpAddresses(NodeNetworkResources *nodeNetworkResources, int *count);

// This method selects an ip address from the available pool 
// of ip addresses and reserves it against the reservationId
// The reserved ip address is populated in out parameter ipAddress
// If no ip addresses are available to be returned, ErrorCode is returned
// This method is idempotent.
DllExport DWORD ReserveAvailableIpAddress(char *reservationId, unsigned int *ipAddress);

// Returns a single ip address
DllExport DWORD GetIpAddressForContainer(char *containerId, unsigned int ipAddress);

// Populates an ip address reserved correspoiding to the reservation id
// returns -1 if reservation id does not exist
DllExport DWORD GetReservedIpAddress(const char *reservationId, unsigned int *ipAddress);

// Releases reservation of an ip address if exists
// Return value 0 indicates that reservation was there and was released
// RESERVATION_NOT_FOUND indicates that reservation was not there, so nothing was released
DllExport DWORD ReleaseIpReservation(const char *reservationId);

// Initialize network for a specific environment and config
DllExport DWORD InitializeNetwork(ClusterConfig *clusterConfig);

// To retrieve the ip address used for communicating with host
DllExport DWORD GetHostLocalIp(unsigned int *ipAddress);

// This passes in a function pointer that can be used to log text string
// first parameter is the logging level and second is the text to log
DllExport DWORD SetupNetworkLogging(void(*log)(int, char*));

