/*++

Copyright (c) 2019  Microsoft Corporation

Module Name:

    parser.cpp

Abstract:

    Parser to generate PluginSpecificTemplate.xml during HostNetAgent startup.

--*/

// parser.cpp : Defines the entry point for the console application.
//
#include "stdafx.h"
#include <tchar.h>
#include <stdio.h>
#include <msxml6.h>
#include <time.h>
#include <string>
#include <psapi.h>
#include <fstream>
#include <sstream>
#include <regex>
#include <comutil.h>
#include "logger/log.h"

#pragma comment(lib, "comsuppw.lib")  

using namespace std;

// Macro that calls a COM method returning HRESULT value.
#define CHK_HR(stmt)        do { hr=(stmt); if (FAILED(hr)) goto CleanUp; } while(0)

// Macro to verify memory allcation.
#define CHK_ALLOC(p)        do { if (!(p)) { hr = E_OUTOFMEMORY; goto CleanUp; } } while(0)

// Macro that releases a COM object if not NULL.
#define SAFE_RELEASE(p)     do { if ((p)) { (p)->Release(); (p) = NULL; } } while(0)


#define WCLUSTER_NAME L"ClusterName"
#define WREGION L"Region"
#define NETAGENTGCSREGION L"NetAgentGcsRegion"
#define PLUGIN_SPECIFIC_FILE_NAME "-pluginSpecificTemplate"
#define PLUGIN_SPECIFIC_OUT_FILENAME "-pluginSpecificOutfile"
#define CLOUD_GROUP_MAPPING_HEADER_FILE "-cloudGroupMappingHeaderFile"
#define INPUT_CLOUD_NAME "-inputCloudName"
#define INPUT_TENANT_NAME "-serviceName"
#define SECRET_CONFIGURATION_FILE "-secretConfigurationFile"
#define SECRET_DEPLOYMENT_FILE "-secretDeploymentFile"
#define SHARED_INI "-sharedIni"
#define DCMT_SETTINGS_INI_FILE "-dcmtSettingsIniFile"
#define MAX_INI_SECTION_SIZE 32767
#define GLOBAL_SETTINGS_INI_FILE "-globalSettingsIniFile"
#define INIBUFFER 1024
#define CLOUD_SECTION "Cloud"
#define GLOBAL_SECTION "GlobalValues"
#define KEY_NOT_FOUND "KeyNotFound"

wchar_t* globalScretsConfigurationJson;
LPCSTR globalSharedIni;
LPCSTR globalSecretsDeploymentIni;
LPCSTR dcmtSettingsIniFile;
LPCSTR globalSettingsIniFile;
std::string cloudGroupName;

/*
 Helper function to create a VT_BSTR variant from a null terminated string.
 */
HRESULT
VariantFromString(
    PCWSTR wszValue,
    VARIANT &Variant)
{
    HRESULT hr = S_OK;
    BSTR bstr = SysAllocString(wszValue);
    CHK_ALLOC(bstr);

    V_VT(&Variant) = VT_BSTR;
    V_BSTR(&Variant) = bstr;

CleanUp:
    return hr;
}

/*
Helper function to create a DOM instance.
*/
HRESULT
CreateAndInitDOM(
    IXMLDOMDocument **ppDoc)
{
    HRESULT hr = CoCreateInstance(__uuidof(DOMDocument60), NULL, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(ppDoc));
    if (SUCCEEDED(hr))
    {
        (*ppDoc)->put_async(VARIANT_FALSE);
        (*ppDoc)->put_validateOnParse(VARIANT_FALSE);
        (*ppDoc)->put_resolveExternals(VARIANT_FALSE);
    }

    return hr;
}

BSTR
toLowerBSTR(
    BSTR input
)
{
    if (input == NULL)
    {
        return input;
    }
    
    BSTR ret = SysAllocString(input);
    size_t len = SysStringLen(ret) + 1;
    _wcslwr_s(ret, len);
    return ret;
}

