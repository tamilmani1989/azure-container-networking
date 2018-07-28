
/*++

Copyright (c) 2008  Microsoft Corporation

Module Name:

ServiceTunnelConfigurator.cpp

Abstract:

Implements methods related to configuration of service tunnel adapter.

--*/

//
// ------------------------------------------------------------------- Includes
//
#include "ServiceTunnelLibrary.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

HRESULT
ServiceTunnelConfigurator::NetIfEnableWeakHostRouting(
	__in ULONG IfIndex)
	/*++

	Routine Description:

	This routine enables weak host send and receive on specified interface.

	Arguments:

	IfIndex - IfIndex of interface to enable weak host send/receive on.

	Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

	--*/
{
	DWORD error;
	PWSTR netshCommand = NULL;
	HRESULT hr = S_OK;
	DWORD exitCode;


	//
	// Enable weak host routing.
	//

	error = FormatString(&netshCommand,
		0,
		L"netsh interface ipv6 set interface %lu weakhostreceive=enable weakhostsend=enable",
		IfIndex);
	if (error != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(error);
		LOGERRW(L"NetIfEnableWeakHostRouting::FormatString Failing HRESULT status detected %x", hr);
		goto Cleanup;
	}

	LOGINFOW(L"Running cmd %s", netshCommand);
	hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
	if (hr != S_OK)
	{
		LOGERRW(L"Command failed ('%ls')", netshCommand);
		goto Cleanup;
	}

	if (exitCode != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(exitCode);
		LOGERRW(L"Command returned non zero exit code %u ('%ls').", exitCode, netshCommand);
		goto Cleanup;
	}

Cleanup:

	STR_FREE(netshCommand);

	return hr;
}

VOID
NetpIpv6AdressChangeCallback(
	__in PVOID Context,
	__in PMIB_UNICASTIPADDRESS_ROW Row,
	__in MIB_NOTIFICATION_TYPE NotificationType
)
/*++

Routine Description:

This function is called when a unicast IP address notification is received.

Arguments:

Context - The CallerContext parameter that is passed to the
NotifyUnicastIpAddressChange function when it is registering the driver
for change notifications.

Row -  A pointer to the MIB_UNICASTIPADDRESS_ROW entry for the unicast IP
address that was changed. This parameter is a NULL pointer when the
MIB_NOTIFICATION_TYPE value that is passed in the NotificationType
parameter to the callback function is set to MibInitialNotification.
This situation can occur only if the InitialNotification parameter that
is passed to NotifyUnicastIpAddressChange was set to TRUE when
registering the driver for change notifications.

NotificationType - The notification type. This member can be one of the
values from the MIB_NOTIFICATION_TYPE enumeration type.

Return Value:

None.

--*/
{
	PNET_UNICAST_IPv6_ADDRESS_CHANGE_CONTEXT changeContext = NULL;

	changeContext = (PNET_UNICAST_IPv6_ADDRESS_CHANGE_CONTEXT)Context;

	//
	// Check if a new IP was added.
	//

	if (NotificationType == MibAddInstance) {

		//
		// Is this our IP?
		//

		if (0 == memcmp(&Row->Address.Ipv6.sin6_addr,
			&changeContext->Ipv6Address,
			sizeof(changeContext->Ipv6Address))) {

			//
			// Set the event.
			//

			SetEvent(changeContext->Event);
		}
	}
}



