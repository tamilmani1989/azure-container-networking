/*++

Copyright (c) Microsoft Corporation

Module Name:

VFPluginUtilities.cpp

Abstract:

This file contains methods send status to NMAgent and Health report to hostagent
Author:

Tamilmani Manoharan (tamanoha) 10-October-2017

--*/

#include <cpprest/http_listener.h>
#include <cpprest/http_client.h>
#include <cpprest/uri.h>
#include <cpprest/asyncrt_utils.h>
#include <cpprest/json.h>
#include "..\Logging\Logging.h"
#include "VFPluginUtilities.h"

using namespace std;
using namespace web;
using namespace http;
using namespace web::http::client;
using namespace experimental::listener;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;


void VFPluginUtilities::GetHostNetAgentURL(
    wchar_t *url, 
    size_t urlSize,
    wchar_t *path
)
/**
Routine description:

    This function constructs wireserver agent url based on path.

Arguments:
    url - result od constructed url will be placed in this buffer.
    urlSize - size of url buffer.
    path - path of url.

Return Value:

**/
{
    HKEY returnKey;
    DWORD type = REG_SZ;
    DWORD size;
    wchar_t addr[MAX_PATH];
    bool openFlag = false;
    LONG nError;
    
    size = (DWORD)sizeof(addr);

    LOGINFOW(L"Going to open registry in Iaas %s", IAAS_KEY);
    
    nError = RegOpenKeyEx(HKEY_LOCAL_MACHINE, IAAS_KEY, NULL, KEY_ALL_ACCESS, &returnKey);    
    
    if (nError == ERROR_SUCCESS)
    {
        LOGINFO("Registry open success");
        openFlag =true;
        LONG nError = RegQueryValueEx(returnKey, HOSTNETAGENT_ADDRESS_KEY, nullptr, &type, (LPBYTE)addr, &size);

        if (nError == ERROR_SUCCESS)
        {
            LOGINFOW(L"Retrieved HostNetAgentAddress from registry %s", addr);
            goto Cleanup;
        }
        LOGERRW(L"Retrieved HostNetAgentAddress from IAAS registry failed %d", nError);
    }
    
    if(openFlag)
    {
        RegCloseKey(returnKey);
        openFlag= false;
    }

    LOGINFOW(L"Going to open registry in paas %s", PAAS_KEY);
    
    nError = RegOpenKeyEx(HKEY_LOCAL_MACHINE, PAAS_KEY, NULL, KEY_ALL_ACCESS, &returnKey);
    
    if (nError != ERROR_SUCCESS)
    {
        LOGWARNW(L"Error while retrieving wireserver url registry key");
        goto Cleanup;
    }
    
    openFlag =true;
    
    nError = RegQueryValueEx(returnKey, HOSTNETAGENT_ADDRESS_KEY, nullptr, &type, (LPBYTE)addr, &size);
    
    if (nError != ERROR_SUCCESS)
    {
        LOGINFOW(L"Retrieved HostNetAgentAddress from IAAS registry %d", nError);
        goto Cleanup;
    }

Cleanup:

    if(openFlag)
    {
        RegCloseKey(returnKey);
    }

    if(nError != ERROR_SUCCESS)
    {
        if( swprintf_s(url, urlSize, L"http://%s:80/%s", DEFAULT_HOSTNETAGENT_ADDRESS, path) < 0)
        {
            LOGERRW(L"swprint_s error constructing url path %s", DEFAULT_HOSTNETAGENT_ADDRESS);
        }
    }
    else
    {
        LOGINFOW(L"Address returned from registry : %s", addr);
        if( swprintf_s(url, urlSize, L"http://%s:80/%s", addr, path) < 0)
        {
            LOGERRW(L"swprint_s error constructing url path addr %s", addr);
        }        
    }

    LOGINFOW(L"wireserver url : %s", url);
}


