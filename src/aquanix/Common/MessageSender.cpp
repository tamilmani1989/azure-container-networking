#include "MessageSender.h"

MessageSender::MessageSender(MessageExchange *pMessageExchange, string *remoteHostAddresses, int remoteHostsCount)
{
	m_pMessageExchange = pMessageExchange;
	m_remoteHostAddresses = remoteHostAddresses;
	m_remoteHostsCount = remoteHostsCount;
	m_exitMessageSending = false;
	m_messageSendingStopped = true;
	thread sendMessagesThread(SendMessagesDaemon, this);
	sendMessagesThread.detach();
}

MessageSender::~MessageSender()
{
	m_pMessageExchange = NULL;
	m_remoteHostAddresses = NULL;
	m_remoteHostsCount = 0;
	//need to cleanup more stuff here
}

DWORD MessageSender::Send(Message message)
{
	DWORD result = E_SUCCESS;

	RawMessage *pRawMessage = message.CreateRawMessage();
	map<string, int>* pSendMap = new map<string, int>();
	for (int i = 0; i < m_remoteHostsCount; i++)
	{
		pSendMap->insert(pair<string, int>(m_remoteHostAddresses[i], 0));
	}
	tuple<RawMessage*, map<string, int>*>* messageMapTuple = new tuple<RawMessage*, map<string, int>*>(pRawMessage, pSendMap);
	const char *msgId = message.GetMessageId().c_str();
	UpdateMessageMap(msgId, messageMapTuple);
	m_conditionSendMessage.notify_one();

//AquaError:
	return result;
}

void MessageSender::UpdateMessageMap(const char *id, tuple<RawMessage*, map<string, int>*>* messageMapTuple)
{
	lock_guard<mutex> lockGuard(m_mapLock);
	m_messageMap[string(id)] = messageMapTuple;
}

void MessageSender::SendMessagesDaemon(MessageSender *pMessageSender)
{
	unique_lock<mutex> mlock(pMessageSender->m_mapLock);
	pMessageSender->m_messageSendingStopped = false;
	do
	{
		pMessageSender->m_conditionSendMessage.wait(mlock);
		if (pMessageSender->m_exitMessageSending)
		{
			pMessageSender->m_messageSendingStopped = true;
			pMessageSender->m_conditionSendingStopped.notify_all();
			break;
		}
		//process message
		pMessageSender->ProcessMessages();
		//pMessageSender->m_pMessageExchange->Send()
	} while (true);
}

void MessageSender::ProcessMessages()
{
	for (map<string, tuple<RawMessage*, map<string, int>*>*>::iterator it = m_messageMap.begin(); it != m_messageMap.end(); ++it)
	{
		string msgId = it->first;
		RawMessage *pRawMessage = get<0>(*it->second);

		map<string, int> *pRemoteHostsMap = get<1>(*it->second);
		for (map<string, int>::iterator it2 = pRemoteHostsMap->begin(); it2 != pRemoteHostsMap->end(); ++it2)
		{
			m_pMessageExchange->Send(it2->first.c_str(), (const char *)pRawMessage->GetData(), pRawMessage->GetDataSize());
		}
	}
}
