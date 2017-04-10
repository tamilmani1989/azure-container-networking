#pragma once
#include "aquabase.h"
#include "MessageExchange.h"
#include "Message.h"
#include <map>
#include <tuple>
#include <mutex>

namespace aqua
{
	class MessageSender
	{
	protected:
		MessageExchange *m_pMessageExchange;
		string *m_remoteHostAddresses;
		int m_remoteHostsCount;
		map<string, tuple<RawMessage*, map<string, int>*>*> m_messageMap;
		mutex m_mapLock;
		mutex m_messageSendSyncLock;
		condition_variable m_conditionSendMessage;
		condition_variable m_conditionSendingStopped;
		bool m_exitMessageSending;
		bool m_messageSendingStopped;


		void UpdateMessageMap(const char *, tuple<RawMessage*, map<string, int>*>* messageMapTuple);
		static void SendMessagesDaemon(MessageSender *pMessageSender);
		void ProcessMessages();
		void RemoveAckedMessageFromMap(string msgId, char *destination);

	public:
		MessageSender(MessageExchange *pMessageExchange, string *remoteHostAddresses, int remoteHostsCount);
		~MessageSender();

		DWORD Send(Message message);
		DWORD SendAck(Message message, char *destination);
		DWORD RemoveAckedMessage(Message message, char *destination);
	};
}