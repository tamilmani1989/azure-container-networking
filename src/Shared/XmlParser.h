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

class XmlParser
{
public:
	static int ParseManifest(wchar_t* xmlFilePath, wchar_t **entryPoint);
};