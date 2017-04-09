
#include <windows.h>
#include <objbase.h>
#include <stdio.h>
#include <string>
#include "HnsApi.h"
#include "HnsApi_i.c"

#include "../NetworkClientLibrary/Common/aquanetworkclient.h"
#include "Config.Containers.HNS.h"
#include <wil/Resource.h>
#include <ifdef.h>
#define SUCCESS L"\"Success\" : true"
using namespace std::string_literals; 

// {c8a58877-c594-47fe-9197-bd808576f1da} as defined in idl file
static const GUID server_clsid =
{ 0xc8a58877, 0xc594, 0x47fe, { 0x91, 0x97, 0xbd, 0x80, 0x85, 0x76, 0xf1, 0xda } };

bool ExecuteHNSRequest(LPWSTR *responsePtr, LPCWSTR Method, LPCWSTR Path, LPCWSTR Request);

std::wstring GetContainerNetworkId(wchar_t *hnsNetworkType)
{
	//LPWSTR responsePtr = NULL;
	wil::unique_cotaskmem_ptr<WCHAR> networkResponseJson;
	std::wstring request = L"";

	printf("%S\n", request.c_str());
	bool failed = ExecuteHNSRequest(wil::out_param(networkResponseJson), L"GET", L"/networks/", request.c_str());
	if(!failed)
	{
		Config::Containers::HNS::HNSNetworkResponse networkResponse;
		int a = Marshal::FromJson(networkResponseJson.get(), &networkResponse).first;

		printf("\nhrval=%d successed=%d\n\n", a, networkResponse.Success);

		for (std::vector<Config::Containers::HNS::HNSNetwork>::iterator nw = networkResponse.Output.begin(); nw != networkResponse.Output.end(); ++nw)
		{
			printf("ID=%S\n", (*nw).ID.c_str());
			printf("Name=%S\n", (*nw).Name.c_str());
			printf("Type=%d\n", (*nw).Type);
			printf("HNS Type=%S\n", (*nw).HNSNetworkType.c_str());
			if (_wcsicmp((*nw).HNSNetworkType.c_str(), hnsNetworkType) == 0)
			{
				return (*nw).ID.c_str();
			}
		}
	}
	
	if(networkResponseJson != NULL)
	{
		networkResponseJson.release();
	}
	printf("Unable to find network of type %S. Please check if it is created or not.", hnsNetworkType);
	return NULL;
}

DllExport int AddEndpoint(std::wstring endpointId, std::wstring macAddress, std::wstring ca, std::wstring pa, std::wstring isRemoteEndpoint)
{
	LPWSTR responsePtr = NULL;
	std::wstring vnetId = GetContainerNetworkId(L"Overlay");
	std::wstring request = L"{ \"ID\" : \"" + endpointId + L"\", \
	\"VirtualNetwork\" : \"" + vnetId + L"\", \
	\"IsRemoteEndpoint\" : " + isRemoteEndpoint + L", \
	\"IPAddress\" : \"" + ca + L"\", \
	\"MacAddress\" : \"" + macAddress + L"\", \
	\"Policies\" : [ { \"Type\": \"PA\", \"PA\" : \""+ pa + L"\"}]}";

	printf("%S\n", request.c_str());
	bool failed = ExecuteHNSRequest(&responsePtr, L"POST", L"/endpoints/", request.c_str());
	if (!failed)
	{
		printf("Remote endpoint creation succeeded for id:%S mac:%S ca:%S pa:%S vnetId:%S HNSResponse:%S\n", 
			endpointId.c_str(), macAddress.c_str(), ca.c_str(), pa.c_str(), vnetId.c_str(), responsePtr);
		if (responsePtr != NULL)
		{
			printf("Going to free response ptr\n");
			CoTaskMemFree(responsePtr);
		}
	}
	else
	{
		printf("Remote endpoint creation failed for id:%S mac:%S ca:%S pa:%S vnetId:%S HNSResponse:%S\n",
			endpointId.c_str(), macAddress.c_str(), ca.c_str(), pa.c_str(), vnetId.c_str(), responsePtr);
		
	}
	return failed;
}

DllExport int DeleteEndpoint(std::wstring endpointId)
{
	LPWSTR responsePtr = NULL;
	std::wstring path = L"/endpoints/" + endpointId;
	bool failed = ExecuteHNSRequest(&responsePtr, L"DELETE", path.c_str(), L"");
	if (!failed)
	{
		printf("Delete endpoint succeeded for id:%S HNSResponse:%S\n", endpointId.c_str(), responsePtr);
		if (responsePtr != NULL)
		{
			printf("Going to free response ptr\n");
			CoTaskMemFree(responsePtr);
			printf("Done \n");
		}
	}
	else
	{
		printf("Delete endpoint failed for id:%S HNSResponse:%S\n", endpointId.c_str(), responsePtr);
	}
	return failed;
}

