#include "WinMessageExchange.h"
#include <process.h>

WinMessageExchange::WinMessageExchange(Protocol protocol, USHORT port, ReceiveFunc receiveFunc)
	:MessageExchange::MessageExchange(protocol, port, receiveFunc)
{
	m_portString = to_string(m_port);
	m_wsaInitialized = false;
	m_serverSocket = INVALID_SOCKET;
	m_clientSocket = INVALID_SOCKET;
}

DWORD WinMessageExchange::Initialize()
{
	DWORD result = E_SUCCESS;
	int iResult;
	WSAData wsaData;
	
	E_WSA(WSAStartup(MAKEWORD(2, 2), &wsaData), "WSAStartup failed: ", E_WSAStartupFailed);
	m_wsaInitialized = true;

	//setup server socket
	struct addrinfo hints;

	ZeroMemory(&hints, sizeof(hints));
	hints.ai_family = AF_INET;
	switch (m_protocol)
	{
		case Protocol::Tcp :
			hints.ai_socktype = SOCK_STREAM;
			hints.ai_protocol = IPPROTO_TCP;
			break;
		case Protocol::Udp:
			hints.ai_socktype = SOCK_DGRAM;
			hints.ai_protocol = IPPROTO_UDP;
			break;
		default:
			hints.ai_socktype = SOCK_DGRAM;
			hints.ai_protocol = IPPROTO_UDP;
			break;
	}
	hints.ai_flags = AI_PASSIVE;

	// Resolve the local address and port to be used by the server
	E_WSA(getaddrinfo(NULL, m_portString.c_str(), &hints, &m_addressInfo), "getaddrinfo failed: ", E_GetAddrFailed);

	//create socket
	m_serverSocket = socket(m_addressInfo->ai_family, m_addressInfo->ai_socktype, m_addressInfo->ai_protocol);
	E_WSA_FAILFOR(INVALID_SOCKET, m_serverSocket,"Server Socket failed.",E_SocketFailed);

	//bind
	E_WSA_FAILFOR(SOCKET_ERROR, bind(m_serverSocket, m_addressInfo->ai_addr, (int)m_addressInfo->ai_addrlen), "bind failed: ", E_BindFailed);

	//create send socket
	m_clientSocket = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
	E_WSA_FAILFOR(INVALID_SOCKET, m_clientSocket, "Client Socket failed.", E_SocketFailed);

	//receive
	_beginthread(ReceiveMessage, 0, this);

AquaError:
	if (result != E_SUCCESS)
	{
		Cleanup();
	}
	return result;
}

void WinMessageExchange::ReceiveMessage(void *param)
{
	WinMessageExchange *pWinSocket = (WinMessageExchange*)param;

	const int bufferLength = 1024;
	char recvBuffer[bufferLength];

	char senderAddressString[256];
	sockaddr_in senderAddress;
	int senderAddressSize = sizeof(senderAddress);

	int iResult = recvfrom(pWinSocket->m_serverSocket, recvBuffer, bufferLength, 0, (SOCKADDR *)&senderAddress, &senderAddressSize);
	while (iResult > 0)
	{
		inet_ntop(senderAddress.sin_family, &senderAddress.sin_addr, &senderAddressString[0], 256);
		pWinSocket->Receive(senderAddressString, &recvBuffer[0], bufferLength);
		iResult = recvfrom(pWinSocket->m_serverSocket, recvBuffer, bufferLength, 0, (SOCKADDR *)&senderAddress, &senderAddressSize);
	}
	_endthread();
}

DWORD WinMessageExchange::Send(const char *receiver, const char *buffer, USHORT bufLength)
{
	DWORD result = E_SUCCESS;
	int iResult;

	sockaddr_in recvAddress;
	recvAddress.sin_family = AF_INET;
	recvAddress.sin_port = htons(m_port);
	//recvAddress.sin_addr.s_addr = inet_addr(receiver);
	if (1 != inet_pton(AF_INET, receiver, &recvAddress.sin_addr.s_addr))
	{
		E_WSA_FAIL("sendto failed", E_SendFailed);
	}

	E_WSA_EX(SOCKET_ERROR, sendto(m_clientSocket, buffer, bufLength, 0, (SOCKADDR *)& recvAddress, sizeof(recvAddress)), "sendto failed", E_SendFailed);

AquaError:
	return result;
}

void WinMessageExchange::Cleanup()
{
	if (m_serverSocket != INVALID_SOCKET)
	{
		//shutdown(m_serverSocket, SD_RECEIVE);
		closesocket(m_serverSocket);
		m_serverSocket = INVALID_SOCKET;
	}

	if (m_clientSocket != INVALID_SOCKET)
	{
		closesocket(m_clientSocket);
		m_clientSocket = INVALID_SOCKET;
	}

	if (m_addressInfo != NULL)
	{
		freeaddrinfo(m_addressInfo);
		m_addressInfo = NULL;
	}

	if (m_wsaInitialized)
	{
		WSACleanup();
		m_wsaInitialized = false;
	}
}

WinMessageExchange::~WinMessageExchange()
{
	Cleanup();
}

