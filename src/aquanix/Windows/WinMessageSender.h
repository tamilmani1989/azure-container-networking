#pragma once
#include "..\Common\MessageSender.h"

class WinMessageSender : public MessageSender
{
public:
	WinMessageSender(MessageExchange *pMessageExchange, string *remoteHostAddresses, int remoteHostsCount);
	virtual void LogMessage(wstring logMessage);
};
