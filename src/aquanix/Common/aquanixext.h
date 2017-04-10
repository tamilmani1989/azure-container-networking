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

DllExport typedef DWORD(*ReceiveMapping) (char *hostAddress, char *containerAddress, char *macAddress);

DllExport DWORD InitializeAquaNix(char *localHostAddress, char *remoteHostAddresses[], ReceiveMapping receiveMapping, USHORT servicePort);

DllExport DWORD SendMapping(char *containerAddress, char *macAddress);

