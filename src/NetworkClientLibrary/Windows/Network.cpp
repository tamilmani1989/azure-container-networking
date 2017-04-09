#ifdef _WIN32

#define _WINSOCK_DEPRECATED_NO_WARNINGS
#include "../Common/aquanetworkclient.h"
#include <stdlib.h>
#include <string.h>
#include "../Common/nclcore.h"
#include <WinSock2.h>
#include <stdio.h>
#include <iostream>
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>
#define HostSuffix "GET /machine/plugins?comp=nmagent&type=getinterfaceinfov1 HTTP/1.1\r\nHost: 169.254.169.254\r\nConnection: close\r\n\r\n"
#define DockerNwCreatePost "POST /networks/create HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Type: application/json\r\nConnection: close\r\n"
#define DockerNwGet "GET /networks/"
#define HttpResponseHeaderSeparator "\r\n\r\n"
#define BUFFER_SIZE 1024
#include <mutex>
std::mutex initWsaMutex;
int wsaInitialized = 0;

extern int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
extern void ParseSubnetAndGatewayIp(char *xmlResponse, char **sunbetIp, char **gatewayIp);
extern bool RetrieveGatewayFromSubnet(char*subnet, char **gatewayIp);

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

	// Using depricated gethostbyname which is ipv4 only
	if ((hp = gethostbyname(host)) == nullptr)
	{
		fprintf(stderr, "gethostbyname failed %d : %s", WSAGetLastError(), host);
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
			fprintf(stderr, "Connection to host failed with error code %d\n", error);
		}
		return INVALID_SOCKET;
	}

	printf("Connected successfully to host.\n");
	return sock;
}


int InsertIntoBuffer(char **recvBuffer, int byteCount, char c)
{
	char *newBuffer = (char *)realloc((void *)(*recvBuffer), byteCount);
	if (newBuffer == nullptr)
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
	int retval = 1;
	if (!wsaInitialized)
	{
		initWsaMutex.lock();
		if (!wsaInitialized)
		{
			if (initWsa())
			{
				wsaInitialized = 1;
				retval = 1;
			}
			else
			{
				fprintf(stderr, "wsa initialization failed\n");
				retval = 0;
			}
		}
		initWsaMutex.unlock();
	}
	return retval;
}

void wsaCleanUp()
{
	if (wsaInitialized)
	{
		initWsaMutex.lock();
		if (wsaInitialized)
		{
			WSACleanup();
			wsaInitialized = 0;
		}
		initWsaMutex.unlock();
	}
}

char* httpSendResponse(SOCKET sock, char *payload)
{

	int bytesRead;
	char *recvBuffer = nullptr;
	int byteCount = 0;

	int bytesWritten = send(sock, payload, (int)strlen(payload), 0);
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
	printf("returning %d as http response\n", ret);
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
		wsaCleanUp();
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
	closesocket((int)sock);
	wsaCleanUp();

	return 0;
}



