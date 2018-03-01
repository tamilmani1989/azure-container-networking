/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

	NetworkContainerTest.cpp

Abstract:

	Tests NetworkContainer Functionalities.

Author:

	Yongli Chen

--*/


#ifndef UNICODE
#define UNICODE
#endif

#define _WINSOCK_DEPRECATED_NO_WARNINGS

#include <windows.h>
#include <stdio.h>
#include <string>
#include "NetworkContainerTest.h"
#undef __WFILE__
#undef __WFUNCTION__
#include "NetworkContainer.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;
using namespace WEX::TestExecution;

bool
SetupLogging(
	__in PCWSTR LogLocation
)
/*++

Routine Description:

	Helper to set up logging.

Arguments:

	LogLocation - Location of the log..

Return Value:

	True if setup succeeded, False otherwise.

--*/
{
	DWORD err;
	HRESULT hr = S_OK;

	if (CreateDirectory(LogLocation, NULL) == 0) {

		err = GetLastError();
		if (err != ERROR_ALREADY_EXISTS) {

			std::wcout << L"Failed to set up logging directory. Error: [0" << hex << err << " ]" << std::endl;
			return false;
		}
	}

	hr = LOGSETUP(LogLocation, L"NetworkContainer");

	return hr == S_OK ? true : false;
}

HRESULT ConvertToINADDR(_In_ wchar_t* Str, _Out_ IN_ADDR& Retval)
/*++

Routine Description:

	This routine convert ip address from string representation to IN_ADDR struct.

Arguments:

	Str: string representation of ip address.

	Retval: IN_ADDR representation of ip address.

Return Value:

	HRESULT - Returns if the conversion succeeds or not.

--*/
{
	DWORD ret;
	HRESULT hr = S_OK;
	PCWSTR temp = NULL;
	ret = RtlIpv4StringToAddress(Str, true, &temp, &Retval);
	if (ret)
	{
		hr = HRESULT_FROM_WIN32(GetLastError());
		if (hr != S_OK)
		{
			wcout << L"NetworkContainer\\Main.cpp:: RtlIpv4StringToAddress function failed with error: " << hex << ret << " "
				<< "Address: " << Str << " "
				<< ": " << hex << hr << endl;
		}
	}
	return hr;
}

NetworkContainerTest::NetworkContainerTest()
/*++

Routine Description:

	This constructor initializes data members of the class as appropriate.

Arguments:

	None.

Return Value:

	None.

--*/
{
}

NetworkContainerTest::~NetworkContainerTest()
/*++

Routine Description:

	Destructor for NetworkContainerTest.

Arguments:

	None.

Return Value:

	None.

--*/
{
}

bool NetworkContainerTest::TestSetup()
/*++

Routine Description:

	TAEF Setup. This API is called before invoking test APIs. 
	
	This API sets up test adapter which is consider as primary nic.

Arguments:

	None

Return Value:

	None

--*/
{
	HRESULT hr = S_OK;
	hr = SetupLogging(LOG_PATH);
	if (FAILED(hr))
	{
		wprintf(L"Failed to setup debug logging due to error [%0x]", hr);
		return false;
	}

	return true;
}

bool NetworkContainerTest::TestCleanup()
/*++

Routine Description:

	TAEF Cleanup. This API is called after invoking test APIs

Arguments:

	None

Return Value:

	None

--*/
{

	HRESULT hr = S_OK;

	wstring name(L"test");

	NetworkContainer nc;
	hr = nc.DeleteLoopbackAdapter(name);

	return hr == S_OK ? true : false;

}

void NetworkContainerTest::TestFindAdapterByDisplayName()
/*++

Routine Description:

	This routine tests FindAdapterByDisplayName.

Argument:

	Name: Display name of the loopback adapter.

Return Value:

	None.

--*/
{
	wstring name(L"test");
	HRESULT hr = S_OK;

	NetworkContainer nc;

	LOGINFOW(L"FindAdapterByDisplayName when no adapter presents. [1/2]");
	hr = nc.FindAdapterByDisplayName(name);
	VERIFY_ARE_EQUAL(hr, 
		HRESULT_FROM_WIN32(ERROR_NOT_FOUND), 
		L"FindAdapterByDisplayName passed when no adapter presents. [1/2]"
	);

	LOGINFOW(L"CreateLoopbackAdapter when no adapter presents.");
	hr = nc.CreateLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"Loopback adapter created");

	LOGINFOW(L"FindAdapterByDisplayName when adapter presents. [2/2]");
	hr = nc.FindAdapterByDisplayName(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"FindAdapterByDisplayName passed when adapter presents. [2/2]"
	);

	LOGINFOW(L"DeleteLoopbackAdapter in TestFindAdapterByDisplayName.")
	hr = nc.DeleteLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr, S_OK, L"Removed adapter.");

}

