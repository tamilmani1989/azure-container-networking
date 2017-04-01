/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    NetworkAgentMock.h

Abstract:

        Mock implementation of NmAgent WCF Contracts


Author:

    T.Venketesan (thvenket) 01-May-2011

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

    TEST_METHOD(InitializeNetwork);

};
