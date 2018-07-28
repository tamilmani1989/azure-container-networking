#ifndef UNICODE
#define UNICODE
#endif

#define _WINSOCK_DEPRECATED_NO_WARNINGS

#include <windows.h>

#include <stdio.h>
#include <string>


#include "ServiceTunnelExtensionTest.h"
#undef __WFILE__
#undef __WFUNCTION__
#include "ServiceTunnelLibrary.h"
	

using namespace std;
using namespace WEX::TestExecution;

int IPV6IfIndex=0;
int IPV4IfIndex=0;
int testIfIndex = 0;
std::wstring IPV6Address;
std::wstring IPV4Address;
LoopbackAdapterInstaller installer;
void
ServiceTunnelExtensionTest::TestCreateNetworkAdapter()
{
	HRESULT hr;
	int getIfIndex = 0;

	hr = installer.GetOrCreateLoopbackAdapter(SERVICE_TUNNEL_INTERFACE_NAME, IPV6IfIndex);
	VERIFY_ARE_EQUAL((HRESULT)S_OK, hr, L"CreateAdapterPassed");
	VERIFY_IS_GREATER_THAN(IPV6IfIndex, 0, L"IfIndex valid value");

	hr = installer.GetOrCreateLoopbackAdapter(SERVICE_TUNNEL_INTERFACE_NAME, getIfIndex);
	VERIFY_ARE_EQUAL((HRESULT)S_OK, hr, L"getAdapterPassed");
	VERIFY_ARE_EQUAL(IPV6IfIndex, getIfIndex, L"Ifindex value is correct");
}

// Test reading from RCF.
void
ServiceTunnelExtensionTest::TestReadFromRcf()
{
	HRESULT hr = S_OK;
	RcfEnumerator rcfEnumerator;
	XmlParserHelper xmlParserHelper;
	ConfigReader configReader;
	ServiceTunnelConfigurator serviceTunnelConfigurator;
	DWORD error;
	LPWSTR rcfFile = NULL;
	LPWSTR rcfDir = NULL;
	LPWSTR rcfLocation = NULL;
	WCHAR curDir[128];
	wstring ipAddr;

	std::wstring instanceName;
	std::set<std::wstring> serviceTunnelIpv6Addresses;

	hr = WcsDupHr(L".\\test\\ServiceTunnelExtensionTest\\rcf", &rcfDir);
	VERIFY_ARE_EQUAL(hr, S_OK, L"WcsDupHr passed");
	
	GetCurrentDirectory(sizeof(curDir), curDir);

	error = AppendStrings(&rcfLocation,
		2,
		curDir,
		rcfDir);
	VERIFY_ARE_EQUAL((DWORD)0, error, L"AppendStrings passed");

	hr = rcfEnumerator.FindLatestRcf(&rcfFile, rcfLocation);
	VERIFY_ARE_EQUAL(hr, S_OK, L"FindLatestRcf passed");

	hr = xmlParserHelper.Initialize(rcfFile);
	VERIFY_ARE_EQUAL(hr, S_OK, L"Xml Initialize passed");

	hr = configReader.GetInstanceId(&xmlParserHelper, instanceName);
	VERIFY_ARE_EQUAL(hr, S_OK, L"GetInstanceId passed");
	VERIFY_ARE_EQUAL((std::wstring)instanceName, L"ContainerTest", L"Instance name is correct");

	hr = configReader.GetInstanceAddress(&xmlParserHelper, instanceName, ipAddr);
	VERIFY_ARE_EQUAL(hr, S_OK, L"GetInstanceAddress passed");
	VERIFY_ARE_EQUAL((std::wstring)ipAddr, L"10.10.20.1", L"Instance Address is correct");


	hr = configReader.GetServiceTunnelIpv6Addresses(&xmlParserHelper, instanceName, serviceTunnelIpv6Addresses);
	VERIFY_ARE_NOT_EQUAL(hr, S_OK, L"GetServiceTunnelIpv6Addresses passed");
	VERIFY_ARE_EQUAL((size_t)0, serviceTunnelIpv6Addresses.size(), L"");

	IPV6Address = serviceTunnelConfigurator.GetSTIpv6AddrFromInstanceIpv4Addr(ipAddr);
	VERIFY_ARE_EQUAL((std::wstring)IPV6Address, L"2603:10e1:100:2::a0a:1401", L"Derived IPV6 Address is correct");

}

