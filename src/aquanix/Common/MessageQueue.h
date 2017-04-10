#pragma once
#include "aquabase.h"
#include "RawMessage.h"
#include <queue>
#include <mutex>
#include <condition_variable>

class MessageQueue
{
private :
	queue<RawMessage> m_messages;
	mutex m_mutex;
	mutex m_messageProcessingSyncLock;
	condition_variable m_conditionProcessMessage;
	condition_variable m_conditionProcessingStopped;
	bool m_exitMessageProcessing;
	bool m_messageProcessingStopped;

	static void ProcessMessagesDaemon(MessageQueue *pMessageQueue);
	void StopMessageProcessing();
	void ProcessMessages();

public:
	MessageQueue();
	~MessageQueue();

	void AddMessage(RawMessage *pMessage);
	RawMessage* TryGetMessage();
};