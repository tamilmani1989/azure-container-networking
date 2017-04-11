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
}

DWORD MessageSender::Send(Message message)
{
	LogMessage(L"Enter MessageSender::Send");

	DWORD result = E_SUCCESS;
	RawMessage *pRawMessage = message.CreateRawMessage();
	E_NULL(pRawMessage, E_CouldNotCreateRawMessage, "Failed to create raw message.");
	map<string, int>* pSendMap = new map<string, int>();
	for (int i = 0; i < m_remoteHostsCount; i++)
	{
		pSendMap->insert(pair<string, int>(m_remoteHostAddresses[i], 0));
	}
	tuple<RawMessage*, map<string, int>*>* messageMapTuple = new tuple<RawMessage*, map<string, int>*>(pRawMessage, pSendMap);
	const char *msgId = message.GetMessageId();
	UpdateMessageMap(msgId, messageMapTuple);
	m_conditionSendMessage.notify_one();

AquaError:
	LogMessage(L"Exit MessageSender::Send");
	return result;
}

DWORD MessageSender::SendAck(Message message, char *destination)
{
	LogMessage(L"Enter MessageSender::SendAck");
	DWORD result = E_SUCCESS;

	USHORT responsePort = message.GetMessageResponsePort();
	RawMessage *pRawMessage = message.CreateAckRawMessage();

	E_AQUA(m_pMessageExchange->Send(destination, responsePort, (const char *)pRawMessage->GetData(), pRawMessage->GetDataSize()), "Send ack failed");
	delete pRawMessage;

AquaError:
	LogMessage(L"Exit MessageSender::SendAck");
	return result;
}

void MessageSender::UpdateMessageMap(const char *id, tuple<RawMessage*, map<string, int>*>* messageMapTuple)
{
	lock_guard<mutex> lockGuard(m_mapLock);
	m_messageMap[string(id)] = messageMapTuple;
}
DWORD MessageSender::RemoveAckedMessage(Message message, char *destination)
{
	LogMessage(L"Enter MessageSender::RemoveAckedMessage");
	DWORD result = E_SUCCESS;
	RemoveAckedMessageFromMap(message.GetMessageId(), destination);
	//AquaError:
	LogMessage(L"Exit MessageSender::RemoveAckedMessage");
	return result;
}

void MessageSender::RemoveAckedMessageFromMap(string msgId, char *destination)
{
	LogMessage(L"Enter MessageSender::RemoveAckedMessageFromMap");
	lock_guard<mutex> lockGuard(m_mapLock);
	map<string, tuple<RawMessage*, map<string, int>*>*>::iterator it;

	it = m_messageMap.find(msgId);
	if (it == m_messageMap.end())
	{
		return;
	}
	map<string, int>* pPerServerMessageMap = get<1>(*it->second);

	map<string, int>::iterator it2;
	it2 = pPerServerMessageMap->find(destination);
	if (it2 == pPerServerMessageMap->end())
	{
		return;
	}
	pPerServerMessageMap->erase(it2);
	if (pPerServerMessageMap->size() == 0)
	{
		delete get<0>(*it->second);
		delete pPerServerMessageMap;
		delete it->second;
		m_messageMap.erase(it);
	}
	LogMessage(L"Exit MessageSender::RemoveAckedMessageFromMap");
}

void MessageSender::SendMessagesDaemon(MessageSender *pMessageSender)
{
	unique_lock<mutex> mlock(pMessageSender->m_mapLock);
	pMessageSender->m_messageSendingStopped = false;
	//pMessageSender->LogMessage(L"Enter SendMessagesDaemon");
	do
	{
		pMessageSender->m_conditionSendMessage.wait(mlock);
		//pMessageSender->LogMessage(L"SendMessagesDaemon notified");
		if (pMessageSender->m_exitMessageSending)
		{
			pMessageSender->LogMessage(L"SendMessagesDaemon stop notification");
			pMessageSender->m_messageSendingStopped = true;
			pMessageSender->m_conditionSendingStopped.notify_all();
			break;
		}
		pMessageSender->LogMessage(L"SendMessagesDaemon ProcessMessage");
		//process message
		pMessageSender->ProcessMessages();
		//pMessageSender->m_pMessageExchange->Send()
	} while (true);
	pMessageSender->LogMessage(L"Exit SendMessagesDaemon");
}

void MessageSender::ProcessMessages()
{
	LogMessage(L"Enter MessageSender::ProcessMessages");
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
	LogMessage(L"Exit MessageSender::ProcessMessages");
}
