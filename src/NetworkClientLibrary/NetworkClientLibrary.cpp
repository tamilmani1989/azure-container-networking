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
#include "Common/aquaerrorcodes.h"
#include <stdlib.h>
#include <string.h>
#include "Common/nclcore.h"
#ifdef _WIN32
#include <WinSock2.h>
#include <stdio.h>
#include <io.h>
#include <iostream>
#define HostSuffix "GET /machine/plugins?comp=nmagent&type=getinterfaceinfov1 HTTP/1.1\r\nHost: 169.254.169.254\r\nConnection: close\r\n\r\n"
#define DockerNwCreatePost "POST /networks/create HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Type: application/json\r\nConnection: close\r\n"
#define DockerNwGet "GET /networks/"
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
		fprintf(stderr, "gethostbyname failed %d : %s", WSAGetLastError(),host);
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

char* httpSendResponse(SOCKET sock, char *payload)
{

	int bytesRead;
	char *recvBuffer = nullptr;
	char *responseLocation = nullptr;
	int byteCount = 0;

	int bytesWritten = send(sock, payload, strlen(payload), 0);
	if (bytesWritten == SOCKET_ERROR)
	{
		fprintf(stderr, "Writing to socket failed errno=%d\n", errno);
	}
	else
	{
		printf("Sent %d bytes on the connection.\nNow ", bytesWritten);
		do
		{
			char c;
			bytesRead = recv(sock, &c, 1, 0);
			if (bytesRead == 1)
			{
				byteCount++;
				if (!InsertIntoBuffer(&recvBuffer, byteCount, c))
				{
					fprintf(stderr, "Failed to insert a char into buffer");
					return NULL;
				}
			}
		} while (bytesRead > 0);

		if (bytesRead == -1)
		{
			fprintf(stderr, "Reading failed. errno=%d \n", errno);
		}
		else if (bytesRead == 0)
		{
			printf("Read finished\n");
			if (!InsertIntoBuffer(&recvBuffer, byteCount + 1, '\0'))
			{
				fprintf(stderr, "Failed to insert EOL into buffer");
				return NULL;
			}
			return recvBuffer;
		}
	}
	return NULL;
}