HRESULT
ServiceTunnelConfigurator::NetIfSetIpv6Address(
	__in ULONG IfIndex,
	__in PCWSTR Ipv6Address
)
/*++

Routine Description:

This routine sets the IPv6 address on specified interface.

N.B. : This code assumes WSAStartup has already been called.

Arguments:

IfIndex - Interface Index for the interface on which to set IP address.

Ipv6Address - IpAddress to set on the Physical Interface.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;
	DWORD error;
	DWORD exitCode;
	USHORT port = 0;
	ULONG scopeId = 0;
	PWSTR netshCommand = NULL;
	HANDLE changeEvent = NULL;
	HANDLE notification = NULL;
	IN6_ADDR ipv6Address = { 0 };
	DWORD result = WAIT_OBJECT_0;
	NET_UNICAST_IPv6_ADDRESS_CHANGE_CONTEXT changeContext = { 0 };

	if (Ipv6Address == NULL) {
		hr = HRESULT_FROM_WIN32(ERROR_INVALID_PARAMETER);
		LOGERRW(L"NetIfSetIpv6Address::IPV6Address is null. Failing HRESULT status detected %x", hr);
		goto Cleanup;
	}

	LOGINFOW(L"NetIfSetIpv6Address: Index:%lu, IP:%ls", IfIndex, Ipv6Address);

	//
	// Create a synchronization event.
	//

	changeEvent = CreateEvent(NULL, TRUE, FALSE, NULL);

	if (changeEvent == NULL) {
		DWORD err = GetLastError();
		if (err != ERROR_SUCCESS)
		{
			LOGERRW(L"CreateEvent GetLastError detected Error[0x] %x", err);
			hr = HRESULT_FROM_WIN32(err);
			goto Cleanup;
		}
	}

	//
	// Register the change notification.
	//

	error = RtlIpv6StringToAddressEx(Ipv6Address, &ipv6Address, &scopeId, &port);

	if (error != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_NT(error);
		if (hr != S_OK)
		{
			LOGERRW(L"NetIfSetIpv6Address:: RtlIpv6StringToAddressEx Failing HRESULT status detected %x", hr);
			goto Cleanup;
		}
	}

	changeContext.Event = changeEvent;
	changeContext.Ipv6Address = ipv6Address;

	error = NotifyUnicastIpAddressChange(AF_INET,
		NetpIpv6AdressChangeCallback,
		(PVOID)&changeContext,
		FALSE,
		&notification);
	if (error != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(error);
		LOGERRW(L"Failed to register for IP address change notification error code %d hr %x", error, hr);
		goto Cleanup;
	}

	//
	// Set the IP address.
	//

	error = FormatString(&netshCommand,
		0,
		L"netsh interface ipv6 add address interface=%lu address=%ls store=persistent",
		IfIndex,
		Ipv6Address);
	if (error != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(error);
		LOGERRW(L"NetIfSetIpv6Address:: FormatString Add address Failing HRESULT status detected %x", hr);
		goto Cleanup;
	}

	LOGINFOW(L"Applying network config: %ls", netshCommand);

	hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
	if (hr != S_OK)
	{
		LOGERRW(L"Command failed ('%ls').", netshCommand);
		goto Cleanup;
	}

	if (exitCode != ERROR_SUCCESS)
	{
		hr = E_UNEXPECTED;
		LOGERRW(L"Command returned non zero exit code %u ('%ls').", exitCode, netshCommand);
		goto Cleanup;
	}

	//
	// Wait for IP address change notification.
	//

	LOGINFOW(L"Waiting for IP address to be updated.");

	result = WaitForSingleObject(changeEvent, NET_UNICAST_IP_ADDRESS_CHANGE_TIMEOUT);
	if (result != WAIT_OBJECT_0) {
		if (GetLastError() != ERROR_SUCCESS)
		{
			hr = HRESULT_FROM_WIN32(GetLastError());
			LOGERRW(L"Wait for IP address change notification failed getlasterror %d", GetLastError());
			goto Cleanup;
		}
	}

	LOGINFOW(L"Successfully updated IP address.");

Cleanup:

	STR_FREE(netshCommand);

	if (notification) {
		CancelMibChangeNotify2(notification);
		notification = NULL;
	}

	if (changeEvent) {
		CloseHandle(changeEvent);
		changeEvent = NULL;
	}

	return hr;

}

HRESULT
ServiceTunnelConfigurator::NetIfAddIPv6Route(
	__in ULONG IfIndex,
	__in PCWSTR Ipv6AddressPrefix,
	__in PCWSTR Ipv6AddressNextHop
)
/*++

Routine Description:

Add IPv6 route to route table.

Arguments:

IfIndex - IfIndex of interface for route.

Ipv6AddressPrefix - prefix for route.

Ipv6AddressNextHop - next hop for the given prefix.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	DWORD error;
	PWSTR netshCommand = NULL;
	HRESULT hr = S_OK;
	DWORD exitCode = 0;


	error = FormatString(&netshCommand,
		0,
		L"netsh interface ipv6 add route %ls %lu %ls",
		Ipv6AddressPrefix,
		IfIndex,
		Ipv6AddressNextHop);
	if (error != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(error);
		LOGERRW(L"NetIfAddIPv6Route:: FormatString Add route Failing HRESULT status detected %x", hr);
		goto Cleanup;
	}

	LOGINFOW(L"Running cmd %s", netshCommand);
	hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
	if (hr != S_OK)
	{
		LOGERRW(L"Command failed ('%ls').", netshCommand);
		goto Cleanup;
	}

	if (exitCode != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(exitCode);
		LOGERRW(L"Command returned non zero exit code %u ('%ls').", exitCode, netshCommand);
		goto Cleanup;
	}

Cleanup:

	STR_FREE(netshCommand);
	return hr;
}

HRESULT
ServiceTunnelConfigurator::NetIfDeleteIPv6Route(
	__in ULONG IfIndex,
	__in PCWSTR Ipv6AddressPrefix,
	__in PCWSTR Ipv6AddressNextHop
)
/*++

Routine Description:

Delete IPv6 route to route table.

Arguments:

IfIndex - IfIndex of interface for route.

Ipv6AddressPrefix - prefix for route.

Ipv6AddressNextHop - next hop for the given prefix.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	DWORD error;
	PWSTR netshCommand = NULL;
	HRESULT hr = S_OK;
	DWORD exitCode = 0;


	//
	// First delete the route
	//

	error = FormatString(&netshCommand,
		0,
		L"netsh interface ipv6 delete route %ls %lu %ls",
		Ipv6AddressPrefix,
		IfIndex,
		Ipv6AddressNextHop);

	if (error != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(error);
		LOGERRW(L"NetIfDeleteIPv6Route:: FormatString delete route Failing HRESULT status detected %x", hr);
		goto Cleanup;
	}

	LOGINFOW(L"Running cmd %s", netshCommand);
	hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);

	if (hr != S_OK)
	{
		LOGERRW(L"Command failed ('%ls').", netshCommand);
		goto Cleanup;
	}

	if (exitCode != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(exitCode);
		LOGERRW(L"Command returned non zero exit code %u ('%ls').", exitCode, netshCommand);
		goto Cleanup;
	}

Cleanup:

	STR_FREE(netshCommand);
	return hr;
}

HRESULT
ServiceTunnelConfigurator::NetIfAddNeighbor(
	__in ULONG IfIndex,
	__in PCWSTR NeighborAddress,
	__in PCWSTR NeighborLinkLayerAddress
)
/*++

Routine Description:

Add neighbor information.

Arguments:

IfIndex - IfIndex of interface to add the neighbor on.

NeighborAddress - IP address of the added neighbor

NeighborLinkLayerAddress - Link layer address of the added/deleted neighbor

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	DWORD error;
	PWSTR netshCommand = NULL;
	HRESULT hr = S_OK;
	DWORD exitCode = 0;

	//
	// Add the neighbor
	//
	error = FormatString(&netshCommand,
		0,
		L"netsh interface ipv6 add neighbors %lu %ls %ls",
		IfIndex,
		NeighborAddress,
		NeighborLinkLayerAddress);
	if (error != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(error);
		LOGERRW(L"NetIfAddNeighbor FormatString add neighbors Failing HRESULT status detected %x", hr);
		goto Cleanup;			
	}

	LOGINFOW(L"Running cmd %s", netshCommand);
	hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
	if (hr != S_OK)
	{
		LOGERRW(L"Command failed ('%ls').", netshCommand);
		goto Cleanup;
	}

	if (exitCode != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(exitCode);
		LOGERRW(L"Command returned non zero exit code %u ('%ls').", exitCode, netshCommand);
		goto Cleanup;
	}

Cleanup:

	STR_FREE(netshCommand);
	return hr;
}


HRESULT
ServiceTunnelConfigurator::NetIfDeleteNeighbor(
	__in ULONG IfIndex,
	__in PCWSTR NeighborAddress,
	__in PCWSTR NeighborLinkLayerAddress
)
/*++

Routine Description:

Delete neighbor information.

Arguments:

IfIndex - IfIndex of interface to delete the neighbor on.

NeighborAddress - IP address of the deleted neighbor

NeighborLinkLayerAddress - Link layer address of the added/deleted neighbor

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	DWORD error;
	PWSTR netshCommand = NULL;
	HRESULT hr = S_OK;
	DWORD exitCode = 0;

	//
	// Delete the neighbor
	//

	error = FormatString(&netshCommand,
		0,
		L"netsh interface ipv6 delete neighbors %lu %ls %ls",
		IfIndex,
		NeighborAddress,
		NeighborLinkLayerAddress);
	if (error != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(error);
		LOGERRW(L"NetIfDeleteNeighbor FormatString delete neighbors Failing HRESULT status detected %x", hr);
		goto Cleanup;
	}

	LOGINFOW(L"Running cmd %s", netshCommand);
	hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);

	if (hr != S_OK)
	{
		LOGERRW(L"Command failed ('%ls').", netshCommand);
		goto Cleanup;
	}

	if (exitCode != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(exitCode);
		LOGERRW(L"Command returned non zero exit code %u ('%ls').", exitCode, netshCommand);
		goto Cleanup;
	}

Cleanup:

	STR_FREE(netshCommand);
	return hr;
}

HRESULT 
ServiceTunnelConfigurator::GetInterfaceByIpAddress(
	__in ULONG IpAddress,
	__out int& InterfaceIndex
)
/*++

Routine Description:

Gets the Interface Index for corresponding IPV4 Address

Arguments:

IpAddress - IPV4 address whose interface index has to be found.
InterfaceIndex - To store the Interface Index

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	/* Variables used by GetIpAddrTable */
	PMIB_IPADDRTABLE pIPAddrTable;
	DWORD dwSize = 0;
	DWORD dwRetVal = 0;
	int i;
	HRESULT hr = S_OK;
	bool ipFound = false;

	pIPAddrTable = (MIB_IPADDRTABLE *)malloc(sizeof(MIB_IPADDRTABLE));

	if (pIPAddrTable == NULL)
	{
		hr = HRESULT_FROM_WIN32(ERROR_INSUFFICIENT_BUFFER);
		LOGERRW(L"GetInterfaceByIpAddress:: Failed to allocate memory for pIPAddrTable");
		goto Cleanup;
	}

	// Make an initial call to GetIpAddrTable to get the
	// necessary size into the dwSize variable

	if (GetIpAddrTable(pIPAddrTable, &dwSize, false) == ERROR_INSUFFICIENT_BUFFER)
	{
		free(pIPAddrTable);
		pIPAddrTable = (MIB_IPADDRTABLE *)malloc(dwSize);

	}

	if (pIPAddrTable == NULL)
	{
		hr = HRESULT_FROM_WIN32(ERROR_INSUFFICIENT_BUFFER);
		LOGERRW(L"GetInterfaceByIpAddress:: Failed to allocate memory for pIPAddrTable after size");
		goto Cleanup;
	}

	// Make a second call to GetIpAddrTable to get the
	// actual data we want
	if ((dwRetVal = GetIpAddrTable(pIPAddrTable, &dwSize, false)) != NO_ERROR)
	{
		HRESULT_FROM_WIN32(dwRetVal);
		LOGERRW(L"GetInterfaceByIpAddress:: GetIpAddrTable returned error code :%d hr %x", dwRetVal, hr);
		goto Cleanup;
	}

	for (i = 0; i < (int)pIPAddrTable->dwNumEntries; i++)
	{
		if (htonl(pIPAddrTable->table[i].dwAddr) == IpAddress)
		{
			InterfaceIndex = pIPAddrTable->table[i].dwIndex;
			ipFound = true;
			break;
		}
	}

	if (!ipFound)
	{
		hr = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
		LOGERRW(L"GetInterfaceByIpAddress:: Ipaddress %d not found in IpAddr Table", IpAddress);
		goto Cleanup;
	}

