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


#define UNSUPPORTED_NETWORK  1
#define INVALID_PARAMETER 2
#define UNSUPPORTED_ENVIRONMENT 3
#define UNREACHABLE_HOST 4
#define RESERVATION_NOT_FOUND 5
#define UNEXPECTED_ERROR 99
