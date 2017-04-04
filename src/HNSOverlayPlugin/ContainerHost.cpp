// ***********************************************************************
// <copyright file="ContainerHost.cpp" company="Microsoft">
//     Copyright (c) Microsoft Corporation. All rights reserved.
// </copyright>
// <summary></summary>
// ***********************************************************************

#include "PluginMain.h"
#include "ContainerHost.h"
#include "tchar.h"
#include "ovPluginException.h"

using namespace Microsoft::Azure::Networking::OverlayPlugin;

////////////////////////////////////////////////////////////////////////////////
// ContainerHost::HandleNetworkNotification
////////////////////////////////////////////////////////////////////////////////
void ContainerHost::HandleNetworkNotification(
    _Inout_ HNSProviderPtr PluginProvider,
    _Inout_ HNSObjPtr RequestObject,
    _In_ bool createNetwork
    )
{
    UNREFERENCED_PARAMETER(createNetwork);
    TRACE_ENTER();
    //
    // currently definition of network in the docker is either subnet or
    // endpoints belongs to same subnet.
    //
    HNSObjListPtr objList;
    DWORD count = 0;
    PCStringPtr hnsNetworkType;

    // TODO: this temporary as HNS doesn't support multiple subnet on Network,
    //       once HNS supports multiple subnet then we need to update it on Subnet
    //       object.
    //
    std::wstring dnsServer;

    PC_THROW_IF_FAILED_LOG(RequestObject->GetString(L"Type", &hnsNetworkType));

    if (::_wcsicmp(hnsNetworkType.get(), L"L2Tunnel") != 0)
    {
        return;
    }

    RequestObject->SetString(L"Type", L"Overlay");
    RequestObject->SetString(L"HNSNetworkType", L"Overlay");

    PC_THROW_IF_FAILED_LOG(RequestObject->GetObjectList(L"Subnets", &objList));


    PC_THROW_IF_FAILED_LOG(objList->ObjectCount(&count));

    for (unsigned int it = 0; it < count; it++)
    {
        HNSObjPtr newObjPolicy;
        HNSObjPtr objSubnet;
        PC_THROW_IF_FAILED_LOG(objList->GetObjectW(it, &objSubnet));
        
        // create object and set object type.
        PC_THROW_IF_FAILED_LOG(PluginProvider->CreateObject(&newObjPolicy));
        PC_THROW_IF_FAILED_LOG(newObjPolicy->SetObjectType(HnsTypePolicy));

        // set vsid.
        PC_THROW_IF_FAILED_LOG(newObjPolicy->SetString(L"Type", L"VSID"));
        PC_THROW_IF_FAILED_LOG(newObjPolicy->SetInteger(L"VSID", 4096));

        objSubnet->AddListObject(L"Policies", newObjPolicy.get());
    }

}

////////////////////////////////////////////////////////////////////////////////
// ContainerHost::HandleEndpointPreCreate
////////////////////////////////////////////////////////////////////////////////
void ContainerHost::HandleEndpointPreCreate(
	_Inout_ HNSProviderPtr PluginProvider,
	_Inout_ HNSObjPtr RequestObject
)
{
	TRACE_ENTER();

	PCStringPtr vNetName;
	HNSObjPtr objNetwork;
	PCStringPtr hnsNetworkType;

	std::pair<std::wstring, std::wstring> endPtPair;

	// get the network first.
	PC_THROW_IF_FAILED_LOG(RequestObject->GetString(L"VirtualNetworkName", &vNetName));
	PC_THROW_IF_FAILED_LOG(PluginProvider->FindNetworkByName(vNetName.get(), &objNetwork));

	// get network type.
	PC_THROW_IF_FAILED_LOG(objNetwork->GetString(L"Type", &hnsNetworkType));

	if (::_wcsicmp(hnsNetworkType.get(), L"Overlay") != 0)
	{
		// no neeed to do anything if it is not overlay
		return;
	}

	BOOLEAN isRemote = FALSE;

	if (S_OK == RequestObject->PropertyExists(L"IsRemoteEndpoint"))
	{
		PC_THROW_IF_FAILED_LOG(RequestObject->GetBoolean(L"IsRemoteEndpoint", &isRemote));
	}

	if (!isRemote)
	{
		HNSObjPtr newObjPolicy;

		// create object and set object type.
		PC_THROW_IF_FAILED_LOG(PluginProvider->CreateObject(&newObjPolicy));
		PC_THROW_IF_FAILED_LOG(newObjPolicy->SetObjectType(HnsTypePolicy));

		PCStringPtr managmentIP;
		objNetwork->GetString(L"ManagementIP", &managmentIP);

		// set vsid.
		PC_THROW_IF_FAILED_LOG(newObjPolicy->SetString(L"Type", L"PA"));
		PC_THROW_IF_FAILED_LOG(newObjPolicy->SetString(L"PA", managmentIP.get()));

		RequestObject->AddListObject(L"Policies", newObjPolicy.get());
	}
}

