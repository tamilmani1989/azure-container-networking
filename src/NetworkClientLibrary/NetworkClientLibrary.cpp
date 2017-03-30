/****************************** Module Header ******************************\
* Module Name:  NetworkClientLibrary.c
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* This is the SDN Library that can be used to setup 
* networking for containers in Azure and OnPrem
*
\***************************************************************************/

#define _WINSOCK_DEPRECATED_NO_WARNINGS
#include "Common/aquanetworkclient.h"
#include <stdlib.h>
#include "Common/nclcore.h"
#ifdef _WIN32
#include <stdio.h>
#include <iostream>
#include <mutex>
#endif
#ifdef __linux__
#include <curl/curl.h>
#endif

extern DWORD setupUnderlayInAzure(ClusterConfig *clusterConfig);
extern int getIpsFromNmAgent(unsigned int *ips);
extern int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
extern void UpdateRepositoryIpList(int ipCount, unsigned int *ips);
extern unsigned int ReserveRepositoryIpAddress(char *reservationId);
extern unsigned long ReleaseRepositoryIpAddress(const char *reservationId);
extern int GetAvailableIpsfromRepository(unsigned int *ips);
extern unsigned int GetReservedIpFromRepository(const char *reservationId);
extern void ParseSubnetAndGatewayIp(char *xmlResponse, char **sunbetIp, char **gatewayIp);

#ifdef __linux__
typedef struct string
{
	char *ptr;
	size_t len;
}string;


void initString(string *s)
{
	s->len = 0;
	s->ptr = (char *)malloc(s->len + 1);
	if (s->ptr == NULL)
	{
		fprintf(stderr, "malloc() failed\n");
		exit(EXIT_FAILURE);
	}
	s->ptr[0] = '\0';
}

void freeStringMemory(string *s)
{
	if (s->len>0)
	{
		free(s->ptr);
	}
}

size_t saveData(void *source, size_t size, size_t nmemb, string *target)
{
	size_t newLength = target->len + size*nmemb;
	target->ptr = (char*)realloc(target->ptr, newLength + 1);
	if (target->ptr == NULL)
	{
		fprintf(stderr, "realloc() failed\n");
		exit(EXIT_FAILURE);
	}
	memcpy(target->ptr + target->len, source, size*nmemb);
	target->ptr[newLength] = '\0';
	target->len = newLength;

	return size*nmemb;
}

int getIpsFromNmAgent(unsigned int *ips)
{
	CURL *handle;
	CURLcode res;
	handle = curl_easy_init();
	int ipCount = 0;

	printf("Going to get ips from nm agent\n");

	if (handle)
	{
		curl_easy_setopt(handle, CURLOPT_URL, "http://169.254.169.254/machine/plugins?comp=nmagent&type=getinterfaceinfov1");
		curl_easy_setopt(handle, CURLOPT_WRITEFUNCTION, saveData);
		string s;
		initString(&s);
		curl_easy_setopt(handle, CURLOPT_WRITEDATA, &s);
		res = curl_easy_perform(handle);
		if (res != CURLE_OK)
		{
			fprintf(stderr, "curl_easy_perform() failed %d\n", res);
			return ipCount;
		}
		curl_easy_cleanup(handle);
		if (ips == NULL)
		{
			ipCount = ParseSecondaryIpAddresses(s.ptr, 1, ips);
			printf("Received only ip count as %d\n", ipCount);
		}
		else
		{
			ipCount = ParseSecondaryIpAddresses(s.ptr, 0, ips);
			printf("Received ips with ip count as %d\n", ipCount);
		}
		freeStringMemory(&s);
	}

	return ipCount;
}

DWORD setupUnderlayInAzure(ClusterConfig *clusterConfig)
{
	return UNSUPPORTED_ENVIRONMENT;
}
#endif

// Returns the number of ip addresses that were returned
// If nodeNetworkResources is null, only the number of ip addresses is returned
// Otherwise, all the ip addresses (reserved + nonreserved) are populated
int GetAllIpAddresses(NodeNetworkResources *nodeNetworkResources)
{
	int ipCount = getIpsFromNmAgent(nodeNetworkResources->ips);
	
	if (nodeNetworkResources->ips)
	{
		UpdateRepositoryIpList(ipCount, nodeNetworkResources->ips);
	}
	return ipCount;
}