Cleanup:

	if (pIPAddrTable)
	{
		free(pIPAddrTable);
		pIPAddrTable = NULL;
	}
	return hr;
}

std::set<string> ServiceTunnelConfigurator::GetIPV6AddressesOfAdapter(
	__in unsigned long interfaceIndex
) 
{
	PIP_ADAPTER_ADDRESSES adapterAddresses = NULL;
	DWORD status = ERROR_SUCCESS;
	HRESULT hr = S_OK;
	ULONG ipFamily = AF_INET6;
	ULONG outBufLen = 0;
	ULONG flags = 0;
	std::set<string> ipv6Addresses;
	
	LOGINFOW(L"GetIPV6Address for interface index %u", interfaceIndex)
	
	status = GetAdaptersAddresses(ipFamily, 0, NULL, adapterAddresses, &outBufLen);
	if (status != ERROR_BUFFER_OVERFLOW)
	{
		LOGERRW(L"GetAdaptersAddresses failed with error status %x", status);
		hr = E_FAIL;
		goto Cleanup;
	}

	adapterAddresses = (PIP_ADAPTER_ADDRESSES) new char[outBufLen];
	flags = GAA_FLAG_SKIP_ANYCAST|GAA_FLAG_SKIP_MULTICAST|GAA_FLAG_SKIP_DNS_SERVER;
	
	status = GetAdaptersAddresses(ipFamily, flags, NULL, adapterAddresses, &outBufLen);
	if (status != ERROR_SUCCESS)
	{
		hr = HRESULT_FROM_WIN32(status);
		LOGERRW(L"GetAdaptersAddresses failed with error hr %x",hr);
		goto Cleanup;
	}

	for (PIP_ADAPTER_ADDRESSES curAddresses = adapterAddresses;
		curAddresses != nullptr;
		curAddresses = curAddresses->Next)
	{
		LOGINFOW(L"Iterating Friendly name :%s Index:%u decription %s", curAddresses->FriendlyName, curAddresses->IfIndex, curAddresses->Description);
		LOGINFO("Adapter Name %s ipv6index %u", curAddresses->AdapterName, curAddresses->Ipv6IfIndex)

		if (curAddresses->IfIndex == interfaceIndex)
		{
			for (PIP_ADAPTER_UNICAST_ADDRESS unicastAddr = curAddresses->FirstUnicastAddress; 
				unicastAddr != nullptr;
				unicastAddr = unicastAddr->Next)
			{
				if (unicastAddr->Address.lpSockaddr->sa_family == AF_INET6)
				{
					char ipv6Str[INET6_ADDRSTRLEN];
					PSOCKADDR_IN6 pSockAddrIn6 = (PSOCKADDR_IN6)unicastAddr->Address.lpSockaddr;
					int size = sizeof(pSockAddrIn6->sin6_addr.u.Byte);
					char ipv6CharArray[sizeof(pSockAddrIn6->sin6_addr.u.Byte)+1];

					inet_ntop(ipFamily, &(pSockAddrIn6->sin6_addr), ipv6Str, INET6_ADDRSTRLEN);
					LOGINFO("Unicast IPv6 Address: %s", ipv6Str);	
					memcpy(ipv6CharArray, pSockAddrIn6->sin6_addr.u.Byte, size);
					ipv6CharArray[size] = '\0';				
					ipv6Addresses.insert(std::string(ipv6CharArray));

				}
			}
		}
	}

Cleanup:
	if (adapterAddresses != NULL)
	{
		delete adapterAddresses;
	}

	return ipv6Addresses;
}