BSTR
concatBSTR(
    BSTR a,
    BSTR b
)
{
    BSTR bstrResult;
    bstrResult = (_bstr_t(a) + _bstr_t(b)).copy();
    return bstrResult;
}

std::string&
ltrim(
    std::string& str,
    const std::string& chars = "\t\n\v\f\r "
)
{
    str.erase(0, str.find_first_not_of(chars));
    return str;
}

std::string&
rtrim(
    std::string& str,
    const std::string& chars = "\t\n\v\f\r "
)
{
    str.erase(str.find_last_not_of(chars) + 1);
    return str;
}

std::string&
trim(
    std::string& str,
    const std::string& chars = "\t\n\v\f\r "
)
{
    return ltrim(rtrim(str, chars), chars);
}

std::string 
toLowerCase(std::string input)
{
    for (int i = 0; i < input.length(); i++)
    {
        input[i] = tolower(input[i]);
    }

    return input;
}

bool
searchString(std::string delimiter, std::string searchKey, std::string value)
{
    value = toLowerCase(trim(value));
    searchKey = toLowerCase(trim(searchKey));
    size_t pos = 0;
    std::string token;
    while ((pos = value.find(delimiter)) != std::string::npos) 
    {
        token = value.substr(0, pos);
        if(token.compare(searchKey) == 0)
        {
            return true;
        }
        value.erase(0, pos + delimiter.length());
    }

    if (value.find(delimiter) == std::string::npos)
    {
        if (value.compare(searchKey) == 0)
        {
            return true;
        }
    } 

    return false;
}

/*
This function will fetch  the $<Setting>$ value from the ini file 
*/
BSTR 
findSettingInIniFile(
    BSTR inputSettingName
)
{
     BSTR lowerSettingName = toLowerBSTR(inputSettingName);
     std::string settingName = _bstr_t(lowerSettingName);
     DWORD status = 0;
     char chIniBuffer[INIBUFFER] = "";
      
    // Look for the setting in the cloud section
    status = GetPrivateProfileString(CLOUD_SECTION,
                                     settingName.c_str(),
                                     KEY_NOT_FOUND,
                                     chIniBuffer,
                                     INIBUFFER,
                                     dcmtSettingsIniFile);

    if (std::string(chIniBuffer).compare(KEY_NOT_FOUND) == 0)
    {
        // Look for the setting in the cloudgroup section  
        status = GetPrivateProfileString(cloudGroupName.c_str(),
                                settingName.c_str(),
                                KEY_NOT_FOUND,
                                chIniBuffer,
                                INIBUFFER,
                                dcmtSettingsIniFile);
    }     

    if (std::string(chIniBuffer).compare(KEY_NOT_FOUND) == 0)
    {
        std::string settingsFileName = (globalSettingsIniFile == NULL || globalSettingsIniFile == 0) ? 
                                   dcmtSettingsIniFile : globalSettingsIniFile;
        // Look for the setting in the global section
        GetPrivateProfileString(GLOBAL_SECTION,
                                settingName.c_str(),
                                KEY_NOT_FOUND,
                                chIniBuffer,
                                INIBUFFER,
                                settingsFileName.c_str());        
        
    }

    SysFreeString(lowerSettingName);    
    return SysAllocString(_bstr_t(chIniBuffer));
}

std::string
replaceGenerateFirstRdsFromVipList(
    std::string towriteLine)
{
    std::string ret;
    std::string tmpStr;
    int i = 0;
    size_t vectLength = 0;
    std::vector<std::string> vect;
    std::istringstream iss(towriteLine);
    while (getline(iss, tmpStr, ','))
    {
        tmpStr = trim(tmpStr);
        vect.push_back(tmpStr);
    }

    tmpStr = vect.at(0);
    std::size_t pos = tmpStr.find("http");
    tmpStr = tmpStr.substr(pos);
    vectLength = vect.size();
    if (vectLength < 2)
    {
        ret = "";
    }
    else
    {
        vect.at(vectLength - 1).pop_back();
        vect.at(vectLength - 1).pop_back();
        ret = ret.append(tmpStr.replace(tmpStr.find("{0}"), 3, vect.at(1)));
    }

    return ret;
}

