/****************************** Module Header ******************************\
* Module Name:  PluginMetadata.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#pragma once


#include <iostream>
#include <algorithm> 

typedef void * HANDLE;
#define MAX_RETRY_COUNT 3

enum ProcessState 
{
    NOT_STARTED=0,
    RUNNING,
    STOPPED
};

class PluginMetadata
{
    public:
    // This is the version that should be running in the VM
    wchar_t *currentVersion = nullptr;
    wchar_t *location = nullptr;
    wchar_t *entryPoint = nullptr;
    wchar_t *name = nullptr;
    wchar_t *arguments = nullptr;
    int numTimesStarted = 0;
    int exitStatus = 0;
    int restartOnExit = 1;
    int blockRole = 0;
    int retryCount = 0;
    bool cleanup =false;
    ProcessState processState = NOT_STARTED; 
    HANDLE processHandle = NULL;
    HANDLE jobObjectHandle = NULL;
};

PluginMetadata* GetPluginInstances(int count);

