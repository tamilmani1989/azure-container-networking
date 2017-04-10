#include "MessageExchange.h"

MessageExchange::MessageExchange(Protocol protocol, USHORT port, ReceiveFunc receiveFunc)
{
	m_protocol = protocol;
	m_port = port;
	Receive = receiveFunc;
}

MessageExchange::~MessageExchange()
{

}