std::string
replaceGenerateRdsVipList(
    std::string towriteLine)
{
    std::string ret;
    std::string tmpStr;
    int i = 0;
    size_t vectLength = 0;
    std::vector<std::string> vect;
    std::istringstream iss(towriteLine);
    while (getline(iss, tmpStr, ','))
    {
        tmpStr = trim(tmpStr);
        vect.push_back(tmpStr);
    }

    tmpStr = vect.at(0);
    std::size_t pos = tmpStr.find("http");
    tmpStr = tmpStr.substr(pos);
    vectLength = vect.size();
    vect.at(vectLength - 1).pop_back();
    vect.at(vectLength - 1).pop_back();
    for (i = 1; i<vectLength; ++i)
    {
        std::string templateStr = tmpStr + ", ";
        ret = ret.append(templateStr.replace(templateStr.find("{0}"), 3, vect.at(i)));
    }

    ret.pop_back();
    ret.pop_back();
    return ret;
}

void getRegionFromIniFile(TCHAR *regionValue)
{
    Log("Reading from %s", globalSharedIni);
    GetPrivateProfileString(_T("General"), _T("Region"), _T(""), regionValue, 1024, globalSharedIni);
}

std::string
replaceFunctionCalls(
    std::string towriteLine
)
{
    std::regex regexGenerateRdsVipList("#Function_GenerateRdsVipList(.*)#");
    std::regex regexGenerateFirstRdsFromVipList("#Function_GenerateFirstRdsFromVipList(.*)#");
    std::regex regexGetClusterRegion("#Function_GetClusterRegion(.*)#");
    if (regex_search(towriteLine, regexGenerateRdsVipList))
    {
        towriteLine = replaceGenerateRdsVipList(towriteLine);
    }
    else if (regex_search(towriteLine, regexGenerateFirstRdsFromVipList))
    {
        towriteLine = replaceGenerateFirstRdsFromVipList(towriteLine);
    }
    else if (regex_search(towriteLine, regexGetClusterRegion))
    {
        TCHAR regionValue[1024];
        getRegionFromIniFile(regionValue);
        towriteLine = regionValue;
    }

    return towriteLine;
}

int ProcessSiblings(IXMLDOMNode *curRootNode, BSTR inputCloudName, BSTR inputTenantName);

