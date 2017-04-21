/****************************** Module Header ******************************\
* Module Name:  nclcore.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Internal header for sdn client library
*
\***************************************************************************/
#pragma once
#include "aquanetworkclient.h"
namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
typedef struct IpStatus
{
    unsigned int ip;
    int isReserved;
    char *id;
}IpStatus;

typedef struct IpAllocations
{
    int count;
    IpStatus *statusForAllIpAddresses;     
}IpAllocations;

DllExport unsigned int PersistedIpCountInRepository();
DllExport unsigned long saveOverlayAddressSpace(ClusterConfig *clusterConfig);
DllExport int GetAvailableIpsfromRepository(unsigned int *ips);
DllExport int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
DllExport void UpdateRepositoryIpList(int ipCount, unsigned int *ips, NetworkType networkType);
DllExport unsigned int ReserveRepositoryIpAddress(char *reservationId);

#define UNSUPPORTED_NETWORK  1
#define INVALID_PARAMETER 2
#define UNSUPPORTED_ENVIRONMENT 3
#define UNREACHABLE_HOST 4
#define RESERVATION_NOT_FOUND 5
#define OVERLAYPLUGIN_INSTALL_FAILED 6
#define OVERLAY_SUBNET_NOT_SPECIFIED 7
#define MALFORMED_SUBNET_IP 8
#define UNREACHABLE_DOCKER_DAEMON 9
#define UNSPECIFIED_NETWORK_NAME 10
#define SUBNET_SIZE_TOO_SMALL 11
#define ERROR_STARTING_VFP 12
#define UNEXPECTED_ERROR 99

} } } } }