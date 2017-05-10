/*++

Copyright (c) Microsoft Corporation

Module Name:

ResponseParser.cpp

Abstract:

This cotains the implementation for methods to parse xml in windows.

Author:

Sushant Sharma (susharm) 24-March-2017

--*/


#ifdef _WIN32
#include <msxml6.h>
#include <stdio.h>
#include "atlbase.h"
#include "comutil.h"
#include <ws2tcpip.h>
#include "../../Logging/Logging.h"
#include "../Common/ResponseParser.h"
#define DllExport   __declspec( dllexport )  


namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
	
// Macro that calls a COM method returning HRESULT value.
#define CHK_HR(stmt)        do { hr=(stmt); if (FAILED(hr)) goto CleanUp; } while(0)

// Macro to verify memory allcation.
#define CHK_ALLOC(p)        do { if (!(p)) { hr = E_OUTOFMEMORY; goto CleanUp; } } while(0)

// Macro that releases a COM object if not NULL.
#define SAFE_RELEASE(p)     do { if ((p)) { (p)->Release(); (p) = nullptr; } } while(0)
 
HRESULT VariantFromString(
	PCWSTR wszValue, 
	VARIANT &Variant)
/*++

Routine Description:

	Creates a VT_BSTR variant from a null terminated string.

Arguments:

	wszValue - string to be converted
	Variant - output variant

Return Value:

	HRESULT indicating success or failure.
		
--*/
{
	HRESULT hr = S_OK;
	BSTR bstr = SysAllocString(wszValue);
	CHK_ALLOC(bstr);

	V_VT(&Variant) = VT_BSTR;
	V_BSTR(&Variant) = bstr;

CleanUp:
	return hr;
}

HRESULT CreateAndInitDOM(
	IXMLDOMDocument **ppDoc)
/*++

Routine Description:

	Creates a DOM instance.

Arguments:

	ppDoc - document

Return Value:

	HRESULT indicating success or failure.
		
--*/
{
	HRESULT hr = CoCreateInstance(__uuidof(DOMDocument60), nullptr, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(ppDoc));
	if (SUCCEEDED(hr))
	{
		// these methods should not fail so don't inspect result
		(*ppDoc)->put_async(VARIANT_FALSE);
		(*ppDoc)->put_validateOnParse(VARIANT_FALSE);
		(*ppDoc)->put_resolveExternals(VARIANT_FALSE);
	}
	return hr;
}

unsigned int convertToIPAddress(
	unsigned char *ipAddressinStringFormat)
/*++

Routine Description:

	converts an ip address in string format to unsigned int.

Arguments:

	ipAddressinStringFormat - ip address in string format.

Return Value:
	
	ip address in unsinged int. 0 in case of failure.
--*/
{
	struct in_addr ipaddr;
	int retval;
	LOGINFO("Entering convertToIPAddress.");
	if(ipAddressinStringFormat == NULL)
	{
		LOGERR("Cannot convert NULL to an ip address");
		return 0;
	}
	retval = inet_pton(AF_INET, (char *)ipAddressinStringFormat, &ipaddr);
	if (retval == 1)
	{
		LOGINFO("Successfully converted ipAdress in string %p to unsigned int %d", ipAddressinStringFormat, ipaddr.s_addr);
		return ipaddr.s_addr;
	}
	
	LOGERR("Uable to convert %p from string to ip adress.", ipAddressinStringFormat);
	return 0;
}


void getAndSetPrimaryIp(
	IXMLDOMNode* pXMLIPSubnetNode, 
	NclRepositoryManager *nclRepositoryManager)