// Test deleting and adding IPV6 address to loopback adapter.
void
ServiceTunnelExtensionTest::TestSetIpv6Address()
{
	ServiceTunnelConfigurator svcTunnelConfigurator;
	HRESULT hr = S_OK;
	hr = svcTunnelConfigurator.NetIfSetIpv6Address(IPV6IfIndex, IPV6Address.c_str());
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfSetIpv6Address passed");
}

// Test Enable weak host routing for both test and loopback adapter.
void
ServiceTunnelExtensionTest::TestEnableWeakHostRouting()
{
	ServiceTunnelConfigurator svcTunnelConfigurator;
	HRESULT hr = S_OK;

	hr = svcTunnelConfigurator.NetIfEnableWeakHostRouting(IPV6IfIndex);
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfEnableWeakHostRouting passed for ipv6 interface");

	hr = svcTunnelConfigurator.NetIfEnableWeakHostRouting(IPV4IfIndex);
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfEnableWeakHostRouting passed for ipv4 interface");
}

// Test getting interface index from specified ipv4 address. The given ipv4 address if for test adapter.
void
ServiceTunnelExtensionTest::TestGetInterfaceByIpAddress()
{
	ServiceTunnelConfigurator svcTunnelConfigurator;
	HRESULT hr = S_OK;
	size_t retSize;
	char ipv4Str[32];

	wcstombs_s(&retSize, ipv4Str, sizeof(ipv4Str), IPV4Address.c_str(), wcslen(IPV4Address.c_str())+1);
	ULONG ipv4Addr = ntohl(inet_addr(ipv4Str));

	hr = svcTunnelConfigurator.GetInterfaceByIpAddress(ipv4Addr, testIfIndex);
	VERIFY_ARE_EQUAL(hr, S_OK, L"");
	VERIFY_ARE_EQUAL(IPV4IfIndex, testIfIndex, L"GetInterfaceByIpAddress passed");

}

// Test deleting and adding neighbor entry on test adapter interface.
void
ServiceTunnelExtensionTest::TestAddDeleteNeighbor()
{
	ServiceTunnelConfigurator svcTunnelConfigurator;
	HRESULT hr = S_OK;

	hr = svcTunnelConfigurator.NetIfAddNeighbor(IPV4IfIndex, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_MAC_ADDRESS);
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfAddNeighbor passed");

	hr = svcTunnelConfigurator.NetIfDeleteNeighbor(IPV4IfIndex, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_MAC_ADDRESS);
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfDeleteNeighbor passed");
}

// Test deleting and adding route entry on test adapter interface.
void
ServiceTunnelExtensionTest::TestAddDeleteIPRoute()
{
	ServiceTunnelConfigurator svcTunnelConfigurator;
	HRESULT hr = S_OK;
	WCHAR stDstPrefix[MAX_PATH];
	WCHAR prefixLen[INET6_ADDRSTRLEN];

	swprintf_s(prefixLen, sizeof(prefixLen), L"%d", SERVICE_TUNNEL_SOURCE_PREFIX_LEN);
	wcscpy_s(stDstPrefix, SERVICE_TUNNEL_SOURCE_PREFIX);
	wcscat_s(stDstPrefix, L"/");
	wcscat_s(stDstPrefix, prefixLen);

	hr = svcTunnelConfigurator.NetIfAddIPv6Route(IPV4IfIndex, stDstPrefix, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS);
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfAddIPv6Route passed");

	hr = svcTunnelConfigurator.NetIfDeleteIPv6Route(IPV4IfIndex, stDstPrefix, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS);
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfDeleteIPv6Route passed");

}

// Check if neighbor exist after disabling and enabling adapter.
void checkNeighborsExist()
{
	ServiceTunnelConfigurator svcTunnelConfigurator;
	ULONG ret = svcTunnelConfigurator.CheckIfNeighborEntryExists(IPV4IfIndex, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS);
	VERIFY_ARE_EQUAL((ULONG)0, ret, L"Neighbor exists");
}