/*
This function return errorCode 1 if there are missing values in the PluginSpecificTemplate
*/
int
ProcessNode(
    IXMLDOMNode *node,
    BSTR inputCloudName,
    BSTR inputTenantName
    )
{
    int errorCode = 0;
    if (!node)
    {
        return errorCode;
    }

    BSTR xmlBSTR = NULL;
    IXMLDOMNode *pFirstChild = NULL;
    VARIANT_BOOL hasChild = FALSE;
    VARIANT val;
    DOMNodeType nodeType;
    BSTR nodeName = NULL;
    BSTR inputSettingName = NULL;
    std::regex rgx("\\$.*?\\$");

    node->get_nodeType(&nodeType);
    node->get_nodeValue(&val);
    node->get_xml(&xmlBSTR);
    node->get_nodeName(&nodeName);

    if (nodeType == NODE_TEXT)
    {
        node->get_nodeValue(&val);
        if (val.bstrVal)
        {
            std::string towriteLine(_com_util::ConvertBSTRToString(val.bstrVal));
            std::string line(_com_util::ConvertBSTRToString(val.bstrVal));
            if (regex_search(line, rgx))
            {
                for (std::sregex_iterator itr = std::sregex_iterator(line.begin(), line.end(), rgx);
                    itr != std::sregex_iterator();
                    ++itr)
                {
                    BSTR ret = NULL;
                    std::smatch match = *itr;
                    size_t lengthOfSettingStr = match.str().length();
                    std::string matchedString = match.str();
                    std::wstring settingToFind(matchedString.begin(), matchedString.end());
                    settingToFind.replace(0, 1, L"").replace(lengthOfSettingStr - 2, 1, L"");
                    inputSettingName = ::SysAllocString(settingToFind.c_str());
                    if (settingToFind.compare(WCLUSTER_NAME) == 0)
                    {
                        ret = ::SysAllocString(inputCloudName);
                    }
                    else if (settingToFind.compare(WREGION) == 0)
                    {
                        //Read .ini file to get Region value
                        TCHAR tmpRet[1024];
                        getRegionFromIniFile(tmpRet);
                        ret = _bstr_t(tmpRet).copy();
                    }
                    else
                    {
                        Log("OriginalinputSettingName=%S",inputSettingName);
                        if (settingToFind.compare(NETAGENTGCSREGION) == 0)
                        {
                            settingToFind = WREGION;
                            inputSettingName = ::SysAllocString(settingToFind.c_str());
                        }
                        Log("inputSettingName=%S",inputSettingName);
                        ret = findSettingInIniFile(inputSettingName);
                        std::string returnString = _bstr_t(ret);
                        if (returnString.compare(KEY_NOT_FOUND) == 0)
                        {
                            errorCode = 1;
                            Log("Skip parsing as setting %S not found", settingToFind.c_str());
                            return errorCode;
                        } 
                    }

                    Log("After processing setting: %S, return value is:%S", settingToFind.c_str(), ret);
                    if (ret)
                    {
                        char* pRetChar = _com_util::ConvertBSTRToString(ret);
                        towriteLine.replace(towriteLine.find(matchedString), matchedString.length(), pRetChar);
                        delete[] pRetChar;
                        SysFreeString(ret);
                    }
                }
            }

            towriteLine = replaceFunctionCalls(towriteLine);
            BSTR towriteLineBSTR = _bstr_t(towriteLine.c_str()).copy();
            node->put_text(towriteLineBSTR);
        }
    }

    node->hasChildNodes(&hasChild);
    if (hasChild)
    {
        node->get_firstChild(&pFirstChild);
        errorCode = ProcessSiblings(pFirstChild, inputCloudName, inputTenantName);
    }

    return errorCode;
}

/*
This function return errorCode 1 if there are missing values in the PluginSpecificTemplate
*/
int 
ProcessSiblings(
    IXMLDOMNode *curRootNode,
    BSTR inputCloudName,
    BSTR inputTenantName
    )
{
    int errorCode = 0;
    if (!curRootNode)
    {
        return errorCode;
    }

    IXMLDOMNode *pSibling = NULL;
    errorCode = ProcessNode(curRootNode, inputCloudName, inputTenantName);
    if (errorCode != 0)
    {
        return errorCode;
    }

    curRootNode->get_nextSibling(&pSibling);
    for (; pSibling != NULL; pSibling->get_nextSibling(&pSibling))
    {
        errorCode = ProcessNode(pSibling, inputCloudName, inputTenantName);
        if (errorCode != 0)
        {
            return errorCode;
        }
    }

    return errorCode;
}

