/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

	NetworkClientTest.h
Abstract:

        Test NetworkClient Functionalities


Author:

    Tamilmani 04-01-2017

--*/

#pragma once

//
// ------------------------------------------------------------------- Includes
//

#include "WexTestClass.h"
#include <ws2tcpip.h>
#include <windows.h>


//
// ------------------------------------------------------------------- Classes
//
class ServiceTunnelExtensionTest : public WEX::TestClass<ServiceTunnelExtensionTest>
{

public:

	ServiceTunnelExtensionTest();

	virtual
    ~ServiceTunnelExtensionTest();

    TEST_CLASS(ServiceTunnelExtensionTest);

    TEST_CLASS_SETUP(TestSetup);

    TEST_CLASS_CLEANUP(TestCleanup);

    TEST_METHOD(TestCreateNetworkAdapter);
	TEST_METHOD(TestReadFromRcf);
	TEST_METHOD(TestSetIpv6Address);
	TEST_METHOD(TestGetInterfaceByIpAddress);
	TEST_METHOD(TestEnableWeakHostRouting);
	TEST_METHOD(TestAddDeleteNeighbor);
	TEST_METHOD(TestAddDeleteIPRoute);
	TEST_METHOD(TestDiableEnableAdapter);
	TEST_METHOD(TestReadFromRcfWithSTAddr);
	TEST_METHOD(TestIdemPotent);
};
#define LOG_FILE_PREFIX L"ServiceTunnelExtensionTest"