// Check if route exist after disabling and enabling adapter.
void checkRouteExist()
{
	ServiceTunnelConfigurator svcTunnelConfigurator;
	ULONG ret = svcTunnelConfigurator.CheckIfRouteExists(IPV4IfIndex, SERVICE_TUNNEL_SOURCE_PREFIX, SERVICE_TUNNEL_SOURCE_PREFIX_LEN, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS);
	VERIFY_ARE_EQUAL((ULONG)0, ret, L"Route exists");
}

// Check if ipv6 address exist after disabling and enabling adapter.
void checkIPV6Exist()
{
	ServiceTunnelConfigurator svcTunnelConfigurator;
	SOCKADDR_IN6 sa;
	char ipv6CharArray[sizeof(sa.sin6_addr.u.Byte) + 1];

	std::set<string> ipv6Addresses = svcTunnelConfigurator.GetIPV6AddressesOfAdapter(IPV6IfIndex);
	
	_bstr_t ip(IPV6Address.c_str());
	const char* ipv6 = ip;

	inet_pton(AF_INET6, ipv6, &(sa.sin6_addr));
	memcpy(ipv6CharArray, sa.sin6_addr.u.Byte, sizeof(sa.sin6_addr.u.Byte));
	ipv6CharArray[sizeof(sa.sin6_addr.u.Byte)] = '\0';

	bool status = ipv6Addresses.find(ipv6CharArray) != ipv6Addresses.end();
	VERIFY_IS_TRUE(status, L"ipv6 exists");
}

// Test Diable and Enable adapter and check all tables.
void
ServiceTunnelExtensionTest::TestDiableEnableAdapter()
{
	DWORD error;
	PWSTR netshCommand = NULL;
	DWORD exitCode = 0;
	HRESULT hr = S_OK;
	MIB_IF_ROW2 row = { 0 };
	
	row.InterfaceIndex = IPV4IfIndex;
	GetIfEntry2(&row);

	error = FormatString(&netshCommand,
		0,
		L"netsh interface set interface \"%s\" admin=disable",
		row.Alias);
	VERIFY_ARE_EQUAL((DWORD)0, error, L"");

	hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
	VERIFY_ARE_EQUAL(hr, S_OK, L"");
	VERIFY_ARE_EQUAL((DWORD)0, exitCode, L"Disabling test adapter passed");

	STR_FREE(netshCommand);

	error = FormatString(&netshCommand,
		0,
		L"netsh interface set interface \"%s\" admin=enable",
		row.Alias);
	VERIFY_ARE_EQUAL((DWORD)0, error, L"");

	hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
	VERIFY_ARE_EQUAL(hr, S_OK, L"");
	VERIFY_ARE_EQUAL((DWORD)0, exitCode, L"Enabling test adapter passed");

	STR_FREE(netshCommand);

	row.InterfaceIndex = IPV6IfIndex;
	GetIfEntry2(&row);

	error = FormatString(&netshCommand,
		0,
		L"netsh interface set interface \"%s\" admin=disable",
		row.Alias);
	VERIFY_ARE_EQUAL((DWORD)0, error, L"");

	hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
	VERIFY_ARE_EQUAL(hr, S_OK, L"");
	VERIFY_ARE_EQUAL((DWORD)0, exitCode, L"Disabling loopback adapter passed");

	STR_FREE(netshCommand);

	error = FormatString(&netshCommand,
		0,
		L"netsh interface set interface \"%s\" admin=enable",
		row.Alias);
	VERIFY_ARE_EQUAL((DWORD)0, error, L"");

	hr = InvokeSystemCommand(netshCommand, &exitCode, INFINITE, FALSE);
	VERIFY_ARE_EQUAL(hr, S_OK, L"");
	VERIFY_ARE_EQUAL((DWORD)0, exitCode, L"Enabling loopback adapter passed");

	STR_FREE(netshCommand);

	checkIPV6Exist();
}

