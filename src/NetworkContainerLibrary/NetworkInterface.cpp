#include "NetworkInterface.h"

#include <Rpc.h>

namespace TSTestCore
{
	NetworkInterfaces NetworkInterface::s_networkInterfaces;
	PhysicalAddressBytes NetworkInterface::_generatedPhysicalAddressBytes;
	PIP_ADAPTER_ADDRESSES NetworkInterface::s_adapterAddresses = NULL;
	IPAddresses NetworkInterface::s_ipAddresses;

	const CString NetworkInterface::s_formatter(_T("%02X"));

	NetworkInterface::NetworkInterface(PIP_ADAPTER_ADDRESSES pAddapterAddresses)
	{
		_pNetworkInterface = pAddapterAddresses;

		DWORD physicalAddressLength = _pNetworkInterface->PhysicalAddressLength;
		for (DWORD i = 0; i < physicalAddressLength; ++i)
		{
			_physicalAddressBytes.push_back(_pNetworkInterface->PhysicalAddress[i]);
		}

		_physicalAddressString.Empty();
		CString temp;
		size_t size = _physicalAddressBytes.size();
		for (size_t i = 0; i < size; ++i)
		{
			temp.Format(s_formatter, _physicalAddressBytes[i]);
			_physicalAddressString += temp;
			_physicalAddressString += s_addressSeparator;
		}
		_physicalAddressString.Delete(_physicalAddressString.GetLength() - 1, 1);
	}

	NetworkInterface::~NetworkInterface() {}

	NetworkInterfaces& NetworkInterface::GetAllNetworkInterfaces(DWORD family)
	{
		RemoveAllNetworkInterfaces();

		DWORD flags = GAA_FLAG_INCLUDE_ALL_INTERFACES | GAA_FLAG_INCLUDE_PREFIX |
			GAA_FLAG_SKIP_ANYCAST | GAA_FLAG_SKIP_DNS_SERVER |
			GAA_FLAG_SKIP_MULTICAST;

		DWORD sizePointer = sizeof(IP_ADAPTER_ADDRESSES);

		GetAdaptersAddresses(family, flags, sizePointer);

		CreateNetworkInterfaceObjects();

		return s_networkInterfaces;
	}

	void NetworkInterface::RemoveAllNetworkInterfaces(void)
	{
		NetworkInterface* pNetworkInterface = NULL;
		size_t size = s_networkInterfaces.size();
		for (size_t i = 0; i < size; ++i)
		{
			pNetworkInterface = s_networkInterfaces[i];
			delete pNetworkInterface;
		}

		s_networkInterfaces.clear();
	}

	LPVOID NetworkInterface::Alloc(SIZE_T size)
	{
		LPVOID retVal = NULL;

		retVal = ::VirtualAlloc(
			NULL,
			size,
			MEM_COMMIT | MEM_RESERVE,
			PAGE_READWRITE
		);

		if (retVal == NULL)
		{
			throw (CannotAllocateMemory);
		}

		return retVal;
	}

	void NetworkInterface::Free(void)
	{
		Free(s_adapterAddresses);
	}

	void NetworkInterface::Free(LPVOID pAddress)
	{
		// Memory is allocated?
		if (pAddress != NULL)
		{
			BOOL rV = ::VirtualFree(
				pAddress,
				0,
				MEM_RELEASE
			);

			pAddress = NULL;
		}
	}

	void NetworkInterface::GetAdaptersAddresses(DWORD family, DWORD flags, DWORD sizePointer)
	{
		s_adapterAddresses = static_cast<PIP_ADAPTER_ADDRESSES> (Alloc(sizePointer));

		DWORD rV = ::GetAdaptersAddresses(
			family,
			flags,
			NULL,
			s_adapterAddresses,
			&sizePointer
		);

		switch (rV)
		{
		case ERROR_ADDRESS_NOT_ASSOCIATED:
			throw (AddressNotAssociated);
			break;
		case ERROR_BUFFER_OVERFLOW:
			// Have to allocate/call until success
			GetAdaptersAddresses(family, flags, sizePointer);
			break;
		case ERROR_INVALID_PARAMETER:
			throw (InvalidParameter);
			break;
		case ERROR_NOT_ENOUGH_MEMORY:
			throw (NotEnoughMemory);
			break;
		case ERROR_NO_DATA:
			throw (NoData);
			break;
		case ERROR_SUCCESS:
			break;
		default:
			break;
		}
	}

	void NetworkInterface::CreateNetworkInterfaceObjects(void)
	{
		// Have to save adapter information with the NetworkInterface object
		PIP_ADAPTER_ADDRESSES pAdapterAddress = s_adapterAddresses;
		NetworkInterface* pNetworkInterface = NULL;
		do
		{
			pNetworkInterface = new NetworkInterface(pAdapterAddress);
			s_networkInterfaces.push_back(pNetworkInterface);
			pAdapterAddress = pAdapterAddress->Next;
		} while (pAdapterAddress != NULL);
	}

