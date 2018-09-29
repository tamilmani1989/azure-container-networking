/*++

Copyright (c) Microsoft Corporation

Module Name:

RestClient.h

Abstract:

This file contains the necessary headers to be used by restclient

Author:

Tamilmani Manoharan(tamanoha) 03-Aug-2017

--*/

#pragma once
#include <cpprest/http_listener.h>
#include <cpprest/http_client.h>
#include <cpprest/uri.h>
#include <cpprest/asyncrt_utils.h>
#include <cpprest/json.h>
#include "../../Logging/Logging.h"

using namespace std;
using namespace web;
using namespace http;
using namespace web::http::experimental::listener;
