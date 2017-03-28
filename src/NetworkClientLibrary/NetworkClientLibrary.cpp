/****************************** Module Header ******************************\
* Module Name:  NetworkClientLibrary.c
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* This is the SDN Library that can be used to setup 
* networking for containers in Azure and OnPrem
*
\***************************************************************************/

#include "Common/aquanetworkclient.h"
#include <stdlib.h>
#include <string.h>
#include "Common/nclcore.h"
#ifdef _WIN32
#include <WinSock2.h>
#include <stdio.h>
#include <io.h>
#include <iostream>
#define HostSuffix "GET /machine/plugins?comp=nmagent&type=getinterfaceinfov1 HTTP/1.1\r\nHost: 169.254.169.254\r\nConnection: close\r\n\r\n"
#define HttpResponseHeaderSeparator "\r\n\r\n"
#define BUFFER_SIZE 1024
std::_Mutex initWsaMutex;
int wsaInitialized = 0;
#endif
#ifdef __linux__
#include <curl/curl.h>
#endif


extern int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
extern void UpdateRepositoryIpList(int ipCount, unsigned int *ips);
extern unsigned int ReserveRepositoryIpAddress(char *reservationId);
extern void ReleaseRepositoryIpAddress(char *reservationId);
extern int GetAvailableIpsfromRepository(unsigned int *ips);
extern unsigned int GetReservedIpFromRepository(char *reservationId);

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
#endif

#ifdef _WIN32

int initWsa()
{
	WORD wVersionRequested;
	WSADATA wsaData;
	int err;
	int success = 1;

	wVersionRequested = MAKEWORD(2, 2);

	err = WSAStartup(wVersionRequested, &wsaData);
	if (err != 0)
	{
		// could not find a usable Winsock DLL.  
		printf("WSAStartup failed with error: %d\n", err);
		success = 0;
	}

	/* If the DLL supports versions greater
	* than 2.2 in addition to 2.2, it will still return
	* 2.2 in wVersion since that is the version we
	* requested.
	* */

	if (LOBYTE(wsaData.wVersion) != 2 || HIBYTE(wsaData.wVersion) != 2)
	{
		fprintf(stderr, "Could not find a usable version of Winsock.dll\n");
		WSACleanup();
		success = 0;
	}
	else
	{
		printf("The Winsock 2.2 dll was found okay\n");
	}

	return success;
}

SOCKET connectToHost(char *host, USHORT port)
{
	struct hostent *hp;
	struct sockaddr_in addr;
	SOCKET sock;	
	
	if ((hp = gethostbyname(host)) == nullptr)
	{
		fprintf(stderr, "gethostbyname failed %d", WSAGetLastError());
		return INVALID_SOCKET;
	}

	memcpy(&addr.sin_addr, hp->h_addr, hp->h_length);
	addr.sin_port = htons(port);
	addr.sin_family = AF_INET;
	sock = socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);

	if (sock == INVALID_SOCKET)
	{
		perror("setsockopt");
		return INVALID_SOCKET;
	}
	
	if (connect(sock, (struct sockaddr *)&addr, sizeof(struct sockaddr)) == -1)
	{
		int error = WSAGetLastError();
		if (error == WSAENETUNREACH)
		{
			fprintf(stderr, "Connection to host failed.\n");
		}
		else
		{
			fprintf(stderr, "Connection to host failed with error code %d", error);
		}
		return INVALID_SOCKET;
	}

	printf("Connected successfully to host.");
	return sock;
}


int InsertIntoBuffer(char **recvBuffer, int byteCount, char c)
{
	char *newBuffer = (char *)realloc((void *) (*recvBuffer), byteCount);
	if(newBuffer == nullptr)
	{
		return 0;
	}
	else
	{
		*recvBuffer = newBuffer;
	}
	(*recvBuffer)[byteCount - 1] = c;
	return 1;
}

int wsaInitialize()
{
	if (!wsaInitialized)
	{
		initWsaMutex._Lock();
		if (!wsaInitialized)
		{
			if (initWsa())
			{
				wsaInitialized = 1;
				initWsaMutex._Unlock();
			}
			else
			{
				fprintf(stderr, "wsa initialization failed\n");
				initWsaMutex._Unlock();
				return 0;
			}
		}
	}
	return 1;
}

void wsaCleanUp()
{
	if (wsaInitialized)
	{
		initWsaMutex._Lock();
		if (wsaInitialized)
		{
			WSACleanup();
			wsaInitialized = 0;
		}
		initWsaMutex._Unlock();
	}
}


