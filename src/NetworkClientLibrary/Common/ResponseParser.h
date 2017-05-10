/*++

Copyright (c) Microsoft Corporation

Module Name:

ResponseParser.h

Abstract:

This cotains the headers that declare the methods to parse xml in windows.

This header is common for both Windows and Linux.

Author:

Sushant Sharma (susharm) 6-May-2017

--*/

#pragma once
#include "../Common/NclRepositoryManager.h"

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
	
unsigned int convertToIPAddress(unsigned char *ipAddressinStringFormat);

void GetSubnetGatewayPrimaryIp(char *xmlResponse, char **subnetIp, char **gatewayIp, NclRepositoryManager * nclRepositoryManager);

void CountOrRetrieveSecondaryIps(char *xmlResponse, int countOnly, unsigned int *count, unsigned int *ips);

DllExport int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);

DllExport void ParseSubnetGatewayPrimaryIp(char *xmlResponse, char **subnetIp, char **gatewayIp, NclRepositoryManager *nclRepositoryManager);
} } } } }