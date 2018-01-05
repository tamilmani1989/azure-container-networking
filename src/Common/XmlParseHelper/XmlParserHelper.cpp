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

    if (bstr == NULL) {
        LOGERRW(L"VariantFromString allocation failed");
        hr = ERROR_NOT_ENOUGH_MEMORY;
        goto Cleanup;
    }

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
    if (hr != S_OK) {
        LOGERRW(L"CreateAndInitDOM:CoCreateInstance failed with error [%0x]", hr);
        goto Cleanup;
    }

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
    if (hr != S_OK) {
        LOGERRW(L"ReportParseError:get_parseError failed with error [%0x]", hr);
        goto Cleanup;
    }

    hr = pXMLErr->get_errorCode(&hrRet);
    if (hr != S_OK) {
        LOGERRW(L"ReportParseError:get_errorCode failed with error [%0x]", hr);
        goto Cleanup;
    }

    hr = pXMLErr->get_reason(&bstrReason);
    if (hr != S_OK) {
        LOGERRW(L"ReportParseError:get_reason failed with error [%0x] - Description - [%s]", hr, Description);
        goto Cleanup;
    }

	LOGERRW(L"ReportParseError error reason %s", bstrReason);

Cleanup:

    RELEASE(pXMLErr);

    if (bstrReason != NULL) {
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
    VARIANT_BOOL varStatus = VARIANT_FALSE;
	int totalTime = 0;

    VariantInit(&varFileName);
    hr = CreateAndInitDOM(&XMLDom);
    if (hr != S_OK) {
        LOGERRW(L"Intialize:CreateAndInitDOM failed with error [%0x]", hr);
        goto Cleanup;
    }

    hr = VariantFromString(XmlFileName, varFileName);
    if (hr != S_OK) {
        LOGERRW(L"Intialize:VariantFromString failed with error [%0x]", hr);
        goto Cleanup;
    }

	do
	{
		totalTime += 5000;

		hr = XMLDom->load(varFileName, &varStatus);
		if (hr != S_OK)
		{
			LOGERRW(L"XML Load failed with errorcode %x", hr);
		}

		if (hr != S_OK && varStatus != VARIANT_TRUE)
		{
			LOGINFOW(L"Call report parse error");
			hr = ReportParseError(XMLDom, XmlFileName);
			if (hr == HRESULT_FROM_WIN32(ERROR_SHARING_VIOLATION))
			{
				LOGINFOW(L"Retry after 5 seconds");
				Sleep(5000);
			}
		}
	}while (hr == HRESULT_FROM_WIN32(ERROR_SHARING_VIOLATION) && totalTime <= 60000);
	

Cleanup:
    if (FAILED(hr)) {
        RELEASE(XMLDom);
    }

    VariantClear(&varFileName);

    return hr;
}