/*++

Routine Description:
	
	Retrives the primary ip from nmagent response and saves it in repository.

Arguments:

	pXMLIPSubnetNode - node to parse
	nclRepositoryManager - handle to repository

Return Value:
	N/A
		
--*/
{
	HRESULT hr = S_OK;
	IXMLDOMNodeList* pXMLIPAddressList = nullptr;
	IXMLDOMNode* pXMLIPAddressNode = nullptr;
	IXMLDOMNamedNodeMap*  pXMLIPAddressAttributes = nullptr;
	IXMLDOMNode *ipAttribute = nullptr;
	BSTR ipAddress = nullptr;
	BSTR ipType = nullptr;
	BSTR ipAddressNodeName = nullptr;

	CHK_HR(pXMLIPSubnetNode->get_childNodes(&pXMLIPAddressList));
	long ipAddressCount;
	CHK_HR(pXMLIPAddressList->get_length(&ipAddressCount));
	for (int l = 0; l<ipAddressCount; l++)
	{
		CHK_HR(pXMLIPAddressList->get_item(l, &pXMLIPAddressNode));
		CHK_HR(pXMLIPAddressNode->get_nodeName(&ipAddressNodeName));
		if (wcscmp(ipAddressNodeName, L"IPAddress") == 0)
		{										
			CHK_HR(pXMLIPAddressNode->get_attributes(&pXMLIPAddressAttributes));
			
			CHK_HR(pXMLIPAddressAttributes->getNamedItem(L"IsPrimary", &ipAttribute));
			CHK_HR(ipAttribute->get_text(&ipType));
			LOGVERBOSE("isPrimary==%S ", ipType);
			SAFE_RELEASE(ipAttribute);

			if(wcscmp(ipType,L"true") == 0) 
			{
				CHK_HR(pXMLIPAddressAttributes->getNamedItem(L"Address", &ipAttribute));
				CHK_HR(ipAttribute->get_text(&ipAddress));
				LOGVERBOSE("ip=%S ", ipAddress);
				SAFE_RELEASE(ipAttribute);		
				char *ipStr = _com_util::ConvertBSTRToString(ipAddress);
				unsigned int ipInt = convertToIPAddress((unsigned char *)ipStr);
				LOGVERBOSE("Primary ip of primary nic:%d",ipInt);
				nclRepositoryManager->SetRepositoryPrimaryIp(ipInt);
				delete[] ipStr;
			}

			SysFreeString(ipAddress);
			SysFreeString(ipType);
			ipAddress=nullptr;
			ipType=nullptr;
			SAFE_RELEASE(pXMLIPAddressAttributes);
		}
		SAFE_RELEASE(pXMLIPAddressNode);
		SysFreeString(ipAddressNodeName);
		ipAddressNodeName = nullptr;
	}

CleanUp:
	SAFE_RELEASE(pXMLIPAddressList);
	SAFE_RELEASE(pXMLIPAddressNode);
	SAFE_RELEASE(pXMLIPAddressAttributes);
	SAFE_RELEASE(ipAttribute);

	SysFreeString(ipAddressNodeName);
	SysFreeString(ipAddress);
	SysFreeString(ipType);
	

}


void GetSubnetGatewayPrimaryIp(
	char *xmlResponse, 
	char **subnetIp, 
	char **gatewayIp, 
	NclRepositoryManager * nclRepositoryManager)
