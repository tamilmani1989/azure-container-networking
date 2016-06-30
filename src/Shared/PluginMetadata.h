/****************************** Module Header ******************************\
* Module Name:  PluginMetadata.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
\***************************************************************************/

#pragma once

#include <iostream>

class PluginMetadata
{
public:
	// This is the version that should be running in the VM
	wchar_t *currentVersion = nullptr;
	wchar_t *location = nullptr;
};