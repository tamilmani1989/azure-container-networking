#pragma once

#include "..\Common\MessageExchange.h"

#include <winsock2.h>
#include <ws2tcpip.h>
#include <iphlpapi.h>

#pragma comment(lib, "Ws2_32.lib")

namespace aqua
{
	class WinMessageExchange : public MessageExchange
	{
	private:
		SOCKET m_clientSocket;
		SOCKET m_serverSocket;
		string m_portString;
		bool m_wsaInitialized;
		struct addrinfo *m_addressInfo;

		void Cleanup();
		static void ReceiveMessage(void *params);

	public:
		WinMessageExchange(Protocol protocol, USHORT port, USHORT remotePort, ReceiveFunc receiveFunc);
		virtual ~WinMessageExchange();

		DWORD Initialize();

		virtual DWORD Send(const char *receiver, const char *buffer, USHORT bufLength);
	};
}