// Returns the number of ip addresses that were returned
// If nodeNetworkResources->ips is null, the available number of ip addresses is returned
// Otherwise, the nonreserved ip addresses are populated
DllExport DWORD GetAvailableIpAddresses(NodeNetworkResources *nodeNetworkResources, int *count)
{
	int ipCount;
	unsigned int *ips;
	ipCount = getIpsFromNmAgent(NULL);

	if (ipCount <= 0)
	{
		return UNEXPECTED_ERROR;
	}

	ips = (unsigned int *)malloc(ipCount * sizeof(unsigned int));
	ipCount = getIpsFromNmAgent(ips);
	printf("Retrieved %d ips fron nm agent along with details\n", ipCount);
	
	for(int i = 0; i<ipCount; i++)
	{
		printf("%d\n", ips[i]);
	}
	if(ips)
	{
		UpdateRepositoryIpList(ipCount, ips);
		free(ips);
	}
	if (nodeNetworkResources) 
	{
		nodeNetworkResources->ipCount = ipCount;
		ipCount = GetAvailableIpsfromRepository(nodeNetworkResources->ips);
	}
	
	*count = ipCount;
	return 0;
}

// This method selects an ip address from the available pool 
// of ip addresses and reserves it against the reservationId
// The reserved ip address is returned
// If no ip addresses are available to be returned, 0 is returned
// This method is idempotent.
DllExport DWORD ReserveAvailableIpAddress(char *reservationId, unsigned int *ipAddress)
{
	unsigned int reservedIp = 0;
	int count = 0;
	GetAvailableIpAddresses(NULL, &count);
	if ( count > 0)
	{
		 reservedIp = ReserveRepositoryIpAddress(reservationId);
	}

	if(reservedIp == 0)
	{
		return UNEXPECTED_ERROR;
	}

	*ipAddress = reservedIp;
	return 0;	
}

// Returns a single ip address
DllExport  DWORD GetIpAddressForContainer(char *containerId, unsigned int ipAddress)
{
	// Reivew: UNREFERENCED_PARAMETER works in linux?
	UNREFERENCED_PARAMETER(containerId);
	UNREFERENCED_PARAMETER(ipAddress);
	return 0;
}

// To initialize the ip address space on the node for overlay network
DllExport void InitializeOverlayAddressSpace(ClusterConfig *clusterConfig)
{
	UNREFERENCED_PARAMETER(clusterConfig);
	return;
}

// Returns an ip address reserved correspoiding to the reservation id
// returns -1 if reservation id does not exist
DllExport DWORD GetReservedIpAddress(const char *reservationId, unsigned int *ipAddress)
{
	unsigned int reservedIp =  GetReservedIpFromRepository(reservationId);
	if(reservedIp == 0)
	{
		return UNEXPECTED_ERROR;
	}

	*ipAddress = reservedIp;
	return 0;
}

// Releases reservation of an ip address if exists
DllExport DWORD ReleaseIpReservation(const char *reservationId)
{
	if(!ReleaseRepositoryIpAddress(reservationId))
	{
		return RESERVATION_NOT_FOUND;
	}

	return 0;
}

DllExport DWORD InitializeNetwork(ClusterConfig *clusterConfig)
{
	if (!clusterConfig)
	{

		return INVALID_PARAMETER;
	}

	printf("Going to initialize network ..\n");

	switch(clusterConfig->net)
	{
		case OverLay: 
			printf("OverLay Network currently not supported\n");
			return UNSUPPORTED_NETWORK;

		case UnderLay: 
			printf("Going to setup underlay network");

			switch(clusterConfig->env)
			{
			case Azure: 
				return setupUnderlayInAzure(clusterConfig);

			case StandAlone: 		
				printf("Underlay in StandAlone is not a supportedconfig");

			default:
				printf("Unsupported environment %d\n", clusterConfig->env);
				return UNSUPPORTED_NETWORK;
			}
		default:
			printf("Unknown network type %d\n", clusterConfig->net);
			return UNSUPPORTED_NETWORK;
	}
}