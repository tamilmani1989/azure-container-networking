
#include <windows.h>
#include <objbase.h>
#include <stdio.h>
#include <string>
#include "HnsApi.h"
#include "HnsApi_i.c"
#include "../NetworkClientLibrary/Common/aquanetworkclient.h"


// {c8a58877-c594-47fe-9197-bd808576f1da} as defined in idl file
static const GUID server_clsid =
{ 0xc8a58877, 0xc594, 0x47fe, { 0x91, 0x97, 0xbd, 0x80, 0x85, 0x76, 0xf1, 0xda } };


void Usage(const wchar_t* progname)
{
    printf("Usage:\n");
    printf("  %S Operation <Method> <Path> <blank or raw Request or request filename>\n\n", progname);
}

DllExport int PersistRemoteEndpoint(char *macAddress, char *ca, char *pa)
{
	UNREFERENCED_PARAMETER(macAddress);
	UNREFERENCED_PARAMETER(ca);
	UNREFERENCED_PARAMETER(pa);
	return 0;
}

DllExport int DeleteRemoteEndpoint(char *macAddress, char *ca, char *pa)
{
	UNREFERENCED_PARAMETER(macAddress);
	UNREFERENCED_PARAMETER(ca);
	UNREFERENCED_PARAMETER(pa);
	return 0;
}

DllExport char * GetContainerNetworkId()
{	
	return NULL;
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


int ExecuteHNSRequest(LPWSTR responsePtr, LPCWSTR Method, LPCWSTR Path, LPCWSTR Request)
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
	hr = pInterface->Request(Method, Path, Request, &responsePtr);

	printf("Call returned 0x%x\n", hr);

	if (responsePtr != NULL)
	{
		printf("%S\n", responsePtr);
		CoTaskMemFree(responsePtr);
	}

	pInterface->Release();

	returnCode = 0;
Cleanup:
	CoUninitialize();

	return returnCode;
}

int wmainmethod(int argc, const wchar_t** argv)
{
    LPWSTR responsePtr = NULL;
    LPCWSTR Method, Path, Request;
    std::wstring RequestData = L"";

    // Determine what to do.
    if (argc < 3)
    {
        Usage(argv[0]);
        return 1;
    }

    Method = argv[1];
    Path = argv[2];

    if (argc == 3)
    {
        // No request
        Request = L"";
    }
    else
    {
        FILE * f = NULL;
        if (argc == 4)
        {
            // Is it a filename?
            _wfopen_s(&f, argv[3], L"rt, ccs=UTF-8");
        }
        if (f != NULL)
        {
            // It is a file.
            fseek(f, 0, SEEK_END);
            long length = ftell(f);
            RequestData.reserve(length * 2);
            fseek(f, 0, SEEK_SET);
            WCHAR wc;
            while ((wc = fgetwc(f)) != WEOF)
            {
                if (wc == 0xFEFF) continue; // For some reason it's emitting the utf8 mark...
                RequestData.push_back(wc); // Read the characters slowly...
            }
            fclose(f);
        }
        else
        {
            // Convert the rest of the parameters into an argument
            for (int n = 3; n < argc; n++)
            {
                if (n != 3) RequestData += L" ";
                RequestData += argv[n];
            }
        }
        Request = RequestData.c_str();
    }

    return ExecuteHNSRequest(responsePtr, Method, Path, Request);
}
