/*++

Copyright (c) Microsoft Corporation

Module Name:

XmlWriterHelper.h

Abstract:

Header contains declaration of methods write xml object

Author:

Tamilmani Manoharan (tamanoha) 10-October-2017

--*/

#include <stdio.h>
#include <tchar.h>
#include <msxml6.h>

// Macro that calls a COM method returning HRESULT value.
#define CHK_HR(stmt)        do { hr=(stmt); if (FAILED(hr)) goto CleanUp; } while(0)

// Macro to verify memory allcation.
#define CHK_ALLOC(p)        do { if (!(p)) { hr = E_OUTOFMEMORY; goto CleanUp; } } while(0)

// Macro that releases a COM object if not NULL.
#define SAFE_RELEASE(p)     do { if ((p)) { (p)->Release(); (p) = NULL; } } while(0)


class XmlWriterHelper
{

    IXMLDOMDocument *pDom = NULL;
    HRESULT VariantFromString(PCWSTR wszValue, VARIANT &Variant);
    
public:

    HRESULT Initialize();
    void UnInitialize();
    HRESULT CreateElement(PCWSTR wszName, IXMLDOMElement **ppElement);
    HRESULT AppendChildToParent(IXMLDOMNode *pChild, IXMLDOMNode *pParent);
    HRESULT CreateAndAddAttributeNode(PCWSTR wszName, PCWSTR wszValue, IXMLDOMElement *pParent);
    HRESULT CreateAndAddPINode(PCWSTR wszTarget, PCWSTR wszData);
    HRESULT CreateAndAddTextNode(PCWSTR wszText, IXMLDOMNode *pParent);
    HRESULT CreateAndAddElementNode(PCWSTR wszName, PCWSTR wszNewline, IXMLDOMNode *pParent, IXMLDOMElement **ppElement);
    HRESULT SaveDOM(VARIANT destination);
    HRESULT GetXMLInString(BSTR *str);
    
};
