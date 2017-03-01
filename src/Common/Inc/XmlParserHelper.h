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
#include <iostream>
#include "RdCommon.h"
#include "RdTrace.h"

using namespace std;

class XmlParserHelper
{
    public:
        XmlParserHelper();
        HRESULT Initialize(__in LPCWSTR XmlFile);
        HRESULT QueryNodeAttributeValues(
            __in LPWSTR NodeName,
            __in LPWSTR AttrtbuteName,
            __inout std::set<wstring>& AttributeValues);
        void UnInitialize();

    private:
        IXMLDOMDocument* XMLDom;

        HRESULT VariantFromString(__in PCWSTR Value, __out VARIANT &Variant);
        HRESULT CreateAndInitDOM(__inout IXMLDOMDocument **Document);
        HRESULT ReportParseError(__in IXMLDOMDocument *Document, __in LPCWSTR Description);
};