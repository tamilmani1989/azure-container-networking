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

DllExport typedef DWORD(*ReceiveMappingCallback) (const char *endpointId, const char *hostAddress, const char *containerAddress, const char *macAddress);

DllExport DWORD InitializeAquaNix(char *localHostAddress, char *remoteHostAddresses[], ReceiveMappingCallback receiveMapping, USHORT servicePort, USHORT remoteServicePort);

DllExport DWORD SendMapping(char *endpointId, char *containerAddress, char *macAddress);

DllExport DWORD SendMappingToLocalServer(char *endpointId, char *containerAddress, char *macAddress);
