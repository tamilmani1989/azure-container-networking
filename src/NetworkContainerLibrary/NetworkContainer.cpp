/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

	NetworkContainer.h

Abstract:

	Implements functions related to network container.

--*/

#include "NetworkContainer.h"
#undef __WFILE__
#undef __WFUNCTION__

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

NetworkContainer::NetworkContainer()
/*++

Routine Description:

	Default onstructor for NetworkContainer. This constructor will enable default settings.

Return Value:
	
	An instance of NetworkContainer object.

--*/
{
	HRESULT hr = ::CoInitialize(NULL);
	if (hr != S_OK)
	{
		LOGERRW(L"ComInitialization failed with error [%0x].", hr);
		return;
	}
}

NetworkContainer::~NetworkContainer()
/*++

Routine Description:

Destructor for NetworkContainer. This destructor will free memory allocated by network container.

Return Value:

None.

--*/
{
	CoUninitialize();
	for (auto &it : adapters)
	{
		wcout << it.first << endl;
		adapters.erase(it.first);
	}
}

NetworkContainer::NetworkContainer(
	_In_ wstring Name,
	_In_ wstring Operation,
	_In_ IN_ADDR Ip,
	_In_ IN_ADDR Netmask,
	_In_ IN_ADDR Gateway,
	_In_ IN_ADDR Dns
) : NetworkContainer::NetworkContainer()
/*++
	
Routine Description:

	Constructor for NetworkContainer. This constructor will enable default settings as well as filling in member variables.
	

Return Value:

	An instance of NetworkContainer object with name, type, ip netmask, dns and gateway filled.

--*/
{
	name = Name;
	ip = Ip;
	netmask = Netmask;
	dns = Dns;
	gateway = Gateway;
}


HRESULT NetworkContainer::FindAdapterByDisplayName(_In_ wstring Name)
/*++

Routine Description:

	This routine creates a loopback adapter object and finds loopback adapter by name.
	
	If the the device instance is found, then fills its information. 
	
	In addition, the device index is saved in the NetworkContainer object as lastIndex for later reuse.

Arguement:

	Name: Display name of the loopback adapter.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;

	adapters[Name] = unique_ptr<LoopbackAdapter>(
		new LoopbackAdapter(
			Name,
			ip,
			netmask,
			dns,
			gateway)
		);

	int instanceIndex = 0;
	hr = adapters[Name]->FindAdapterByDisplayName(Name, instanceIndex);
	if (hr == HRESULT_FROM_WIN32(ERROR_NOT_FOUND)) { return hr; }

	// If lastIndex is set, then we found an existing adapter with the same name.
	lastIndex = instanceIndex;
	if (lastIndex)
	{
		adapters[Name]->SetIndex(lastIndex);
		adapters[Name]->FillDevInfo(lastIndex);
	}

	return hr;
}

HRESULT NetworkContainer::CreateLoopbackAdapter(_In_ wstring Name)
/*++

Routine Description:

	This routine dynamically allocates an LoopbackAdapter class instance, and stores pointer to it to adapters.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.
--*/
{
	HRESULT hr = S_OK;

	hr = FindAdapterByDisplayName(Name);
	// If lastIndex is set, then we found an existing adapter with the same name.
	if (lastIndex)
	{
		hr = HRESULT_FROM_WIN32(ERROR_ALIAS_EXISTS);
		return hr;
	}
	if (hr != HRESULT_FROM_WIN32(ERROR_NOT_FOUND))
	{
		return hr;
	}

	hr = adapters[Name]->Create();

	return hr;
}

HRESULT NetworkContainer::ConfigLoopbackAdapter(
	_In_ wstring Name, 
	_In_ IN_ADDR Ip,
	_In_ IN_ADDR Netmask,
	_In_ IN_ADDR Gateway,
	_In_ IN_ADDR Dns)
/*++

Routine Description:

	This routine configures the loopback adapter.

Arguments:

	Name: Name of the adapter.

	Ip: Ip address to be assigned, in IN_ADDR format.

	Netmask: Netmask associated with Ip, in IN_ADDR format.

	Gateway: Gateway address to be assigned, in IN_ADDR format.

	Dns:  DNS address to be assigned, in IN_ADDR format.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	HRESULT hr = S_OK;

	FindAdapterByDisplayName(Name);
	// If lastIndex is not set, then there's no adapter with name: Name.
	if (!lastIndex)
	{
		hr = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
		return hr;
	}

	// IP is required. Checking is done in caller function.
	hr = adapters[Name]->SetIp(Ip);
	if (hr != S_OK) { return hr; }

	if (Netmask.S_un.S_addr != INADDR_NONE)
	{
		hr = adapters[Name]->SetNetmask(Netmask);
		if (hr != S_OK) { return hr; }
	}

	if (Gateway.S_un.S_addr != INADDR_NONE)
	{
		hr = adapters[Name]->SetGateway(Gateway);
		if (hr != S_OK) { return hr; }
	}

	if (Dns.S_un.S_addr != INADDR_NONE)
	{
		hr = adapters[Name]->SetDns(Dns);
		if (hr != S_OK) { return hr; }
	}

	return hr;
}

HRESULT NetworkContainer::DeleteLoopbackAdapter(_In_ wstring Name)
/*++

Routine Description:

	This routine deletes the loopback adapter.

Arguments:

	Name: Name of the adapter.

Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.
--*/
{
	HRESULT hr = S_OK;

	FindAdapterByDisplayName(Name);
	// If lastIndex is not set, then there's no adapter with name: Name.
	if (!lastIndex)
	{
		hr = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
		return hr;
	}

	hr = adapters[Name]->Delete();

	return hr;
}

const map<wstring, 
	std::unique_ptr<LoopbackAdapter>>& 
	NetworkContainer::GetAdapters() 
/*++

Routine Description:

	Returns a constant reference to adapters.

Return value:

	A constant reference to adapters.

--*/
{
	return adapters;
}