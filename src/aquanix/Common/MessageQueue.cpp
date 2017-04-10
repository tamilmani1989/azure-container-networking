#include "MessageQueue.h"


void MessageQueue::AddMessage(RawMessage *pMessage)
{
	lock_guard<mutex> lockGuard(m_mutex);
	m_messages.push(*pMessage);
	m_conditionProcessMessage.notify_one();
}

RawMessage *MessageQueue::TryGetMessage()
{
	lock_guard<mutex> lockGuard(m_mutex);
	RawMessage *pMessage = NULL;
	if (!m_messages.empty())
	{
		RawMessage message = m_messages.front();
		pMessage = &message;
	}
	return pMessage;
}

MessageQueue::MessageQueue()
{
	lock_guard<mutex> lockGuard(m_messageProcessingSyncLock);
	m_exitMessageProcessing = false;
	m_messageProcessingStopped = true;
	thread processMessagesThread(ProcessMessagesDaemon, this);
	processMessagesThread.detach();
}

void MessageQueue::ProcessMessagesDaemon(MessageQueue *pMessageQueue)
{
	unique_lock<mutex> mlock(pMessageQueue->m_messageProcessingSyncLock);
	pMessageQueue->m_messageProcessingStopped = false;
	do
	{
		pMessageQueue->m_conditionProcessMessage.wait(mlock);
		if (pMessageQueue->m_exitMessageProcessing)
		{
			pMessageQueue->m_messageProcessingStopped = true;
			pMessageQueue->m_conditionProcessingStopped.notify_all();
			break;
		}
		//process message
		pMessageQueue->ProcessMessages();
	} while (true);
}

void MessageQueue::ProcessMessages()
{
	RawMessage *pMessage = NULL;
	while ((pMessage = TryGetMessage()) != NULL)
	{
		//ProcessMessage(pMessage);
	}
}

void MessageQueue::StopMessageProcessing()
{
	unique_lock<mutex> mlock(m_messageProcessingSyncLock);
	if (!m_messageProcessingStopped)
	{
		m_exitMessageProcessing = true;
		m_conditionProcessMessage.notify_all();
		m_conditionProcessingStopped.wait(mlock);
	}
}

MessageQueue::~MessageQueue()
{
	StopMessageProcessing();
	RawMessage *pMessage = NULL;
	while ((pMessage = TryGetMessage()) != NULL)
	{
		delete pMessage;
	}
}