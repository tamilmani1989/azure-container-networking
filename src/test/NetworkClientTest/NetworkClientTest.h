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

//
// ------------------------------------------------------------------- Classes
//

class NetworkClientTest : public WEX::TestClass<NetworkClientTest>
{

public:

    NetworkClientTest();

	virtual
    ~NetworkClientTest();

    TEST_CLASS(NetworkClientTest);

    TEST_CLASS_SETUP(TestSetup);

    TEST_CLASS_CLEANUP(TestCleanup);

    TEST_METHOD(TestInitializeNetwork);
	TEST_METHOD(TestReserveIpAddress);
	TEST_METHOD(TestGetReservedIpAddress);
	TEST_METHOD(TestReleaseIpReservation);

};