#define  LogInFile(info) \
{ \
	HANDLE hFile = CreateFile(L"D:\\log.txt", GENERIC_WRITE, 0, NULL, OPEN_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL); \
	unsigned long writtenBytes; \
	SetFilePointer(hFile, 0, 0, FILE_END); \
	WriteFile(hFile, info.get(), (DWORD)(sizeof(wchar_t) * wcslen(info.get())), &writtenBytes, NULL); \
	CloseHandle(hFile); \
}

bool retrieveEndpointDetails(
	_Inout_ HNSProviderPtr& PluginProvider, 
	_Inout_ HNSObjPtr& RequestObject, 
	_Inout_ PCStringPtr& vNetName, 
	_Inout_ HNSObjPtr& objNetwork, 
	_Inout_ PCStringPtr& hnsNetworkType, 
	_Inout_ PCStringPtr& macAddress, 
	_Inout_ PCStringPtr& pa, 
	_Inout_ PCStringPtr& containerAddress)
{
	std::pair<std::wstring, std::wstring> endPtPair;

	// get the network first.
	PC_THROW_IF_FAILED_LOG(RequestObject->GetString(L"VirtualNetworkName", &vNetName));
	PC_THROW_IF_FAILED_LOG(PluginProvider->FindNetworkByName(vNetName.get(), &objNetwork));

	// get network type.
	PC_THROW_IF_FAILED_LOG(objNetwork->GetString(L"Type", &hnsNetworkType));

	if (::_wcsicmp(hnsNetworkType.get(), L"Overlay") != 0)
	{
		// we don't do anything if it is not overlay
		return true;
	}

	BOOLEAN isRemote = FALSE;

	if (S_OK == RequestObject->PropertyExists(L"IsRemoteEndpoint"))
	{
		PC_THROW_IF_FAILED_LOG(RequestObject->GetBoolean(L"IsRemoteEndpoint", &isRemote));
	}

	if (isRemote) // we gossip only about deletion of local endpoints
	{
		return true;
	}

	// Retrieve MAC
	if (S_OK == RequestObject->PropertyExists(L"MacAddress"))
	{
		PC_THROW_IF_FAILED_LOG(RequestObject->GetString(L"MacAddress", &macAddress));
		// let other's know about the mac address
		LogInFile(macAddress);
	}

	// Retrieve PA
	HNSObjListPtr objectList;
	RequestObject->GetObjectList(L"Policies", &objectList);
	DWORD count;
	objectList->ObjectCount(&count);
	for (DWORD i = 0; i<count; i++)
	{
		HNSObjPtr obj;
		objectList->GetObjectW(i, &obj);
		if (S_OK == obj->PropertyExists(L"PA"))
		{
			obj->GetString(L"PA", &pa);
			LogInFile(pa);
		}
	}

	//Retrieve CA
	if (S_OK == RequestObject->PropertyExists(L"IPAddress"))
	{
		PC_THROW_IF_FAILED_LOG(RequestObject->GetString(L"IPAddress", &containerAddress));
		LogInFile(containerAddress);
	}
	return false;
}

////////////////////////////////////////////////////////////////////////////////
// ContainerHost::HandleEndpointPostCreate
////////////////////////////////////////////////////////////////////////////////
void ContainerHost::HandleEndpointPostCreate(
	_Inout_ HNSProviderPtr PluginProvider,
	_Inout_ HNSObjPtr RequestObject
)
{
	TRACE_ENTER();

	PCStringPtr vNetName;
	HNSObjPtr objNetwork;
	PCStringPtr hnsNetworkType;
	PCStringPtr macAddress;
	PCStringPtr pa;
	PCStringPtr containerAddress;

	retrieveEndpointDetails(
		PluginProvider,
		RequestObject,
		vNetName,
		objNetwork,
		hnsNetworkType,
		macAddress,
		pa,
		containerAddress);

		// inform other nodes in overlay about macAddress, pa, and containerAddress 
}

////////////////////////////////////////////////////////////////////////////////
// ContainerHost::HandleEndpointPostDelete
////////////////////////////////////////////////////////////////////////////////
void ContainerHost::HandleEndpointPostDelete(
	_Inout_ HNSProviderPtr PluginProvider,
	_Inout_ HNSObjPtr RequestObject
)
{
	TRACE_ENTER();

	PCStringPtr vNetName;
	HNSObjPtr objNetwork;
	PCStringPtr hnsNetworkType;
	PCStringPtr macAddress;
	PCStringPtr pa;
	PCStringPtr containerAddress;

	retrieveEndpointDetails(
		PluginProvider,
		RequestObject,
		vNetName,
		objNetwork,
		hnsNetworkType,
		macAddress,
		pa,
		containerAddress);

	// inform other nodes in overlay about deletion of macAddress, pa, and containerAddress 
}