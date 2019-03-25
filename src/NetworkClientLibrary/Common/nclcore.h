/*++

Copyright (c) Microsoft Corporation

Module Name:

NclCore.c

Abstract:

This file contains internal headers for sdn client library that are not required by external partners.

Author:

Sushant Sharma (susharm) 24-March-2017

--*/
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
    unsigned int primaryIp;     
}IpAllocations;

DllExport unsigned long SaveOverlayAddressSpace(ClusterConfig *clusterConfig);
DllExport int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
DllExport void SetEnvironment(NetworkType networkType);
DllExport DWORD SetupNetworkEtwLogging(bool additionConsoleLoggingEnabled);

/*
 * Constants and macros.
 */
#define HostSuffix "GET /machine/plugins?comp=nmagent&type=getinterfaceinfov1 HTTP/1.1\r\nHost: 168.63.129.16\r\nConnection: close\r\n\r\n"
#define DockerNwCreatePost "POST /networks/create HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Type: application/json\r\nConnection: close\r\n"
#define DockerNwGet "GET /networks/"
#define HttpResponseHeaderSeparator "\r\n\r\n"
#define BUFFER_SIZE 1024

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