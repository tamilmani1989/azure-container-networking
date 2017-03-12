/*++

Copyright (c) Microsoft Corporation

Module Name:

    XmlParserHelper.h

Abstract:

    Implements helpers to support for parsing of Xml documents.

Environment:

    User-mode

--*/

#include "XmlParserHelper.h"

XmlParserHelper::XmlParserHelper(
    )
/*++

Routine Description:

    Destructor for XmlParserHelper

Arguments:

    None.
    
Return Value:

    None.

--*/
{
    XMLDom = NULL;
}

void 
XmlParserHelper::UnInitialize(
    )
/*++

Routine Description:

    Uninitialize the XML parser helper.
    Frees up resources allocated to the XML Parser helper.

Arguments:

    None.
    
Return Value:

    None.

--*/
{
    RELEASE(XMLDom);
}


HRESULT 
XmlParserHelper::VariantFromString(
    __in PCWSTR Value, 
    __out VARIANT &Variant)
/*++

Routine Description:

    Helper function to create a VT_BSTR variant from a null terminated string. 

Arguments:

    Value - Null terminated string.

    Varaint - Out varaibale for the created  VT_BSTR variant.
    
Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    HRESULT hr = S_OK;
    BSTR bstr = SysAllocString(Value);
    RD_TRACE_M_EXIT(bstr, (RD_TRACE_ERROR, L"VariantFromString allocation failed"), Cleanup);
    
    V_VT(&Variant)   = VT_BSTR;
    V_BSTR(&Variant) = bstr;

Cleanup:
    return hr;
}

//  
HRESULT 
XmlParserHelper::CreateAndInitDOM(
    __inout IXMLDOMDocument** Document)
/*++

Routine Description:

    Helper function to create a DOM instance.

Arguments:

    Document - Out parameter to hold the reference to the create DOM instance.
    
Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    HRESULT hr = CoCreateInstance(__uuidof(DOMDocument60), NULL, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(Document));
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"CreateAndInitDOM:CoCreateInstance failed with error [%0x]", hr),
        hr,
        Cleanup);

    // These methods should not fail so don't inspect result
    (*Document)->put_async(VARIANT_FALSE);
    (*Document)->put_validateOnParse(VARIANT_FALSE);
    (*Document)->put_resolveExternals(VARIANT_FALSE);
    (*Document)->put_preserveWhiteSpace(VARIANT_TRUE);

Cleanup:
    return hr;
}


HRESULT 
XmlParserHelper::ReportParseError(
    IXMLDOMDocument *Document, 
    LPCWSTR Description)
/*++

Routine Description:

    Helper function to  display parse error.

Arguments:

    Document - Reference to the DOM instance.

    Description - Additional parameter used for logging.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
HRESULT hr = S_OK;
HRESULT hrRet = E_FAIL; // Default error code if failed to get from parse error.
IXMLDOMParseError *pXMLErr = NULL;
BSTR bstrReason = NULL;

hr = Document->get_parseError(&pXMLErr);
RD_TRACE_E_HR((RD_TRACE_ERROR,
    L"ReportParseError:get_parseError failed with error [%0x]", hr),
    hr,
    Cleanup);
hr = pXMLErr->get_errorCode(&hrRet);
RD_TRACE_E_HR((RD_TRACE_ERROR,
    L"ReportParseError:get_errorCode failed with error [%0x]", hr),
    hr,
    Cleanup);

hr = pXMLErr->get_reason(&bstrReason);
RD_TRACE_E_HR((RD_TRACE_ERROR,
    L"ReportParseError:get_reason failed with error [%0x] - Description - [%s]", hr, Description),
    hr,
    Cleanup);

Cleanup:

RELEASE(pXMLErr);

if (bstrReason != NULL)
{
    SysFreeString(bstrReason);
}

return hrRet;
}

HRESULT
XmlParserHelper::Initialize(
    LPCWSTR XmlFileName
)
/*++

Routine Description:

    Helper to create an associate a Dom instance and associate to an Xml file that will be parsed.

Arguments:

    XmlFileName - Full path to the XML file to be parsed.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    HRESULT hr = S_OK;
    VARIANT varFileName;
    VARIANT_BOOL varStatus;
    VariantInit(&varFileName);
    hr = CreateAndInitDOM(&XMLDom);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"Intialize:CreateAndInitDOM failed with error [%0x]", hr),
        hr,
        Cleanup);

    hr = VariantFromString(XmlFileName, varFileName);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"Intialize:VariantFromString failed with error [%0x]", hr),
        hr,
        Cleanup);

    hr = XMLDom->load(varFileName, &varStatus);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"Intialize:Load failed with error [%0x]", hr),
        hr,
        Cleanup);
    if (varStatus != VARIANT_TRUE)
    {
        hr = ReportParseError(XMLDom, XmlFileName);
    }

Cleanup:
    if (FAILED(hr))
    {
        RELEASE(XMLDom);
    }

    VariantClear(&varFileName);

    return hr;
}

HRESULT 
XmlParserHelper::QueryNodeAttributeValues(
    __in LPCWSTR NodePathQuery,
    __in LPCWSTR AttributeName,
    __inout std::set<wstring>& AttributeValues
    )
/*++

Routine Description:

    Helper to query the values associated with a given node and an attribute.

Arguments:

    NodePathQuery - Node path query in Xpath format.

    AttirbuteName = Name of the attribute to be queried

    AttributeValues - Set to hold the list of values associated with the attribute.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    HRESULT hr = S_OK;
    DWORD error;
    IXMLDOMNodeList *pNodes = NULL;
    IXMLDOMNode *pNode = NULL;
    IXMLDOMNamedNodeMap *pIXMLDOMNamedNodeMap;
    IXMLDOMNode *pIXMLDOMNode;
    int index;
    long length;
    BSTR nodeQuery = NULL;
    BSTR bstrNodeName = NULL;
    BSTR bstrNodeValue = NULL;
    BSTR bstrAttributeName = NULL;
    VARIANT variantAttributeValue;
    BSTR bstrAttributeValue;
    VARIANT_BOOL varStatus;
    VARIANT varFileName;
    VariantInit(&varFileName);
    std::wstring attributeValue;
    RD_TRACE_LOG((RD_TRACE_INFO, L"NodeQuery:%s, Attribute:%s.", NodePathQuery, AttributeName));

    nodeQuery = SysAllocString(NodePathQuery);
    RD_TRACE_M_EXIT(nodeQuery, 
        (RD_TRACE_ERROR, 
        L"QueryNodeAttributeValues:SysAllocString Allocation failed for node query."), 
        Cleanup);

    bstrAttributeName = SysAllocString(AttributeName);
    RD_TRACE_M_EXIT(bstrAttributeName, 
        (RD_TRACE_ERROR, 
        L"QueryNodeAttributeValues:SysAllocString Allocation failed for attribute query."), 
        Cleanup);

    hr = XMLDom->selectNodes(nodeQuery, &pNodes);
    RD_TRACE_E_HR((RD_TRACE_ERROR,
        L"QueryNodeAttributeValues:selectNodes failed with error [%0x]", hr),
        hr,
        Cleanup);

    if(pNodes != NULL)
    {
        hr = pNodes->get_length(&length);
        RD_TRACE_E_HR((RD_TRACE_ERROR,
            L"QueryNodeAttributeValues:selectNodes failed with error [%0x]", hr),
            hr,
            Cleanup);

        for (index = 0; index < length; index++)
        {
            hr = pNodes->get_item(index, &pNode);
            RD_TRACE_E_HR((RD_TRACE_ERROR,
                L"QueryNodeAttributeValues:get_item failed with error [%0x]", hr),
                hr,
                Cleanup);

            hr = pNode->get_nodeName(&bstrNodeName);
            RD_TRACE_E_HR((RD_TRACE_ERROR,
                L"QueryNodeAttributeValues:get_nodeName failed with error [%0x]", hr),
                hr,
                Cleanup);

            SysFreeString(bstrNodeName);
            bstrNodeName = NULL;
            pIXMLDOMNamedNodeMap = NULL;
            hr = pNode->get_attributes(&pIXMLDOMNamedNodeMap);
            if (SUCCEEDED(hr) && pIXMLDOMNamedNodeMap)
            {
                hr = pIXMLDOMNamedNodeMap->getNamedItem(bstrAttributeName, &pIXMLDOMNode);
                if (SUCCEEDED(hr) && pIXMLDOMNode)
                {
                    pIXMLDOMNode->get_nodeValue(&variantAttributeValue);
                    bstrAttributeValue = V_BSTR(&variantAttributeValue);
                    attributeValue.clear();
                    attributeValue.assign(bstrAttributeValue, SysStringLen(bstrAttributeValue));
                    wprintf(L"Attribute value, <%s>\n", attributeValue.c_str());
                    AttributeValues.insert(attributeValue);
                    RELEASE(pIXMLDOMNode);
                }
                RELEASE(pIXMLDOMNamedNodeMap);
            }

            RELEASE(pNode);
        }
    }
    else
    {
        hr = ReportParseError(XMLDom, L"Error while calling selectNodes.");
    }

Cleanup:

    RELEASE(pNodes);
    RELEASE(pNode);
    if (nodeQuery != NULL)
    {
        SysFreeString(nodeQuery);
    }

    if (bstrNodeName != NULL)
    {
        SysFreeString(bstrNodeName);
    }

    if (bstrAttributeName != NULL)
    {
        SysFreeString(bstrAttributeName);
    }

    return hr;
}

std::wstring 
XmlParserHelper::PrepareAttributeQuery(
    std::wstring Name,
    std::wstring Value)
    /*++

    Routine Description:

    Helper to prepare an attribute query.

    Arguments:

    Name - Name of the attribute.

    Value - Value of the attribute.

    Return Value:

    string representing the attribute query in the format to be used in Xpath queries.

    --*/
{
    std::wstring query;

    query.append(L"[@");
    query.append(Name);
    query.append(L"='");
    query.append(Value);
    query.append(L"']");

    return query;
}