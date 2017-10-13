/****************************** Module Header ******************************\
* Module Name:  aquanix.cpp
* Project:      Aquarius Network Interexchange
* Copyright (c) Microsoft Corporation.
*
* This is the SDN Library that can be used to notify all hosts to exchange container information
*
\***************************************************************************/
#include "Common\aquanix.h"

#ifdef _WIN32
#include "Windows\WinMessageExchange.h"
#include "Windows\WinMessageSender.h"
#endif

AquaNix *AquaNixFactory::s_pAquaNix = NULL;

DllExport DWORD InitializeAquaNix(
	char *localHostAddress, 
	char *remoteHostAddresses[], 
	USHORT numberOfRemoteHosts,
	char *mappingsSenderIP,
	AddRemoteMappingCallback receiveMapping,
	DeleteRemoteMappingCallback deleteMapping,
	GetLocalMappingsCallback getLocalMappings,
	FreeEndpointMappingsCallback freeEndpointMappings,
	USHORT servicePort, 
	USHORT remotePort)
{
	ENTER(L"InitializeAquaNix");

	if (localHostAddress == NULL ||
		remoteHostAddresses == NULL ||
		receiveMapping == NULL ||
		deleteMapping == NULL ||
		getLocalMappings == NULL ||
		freeEndpointMappings == NULL)
	{
		E_AQ_FAIL(E_InvalidArguments, L"Invalid arguments passed");
	}

	E_AQUA(AquaNixFactory::Create(
		localHostAddress, 
		remoteHostAddresses, 
		numberOfRemoteHosts,
		mappingsSenderIP, 
		receiveMapping, 
		deleteMapping,
		getLocalMappings,
		freeEndpointMappings,
		servicePort, 
		remotePort), L"AquaNix create failed.");

AquaError:
	EXIT();
}

DllExport DWORD SetRemoteHostAddresses(char *remoteHostAddresses[], USHORT numberOfRemoteHosts)
{
	ENTER(L"SetRemoteHostAddresses");

	AquaNix *pAquaNix = AquaNixFactory::GetInstance();
	E_AQUA(pAquaNix->SetRemoteHosts(remoteHostAddresses, numberOfRemoteHosts), L"AquaNix SetRemoteHostAddresses failed.");

AquaError:
	EXIT();
}

DllExport DWORD SendLocalMapping(const char *endpointId, const char *containerAddress, const char *macAddress)
{
	ENTER(L"SendLocalMapping");
	
	AquaNix *pAquaNix = AquaNixFactory::GetInstance();
	E_AQUA(pAquaNix->SendMapping(string(endpointId), string(containerAddress), string(macAddress)), L"AquaNix create failed.");

AquaError:
	EXIT();
}

DllExport DWORD DeleteLocalMapping(
	const char *endpointId)
{
	ENTER(L"DeleteLocalMapping");

	AquaNix *pAquaNix = AquaNixFactory::GetInstance();
	E_AQUA(pAquaNix->DeleteMapping(string(endpointId)), L"AquaNix delete local mapping failed.");

AquaError:
	EXIT();
}

DllExport DWORD RequestRemoteMappings()
{
	ENTER(L"RequestRemoteMappings");

	AquaNix *pAquaNix = AquaNixFactory::GetInstance();
	E_AQUA(pAquaNix->RequestRemoteMappings(), L"AquaNix request remote mappings failed.");

AquaError:
	EXIT();
}

DWORD AquaNixFactory::Create(
	char *localHostAddress, 
	char *remoteHostAddresses[],
	USHORT numberOfRemoteHosts,
	char *mappingsSenderIP,
	AddRemoteMappingCallback receiveMapping,
	DeleteRemoteMappingCallback deleteRemoteMapping,
	GetLocalMappingsCallback getLocalMappings,
	FreeEndpointMappingsCallback freeEndpointMappings,
	USHORT servicePort, 
	USHORT remoteServicePort)
{
	ENTER(L"AquaNixFactory::Create");

	E_NULL(localHostAddress, E_InvalidArguments, L"localHostAddress is NULL");
	E_NULL(remoteHostAddresses, E_InvalidArguments, L"remoteHostAddresses is NULL");
	E_NULL(receiveMapping, E_InvalidArguments, L"receiveMapping is NULL");
	E_NULL(servicePort, E_InvalidArguments, L"servicePort is NULL");
	E_NULL(remoteServicePort, E_InvalidArguments, L"remoteServicePort is NULL");

	s_pAquaNix = new (nothrow) AquaNix(
		localHostAddress, 
		remoteHostAddresses, 
		numberOfRemoteHosts,
		mappingsSenderIP, 
		receiveMapping, 
		deleteRemoteMapping, 
		getLocalMappings,
		freeEndpointMappings,
		servicePort, 
		remoteServicePort);
	E_NULLPTR(s_pAquaNix, E_CreateAquaNixFailed, L"CreateAquaNixFailed");

AquaError:
	EXIT();
}

