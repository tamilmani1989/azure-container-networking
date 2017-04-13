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
#include "../../Logging/Logging.h"
std::mutex initWsaMutex;
int wsaInitialized = 0;

extern int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
extern void ParseSubnetAndGatewayIp(char *xmlResponse, char **sunbetIp, char **gatewayIp);
extern bool RetrieveGatewayFromSubnet(char*subnet, char **gatewayIp);

int initWsa()
{
	char logMsg[2048];
	size_t msgSize = 2048;
	WORD wVersionRequested;
	WSADATA wsaData;
	int wsaerr;
	int success = 1;

	wVersionRequested = MAKEWORD(2, 2);

	wsaerr = WSAStartup(wVersionRequested, &wsaData);
	if (wsaerr != 0)
	{
		// could not find a usable Winsock DLL.  
		sprintf_s(logMsg, msgSize, "WSAStartup failed with error: %d\n", wsaerr);
		AquaLogging::Log(info, logMsg);
		success = 0;
	}

	/* If the DLL supports versions greater
	* than 2.2 in addition to 2.2, it will still return
	* 2.2 in wVersion since that is the version we
	* requested.
	* */

	if (LOBYTE(wsaData.wVersion) != 2 || HIBYTE(wsaData.wVersion) != 2)
	{
		sprintf_s(logMsg, msgSize, "Could not find a usable version of Winsock.dll\n");
		AquaLogging::Log(info, logMsg);
		WSACleanup();
		success = 0;
	}
	else
	{
		sprintf_s(logMsg, msgSize, "The Winsock 2.2 dll was found okay\n");
		AquaLogging::Log(info, logMsg);
	}

	return success;
}

SOCKET connectToHost(char *host, USHORT port)
{
	char logMsg[2048];
	size_t msgSize = 2048;
	struct hostent *hp;
	struct sockaddr_in addr;
	SOCKET sock;

	// Using depricated gethostbyname which is ipv4 only
	if ((hp = gethostbyname(host)) == nullptr)
	{
		sprintf_s(logMsg, msgSize,  "gethostbyname failed %d : %s", WSAGetLastError(), host);
		AquaLogging::Log(info, logMsg);
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
			sprintf_s(logMsg, msgSize,  "Connection to host failed.\n");
			AquaLogging::Log(info, logMsg);
		}
		else
		{
			sprintf_s(logMsg, msgSize,  "Connection to host failed with error code %d\n", error);
			AquaLogging::Log(info, logMsg);
		}
		return INVALID_SOCKET;
	}

	sprintf_s(logMsg, msgSize, "Connected successfully to host.\n");
	AquaLogging::Log(info, logMsg);
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
	char logMsg[2048];
	size_t msgSize = 2048;
	
	initWsaMutex.lock();
	if (wsaInitialized==0)
	{
		if (initWsa())
		{
			wsaInitialized = 1;
			retval = 1;
		}
		else
		{
			sprintf_s(logMsg, msgSize,  "wsa initialization failed\n");
			AquaLogging::Log(info, logMsg);
			retval = 0;
		}
	}
	else
	{
		wsaInitialized++;
	}
	initWsaMutex.unlock();
	
	return retval;
}

void wsaCleanUp()
{
	if (wsaInitialized > 0)
	{
		initWsaMutex.lock();
		if (wsaInitialized > 0)
		{
			wsaInitialized--;

			if (wsaInitialized == 0)
			{
				WSACleanup();
			}
		}
		initWsaMutex.unlock();
	}
}