ULONG ServiceTunnelConfigurator::CheckIfNeighborEntryExists( 
	__in int interfaceIndex,
	__in PWCHAR ipv6Address
)
{
	USHORT port = 0;
	DWORD error;
	ULONG scopeId = 0;
	MIB_IPNET_ROW2 mib = {0};
	char ipv6Str[INET6_ADDRSTRLEN];
	
	mib.InterfaceIndex = interfaceIndex;
	mib.Address.si_family = AF_INET6;
	mib.Address.Ipv6.sin6_family = AF_INET6;

	LOGINFOW(L"Check if Neighbor entry exist for IP %s exists on this ifindex %d", 
				ipv6Address,
				interfaceIndex);

	error = RtlIpv6StringToAddressEx(ipv6Address, &(mib.Address.Ipv6.sin6_addr), &scopeId, &port);
	if (error != ERROR_SUCCESS)
	{
		LOGERRW(L"Failed to convert ipv6 %s to in6_addr error:%d", ipv6Address, error)
		return error;
	}

	inet_ntop(AF_INET6, &(mib.Address.Ipv6.sin6_addr), ipv6Str, INET6_ADDRSTRLEN);
	LOGINFO("gateway IPv6 Addresses: %s", ipv6Str);	

	error = GetIpNetEntry2(&mib);
	if (error == ERROR_SUCCESS)
	{
		if (mib.State != NlnsPermanent)
		{
			return ERROR_NOT_FOUND;
		}
	}

	return error;
}