int retrieveHttpStatus(char *recvBuffer)
{
	int i = 0;
	int k = 0;
	char status[8];
	int ret = 0;
	while (recvBuffer[i] != ' ')
	{
		i++;
	}
	
	i++;

	while (recvBuffer[i] != ' ')
	{
		status[k] = recvBuffer[i];
		k++;
		i++;
	}
	status[k] = '\0';
	if (k > 0)
	{
		ret = atoi(status);
	}
	return ret;
}
int getSubnetGwIpfromNMAgent(char **subnetIp, char **gwIp)
{
	char *recvBuffer = nullptr;
	char *responseLocation = nullptr;

	if (!wsaInitialize())
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
	
	recvBuffer = httpSendResponse(sock, HostSuffix);
	if (!recvBuffer) {
		printf("Error while retrieving xml file from NMAgent");
		goto cleanup_subgw;
	}
	//printf("xml recv buf:%s\n", recvBuffer);

	int ret = retrieveHttpStatus(recvBuffer);
	printf("status:%d\n", ret);
	if (ret != 200)
	{
		printf("Error while retrieving xml file from NMAgent. Invalid Status code:%s\n", recvBuffer);
		goto cleanup_subgw;
	}
	
	responseLocation = strstr(recvBuffer, HttpResponseHeaderSeparator);
	responseLocation += strlen(HttpResponseHeaderSeparator);
	printf(responseLocation);
		
	if (responseLocation != nullptr)
	{
		ParseSubnetAndGatewayIp(responseLocation, subnetIp, gwIp);
	}
	else
	{
		fprintf(stderr, "Unable to find portion of http response header that contains valid nm agent response response %s", recvBuffer);
	}

cleanup_subgw:
	if (recvBuffer)
	{
		free(recvBuffer);
		recvBuffer = nullptr;
	}
	shutdown(sock, SD_BOTH);
	_close((int)sock);
	wsaCleanUp();

	return 0;
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


int networkExists(SOCKET sock, char *networkName)
{
	char *payload = nullptr;
	char *recvBuf = nullptr;
	int payloadLen;
	char *httpreq = "HTTP/1.1\r\nHost:127.0.0.1\r\nContent-Type:application/json\r\nConnection:close\r\n\r\n";

	printf("Checking if network already exists..\n");

	payloadLen = strlen(DockerNwGet) + strlen(networkName) + strlen(httpreq);
	payload = (char *)malloc(payloadLen + 1);
	sprintf(payload, "%s%s %s", DockerNwGet, networkName, httpreq);

	recvBuf = httpSendResponse(sock, payload);
	if (!recvBuf)
	{
		printf("Unexpected error while check for existing network:%s\n", networkName);
		free(payload);
		return 0;
	}

	printf("receive buf:%s\n", recvBuf);
	int ret = retrieveHttpStatus(recvBuf);
	if (ret == 200)
	{
		printf("Network Initialized already\n");
		free(recvBuf);
		return 0;
	}
	free(recvBuf);

	if (payload)
		free(payload);
	return 1;
}

DWORD setupUnderlayInAzure(ClusterConfig *clusterConfig)
{
	int conLength = 0, ipamLength = 0;
	char *nwCreateReq = nullptr;
	char *recvBuf = nullptr;
	char *responseLocation = nullptr;
	int totalLength = 0;
	char *subnetIp = nullptr;
	char *gwIp = nullptr;
	char ipamConfig[512];
	unsigned long retval = 0;


	if (!wsaInitialize())
	{
		return UNEXPECTED_ERROR;
	}

	SOCKET sock = connectToHost("localhost", 2375);
	if (sock == INVALID_SOCKET)
	{
		fprintf(stderr, "Connect to host failed.\n");
		return UNREACHABLE_HOST;
	}

	if (networkExists(sock, clusterConfig->networkName))
	{
		shutdown(sock, SD_BOTH);
		_close((int)sock);
		wsaCleanUp();

		getSubnetGwIpfromNMAgent(&subnetIp, &gwIp);

		sprintf(ipamConfig, "\"IPAM\":{\"Config\":[{\"Subnet\":\"%s\",\"Gateway\":\"%s\"}]}", subnetIp, gwIp);
		printf("ipamconfig:%s\n", ipamConfig);
		if (!wsaInitialize())
		{
			return UNEXPECTED_ERROR;
		}

		sock = connectToHost("localhost", 2375);

		ipamLength = strlen(ipamConfig);
		conLength = strlen(clusterConfig->networkName) + 9 + 19 + 4 + ipamLength;
		totalLength = strlen(DockerNwCreatePost) + conLength + ipamLength + 64;
		nwCreateReq = (char *)malloc(totalLength + 1);

		sprintf(nwCreateReq, "%sContent-Length:%d\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s}", DockerNwCreatePost, conLength, clusterConfig->networkName, "l2tunnel", ipamConfig);

		printf("Now going to get create network driver... cmd:%s:%d:%llu:%d\n", nwCreateReq, totalLength, strlen(nwCreateReq), conLength);

		recvBuf = httpSendResponse(sock, nwCreateReq);
		if (!recvBuf) 
		{
			printf("Error while creating docker network");
			retval = UNEXPECTED_ERROR;
			goto cleanup2;
		}

		int ret = retrieveHttpStatus(recvBuf);
		printf("status:%d\n", ret);
		if (ret == 201)
		{
			printf("docker network create succeeded\n");
		}
		else
		{
			printf("docker network create failed:%s\n", recvBuf);
			retval = UNEXPECTED_ERROR;
		}
	}

cleanup2:
	if (nwCreateReq)
		free(nwCreateReq);
	if (recvBuf)
		free(recvBuf);
	if (subnetIp)
		free(subnetIp);
	if (gwIp)
		free(gwIp);

	shutdown(sock, SD_BOTH);
	_close((int)sock);
	wsaCleanUp();
	return retval;
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
DllExport unsigned int GetReservedIpAddress(const char *reservationId)
{
	return GetReservedIpFromRepository(reservationId);
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