#pragma once

#include <winsock2.h>
#include <Iphlpapi.h>

#include <vector>

#include <atlstr.h>

#include <sal.h>

#include "NetworkInterfaceExceptions.h"

namespace TSTestCore
{
	class NetworkInterface;

	typedef std::vector<byte> PhysicalAddressBytes;
	typedef std::vector<NetworkInterface*> NetworkInterfaces;
	typedef std::vector<char> IPAddress;
	typedef std::vector<IPAddress*> IPAddresses;

	class NetworkInterface
	{
	public:
		NetworkInterface(PIP_ADAPTER_ADDRESSES pAddapterAddresses);
		virtual ~NetworkInterface(void);

		// Class functions:
		static NetworkInterfaces& GetAllNetworkInterfaces(DWORD family);
		static void Free(void);
		static PhysicalAddressBytes& GeneratePhysicalAddress(void);
		static IPAddresses& MachineIPAddresses(void);
		static void FreeIPAddresses(void);
		static NetworkInterface* GetFirstDHCPv4EnabledAdapter(void);

		// Object functions
		DWORD GetInterfaceType(void);
		DWORD GetFlags(void);
		const IN_ADDR& GetInAddr(void) const;       // Throws NetworkInterfaceExceptions
		CString& GetAdapterIPAddress(void);
		PhysicalAddressBytes& GetPhysicalAddressBytes(void);
		CString& GetPhisicalAddressString(void);
		bool Dhcpv4Enabled(void);
		bool AdapterIsActive(void);

	private:
		// Class functions:
		static void RemoveAllNetworkInterfaces(void);
		static void GetAdaptersAddresses(DWORD family, DWORD flags, DWORD sizePointer);
		static void CreateNetworkInterfaceObjects(void);
		static LPVOID Alloc(SIZE_T size);
		static void Free(LPVOID pAddress);

		// Class variables:
		static NetworkInterfaces   s_networkInterfaces;
		static PIP_ADAPTER_ADDRESSES s_adapterAddresses;
		static const TCHAR s_addressSeparator = _T('-');
		static const CString s_formatter;
		static PhysicalAddressBytes _generatedPhysicalAddressBytes;
		static IPAddresses s_ipAddresses;

		// Object variables:
		PIP_ADAPTER_ADDRESSES _pNetworkInterface;
		CString _ipAddress;

		PhysicalAddressBytes _physicalAddressBytes;
		CString _physicalAddressString;

		CString _friendlyName;
	};
}

