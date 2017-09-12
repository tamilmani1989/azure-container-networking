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

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

json::value GetPluginsToStart(json::value val)
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
    wchar_t szMessage[MAX_PATH];

	json::array arrval = val.as_array();
		
	for (auto arrIter = arrval.cbegin(); arrIter != arrval.cend(); ++arrIter)	
	{
		for(auto handlerIter = arrIter->as_object().cbegin(); handlerIter != arrIter->as_object().cend(); ++handlerIter)
		{
			const utility::string_t &handlerKey = handlerIter->first;
			const json::value &handlerValue = handlerIter->second;
            
            if (wcscmp(handlerKey.c_str(), L"handlerSettings")!=0)
            {
                StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Expecting handlerSettings but found %s", handlerKey.c_str());
                AquariusUtilities::WriteEventLogEntry(szMessage);
                return NULL;
            }

			for (auto publicIter = handlerValue.as_object().cbegin(); publicIter != handlerValue.as_object().cend(); ++publicIter)
			{
				const utility::string_t &publicKey = publicIter->first;
				const json::value &publicValue = publicIter->second;

                if (wcscmp(publicKey.c_str(), L"publicSettings")!=0)
                {
                    StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Expecting publicSettings but found %s", publicKey.c_str());
                    AquariusUtilities::WriteEventLogEntry(szMessage);
                    return NULL;
                }

				for(auto pluginIter = publicValue.as_object().cbegin(); pluginIter != publicValue.as_object().cend(); ++pluginIter)
				{
					const utility::string_t &pluginKey = pluginIter->first;
					const json::value &pluginValue = pluginIter->second;
                    
                    if (wcscmp(pluginKey.c_str(), L"PluginsToEnable")!=0)
                    {
                        StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Expecting PluginsToEnable but found %s", pluginKey.c_str());
                        AquariusUtilities::WriteEventLogEntry(szMessage);
                        return NULL;
                    }

                    LOGINFO("Got the key pluginstoenable");
					enablePluginsVal = pluginValue;
					break;
                }
				break;
			}
			break;
		}
		break;
	}

	return enablePluginsVal;
}

wchar_t* readConfigFolderField(json::value val)
/*++

Routine Description:
	This is helper function to find "configFolder" key from passed json and retrieves corresponding value

Arguments:
	jsonValue - json of handlerenvironment 

Return Value:
	returns config directory path on success else returns NULL

--*/
{
	wchar_t *configFolderPath=NULL;
	wchar_t *path = NULL;
    errno_t err;

	for (auto iter = val.as_object().cbegin(); iter != val.as_object().cend(); ++iter)
	{
		const utility::string_t &key = iter->first;
		const json::value &configVal = iter->second;

		if(wcscmp(L"configFolder", key.c_str()) == 0)
		{
			path = (wchar_t *)configVal.as_string().c_str();
			break;
		}
	}

	if(path==NULL)
	{
		return NULL;
	}

	size_t len = wcslen(path)+1;
	configFolderPath = new wchar_t[len];
    
    if (!configFolderPath)
    {
        LOGINFO("Error in Allocating ConfigFolderPath\n");
        return NULL;
    }

	err = wcscpy_s(configFolderPath, len, path);
	
    if(err != 0)
    {
        LOGINFO("Error while using wcscpy_s to copy config dir path %d\n", err);
        delete[] configFolderPath;
        return NULL;
    }

	return configFolderPath;
}

wchar_t* GetConfigFolderLocation(wchar_t *exeLocation)
/*++

Routine Description:
	This function retrieves config directory path from HandlerEnvironment.json

Arguments:
	exeLocation - Location of Aquarius Execuatable

Return Value:
	returns config directory path on success else returns NULL

--*/
{
	
	std::wstringstream ss;
	std::string str;
	wchar_t *configFolderPath=NULL;
    wchar_t *envJsonPath;
    size_t envFilePathLen = wcslen(exeLocation) + wcslen(ENV_JSON_FILE) + 2;

    envJsonPath = new wchar_t[envFilePathLen];

    if (!envJsonPath)
    {
        LOGINFO("Error in Allocating envJsonPath\n");
        return NULL;
    }

    if (swprintf_s(envJsonPath, envFilePathLen, L"%s\\%s", exeLocation, ENV_JSON_FILE) < 0)
    {
        LOGINFO("Swprinf_s failed while copying envjsonpath %d", GetLastError());
  		delete[] envJsonPath;
    	return NULL;
    }

	std::fstream fp(envJsonPath);
	if(!fp)
	{
		LOGINFOW(L"Not able to open file %s", envJsonPath);
  		delete[] envJsonPath;
    	return NULL;
	}

	fp.seekg(0, std::ios::end);  
	
	std::streamoff size = fp.tellg();
	
	if(size < 0)
	{
		LOGINFO("tellg() function failed in GetConfigFolferLocation");
  		delete[] envJsonPath;
    	return NULL;
	}

	str.reserve(size);
	fp.seekg(0, std::ios::beg);

	str.assign((std::istreambuf_iterator<char>(fp)),
				std::istreambuf_iterator<char>());

	LOGINFO("Handler Env file %s\n", str.c_str());

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
				configFolderPath = readConfigFolderField(handlerVal);
			}
		}
	}

    delete[] envJsonPath;
    return configFolderPath;
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
        LOGINFO("getBaseFileName returns NULL");
        return 0;
    }

	LOGINFOW(L"file1 %s file2 %s\n", file1.c_str(), file2.c_str());

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

wchar_t *GetLatestConfigFilePath(wchar_t *configDir)
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

	size_t len = wcslen(fdata.back().cFileName)+1;
	wchar_t *latestFile = new wchar_t[len];
	
	wcscpy_s(latestFile, len, fdata.back().cFileName);
	
	return latestFile;
}

std::string ReadConfigFile(wchar_t *configFilePath)
/*++

Routine Description:
	This function reads config file passed in parameter

Arguments:
	configFilePath - Path of config file

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

    return str;
}