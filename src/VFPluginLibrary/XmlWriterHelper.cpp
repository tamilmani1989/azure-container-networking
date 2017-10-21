/*++

Copyright (c) Microsoft Corporation

Module Name:

XmlWriterHelper.cpp

Abstract:

This file contains helper methods to support constructing xml object

Author:

Tamilmani Manoharan (tamanoha) 10-October-2017

--*/

#include "XmlWriterHelper.h"

HRESULT XmlWriterHelper::VariantFromString(PCWSTR wszValue, VARIANT &Variant)
{
    HRESULT hr = S_OK;
    BSTR bstr = SysAllocString(wszValue);
    CHK_ALLOC(bstr);
    
    V_VT(&Variant)   = VT_BSTR;
    V_BSTR(&Variant) = bstr;

CleanUp:
    return hr;
}

HRESULT XmlWriterHelper::Initialize()
{
    HRESULT hr = CoCreateInstance(__uuidof(DOMDocument60), NULL, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&pDom));
    if (SUCCEEDED(hr))
    {
        // these methods should not fail so don't inspect result
        (pDom)->put_async(VARIANT_FALSE);  
        (pDom)->put_validateOnParse(VARIANT_FALSE);
        (pDom)->put_resolveExternals(VARIANT_FALSE);
        (pDom)->put_preserveWhiteSpace(VARIANT_TRUE);
    }
    return hr;
}

void XmlWriterHelper::UnInitialize()
{
    SAFE_RELEASE(pDom);
}

// Helper function to append a child to a parent node.
HRESULT XmlWriterHelper::AppendChildToParent(IXMLDOMNode *pChild, IXMLDOMNode *pParent)
{
    HRESULT hr = S_OK;
    IXMLDOMNode *pChildOut = NULL;
    
    if(!pParent)
    {
        pParent = pDom;
    }

    CHK_HR(pParent->appendChild(pChild, &pChildOut));

CleanUp:
    SAFE_RELEASE(pChildOut);
    return hr;
}

HRESULT XmlWriterHelper::CreateAndAddAttributeNode(PCWSTR wszName, PCWSTR wszValue, IXMLDOMElement *pParent)
{
    HRESULT hr = S_OK;
    IXMLDOMAttribute *pAttribute = NULL;
    IXMLDOMAttribute *pAttributeOut = NULL; // Out param that is not used

    BSTR bstrName = NULL;
    VARIANT varValue;
    VariantInit(&varValue);

    bstrName = SysAllocString(wszName);
    CHK_ALLOC(bstrName);
    CHK_HR(VariantFromString(wszValue, varValue));

    CHK_HR(pDom->createAttribute(bstrName, &pAttribute));
    CHK_HR(pAttribute->put_value(varValue));
    CHK_HR(pParent->setAttributeNode(pAttribute, &pAttributeOut));

CleanUp:
    SAFE_RELEASE(pAttribute);
    SAFE_RELEASE(pAttributeOut);
    SysFreeString(bstrName);
    VariantClear(&varValue);
    return hr;
}

HRESULT XmlWriterHelper::CreateAndAddPINode(PCWSTR wszTarget, PCWSTR wszData)
{
    HRESULT hr = S_OK;
    IXMLDOMProcessingInstruction *pPI = NULL;

    BSTR bstrTarget = SysAllocString(wszTarget);
    BSTR bstrData = SysAllocString(wszData);
    CHK_ALLOC(bstrTarget && bstrData);
    
    CHK_HR(pDom->createProcessingInstruction(bstrTarget, bstrData, &pPI));
    CHK_HR(AppendChildToParent(pPI, pDom));

CleanUp:
    SAFE_RELEASE(pPI);
    SysFreeString(bstrTarget);
    SysFreeString(bstrData);
    return hr;
}

HRESULT XmlWriterHelper::CreateElement(PCWSTR wszName, IXMLDOMElement **ppElement)
{
    HRESULT hr = S_OK;
    *ppElement = NULL;

    BSTR bstrName = SysAllocString(wszName);
    CHK_ALLOC(bstrName);
    CHK_HR(pDom->createElement(bstrName, ppElement));

CleanUp:
    SysFreeString(bstrName);
    return hr;
}

// Helper function to create and append a text node to a parent node.
HRESULT XmlWriterHelper::CreateAndAddTextNode(PCWSTR wszText, IXMLDOMNode *pParent)
{
    HRESULT hr = S_OK;    
    IXMLDOMText *pText = NULL;

    BSTR bstrText = SysAllocString(wszText);
    CHK_ALLOC(bstrText);

    CHK_HR(pDom->createTextNode(bstrText, &pText));
    CHK_HR(AppendChildToParent(pText, pParent));

CleanUp:
    SAFE_RELEASE(pText);
    SysFreeString(bstrText);
    return hr;
}

HRESULT XmlWriterHelper::CreateAndAddElementNode(PCWSTR wszName, PCWSTR wszNewline, IXMLDOMNode *pParent, IXMLDOMElement **ppElement = NULL)
{
    HRESULT hr = S_OK;
    IXMLDOMElement* pElement = NULL;

    CHK_HR(CreateElement(wszName, &pElement));
    // Add NEWLINE+TAB for identation before this element.
    CHK_HR(CreateAndAddTextNode(wszNewline, pParent));
    // Append this element to parent.
    CHK_HR(AppendChildToParent(pElement, pParent));


CleanUp:
    if (ppElement)
        *ppElement = pElement;  // Caller is repsonsible to release this element.
    else
        SAFE_RELEASE(pElement); // Caller is not interested on this element, so release it.

    return hr;
}

HRESULT  XmlWriterHelper::SaveDOM(VARIANT destination)
{
    return pDom->save(destination);
}

HRESULT XmlWriterHelper::GetXMLInString(BSTR *xmlString)
{
    return pDom->get_xml(xmlString);
}