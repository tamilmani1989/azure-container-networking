
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
#include <cpprest/json.h>

using namespace std;
using namespace web;

#define  ENV_JSON_FILE L"..\\HandlerEnvironment.json"
#define CONFIG_EXTENSION L"\\*.settings"

json::value GetPluginsToStart(json::value val);
wchar_t* readConfigFolderField(json::value val);
wchar_t* GetConfigFolderLocation(wchar_t *exeLocation);
std::wstring getBaseFilename(const wchar_t *filename);
wchar_t *GetLatestConfigFilePath(wchar_t *configDir);
std::string ReadConfigFile(wchar_t *configFilePath);

