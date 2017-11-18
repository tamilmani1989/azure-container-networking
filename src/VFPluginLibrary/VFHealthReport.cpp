/*++

Copyright (c) Microsoft Corporation

Module Name:

VFHealthReport.cpp

Abstract:

This file contains implementation of methods reporting helath status to hostnetagent.

Author:

Tamilmani Manoharan (tamanoha) 14-November-2017

--*/

#include "VFHealthReport.h"
#include <cpprest/json.h>
#include <cpprest/http_client.h>
#include "..\Logging\Logging.h"

using namespace std;
using namespace web;
using namespace http;
using namespace web::http::client;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

VFHealthReport::VFHealthReport()
/**
Routine description:

The constructor initializes hostnetagent url.

Arguments:

Return Value:

**/
{
	VFPluginUtilities::GetHostNetAgentURL(healthUrl, sizeof(healthUrl), HEALTH_REPORT_PATH);
}

void VFHealthReport::ResetHealthObject(
)
/**
Routine description:

This function is used to reset health report object to default values.

Arguments:
report - health report object.

Return Value:

**/
{
	report.DebugMessage.assign(L"");
	report.DriverInstallPreSetupSucceeded = false;
	report.DriverInstallPreSetupFailed = false;
	report.DriverInstallPreSetupNotAttempted = false;
	report.DriverInstallNotAttempted = false;
	report.DriverInstallAttempted = false;
	report.DriverInstallSucceeded = false;
	report.DriverInstallFailed = false;
	report.TotalVFDevicesCount = 0;
	report.NonFunctionalVFDevicesCount = 0;
	report.ErrorCode = 0;
}

void VFHealthReport::SendHealthStatusToHostAgent(
)
/**
Routine description:

This function sends health report to host netagent.

Arguments:
url - url of host netagent.
report - health report info.

Return Value:

**/
{
	http_client client(healthUrl);
	http_request requester;
	json::value obj = NULL;

	requester.set_method(methods::POST);
	requester.headers().set_content_type(U("application/json"));

	obj = report.TranslateToJsonValue();

	requester.set_body(obj);

	pplx::task<void> requestTask = client.request(requester).then([](http_response response)
	{
		if (response.status_code() == status_codes::OK)
		{
			LOGINFOW(L"Health Report status sent success");
		}
		else
		{
			LOGERRW(L"Error::Health Report Server HTTP post status code %d", response.status_code());
		}
	});

	try
	{
		requestTask.wait();
	}
	catch (const std::exception &e)
	{
		LOGINFO("Error::Health Report Server exception:%s\n", e.what());
	}
}
