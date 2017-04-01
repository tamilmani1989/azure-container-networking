#ifndef UNICODE
#define UNICODE
#endif


#include <windows.h>

#include <stdio.h>
#include <string>
#include "stdafx.h"

#include "NetworkClientTest.h"
#include "aquanetworkclient.h"
#define RDPRINT_VALUE(ElementName, Value) if (Value != NULL) {                              \
                                                                                            \
                                              wprintf(L"%ws - %ws\n", ElementName, Value);  \
                                          }
#define AGENT_TRACE_FLAGS (RD_TRACE_ALL & ~RD_TRACE_PAINFUL)

#include "TraceLogging.h"

TRACELOGGING_DEFINE_PROVIDER(
    g_hMyComponentProvider,
    "SimpleTraceLoggingProvider",
    (0x3970f9cf, 0x2c0c, 0x4f11, 0xb1, 0xcc, 0xe3, 0xa1, 0xe9, 0x95, 0x88, 0x33));
	
using namespace std;
//using namespace Mock10;
using namespace WEX::TestExecution;



HRESULT
InternalInitialize()
{
	/*ClusterConfig cfg;
	size_t netLen = strlen("azurenet") + sizeof(char);
	cfg.networkName = (char *) malloc(netLen);
	printf("%llu", netLen);
	strncpy_s(cfg.networkName, netLen, "azurenet", 9);
	cfg.networkName[8] = '\0';	
	cfg.net = UnderLay;
	cfg.env = Azure;
	printf("nw main name:%s\n",cfg.networkName);
 
	//InitializeNetwork(&cfg);*/
	return S_OK;
}

void
NetworkClientTest::InitializeNetwork()
/*++

Routine Description:

    This route is to test creation of network adapter for docker

Arguments:

    None.

Return Value:

    None.

--*/
{
	HRESULT hr = S_OK;

    //
    // Only produce the logs on failure for Verify functions.
    //

	//hr = InternalInitialize();

    VERIFY_ARE_EQUAL(S_OK, hr, L"NetworkClientTest Works!");
}




bool NetworkClientTest::TestSetup()
/*++

Routine Description:

    TAEF Setup. This API is called before invoking test APIs

Arguments:

    None

Return Value:

    None

--*/
{
    //
    // Initialize tracing.
    //
   char sampleValue[] = "Sample value";

/*    RD_TRACE_INITIALIZE(RD_TRACE_OUTPUT_FILE | RD_TRACE_OUTPUT_CONSOLE,
                        AGENT_TRACE_FLAGS,
                        NULL,
                        NULL,
                        0,
                        NULL);
						*/
    TraceLoggingRegister(g_hMyComponentProvider);
    // Log an event
    TraceLoggingWrite(g_hMyComponentProvider, // handle to my provider
        "HelloWorldTestEvent",              // Event Name that should uniquely identify your event.
        TraceLoggingValue(sampleValue, "TestMessage")); // Field for your event in the form of (value, field name).

/*	TraceLoggingWrite(g_hMyComponentProvider, // handle to my provider
		"HelloWorldTestEvent",              // Event Name that should uniquely identify your event.
		TraceLoggingValue("Visible msg", "TestMessage")); 

	TraceLoggingWrite(
		g_hMyComponentProvider,
		"Strings and Chars",
		TraceLoggingString("String1", "String (char)"),
		TraceLoggingWideString(L"LString2", "String (wide char)"),
		TraceLoggingChar('A', "Single char")
	);
	*/

    return true;
}

bool
NetworkClientTest::TestCleanup()
/*++

Routine Description:

    TAEF Cleanup. This API is called after invoking test APIs

Arguments:

    None

Return Value:

    None

--*/
{
   // RD_TRACE_TERMINATE();
	TraceLoggingUnregister(g_hMyComponentProvider);

    return true;

}

NetworkClientTest::NetworkClientTest()
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


NetworkClientTest::~NetworkClientTest()
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

    NetworkClientTest* parserTest = new NetworkClientTest();
    parserTest->TestSetup();
    parserTest->InitializeNetwork();
    parserTest->TestCleanup();

    return 0;
}

