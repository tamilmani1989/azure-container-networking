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

DllExport typedef DWORD(*ReceiveMappingCallback) (char *endpointId, char *hostAddress, char *containerAddress, char *macAddress);

DllExport DWORD InitializeAquaNix(char *localHostAddress, char *remoteHostAddresses[], ReceiveMappingCallback receiveMapping, USHORT servicePort);

DllExport DWORD SendMapping(char *endpointId, char *containerAddress, char *macAddress);

