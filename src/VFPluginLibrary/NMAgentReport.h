/*++

Copyright (c) Microsoft Corporation

Module Name:

VFHealthReport.h

Abstract:

This header contains declaration of methods to send status to nmagent

Author:

Tamilmani Manoharan (tamanoha) 14-November-2017

--*/
#pragma once

#include "VFPluginUtilities.h"
#include <SetupAPI.h>


#define HOSTAGENT_PATH L"machine/plugins?comp=nmagent&type=PortContainerIOVInformationV1"

class NMAgentReport
{

public:

	BOOL IsInstalled;
	wchar_t OSFamily[MAX_PATH] = L"Windows";
	wchar_t DriverVersion[MAX_INF_STRING_LENGTH];
	wchar_t nmAgentUrl[MAX_PATH];
	int NumNonFunctioningDevices;

	NMAgentReport();

	int SendStatusToNMAgent(
		wchar_t *statusInfo
	);

	HRESULT RetrieveStatusInfoAsXMLStr(
		BSTR *statusInfo
	);

};