void VFPluginUtilities::SendStatusToWireServer(
    wchar_t *url,
    wchar_t *stateInfo
)
/**
Routine description:

    This function sends status of VF driver as xml string to NMAgent.

Arguments:
    url - url to send status.
    stateInfo - string contains status information.

Return Value:

**/
{
    http_client client(url);
    http_request requester;

    requester.set_method(methods::POST);
    requester.headers().set_content_type(U("text/xml;charset=utf-8"));
	
    json::value postData = web::json::value::string(stateInfo);
    LOGINFOW(L"Going to send in string %s", postData.as_string().c_str());

    requester.set_body(postData.as_string().c_str());   

     pplx::task<void> requestTask = client.request(requester).then([](http_response response)
    {
        if(response.status_code() == status_codes::OK)
        {
            LOGINFOW(L"NmAgent Server Status sent success");
        }
        else
        {
            LOGERRW(L"Error:: NmAgent Server HTTP post status code %d", response.status_code());
        }
    });
    
    try
    {
        requestTask.wait();
    }
    catch (const std::exception &e)
    {
        LOGINFO("NmAgent Server Error exception:%s\n", e.what());
    }
}

HRESULT VFPluginUtilities::RetrieveStatusInfoAsXMLStr(
    BOOL isInstalled, 
    wchar_t *installedVersion, 
    wchar_t *osVersion,
    int numNonFunctioningDevices,
    wchar_t **statusInfo
)
/**
Routine description:

    This function converts status info in string to xml based string.

Arguments:
    isInstalled - indicates if driver installed or not.
    installedVersion - This contains version value in string if driver is installed.
    osVersion - OS version info.
    numNonFunctioningDevices - Number of non functioning devices in VM using mellanox driver.
    statusInfo - string contains status information.

Return Value:
    returns S_OK if succeed else error.

**/
{
    *statusInfo = NULL;
    IXMLDOMElement *pRoot = NULL;
    IXMLDOMElement *osNode = NULL;
    IXMLDOMElement *mlxVersionNode = NULL;
    IXMLDOMElement *devStatusNode = NULL;
    XmlWriterHelper xmlWriter;
    HRESULT hr = S_OK;
    wchar_t numNonFunctioningInStr[MAX_PATH];
    
    CHK_HR(CoInitialize(NULL));

    xmlWriter.Initialize();

    CHK_HR(xmlWriter.CreateAndAddPINode(L"xml", L"version='1.0'"));
    CHK_HR(xmlWriter.CreateElement(L"ContainerIOVInfo", &pRoot));

    CHK_HR(xmlWriter.CreateElement(L"key", &osNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Name", L"ContainerOS", osNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Value", osVersion, osNode));

    CHK_HR(xmlWriter.CreateElement(L"key", &mlxVersionNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Name", L"MlxVFNicDriverVersion", mlxVersionNode));

    if(isInstalled)
    {
        CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Value", installedVersion, mlxVersionNode));
    }
    else
    {
        CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Value", L"", mlxVersionNode));
    }

    if(swprintf_s(numNonFunctioningInStr, MAX_PATH, L"%d", numNonFunctioningDevices) < 0)
    {
        LOGERRW(L"Error in swprintf_s to convert num non fn devices to str :%d", numNonFunctioningDevices);
        hr = E_FAIL;
        goto CleanUp;
    }

    CHK_HR(xmlWriter.CreateElement(L"key", &devStatusNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Name", L"MlxVFNumNonFunctioningDevices", devStatusNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Value", numNonFunctioningInStr, devStatusNode));

    CHK_HR(xmlWriter.AppendChildToParent(osNode, pRoot));
    CHK_HR(xmlWriter.AppendChildToParent(mlxVersionNode, pRoot));
    CHK_HR(xmlWriter.AppendChildToParent(devStatusNode, pRoot));    
    CHK_HR(xmlWriter.AppendChildToParent(pRoot, NULL));

    CHK_HR(xmlWriter.GetXMLInString(statusInfo));

    LOGINFOW(L"Xml string %s", *statusInfo);

CleanUp:

    xmlWriter.UnInitialize();
    SAFE_RELEASE(pRoot);
    SAFE_RELEASE(osNode);
    SAFE_RELEASE(mlxVersionNode);    
    SAFE_RELEASE(devStatusNode);
    CoUninitialize();

    return hr;
}

void VFPluginUtilities::SetupLogging(wchar_t *logFolder, wchar_t *logPrefix)
/**
Routine description:

    This function sets up logging for VFPlugin.

Arguments:
    logFolder - logFolder path.
    logPrefix -  logprefix for filename.

Return Value:

**/
{
    wchar_t *logDestFolder = NULL;
    
    size_t len = wcslen(logFolder) + wcslen(L"\\") + wcslen(logPrefix) + sizeof(wchar_t);

    logDestFolder = new wchar_t[len];
    if(!logDestFolder)
    {
        wprintf(L"Allocation of logdest folder failed\n");
        return;
    }

    if(swprintf_s(logDestFolder, len, L"%s\\%s", logFolder, logPrefix) < 0)
    {
        wprintf(L"swprintf_s failed constructing log dest folder\n");
        delete[] logDestFolder;
        return;
    }

    if (!CreateDirectory(logDestFolder, NULL))
    {
        DWORD error = GetLastError();
        if (error != ERROR_ALREADY_EXISTS)
        {
            wprintf(L"Failed to create log folder [%s] due to erroro (%d)\n", logDestFolder, error);
            delete[] logDestFolder;
            return;
        }
    }

    LOGSETUP(logDestFolder, logPrefix);
    delete[] logDestFolder;
}

void VFPluginUtilities::SaveInstalledState()
/**
Routine description:

    This function saves state if netagent installs mellanox driver in registry.

Arguments:

Return Value:

**/
{
    HKEY returnKey;
    DWORD type = REG_SZ;
    bool openFlag = false;
    LONG nError;

    LOGINFOW(L"Going to open registry in Iaas for saving install state %s", IAAS_KEY);
    
    nError = RegOpenKeyEx(HKEY_LOCAL_MACHINE, IAAS_KEY, NULL, KEY_ALL_ACCESS, &returnKey);    
    
    if (nError == ERROR_SUCCESS)
    {
        openFlag = true;
        nError = RegSetKeyValue(returnKey, nullptr, INSTALL_KEY, type, INSTALL_VALUE, (DWORD) (wcsnlen(INSTALL_VALUE, MAX_PATH) + 1) * sizeof(wchar_t));
        if (nError == ERROR_SUCCESS)
        {
            LOGINFO("Registry save succeeded");
            goto Cleanup;
        }
    }
   
    if(openFlag)
    {
        RegCloseKey(returnKey);
        openFlag= false;
    }

    LOGINFOW(L"Going to open registry in paas for saving install state %s", PAAS_KEY);
    
    nError = RegOpenKeyEx(HKEY_LOCAL_MACHINE, PAAS_KEY, NULL, KEY_ALL_ACCESS, &returnKey);
    
    if (nError != ERROR_SUCCESS)
    {
        LOGWARNW(L"Error while retrieving Wireserver url registry key");
        goto Cleanup;
    }
    
    openFlag =true;
           
    nError = RegSetKeyValue(returnKey, nullptr, INSTALL_KEY, type, INSTALL_VALUE, (DWORD) (wcsnlen(INSTALL_VALUE, MAX_PATH) + 1) * sizeof(wchar_t));
    
    if (nError != ERROR_SUCCESS)
    {
        LOGINFO("Registry save failed");
        goto Cleanup;
    }
    
    LOGINFO("Registry save succeeded");
Cleanup:

    if(openFlag)
    {
        RegCloseKey(returnKey);
    } 
}

void VFPluginUtilities::SendHealthStatusToHostAgent(
    wchar_t *url,
    HealthReport& report
)
/**
Routine description:

    This function sends health report to host netagent.

Arguments:
    url - url of host netagent.
    report - health report info.

Return Value:

**/
{
    http_client client(url);
    http_request requester;
    json::value obj = NULL;

    requester.set_method(methods::POST);
    requester.headers().set_content_type(U("application/json"));

    obj = report.TranslateToJsonValue();

    requester.set_body(obj);   

     pplx::task<void> requestTask = client.request(requester).then([](http_response response)
    {
        if(response.status_code() == status_codes::OK)
        {
            LOGINFOW(L"Health Report status sent success");
        }
        else
        {
            LOGERRW(L"Error::Health Report Server HTTP post status code %d", response.status_code());
        }
    });
    
    try
    {
        requestTask.wait();
    }
    catch (const std::exception &e)
    {
        LOGINFO("Error::Health Report Server exception:%s\n", e.what());
    }
}

void VFPluginUtilities::ResetHealthObject(
        HealthReport& report
)
/**
Routine description:

    This function is used to reset health report object to default values.

Arguments:
    report - health report object.

Return Value:

**/
{
    report.GuestOSFamily.assign(L"Windows");
    report.DriverInstallNotAttempted = false;
    report.DriverInstallAttempted = false;
    report.DriverInstallSucceeded = false;
    report.DriverInstallFailed = false;
    report.TotalVFDevicesCount = 0;
    report.NonFunctionalVFDevicesCount = 0;
    report.ErrorCode = 0;
}