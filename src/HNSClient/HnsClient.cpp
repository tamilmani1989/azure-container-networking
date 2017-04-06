
#include <windows.h>
#include <objbase.h>
#include <stdio.h>
#include <string>
#include "HnsApi.h"
#include "HnsApi_i.c"

#include "../NetworkClientLibrary/Common/aquanetworkclient.h"
#include "Config.Containers.HNS.h"
#include <wil/Resource.h>
#define SUCCESS L"\"Success\" : true"
using namespace std::string_literals; 

// {c8a58877-c594-47fe-9197-bd808576f1da} as defined in idl file
static const GUID server_clsid =
{ 0xc8a58877, 0xc594, 0x47fe, { 0x91, 0x97, 0xbd, 0x80, 0x85, 0x76, 0xf1, 0xda } };

int ExecuteHNSRequest(LPWSTR *responsePtr, LPCWSTR Method, LPCWSTR Path, LPCWSTR Request);

std::wstring GetContainerNetworkId(wchar_t *hnsNetworkType)
{
	//LPWSTR responsePtr = NULL;
	wil::unique_cotaskmem_ptr<WCHAR> networkResponseJson;
	std::wstring request = L"";

	printf("%S\n", request.c_str());
	ExecuteHNSRequest(wil::out_param(networkResponseJson), L"GET", L"/networks/", request.c_str());

	//ChkHR(HNSCall(L"POST", L"/networks/", networkJson.c_str(), wil::out_param(networkResponseJson)));
	
	Config::Containers::HNS::HNSNetworkResponse networkResponse;
	int a = Marshal::FromJson(networkResponseJson.get(), &networkResponse).first;

	printf("\nhrval=%d successed=%d\n\n", a, networkResponse.Success);

	for (std::vector<Config::Containers::HNS::HNSNetwork>::iterator nw = networkResponse.Output.begin(); nw != networkResponse.Output.end(); ++nw)
	{
		printf("ID=%S\n", (*nw).ID.c_str());
		printf("Name=%S\n", (*nw).Name.c_str());
		printf("Type=%d\n", (*nw).Type);
		printf("HNS Type=%S\n", (*nw).HNSNetworkType.c_str());
		if(_wcsicmp((*nw).HNSNetworkType.c_str(), hnsNetworkType)==0)
		{
			return (*nw).ID.c_str();
		}
	}
	printf("Unable to find network of type %S. Please check if it is created or not.", hnsNetworkType);
	return NULL;
}

DllExport int PersistRemoteEndpoint(std::wstring endpointId, std::wstring macAddress, std::wstring ca, std::wstring pa)
{
	LPWSTR responsePtr = NULL;
	std::wstring vnetId = GetContainerNetworkId(L"Overlay");
	std::wstring request = L"{ \"ID\" : \"" + endpointId + L"\", \
	\"VirtualNetwork\" : \"" + vnetId + L"\", \
	\"IsRemoteEndpoint\" : true, \
	\"IPAddress\" : \"" + ca + L"\", \
	\"MacAddress\" : \"" + macAddress + L"\", \
	\"Policies\" : [ { \"Type\": \"PA\", \"PA\" : \""+ pa + L"\"}]}";

	printf("%S\n", request.c_str());
	ExecuteHNSRequest(&responsePtr, L"POST", L"/endpoints/", request.c_str());
	printf("got back %ls\n", responsePtr);
	if(responsePtr != NULL)
	{
		printf("Going to free response ptr\n");
		CoTaskMemFree(responsePtr);
	}
	return 0;
}

DllExport int DeleteRemoteEndpoint(char *macAddress, char *ca, char *pa)
{
	UNREFERENCED_PARAMETER(macAddress);
	UNREFERENCED_PARAMETER(ca);
	UNREFERENCED_PARAMETER(pa);
	return 0;
}

DllExport int CreateHostLocalEndpoint(char *hostLocalIp)
{
	UNREFERENCED_PARAMETER(hostLocalIp);
	return 0;
}

DllExport int DeleteHostLocalEndpoint(char *hostLocalIp)
{
	UNREFERENCED_PARAMETER(hostLocalIp);
	return 0;
}


int ExecuteHNSRequest(LPWSTR *responsePtr, LPCWSTR Method, LPCWSTR Path, LPCWSTR Request)
{
	int returnCode = 1;

	HRESULT hr;
	hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
	if (FAILED(hr))
	{
		printf("Failed to CoInitializeEx 0x%x\n", hr);
		return 1;
	}

	IHNSApi* pInterface = NULL;

	hr = CoCreateInstance(server_clsid, NULL, CLSCTX_LOCAL_SERVER, 
										IID_IHNSApi, (void**)&pInterface);
	if (FAILED(hr))
	{
		printf("Failed to CoCreateInstance 0x%x\n", hr);
		goto Cleanup;
	}

	printf("Got class object, calling...  ");


	printf("Request('%S','%S','%S');\n", Method, Path, Request); 
	hr = pInterface->Request(Method, Path, Request, responsePtr);

	printf("Call returned 0x%x\n", hr);

	if (*responsePtr != NULL)
	{
		printf("%S\n", *responsePtr);		
	}

	pInterface->Release();

	returnCode = 0;
Cleanup:
	CoUninitialize();

	return returnCode;
}

int wmain(int argc, const wchar_t** argv)
{
	UNREFERENCED_PARAMETER(argc);
	UNREFERENCED_PARAMETER(argv);
	PersistRemoteEndpoint(L"c64f5c74-c657-497a-bddc-6c5aaaa8ea88", 
							L"00-15-5D-C1-16-66", 
							L"10.10.0.5", 
							L"11.1.0.124");
}