int getIpsFromNmAgent(unsigned int *ips)
{
	int ipCount = 0;
	char buffer[BUFFER_SIZE];
	int bytesRead;
	char *recvBuffer = nullptr;
	char *responseLocation = nullptr;
	int byteCount = 0;
	
	if(!wsaInitialize())
	{
		return -1;
	}

	printf("Trying to connect to nm agent.\n");
	SOCKET sock = connectToHost("169.254.169.254", 80);

	if (sock == INVALID_SOCKET)
	{
		fprintf(stderr, "Connect to host failed.\n");
		return -1;
	}
	printf("Now going to get ips from nm agent... fd: %llu\n", sock);
	int bytesWritten = send(sock, HostSuffix, strlen(HostSuffix), 0);
	if(bytesWritten == SOCKET_ERROR)
	{
		fprintf(stderr, "Writing to socket failed errno=%d\n", errno );
	}
	else 
	{
		printf("Sent %d bytes on the connection.\nNow ", bytesWritten);
		//bzero(buffer, BUFFER_SIZE);
		memset((buffer), '\0', (BUFFER_SIZE));
		printf("Reading into buffer..\n");
		do
		{
			memset((buffer), '\0', (BUFFER_SIZE));
			char c;
			bytesRead = recv(sock, &c, 1, 0);
			if (bytesRead == 1)
			{
				byteCount++;
				if(!InsertIntoBuffer(&recvBuffer, byteCount, c))
				{
					fprintf(stderr, "Failed to insert a char into buffer");
					ipCount = -1;
					goto cleanup;
				}
			}
		} while (bytesRead > 0);

		if (bytesRead == -1)
		{
			fprintf(stderr, "Reading failed. errno=%d \n", errno);
		}
		else if (bytesRead == 0)
		{
			if(!InsertIntoBuffer(&recvBuffer, byteCount+1, '\0'))
			{
				fprintf(stderr, "Failed to insert EOL into buffer");
				ipCount = -1;
				goto cleanup;
			}

			printf("\nSuccessfully finished reading. Printing response below..\n");			
			responseLocation = strstr(recvBuffer, HttpResponseHeaderSeparator);
			responseLocation += strlen(HttpResponseHeaderSeparator);
			printf(responseLocation);			
		}
		if (responseLocation != nullptr)
		{
			if (ips == NULL)
			{
				ipCount = ParseSecondaryIpAddresses(responseLocation, 1, ips);
				printf("Received only ip count as %d\n", ipCount);
			}
			else
			{
				ipCount = ParseSecondaryIpAddresses(responseLocation, 0, ips);
				printf("Received ips with ip count as %d\n", ipCount);
			}
		}
		else
		{
			fprintf(stderr, "Unable to find portion of http response header that contains valid nm agent response response %s", recvBuffer);
		}
		free(recvBuffer);
		recvBuffer = nullptr;
	}
	
	cleanup:
	if (recvBuffer)
	{
		free(recvBuffer);
		recvBuffer = nullptr;
	}
	shutdown(sock, SD_BOTH);
	_close((int)sock);

	wsaCleanUp();
	return ipCount;
}

#endif

// Returns the number of ip addresses that were returned
// If nodeNetworkResources is null, only the number of ip addresses is returned
// Otherwise, all the ip addresses (reserved + nonreserved) are populated
int GetAllIpAddresses(NodeNetworkResources *nodeNetworkResources)
{
	int ipCount = 0;

	ipCount = getIpsFromNmAgent(nodeNetworkResources->ips);
	
	if (nodeNetworkResources->ips)
	{
		UpdateRepositoryIpList(ipCount, nodeNetworkResources->ips);
	}
	return ipCount;
}

// Returns the number of ip addresses that were returned
// If nodeNetworkResources->ips is null, the available number of ip addresses is returned
// Otherwise, the nonreserved ip addresses are populated
DllExport int GetAvailableIpAddresses(NodeNetworkResources *nodeNetworkResources)
{
	int ipCount = 0;
	unsigned int *ips;
	ipCount = getIpsFromNmAgent(NULL);

	if (ipCount <= 0)
	{
		ipCount = 0;
		return ipCount;
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
	
	return ipCount;
}

// This method selects an ip address from the available pool 
// of ip addresses and reserves it against the reservationId
// The reserved ip address is returned
// If no ip addresses are available to be returned, 0 is returned
// This method is idempotent.
DllExport unsigned int ReserveAvailableIpAddress(char *reservationId)
{
	if (GetAvailableIpAddresses(NULL) > 0)
	{
		return ReserveRepositoryIpAddress(reservationId);
	}

	return 0;	
}

// Returns a single ip address
DllExport unsigned int GetIpAddressForContainer(char *containerId)
{
	return 0;
}

// To initialize the ip address space on the node for overlay network
DllExport void InitializeOverlayAddressSpace(ClusterConfig *clusterConfig)
{
	return;
}

// Returns an ip address reserved correspoiding to the reservation id
// returns -1 if reservation id does not exist
DllExport unsigned int GetReservedIpAddress(char *reservationId)
{
	return GetReservedIpFromRepository(reservationId);
}

// Releases reservation of an ip address if exists
DllExport void ReleaseIpReservation(char *reservationId)
{
	ReleaseRepositoryIpAddress(reservationId);
}