int getIpsFromNmAgent(unsigned int *ips)
{
	int ipCount = 0;
	char *recvBuffer = nullptr;
	int bytesRead;
	char *responseLocation = nullptr;
	int byteCount = 0;

	if (!wsaInitialize())
	{
		return -1;
	}

	printf("Trying to connect to nm agent.\n");
	SOCKET sock = connectToHost("169.254.169.254", 80);

	if (sock == INVALID_SOCKET)
	{
		fprintf(stderr, "Connect to host failed.\n");
		wsaCleanUp();
		return -1;
	}
	printf("Now going to get ips from nm agent... fd: %llu\n", sock);
	int bytesWritten = send(sock, HostSuffix, (int)strlen(HostSuffix), 0);
	if (bytesWritten == SOCKET_ERROR)
	{
		fprintf(stderr, "Writing to socket failed errno=%d\n", errno);
	}
	else
	{
		printf("Sent %d bytes on the connection.\nNow ", bytesWritten);

		printf("Reading into buffer..\n");
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
			if (!InsertIntoBuffer(&recvBuffer, byteCount + 1, '\0'))
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

	if (closesocket(sock) != 0)
	{
		printf("+++++++++ERROR While closing Socket!+++++++++++++++++\n");
	}

	wsaCleanUp();
	return ipCount;
}


int networkExists(SOCKET sock, char *networkName)
{
	char *payload = nullptr;
	char *recvBuf = nullptr;
	size_t payloadLen;
	char *httpreq = "HTTP/1.1\r\nHost:127.0.0.1\r\nContent-Type:application/json\r\nConnection:close\r\n\r\n";

	printf("Checking if network already exists.. %s\n", networkName);

	payloadLen = strlen(DockerNwGet) + strlen(networkName) + strlen(httpreq);

	payload = (char *)malloc(payloadLen + 2);
	sprintf_s(payload, payloadLen + 2, "%s%s %s", DockerNwGet, networkName, httpreq);
	payload[payloadLen + 1] = '\0';
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
		if (payload) free(payload);
		free(recvBuf);
		return 0;
	}

	if (payload) free(payload);
	free(recvBuf);
	return 1;
}