ULONG ServiceTunnelConfigurator::CheckIfRouteExists(
	__in int interfaceIndex,
	__in PWCHAR destinationIPv6Prefix,
	__in int destinationIPv6PrefixLen,
	__in PWCHAR gatewayIPv6
)

{
	PMIB_IPFORWARD_TABLE2 forwardTable = NULL;

	_bstr_t dst(destinationIPv6Prefix);
	const char* dstIp = dst;	

	_bstr_t gw(gatewayIPv6);
	const char* gwIp = gw;	


	LOGINFO("Check route exist for dst %s gw %s ifindex %d", dstIp, gwIp, interfaceIndex);

	ULONG ret = GetIpForwardTable2(AF_INET6, &forwardTable);	
	if (ret == 0) {
		for (ULONG tableIndex = 0; tableIndex < forwardTable->NumEntries; tableIndex++) { 
			char dstIpv6[INET6_ADDRSTRLEN];
			char gwIpv6[INET6_ADDRSTRLEN];

			inet_ntop(AF_INET6, &(forwardTable->Table[tableIndex].DestinationPrefix.Prefix.Ipv6.sin6_addr), dstIpv6, INET6_ADDRSTRLEN);
			inet_ntop(AF_INET6, &(forwardTable->Table[tableIndex].NextHop.Ipv6.sin6_addr), gwIpv6, INET6_ADDRSTRLEN);
			LOGINFO("dst addr %s gateway IPv6 addr: %s ifindex: %d", dstIpv6, gwIpv6, forwardTable->Table[tableIndex].InterfaceIndex);	

			if (forwardTable->Table[tableIndex].DestinationPrefix.PrefixLength == destinationIPv6PrefixLen && 
				strcmp(dstIpv6, dstIp) == 0 &&
				strcmp(gwIpv6, gwIp) == 0 && 
				forwardTable->Table[tableIndex].InterfaceIndex == (NET_IFINDEX)interfaceIndex)
			{
				ret = 0;
				goto Cleanup;
			}
		}

		ret = ERROR_NOT_FOUND;
	} 

Cleanup:

	if(forwardTable) 
	{
		FreeMibTable(forwardTable);
	}

	return ret;
}

