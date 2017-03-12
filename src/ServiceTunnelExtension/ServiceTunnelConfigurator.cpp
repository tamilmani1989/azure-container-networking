
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
#include "ServiceTunnelConfigurator.h"

HRESULT
ServiceTunnelConfigurator::NetIfEnableWeakHostRouting(
    __in ULONG IfIndex)
/*++

Routine Description:

    This routine enables weak host send and receive on specified interface.

Arguments:

    IfIndex - IfIndex of interface to enable weak host send/receive on.

    IsIpv6 - whether weak host setting is for IPv6. FALSE means IPv4.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    DWORD error;
    PWSTR netshCommand = NULL;
    HRESULT hr = S_OK;
    DWORD exitCode;

    RD_TRACE_STATIC_ENTRY();

    //
    // Enable weak host routing.
    //

    error = FormatString(&netshCommand,
        0,
        L"netsh interface ipv6 set interface %lu weakhostreceive=enable weakhostsend=enable",
        IfIndex);
    RD_E_WIN32(error, Cleanup);

    hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"Command failed ('%ls').",
        netshCommand),
        hr,
        Cleanup);

    if (exitCode != ERROR_SUCCESS)
    {
        hr = HRESULT_FROM_WIN32(exitCode);
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"Command returned non zero exit code %u ('%ls').",
            exitCode,
            netshCommand),
            hr,
            Cleanup);
    }

Cleanup:

    STR_FREE(netshCommand);

    RD_TRACE_STATIC_EXIT(hr);
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

    RD_TRACE_STATIC_ENTRY();

    if (Ipv6Address == NULL) {
        RD_E_WIN32(ERROR_INVALID_PARAMETER, Cleanup);
    }

    RD_TRACE_LOG((RD_TRACE_INFO,
        L"NetIfSetIpv6Address: Index:%lu, IP:%ls",
        IfIndex,
        Ipv6Address));

    //
    // Create a synchronization event.
    //

    changeEvent = CreateEvent(NULL, TRUE, FALSE, NULL);

    if (changeEvent == NULL) {
        RD_E_GLE(Cleanup);
    }

    //
    // Register the change notification.
    //

    error = RtlIpv6StringToAddressEx(Ipv6Address, &ipv6Address, &scopeId, &port);

    if (error != ERROR_SUCCESS) 
    {
        hr = HRESULT_FROM_NT(error);
        RD_E_HR(hr, Cleanup);
    }

    changeContext.Event = changeEvent;
    changeContext.Ipv6Address = ipv6Address;

    error = NotifyUnicastIpAddressChange(AF_INET,
        NetpIpv6AdressChangeCallback,
        (PVOID)&changeContext,
        FALSE,
        &notification);
    RD_TRACE_E_WIN32((RD_TRACE_ERROR,
        L"Failed to register for IP address change notification"),
        error,
        Cleanup);

    //
    // Delete the IP address first, ignore error.
    //

    error = FormatString(&netshCommand,
        0,
        L"netsh interface ipv6 delete address interface=%lu address=%ls",
        IfIndex,
        Ipv6Address);
    RD_E_WIN32(error, Cleanup);

    hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
    RD_TRACE_LOG((RD_TRACE_INFO,
        L"Command '%ls' result: hr = %u, exitCode = %u.",
        netshCommand,
        hr,
        exitCode));

    STR_FREE(netshCommand);

    //
    // Set the IP address.
    //

    error = FormatString(&netshCommand,
        0,
        L"netsh interface ipv6 add address interface=%lu address=%ls store=persistent",
        IfIndex,
        Ipv6Address);
    RD_E_WIN32(error, Cleanup);

    RD_TRACE_LOG((RD_TRACE_INFO,
        L"Applying network config: %ls",
        netshCommand));

    hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"Command failed ('%ls').",
        netshCommand),
        hr,
        Cleanup);

    if (exitCode != ERROR_SUCCESS)
    {
        hr = E_UNEXPECTED;
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"Command returned non zero exit code %u ('%ls').",
            exitCode,
            netshCommand),
            hr,
            Cleanup);
    }

    //
    // Wait for IP address change notification.
    //

    RD_TRACE_LOG((RD_TRACE_INFO,
        L"Waiting for IP address to be updated."));

    result = WaitForSingleObject(changeEvent, NET_UNICAST_IP_ADDRESS_CHANGE_TIMEOUT);
    if (result != WAIT_OBJECT_0) {
        RD_TRACE_E_GLE((RD_TRACE_ERROR,
            L"Wait for IP address change notification failed"),
            Cleanup);
    }

    RD_TRACE_LOG((RD_TRACE_INFO,
        L"Successfully updated IP address."));

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

    RD_TRACE_STATIC_EXIT(hr);
    return hr;

}

HRESULT
ServiceTunnelConfigurator::NetIfAddDeleteIPv6Route(
    __in ULONG IfIndex,
    __in PCWSTR Ipv6AddressPrefix,
    __in PCWSTR Ipv6AddressNextHop,
    __in BOOL IsAdd
)
/*++

Routine Description:

    Add or Delete IPv6 route to route table.

Arguments:

    IfIndex - IfIndex of interface for route.

    Ipv6AddressPrefix - prefix for route.

    Ipv6AddressNextHop - next hop for the given prefix.

    IsAdd - Flag to indicate if the requested operation is "Add" or "Delete"

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    DWORD error;
    PWSTR netshCommand = NULL;
    HRESULT hr = S_OK;
    HRESULT tmpHr = S_OK;
    DWORD exitCode = 0;

    RD_TRACE_STATIC_ENTRY();

    //
    // First delete the route
    //

    error = FormatString(&netshCommand,
        0,
        L"netsh interface ipv6 delete route %ls %lu %ls",
        Ipv6AddressPrefix,
        IfIndex,
        Ipv6AddressNextHop);
    RD_E_WIN32(error, Cleanup);

    tmpHr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);

    if (!IsAdd)
    {

        hr = tmpHr;

        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"Command failed ('%ls').",
            netshCommand),
            hr,
            Cleanup);

        if (exitCode != ERROR_SUCCESS)
        {
            hr = HRESULT_FROM_WIN32(exitCode);
            RD_TRACE_E_HR((RD_TRACE_ERROR,
                L"Command returned non zero exit code %u ('%ls').",
                exitCode,
                netshCommand),
                hr,
                Cleanup);
        }
    }
    else
    {
        if (exitCode != ERROR_SUCCESS)
        {
            RD_TRACE_LOG((RD_TRACE_WARNING, L"Command returned non zero exit code %u ('%ls').",
                exitCode,
                netshCommand));
        }
        //
        // Add the route
        //

        STR_FREE(netshCommand);

        error = FormatString(&netshCommand,
            0,
            L"netsh interface ipv6 add route %ls %lu %ls",
            Ipv6AddressPrefix,
            IfIndex,
            Ipv6AddressNextHop);
        RD_E_WIN32(error, Cleanup);

        hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"Command failed ('%ls').",
            netshCommand),
            hr,
            Cleanup);

        if (exitCode != ERROR_SUCCESS)
        {
            hr = HRESULT_FROM_WIN32(exitCode);
            RD_TRACE_E_HR((RD_TRACE_ERROR,
                L"Command returned non zero exit code %u ('%ls').",
                exitCode,
                netshCommand),
                hr,
                Cleanup);
        }
    }

Cleanup:

    STR_FREE(netshCommand);

    RD_TRACE_STATIC_EXIT(hr);
    return hr;
}

HRESULT
ServiceTunnelConfigurator::NetIfAddDeleteNeighbor(
    __in ULONG IfIndex,
    __in PCWSTR NeighborAddress,
    __in PCWSTR NeighborLinkLayerAddress,
    __in BOOL   IsAdd
)
/*++

Routine Description:

    Add or Delete neighbor information.

Arguments:

    IfIndex - IfIndex of interface to add/delete the neighbor on.

    NeighborAddress - IP address of the added/deleted neighbor

    NeighborLinkLayerAddress - Link layer address of the added/deleted neighbor

    IsAdd - Flag to indicate if this operation is for adding or for deleting neighbor

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    DWORD error;
    PWSTR netshCommand = NULL;
    HRESULT hr = S_OK;
    HRESULT tmpHr = S_OK;
    DWORD exitCode = 0;

    RD_TRACE_STATIC_ENTRY();

    //
    // Delete the neighbor first
    //

    error = FormatString(&netshCommand,
        0,
        L"netsh interface ipv6 delete neighbors %lu %ls %ls",
        IfIndex,
        NeighborAddress,
        NeighborLinkLayerAddress);
    RD_E_WIN32(error, Cleanup);

    tmpHr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);

    if (!IsAdd)
    {
        hr = tmpHr;
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"Command failed ('%ls').",
            netshCommand),
            hr,
            Cleanup);

        if (exitCode != ERROR_SUCCESS)
        {
            hr = HRESULT_FROM_WIN32(exitCode);
            RD_TRACE_E_HR((RD_TRACE_ERROR,
                L"Command returned non zero exit code %u ('%ls').",
                exitCode,
                netshCommand),
                hr,
                Cleanup);
        }
    }
    else
    {

        if (exitCode != ERROR_SUCCESS)
        {
            RD_TRACE_LOG((RD_TRACE_WARNING,
                L"Command returned non zero exit code %u ('%ls').",
                exitCode,
                netshCommand));
        }

        //
        // Add the neighbor
        //

        STR_FREE(netshCommand);

        error = FormatString(&netshCommand,
            0,
            L"netsh interface ipv6 add neighbors %lu %ls %ls",
            IfIndex,
            NeighborAddress,
            NeighborLinkLayerAddress);
        RD_E_WIN32(error, Cleanup);

        hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"Command failed ('%ls').",
            netshCommand),
            hr,
            Cleanup);

        if (exitCode != ERROR_SUCCESS)
        {
            hr = HRESULT_FROM_WIN32(exitCode);
            RD_TRACE_E_HR((RD_TRACE_ERROR,
                L"Command returned non zero exit code %u ('%ls').",
                exitCode,
                netshCommand),
                hr,
                Cleanup);
        }
    }

Cleanup:

    STR_FREE(netshCommand);

    RD_TRACE_STATIC_EXIT(hr);
    return hr;
}


HRESULT
ServiceTunnelConfigurator::SetupLoopbackAdapter(
    __in std::set<std::wstring> Ipv6AddressList
    )
/*++

Routine Description:

    Sets up the loopback adapter for service tunnel extension.

Arguments:

    ipv6AddressList - List of IPV6 address to be configured on the loopback adapter.
    
Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    HRESULT hr = S_OK;
    int interfaceIndex = 0;
    std::set<wstring>::iterator ipv6AddressListIterator;
    hr = installer.GetOrCreateLoopbackAdapter(SERVICE_TUNNEL_INTERFACE_NAME, interfaceIndex);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"SetupLoopbackAdapter:: Failed to set up loopback adapter [%0x]",
        hr),
        hr,
        Cleanup);

    for (ipv6AddressListIterator = Ipv6AddressList.begin();
        ipv6AddressListIterator != Ipv6AddressList.end();
        ++ipv6AddressListIterator)
    {
        RD_TRACE_LOG((RD_TRACE_INFO,
            L"\tAdding IPv6Address:[%s] to interface with index [%d]\n", 
            (*ipv6AddressListIterator).c_str(),
            interfaceIndex));
        hr = NetIfSetIpv6Address(interfaceIndex, (*ipv6AddressListIterator).c_str());
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"SetupLoopbackAdapter:: Failed to add address [%s] on interface with index[%d]. Error:[%0x]",
            (*ipv6AddressListIterator).c_str(),
            interfaceIndex,
            hr),
            hr,
            Cleanup);
    }

    hr = NetIfEnableWeakHostRouting(interfaceIndex);

    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"SetupLoopbackAdapter:: Failed to set weak host routing on interface with index [%d]. Error:[%0x]",
        interfaceIndex,
        hr),
        hr,
        Cleanup);
    //
    // Add fake GW
    //
    hr = NetIfAddDeleteIPv6Route(interfaceIndex,
        SERVICE_TUNNEL_SOURCE_PREFIX,
        SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS,
        TRUE);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"SetupLoopbackAdapter:: Failed to add fake GW on interface with index [%d]. Error:[%0x]",
        interfaceIndex,
        hr),
        hr,
        Cleanup);

    //
    // Add fake neighbor
    //
    hr = NetIfAddDeleteNeighbor(interfaceIndex,
        SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS,
        SERVICE_TUNNEL_ENDPOINT_FAKE_GW_MAC_ADDRESS,
        TRUE);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"RuntimeServiceTunnelInterface:: Failed to add fake GW MAC to neighbor table for "
        L"interface with index [%d]. Error:[%0x]",
        interfaceIndex,
        hr),
        hr,
        Cleanup);

Cleanup:
    return hr;
}