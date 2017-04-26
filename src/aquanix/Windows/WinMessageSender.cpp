#include "WinMessageSender.h"
#include "WinLogger.h"


WinMessageSender::WinMessageSender(MessageExchange *pMessageExchange, string *remoteHostAddresses, int remoteHostsCount)
	:MessageSender(pMessageExchange, remoteHostAddresses, remoteHostsCount)
{

}

void WinMessageSender::LogMessage(wstring logMessage)
{
	WinLogger::LogInformation(logMessage);
}