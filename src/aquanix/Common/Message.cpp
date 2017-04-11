#include "Message.h"
#include <winsock2.h>

extern "C"
{
#ifdef WIN32
#include <Rpc.h>
#else
#include <uuid/uuid.h>
#endif
}

string Message::CreateUuid()
{
#ifdef WIN32
	UUID uuid;
	UuidCreate(&uuid);

	unsigned char * str;
	UuidToStringA(&uuid, &str);

	std::string s((char*)str);

	RpcStringFreeA(&str);
#else
	uuid_t uuid;
	uuid_generate_random(uuid);
	char s[37];
	uuid_unparse(uuid, s);
#endif
	return s;
}

Message::Message(UCHAR *buffer, USHORT bufferLen)
{
	UNREFERENCED_PARAMETER(bufferLen);
	Init(buffer);
}

Message::Message(RawMessage rawMessage)
{
	Init((UCHAR *)rawMessage.GetData());
}


Message::Message(string endpointId, string hostAddress, string containerAddress, string macAddress)
{
	m_id = CreateUuid();
	m_messageCode = MessageCode::SetMapping;
	m_endpointId = endpointId;
	m_hostAddress = hostAddress;
	m_containerAddress = containerAddress;
	m_macAddress = macAddress;
	m_messageResponsePort = s_serviceResponsePort;
}

USHORT Message::s_serviceResponsePort = 0;

void Message::SetServiceResponsePort(USHORT serviceResponsePort)
{
	s_serviceResponsePort = serviceResponsePort;
}

void Message::Init(UCHAR *buffer)
{
	UCHAR *cur = buffer;
	m_messageCode = (MessageCode)*cur;
	cur += 1;

	USHORT nPort;
	char nPortSize = sizeof(nPort);
	memcpy(&nPort, cur, nPortSize);
	m_messageResponsePort = ntohs(nPort);
	cur += nPortSize;

	UCHAR segmentLength = *cur;
	cur++;
	m_id = string(reinterpret_cast<char*>(cur));
	cur += segmentLength;

	if (m_messageCode == MessageCode::AckMapping)
	{
		return;
	}

	segmentLength = *cur;
	cur++;
	m_endpointId = string(reinterpret_cast<char*>(cur));
	cur += segmentLength;

	segmentLength = *cur;
	cur++;
	m_containerAddress = string(reinterpret_cast<char*>(cur));
	cur += segmentLength;

	segmentLength = *cur;
	cur++;
	m_macAddress = string(reinterpret_cast<char*>(cur));
	cur += segmentLength;

	segmentLength = *cur;
	cur++;
	m_hostAddress = string(reinterpret_cast<char*>(cur));
	cur += segmentLength;
}

RawMessage *Message::CreateAckRawMessage()
{
	return CreateAckRawMessage(m_id.c_str(), (UCHAR)m_id.length());
}

RawMessage * Message::CreateAckRawMessage(const char *id, UCHAR idLen)
{
	USHORT bufSize = idLen + 2;//account for null termination and one char for size
	
	UCHAR msgCode = (UCHAR)MessageCode::AckMapping;
	char messageCodeSize = sizeof(UCHAR);
	bufSize += messageCodeSize;

	//message response port
	USHORT nPort = htons(m_messageResponsePort);
	char nPortSize = sizeof(nPort);
	bufSize += nPortSize;

	UCHAR *buffer = new UCHAR[bufSize];
	
	//copy msg code
	memcpy(buffer, &msgCode, messageCodeSize);
	UCHAR *cur = buffer + messageCodeSize;

	//copy msg response port in network byte order
	memcpy(cur, &nPort, nPortSize);
	cur += nPortSize;

	//copy msg Id with its size
	UCHAR segmentLength = (UCHAR)(idLen + 1);
	memcpy(cur, &segmentLength, 1);
	cur += 1;
	memcpy(cur, id, idLen + 1);

	return new RawMessage(buffer, bufSize, false);
}

RawMessage * Message::CreateRawMessage()
{
	if (m_messageCode == MessageCode::AckMapping)
	{
		return CreateAckRawMessage(m_id.c_str(), (UCHAR)m_id.length());
	}

	UCHAR bufSize = (UCHAR)m_id.length() + 2;//account for null termination and one char for size

	//message code
	UCHAR msgCode = (UCHAR)m_messageCode;
	char messageCodeSize = sizeof(msgCode);
	bufSize += messageCodeSize;

	//message response port
	USHORT nPort = htons(m_messageResponsePort);
	char nPortSize = sizeof(nPort);
	bufSize += nPortSize;

	//endpoint id
	bufSize += (UCHAR)m_endpointId.length() + 2;

	//container address
	bufSize += (UCHAR)m_containerAddress.length() + 2;

	//container mac
	bufSize += (UCHAR)m_macAddress.length() + 2;

	//host address
	bufSize += (UCHAR)m_hostAddress.length() + 2;


	//create buffer
	UCHAR *buffer = new UCHAR[bufSize];
	UCHAR *cur = buffer;
	UCHAR segmentLength;

	//copy msg code
	memcpy(cur, &msgCode, messageCodeSize);
	cur += messageCodeSize;

	//copy msg response port in network byte order
	memcpy(cur, &nPort, nPortSize);
	cur += nPortSize;

	//copy msg Id with its size
	segmentLength = (UCHAR)m_id.length() + 1;
	memcpy(cur, &segmentLength, 1);
	cur += 1;
	memcpy(cur, m_id.c_str(), segmentLength);
	cur += segmentLength;

	//copy endpointid with its size
	segmentLength = (UCHAR)m_endpointId.length() + 1;
	memcpy(cur, &segmentLength, 1);
	cur += 1;
	memcpy(cur, m_endpointId.c_str(), segmentLength);
	cur += segmentLength;

	//copy container address with its size
	segmentLength = (UCHAR)m_containerAddress.length() + 1;
	memcpy(cur, &segmentLength, 1);
	cur += 1;
	memcpy(cur, m_containerAddress.c_str(), segmentLength);
	cur += segmentLength;

	//copy m_macAddress with its size
	segmentLength = (UCHAR)m_macAddress.length() + 1;
	memcpy(cur, &segmentLength, 1);
	cur += 1;
	memcpy(cur, m_macAddress.c_str(), segmentLength);
	cur += segmentLength;

	//copy m_hostAddress with its size
	segmentLength = (UCHAR)m_hostAddress.length() + 1;
	memcpy(cur, &segmentLength, 1);
	cur += 1;
	memcpy(cur, m_hostAddress.c_str(), segmentLength);
	cur += segmentLength;

	return new RawMessage(buffer, bufSize, false);
}

const char *Message::GetMessageId()
{
	return m_id.c_str();
}

const char *Message::GetEndpointId()
{
	return m_endpointId.c_str();
}

const char *Message::GetHostAddress()
{
	return m_hostAddress.c_str();
}

const char *Message::GetContainerAddress()
{
	return m_containerAddress.c_str();
}

const char *Message::GetMacAddress()
{
	return m_macAddress.c_str();
}

void Message::SetHostAddress(string hostAddress)
{
	m_hostAddress = hostAddress;
}

MessageCode Message::GetMessageCode()
{
	return m_messageCode;
}

USHORT Message::GetMessageResponsePort()
{
	return m_messageResponsePort;
}
