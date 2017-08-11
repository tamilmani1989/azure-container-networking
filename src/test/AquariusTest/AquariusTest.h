/*++

Copyright (c) Microsoft Corporation

Module Name:

AquariusTest.h

Abstract:

This header Aquarius Test class and the methods to test Aquarius functionalities

Author:

Tamilmani Manoharan (tamanoha) 08-Aug-2017

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
class AquariusTest : public WEX::TestClass<AquariusTest>
{

public:

	AquariusTest();

	virtual
    ~AquariusTest();

    TEST_CLASS(AquariusTest);

    TEST_CLASS_SETUP(TestSetup);

    TEST_CLASS_CLEANUP(TestCleanup);

    TEST_METHOD(TestInstallUninstall);
	TEST_METHOD(TestInitializeService);
	TEST_METHOD(TestGetPluginMetadata);
	TEST_METHOD(TestLoadPlugins);
};
