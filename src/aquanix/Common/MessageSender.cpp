#include "MessageSender.h"

#include "..\Windows\WinLogger.h"

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

DWORD MessageSender::SetRemoteHostsForSendingMessages(string *remoteHostAddresses, int remoteHostsCount)
{
	m_RemoteHostsMutex.lock();
	m_remoteHostAddresses = remoteHostAddresses;
	m_remoteHostsCount = remoteHostsCount;
	m_RemoteHostsMutex.unlock();
	return 0;
}

DWORD MessageSender::Send(Message &message)
{
	ENTER(L"MessageSender::Send");

	const char *messageId = message.GetMessageId();
	const char *endpointId = message.GetEndpointId();
	const char *hostAddress = message.GetHostAddress();
	const char *containerAddress = message.GetContainerAddress();
	const char *macAddress = message.GetMacAddress();
	USHORT msgResponsePort = message.GetMessageResponsePort();
	LogXABCDEF(L"Adding send message for MessageId:%S msgResponsePort:%d endpointId:%S hostAddress:%S containerAddress:%S macAddress:%S",
		messageId, msgResponsePort, endpointId, hostAddress, containerAddress, macAddress);

	RawMessage *pRawMessage = message.CreateRawMessage();
	E_NULL(pRawMessage, E_CouldNotCreateRawMessage, L"Failed to create raw message.");
	map<string, int>* pSendMap = new map<string, int>();
	m_RemoteHostsMutex.lock();
	for (int i = 0; i < m_remoteHostsCount; i++)
	{
		LogXA(L"Adding send message for remote host:%S", m_remoteHostAddresses[i].c_str());
		pSendMap->insert(pair<string, int>(m_remoteHostAddresses[i], 0));
	}
	m_RemoteHostsMutex.unlock();
	tuple<RawMessage*, map<string, int>*>* messageMapTuple = new tuple<RawMessage*, map<string, int>*>(pRawMessage, pSendMap);
	const char *msgId = message.GetMessageId();
	UpdateMessageMap(msgId, messageMapTuple);
	m_conditionSendMessage.notify_one();

AquaError:
	EXIT();
}

DWORD MessageSender::Send(
	Message &message,
	char *recipientIPAddress,
	bool notifyLater)
{
	ENTER(L"MessageSender::Send");

	RawMessage *pRawMessage = message.CreateRawMessage();
	E_NULL(pRawMessage, E_CouldNotCreateRawMessage, L"Failed to create raw message.");
	map<string, int>* pSendMap = new map<string, int>();
	pSendMap->insert(pair<string, int>(string(recipientIPAddress), 0));
	tuple<RawMessage*, map<string, int>*>* messageMapTuple = new tuple<RawMessage*, map<string, int>*>(pRawMessage, pSendMap);
	const char *msgId = message.GetMessageId();
	UpdateMessageMap(msgId, messageMapTuple);
	if (!notifyLater)
	{
		m_conditionSendMessage.notify_one();
	}

AquaError:
	EXIT();
}

void MessageSender::NotifySender()
{
	m_conditionSendMessage.notify_one();
}

DWORD MessageSender::SendAck(Message &message, char *destination)
{
	ENTER(L"MessageSender::SendAck");
	RawMessage *pRawMessage = NULL;

	USHORT responsePort = message.GetMessageResponsePort();
	const char *messageId = message.GetMessageId();
	LogXABC(L"Sending ack for MessageId:%S on responsePort: %d to %S", messageId, responsePort, destination);

	pRawMessage = message.CreateAckRawMessage();
	E_AQUA(m_pMessageExchange->Send(destination, responsePort, (const char *)pRawMessage->GetData(), pRawMessage->GetDataSize()), 
		L"Send ack failed");

AquaError:
	if (pRawMessage != NULL)
	{
		delete pRawMessage;
		pRawMessage = NULL;
	}

	EXIT();
}

void MessageSender::UpdateMessageMap(const char *id, tuple<RawMessage*, map<string, int>*>* messageMapTuple)
{
	lock_guard<mutex> lockGuard(m_mapLock);
	m_messageMap[string(id)] = messageMapTuple;
}

DWORD MessageSender::RemoveAckedMessage(Message &message, char *destination)
{
	ENTER(L"MessageSender::RemoveAckedMessage");

	const char *messageId = message.GetMessageId();
	LogXAB(L"Removing send message for messageId:%S destination:%S", messageId, destination);
	RemoveAckedMessageFromMap(messageId, destination);
	//AquaError:
	LogMessage(L"Exit MessageSender::RemoveAckedMessage");
	return result;
}

DWORD MessageSender::RemoveAckedMessageFromMap(string msgId, char *destination)
{
	ENTER(L"MessageSender::RemoveAckedMessageFromMap");
	lock_guard<mutex> lockGuard(m_mapLock);
	map<string, tuple<RawMessage*, map<string, int>*>*>::iterator it;

	it = m_messageMap.find(msgId);
	if (it == m_messageMap.end())
	{
		return E_SUCCESS;
	}
	map<string, int>* pPerServerMessageMap = get<1>(*it->second);

	map<string, int>::iterator it2;
	it2 = pPerServerMessageMap->find(destination);
	if (it2 == pPerServerMessageMap->end())
	{
		return E_SUCCESS;
	}
	LogXA(L"Erasing server:%S from send msg map", destination);
	pPerServerMessageMap->erase(it2);
	if (pPerServerMessageMap->size() == 0)
	{
		LogXA(L"Erasing msgid:%S from send msg map", msgId.c_str());
		delete get<0>(*it->second);
		delete pPerServerMessageMap;
		delete it->second;
		m_messageMap.erase(it);
	}
	EXIT();
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