	DWORD NetworkInterface::GetInterfaceType(void)
	{
		return _pNetworkInterface->IfType;
	}

	DWORD NetworkInterface::GetFlags(void)
	{
		return _pNetworkInterface->Flags;
	}

	CString& NetworkInterface::GetAdapterIPAddress(void)
	{
		_ipAddress.Empty();

		try
		{
			_ipAddress = CString(inet_ntoa(GetInAddr()));
		}
		catch (NetworkInterfaceExceptions& ex)
		{
			// We will return empty string here
		}

		return _ipAddress;
	}

	const IN_ADDR& NetworkInterface::GetInAddr(void) const
	{
		if (_pNetworkInterface->FirstUnicastAddress == NULL)
		{
			throw (BadPointer);
		}
		else
		{
			PSOCKADDR_IN pSockAddrIn = reinterpret_cast<PSOCKADDR_IN> (_pNetworkInterface->FirstUnicastAddress->Address.lpSockaddr);
			return pSockAddrIn->sin_addr;
		}
	}

	PhysicalAddressBytes& NetworkInterface::GetPhysicalAddressBytes(void)
	{
		return _physicalAddressBytes;
	}

	CString& NetworkInterface::GetPhisicalAddressString(void)
	{
		return _physicalAddressString;
	}

	PhysicalAddressBytes& NetworkInterface::GeneratePhysicalAddress(void)
	{
		_generatedPhysicalAddressBytes.clear();

		UUID uuid;
		::SecureZeroMemory(&uuid, sizeof(UUID));

		UuidCreate(&uuid);
		byte* pUUID = reinterpret_cast<byte*>(&uuid);

		for (int i = 0; i < sizeof(UUID); ++i)
		{
			_generatedPhysicalAddressBytes.push_back(pUUID[i]);
		}

		return _generatedPhysicalAddressBytes;
	}

	IPAddresses& NetworkInterface::MachineIPAddresses(void)
	{
		// Free IP Addresses
		FreeIPAddresses();

		// Initialize Winsock
		WSADATA wsaData;
		int nResult = WSAStartup(MAKEWORD(2, 2), &wsaData);

		hostent* pHostent = ::gethostbyname(NULL);
		if (pHostent == NULL)
		{
			// TODO: Report an error here
		}

		short addrLength = pHostent->h_length;
		char* ipAddr = NULL;

		// Save every IP Address
		for (int numberOfIPAddresses = 0; pHostent->h_addr_list[numberOfIPAddresses] != NULL; ++numberOfIPAddresses)
		{
			IPAddress* pIPAddress = new IPAddress();
			ipAddr = pHostent->h_addr_list[numberOfIPAddresses];
			// Save every IP Address byte
			for (int j = 0; j < addrLength; ++j)
			{
				pIPAddress->push_back(ipAddr[j]);
			}

			// Save new CAtlArray object
			s_ipAddresses.push_back(pIPAddress);
		}

		// TODO: Uninitialize WinSock
		WSACleanup();

		return s_ipAddresses;
	}

	void NetworkInterface::FreeIPAddresses(void)
	{
		IPAddress* pIPAddress = NULL;
		size_t count = s_ipAddresses.size();
		for (size_t i = 0; i < count; ++i)
		{
			pIPAddress = s_ipAddresses[i];
			pIPAddress->clear();
			delete pIPAddress;
		}

		s_ipAddresses.clear();
	}

	NetworkInterface* NetworkInterface::GetFirstDHCPv4EnabledAdapter(void)
	{
		bool networkInterfaceFound = false;
		NetworkInterface* pNetworkInterface = NULL;

		NetworkInterfaces networkInterfaces = NetworkInterface::GetAllNetworkInterfaces(AF_INET);
		size_t count = networkInterfaces.size();
		for (size_t i = 0; i < count; ++i)
		{
			pNetworkInterface = networkInterfaces[i];
			if (pNetworkInterface->Dhcpv4Enabled() && pNetworkInterface->AdapterIsActive())
			{
				networkInterfaceFound = true;
				break;
			}
		}

		if (!networkInterfaceFound)
		{
			throw (CantFindDHCPv4EnabledAdapter);
		}

		return pNetworkInterface;
	}

	bool NetworkInterface::Dhcpv4Enabled(void)
	{
		bool retVal = false;

		if ((_pNetworkInterface->Flags && IP_ADAPTER_DHCP_ENABLED) &&
			(_pNetworkInterface->Flags && IP_ADAPTER_IPV4_ENABLED))
		{
			retVal = true;
		}

		return retVal;
	}

	bool NetworkInterface::AdapterIsActive(void)
	{
		bool retVal = false;

		if (_pNetworkInterface->OperStatus == IfOperStatusUp)
		{
			retVal = true;
		}

		return retVal;
	}
}