void
loadDOMRaw(
    BSTR pluginSpecificFileName,
    BSTR pluginSpecificOutFileName,
    BSTR inputCloudName,
    BSTR inputTenantName,
    int &errorCode)
{
    HRESULT hr = S_OK;
    IXMLDOMParseError *pXMLErr = NULL;
    IXMLDOMElement *pIXMLDOMElement = NULL;

    BSTR inputSettingName = NULL;
    BSTR bstrErr = NULL;
    VARIANT_BOOL varStatus;
    std::string line;
    int line_no = 0;

    IXMLDOMElement *pPluginSpecificTemplateRoot = NULL;
    IXMLDOMDocument *pReadXMLDom = NULL;
    VARIANT dest;
    VARIANT pluginSpecificTemplateFileName;

    CHK_HR(CreateAndInitDOM(&pReadXMLDom));
    CHK_HR(VariantFromString(pluginSpecificFileName, pluginSpecificTemplateFileName));
    CHK_HR(pReadXMLDom->load(pluginSpecificTemplateFileName, &varStatus));
    if (varStatus == VARIANT_FALSE)
    {
            // Failed to load flattened xml, get last parsing error
        CHK_HR(pReadXMLDom->get_parseError(&pXMLErr));
        CHK_HR(pXMLErr->get_reason(&bstrErr));
        Log("Failed to load DOM from %S. %S", pluginSpecificFileName, bstrErr);
        errorCode = 200;
    }
    else
    {
        Log("Reading the pluginspecifictemplate");
        pReadXMLDom->get_documentElement(&pPluginSpecificTemplateRoot);
        Log("Processing siblings");
        errorCode = ProcessSiblings(pPluginSpecificTemplateRoot, inputCloudName, inputTenantName);
        if (errorCode == 0)
        {
            dest.vt = VT_BSTR;
            dest.bstrVal = pluginSpecificOutFileName;
            Log("Saving pluginSpecificConfig file");
            pReadXMLDom->save(dest);
            SAFE_RELEASE(pReadXMLDom);
        }
    }
    
CleanUp:
    SAFE_RELEASE(pXMLErr);
    SysFreeString(bstrErr);
    SysFreeString(inputSettingName);
}


void
PrintMemoryInfo()
{
    PROCESS_MEMORY_COUNTERS pmc;
    BOOL result = GetProcessMemoryInfo(GetCurrentProcess(),
        &pmc,
        sizeof(pmc));

    if (result)
    {
        Log("Memory Usage:\n");
        Log("\tPagefileUsage: %d bytes", pmc.PagefileUsage);
        Log("\tPeakPagefileUsage: %d bytes",
            pmc.PeakPagefileUsage);

    }
}

/*
This function will return the cloudgroup corresponding to the environment
*/
std::string 
GetCloudGroupName(BSTR cloudGroupMappingHeaderFile, BSTR inputCloudName)
{
    std::string line = "";
    std::string sectionName = "";
    BSTR inputCloudNameLower = toLowerBSTR(inputCloudName);
    char buffer[MAX_INI_SECTION_SIZE];
    int charsRead = GetPrivateProfileSectionNames(buffer, MAX_INI_SECTION_SIZE, _bstr_t(cloudGroupMappingHeaderFile));
    int i = 0;
    int flag = 0;
    while (i < charsRead)
    {
        if (buffer[i] == '\0')
        {
            char keyValueBuffer[MAX_INI_SECTION_SIZE];
            int count = GetPrivateProfileSection(line.c_str(), keyValueBuffer, MAX_INI_SECTION_SIZE, _bstr_t(cloudGroupMappingHeaderFile));
            std::string keyValue = keyValueBuffer;
            std::string key = keyValue.substr(0, keyValue.find("="));
            std::string value = keyValue.substr(keyValue.find("=") + 1);
            if(searchString(",", std::string(_bstr_t(inputCloudNameLower)), value))
            {
                sectionName = key;
                break;
            }
            line = "";
        }
        else
        {
            line = line + buffer[i];
        }

        i++;
    }

    return sectionName;
}

