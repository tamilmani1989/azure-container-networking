/*++

Copyright (c) Microsoft Corporation

Module Name:

RestClient.cpp

Abstract:

Implementation of a rest client to get plugin status from aquarius.

Author:

Tamilmani Manoharan (tamanoha) 03-August-2017

--*/

#include "RestClient.h"
#ifdef _WIN32
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#endif
#include "../Shared/Utilities.h"
#include "../Shared/PluginMetadata.h"
#include "../Aquarius/AquariusService.h"

using namespace web::http;
using namespace web::http::client;

int exitStatus=1;

int processInnerJson(json::value jsonValue)
/*++

Routine Description:
	This function process the inner json which contains plugin running state and its exit status

Arguments:
	jsonValue - State and exit status of plugin in json format

Return Value:
	returns 0 on successful case else returns 1

--*/
{
	int status = 0;
	int state = 0;

	for(auto iterInner = jsonValue.as_object().cbegin(); iterInner != jsonValue.as_object().cend(); ++iterInner)
	{
		const utility::string_t &propertyName = iterInner->first;
		const json::value &propertyValue = iterInner->second;
		wprintf(L"inner json %s %d\n", propertyName.c_str(), propertyValue.as_integer());
		if (wcscmp(propertyName.c_str(), L"exitStatus") == 0 && propertyValue.as_integer() != 0)
			status=propertyValue.as_integer();
		else if (wcscmp(propertyName.c_str(), L"state") == 0 && propertyValue.as_integer() != 0)
			state=propertyValue.as_integer();

	}

	if ((state == STOPPED && status != 0) || state == NOT_STARTED)
	{
		return 1;		
	}

	return 0;
}

int getPluginStatus()
/*++

Routine Description:
	This function initiates a rest client to connect to rest server and retrieve the status of all plugins

Arguments:

Return Value:
	returns 0 on successful case else returns 1

--*/
{
	wchar_t url[128];

	swprintf_s(url, sizeof(url), L"%s://%s:%s/%s", HTTP_PROTO, LOCALHOST, RESTSERVER_PORT, PLUGIN_STATUS);
	http_client client(url);

	// Make the request and asynchronously process the response. 
	client.request(methods::GET).then([](http_response response)
	{
		std::wostringstream ss;
		
		ss << L"Server returned returned status code " << response.status_code() << L'.' << std::endl;
		std::wcout << ss.str();

		pplx::task_status status = pplx::not_complete;
		pplx::task<json::value> js = response.extract_json();
		
		try
		{
			status = js.wait();
		}
		catch (const std::exception &ex)
		{
			LOGERR("Waiting for task (to extract json from POST) encountered exception. %s", ex.what());
		}
		
		if (status != pplx::completed)
		{
			LOGERR("Task to extract json from GET did not finish. status: %d", status);
		}
		else
		{
			int ret = 0;
			json::value v = js.get();

			if (!v.is_null())
			{			
				for (auto iter = v.as_object().cbegin(); iter != v.as_object().cend(); ++iter)
				{
					const utility::string_t &ca = iter->first;
					const json::value &innerJSON = iter->second;
					LOGINFOW(L"Received the json on server as %s\n", ca.c_str());
					ret = ret | processInnerJson(innerJSON);				
				}
			}
			else
			{
				LOGINFO("Server received NULL/Invalid json. This API require json object as input.");
				ret=1;
			}

			LOGINFO("Role block Status %d", ret );
			exitStatus=ret;
		}

	}).wait();

	return exitStatus;
}

int wmain(
	int argc,
	wchar_t *argv[]
)
{
	UNREFERENCED_PARAMETER(argc);
	UNREFERENCED_PARAMETER(argv);	
	int status = getPluginStatus();
	LOGINFO("Restclient exit status %d\n", status);
	return status;
}
