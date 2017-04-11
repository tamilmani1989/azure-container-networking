#pragma once
#include "aquanixext.h"
#include "MessageExchange.h"
#include "MessageSender.h"

namespace aqua
{
	class AquaNix
	{
	private:
		string m_localHostAddress;
		string *m_remoteHostAddresses;
		int m_remoteHostsCount;
		AddRemoteMappingCallback ReceiveMapping;
		DeleteRemoteMappingCallback DeleteRemoteMapping;
		USHORT m_servicePort;
		USHORT m_remoteServicePort;
		MessageExchange *m_pMessageExchange;
		MessageSender *m_pMessageSender;
		static DWORD ReceiveMessage(char *sender, void *buffer, int bufLength);

	public:
		AquaNix(char *localHostAddress, char *remoteHostAddresses[], AddRemoteMappingCallback receiveMapping, DeleteRemoteMappingCallback deleteRemoteMapping, USHORT servicePort, USHORT remoteServicePort);
		~AquaNix();

		DWORD SendMapping(string endpointId, string containerAddress, string macAddress);
		DWORD DeleteMapping(string endpointId);
	};

	class AquaNixFactory
	{
	private:
		static AquaNix *s_pAquaNix;
	public:
		static DWORD Create(char *localHostAddress, char *remoteHostAddresses[], AddRemoteMappingCallback receiveMapping, DeleteRemoteMappingCallback deleteRemoteMapping, USHORT servicePort, USHORT remotePort);
		static AquaNix *GetInstance();
		//static DWORD Cleanup();
	};
}