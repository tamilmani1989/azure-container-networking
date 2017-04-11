/****************************** Module Header ******************************\
* Module Name:  aquanixext.h
* Project:      Aquarius Network Interexchange
* Copyright (c) Microsoft Corporation.
*
* Header to be consumed by partners.
* Contains external contract.
*
\***************************************************************************/

#pragma once

#include "aquabase.h"

struct EndpointMapping
{
	char *endpointId;
	char *containerAddress;
	char *macAddress;
};

DllExport typedef DWORD(*AddRemoteMappingCallback) (
	const char *endpointId, 
	const char *hostAddress, 
	const char *containerAddress, 
	const char *macAddress);

DllExport typedef DWORD(*DeleteRemoteMappingCallback) (
	const char *endpointId);

/***********************
Memory is allocated by the caller and mappingsCount indicate number of elements endpointMappings argument can hold.
Initially it is set to 0 and callee is expected to return InsufficientMemory error code and fill mappingsCount to 
required number of elements.
In subsequent call, caller will allocate memory and is responsible to free the memory.
**********************/
DllExport typedef DWORD(*GetLocalMappingsCallback) (
	EndpointMapping *endpointMappings,
	USHORT *mappingsCount);

DllExport DWORD InitializeAquaNix(
	char *localHostAddress,
	char *remoteHostAddresses[], 
	AddRemoteMappingCallback receiveMapping,
	DeleteRemoteMappingCallback deleteMapping, 
	USHORT servicePort, 
	USHORT remoteServicePort);

DllExport DWORD SendLocalMapping(
	const char *endpointId, 
	const char *containerAddress,
	const char *macAddress);

DllExport DWORD DeleteLocalMapping(
	const char *endpointId);

DllExport DWORD RequestRemoteMappings();