void
ServiceTunnelExtensionTest::TestReadFromRcfWithSTAddr()
{
	HRESULT hr = S_OK;
	RcfEnumerator rcfEnumerator;
	XmlParserHelper xmlParserHelper;
	ConfigReader configReader;
	DWORD error;
	LPWSTR rcfFile = NULL;
	LPWSTR rcfDir = NULL;
	LPWSTR rcfLocation = NULL;
	WCHAR curDir[128];
	WCHAR stIpv6Addr[32];
	wstring ipAddr;
	std::wstring instanceName;
	std::set<std::wstring> serviceTunnelIpv6Addresses;
	std::set<std::wstring>::iterator serviceTunnelIpv6AddressesIter;

	hr = WcsDupHr(L".\\test\\ServiceTunnelExtensionTest\\rcf2", &rcfDir);
	VERIFY_ARE_EQUAL(hr, S_OK, L"WcsDupHr passed");

	GetCurrentDirectory(sizeof(curDir), curDir);

	error = AppendStrings(&rcfLocation,
		2,
		curDir,
		rcfDir);
	VERIFY_ARE_EQUAL((DWORD)0, error, L"AppendStrings passed");

	hr = rcfEnumerator.FindLatestRcf(&rcfFile, rcfLocation);
	VERIFY_ARE_EQUAL(hr, S_OK, L"FindLatestRcf passed");

	hr = xmlParserHelper.Initialize(rcfFile);
	VERIFY_ARE_EQUAL(hr, S_OK, L"Xml Initialize passed");

	hr = configReader.GetInstanceId(&xmlParserHelper, instanceName);
	VERIFY_ARE_EQUAL(hr, S_OK, L"GetInstanceId passed");
	VERIFY_ARE_EQUAL((std::wstring)instanceName, L"ContainerTest", L"Instance name is correct");

	hr = configReader.GetInstanceAddress(&xmlParserHelper, instanceName, ipAddr);
	VERIFY_ARE_EQUAL(hr, S_OK, L"GetInstanceAddress passed");
	VERIFY_ARE_EQUAL((std::wstring)ipAddr, L"10.10.20.1", L"Instance Address is correct");


	hr = configReader.GetServiceTunnelIpv6Addresses(&xmlParserHelper, instanceName, serviceTunnelIpv6Addresses);
	VERIFY_ARE_EQUAL(hr, S_OK, L"GetServiceTunnelIpv6Addresses passed");
	VERIFY_IS_GREATER_THAN(serviceTunnelIpv6Addresses.size(), (size_t)0, L"");

	for (serviceTunnelIpv6AddressesIter = serviceTunnelIpv6Addresses.begin(); serviceTunnelIpv6AddressesIter != serviceTunnelIpv6Addresses.end(); ++serviceTunnelIpv6AddressesIter)
	{
		wcscpy_s(stIpv6Addr, serviceTunnelIpv6AddressesIter->c_str());
	}
	int ret = wcscmp(stIpv6Addr, L"2603:10e1:100:2::a0a:1401");
	VERIFY_ARE_EQUAL(0, ret, L"Derived IPV6 Address is correct");

}


//Test APIs are idempotent even if they called any number of times.
void
ServiceTunnelExtensionTest::TestIPRouteNeighExist()
{
	ServiceTunnelConfigurator svcTunnelConfigurator;
	HRESULT hr = S_OK;
	WCHAR stDstPrefix[MAX_PATH];
	WCHAR prefixLen[INET6_ADDRSTRLEN];

	checkIPV6Exist();

	hr = svcTunnelConfigurator.NetIfAddNeighbor(IPV4IfIndex, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_MAC_ADDRESS);
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfAddNeighbor passed");

	checkNeighborsExist();

	swprintf_s(prefixLen, sizeof(prefixLen), L"%d", SERVICE_TUNNEL_SOURCE_PREFIX_LEN);
	wcscpy_s(stDstPrefix, SERVICE_TUNNEL_SOURCE_PREFIX);
	wcscat_s(stDstPrefix, L"/");
	wcscat_s(stDstPrefix, prefixLen);

	hr = svcTunnelConfigurator.NetIfAddIPv6Route(IPV4IfIndex, stDstPrefix, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS);
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfAddIPv6Route passed");

	checkRouteExist();
}

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

