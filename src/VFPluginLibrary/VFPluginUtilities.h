#include <Windows.h>
#include "XmlWriterHelper.h"

#define WIRESERVER_URL L"http://127.0.0.1:5555/PluginStatus"
#define IAAS_KEY L"SOFTWARE\\Microsoft\\NetAgent"
#define PAAS_KEY L"SOFTWARE\\Microsoft\\GuestAgent"
#define HOSTNETAGENT_ADDRESS_KEY L"HostNetAgentAddress"
#define DEFAULT_HOSTNETAGENT_ADDRESS L"169.254.169.254"
#define HOSTAGENT_PATH L"machine/plugins?comp=nmagent&type=PortContainerIOVInformationV1"
#define INSTALL_KEY L"VF_INSTALL"
#define INSTALL_VALUE L"True"

class VFPluginUtilities
{
public:
    static void GetHostNetAgentURL(    
        wchar_t *url, 
        size_t urlSize
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
 
    static void SetupLogging(wchar_t *logFolder, wchar_t *logPrefix);

    static void SaveInstalledState();
};