DWORD setupUnderlayInAzure(ClusterConfig *clusterConfig)
{
	size_t conLength = 0, ipamLength = 0;
	char *nwCreateReq = nullptr;
	char *recvBuf = nullptr;
	size_t totalLength = 0;
	char *subnetIp = nullptr;
	char *gwIp = nullptr;
	char ipamConfig[2048];
	unsigned long retval = 0;


	if (!wsaInitialize())
	{
		return UNEXPECTED_ERROR;
	}
	printf("wsa initialization finished\n");
	SOCKET sock = connectToHost("localhost", 2375);
	if (sock == INVALID_SOCKET)
	{
		fprintf(stderr, "Connect to host failed.\n");
		wsaCleanUp();
		return UNREACHABLE_HOST;
	}

	if (networkExists(sock, clusterConfig->networkName))
	{
		printf("Network %s does not exist, going to create it\n", clusterConfig->networkName);

		shutdown(sock, SD_BOTH);
		closesocket(sock);
		wsaCleanUp();
		printf("wsa cleanup finished\n");

		getSubnetGwIpfromNMAgent(&subnetIp, &gwIp);

		printf("retrieved subnet and gateway %s %s\n", subnetIp, gwIp);

		sprintf_s(ipamConfig, sizeof(ipamConfig), "\"IPAM\":{\"Config\":[{\"Subnet\":\"%s\",\"Gateway\":\"%s\"}]}", subnetIp, gwIp);

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

		sprintf_s(nwCreateReq, totalLength + 1, "%sContent-Length:%d\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s}", DockerNwCreatePost, (int)conLength, clusterConfig->networkName, "l2tunnel", ipamConfig);

		printf("Now going to get create network driver... cmd:%s:%d:%llu:%d\n", nwCreateReq, (int)totalLength, strlen(nwCreateReq), (int)conLength);

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
	closesocket(sock);
	wsaCleanUp();
	return retval;
}

DWORD installHNSOverlayPlugin()
{
	int retval;
	
	system("start powershell.exe Set-ExecutionPolicy RemoteSigned \n");
	retval = system("start powershell.exe InstallPlugin.ps1");
	
	return retval;
}

unsigned long CreateDockerNetwork(char *networkName, char ipamConfig[2048])
{
	char *recvBuf = NULL;
	unsigned long retval = 0;
	SOCKET sock = connectToHost("localhost", 2375);
	if(sock == INVALID_SOCKET)
	{
		return UNREACHABLE_DOCKER_DAEMON;
	}

	size_t ipamLength = strlen(ipamConfig);
	size_t conLength = strlen(networkName) + 9 + 19 + 4 + ipamLength;
	size_t totalLength = strlen(DockerNwCreatePost) + conLength + ipamLength + 64;
	char *nwCreateReq = (char *)malloc(totalLength + 1);

	sprintf_s(nwCreateReq, totalLength + 1, 
	          "%sContent-Length:%d\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s}", 
	          DockerNwCreatePost, (int)conLength, networkName, "l2tunnel", ipamConfig);

	printf("Now going to get create network driver... cmd:%s:%d:%llu:%d\n", nwCreateReq, (int)totalLength, strlen(nwCreateReq), (int)conLength);

	recvBuf = httpSendResponse(sock, nwCreateReq);
	if (!recvBuf)
	{
		printf("Error while creating docker network");
		retval = UNEXPECTED_ERROR;
	}
	else
	{
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

	if (recvBuf)
	{
		free(recvBuf);
	}
	if (nwCreateReq)
	{
		free(nwCreateReq);
	}

	shutdown(sock, SD_BOTH);
	closesocket(sock);
	return retval;
}

DWORD setupOverlay(ClusterConfig *clusterConfig)
{

	size_t conLength = 0, ipamLength = 0;
	char *recvBuf = nullptr;
	size_t totalLength = 0;
	char *gwIp = nullptr;
	char ipamConfig[2048];
	unsigned long retval = 0;

	if (clusterConfig->networkName == NULL)
	{
		return UNSPECIFIED_NETWORK_NAME;
	}

	if (!wsaInitialize())
	{
		return UNEXPECTED_ERROR;
	}

	printf("wsa initialization finished to setup overlay\n");

	SOCKET sock = connectToHost("localhost", 2375);
	if (sock == INVALID_SOCKET)
	{
		fprintf(stderr, "Connect to host failed (overlay).\n");
		wsaCleanUp();
		return UNREACHABLE_DOCKER_DAEMON;
	}

	if (networkExists(sock, clusterConfig->networkName))
	{
		printf("Overlay Network %s does not exist, going to create it\n", clusterConfig->networkName);

		shutdown(sock, SD_BOTH);
		closesocket(sock);
		wsaCleanUp();
		printf("wsa cleanup finished\n");

		if(clusterConfig->overlaySubnet == NULL)
		{
			return OVERLAY_SUBNET_NOT_SPECIFIED;
		}

		if(clusterConfig->overlaySubnet->prefixLength > 28)
		{
			return SUBNET_SIZE_TOO_SMALL;
		}

		printf("Going to get gateway ip from %d\n", clusterConfig->overlaySubnet->ipAddress);
		char gatewayIp[INET_ADDRSTRLEN];
		if(inet_ntop(AF_INET, &clusterConfig->overlaySubnet->ipAddress, gatewayIp, INET_ADDRSTRLEN)==NULL)
		{
			return MALFORMED_SUBNET_IP;
		}

		if(RetrieveGatewayFromSubnet(gatewayIp, &gwIp) == false)
		{
			return MALFORMED_SUBNET_IP;
		}

		printf("retrieved subnet and gateway %s/%d %s\n", gatewayIp, clusterConfig->overlaySubnet->prefixLength, gwIp);

		sprintf_s(ipamConfig, sizeof(ipamConfig), 
			"\"IPAM\":{\"Config\":[{\"Subnet\":\"%s/%d\",\"Gateway\":\"%s\"}]}", 
			gatewayIp, clusterConfig->overlaySubnet->prefixLength, gwIp);

		printf("ipamconfig:%s\n", ipamConfig);
		if (!wsaInitialize())
		{
			return UNEXPECTED_ERROR;
		}

		retval = CreateDockerNetwork(clusterConfig->networkName, ipamConfig);
	}
	else 
	{
		if (recvBuf)
		{
			free(recvBuf);
			recvBuf = NULL;
		}
		shutdown(sock, SD_BOTH);
		closesocket(sock);
	}

	if (gwIp)
	{
		free(gwIp);
		gwIp = NULL;
	}

	wsaCleanUp();
	return retval;
}

#endif