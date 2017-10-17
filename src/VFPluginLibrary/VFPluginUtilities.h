/*++

Copyright (c) Microsoft Corporation

Module Name:

VFPluginUtilities.h

Abstract:

Header contains declaration of methods to send status to NMAgent and Health report to hostagent
Author:

Tamilmani Manoharan (tamanoha) 10-October-2017

--*/

#include <Windows.h>
#include "XmlWriterHelper.h"
#include "HealthReport.h"

#define WIRESERVER_URL L"http://127.0.0.1:5555/PluginStatus"
#define IAAS_KEY L"SOFTWARE\\Microsoft\\NetAgent"
#define PAAS_KEY L"SOFTWARE\\Microsoft\\GuestAgent"
#define HOSTNETAGENT_ADDRESS_KEY L"HostNetAgentAddress"
#define DEFAULT_HOSTNETAGENT_ADDRESS L"169.254.169.254"
#define HOSTAGENT_PATH L"machine/plugins?comp=nmagent&type=PortContainerIOVInformationV1"
#define HEALTH_REPORT_PATH L"machine/plugins?comp=netagent&type=healthreport"

#define INSTALL_KEY L"VF_INSTALL"
#define INSTALL_VALUE L"True"

class VFPluginUtilities
{
public:
    static void GetHostNetAgentURL(    
        wchar_t *url, 
        size_t urlSize,
        wchar_t *path
    );
    
    static HRESULT RetrieveStatusInfoAsXMLStr(
        BOOL isInstalled, 
        wchar_t *installedVersion, 
        wchar_t *osVersion,
        int  numNonFunctioningDevices,
        wchar_t **statusInfo
    );

    static void SendStatusToWireServer(
        wchar_t *url,
        wchar_t *stateInfo
    );
 
    static void VFPluginUtilities::SendHealthStatusToHostAgent(
        wchar_t *url,
        HealthReport& report
    );

    static void VFPluginUtilities::ResetHealthObject(
        HealthReport& report
    );

    static void SetupLogging(wchar_t *logFolder, wchar_t *logPrefix);

    static void SaveInstalledState();
};