void NetworkContainerTest::TestCreateLoopbackAdapter()
/*++
	
Routine Description:

	This routine tests if the loopback adapter is created successfully or not.

Argument:

	None.

Return Value:

	None.

--*/
{	
	HRESULT hr = S_OK;
	wstring name(L"test");

	NetworkContainer nc;
	LOGINFOW(L"CreateLoopbackAdapter when no adapter presents.")
		hr = nc.CreateLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"CreateLoopbackAdapter passed when no adapter presents. [1/2]"
	);

	LOGINFOW(L"CreateLoopbackAdapter.")
	hr = nc.CreateLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"CreateLoopbackAdapter passed when adapter presents. [2/2]"
	);

	LOGINFOW(L"DeleteLoopbackAdapter in TestCreateLoopbackAdapter.")
	hr = nc.DeleteLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr, S_OK, L"Removed adapter.");

}

void NetworkContainerTest::TestConfigLoopbackAdapter()
/*++

Routine Description:

	This routine tests if the loopback adapter is created successfully or not.

Argument:

	None.

Return Value:

	None.

--*/
{
	HRESULT hr = S_OK;

	wstring name(L"test");
	wstring ipStr = L"1.2.3.4";
	wstring netmaskStr = L"255.255.255.0";
	wstring gatewayStr = L"4.3.2.1";
	wstring dnsStr1 = L"8.8.8.8";
	wstring dnsStr2 = L"7.7.7.7";
	IN_ADDR ip, netmask, gateway, dns1, dns2 = { 0 };
	bool whsFlag = false, whrFlag = true;
	hr = ConvertToINADDR((wchar_t*)ipStr.c_str(), ip);
	VERIFY_ARE_EQUAL(hr, S_OK, L"ip address parsed successfully.");
	hr = ConvertToINADDR((wchar_t*)netmaskStr.c_str(), netmask);
	VERIFY_ARE_EQUAL(hr, S_OK, L"netmask address parsed successfully.");
	hr = ConvertToINADDR((wchar_t*)gatewayStr.c_str(), gateway);
	VERIFY_ARE_EQUAL(hr, S_OK, L"gateway address parsed successfully.");
	hr = ConvertToINADDR((wchar_t*)dnsStr1.c_str(), dns1);
	VERIFY_ARE_EQUAL(hr, S_OK, L"dns1 address parsed successfully.");
	hr = ConvertToINADDR((wchar_t*)dnsStr2.c_str(), dns2);
	VERIFY_ARE_EQUAL(hr, S_OK, L"dns2 address parsed successfully.");
	vector<IN_ADDR> dnsVec;
	dnsVec.push_back(dns1);
	dnsVec.push_back(dns2);

	NetworkContainer nc(name, ip, netmask, gateway, dnsVec);
	LOGINFOW(L"ConfigLoopbackAdapter when no adapter presents.")
	hr = nc.ConfigLoopbackAdapter(name, ip, netmask, gateway, dnsVec, whsFlag, whrFlag);
	VERIFY_ARE_EQUAL(hr, 
		HRESULT_FROM_WIN32(ERROR_NOT_FOUND), 
		L"ConfigLoopbackAdapter test passed when no adapter presents. [1/2]"
	);

	LOGINFOW(L"CreateLoopbackAdapter.")
	hr = nc.CreateLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"Created loopback adapter"
	);

	LOGINFOW(L"ConfigLoopbackAdapter when adapter presents.")
	hr = nc.ConfigLoopbackAdapter(name, ip, netmask, gateway, dnsVec, whsFlag, whrFlag);
	VERIFY_ARE_EQUAL(S_OK,
		S_OK,
		L"ConfigLoopbackAdapter test passed when adapter presents. [2/2]"
	);

	LOGINFOW(L"DeleteLoopbackAdapter in TestConfigLoopbackAdapter.")
	hr = nc.DeleteLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr, S_OK, L"Removed adapter.");
}

void NetworkContainerTest::TestDeleteLoopbackAdapter()
/*++

Routine Description:

	This routine tests if the loopback adapter is deleted successfully or not.

Argument:

	Name: Display name of the loopback adapter.

Return Value:

	None.

--*/
{
	HRESULT hr = S_OK;
	wstring name(L"test");

	NetworkContainer nc;
	LOGINFOW(L"DeleteLoopbackAdapter when no adapter presents.")
	hr = nc.DeleteLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"DeleteLoopbackAdapter passed when no adapter presents. [1/2]"
	);

	LOGINFOW(L"CreateLoopbackAdapter.")
		hr = nc.CreateLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"Created loopback adapter"
	);

	LOGINFOW(L"DeleteLoopbackAdapter when adapter presents.")
		hr = nc.DeleteLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"DeleteLoopbackAdapter passed when no adapter presents. [1/2]"
	);
}

int wmain(int argc, wchar_t *argv[])
{
	UNREFERENCED_PARAMETER(argc);
	UNREFERENCED_PARAMETER(argv);
	return 0;
}