/*++

Routine Description:

	Parses the nm agent response for gateway and subnet ip.
	Also persists primary ip in repository.

Arguments:

	xmlResponse - nm agent response to parse
	subnetIp - (out praram) will contain subnet ip
	gatewayIp - (out param) will contain gateway ip
	nclRepositoryIp - repository handle to save praimry CA

Return Value:

	N/A
		
--*/
{
	LOGINFO("Entering GetSubnetGatewayPrimaryIp");
	HRESULT hr = S_OK;
	IXMLDOMDocument *pXMLDom = nullptr;
	IXMLDOMParseError *pXMLErr = nullptr;
	IXMLDOMNodeList* pXMLNodeList = nullptr;
	IXMLDOMNode* pXMLNode = nullptr;
	IXMLDOMNode* pXMLChildNode = nullptr;
	IXMLDOMNodeList* pXMLChildNodeList = nullptr;
	IXMLDOMNodeList* pXMLIPSubnetList = nullptr;
	IXMLDOMNode* pXMLIPSubnetNode = nullptr;
	IXMLDOMNamedNodeMap*  pXMLInterfaceAttributes = nullptr;
	IXMLDOMNode *primaryAttribute = nullptr;
	IXMLDOMNamedNodeMap*  pXMLSubnetAttributes = nullptr;
	IXMLDOMNode *prefixAttribute = nullptr;
	
	BSTR subnetAddress = nullptr;
	BSTR bstrXML = nullptr;
	BSTR bstrErr = nullptr;
	BSTR nodeName = nullptr;
	BSTR childNodeName = nullptr;
	BSTR ipSubnetNodeName = nullptr;
	BSTR primary = nullptr;
	VARIANT_BOOL varStatus = 0;
	VARIANT varFileName;

	VariantInit(&varFileName);
	CreateAndInitDOM(&pXMLDom);
	LOGVERBOSE("Going to load xml from string->\n %s", xmlResponse);
	CComBSTR combstr(xmlResponse);
	CHK_HR(pXMLDom->loadXML(combstr, &varStatus));
	if (varStatus == VARIANT_TRUE)
	{
		CHK_HR(pXMLDom->get_xml(&bstrXML));
		LOGVERBOSE("%S", bstrXML);
		CHK_HR(pXMLDom->get_childNodes(&pXMLNodeList));
		long lCount;
		pXMLNodeList->get_length(&lCount);
		LOGVERBOSE("Retrieved %d nodes from response.", lCount);
		for (int i = 0; i < lCount; i++)
		{
			CHK_HR(pXMLNodeList->get_item(i, &pXMLNode));
			CHK_HR(pXMLNode->get_nodeName(&nodeName));
			LOGVERBOSE("Node %i -> Name: %ls.", i, nodeName);
			if (wcscmp(nodeName, L"Interfaces") == 0)
			{
				CHK_HR(pXMLNode->get_childNodes(&pXMLChildNodeList));
				long childCount;
				CHK_HR(pXMLChildNodeList->get_length(&childCount));
				for (int j = 0; j < childCount; j++)
				{
					CHK_HR(pXMLChildNodeList->get_item(j, &pXMLChildNode));
					CHK_HR(pXMLChildNode->get_nodeName(&childNodeName));
					LOGVERBOSE("Response -> Node %ls -> ChildNode -> %ls.", nodeName, childNodeName);

					if (wcscmp(childNodeName, L"Interface") == 0)
					{
						CHK_HR(pXMLChildNode->get_attributes(&pXMLInterfaceAttributes));
						CHK_HR(pXMLInterfaceAttributes->getNamedItem(L"IsPrimary", &primaryAttribute));
						CHK_HR(primaryAttribute->get_text(&primary));
						SAFE_RELEASE(primaryAttribute);
						SAFE_RELEASE(pXMLInterfaceAttributes);
						LOGVERBOSE("Primary flag:%S\n", primary);
						if (wcscmp(primary, L"true") == 0)
						{
							LOGVERBOSE("Found primary interface.");
							SysFreeString(primary);
							primary = nullptr;
							SysFreeString(childNodeName);
							childNodeName = nullptr;
							break; 
						}
						SysFreeString(primary);
						primary = nullptr;
					}
			
					SAFE_RELEASE(pXMLChildNode);
					SysFreeString(childNodeName);
					childNodeName = nullptr;
				}

				CHK_HR(pXMLChildNode->get_childNodes(&pXMLIPSubnetList));
				SAFE_RELEASE(pXMLChildNode);
				long subnetCount;
				CHK_HR(pXMLIPSubnetList->get_length(&subnetCount));				
				CHK_HR(pXMLIPSubnetList->get_item(0, &pXMLIPSubnetNode));
				CHK_HR(pXMLIPSubnetNode->get_nodeName(&ipSubnetNodeName));

				if (wcscmp(ipSubnetNodeName, L"IPSubnet") == 0)
				{
					CHK_HR(pXMLIPSubnetNode->get_attributes(&pXMLSubnetAttributes));

					CHK_HR(pXMLSubnetAttributes->getNamedItem(L"Prefix", &prefixAttribute));
					CHK_HR(prefixAttribute->get_text(&subnetAddress));
					size_t wlen = wcslen(subnetAddress) + 1;
					*subnetIp = (char *)malloc(wlen);
					size_t bytesCopied = 0;
					wcstombs_s(&bytesCopied, *subnetIp, wlen, subnetAddress, wlen);
					LOGVERBOSE("Subnet ip=%s.", *subnetIp);
					SysFreeString(subnetAddress);
					subnetAddress = nullptr;
					SAFE_RELEASE(prefixAttribute);
					SAFE_RELEASE(pXMLSubnetAttributes);
					char *subTemp, *gwTemp;
					size_t len = strlen(*subnetIp);
					LOGVERBOSE("Len:%d", (int) len);
					*gatewayIp = (char *)malloc(len+1);
					int dotCount = 0;
					int k = 0;
					subTemp = *subnetIp;
					gwTemp = *gatewayIp;
					for (k = 0; k < len+1; k++)
					{
						if (subTemp[k] == '.') {
							dotCount++;
						}
						gwTemp[k] = subTemp[k];
						if (dotCount == 3)
						{
							break;
						}
					}
					gwTemp[++k] = '1';
					gwTemp[++k] = '\0';
					LOGVERBOSE("gateway Ip=%s\n", gwTemp);
					getAndSetPrimaryIp(pXMLIPSubnetNode, nclRepositoryManager);
				}
				
				SAFE_RELEASE(pXMLChildNodeList);
				SAFE_RELEASE(pXMLIPSubnetList);
				SAFE_RELEASE(pXMLIPSubnetNode);
				SysFreeString(ipSubnetNodeName);
				ipSubnetNodeName = nullptr;
			}

			// cleanup the node
			SAFE_RELEASE(pXMLNode);
			SysFreeString(nodeName);
			nodeName = nullptr;
		}
	}
	else
	{
		// Failed to load xml, get last parsing error
		CHK_HR(pXMLDom->get_parseError(&pXMLErr));
		CHK_HR(pXMLErr->get_reason(&bstrErr));
		LOGERR( "Failed to load/parse the xml document from string %s. Error: %ls", xmlResponse, bstrErr);
	}


CleanUp:
	SAFE_RELEASE(pXMLDom);
	SAFE_RELEASE(pXMLErr);
	SAFE_RELEASE(pXMLChildNode);
	SAFE_RELEASE(pXMLChildNodeList);
	SAFE_RELEASE(pXMLIPSubnetNode);
	SAFE_RELEASE(pXMLIPSubnetList);
	SAFE_RELEASE(pXMLNode);
	SAFE_RELEASE(pXMLNodeList);
	SAFE_RELEASE(pXMLSubnetAttributes);

	SysFreeString(bstrXML);
	SysFreeString(bstrErr);
	SysFreeString(nodeName);
	SysFreeString(childNodeName);
	SysFreeString(subnetAddress);
	SysFreeString(ipSubnetNodeName);
	VariantClear(&varFileName);
	LOGINFO("Exiting GetSubnetGatewayPrimaryIp");
}

