#include "MessageExchange.h"

MessageExchange::MessageExchange(Protocol protocol, USHORT port, USHORT remotePort, ReceiveFunc receiveFunc)
{
	m_protocol = protocol;
	m_port = port;
	m_remotePort = remotePort;
	Receive = receiveFunc;
}

MessageExchange::~MessageExchange()
{

}

