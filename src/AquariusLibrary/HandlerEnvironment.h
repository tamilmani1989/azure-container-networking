/*++

Copyright (c) Microsoft Corporation

Module Name:

HandlerEnvironment.h

Abstract:

This contains method related to parsing HandlerEnvironment json file and will be used by Aquarius.

Author:

Tamilmani Manoharan (tamanoha) 26-Sep-2017

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

#define  ENV_JSON_FILE L"..\\HandlerEnvironment.json"

class HandlerEnvironment
{
    public:

    wchar_t *logFolderPath=nullptr;
    wchar_t *configFolderPath=nullptr;

    int ReadHandlerEnvironmentFile(wchar_t *exeLocation);
    void Cleanup();
};