char* httpSendResponse(SOCKET sock, char *payload)
{

	int bytesRead;
	char *recvBuffer = nullptr;
	int byteCount = 0;
	char logMsg[2048];
	size_t msgSize = 2048;

	int bytesWritten = send(sock, payload, (int)strlen(payload), 0);
	if (bytesWritten == SOCKET_ERROR)
	{
		sprintf_s(logMsg, msgSize,  "Writing to socket failed errno=%d\n", errno);
		AquaLogging::Log(info, logMsg);
	}
	else
	{
		sprintf_s(logMsg, msgSize, "Sent %d bytes on the connection.\nReading response (may take some time).... ", bytesWritten);
		AquaLogging::Log(info, logMsg);
		do
		{
			char c;
			bytesRead = recv(sock, &c, 1, 0);
			if (bytesRead == 1)
			{
				byteCount++;
				if (!InsertIntoBuffer(&recvBuffer, byteCount, c))
				{
					sprintf_s(logMsg, msgSize,  "Failed to insert a char into buffer");
					AquaLogging::Log(err, logMsg);
					return NULL;
				}
			}
		} while (bytesRead > 0);

		if (bytesRead == -1)
		{
			sprintf_s(logMsg, msgSize,  "Reading failed. errno=%d \n", errno);
			AquaLogging::Log(err, logMsg);
		}
		else if (bytesRead == 0)
		{
			sprintf_s(logMsg, msgSize, "Read finished\n");
			if (!InsertIntoBuffer(&recvBuffer, byteCount + 1, '\0'))
			{
				sprintf_s(logMsg, msgSize,  "Failed to insert EOL into buffer");
				AquaLogging::Log(err, logMsg);
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
	char logMsg[2048];
	size_t msgSize = 2048;
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
	sprintf_s(logMsg, msgSize, "returning %d as http response\n", ret);
	AquaLogging::Log(info, logMsg);
	return ret;
}
int getSubnetGwIpfromNMAgent(char **subnetIp, char **gwIp)
{
	char *recvBuffer = nullptr;
	char *responseLocation = nullptr;
	char logMsg[2048];
	size_t msgSize = 2048;

	if (!wsaInitialize())
	{
		return -1;
	}

	sprintf_s(logMsg, msgSize, "Trying to connect to nm agent.\n");
	AquaLogging::Log(info, logMsg);
	SOCKET sock = connectToHost("169.254.169.254", 80);

	if (sock == INVALID_SOCKET)
	{
		sprintf_s(logMsg, msgSize,  "Connect to host failed.\n");
		AquaLogging::Log(err, logMsg);
		wsaCleanUp();
		return -1;
	}

	recvBuffer = httpSendResponse(sock, HostSuffix);
	if (!recvBuffer) {
		sprintf_s(logMsg, msgSize, "Error while retrieving xml file from NMAgent");
		AquaLogging::Log(err, logMsg);
		goto cleanup_subgw;
	}
	//sprintf_s(logMsg, msgSize, "xml recv buf:%s\n", recvBuffer);

	int ret = retrieveHttpStatus(recvBuffer);
	sprintf_s(logMsg, msgSize, "status:%d\n", ret);
	AquaLogging::Log(info, logMsg);
	if (ret != 200)
	{
		sprintf_s(logMsg, msgSize, "Error while retrieving xml file from NMAgent. Invalid Status code:%s\n", recvBuffer);
		AquaLogging::Log(err, logMsg);
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
		sprintf_s(logMsg, msgSize,  "Unable to find portion of http response header that contains valid nm agent response response %s", recvBuffer);
		AquaLogging::Log(err, logMsg);
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
	char logMsg[2048];
	size_t msgSize = 2048;

	if (!wsaInitialize())
	{
		return -1;
	}

	sprintf_s(logMsg, msgSize, "Trying to connect to nm agent.\n");
	AquaLogging::Log(info, logMsg);
	SOCKET sock = connectToHost("169.254.169.254", 80);

	if (sock == INVALID_SOCKET)
	{
		sprintf_s(logMsg, msgSize,  "Connect to host failed.\n");
		AquaLogging::Log(err, logMsg);
		wsaCleanUp();
		return -1;
	}
	sprintf_s(logMsg, msgSize, "Now going to get ips from nm agent... fd: %llu\n", sock);
	AquaLogging::Log(info, logMsg);
	int bytesWritten = send(sock, HostSuffix, (int)strlen(HostSuffix), 0);
	if (bytesWritten == SOCKET_ERROR)
	{
		sprintf_s(logMsg, msgSize,  "Writing to socket failed errno=%d\n", errno);
		AquaLogging::Log(err, logMsg);
	}
	else
	{
		sprintf_s(logMsg, msgSize, "Sent %d bytes on the connection.\nNow reading into buffer..", bytesWritten);
		AquaLogging::Log(info, logMsg);
		do
		{
			char c;
			bytesRead = recv(sock, &c, 1, 0);
			if (bytesRead == 1)
			{
				byteCount++;
				if (!InsertIntoBuffer(&recvBuffer, byteCount, c))
				{
					sprintf_s(logMsg, msgSize,  "Failed to insert a char into buffer");
					AquaLogging::Log(err, logMsg);
					ipCount = -1;
					goto cleanup;
				}
			}
		} while (bytesRead > 0);

		if (bytesRead == -1)
		{
			sprintf_s(logMsg, msgSize,  "Reading failed. errno=%d \n", errno);
			AquaLogging::Log(err, logMsg);
		}
		else if (bytesRead == 0)
		{
			if (!InsertIntoBuffer(&recvBuffer, byteCount + 1, '\0'))
			{
				sprintf_s(logMsg, msgSize,  "Failed to insert EOL into buffer");
				AquaLogging::Log(err, logMsg);
				ipCount = -1;
				goto cleanup;
			}

			sprintf_s(logMsg, msgSize, "\nSuccessfully finished reading. Printing response below..\n");
			AquaLogging::Log(info, logMsg);
			responseLocation = strstr(recvBuffer, HttpResponseHeaderSeparator);
			responseLocation += strlen(HttpResponseHeaderSeparator);
			printf(responseLocation);
		}
		if (responseLocation != nullptr)
		{
			if (ips == NULL)
			{
				ipCount = ParseSecondaryIpAddresses(responseLocation, 1, ips);
				sprintf_s(logMsg, msgSize, "Received only ip count as %d\n", ipCount);
				AquaLogging::Log(info, logMsg);
			}
			else
			{
				ipCount = ParseSecondaryIpAddresses(responseLocation, 0, ips);
				sprintf_s(logMsg, msgSize, "Received ips with ip count as %d\n", ipCount);
				AquaLogging::Log(info, logMsg);
			}
		}
		else
		{
			sprintf_s(logMsg, msgSize,  "Unable to find portion of http response header that contains valid nm agent response response %s", recvBuffer);
			AquaLogging::Log(err, logMsg);
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
		sprintf_s(logMsg, msgSize, "+++++++++ERROR While closing Socket!+++++++++++++++++\n");
		AquaLogging::Log(err, logMsg);
	}

	wsaCleanUp();
	return ipCount;
}

DWORD GetHostIp(unsigned int *ip)
{
	struct hostent *hp;
	char hostName[255]; //RFC says it cannot exceed 255 chars
	unsigned long retval = 0;
	char logMsg[2048];
	size_t msgSize = 2048;

	if (!wsaInitialize())
	{
		return UNEXPECTED_ERROR;
	}

	if (gethostname(hostName, 255) != 0)
	{
		sprintf_s(logMsg, msgSize, "error happenned during hethostname() %d\n", WSAGetLastError());
		AquaLogging::Log(err, logMsg);
		retval = UNEXPECTED_ERROR;
	}
	else
	{
		if ((hp = gethostbyname(hostName)) == nullptr)
		{
			sprintf_s(logMsg, msgSize,  "gethostbyname failed %d : %s", WSAGetLastError(), hostName);
			AquaLogging::Log(err, logMsg);
			retval = UNEXPECTED_ERROR;
		}
		else
		{
			(*ip) = *(u_long*)hp->h_addr_list[0];
			sprintf_s(logMsg, msgSize, "h_name = %s\n", hp->h_name);
		}
	}
	sprintf_s(logMsg, msgSize, "retrieved hostip 0x%08x", *ip);
	AquaLogging::Log(info, logMsg);
	wsaCleanUp();
	sprintf_s(logMsg, msgSize, "GetHostIp returning %d\n", retval);
	AquaLogging::Log(info, logMsg);
	return retval;
}

int networkExists(SOCKET sock, char *networkName)
{
	char *payload = nullptr;
	char *recvBuf = nullptr;
	size_t payloadLen;
	char logMsg[2048];
	size_t msgSize = 2048;
	char *httpreq = "HTTP/1.1\r\nHost:127.0.0.1\r\nContent-Type:application/json\r\nConnection:close\r\n\r\n";

	sprintf_s(logMsg, msgSize, "Checking if network already exists.. %s\n", networkName);
	AquaLogging::Log(info, logMsg);

	payloadLen = strlen(DockerNwGet) + strlen(networkName) + strlen(httpreq);

	payload = (char *)malloc(payloadLen + 2);
	sprintf_s(payload, payloadLen + 2, "%s%s %s", DockerNwGet, networkName, httpreq);
	payload[payloadLen + 1] = '\0';
	recvBuf = httpSendResponse(sock, payload);
	if (!recvBuf)
	{
		sprintf_s(logMsg, msgSize, "Unexpected error while check for existing network:%s\n", networkName);
		AquaLogging::Log(err, logMsg);
		free(payload);
		return 0;
	}

	sprintf_s(logMsg, msgSize, "receive buf:%s\n", recvBuf);
	AquaLogging::Log(info, logMsg);
	int ret = retrieveHttpStatus(recvBuf);
	if (ret == 200)
	{
		sprintf_s(logMsg, msgSize, "Network Initialized already\n");
		AquaLogging::Log(warn, logMsg);
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
	char logMsg[2048];
	size_t msgSize = 2048;

	if (!wsaInitialize())
	{
		return UNEXPECTED_ERROR;
	}
	sprintf_s(logMsg, msgSize, "wsa initialization finished\n");
	AquaLogging::Log(info, logMsg);
	SOCKET sock = connectToHost("localhost", 2375);
	if (sock == INVALID_SOCKET)
	{
		sprintf_s(logMsg, msgSize,  "Connect to host failed.\n");
		AquaLogging::Log(err, logMsg);
		wsaCleanUp();
		return UNREACHABLE_HOST;
	}

	if (networkExists(sock, clusterConfig->networkName))
	{
		sprintf_s(logMsg, msgSize, "Network %s does not exist, going to create it\n", clusterConfig->networkName);
		AquaLogging::Log(info, logMsg);

		shutdown(sock, SD_BOTH);
		closesocket(sock);
		wsaCleanUp();
		sprintf_s(logMsg, msgSize, "wsa cleanup finished\n");
		AquaLogging::Log(info, logMsg);

		getSubnetGwIpfromNMAgent(&subnetIp, &gwIp);

		sprintf_s(logMsg, msgSize, "retrieved subnet and gateway %s %s\n", subnetIp, gwIp);
		AquaLogging::Log(info, logMsg);

		sprintf_s(ipamConfig, sizeof(ipamConfig), "\"IPAM\":{\"Config\":[{\"Subnet\":\"%s\",\"Gateway\":\"%s\"}]}", subnetIp, gwIp);
		AquaLogging::Log(info, logMsg);

		sprintf_s(logMsg, msgSize, "ipamconfig:%s\n", ipamConfig);
		AquaLogging::Log(info, logMsg);
		if (!wsaInitialize())
		{
			return UNEXPECTED_ERROR;
		}

		sock = connectToHost("localhost", 2375);

		ipamLength = strlen(ipamConfig);
		conLength = strlen(clusterConfig->networkName) + 9 + 19 + 4 + ipamLength;
		totalLength = strlen(DockerNwCreatePost) + conLength + ipamLength + 64;
		nwCreateReq = (char *)malloc(totalLength + 1);

		sprintf_s(nwCreateReq, totalLength + 1, 
			"%sContent-Length:%d\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s}", 
				DockerNwCreatePost, (int)conLength, clusterConfig->networkName, "l2tunnel", ipamConfig);	
		AquaLogging::Log(info, logMsg);

		sprintf_s(logMsg, msgSize, "Now going to get create network driver... cmd:%s:%d:%llu:%d\n", 
			nwCreateReq, (int)totalLength, strlen(nwCreateReq), (int)conLength);
		AquaLogging::Log(info, logMsg);
		recvBuf = httpSendResponse(sock, nwCreateReq);
		if (!recvBuf)
		{
			sprintf_s(logMsg, msgSize, "Error while creating docker network");
			AquaLogging::Log(err, logMsg);
			retval = UNEXPECTED_ERROR;
			goto cleanup2;
		}

		int ret = retrieveHttpStatus(recvBuf);
		sprintf_s(logMsg, msgSize, "status:%d\n", ret);
		if (ret == 201)
		{
			sprintf_s(logMsg, msgSize, "docker network create succeeded\n");
			AquaLogging::Log(err, logMsg);
		}
		else
		{
			sprintf_s(logMsg, msgSize, "docker network create failed:%s\n", recvBuf);
			AquaLogging::Log(err, logMsg);
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
	sprintf_s(logMsg, msgSize, "setup underlay returning %d", retval);
	AquaLogging::Log(info, logMsg);
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
	char logMsg[2048];
	size_t msgSize = 2048;

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
	AquaLogging::Log(info, logMsg);

	sprintf_s(logMsg, msgSize, "Now going to get create network driver... cmd:%s:%d:%llu:%d\n",
		nwCreateReq, (int)totalLength, strlen(nwCreateReq), (int)conLength);
	AquaLogging::Log(info, logMsg);

	recvBuf = httpSendResponse(sock, nwCreateReq);
	if (!recvBuf)
	{
		sprintf_s(logMsg, msgSize, "Error while creating docker network");
		AquaLogging::Log(err, logMsg);
		retval = UNEXPECTED_ERROR;
	}
	else
	{
		int ret = retrieveHttpStatus(recvBuf);
		sprintf_s(logMsg, msgSize, "status:%d\n", ret);
		AquaLogging::Log(info, logMsg);
		if (ret == 201)
		{
		AquaLogging::Log(err, logMsg);
			sprintf_s(logMsg, msgSize, "docker network create succeeded\n");
			AquaLogging::Log(info, logMsg);
		}
		else
		{
			sprintf_s(logMsg, msgSize, "docker network create failed:%s\n", recvBuf);
			AquaLogging::Log(err, logMsg);
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

	sprintf_s(logMsg, msgSize, "create docker network returning %d\n", retval);
	AquaLogging::Log(info, logMsg);
	return retval;
}

DWORD setupOverlay(ClusterConfig *clusterConfig)
{
	char *recvBuf = nullptr;
	char *gwIp = nullptr;
	char ipamConfig[2048];
	unsigned long retval = 0;
	char logMsg[2048];
	size_t msgSize = 2048;

	if (clusterConfig->networkName == NULL)
	{
		return UNSPECIFIED_NETWORK_NAME;
	}

	if (!wsaInitialize())
	{
		return UNEXPECTED_ERROR;
	}

	sprintf_s(logMsg, msgSize, "wsa initialization finished to setup overlay\n");
	AquaLogging::Log(info, logMsg);

	SOCKET sock = connectToHost("localhost", 2375);
	if (sock == INVALID_SOCKET)
	{
		sprintf_s(logMsg, msgSize,  "Connect to host failed (overlay).\n");
		AquaLogging::Log(err, logMsg);
		wsaCleanUp();
		return UNREACHABLE_DOCKER_DAEMON;
	}

	if (networkExists(sock, clusterConfig->networkName))
	{
		sprintf_s(logMsg, msgSize, "Overlay Network %s does not exist, going to create it\n", clusterConfig->networkName);
		AquaLogging::Log(info, logMsg);

		shutdown(sock, SD_BOTH);
		closesocket(sock);
		wsaCleanUp();
		sprintf_s(logMsg, msgSize, "wsa cleanup finished\n");
		AquaLogging::Log(info, logMsg);

		if(clusterConfig->overlaySubnet == NULL)
		{
			return OVERLAY_SUBNET_NOT_SPECIFIED;
		}

		if(clusterConfig->overlaySubnet->prefixLength > 30)
		{
			return SUBNET_SIZE_TOO_SMALL;
		}

		sprintf_s(logMsg, msgSize, "Going to get gateway ip from %d\n", clusterConfig->overlaySubnet->ipAddress);
		AquaLogging::Log(info, logMsg);
		char gatewayIp[INET_ADDRSTRLEN];
		if(inet_ntop(AF_INET, &clusterConfig->overlaySubnet->ipAddress, gatewayIp, INET_ADDRSTRLEN)==NULL)
		{
			return MALFORMED_SUBNET_IP;
		}

		if(RetrieveGatewayFromSubnet(gatewayIp, &gwIp) == false)
		{
			return MALFORMED_SUBNET_IP;
		}

		sprintf_s(logMsg, msgSize, "retrieved subnet and gateway %s/%d %s\n", gatewayIp, clusterConfig->overlaySubnet->prefixLength, gwIp);
		AquaLogging::Log(info, logMsg);

		sprintf_s(ipamConfig, sizeof(ipamConfig), 
			"\"IPAM\":{\"Config\":[{\"Subnet\":\"%s/%d\",\"Gateway\":\"%s\"}]}", 
			gatewayIp, clusterConfig->overlaySubnet->prefixLength, gwIp);
		AquaLogging::Log(info, logMsg);

		sprintf_s(logMsg, msgSize, "ipamconfig:%s\n", ipamConfig);
		AquaLogging::Log(info, logMsg);
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
	sprintf_s(logMsg, msgSize, "setupOverlay returning %d\n", retval);
	AquaLogging::Log(info, logMsg);
	return retval;
}

#endif