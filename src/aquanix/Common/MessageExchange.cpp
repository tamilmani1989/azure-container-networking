#include "MessageExchange.h"

MessageExchange::MessageExchange(string localHostIPAddress, Protocol protocol, USHORT port, USHORT remotePort, ReceiveFunc receiveFunc)
{
	m_protocol = protocol;
	m_port = port;
	m_remotePort = remotePort;
	Receive = receiveFunc;
	m_localHostIPAddress = localHostIPAddress;
}

MessageExchange::~MessageExchange()
{

}

