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

#ifdef _WIN32
int wmain()
#endif
#ifdef __linux__
int main()
#endif
{
	printf("hello nisheeth");
	char *localHostAddress = "172.18.80.212";
	char **remoteHostAddresses = new char*{"172.18.80.212"};


	InitializeAquaNix(localHostAddress, remoteHostAddresses, ReceiveMapping, 10050, 10501);

	//DWORD result = SendMapping("endpoint_012345", "172.2.3.4", "6E-34-21-56-31");
	//cout << result;

	string myinput;
	cin >> myinput;
	return 0;
}

