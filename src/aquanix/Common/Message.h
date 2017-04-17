#pragma once
#include "RawMessage.h"

namespace aqua
{
	enum MessageCode
	{
		SetMapping = 0,
		AckMapping = 1,
		DeleteMapping = 2,
		RequestAllRemoteMappings = 3,
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

		static USHORT s_serviceResponsePort;
		static string s_localHostAddress;
		USHORT m_messageResponsePort;

		string CreateUuid();
		void Message::Init(UCHAR *buffer);

	public:
		Message(const Message& message);
		Message(MessageCode messageCode);
		Message(UCHAR *buffer, USHORT bufferLen);
		Message(RawMessage rawMessage);
		Message(string endpointId, string hostAddress, string containerAddress, string macAddress);
		Message(string endpointId, string hostAddress);

		const char *GetMessageId();
		const char *GetEndpointId();
		const char *GetHostAddress();
		const char *GetContainerAddress();
		const char *GetMacAddress();
		void SetHostAddress(string hostAddress);
		MessageCode GetMessageCode();
		USHORT GetMessageResponsePort();
		static void SetLocalSettings(USHORT serviceResponsePort, string localHostAddress);


		RawMessage *CreateRawMessage();
		RawMessage *CreateAckRawMessage();
		RawMessage *CreateAckRawMessage(const char *id, UCHAR idLen);
		RawMessage *CreateSendMappingRawMessage();
		RawMessage *CreateDeleteMappingRawMessage();
		RawMessage *CreateRequestAllRemoteMappingsRawMessage();
	};
}