AquaNix *AquaNixFactory::GetInstance()
{
	return s_pAquaNix;
}

AquaNix::AquaNix(
	char *localHostAddress,
	char *remoteHostAddresses[],
	USHORT numberOfRemoteHosts,
	char *mappingsSenderIP,
	AddRemoteMappingCallback receiveMapping,
	DeleteRemoteMappingCallback deleteRemoteMapping,
	GetLocalMappingsCallback getLocalMappings,
	FreeEndpointMappingsCallback freeEndpointMappings,
	USHORT servicePort,
	USHORT remoteServicePort)
{
	ENTER(L"AquaNix::AquaNix");

	m_localHostAddress = string(localHostAddress);
	m_remoteHostsCount = numberOfRemoteHosts;
	LogXAB(L"localHostAddress:%S RemoteHostsCount:%d", localHostAddress, m_remoteHostsCount);

	if (m_remoteHostsCount > 0)
	{
		m_remoteHostAddresses = new string[m_remoteHostsCount];
		for (int i = 0; i < m_remoteHostsCount; i++)
		{
			m_remoteHostAddresses[i] = string(remoteHostAddresses[i]);
			LogXA(L"Added remote IP:%S", m_remoteHostAddresses[i].c_str());
		}
	}
	else
	{
		m_remoteHostAddresses = NULL;
	}

	if (mappingsSenderIP != NULL)
	{
		m_mappingsSenderIP = string(mappingsSenderIP);
	}

	ReceiveMapping = receiveMapping;
	DeleteRemoteMapping = deleteRemoteMapping;
	GetLocalMappings = getLocalMappings;
	FreeEndpointMappings = freeEndpointMappings;
	m_servicePort = servicePort;
	m_remoteServicePort = remoteServicePort;
	Message::SetLocalSettings(servicePort, m_localHostAddress);

#ifdef _WIN32
	m_pMessageExchange = new WinMessageExchange(m_localHostAddress, Protocol::Udp, servicePort, remoteServicePort, ReceiveMessage);
#endif

	//Initialize the exchange
	m_pMessageExchange->Initialize();

#ifdef _WIN32
	// create the sender
	m_pMessageSender = new WinMessageSender(m_pMessageExchange, m_remoteHostAddresses, m_remoteHostsCount);
#endif

	EXITNORETURN();
}

DWORD AquaNix::SendMapping(string endpointId, string containerAddress, string macAddress)
{
	ENTER(L"AquaNix::SendMapping");

	Message message = Message(endpointId, m_localHostAddress, containerAddress, macAddress);
	E_AQUA(m_pMessageSender->Send(message), L"Message send failed.");

AquaError:
	EXIT();
}

DWORD AquaNix::SetRemoteHosts(char *remoteHostAddresses[], USHORT numberOfRemoteHosts)
{
	m_RemoteHostsMutex.lock();
	
	ENTER(L"AquaNix::SetRemoteHosts");
	
	// free existing if any.
	LogXA(L"Existing RemoteHostsCount was:%d", m_remoteHostsCount);
	if (m_remoteHostsCount > 0)
	{
		if(m_remoteHostAddresses != NULL)
		{
			free(m_remoteHostAddresses);
		}
	}

	m_remoteHostsCount = numberOfRemoteHosts;
	LogXA(L"New RemoteHostsCount:%d", m_remoteHostsCount);

	m_remoteHostAddresses = new string[m_remoteHostsCount];
	for (int i = 0; i < m_remoteHostsCount; i++)
	{
		m_remoteHostAddresses[i] = string(remoteHostAddresses[i]);
		LogXA(L"Added remote IP:%S", m_remoteHostAddresses[i].c_str());
	}

	E_AQUA(m_pMessageSender->SetRemoteHostsForSendingMessages(m_remoteHostAddresses,
															  m_remoteHostsCount), 
															  L"SetRemoteHostsForSendingMessages failed");

AquaError:
	m_RemoteHostsMutex.unlock();
	EXIT();
}

DWORD AquaNix::DeleteMapping(string endpointId)
{
	ENTER(L"AquaNix::DeleteMapping");

	Message message = Message(endpointId, m_localHostAddress);
	E_AQUA(m_pMessageSender->Send(message), L"Message send failed.");

AquaError:
	EXIT();
}

DWORD AquaNix::RequestRemoteMappings()
{
	ENTER(L"AquaNix::RequestRemoteMappings");

	Message message = Message(MessageCode::RequestAllRemoteMappings);
	E_AQUA(m_pMessageSender->Send(message), L"Message send failed.");

AquaError:
	EXIT();
}

