/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

	RequestHandlers.h

Abstract:

	Header to Rest handler functions and helpers.

Environment:

	User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//

#include <stdio.h>
#include <stdlib.h>
#include <cpprest/http_listener.h>
#include <cpprest/uri.h>
#include <cpprest/asyncrt_utils.h>
#include <cpprest/json.h>
#include <cpprest/filestream.h>
#include <cpprest/containerstream.h>
#include <cpprest/producerconsumerstream.h>
#include "cpprest/filestream.h"
#include "PackagesInfo.h"
#include "GenevaLogger.h"
#include "HealthReport.h"

using namespace web;
using namespace http;

class RequestHandlers
{

private:

	static
	void
	GetPackageVersion(
		__inout http_request Request);

	static
	void
	GetPackageUris(
		__inout http_request Request);

	static
	HRESULT
	ExtractContainerIdAndRequestType(
		web::uri Uri,
		wstring& ContainerId,
		wstring& RequestType);

public:

	static
	void
	ProcessPost(
		__inout http_request Request);

	static
	void
	ProcessGet(
		__inout http_request Request);

};