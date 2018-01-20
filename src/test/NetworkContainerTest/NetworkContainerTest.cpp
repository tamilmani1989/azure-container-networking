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

HRESULT SetupLogging(
)
{
	DWORD error;
	HRESULT hr;
	wchar_t executablePath[MAX_PATH];

	LPWSTR executableFolder = NULL;
	LPWSTR logFolder = NULL;
	LPWSTR logFilePathWithPrefix = NULL;
	if (!GetModuleFileName(NULL, executablePath, MAX_PATH))
	{
		error = GetLastError();
		wprintf(L"Failed to retrieve process working directory (%d)\n", error);
		RD_E_WIN32(error, Cleanup);
	}

	hr = GetDirectoryFromPath(executablePath, &executableFolder);

	wprintf(L"Retrieved executable folder[%s] [%0x]", executableFolder, hr);

	RD_E_HR(hr, Cleanup);

	error = AppendStrings(&logFolder,
		2,
		executableFolder,
		L"\\Logs\\");


	wprintf(L"Creating log folder[%s] [%d]", logFolder, error);

	RD_E_WIN32(error, Cleanup);

	if (!CreateDirectory(logFolder, NULL))
	{
		error = GetLastError();
		if (error != ERROR_ALREADY_EXISTS)
		{
			wprintf(L"Failed to create log folder [%s] due to erroro (%d)\n", logFolder, error);
			RD_E_WIN32(error, Cleanup);
		}
	}

	error = AppendStrings(&logFilePathWithPrefix,
		2,
		logFolder,
		LOG_FILE_PREFIX);
	RD_E_WIN32(error, Cleanup);

	hr = RD_TRACE_INITIALIZE_CUSTOM_SIZE(RD_TRACE_OUTPUT_FILE,
		RD_TRACE_ALL,
		logFilePathWithPrefix,
		NULL,
		0,
		NULL,
		RD_TRACE_ALL,
		10,
		RD_TRACE_LOG_LIMIT);

Cleanup:

	STR_FREE(executableFolder);
	STR_FREE(logFolder);
	STR_FREE(logFilePathWithPrefix);

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

	hr = SetupLogging();
	if (FAILED(hr))
	{
		wprintf(L"Failed to setup logging due to error [%0x]", hr);
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
	return true;
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
	hr = nc.FindAdapterByDisplayName(name);
	VERIFY_ARE_EQUAL(hr, 
		HRESULT_FROM_WIN32(ERROR_NOT_FOUND), 
		L"FindAdapterByDisplayName passed when no adapter presents. [1/2]");

	hr = nc.CreateLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"Loopback adapter created");

	hr = nc.FindAdapterByDisplayName(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"FindAdapterByDisplayName passed when adapter presents. [2/2]");

	hr = nc.DeleteLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"Removed adapter.");
}


void NetworkContainerTest::TestCreateLoopbackAdapter()
/*++
	
Routine Description:

	This routine tests if the loopback adapter is created successfully or not.

Argument:

	Name: Display name of the loopback adapter.

Return Value:

	None.

--*/
{
	HRESULT hr = S_OK;
	wstring name(L"test");
	NetworkContainer nc;
	hr = nc.CreateLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		S_OK,
		L"CreateLoopbackAdapter passed when no adapter presents. [1/2]");

	hr = nc.CreateLoopbackAdapter(name);
	VERIFY_ARE_EQUAL(hr,
		HRESULT_FROM_WIN32(ERROR_ALIAS_EXISTS),
		L"CreateLoopbackAdapter passed when adapter presents. [2/2]");
}

int wmain(int argc, wchar_t *argv[])
{
	UNREFERENCED_PARAMETER(argc);
	UNREFERENCED_PARAMETER(argv);
	return 0;
}