void RemoveBraces(WCHAR* endpointIdStr, WCHAR** endpointIdStrWithoutBraces)
{
	printf("Going to remove braces from %S", endpointIdStr);
	getchar();
	size_t endpointIdStrLen = wcslen(endpointIdStr);
	printf("len %d\n", static_cast<int>(endpointIdStrLen));
	*endpointIdStrWithoutBraces = (WCHAR*)malloc((endpointIdStrLen - 1) * sizeof(WCHAR));
	printf("going to copy %llu %S", (endpointIdStrLen - 2), endpointIdStr + 1);
	wcsncpy_s((*endpointIdStrWithoutBraces), endpointIdStrLen - 2, (endpointIdStr + 1), endpointIdStrLen-2);
	getchar();
	(*endpointIdStrWithoutBraces)[endpointIdStrLen - 2] = '\0';
	printf("removed %S", *endpointIdStrWithoutBraces);
	getchar();
}

bool ExecuteHNSAttachEndpoint(const _GUID endpointId, UINT32 compartmentId)
{
	bool failed = true;

	HRESULT hr;
	hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
	if (FAILED(hr))
	{
		printf("Failed to CoInitializeEx 0x%x\n", hr);
		return failed;
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

	WCHAR* idstr = NULL;

	StringFromCLSID(endpointId, &idstr);
	printf("Attachendpoint('%S','%d');\n", idstr, compartmentId);
	if (idstr != NULL)
	{
		CoTaskMemFree(idstr);
	}

	hr = pInterface->AttachEndpoint(&endpointId, compartmentId);

	printf("Call returned 0x%x\n", hr);

	pInterface->Release();

	failed = false;
Cleanup:
	CoUninitialize();

	return failed;

}
bool ExecuteHNSDetachEndpoint(const _GUID endpointId)
{
	bool failed = true;

	HRESULT hr;
	hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
	if (FAILED(hr))
	{
		printf("Failed to CoInitializeEx 0x%x\n", hr);
		return failed;
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

	WCHAR* idstr = NULL;

	StringFromCLSID(endpointId, &idstr);
	printf("Detachendpoint('%S');\n", idstr);
	if (idstr != NULL)
	{
		CoTaskMemFree(idstr);
	}

	hr = pInterface->DetachEndpoint(&endpointId);

	printf("Call returned 0x%x\n", hr);

	pInterface->Release();

	failed = false;
Cleanup:
	CoUninitialize();

	return failed;

}

DllExport int AddHostLocalEndpoint(std::wstring hostLocalIp)
{
	//LPWSTR responsePtr = NULL;
	wil::unique_cotaskmem_ptr<WCHAR> endPointCreateResponseJson;
	Config::Containers::HNS::HNSEndpointResponse endPointResponse;
	std::wstring vnetId = GetContainerNetworkId(L"Overlay");
	std::wstring request = L"{ \"VirtualNetwork\" : \"" + vnetId + L"\", \
	\"IPAddress\" : \"" + hostLocalIp + L"\"}]}";
	WCHAR* endpointIdStr = NULL;
	WCHAR *endpointIdStrWithoutBraces = NULL;
	wchar_t *req = NULL;

	printf("HNS Request: %S\n", request.c_str());
	bool failed = ExecuteHNSRequest(wil::out_param(endPointCreateResponseJson), L"POST", L"/endpoints/", request.c_str());
	if (!failed)
	{
		if (endPointCreateResponseJson != NULL)
		{
			printf("HostLocal endpoint creation succeeded for pa:%S vnetId:%S HNSResponse:%S\n",
				hostLocalIp.c_str(), vnetId.c_str(), endPointCreateResponseJson.get());
			Marshal::FromJson(endPointCreateResponseJson.get(), &endPointResponse);
			StringFromCLSID(endPointResponse.Output.ID, &endpointIdStr);
			// The content id we want to return is the GUID without the braces {}.
			
			/*if (endpointIdStr != NULL)
			{
				RemoveBraces(endpointIdStr, &endpointIdStrWithoutBraces);
			}*/
			endpointIdStrWithoutBraces = endpointIdStr + 1;
			endpointIdStrWithoutBraces[wcslen(endpointIdStr) - 2] = L'\0';
			printf("Created hostlocal EndPoint with Id: %S %S\n", endpointIdStr, endpointIdStrWithoutBraces);
			printf("Going to release response ptr\n");
			endPointCreateResponseJson.release();
			printf("Done\n");
		}
		else
		{
			failed = true;
			printf("HostLocal endpoint creation succeeded but returned nothing in response for pa:%S vnetId:%S\n",
				hostLocalIp.c_str(), vnetId.c_str());			
		}
	}
	else
	{
		printf("HostLocal endpoint creation failed for pa:%S vnetId:%S HNSResponse:%S\n",
			hostLocalIp.c_str(), vnetId.c_str(), endPointCreateResponseJson.get());
		endPointCreateResponseJson.release();
	}

	LPWSTR responsePtr = NULL;
	if (!failed)
	{
		printf("Going to attach hostlocal ip %S with ID %S to overlay.\n", hostLocalIp.c_str(), endpointIdStrWithoutBraces);
		failed = ExecuteHNSAttachEndpoint(endPointResponse.Output.ID, NET_IF_COMPARTMENT_ID_PRIMARY);

		if (!failed)
		{
			printf("HostLocal endpoint attach succeeded for pa:%S vnetId:%S endpointID:%S\n",
				hostLocalIp.c_str(), vnetId.c_str(), endpointIdStrWithoutBraces);			
		}
		else
		{
			printf("HostLocal endpoint attach failed for pa:%S vnetId:%S HNSResponse:%S\nWill delete the host local endpoint..",
				hostLocalIp.c_str(), vnetId.c_str(), responsePtr);
			if(DeleteEndpoint(endpointIdStrWithoutBraces)==1)
			{
				printf("Deletion of host local endpoint failed");
			}

		}
	}
	if (responsePtr != NULL)
	{
		printf("Going to free response ptr\n");
		CoTaskMemFree(responsePtr);
	}
	if(endpointIdStr != NULL)
	{
		CoTaskMemFree(endpointIdStr);
	}
	if(req != NULL)
	{
		free(req);
	}

	return failed;
}

DllExport int DeleteLocalEndpointWithIp(wchar_t *hostLocalIp, bool detachIfHostLocalIp)
{
	wil::unique_cotaskmem_ptr<WCHAR> endpointsResponseJson;
	std::wstring request = L"";
	int retval = 0;

	printf("%S\n", request.c_str());
	bool failed = ExecuteHNSRequest(wil::out_param(endpointsResponseJson), L"GET", L"/endpoints/", request.c_str());
	if (!failed)
	{
		Config::Containers::HNS::HNSEndpointsResponse endpointsResponse;
		int hr = Marshal::FromJson(endpointsResponseJson.get(), &endpointsResponse).first;
		if(hr!=1) //failed
		{
			
		}
		printf("\nhrval=%d successed=%d\n\n", hr, endpointsResponse.Success);

		for (std::vector<Config::Containers::HNS::HNSEndpoint>::iterator ep = endpointsResponse.Output.begin(); ep != endpointsResponse.Output.end(); ++ep)
		{
			printf("Checking IP=%S\n", (*ep).IPAddress.c_str());
			printf("Name=%S\n", (*ep).Name.c_str());
			printf("VnetName=%ws\n", (*ep).VirtualNetworkName.c_str());
			printf("Gateway=%S\n", (*ep).GatewayAddress.c_str());
			if (_wcsicmp((*ep).IPAddress.c_str(), hostLocalIp) == 0)
			{
				printf("Endpoint with IP=%S will be deleted as per request..\n", (*ep).IPAddress.c_str());
				WCHAR * epwb;
				StringFromCLSID((*ep).ID, &epwb);
				WCHAR *epwtb = epwb + 1;
				epwtb[wcslen(epwb) - 2] = L'\0';
				// first detach, then delete

				if (detachIfHostLocalIp)
				{
					ExecuteHNSDetachEndpoint((*ep).ID);
				}

				if(DeleteEndpoint(epwtb)==1)
				{
					printf("Removal of endpoint failed\n");
					retval = 1;
				}
				free(epwb);
				break;
			}

			printf("Endpoint with IP=%S is not the one to be deleted as per delete request..\n", (*ep).IPAddress.c_str());		
		}
	}

	if (endpointsResponseJson != NULL)
	{
		endpointsResponseJson.release();
	}
	return retval;
}

bool ExecuteHNSRequest(LPWSTR *responsePtr, LPCWSTR Method, LPCWSTR Path, LPCWSTR Request)
{
	bool failed = true;

	HRESULT hr;
	hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
	if (FAILED(hr))
	{
		printf("Failed to CoInitializeEx 0x%x\n", hr);
		return failed;
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
	// pInterface->AttachEndpoint()

	printf("Call returned 0x%x\n", hr);

	if (*responsePtr != NULL)
	{
		printf("%S\n", *responsePtr);		
	}

	pInterface->Release();

	failed = false;
Cleanup:
	CoUninitialize();

	return failed;
}

int wmain(int argc, const wchar_t** argv)
{
	UNREFERENCED_PARAMETER(argc);

	UNREFERENCED_PARAMETER(argv);
	AddEndpoint(L"c64f5c74-c657-497a-bddc-6c5aaaa8ea88",
							L"00-15-5D-C1-16-66", 
							L"10.10.0.5", 
							L"11.1.0.124",
							L"true");
	getchar();
	DeleteEndpoint(L"c64f5c74-c657-497a-bddc-6c5aaaa8ea88");
	getchar();
	AddHostLocalEndpoint(L"10.10.0.28");
	getchar();
	DeleteLocalEndpointWithIp(L"10.10.0.28", true);
}