void CountOrRetrieveSecondaryIps(
	char *xmlResponse, 
	int countOnly, 
	unsigned int *count, 
	unsigned int *ips)
/*++

Routine Description:

	Parses nm agent response for secondary IPs.

Arguments:

	xmlResponse - nm agent response
	countOnly - boolena indicating whether to only count secondary CAs or to also pupulate the out param
	count - (out parameter) count for seoncary CAs
	ips - (out parameter) values for secondary CAs

Return Value:

	N/A
		
--*/
{
	LOGINFO("Entering CountOrRetrieveSecondaryIps.");
	HRESULT hr = S_OK;
	IXMLDOMDocument *pXMLDom = nullptr;
	IXMLDOMParseError *pXMLErr = nullptr;
	IXMLDOMNodeList* pXMLNodeList = nullptr;
	IXMLDOMNode* pXMLNode = nullptr;
	IXMLDOMNode* pXMLChildNode = nullptr;
	IXMLDOMNodeList* pXMLChildNodeList = nullptr;
	IXMLDOMNodeList* pXMLIPSubnetList = nullptr;
	IXMLDOMNodeList* pXMLIPAddressList = nullptr;
	IXMLDOMNode* pXMLIPSubnetNode = nullptr;
	IXMLDOMNode* pXMLIPAddressNode = nullptr;
	IXMLDOMNamedNodeMap*  pXMLIPAddressAttributes = nullptr;
	IXMLDOMNode *ipAttribute = nullptr;

	BSTR bstrXML = nullptr;
	BSTR bstrErr = nullptr;
	BSTR nodeName = nullptr;
	BSTR childNodeName = nullptr;
	BSTR ipSubnetNodeName = nullptr;
	BSTR ipAddressNodeName = nullptr;
	BSTR ipAddress = nullptr;
	BSTR ipType = nullptr;
	VARIANT_BOOL varStatus = 0;
	VARIANT varFileName;
	VariantInit(&varFileName);
	CreateAndInitDOM(&pXMLDom);
	LOGVERBOSE("Going to load xml from string-> %s", xmlResponse);
	CComBSTR combstr(xmlResponse);
	CHK_HR(pXMLDom->loadXML(combstr, &varStatus));

	if (varStatus == VARIANT_TRUE)
	{
		CHK_HR(pXMLDom->get_xml(&bstrXML));
		LOGVERBOSE("%S", bstrXML);		
		CHK_HR(pXMLDom->get_childNodes(&pXMLNodeList));
		long lCount;
		pXMLNodeList->get_length(&lCount);
		LOGVERBOSE("Retrieved %d nodes from response.", lCount);

		for (int i = 0; i < lCount; i++)
		{
			CHK_HR(pXMLNodeList->get_item(i, &pXMLNode));
			CHK_HR(pXMLNode->get_nodeName(&nodeName));
			LOGVERBOSE("Node %i -> Name: %ls.", i, nodeName);

			if (wcscmp(nodeName, L"Interfaces") == 0)
			{
				CHK_HR(pXMLNode->get_childNodes(&pXMLChildNodeList));
				long childCount;
				CHK_HR(pXMLChildNodeList->get_length(&childCount));
				for (int j = 0; j < childCount; j++)
				{
					CHK_HR(pXMLChildNodeList->get_item(j, &pXMLChildNode));
					CHK_HR(pXMLChildNode->get_nodeName(&childNodeName));
					LOGVERBOSE("Response -> Node %ls -> ChildNode -> %ls.", nodeName, childNodeName);

					if (wcscmp(childNodeName, L"Interface") == 0)
					{
						LOGVERBOSE("Located an interface.");
						
						CHK_HR(pXMLChildNode->get_childNodes(&pXMLIPSubnetList));
						long subnetCount;
						CHK_HR(pXMLIPSubnetList->get_length(&subnetCount));
						for (int k = 0; k < subnetCount; k++)
						{
							CHK_HR(pXMLIPSubnetList->get_item(k, &pXMLIPSubnetNode));
							CHK_HR(pXMLIPSubnetNode->get_nodeName(&ipSubnetNodeName));
							if (wcscmp(ipSubnetNodeName, L"IPSubnet") == 0)
							{
								LOGVERBOSE("Located an ip subnet.");
								CHK_HR(pXMLIPSubnetNode->get_childNodes(&pXMLIPAddressList));
								long ipAddressCount;
								CHK_HR(pXMLIPAddressList->get_length(&ipAddressCount));
								for (int l = 0; l<ipAddressCount; l++)
								{
									CHK_HR(pXMLIPAddressList->get_item(l, &pXMLIPAddressNode));
									CHK_HR(pXMLIPAddressNode->get_nodeName(&ipAddressNodeName));
									if (wcscmp(ipAddressNodeName, L"IPAddress") == 0)
									{										
										CHK_HR(pXMLIPAddressNode->get_attributes(&pXMLIPAddressAttributes));										
										CHK_HR(pXMLIPAddressAttributes->getNamedItem(L"Address", &ipAttribute));
										CHK_HR(ipAttribute->get_text(&ipAddress));
										LOGVERBOSE("ip=%S ", ipAddress);
										SAFE_RELEASE(ipAttribute);										
										CHK_HR(pXMLIPAddressAttributes->getNamedItem(L"IsPrimary", &ipAttribute));
										CHK_HR(ipAttribute->get_text(&ipType));
										LOGVERBOSE("isPrimary==%S ", ipType);
										SAFE_RELEASE(ipAttribute);
										char *ipStr = _com_util::ConvertBSTRToString(ipAddress);
										char *primary = _com_util::ConvertBSTRToString(ipType);

										if(strncmp(primary, "false", 5) == 0)
										{
											unsigned int ipInt = convertToIPAddress((unsigned char *)ipStr);
											if(!countOnly)
											{
												ips[*count] = ipInt;
											}
											(*count) = (*count) + 1;
											LOGVERBOSE("Secondary IP address in unsigned int=%u", ipInt);
										}

										SysFreeString(ipAddress);
										SysFreeString(ipType);
										ipAddress=nullptr;
										ipType=nullptr;
										delete[] ipStr;
										delete[] primary;
										SAFE_RELEASE(pXMLIPAddressAttributes);
									}
									SAFE_RELEASE(pXMLIPAddressNode);
									SysFreeString(ipAddressNodeName);
									ipAddressNodeName = nullptr;
								}
								SAFE_RELEASE(pXMLIPAddressList);
							}
							SAFE_RELEASE(pXMLIPSubnetNode);
							SysFreeString(ipSubnetNodeName);
							ipSubnetNodeName = nullptr;
						}
						SAFE_RELEASE(pXMLIPSubnetList);
					}
					SAFE_RELEASE(pXMLChildNode);
					SysFreeString(childNodeName);
					childNodeName = nullptr;
				}
				SAFE_RELEASE(pXMLChildNodeList);
			}
			// cleanup the node
			SAFE_RELEASE(pXMLNode);
			SysFreeString(nodeName);
			nodeName = nullptr;
		}
	}
	else
	{
		// Failed to load xml, get last parsing error
		CHK_HR(pXMLDom->get_parseError(&pXMLErr));
		CHK_HR(pXMLErr->get_reason(&bstrErr));
		LOGERR("Failed to load/parse the xml document from string %s. Error: %ls", xmlResponse, bstrErr);
	}

CleanUp:
	SAFE_RELEASE(pXMLDom);
	SAFE_RELEASE(pXMLErr);
	SAFE_RELEASE(pXMLChildNode);
	SAFE_RELEASE(pXMLChildNodeList);
	SAFE_RELEASE(pXMLIPSubnetNode);
	SAFE_RELEASE(pXMLIPAddressNode);
	SAFE_RELEASE(pXMLIPAddressList);
	SAFE_RELEASE(pXMLIPSubnetList);
	SAFE_RELEASE(pXMLNode);
	SAFE_RELEASE(pXMLNodeList);
	SAFE_RELEASE(pXMLIPAddressAttributes);
	SAFE_RELEASE(ipAttribute);
	
	SysFreeString(bstrXML);
	SysFreeString(bstrErr);
	SysFreeString(nodeName);
	SysFreeString(childNodeName);
	SysFreeString(ipSubnetNodeName);
	SysFreeString(ipAddressNodeName);
	SysFreeString(ipAddress);
	SysFreeString(ipType);

	VariantClear(&varFileName);

	LOGINFO("Exiting CountOrRetrieveSecondaryIps.");
}

