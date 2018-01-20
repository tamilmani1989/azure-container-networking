/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

	NetworkContainerTest.h

Abstract:

	Test NetworkContainer Functionalities

Author:

	Yongli Chen 1-18-2017

--*/

#pragma once

//
// ------------------------------------------------------------------- Includes
//

#include "WexTestClass.h"
#include <ws2tcpip.h>
#include <windows.h>

#define LOG_FILE_PREFIX L"NetworkContainerTest"
#define LOG_PATH L"C:\\Logs\\NetworkContainerTest"

class NetworkContainerTest : public WEX::TestClass<NetworkContainerTest>
{

public:

	NetworkContainerTest();
	virtual ~NetworkContainerTest();

	TEST_CLASS(NetworkContainerTest);
	TEST_CLASS_SETUP(TestSetup);
	TEST_CLASS_CLEANUP(TestCleanup);

	TEST_METHOD(TestFindAdapterByDisplayName);
	TEST_METHOD(TestCreateLoopbackAdapter);
};