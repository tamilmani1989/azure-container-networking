/****************************** Module Header ******************************\
* Module Name:  nclcore.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Internal header for sdn client library
*
\***************************************************************************/
#pragma once
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