int
_tmain(
    int argc,
    _TCHAR* argv[])
{
    clock_t t = NULL;
    double time_taken = 0;
    BSTR pluginSpecificFileName = NULL;
    BSTR pluginSpecificOutFileName = NULL;
    BSTR cloudGroupMappingHeaderFile = NULL;
    BSTR inputCloudName = NULL;
    BSTR inputTenantName = NULL;
    int errorCode = 0;
    int count = 0;
    
    if (argc < 13 || argc > 21)
    {
        Log("Incorrect invocation of program. Use following format to execute");
        Log("%s -pluginSpecificTemplate pluginSpecificTemplate -pluginSpecificOutFile pluginSpecificOutFile -cloudGroupMappingHeaderFile cloudGroupMappingHeaderFile -dcmtSettingsIniFile dcmtSettingsIniFile -inputCloudName inputCloudName -serviceName serviceName -secretConfigurationFile secretConfigurationFile -sharedIni sharedIni -secretDeploymentFile secretDeploymentFile -globalSettingsIniFile globalSettingsIniFile", argv[0]);
        Log("Secretsconfiguration.json and secret deployment.ini are optional params");
        return 87;
    }
   
    for( int i = 1 ; i < argc ; i+=2)
    {
        if (_tcscmp(argv[i], PLUGIN_SPECIFIC_FILE_NAME) == 0)
        {
            pluginSpecificFileName = _bstr_t(argv[i+1]).copy();
            count++;
        }

        if (_tcscmp(argv[i], PLUGIN_SPECIFIC_OUT_FILENAME) == 0)
        {
            pluginSpecificOutFileName = _bstr_t(argv[i+1]).copy();
            count++;
        }
        
        if (_tcscmp(argv[i], CLOUD_GROUP_MAPPING_HEADER_FILE) == 0)
        {
            cloudGroupMappingHeaderFile = _bstr_t(argv[i+1]).copy();
            count++;
        }

        if (_tcscmp(argv[i], INPUT_CLOUD_NAME) == 0)
        {
            BSTR tmparg4 = _bstr_t(argv[i+1]).copy();
            wchar_t *nextToken = NULL;
            wcstok_s(tmparg4, L"-", &nextToken);
            inputCloudName = _bstr_t(tmparg4).copy();
            count++;
        }

        if (_tcscmp(argv[i], INPUT_TENANT_NAME) == 0)
        {
            inputTenantName = _bstr_t(argv[i+1]).copy();
            count++;
        }

        if (_tcscmp(argv[i], SECRET_CONFIGURATION_FILE) == 0)
        {
            globalScretsConfigurationJson = _bstr_t(argv[i+1]).copy();
        }

        if (_tcscmp(argv[i], SECRET_DEPLOYMENT_FILE) == 0)
        {
            globalSecretsDeploymentIni = argv[i+1] ;
        }

        if (_tcscmp(argv[i], SHARED_INI) == 0)
        {
            globalSharedIni = argv[i+1] ;
            count++;
        }

        if (_tcscmp(argv[i], DCMT_SETTINGS_INI_FILE) == 0)
        {
            dcmtSettingsIniFile = argv[i+1] ;
            count++;
        }

        if (_tcscmp(argv[i], GLOBAL_SETTINGS_INI_FILE) == 0)
        {
            globalSettingsIniFile = argv[i+1] ;
        }
    }

    if (count < 7)
    {
        Log("Incorrect invocation of program. Missing mandatory params");
        Log("PluginSpecificTemplate PluginSpecificOutFile DcmtSettingsIniFile InputCloudName ServiceName SharedIni CloudGroupMappingHeaderFile are mandatory params");
        return 87;
    }

    t = clock();
    cloudGroupName = GetCloudGroupName(cloudGroupMappingHeaderFile,inputCloudName);
    HRESULT hr = CoInitialize(NULL);

    if (SUCCEEDED(hr))
    {
        loadDOMRaw(pluginSpecificFileName, pluginSpecificOutFileName, inputCloudName, inputTenantName, errorCode);
        CoUninitialize();
    }

    t = clock() - t;
    time_taken = ((double)t) / CLOCKS_PER_SEC; // in seconds
    Log("Time taken: %f seconds", time_taken);
    PrintMemoryInfo();
    return errorCode;
}

