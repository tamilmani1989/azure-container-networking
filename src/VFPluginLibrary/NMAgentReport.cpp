/*++

Copyright (c) Microsoft Corporation

Module Name:

NMAgentReport.cpp

Abstract:

This file contains implementation of methods reporting status to nmagent.

Author:

Tamilmani Manoharan (tamanoha) 14-November-2017

--*/
#include "NMAgentReport.h"
#include <cpprest/json.h>
#include <cpprest/http_client.h>
#include "..\Logging\Logging.h"
#include "XmlWriterHelper.h"

using namespace std;
using namespace web;
using namespace http;
using namespace web::http::client;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

int status = 0;

NMAgentReport::NMAgentReport()
/**
Routine description:

The constructor initializes nmAgenturl.

Arguments:

Return Value:

**/
{
    VFPluginUtilities::GetHostNetAgentURL(nmAgentUrl, sizeof(nmAgentUrl), HOSTAGENT_PATH);
}

int NMAgentReport::SendStatusToNMAgent(
    wchar_t *statusInfo
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
    http_client client(nmAgentUrl);
    http_request requester;

    requester.set_method(methods::POST);
    requester.headers().set_content_type(U("text/xml;charset=utf-8"));

    json::value postData = web::json::value::string(statusInfo);
    LOGINFOW(L"Going to send in string %s", postData.as_string().c_str());

    requester.set_body(postData.as_string().c_str());

    pplx::task<void> requestTask = client.request(requester).then([](http_response response)
    {
        if (response.status_code() == status_codes::OK)
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
        status = 1;
    }

    return status;
}

HRESULT NMAgentReport::RetrieveStatusInfoAsXMLStr(
    BSTR *statusInfo
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
    HRESULT hr = S_OK;
    *statusInfo = NULL;
    XmlWriterHelper xmlWriter;
    IXMLDOMElement *pRoot = NULL;
    IXMLDOMElement *osNode = NULL;
    IXMLDOMElement *osVersionNode = NULL;
    IXMLDOMElement *mlxVersionNode = NULL;
    IXMLDOMElement *devStatusNode = NULL;

    wchar_t numNonFunctioningInStr[MAX_PATH];

    xmlWriter.Initialize();

    CHK_HR(xmlWriter.CreateAndAddPINode(L"xml", L"version='1.0'"));
    CHK_HR(xmlWriter.CreateElement(L"ContainerIOVInfo", &pRoot));

    CHK_HR(xmlWriter.CreateElement(L"key", &osNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Name", L"ContainerOS", osNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Value", OSFamily.c_str(), osNode));

    CHK_HR(xmlWriter.CreateElement(L"key", &mlxVersionNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Name", L"MlxVFNicDriverVersion", mlxVersionNode));

    if (IsInstalled)
    {
        CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Value", DriverVersion, mlxVersionNode));
    }
    else
    {
        CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Value", L"", mlxVersionNode));
    }

    if (swprintf_s(numNonFunctioningInStr, MAX_PATH, L"%d", NumNonFunctioningDevices) < 0)
    {
        LOGERRW(L"Error in swprintf_s to convert num non fn devices to str :%d", NumNonFunctioningDevices);
        hr = E_FAIL;
        goto CleanUp;
    }

    CHK_HR(xmlWriter.CreateElement(L"key", &devStatusNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Name", L"MlxVFNumNonFunctioningDevices", devStatusNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Value", numNonFunctioningInStr, devStatusNode));

    CHK_HR(xmlWriter.CreateElement(L"key", &osVersionNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Name", L"ContainerOSVersion", osVersionNode));
    CHK_HR(xmlWriter.CreateAndAddAttributeNode(L"Value", OSVersion.c_str(), osVersionNode));

    CHK_HR(xmlWriter.AppendChildToParent(osNode, pRoot));
    CHK_HR(xmlWriter.AppendChildToParent(mlxVersionNode, pRoot));
    CHK_HR(xmlWriter.AppendChildToParent(devStatusNode, pRoot));
    CHK_HR(xmlWriter.AppendChildToParent(osVersionNode, pRoot));
    CHK_HR(xmlWriter.AppendChildToParent(pRoot, NULL));

    CHK_HR(xmlWriter.GetXMLInString(statusInfo));

    LOGINFOW(L"Xml string %s", *statusInfo);

CleanUp:

    xmlWriter.UnInitialize();
    SAFE_RELEASE(osNode);
    SAFE_RELEASE(mlxVersionNode);
    SAFE_RELEASE(devStatusNode);
    SAFE_RELEASE(osVersionNode);
    SAFE_RELEASE(pRoot);
    return hr;
}