DllExport int ParseSecondaryIpAddresses(
	char *xmlResponse, 
	int countOnly, 
	unsigned int *ips)
/*++

Routine Description:

	Initializez COM and calls method to parse nm agent response.

Arguments:

	xmlResponse - nm agent response
	countOnly - boolena indicating whether to only count secondary CAs or to also pupulate the out param

Return Value:

	number of secondary CAs parsed, 0 upon failure.
		
--*/
{
	LOGINFO("Entering ParseSecondaryIpAddresses");
	unsigned int ipCount = 0;
	bool shouldCoUnInitialize = true;
	HRESULT hr = CoInitializeEx(nullptr, COINIT_MULTITHREADED);
	
	if(hr == RPC_E_CHANGED_MODE) // this is considered as failed call, and only successfull calls should have couninitiallize
	{
		LOGWARN("CoInitializeEx returned RPC_E_CHANGED_MODE. code: 0x%X", hr);
		shouldCoUnInitialize = false;
	}
	if (hr != E_INVALIDARG && 
		hr != E_OUTOFMEMORY &&
		hr != E_UNEXPECTED)
	{
		try
		{
			CountOrRetrieveSecondaryIps(xmlResponse, countOnly, &ipCount, ips);
		}
		catch (...)
		{
			LOGERR("Exception in CountOrRetrieveSecondaryIps");
		}

		if (shouldCoUnInitialize)
		{
			CoUninitialize();
		}
	}
	else
	{
		LOGERR("Coinitialize failed with code 0x%X", hr);
	}
	LOGINFO("Exiting ParseSecondaryIpAddresses");
	return ipCount;
}

