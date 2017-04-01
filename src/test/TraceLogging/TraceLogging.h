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
#include <windows.h>
#include <TraceLoggingProvider.h>  // The native TraceLogging API

// Forward-declare the g_hMyComponentProvider variable that you will use for tracing in this component
TRACELOGGING_DECLARE_PROVIDER( g_hMyComponentProvider );

