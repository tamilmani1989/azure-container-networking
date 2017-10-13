/*++

Copyright (c) Microsoft Corporation

Module Name:

HandlerEnvironment.cpp

Abstract:

This file contains methods to read config folder and log folder from handler environment json file.

Author:

Tamilmani Manoharan (tamanoha) 26-September-2017

--*/

#include "../Shared/Utilities.h"
#include "../Logging/Logging.h"
#include "HandlerEnvironment.h"
#include <cpprest/json.h>

using namespace web;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

int readHandlerEnvFields(json::value val, HandlerEnvironment *envObj)
/*++

Routine Description:
	This helper function is to retrieve necessary fields from the json object

Arguments:
	jsonValue - json of handlerenvironment 

Return Value:
	returns 0 on success else returns -1

--*/
{

	wchar_t *configPath = NULL, *logPath = NULL;
	wchar_t logDirPath[MAX_PATH];
	wchar_t configDirPath[MAX_PATH];

    errno_t err = 0;
	
	LOGINFOW(L" Inside readHandlerEnvFields\n");
	
	for (auto iter = val.as_object().cbegin(); iter != val.as_object().cend(); ++iter)
	{
		const utility::string_t &key = iter->first;
		const json::value &configVal = iter->second;

		if(wcscmp(L"configFolder", key.c_str()) == 0)
		{
			configPath = (wchar_t *)configVal.as_string().c_str();
		}

		else if(wcscmp(L"logFolder", key.c_str()) == 0)
		{
			logPath = (wchar_t *)configVal.as_string().c_str();
		}
	}
	
	LOGINFOW(L" Retrieved config Folder Path: %s Log Folder :%s", configPath, logPath);

	if(configPath==NULL || logPath == NULL)
	{
		return -1;
	}

	if(wcslen(configPath)==0)
	{
		AquariusUtilities::PopulateServiceExeLocation(configDirPath);
		if(wcscat_s(configDirPath, MAX_PATH, L"\\RuntimeSettings") != 0)
		{
			LOGERRW(L"wcscat failed for constructing config folder:%s", configDirPath);
			return -1;
		}			

		configPath = configDirPath;
	}

	if(wcslen(logPath)==0)
	{
		AquariusUtilities::PopulateServiceExeLocation(logDirPath);
		
		if(wcscat_s(logDirPath, MAX_PATH, L"\\Logs") != 0)
		{
			LOGERRW(L"wcscat failed for constructing log folder:%s", logDirPath);
			return -1;
		}	

		if (!CreateDirectory(logDirPath, NULL))
		{
			err = GetLastError();
			if (err != ERROR_ALREADY_EXISTS)
			{
				LOGERRW(L"Failed to create log folder [%s] due to erroro (%d)\n", logDirPath, err);
				return -1;
			}
		}

		logPath = logDirPath;
	}

	size_t len = wcslen(configPath)+1;
	envObj->configFolderPath = new wchar_t[len];
    
    if (!envObj->configFolderPath)
    {
        LOGINFO("Error in Allocating ConfigFolderPath\n");
        return -1;
    }

	err = wcscpy_s(envObj->configFolderPath, len, configPath);
	
    if(err != 0)
    {
        LOGINFO("Error while using wcscpy_s to copy config dir path %d\n", err);
        delete[] envObj->configFolderPath;
		envObj->configFolderPath = nullptr;
        return -1;
    }

	len = wcslen(logPath)+1;
	envObj->logFolderPath = new wchar_t[len];
    
    if (!envObj->logFolderPath)
    {
        LOGINFO("Error in Allocating ConfigFolderPath\n");
        return -1;
    }

	err = wcscpy_s(envObj->logFolderPath, len, logPath);
	
    if(err != 0)
    {
        LOGINFO("Error while using wcscpy_s to copy config dir path %d\n", err);
        delete[] envObj->logFolderPath;
		envObj->logFolderPath = nullptr;
        return -1;
    }

	return 0;
}

int HandlerEnvironment::ReadHandlerEnvironmentFile(wchar_t * exeLocation)
/*++

Routine Description:
	This function retrieves config and log folder location from HandlerEnvironment.json

Arguments:
	exeLocation - Location of Aquarius Execuatable

Return Value:
	returns 0 on success else returns -1

--*/
{
	std::wstringstream ss;
	std::string str;
    wchar_t *envJsonPath;
    size_t envFilePathLen = wcslen(exeLocation) + wcslen(ENV_JSON_FILE) + 2;

    envJsonPath = new wchar_t[envFilePathLen];

    if (!envJsonPath)
    {
        LOGINFO("Error in Allocating envJsonPath\n");
        return -1;
    }

    if (swprintf_s(envJsonPath, envFilePathLen, L"%s\\%s", exeLocation, ENV_JSON_FILE) < 0)
    {
        LOGINFO("Swprinf_s failed while copying envjsonpath %d", GetLastError());
  		delete[] envJsonPath;
    	return -1;
    }

	std::fstream fp(envJsonPath);
	if(!fp)
	{
		LOGINFOW(L"Not able to open file %s", envJsonPath);
  		delete[] envJsonPath;
    	return -1;
	}

	fp.seekg(0, std::ios::end);  
	
	std::streamoff size = fp.tellg();
	
	if(size < 0)
	{
		LOGINFO("tellg() function failed in GetConfigFolferLocation");
  		delete[] envJsonPath;
    	return -1;
	}

	str.reserve(size);
	fp.seekg(0, std::ios::beg);

	str.assign((std::istreambuf_iterator<char>(fp)),
				std::istreambuf_iterator<char>());

	LOGINFO("Handler Env file %s\n", str.c_str());

    delete[] envJsonPath;

	ss << str.c_str();
	json::value val = json::value::parse(ss);
	json::array arrval = val.as_array();

	for (auto arrIter = arrval.cbegin(); arrIter != arrval.cend(); ++arrIter)	
	{
		for (auto iter = arrIter->as_object().cbegin(); iter != arrIter->as_object().cend(); ++iter)
		{
			const utility::string_t &key = iter->first;
			LOGINFOW(L"key:%s\n",key.c_str());
			
			if(wcscmp(L"handlerEnvironment", key.c_str()) == 0)
			{
				const json::value &handlerVal = iter->second;
				return readHandlerEnvFields(handlerVal, this);
			}
		}
	}
	return -1;
}

void HandlerEnvironment::Cleanup()
/*++

Routine Description:
	This function cleans up the resources allocated by handler environment object

Arguments:

Return Value:

--*/
{
	if(logFolderPath)
	{
		delete[] logFolderPath;
		logFolderPath = NULL;
	}

	if(configFolderPath)
	{
		delete[] configFolderPath;
		configFolderPath = NULL;
	}
}