void
ServiceTunnelExtensionTest::TestUninstallAdapter()
{
	ServiceTunnelConfigurator svcTunnelConfigurator;
	WCHAR stDstPrefix[MAX_PATH];
	WCHAR prefixLen[INET6_ADDRSTRLEN];

	swprintf_s(prefixLen, sizeof(prefixLen), L"%d", SERVICE_TUNNEL_SOURCE_PREFIX_LEN);
	wcscpy_s(stDstPrefix, SERVICE_TUNNEL_SOURCE_PREFIX);
	wcscat_s(stDstPrefix, L"/");
	wcscat_s(stDstPrefix, prefixLen);
	HRESULT hr = installer.RemoveLoopbackAdapter(SERVICE_TUNNEL_INTERFACE_NAME);
	VERIFY_ARE_EQUAL(hr, S_OK, L"UninstallAdapter passed");

	hr = svcTunnelConfigurator.NetIfDeleteIPv6Route(IPV4IfIndex, stDstPrefix, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS);
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfDeleteIPv6Route passed");

	hr = svcTunnelConfigurator.NetIfDeleteNeighbor(IPV4IfIndex, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_IP_ADDRESS, SERVICE_TUNNEL_ENDPOINT_FAKE_GW_MAC_ADDRESS);
	VERIFY_ARE_EQUAL(hr, S_OK, L"NetIfDeleteNeighbor passed");
}

bool ServiceTunnelExtensionTest::TestSetup()
/*++

Routine Description:

    TAEF Setup. This API is called before invoking test APIs. This API sets up test adapter which is consider as primary nic.

Arguments:

    None

Return Value:

    None

--*/
{
    //
    // Initialize tracing.
    //
	HRESULT hr;
	PIP_ADAPTER_INFO pAdapterInfo;
	DWORD dwRetVal;
	MIB_IF_ROW2 row = { 0 };

	hr = SetupLogging();
	if (FAILED(hr))
	{
		wprintf(L"Failed to setup logging due to error [%0x]", hr);
		return false;
	}
	hr = ::CoInitialize(NULL);
	VERIFY_ARE_EQUAL((HRESULT)S_OK, hr, L"CoInitialize passed");

	ULONG ulOutBufLen = sizeof(IP_ADAPTER_INFO);
	pAdapterInfo = (IP_ADAPTER_INFO *)malloc(sizeof(IP_ADAPTER_INFO));
	if (pAdapterInfo == NULL)
	{
		wprintf(L"Error allocating memory needed to call GetAdaptersinfo\n");
		return false;
	}

	// Make an initial call to GetAdaptersInfo to get
	// the necessary size into the ulOutBufLen variable
	if (GetAdaptersInfo(pAdapterInfo, &ulOutBufLen) == ERROR_BUFFER_OVERFLOW)
	{
		free(pAdapterInfo);
		pAdapterInfo = (IP_ADAPTER_INFO *)malloc(ulOutBufLen);
		if (pAdapterInfo == NULL)
		{
			wprintf(L"Error allocating memory needed to call GetAdaptersinfo\n");
			return false;
		}
	}

	if ((dwRetVal = GetAdaptersInfo(pAdapterInfo, &ulOutBufLen)) != NO_ERROR)
	{
		wprintf(L"GetAdaptersInfo failed with error %d\n", dwRetVal);
		return false;
	}

	while (pAdapterInfo)
	{
		if (pAdapterInfo->DhcpEnabled)
			break;
		pAdapterInfo = pAdapterInfo->Next;
	}

	if (!pAdapterInfo)
	{
		wprintf(L"No Adapter found with dhcp enabled\n");
		return false;
	}
	size_t retVal = 0;
	wchar_t wipv4[32];
	IPV4IfIndex = pAdapterInfo->Index;
	mbstowcs_s(&retVal, wipv4, sizeof(wipv4), pAdapterInfo->IpAddressList.IpAddress.String, sizeof(pAdapterInfo->IpAddressList.IpAddress.String));
	IPV4Address = std::wstring(wipv4);
    return true;
}

bool
ServiceTunnelExtensionTest::TestCleanup()
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

ServiceTunnelExtensionTest::ServiceTunnelExtensionTest()
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


ServiceTunnelExtensionTest::~ServiceTunnelExtensionTest()
/*++

Routine Description:

    Destructor for NetworkClientTest

Arguments:

    None.

Return Value:

    None.

--*/
{

}

int wmain(int argc, wchar_t *argv[])
{
    UNREFERENCED_PARAMETER(argc);
    UNREFERENCED_PARAMETER(argv);
    return 0;
}
