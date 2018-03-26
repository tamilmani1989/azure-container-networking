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
#include<string>

#define HOSTAGENT_PATH L"machine/plugins?comp=nmagent&type=PortContainerIOVInformationV1"

using namespace std;

class NMAgentReport
{

public:

    BOOL IsInstalled;
    wstring OSFamily;
    wstring OSVersion;
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