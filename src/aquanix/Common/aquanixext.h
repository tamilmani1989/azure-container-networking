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

DllExport typedef DWORD(*GetLocalMappingsCallback) (
	EndpointMapping **endpointMappings,
	USHORT *mappingsCount);

DllExport typedef DWORD(*FreeEndpointMappingsCallback) (
	EndpointMapping *endpointMappings,
	USHORT mappingsCount);

DllExport DWORD InitializeAquaNix(
	char *localHostAddress,
	char *remoteHostAddresses[],
	USHORT numberOfRemoteHosts,
	char *mappingsSenderIP,
	AddRemoteMappingCallback receiveMapping,
	DeleteRemoteMappingCallback deleteMapping,
	GetLocalMappingsCallback getLocalMappings,
	FreeEndpointMappingsCallback freeEndpointMappings,
	USHORT servicePort,
	USHORT remoteServicePort);

DllExport DWORD SendLocalMapping(
	const char *endpointId, 
	const char *containerAddress,
	const char *macAddress);

DllExport DWORD DeleteLocalMapping(
	const char *endpointId);

DllExport DWORD RequestRemoteMappings();

DllExport DWORD SetRemoteHostAddresses(
	char *remoteHostAddresses[],
	USHORT numberOfRemoteHosts);
