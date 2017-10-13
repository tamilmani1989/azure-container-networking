/*++

Copyright (c) Microsoft Corporation

Module Name:

RestServer.cpp

Abstract:

Implementation of a rest server.

Author:

Sushant Sharma (susharm) 18-July-2017

--*/

#include "RestServer.h"
#ifdef _WIN32
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#endif
#include "../Shared/Utilities.h"
#define SUCCESSVAL 0

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace Server {

void RestServer::ListenNoReturn(RestServer *restServer)
/*++

Routine Description:
	It starts a restful listner on the address and port and waits.

Arguments:

Return Value:
	N/A

--*/
{
	LOGINFO("Entering ListenNoReturn.");
	wstring server;
	
	server.append(restServer->protocol); // append does not fail
	server.append(L"://");
	server.append(restServer->address);
	server.append(L":");
	server.append(restServer->port);
	server.append(L"/");
	server.append(restServer->path);

	LOGINFOW(L"Going to create a uri builder using %ls", server.c_str());

	uri_builder uriBuilder(server);

	http_listener listner(uriBuilder.to_uri());
	
	if(restServer->getHandler)
	{
		LOGINFO("Going to set the listner for GET.");
		listner.support(methods::GET, restServer->getHandler);
	}

	if(restServer->postHandler)
	{
		LOGINFO("Going to set the listner for POST.");
		listner.support(methods::POST, restServer->postHandler);
	}
	
	LOGINFO("Going to open the listner.");
	pplx::task_status ret = listner.open().then([&listner]() {LOGINFO("BEGIN LISTEN")}).wait();

	if (ret != pplx::completed)
	{
		LOGERR("Unable to open the listener. Ret = %d", ret);
	}
	else
	{
		LOGINFO("Successfully listening on port specified")
		LOGINFOW(L"Successfully listening @ %ls", server.c_str());
	}
	

	while (true) { Sleep(SleepIntervalInMiliSecs); }
	LOGINFO("Exiting ListenNoReturn.");
}

DllExport DWORD RestServer::Listen()
/*++

Routine Description:
	It starts a restful listner on the address and port.

Arguments:

Return Value:

0 if service successful, error code otherwise.

--*/
{
	LOGINFO("Entering Listen.");	
	DWORD retval = 0;
	std::thread listnerThread(ListenNoReturn, this);
	listnerThread.detach();
	LOGINFO("Exiting Listen.");
	return retval;
}
DllExport DWORD RestServer::SetMethodToProcessPOST(
	PostHandler postHandler)
	/*++

	Routine Description:
	Sets the method to handle post requests by the server.

	Arguments:

	Return Value:

	0 if service successful, error code otherwise.

	--*/
{
	LOGINFO("Entering SetMethodToProcessPOST.");
	if (postHandler == nullptr)
	{
		LOGERR("postHandler should not be null");
		return INVALID_PARAMETER;
	}

	this->postHandler = postHandler;
	LOGINFO("Exiting SetMethodToProcessPOST.");
	return SUCCESSVAL;
}

DllExport DWORD RestServer::SetMethodToProcessGET(
	GetHandler getHandler)
	/*++

	Routine Description:
	Sets the method to handle GET requests by the server.

	Arguments:

	Return Value:

	0 if service successful, error code otherwise.

	--*/
{
	LOGINFO("Entering SetMethodToProcessGET.");
	if (getHandler == nullptr)
	{
		LOGERR("getHandler should not be null");
		return INVALID_PARAMETER;
	}

	this->getHandler = getHandler;
	LOGINFO("Exiting SetMethodToProcessGET.");
	return SUCCESSVAL;
}

} } } } }