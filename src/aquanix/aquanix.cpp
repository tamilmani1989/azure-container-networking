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
	char *mappingsSenderIP,
	AddRemoteMappingCallback receiveMapping,
	DeleteRemoteMappingCallback deleteMapping,
	GetLocalMappingsCallback getLocalMappings,
	FreeEndpointMappingsCallback freeEndpointMappings,
	USHORT servicePort, 
	USHORT remotePort)
{
	UNREFERENCED_PARAMETER(deleteMapping);
	WinLogger::LogInformation(L"InitializeAquaNix");

	DWORD result = E_SUCCESS;

	E_AQUA(AquaNixFactory::Create(
		localHostAddress, 
		remoteHostAddresses, 
		mappingsSenderIP, 
		receiveMapping, 
		deleteMapping,
		getLocalMappings,
		freeEndpointMappings,
		servicePort, 
		remotePort), "AquaNix create failed.");

AquaError:
	return result;
}

DllExport DWORD SendLocalMapping(const char *endpointId, const char *containerAddress, const char *macAddress)
{
	WinLogger::LogInformation(L"Enter SendLocalMapping");
	DWORD result = E_SUCCESS;
	
	AquaNix *pAquaNix = AquaNixFactory::GetInstance();
	E_AQUA(pAquaNix->SendMapping(string(endpointId), string(containerAddress), string(macAddress)), "AquaNix create failed.");

AquaError:
	WinLogger::LogInformation(L"Exit SendLocalMapping");
	return result;
}

DllExport DWORD DeleteLocalMapping(
	const char *endpointId)
{
	UNREFERENCED_PARAMETER(endpointId);
	WinLogger::LogInformation(L"Enter DeleteLocalMapping");
	DWORD result = E_SUCCESS;

	AquaNix *pAquaNix = AquaNixFactory::GetInstance();
	E_AQUA(pAquaNix->DeleteMapping(string(endpointId)), "AquaNix delete local mapping failed.");

AquaError:
	WinLogger::LogInformation(L"Exit DeleteLocalMapping");
	return result;
}

DllExport DWORD RequestRemoteMappings()
{
	WinLogger::LogInformation(L"Enter RequestRemoteMappings");
	DWORD result = E_SUCCESS;

	AquaNix *pAquaNix = AquaNixFactory::GetInstance();
	E_AQUA(pAquaNix->RequestRemoteMappings(), "AquaNix request remote mappings failed.");

AquaError:
	WinLogger::LogInformation(L"Exit RequestRemoteMappings");
	return result;
}

DWORD AquaNixFactory::Create(
	char *localHostAddress, 
	char *remoteHostAddresses[],
	char *mappingsSenderIP,
	AddRemoteMappingCallback receiveMapping,
	DeleteRemoteMappingCallback deleteRemoteMapping,
	GetLocalMappingsCallback getLocalMappings,
	FreeEndpointMappingsCallback freeEndpointMappings,
	USHORT servicePort, 
	USHORT remoteServicePort)
{
	WinLogger::LogInformation(L"Enter Create AquaNix");
	DWORD result = E_SUCCESS;

	E_NULL(localHostAddress, E_InvalidArguments, "localHostAddress is NULL");
	E_NULL(remoteHostAddresses, E_InvalidArguments, "remoteHostAddresses is NULL");
	E_NULL(receiveMapping, E_InvalidArguments, "receiveMapping is NULL");
	E_NULL(servicePort, E_InvalidArguments, "servicePort is NULL");
	E_NULL(remoteServicePort, E_InvalidArguments, "remoteServicePort is NULL");

	s_pAquaNix = new (nothrow) AquaNix(
		localHostAddress, 
		remoteHostAddresses, 
		mappingsSenderIP, 
		receiveMapping, 
		deleteRemoteMapping, 
		getLocalMappings,
		freeEndpointMappings,
		servicePort, 
		remoteServicePort);
	E_NULLPTR(s_pAquaNix, E_CreateAquaNixFailed, "CreateAquaNixFailed");

AquaError:
	WinLogger::LogInformation(L"Exit Create AquaNix");
	return result;
}

AquaNix *AquaNixFactory::GetInstance()
{
	return s_pAquaNix;
}

