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
#endif

AquaNix *AquaNixFactory::s_pAquaNix = NULL;

DllExport DWORD InitializeAquaNix(char *localHostAddress, char *remoteHostAddresses[], ReceiveMappingCallback receiveMapping, USHORT servicePort, USHORT remotePort)
{
	DWORD result = E_SUCCESS;

	E_AQUA(AquaNixFactory::Create(localHostAddress, remoteHostAddresses, receiveMapping, servicePort, remotePort), "AquaNix create failed.");

AquaError:
	return result;
}

DllExport DWORD SendMapping(char *endpointId, char *containerAddress, char *macAddress)
{
	DWORD result = E_SUCCESS;
	AquaNix *pAquaNix = AquaNixFactory::GetInstance();
	
	E_AQUA(pAquaNix->SendMapping(string(endpointId), string(containerAddress), string(macAddress)), "AquaNix create failed.");

AquaError:
	return result;
}

DWORD AquaNixFactory::Create(char *localHostAddress, char *remoteHostAddresses[], ReceiveMappingCallback receiveMapping, USHORT servicePort, USHORT remoteServicePort)
{
	DWORD result = E_SUCCESS;

	E_NULL(localHostAddress, E_InvalidArguments, "localHostAddress is NULL");
	E_NULL(remoteHostAddresses, E_InvalidArguments, "remoteHostAddresses is NULL");
	E_NULL(receiveMapping, E_InvalidArguments, "receiveMapping is NULL");
	E_NULL(servicePort, E_InvalidArguments, "servicePort is NULL");
	E_NULL(remoteServicePort, E_InvalidArguments, "remoteServicePort is NULL");

	s_pAquaNix = new (nothrow) AquaNix(localHostAddress, remoteHostAddresses, receiveMapping, servicePort, remoteServicePort);
	E_NULLPTR(s_pAquaNix, E_CreateAquaNixFailed, "CreateAquaNixFailed");

AquaError:
	return result;
}

AquaNix *AquaNixFactory::GetInstance()
{
	return s_pAquaNix;
}

AquaNix::AquaNix(char *localHostAddress, char *remoteHostAddresses[], ReceiveMappingCallback receiveMapping, USHORT servicePort, USHORT remoteServicePort)
{
	m_localHostAddress = string(localHostAddress);
	m_remoteHostsCount = sizeof(remoteHostAddresses) / sizeof(remoteHostAddresses[0]);
	m_remoteHostAddresses = new string[m_remoteHostsCount];
	for (int i = 0; i < m_remoteHostsCount; i++)
	{
		m_remoteHostAddresses[i] = string(remoteHostAddresses[i]);
	}

	ReceiveMapping = receiveMapping;
	m_servicePort = servicePort;
	m_remoteServicePort = remoteServicePort;

#ifdef _WIN32
	m_pMessageExchange = new WinMessageExchange(Protocol::Udp, servicePort, remoteServicePort, ReceiveMessage);
#endif

	// create the sender
	m_pMessageSender = new MessageSender(m_pMessageExchange, m_remoteHostAddresses, m_remoteHostsCount);

	//Initialize the exchange
	m_pMessageExchange->Initialize();
}

DWORD AquaNix::SendMapping(string endpointId, string containerAddress, string macAddress)
{
	DWORD result = E_SUCCESS;

	Message message = Message(endpointId, m_localHostAddress, containerAddress, macAddress);
	E_AQUA(m_pMessageSender->Send(message),"Message send failed.");

AquaError:
	return result;
}

DWORD AquaNix::ReceiveMessage(char *sender, void *buffer, int bufLength)
{
	UNREFERENCED_PARAMETER(sender);
	DWORD result = E_SUCCESS;
	Message message = Message((UCHAR *)buffer, (USHORT)bufLength);
	AquaNix *aquaNix = AquaNixFactory::GetInstance();
	const char *host = message.GetHostAddress();
	if (strcmp(host, "127.0.0.1") == 0)
	{
		//need to send this to remote servers
		message.SetHostAddress(aquaNix->m_localHostAddress);
		E_AQUA(aquaNix->m_pMessageSender->Send(message), "Message send failed.");
	}
	else
	{
		aquaNix->ReceiveMapping(message.GetEndpointId(), message.GetHostAddress(), message.GetContainerAddress(), message.GetMacAddress());
	}

AquaError:
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
