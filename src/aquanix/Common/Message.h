#pragma once
#include "RawMessage.h"

namespace aqua
{
	enum MessageCode
	{
		SetMapping = 0,
		AckMapping = 1,
	};

	class Message
	{
	private:
		string m_id;
		MessageCode m_messageCode;
		string m_endpointId;
		string m_hostAddress;
		string m_containerAddress;
		string m_macAddress;
		string CreateUuid();
		void Message::Init(UCHAR *buffer);

	public:
		Message(UCHAR *buffer, USHORT bufferLen);
		Message(RawMessage rawMessage);
		Message(string endpointId, string hostAddress, string containerAddress, string macAddress);
		
		const char *GetMessageId();
		const char *GetEndpointId();
		const char *GetHostAddress();
		const char *GetContainerAddress();
		const char *GetMacAddress();
		void SetHostAddress(string hostAddress);
		MessageCode GetMessageCode();


		RawMessage *CreateRawMessage();
		RawMessage *CreateAckRawMessage();
		RawMessage *CreateAckRawMessage(const char *id, UCHAR idLen);
	};
}