HRESULT
ServiceTunnelConfigurator::SetupLoopbackAdapter(
	__in std::wstring InstanceAddress,
	__in std::set<std::wstring> Ipv6AddressList,
	__in int NumRetries,
	__in int WaitTimeStartInSecs,
	__in int TotalWaitTimeInSecs
)
/*++

Routine Description:

Sets up the loopback adapter for service tunnel extension.

Arguments:

InstanceAddress		- IPV4 address of neighbor interface.
Ipv6AddressList		- List of IPV6 address to be configured on the loopback adapter.
NumRetries			- Max retry count.
WaitTimeStartInSecs - Start wait time before a retry.
TotalWaitTimeInSecs	- Total wait time for retrying the scenario.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;
	int waitTime = 0;
	int retryCount = 0;
	int waitTimeSum = 0;

	do
	{
		int interfaceIndex = 0;
		int ipv4interfaceIndex = 0;
		std::set<wstring>::iterator ipv6AddressListIterator;
		std::set<string>::iterator ipv6AddressItr;
		std::set<string> ipv6AddressesofAdapter;
		ULONG ipv4Addr = 0;
		PCHAR buffer = NULL;
		LPCWSTR wipv4;
		int ret;
		size_t wlen;

		Sleep(waitTime * 1000);

		hr = installer.GetOrCreateLoopbackAdapter(SERVICE_TUNNEL_INTERFACE_NAME, interfaceIndex);
		if (hr != S_OK)
		{
			LOGERRW(L"SetupLoopbackAdapter::Failed to set up loopback adapter [%0x]", hr);
			goto Cleanup;
		}

		ipv6AddressesofAdapter = GetIPV6AddressesOfAdapter((unsigned long) interfaceIndex);
		

		for (ipv6AddressListIterator = Ipv6AddressList.begin();
			ipv6AddressListIterator != Ipv6AddressList.end();
			++ipv6AddressListIterator)
		{
			SOCKADDR_IN6 sa;
			int size = sizeof(sa.sin6_addr.u.Byte);
			char ipv6CharArray[sizeof(sa.sin6_addr.u.Byte)+1];
			std::wstring_convert<std::codecvt_utf8_utf16<wchar_t>> converter;
			std::string ipv6ToCheck = converter.to_bytes(*ipv6AddressListIterator);

			LOGINFO("ipv6ToCheck %s", ipv6ToCheck.c_str())

			inet_pton(AF_INET6, ipv6ToCheck.c_str(), &(sa.sin6_addr));
			memcpy(ipv6CharArray, sa.sin6_addr.u.Byte, size);
			ipv6CharArray[size] = '\0';

			if (ipv6AddressesofAdapter.find(std::string(ipv6CharArray)) == ipv6AddressesofAdapter.end())  {
				LOGINFOW(
					L"\tAdding IPv6Address:[%s] to interface with index [%d]\n",
					(*ipv6AddressListIterator).c_str(),
					interfaceIndex);

				hr = NetIfSetIpv6Address(interfaceIndex, (*ipv6AddressListIterator).c_str());
				if (hr != S_OK)
				{
					LOGERRW(L"SetupLoopbackAdapter:: Failed to add address [%s] on interface with index[%d]. Error:[%0x]", (*ipv6AddressListIterator).c_str(), interfaceIndex, hr);
					goto Cleanup;
				}
			}
			else 
			{
				LOGINFO("IPV6 Address already set on the adapter")
			}
		}

		// to forwards packets from one interface to other.
		hr = NetIfEnableWeakHostRouting(interfaceIndex);
		if (hr != S_OK)
		{
			LOGERRW(L"SetupLoopbackAdapter:: Failed to set weak host routing on interface with index [%d]. Error:[%0x]", interfaceIndex, hr);
			goto Cleanup;
		}

		wipv4 = InstanceAddress.c_str();
		wlen = wcslen(wipv4) + 1;

		buffer = (char *)malloc(wlen * sizeof(char));
		if (!buffer)
		{
			hr = HRESULT_FROM_WIN32(ERROR_INSUFFICIENT_BUFFER);
			LOGERRW(L"SetupLoopbackAdapter:: Malloc failed for allocating ipv4 address[%d]. Error:[%0x]", interfaceIndex, hr);
			goto Cleanup;
		}

		//check for NULL
		size_t bytesCopied = 0;
		ret = wcstombs_s(&bytesCopied, buffer, wlen, wipv4, wlen);
		if (ret != 0)
		{
			hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
			LOGERRW(L"SetupLoopbackAdapter:: wcstombs_s failed. Return code [%d]", ret);
			goto Cleanup;
		}

		ipv4Addr = ntohl(inet_addr(buffer));

		hr = GetInterfaceByIpAddress(ipv4Addr, ipv4interfaceIndex);
		if (hr != S_OK)
		{
			LOGERRW(L"SetupLoopbackAdapter::  GetInterfaceByIpAddress function failed to find Interface Index for %s address. Error[%0x]",
				wipv4,hr);
			goto Cleanup;
		}

		LOGINFOW(L"Interface Index for %s address is %d",
			wipv4,
			ipv4interfaceIndex);

		hr = NetIfEnableWeakHostRouting(ipv4interfaceIndex);
		if (hr != S_OK)
		{
			LOGERRW(
				L"SetupLoopbackAdapter:: Failed to set weak host routing on interface with index [%d]. Error:[%0x]",
				ipv4interfaceIndex,
				hr);
			goto Cleanup;
		}

		ULONG status = CheckIfNeighborEntryExists(ipv4interfaceIndex, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS);
	
		if (status != 0) 
		{
			LOGINFO("CheckIfNeighborEntryExists return value %d", status)
			//
			// Add fake neighbor
			//
			hr = NetIfAddNeighbor(ipv4interfaceIndex,
				SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS,
				SERVICE_TUNNEL_ENDPOINT_FAKE_GW_MAC_ADDRESS
				);
			if (hr != S_OK)
			{
				LOGERRW(
					L"SetupLoopbackAdapter::  Failed to add fake GW MAC to neighbor table for "
					L"interface with index [%d]. Error:[%0x]",
					ipv4interfaceIndex,
					hr);
				goto Cleanup;
			}
		}
		else 
		{
			LOGINFO("Neighbor entry exists. No need to add");
		}

		status = CheckIfRouteExists(
			ipv4interfaceIndex, 
			SERVICE_TUNNEL_SOURCE_PREFIX, 
			SERVICE_TUNNEL_SOURCE_PREFIX_LEN, 
			SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS
		);
		if (status != 0)
		{
			WCHAR stDstPrefix[MAX_PATH];
			WCHAR prefixLen[INET6_ADDRSTRLEN];

			LOGINFO("CheckIfRouteExists return value %d", status)
			//
			// Add fake GW
			//
			swprintf_s(prefixLen, sizeof(prefixLen), L"%d", SERVICE_TUNNEL_SOURCE_PREFIX_LEN);
			wcsncpy_s(stDstPrefix, sizeof(stDstPrefix), SERVICE_TUNNEL_SOURCE_PREFIX, wcslen(SERVICE_TUNNEL_SOURCE_PREFIX));
			wcscat_s(stDstPrefix, L"/");
			wcscat_s(stDstPrefix, prefixLen);

			hr = NetIfAddIPv6Route(ipv4interfaceIndex,
				stDstPrefix,
				SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS
				);
			if (hr != S_OK)
			{
				LOGERRW(
					L"SetupLoopbackAdapter:: Failed to add fake GW on interface with index [%d]. Error:[%0x]",
					ipv4interfaceIndex,
					hr);
				goto Cleanup;
			}
		}
		else 
		{
			LOGINFO("Route entry exists. No need to add")
		}
	Cleanup:
		if (buffer)
		{
			free(buffer);
		}

		if (hr != S_OK)
		{
			if (!waitTime)
			{
				waitTime = WaitTimeStartInSecs;
			}
			else
			{
				waitTime = waitTime * 2;
			}
			waitTimeSum = waitTimeSum + waitTime;
			retryCount++;
			LOGINFOW(L"Retrying..count:%d TotalWaitTime:%d curWaitTime:%d", retryCount, waitTimeSum, waitTime);
		}

	} while (hr != S_OK && retryCount <= NumRetries && waitTimeSum < TotalWaitTimeInSecs);

	return hr;
}

std::wstring
ServiceTunnelConfigurator::GetSTIpv6AddrFromInstanceIpv4Addr(
	__in std::wstring InstanceIpv4Addr
)
/*++

Routine Description:

Helper to retrieve IPV6 address from IPV4 address.

Arguments:

InstanceIpv4Addr - IPV4 Address to which IPV6 address has to be computed.

Return Value:

std::wstring - Returns corresponding IPV6 address for IPV4 address

--*/
{
	unsigned short ip[4];
	std::string result[4];
	std::string ipv6Addr;

	swscanf_s(InstanceIpv4Addr.c_str(), L"%hu.%hu.%hu.%hu", &ip[0], &ip[1], &ip[2], &ip[3]);

	for (int i = 0; i < 4; i++)
	{
		std::stringstream sstream;
		if (i % 2 == 0)
		{
			sstream << std::hex << ip[i];
		}
		else
		{
			sstream << std::setw(2) << std::setfill('0') << std::hex << ip[i];
		}
		result[i] = sstream.str();
	}

	ipv6Addr = SERVICE_TUNNEL_IPV6_ADDRESS_PREFIX + result[0] + result[1] + ":" + result[2] + result[3];
	std::wstring wsIpValue(ipv6Addr.begin(), ipv6Addr.end());
	LOGINFOW(L"Derived Ipv6 Address :%s from %s", wsIpValue.c_str(), InstanceIpv4Addr.c_str());

	return wsIpValue;
}

