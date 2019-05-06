/*++

Copyright (c) Microsoft Corporation

Module Name:

ConfigParser.cpp

Abstract:

This file contains methods to read plugins that has to be started from settings file.

Author:

Tamilmani Manoharan (tamanoha) 06-September-2017

--*/

#include "ConfigParser.h"
#include "../Shared/Utilities.h"
#include "../Logging/Logging.h"
#include <cpprest/json.h>

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;
using namespace web;


void readNetAgentConfig(json::value val, HandlerConfig *config)
/*++

Routine Description:
    This function process the json settings file to retrieve the plugins that has to be started.

Arguments:
    jsonValue - json format of configurtion file

Return Value:
    returns array of plugins as json::value on success else returns NULL

--*/
{
    json::value enablePluginsVal=NULL;
    json::array arrval = val.as_array();

    for (auto arrIter = arrval.cbegin(); arrIter != arrval.cend(); ++arrIter)
    {
        for(auto handlerIter = arrIter->as_object().cbegin(); handlerIter != arrIter->as_object().cend(); ++handlerIter)
        {
            const utility::string_t &handlerKey = handlerIter->first;
            const json::value &handlerValue = handlerIter->second;

            if (wcscmp(handlerKey.c_str(), L"handlerSettings") == 0)
            {
                for (auto publicIter = handlerValue.as_object().cbegin(); publicIter != handlerValue.as_object().cend(); ++publicIter)
                {
                    const utility::string_t &publicKey = publicIter->first;
                    const json::value &publicValue = publicIter->second;

                    if (wcscmp(publicKey.c_str(), L"publicSettings") == 0)
                    {
                        for(auto pluginIter = publicValue.as_object().cbegin(); pluginIter != publicValue.as_object().cend(); ++pluginIter)
                        {
                            const utility::string_t &configKey = pluginIter->first;
                            const json::value &configValue = pluginIter->second;

                            if (wcscmp(configKey.c_str(), L"PluginsToEnable") == 0)
                            {
                                LOGINFO("Got the key pluginstoenable");
                                json::array pluginsArr = configValue.as_array();

                                for (auto iter = pluginsArr.cbegin(); iter != pluginsArr.cend(); ++iter)
                                {
                                    std::wstring plugin = iter->as_string();
                                    LOGINFOW(L"Insert plugin:%s\n", plugin.c_str());
                                    config->pluginsToStart.insert(plugin);
                                }
                            }
                            else if (wcscmp(configKey.c_str(), L"RestServerPort") == 0)
                            {
                                config->restServerPort =  configValue.as_string();
                                LOGINFOW(L"Got the key restserverport value %s", config->restServerPort.c_str());
                            }
                            else if (wcscmp(configKey.c_str(), L"DisableRestServer") == 0)
                            {
                                config->disableRestServer = configValue.as_bool();
                                LOGINFOW(L"Got the key disablerestserver value %d", config->disableRestServer);
                            }
                            else
                            {
                                LOGWARNW(L"Unknown key %s", configKey.c_str())
                            }
                        }
                    }
                }
            }
        }
    }
}

std::wstring getBaseFilename(const wchar_t *filename)
/*++

Routine Description:
    This helper function is to get base name of file without extension

Arguments:
    exeLocation - Filename with extension

Return Value:
        returns basename on success else NULL
--*/
{
    std::wstring fName(filename);
    size_t lastindex  = fName.find_last_of(L".");

    if (lastindex == string::npos)
    {
        return NULL;
    }

    std::wstring rawname = fName.substr(0, lastindex);

    return rawname;
}


int sortFileBySeqNumber(
    const void *elem1,
    const void *elem2
)
/*++

Routine Description:

Helper to compare WIN32_FIND_DATA objects by ConfigSequenceNumber

Arguments:

elem1 -To be compared object1.

elem2 - To be compared object2.

Return Value:

return number < 0 if elem1 < elem2 and > 0 if elem1 > elem2 and 0 if eleme1 is equal to elem2.

--*/
{
    WIN32_FIND_DATA *fd1 = (WIN32_FIND_DATA*)elem1;
    WIN32_FIND_DATA *fd2 = (WIN32_FIND_DATA*)elem2;

    std::wstring file1 = getBaseFilename(fd1->cFileName);
    std::wstring file2 = getBaseFilename(fd2->cFileName);

    if (file1.empty() || file2.empty())
    {
        LOGINFO("getBaseFileName returns empty");
        return 0;
    }

    int num1 = _wtoi(file1.c_str());
    int num2 = _wtoi(file2.c_str());

    if (num1 < num2)
    {
        return -1;
    }
    else if(num1 > num2)
    {
        return 1;
    }

    return 0;
}

wchar_t *HandlerConfig::getLatestConfigFilePath(wchar_t *configDir)
/*++

Routine Description:
    This function retrieves the latest config settings file

Arguments:
    configDir - Config Directory Path

Return Value:
    returns latest config file path on success else NULL
--*/
{
    WIN32_FIND_DATA fd;
    HANDLE hFind = INVALID_HANDLE_VALUE;
    std::wstring fileName;
    std::vector<WIN32_FIND_DATA> fdata;

    LOGINFOW(L"Inside getLatestConfigFilePath");
    LOGINFOW(L"Config dir%s\n", configDir);

    hFind = ::FindFirstFile(configDir, &fd);

    while (INVALID_HANDLE_VALUE != hFind) {
        fdata.push_back(fd);

        if (!::FindNextFile(hFind, &fd)) {
            ::FindClose(hFind);
            break;
        }
    }

    if (fdata.size() <= 0)
    {
        LOGINFO("fdata vector which holds files info is empty\n")
        return NULL;
    }

    qsort(
        &fdata[0],
        fdata.size(),
        sizeof(WIN32_FIND_DATA),
        sortFileBySeqNumber);

    this->latestSequenceNumber = getBaseFilename(fdata.back().cFileName);
    size_t len = wcslen(fdata.back().cFileName)+1;
    wchar_t *latestFile = new wchar_t[len];

    wcscpy_s(latestFile, len, fdata.back().cFileName);

    return latestFile;
}


