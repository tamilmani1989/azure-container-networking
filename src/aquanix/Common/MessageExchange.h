#pragma once
#include "aquabase.h"

namespace aqua
{
	typedef DWORD (*ReceiveFunc) (char *sender, void *buffer, int bufLength);

	enum Protocol
	{
		Udp = 0,
		Tcp = 1,
	};

	class MessageExchange
	{
	protected:
		Protocol m_protocol;
		USHORT m_port;
		USHORT m_remotePort;
		ReceiveFunc Receive;

	public:
		MessageExchange(Protocol protocol, USHORT port, USHORT remotePort, ReceiveFunc receiveFunc);
		virtual ~MessageExchange();

		virtual DWORD Initialize() = 0;
		virtual DWORD Send(const char *receiver, const char *buffer, USHORT bufLength) = 0;
	};

}
