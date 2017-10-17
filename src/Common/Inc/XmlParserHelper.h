/*++

Copyright (c) Microsoft Corporation

Module Name:

    XmlParserHelper.h

Abstract:

    Class definition for XmlParserHelper.

Environment:

    User-mode

--*/
#pragma once


#ifndef ASSERT
#define ASSERT( exp ) ((void) 0)
#endif
#include <msxml6.h>

#include <stdio.h>
#include <wchar.h>
#include <windows.h>
#include <regex>
#include <set>
#include <map>
#include <iostream>


#include "Logging.h"
using namespace std;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

#define XPATH_ROOT_START_TAG L"//"
#define XPATH_PATH_SEPARATOR_TAG L"/"

class XmlParserHelper
{
    public:

        XmlParserHelper();

        HRESULT Initialize(__in LPCWSTR XmlFile);

        HRESULT QueryNodeAttributeValues(__in LPCWSTR NodeName, __in LPCWSTR AttrtbuteName, __inout std::set<wstring>& AttributeValues);

        void UnInitialize();

        std::wstring PrepareAttributeQuery(std::wstring Name, std::wstring Value);

		HRESULT QueryNodeValue(LPCWSTR NodePathQuery, std::wstring & Value);
    private:

        IXMLDOMDocument* XMLDom;

        HRESULT VariantFromString(__in PCWSTR Value, __out VARIANT &Variant);

        HRESULT CreateAndInitDOM(__inout IXMLDOMDocument **Document);

        HRESULT ReportParseError(__in IXMLDOMDocument *Document, __in LPCWSTR Description);
};