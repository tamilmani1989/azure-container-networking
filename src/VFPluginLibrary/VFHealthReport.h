/*++

Copyright (c) Microsoft Corporation

Module Name:

VFHealthReport.h

Abstract:

This header contains declaration of methods to send Health report to hostagent

Author:

Tamilmani Manoharan (tamanoha) 14-November-2017

--*/

#pragma once

#include "HealthReport.h"
#include "VFPluginUtilities.h"

#define HEALTH_REPORT_PATH L"machine/plugins?comp=netagent&type=healthreport"

class VFHealthReport
{

public:

	HealthReport report;
	wchar_t healthUrl[MAX_PATH];

	VFHealthReport();

	void SendHealthStatusToHostAgent(
	);

	void ResetHealthObject(
	);

};