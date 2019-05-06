
/*++

Copyright (c) Microsoft Corporation

Module Name:

ConfigParser.h

Abstract:

This contains method related to config file parser and will be used by Aquarius.

Author:

Tamilmani Manoharan (tamanoha) 06-Sep-2017

--*/

#pragma once
#include<sstream>
#include<iostream>
#include <windows.h>
#include <vector>
#include <errno.h>
#include <set>
#include<sstream>

using namespace std;


#define CONFIG_EXTENSION L"\\*.settings"

class HandlerConfig
{
    public:
    std::set<std::wstring> pluginsToStart;
    std::wstring restServerPort;
    bool disableRestServer = false;
    std::wstring latestSequenceNumber;

    wchar_t *getLatestConfigFilePath(wchar_t *configDir);
    std::string readConfigFile(wchar_t *configFilePath);

    int ParseConfigFile(std::string str);
    std::string ReadLatestConfigFile(wchar_t *configFolder);
    void Cleanup();
};
