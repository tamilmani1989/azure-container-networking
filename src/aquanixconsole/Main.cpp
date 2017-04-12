#include <stdio.h>
#include <stdlib.h>
#ifdef _WIN32
#include <WinSock2.h>
#include <ws2ipdef.h>
#include <ws2tcpip.h>
#endif
#ifdef __linux__
#include <arpa/inet.h>
#include <string.h>
#include <cstdlib>
#endif
#include "../aquanix/Common/aquanixext.h"

DWORD ReceiveMapping(const char *endpointId, const char *hostAddress, const char *containerAddress, const char *macAddress)
{
	cout << endpointId << hostAddress << containerAddress << macAddress;
	return 0;
}

DWORD DeleteRemoteMapping(
	const char *endpointId)
{
	cout << endpointId;
	return 0;
}

DWORD GetLocalMappings(
	EndpointMapping *endpointMappings,
	USHORT *mappingsCount)
{
	endpointMappings = NULL;
	*mappingsCount = 0;
	return 0;
}

#ifdef _WIN32
int wmain()
#endif
#ifdef __linux__
int main()
#endif
{
	printf("hello nisheeth");
	char *localHostAddress = "172.27.18.40";
	char **remoteHostAddresses = new char*{"172.27.18.40"};
	char *mappingsSenderIP = "127.0.0.1";


	InitializeAquaNix(localHostAddress, remoteHostAddresses, mappingsSenderIP, ReceiveMapping, DeleteRemoteMapping, GetLocalMappings, 10050, 10050);
	string myinput;
	cin >> myinput;
	DWORD result = SendLocalMapping("endpoint_012345", "172.2.3.4", "6E-34-21-56-31");
	cout << result;
	cin >> myinput;
	
	return 0;
}