AquaNix::AquaNix(
	char *localHostAddress, 
	char *remoteHostAddresses[],
	char *mappingsSenderIP,
	AddRemoteMappingCallback receiveMapping, 
	DeleteRemoteMappingCallback deleteRemoteMapping,
	GetLocalMappingsCallback getLocalMappings,
	FreeEndpointMappingsCallback freeEndpointMappings,
	USHORT servicePort, 
	USHORT remoteServicePort)
{
	m_localHostAddress = string(localHostAddress);
	m_remoteHostsCount = sizeof(remoteHostAddresses) / sizeof(remoteHostAddresses[0]);
	m_remoteHostAddresses = new string[m_remoteHostsCount];
	for (int i = 0; i < m_remoteHostsCount; i++)
	{
		m_remoteHostAddresses[i] = string(remoteHostAddresses[i]);
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
}

DWORD AquaNix::SendMapping(string endpointId, string containerAddress, string macAddress)
{
	DWORD result = E_SUCCESS;

	Message message = Message(endpointId, m_localHostAddress, containerAddress, macAddress);
	E_AQUA(m_pMessageSender->Send(message),"Message send failed.");

AquaError:
	return result;
}

DWORD AquaNix::DeleteMapping(string endpointId)
{
	DWORD result = E_SUCCESS;

	Message message = Message(endpointId, m_localHostAddress);
	E_AQUA(m_pMessageSender->Send(message), "Message send failed.");

AquaError:
	return result;
}

DWORD AquaNix::RequestRemoteMappings()
{
	DWORD result = E_SUCCESS;

	Message message = Message(MessageCode::RequestAllRemoteMappings);
	E_AQUA(m_pMessageSender->Send(message), "Message send failed.");

AquaError:
	return result;
}

DWORD AquaNix::ReceiveMessage(char *sender, void *buffer, int bufLength)
{
	ENTER(L"AquaNix::ReceiveMessage");

	UNREFERENCED_PARAMETER(sender);
	EndpointMapping *endpointMappings = NULL;
	USHORT mappingsCount = 0;
	AquaNix *aquaNix = AquaNixFactory::GetInstance();

	Message message = Message((UCHAR *)buffer, (USHORT)bufLength);
	MessageCode messageCode = message.GetMessageCode();
	const char *messageId = message.GetMessageId();
	LogXAB(L"Received message for messageCode=%d MessageId:%S", messageCode, messageId);
	if (messageCode == MessageCode::AckMapping)
	{
		LogXAB(L"Received ack for messageCode=%d MessageId:%S", messageCode, messageId);
		aquaNix->m_pMessageSender->RemoveAckedMessage(message, sender);
	}
	else
	{
		const char *host = message.GetHostAddress();
		LogXAB(L"Received ack for messageCode=%d MessageId:%S", messageCode, messageId);
		if (aquaNix->m_mappingsSenderIP.compare(host) == 0)
		{
			LogX(L"Message received to be forwarded to remote servers");
			//need to send this to remote servers
			Message messageToForward = Message(message);
			E_AQUA(aquaNix->m_pMessageSender->Send(messageToForward), "Message send failed.");
			WinLogger::LogInformation(L"Message set to be forwarded to remote servers");
		}
		else
		{
			WinLogger::LogInformation(L"External receive callback");
			if (messageCode == MessageCode::SetMapping)
			{
				WinLogger::LogInformation(L"External recieve ...");
				const char *endpointId = message.GetEndpointId();
				const char *hostAddress = message.GetHostAddress();
				const char *containerAddress = message.GetContainerAddress();
				const char* macAddress = message.GetMacAddress();
				LogXA(L"Calling external receive mapping for MessageId:%S", messageId);
				aquaNix->ReceiveMapping(endpointId, hostAddress, containerAddress, macAddress);
				WinLogger::LogInformation(L"External receive mapping returned");
			}
			else if (messageCode == MessageCode::DeleteMapping)
			{
				WinLogger::LogInformation(L"External delete ...");
				aquaNix->DeleteRemoteMapping(message.GetEndpointId());
				WinLogger::LogInformation(L"External delete returned");
			}
			else if (messageCode == MessageCode::RequestAllRemoteMappings)
			{
				WinLogger::LogInformation(L"External GetLocalMappings 0 ...");
				E_ZERO(aquaNix->GetLocalMappings(&endpointMappings, &mappingsCount), "Get local mappings failed.", E_GetLocalMappingsFailed);
				WinLogger::LogInformation(L"External GetLocalMappings 0 returned");

				for (int i = 0; i < mappingsCount; i++)
				{
					Message specificMessage = Message(
						endpointMappings[i].endpointId,
						aquaNix->m_localHostAddress,
						endpointMappings[i].containerAddress,
						endpointMappings[i].macAddress);
					E_AQUA(aquaNix->m_pMessageSender->Send(specificMessage), "Message send failed.");
				}
				aquaNix->m_pMessageSender->NotifySender();
			}
			WinLogger::LogInformation(L"External receive callback returned");
		}
		WinLogger::LogInformation(L"Sending ack");
		aquaNix->m_pMessageSender->SendAck(message, sender);
		WinLogger::LogInformation(L"Ack sent");
	}

AquaError:

	if (endpointMappings != NULL)
	{
		aquaNix->FreeEndpointMappings(endpointMappings, mappingsCount);
		endpointMappings = NULL;
	}

	WinLogger::LogInformation(L"AquaNix::ReceiveMessage");
	return result;
}

AquaNix::~AquaNix()
{
	if (m_remoteHostAddresses != nullptr)
	{
		delete[] m_remoteHostAddresses;
		m_remoteHostAddresses = nullptr;
	}

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
