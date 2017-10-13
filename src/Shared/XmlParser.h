/****************************** Module Header ******************************\
* Module Name:  XmlParser.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
*
\***************************************************************************/

#pragma once

#include <stdio.h>
#include <tchar.h>
#include <msxml6.h>
#include "PluginMetadata.h"
class XmlParser
{
public:
	static int ParseManifest(wchar_t* xmlFilePath, PluginMetadata *pluginData);
};