DWORD AquaNix::ReceiveMessage(char *sender, void *buffer, int bufLength)
{
	ENTER(L"AquaNix::ReceiveMessage");

	EndpointMapping *endpointMappings = NULL;
	USHORT mappingsCount = 0;
	AquaNix *aquaNix = AquaNixFactory::GetInstance();

	Message message = Message((UCHAR *)buffer, (USHORT)bufLength);
	MessageCode messageCode = message.GetMessageCode();
	const char *messageId = message.GetMessageId();
	LogXABC(L"Received message for messageCode=%d MessageId:%S from:%S", messageCode, messageId, sender);
	if (messageCode == MessageCode::AckMapping)
	{
		LogXAB(L"Received ack for messageCode=%d MessageId:%S", messageCode, messageId);
		aquaNix->m_pMessageSender->RemoveAckedMessage(message, sender);
	}
	else
	{
		const char *host = message.GetHostAddress();
		USHORT msgResponsePort = message.GetMessageResponsePort();
		LogXABC(L"Received non-ack message for messageCode=%d MessageId:%S msgResponsePort:%d", messageCode, messageId, msgResponsePort);
		if (aquaNix->m_mappingsSenderIP.compare(host) == 0)
		{
			LogX(L"Message received to be forwarded to remote servers");
			//need to send this to remote servers
			Message messageToForward = Message(message);
			E_AQUA(aquaNix->m_pMessageSender->Send(messageToForward), L"Message send failed.");
			WinLogger::LogInformation(L"Message set to be forwarded to remote servers");
		}
		else
		{
			WinLogger::LogInformation(L"External receive callback");
			if (messageCode == MessageCode::SetMapping)
			{
				const char *endpointId = message.GetEndpointId();
				const char *hostAddress = message.GetHostAddress();
				const char *containerAddress = message.GetContainerAddress();
				const char *macAddress = message.GetMacAddress();
				LogXABCDE(L"Calling external receive mapping for MessageId:%S endpointId:%S hostAddress:%S containerAddress:%S macAddress:%S", 
					messageId, endpointId, hostAddress, containerAddress, macAddress);
				aquaNix->ReceiveMapping(endpointId, hostAddress, containerAddress, macAddress);
				WinLogger::LogInformation(L"External receive mapping returned");
			}
			else if (messageCode == MessageCode::DeleteMapping)
			{
				const char *endpointId = message.GetEndpointId();
				const char *hostAddress = message.GetHostAddress();
				LogXABC(L"Calling external delete mapping for MessageId:%S endpointId:%S hostAddress:%S",
					messageId, endpointId, hostAddress);
				aquaNix->DeleteRemoteMapping(message.GetEndpointId());
				WinLogger::LogInformation(L"External delete returned");
			}
			else if (messageCode == MessageCode::RequestAllRemoteMappings)
			{
				LogX(L"External GetLocalMappings callback...");
				E_ZERO(aquaNix->GetLocalMappings(&endpointMappings, &mappingsCount), L"Get local mappings failed.", E_GetLocalMappingsFailed);
				LogXA(L"External GetLocalMappings returned %d mappings", mappingsCount);
				if (mappingsCount > 0)
				{
					for (int i = 0; i < mappingsCount; i++)
					{
						LogX(L"Creating send message for local mapping to be sent to remote server.");
						Message specificMessage = Message(
							endpointMappings[i].endpointId,
							aquaNix->m_localHostAddress,
							endpointMappings[i].containerAddress,
							endpointMappings[i].macAddress);
						E_AQUA(aquaNix->m_pMessageSender->Send(specificMessage, sender, true), L"Message send failed.");
					}
					aquaNix->m_pMessageSender->NotifySender();
				}
			}
			WinLogger::LogInformation(L"External receive callback returned");
		}
		aquaNix->m_pMessageSender->SendAck(message, sender);
	}

AquaError:

	if (endpointMappings != NULL)
	{
		LogXA(L"External FreeEndpointMappings for %d mappings", mappingsCount);
		aquaNix->FreeEndpointMappings(endpointMappings, mappingsCount);
		LogX(L"External FreeEndpointMappings completed");
		endpointMappings = NULL;
	}

	EXIT();
}

AquaNix::~AquaNix()
{
	m_RemoteHostsMutex.lock();
	if (m_remoteHostAddresses != nullptr)
	{
		delete[] m_remoteHostAddresses;
		m_remoteHostAddresses = nullptr;
	}
	m_RemoteHostsMutex.unlock();

	if (m_pMessageExchange != NULL)
	{
		delete m_pMessageExchange;
		m_pMessageExchange = NULL;
	}

	if (m_pMessageSender != NULL)
	{
		delete m_pMessageSender;
		m_pMessageSender = NULL;
	}
}
