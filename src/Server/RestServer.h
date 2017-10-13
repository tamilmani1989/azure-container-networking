/*++

Copyright (c) Microsoft Corporation

Module Name:

RestServer.h

Abstract:

This cotains the declaration of methods to be used by Aqua Rest Server.

Author:

Sushant Sharma (susharm) 05-May-2017

--*/

#pragma once
#include <cpprest/http_listener.h>
#include <cpprest/uri.h>
#include <cpprest/asyncrt_utils.h>
#include <cpprest/json.h>
#include <cpprest/filestream.h>
#include <cpprest/containerstream.h>
#include <cpprest/producerconsumerstream.h>
#include "../HNSClient/marshal/inc/MarshalBase.h"
#include "../../Logging/Logging.h"
#include "../../NetworkClientLibrary/Common/nclcore.h"


#ifdef _WIN32
#include <thread>
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#endif

using namespace std;
using namespace web;
using namespace http;
using namespace experimental::listener;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

typedef void(* PostHandler)(http_request request);
typedef void(*GetHandler)(http_request request);

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace Server {

	static const int SleepIntervalInMiliSecs = 5000;

	class RestServer
	{	
		bool initialized = false;

		wstring protocol;
		wstring address;
		wstring port;
		wstring path;
		PostHandler postHandler = NULL;
		GetHandler getHandler = NULL;

	public:
		RestServer () {}
		RestServer(
			const wstring proto,
			const wstring serverAddress,
			const wstring serverPort,
			const wstring urlPath,
			PostHandler ipostHandler,
			GetHandler igetHandler)
		{
			UNREFERENCED_PARAMETER(ipostHandler);
			UNREFERENCED_PARAMETER(igetHandler);
			if (proto.empty() || serverAddress.empty() || serverPort.empty())
			{
				LOGERRW(L"Invalid input, one of protocol, serveraddress, or port is empty. \
					protocol=%ls server=%ls port=%ls. Server will not be started.", proto.c_str(), serverAddress.c_str(), serverPort.c_str());
				initialized = false;
			}
			else
			{				
				LOGINFOW(L"Creating Rest server: proto: %ws, server: %ws, port: %ws, path: %ws",
					proto.c_str(), serverAddress.c_str(), serverPort.c_str(), urlPath.c_str());
				protocol.assign(proto.begin(), proto.end());
				address.assign(serverAddress.begin(), serverAddress.end());
				port.assign(serverPort.begin(), serverPort.end());
				LOGINFO("restserver");
				
				if(ipostHandler==NULL)
				{
					LOGWARN("NULL passed as post handler");
				}

				postHandler = ipostHandler;

				if (igetHandler == NULL)
				{
					LOGWARN("NULL passed as get handler");
				}

				getHandler = igetHandler;

				path.assign(urlPath.begin(), urlPath.end());
				initialized = true;
			}
		}

		static void ListenNoReturn(RestServer *);
		DllExport DWORD Listen();
		DllExport DWORD SetMethodToProcessPOST(PostHandler postHandler);
		DllExport DWORD SetMethodToProcessGET(GetHandler getHandler);
		DllExport void dummy();
	};
} } } } }