HRESULT
XmlParserHelper::QueryNodeValue(
    __in LPCWSTR NodePathQuery,
    __inout std::wstring& Value
)
/*++

Routine Description:

    Helper to query value associated with a node.

Arguments:

    NodePathQuery - Node path query in Xpath format.

    Value - Retrieved value associated with the node.

Return Value:

    S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
    HRESULT hr = S_OK;
    IXMLDOMNode *pNode = NULL;
    BSTR nodeQuery = NULL;
    BSTR bstrNodeValue = NULL;

    LOGINFOW(L"NodeQuery:%s.", NodePathQuery);

    nodeQuery = SysAllocString(NodePathQuery);
    if (nodeQuery == NULL) {
        hr = ERROR_NOT_ENOUGH_MEMORY;
        LOGERRW(L"QueryNodeAttributeValues:SysAllocString Allocation failed for node query.");
        goto Cleanup;
    }

    hr = XMLDom->selectSingleNode(nodeQuery, &pNode);
    if (hr != S_OK) {
        LOGERRW(L"QueryNodeAttributeValues:selectNodes failed with error [%0x]", hr);
        goto Cleanup;
    }
    pNode->get_text(&bstrNodeValue);
    Value.assign(bstrNodeValue, SysStringLen(bstrNodeValue));
    wprintf(L"Attribute value, <%s>\n", Value.c_str());

Cleanup:
    RELEASE(pNode);

    if (nodeQuery != NULL) {
        SysFreeString(nodeQuery);
    }

    if (bstrNodeValue != NULL) {
        SysFreeString(bstrNodeValue);
    }
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
    IXMLDOMNodeList *pNodes = NULL;
    IXMLDOMNode *pNode = NULL;
    IXMLDOMNamedNodeMap *pIXMLDOMNamedNodeMap;
    IXMLDOMNode *pIXMLDOMNode;
    int index;
    long length;
    BSTR nodeQuery = NULL;
    BSTR bstrNodeName = NULL;
    BSTR bstrAttributeName = NULL;
    VARIANT variantAttributeValue;
    BSTR bstrAttributeValue;
    std::wstring attributeValue;

    LOGINFOW(L"NodeQuery:%s, Attribute:%s.", NodePathQuery, AttributeName);

    nodeQuery = SysAllocString(NodePathQuery);
    if (nodeQuery == NULL) {
        hr = ERROR_NOT_ENOUGH_MEMORY;
        LOGERRW(L"QueryNodeAttributeValues:SysAllocString Allocation failed for node query.");
        goto Cleanup;
    }

    bstrAttributeName = SysAllocString(AttributeName);
    if (bstrAttributeName == NULL) {
        LOGERRW(L"QueryNodeAttributeValues:SysAllocString Allocation failed for attribute query.");
        goto Cleanup;
    }


    hr = XMLDom->selectNodes(nodeQuery, &pNodes);
    if (hr != S_OK) {
        LOGERRW(L"QueryNodeAttributeValues:selectNodes failed with error [%0x]", hr);
        goto Cleanup;
    }

    if(pNodes != NULL) {
        hr = pNodes->get_length(&length);
        if (hr != S_OK) {
            LOGERRW(L"QueryNodeAttributeValues:selectNodes failed with error [%0x]", hr);
            goto Cleanup;
        }

        for (index = 0; index < length; index++) {
            hr = pNodes->get_item(index, &pNode);
            if (hr != S_OK) {
                LOGERRW(L"QueryNodeAttributeValues:get_item failed with error [%0x]", hr);
                goto Cleanup;
            }

            hr = pNode->get_nodeName(&bstrNodeName);
            if (hr != S_OK) {
                LOGERRW(L"QueryNodeAttributeValues:get_nodeName failed with error [%0x]", hr);
                goto Cleanup;
            }

            SysFreeString(bstrNodeName);
            bstrNodeName = NULL;
            pIXMLDOMNamedNodeMap = NULL;
            hr = pNode->get_attributes(&pIXMLDOMNamedNodeMap);
            if (SUCCEEDED(hr) && pIXMLDOMNamedNodeMap) {
                hr = pIXMLDOMNamedNodeMap->getNamedItem(bstrAttributeName, &pIXMLDOMNode);
                if (SUCCEEDED(hr) && pIXMLDOMNode) {
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
    } else {
        hr = ReportParseError(XMLDom, L"Error while calling selectNodes.");
    }

Cleanup:

    RELEASE(pNodes);
    RELEASE(pNode);
    if (nodeQuery != NULL) {
        SysFreeString(nodeQuery);
    }

    if (bstrNodeName != NULL) {
        SysFreeString(bstrNodeName);
    }

    if (bstrAttributeName != NULL) {
        SysFreeString(bstrAttributeName);
    }

    if (AttributeValues.size() > 0) {
        LOGINFOW(L"Got atleast one attribute value. So considering it as success");
        hr = S_OK;
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

HRESULT 
XmlParserHelper::CheckNodeExists(
    __in LPCWSTR NodePathQuery
)
/*++

Routine Description:

This function checks if given node exists in XML

Arguments:

NodePathQuery - XML Path of the node.

Return Value:

returns S_OK if node is found else returns error
--*/
{
    HRESULT hr = S_OK;
    IXMLDOMNode *pNode = NULL;

    BSTR nodeQuery = NULL;
    
    nodeQuery = SysAllocString(NodePathQuery);
    if(nodeQuery == NULL)
    {
        LOGERRW(L"QueryNodeAttributeValues:SysAllocString Allocation failed for CheckNodeExists.");
        goto Cleanup; 
    }

    hr = XMLDom->selectSingleNode(nodeQuery, &pNode);
    
Cleanup:
    RELEASE(pNode);
    if (nodeQuery != NULL)
    {
        SysFreeString(nodeQuery);
    }
    return hr;
}