HRESULT
ServiceTunnelConfigurator::CleanupLoopbackAdapter(
	__in std::wstring InstanceAddress
)
{	
	IN_ADDR ipv4Addr;
	DWORD ret = 0;
	int ipv4interfaceIndex=0;
	PCWSTR temp = NULL;
	WCHAR stDstPrefix[MAX_PATH];
	WCHAR prefixLen[INET6_ADDRSTRLEN];

	HRESULT hr = installer.RemoveLoopbackAdapter(SERVICE_TUNNEL_INTERFACE_NAME);
	if (hr != S_OK)
	{
		LOGERRW(L"Remove LoopBackAdapter failed:%x.",hr);
	}
	else
	{
		LOGINFOW(L"Removed LoopBackAdapter");
	}
	

	LOGINFOW(L"Calling RtlIpv4StringToAddress %s", InstanceAddress.c_str());
	ret = RtlIpv4StringToAddress(InstanceAddress.c_str(), true, &temp, &ipv4Addr);
	
	if (ret != 0)
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			LOGERRW(
				L"CleanupLoopbackAdapter:: RtlIpv4StringToAddress function failed with error %d Address %s hr %x",
				ret,
				InstanceAddress.c_str(),
				hr);
			goto Cleanup;
		}
	}

	hr = GetInterfaceByIpAddress(ntohl(ipv4Addr.s_addr), ipv4interfaceIndex);
	if (hr != S_OK)
	{
		LOGERRW(
			L"CleanupLoopbackAdapter:: GetInterfaceByIpAddress function failed to find Interface Index for %s address error %x",
			InstanceAddress.c_str(),
			hr);
		goto Cleanup;
	}
	
	LOGINFOW(L"Deleting neighbor and route entry for interface index %d", ipv4interfaceIndex);
	
	hr = NetIfDeleteNeighbor(ipv4interfaceIndex, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_MAC_ADDRESS);
	LOGINFOW(L"CleanupLoopbackAdapter:: NetIfDeleteNeighbor for Interface %d returned with %x",ipv4interfaceIndex, hr);

	swprintf_s(prefixLen, sizeof(prefixLen), L"%d", SERVICE_TUNNEL_SOURCE_PREFIX_LEN);
	wcsncpy_s(stDstPrefix, sizeof(stDstPrefix), SERVICE_TUNNEL_SOURCE_PREFIX, wcslen(SERVICE_TUNNEL_SOURCE_PREFIX));
	wcscat_s(stDstPrefix, L"/");
	wcscat_s(stDstPrefix, prefixLen);

	hr = NetIfDeleteIPv6Route(ipv4interfaceIndex, stDstPrefix, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS);
	LOGINFOW(L"CleanupLoopbackAdapter:: NetIfDeleteRoute for Interface %d returned with  %x",ipv4interfaceIndex, hr);

	hr = S_OK;


Cleanup:	
	return hr;

}