DllExport void ParseSubnetGatewayPrimaryIp(
	char *xmlResponse, 
	char **subnetIp, 
	char **gatewayIp, 
	NclRepositoryManager *nclRepositoryManager)
/*++

Routine Description:

	Initializes COM and calls method to parse nm agent response.
	
Arguments:

	xmlResponse - nm agent response to parse
	subnetIp - (out praram) will contain subnet ip
	gatewayIp - (out param) will contain gateway ip
	nclRepositoryIp - repository handle to save praimry CA

Return Value:

	N/A
		
--*/
{
	LOGINFO("Entering ParseSubnetGatewayPrimaryIp.");
	bool shouldCoUnInitialize = true;
	HRESULT hr = CoInitializeEx(nullptr, COINIT_MULTITHREADED);

	if (hr == RPC_E_CHANGED_MODE) // this is considered as failed call, and only successfull calls should have couninitiallize
	{
		LOGWARN("CoInitializeEx returned RPC_E_CHANGED_MODE. code: 0x%X", hr);
		shouldCoUnInitialize = false;
	}

	if (hr != E_INVALIDARG &&
		hr != E_OUTOFMEMORY &&
		hr != E_UNEXPECTED)
	{
		try
		{
			GetSubnetGatewayPrimaryIp(xmlResponse, subnetIp, gatewayIp, nclRepositoryManager);
		}
		catch (...)
		{
			LOGERR("Exception in RetrieveSubnetGatewayIp");
		}

		if (shouldCoUnInitialize)
		{
			CoUninitialize();
		}
	}
	else
	{
		LOGERR("Coinitialize failed with code 0x%X\n", hr);
	}
	LOGINFO("Exiting ParseSubnetGatewayPrimaryIp.");
}
} } } } }
#endif