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
#include "../aquanix/Common/aquanixext.h"
//#include "../Logging/Logging.h"

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


void  LogStringInFile(wchar_t *info)
{
	HANDLE hFile = CreateFile(L"D:\\log.txt", GENERIC_WRITE, 0, NULL, OPEN_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
	unsigned long writtenBytes; \
		SetFilePointer(hFile, 0, 0, FILE_END); \
		WriteFile(hFile, info, (DWORD)(wcslen(info)), &writtenBytes, NULL);
	CloseHandle(hFile);
}
void  LogNarrowStringInFile(const char *info)
{
	HANDLE hFile = CreateFile(L"D:\\log.txt", GENERIC_WRITE, 0, NULL, OPEN_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
	unsigned long writtenBytes; \
		SetFilePointer(hFile, 0, 0, FILE_END); \
		WriteFile(hFile, info, (DWORD)(sizeof(char) * strlen(info)), &writtenBytes, NULL);
	CloseHandle(hFile);
}

bool retrieveLocalEndpointDetails(
	_Inout_ HNSProviderPtr& PluginProvider, 
	_Inout_ HNSObjPtr& RequestObject, 
	_Inout_ PCStringPtr& vNetName, 
	_Inout_ HNSObjPtr& objNetwork, 
	_Inout_ PCStringPtr& hnsNetworkType, 
	_Inout_ PCStringPtr& macAddress, 
	_Inout_ PCStringPtr& pa, 
	_Inout_ PCStringPtr& containerAddress, 
	_Inout_ GUID& ID,
	int saveOrDelete)
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
		}
	}

	//Retrieve CA
	if (S_OK == RequestObject->PropertyExists(L"IPAddress"))
	{
		PC_THROW_IF_FAILED_LOG(RequestObject->GetString(L"IPAddress", &containerAddress));
	}

	//Retrieve ID
	if (S_OK == RequestObject->PropertyExists(L"ID"))
	{
		//PC_THROW_IF_FAILED_LOG(
			RequestObject->GetGuid(L"ID", &ID);
			WCHAR *strId;
			StringFromCLSID(ID, &strId);
	}


	WCHAR * idwb;
	StringFromCLSID(ID, &idwb);
	WCHAR *idwtb = idwb + 1;
	idwtb[wcslen(idwb) - 2] = L'\0';
	LogStringInFile(idwtb);
	size_t len = wcslen(idwtb);
	char *idwtb_s = (char *)malloc((len + 2) * sizeof(char));
	size_t readlen;
	wcstombs_s(&readlen, idwtb_s, len + 1, idwtb, len + 1);

	LogNarrowStringInFile(idwtb_s);
	WCHAR * pa_ch = pa.get();
	LogStringInFile(pa_ch);

	len = wcslen(pa_ch);
	char *pa_ch_s = (char *)malloc((len + 2) * sizeof(char));
	wcstombs_s(&readlen, pa_ch_s, len + 1, pa_ch, len + 1);

	WCHAR *ca_ch = containerAddress.get();
	LogStringInFile(ca_ch);
	len = wcslen(ca_ch);
	char *ca_ch_s = (char *)malloc((len + 2) * sizeof(char));
	wcstombs_s(&readlen, ca_ch_s, len + 1, ca_ch, len + 1);


	WCHAR *mac_ch = macAddress.get();
	LogStringInFile(mac_ch);
	len = wcslen(mac_ch);
	char *mac_ch_s = (char *)malloc((len + 2) * sizeof(char));
	wcstombs_s(&readlen, mac_ch_s, len + 1, mac_ch, len + 1);

	if (saveOrDelete == 1)
	{
		SendLocalMapping(idwtb_s, ca_ch_s, mac_ch_s);
	}
	else
	{
		LogStringInFile(L"Going to delete local mapping via nix");
		DeleteLocalMapping(idwtb_s);
		LogStringInFile(L"Deleted local mapping via nix");
	}

	free(idwtb_s);
	free(pa_ch_s);
	free(ca_ch_s);
	free(mac_ch_s);
	free(idwb);

	/*
	PCStringPtr newLine = (wil::unique_cotaskmem_string) L"\n";
	LogInFile(newLine);*/
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
	GUID ID;
	
	retrieveLocalEndpointDetails(
		PluginProvider,
		RequestObject,
		vNetName,
		objNetwork,
		hnsNetworkType,
		macAddress,
		pa,
		containerAddress,
		ID,
		1);
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
	GUID ID;
	LogStringInFile(L"Going ot delete endpoint");
	retrieveLocalEndpointDetails(
		PluginProvider,
		RequestObject,
		vNetName,
		objNetwork,
		hnsNetworkType,
		macAddress,
		pa,
		containerAddress,
		ID,
		2);

	LogStringInFile(L"Deleted endpoint");
	// inform other nodes in overlay about deletion of macAddress, pa, and containerAddress 
}

DWORD ReceiveMapping(const char *endpointId, const char *hostAddress, const char *containerAddress, const char *macAddress)
{
	LogNarrowStringInFile(endpointId);
	LogNarrowStringInFile(hostAddress);
	LogNarrowStringInFile(containerAddress);
	LogNarrowStringInFile(macAddress);
	return 0;
}


DWORD DeleteRemoteMapping(const char *endpointId)
{
	//LogNarrowStringInFile("DeleteRemoteendpoint called\n");
	LogNarrowStringInFile(endpointId);
	return 0;
}

DWORD GetLocalMappings(EndpointMapping **endpointMappings, USHORT *mappingsCount)
{
	UNREFERENCED_PARAMETER(endpointMappings);
	UNREFERENCED_PARAMETER(mappingsCount);
	return 0;
}

DWORD FreeEndpointMappings (EndpointMapping *endpointMappings, USHORT mappingsCount)
{
	UNREFERENCED_PARAMETER(endpointMappings);
	UNREFERENCED_PARAMETER(mappingsCount);
	return 0;
}
////////////////////////////////////////////////////////////////////////////////
// ContainerHost::HandleEndpointPostCreate
////////////////////////////////////////////////////////////////////////////////
void ContainerHost::Initialize(_Inout_ HNSProviderPtr PluginProvider)
{
	LogStringInFile(L"Initialize Called");
	//AquaLogging::Log(info, "Initializing HNS Overlay Plugin");
	char **remoteHostAddresses = new char*{ "127.0.0.1" };
	InitializeAquaNix("127.0.0.1", remoteHostAddresses, NULL, ReceiveMapping, DeleteRemoteMapping, GetLocalMappings, FreeEndpointMappings, 10051, 10050);
	//GetLocalMappings()
	delete []remoteHostAddresses;
	LogStringInFile(L"Initialize finished");
}