std::string HandlerConfig::readConfigFile(wchar_t *configFilePath)
/*++

Routine Description:
    This is helper function to read config file and retrieve its contents

Arguments:
    configFilePath - path of config file

Return Value:
    returns contents of config file read in string
--*/
{
    std::string str;
    std::fstream fp(configFilePath);

    if(!fp)
    {
        LOGINFOW(L"Not able to open file %s", configFilePath);
        return str;
    }

    fp.seekg(0, std::ios::end);

    std::streamoff size = fp.tellg();

    if(size < 0)
    {
        LOGINFO("tellg() function failed in read config file");
        return str;
    }

    str.reserve(size);
    fp.seekg(0, std::ios::beg);

    str.assign((std::istreambuf_iterator<char>(fp)),
                std::istreambuf_iterator<char>());

    LOGINFO("HandlerConfig file %s", str.c_str());

    return str;

}

std::string HandlerConfig::ReadLatestConfigFile(wchar_t *configFolderPath)
/*++

Routine Description:
    This function finds and reads the latest config file

Arguments:
    configFolderPath - config folder location

Return Value:
    returns contents of config file read in string
--*/
{
    std::string str;
    errno_t err;
    wchar_t *configFilePath=NULL;
    wchar_t *configFile=NULL;
    size_t configDirLen = wcslen(configFolderPath);
    size_t configExtLen = wcslen(CONFIG_EXTENSION);
    size_t totalLen = configDirLen +  configExtLen + 1;

    LOGINFOW(L"Inside readHandlerEnvFields");

    wchar_t *configDir = new wchar_t[totalLen];

    if (!configDir)
    {
        LOGINFO("Error while allocating for configDir\n");
        goto Cleanup;
    }

    err = wcscpy_s(configDir, totalLen, configFolderPath);

    if(err != 0)
    {
        LOGINFO("Error while copying config dir path:%d\n", err);
        goto Cleanup;
    }

    err = wcscat_s(configDir, totalLen, CONFIG_EXTENSION);

    if(err != 0)
    {
        LOGINFO("Error while concatenating config dir path with extension:%d\n", err);
        goto Cleanup;
    }

    configFile = getLatestConfigFilePath(configDir);

    LOGINFOW(L"Latest config file :%s", configFile);

    if(!configFile)
    {
        LOGINFOW(L"GetLatestConfigFilePath function returns NULL configDir :%s\n", configDir);
        goto Cleanup;
    }

    size_t configFileLen = wcslen(configFile);
    totalLen = configDirLen +  configFileLen + 2;
    configFilePath = new wchar_t[totalLen];

    if(!configFilePath)
    {
        LOGINFO("Error while allocating for configFilePath\n");
        goto Cleanup;
    }

    err = wcscpy_s(configFilePath, totalLen, configFolderPath);

    if(err != 0)
    {
        LOGINFO("Error while copying config file path:%d\n", err);
        goto Cleanup;
    }

    err = wcscat_s(configFilePath, totalLen, L"\\");

    if(err != 0)
    {
        LOGINFO("Error while concatenating config file path with \\:%d\n", err);
        goto Cleanup;;
    }

    err = wcscat_s(configFilePath, totalLen, configFile);

    if(err != 0)
    {
        LOGINFO("Error while concatenating to config file path %d\n", err);
        goto Cleanup;
    }

    LOGINFOW(L"Config File path:%s\n", configFilePath);

    str = readConfigFile(configFilePath);

Cleanup:

    if(configDir)
    {
        delete[] configDir;
    }

    if(configFilePath)
    {
        delete[] configFilePath;
    }

    if(configFile)
    {
        delete[] configFile;
    }

    return str;
}

int HandlerConfig::ParseConfigFile(std::string str)
/*++

Routine Description:
    This function parses the config file and retrieves the plugins that needs to be started by netagent.

Arguments:
    str - config file contents in string

Return Value:
    returns 0 on success else -1
--*/
{
    std::wstringstream ss;

    ss << str.c_str();
    json::value val = json::value::parse(ss);

    for (auto iter = val.as_object().cbegin(); iter != val.as_object().cend(); ++iter)
    {
        const utility::string_t &key = iter->first;
        const json::value &val = iter->second;

        if (wcscmp(key.c_str(), L"runtimeSettings") != 0)
        {
            LOGINFOW(L"Expecting runtimeSettings but found %s", key.c_str());
            return -1;
        }

        readNetAgentConfig(val,this);

    }

    return 0;
}

void HandlerConfig::Cleanup()
/*++

Routine Description:
    This function cleans up the resources allocated by HandlerConfig Object

Arguments:

Return Value:

--*/
{
    while (!pluginsToStart.empty())
    {
        pluginsToStart.erase(pluginsToStart.begin());
    }
}