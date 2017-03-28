#ifdef _WIN32
/****************************** Module Header ******************************\
* Module Name:  ReponseParser.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Parser for nm agent response
*
\***************************************************************************/

#include <msxml6.h>
#include <stdio.h>
#include "atlbase.h"
#include "comutil.h"
#include <ws2tcpip.h>

// Macro that calls a COM method returning HRESULT value.
#define CHK_HR(stmt)        do { hr=(stmt); if (FAILED(hr)) goto CleanUp; } while(0)

// Macro to verify memory allcation.
#define CHK_ALLOC(p)        do { if (!(p)) { hr = E_OUTOFMEMORY; goto CleanUp; } } while(0)

// Macro that releases a COM object if not NULL.
#define SAFE_RELEASE(p)     do { if ((p)) { (p)->Release(); (p) = nullptr; } } while(0)

// Helper function to create a VT_BSTR variant from a null terminated string. 
HRESULT VariantFromString(PCWSTR wszValue, VARIANT &Variant)
{
	HRESULT hr = S_OK;
	BSTR bstr = SysAllocString(wszValue);
	CHK_ALLOC(bstr);

	V_VT(&Variant) = VT_BSTR;
	V_BSTR(&Variant) = bstr;

CleanUp:
	return hr;
}

// Helper function to create a DOM instance. 
HRESULT CreateAndInitDOM(IXMLDOMDocument **ppDoc)
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

unsigned int ConvertToIPAddress(unsigned char *ipAddressinStringFormat)
{
	struct in_addr ipaddr;
	int retval;
	retval = inet_pton(AF_INET, (char *)ipAddressinStringFormat, &ipaddr);
	if (retval == 1)
	{
		return ipaddr.s_addr;
	}
	else
	{
		fprintf(stderr, "Uable to convert %s from string to ip adress.", ipAddressinStringFormat);
	}
	return 0;
}


void RetrieveSubnetGatewayIp(char *xmlResponse, char **subnetIp, char **gatewayIp)
{
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

	//TM
	IXMLDOMNamedNodeMap*  pXMLInterfaceAttributes = nullptr;
	IXMLDOMNode *primaryAttribute = nullptr;
	BSTR primary = nullptr;

	IXMLDOMNamedNodeMap*  pXMLSubnetAttributes = nullptr;
	IXMLDOMNode *prefixAttribute = nullptr;
	BSTR subnetAddress = nullptr;

	//TM

	IXMLDOMNode *ipAttribute = nullptr;

	BSTR bstrXML = nullptr;
	BSTR bstrErr = nullptr;
	BSTR nodeName = nullptr;
	BSTR childNodeName = nullptr;
	BSTR ipSubnetNodeName = nullptr;
	BSTR ipAddressNodeName = nullptr;
	BSTR programName = nullptr;
	BSTR ipAddress = nullptr;
	BSTR ipType = nullptr;
	VARIANT_BOOL varStatus = 0;
	VARIANT varFileName;
	VariantInit(&varFileName);
	wchar_t *entryPoint = nullptr;
	CreateAndInitDOM(&pXMLDom);
	printf("Going to load xml from string->\n %s\n", xmlResponse);
	CComBSTR combstr(xmlResponse);
	CHK_HR(pXMLDom->loadXML(combstr, &varStatus));
	if (varStatus == VARIANT_TRUE)
	{
		CHK_HR(pXMLDom->get_xml(&bstrXML));
		// printf("XML DOM loaded from stocks.xml:\n%S\n", bstrXML);
		printf("%S\n\n", bstrXML);

		CHK_HR(pXMLDom->get_childNodes(&pXMLNodeList));
		long lCount;
		pXMLNodeList->get_length(&lCount);
		printf("Retrieved %d nodes from response. \n", lCount);
		for (int i = 0; i < lCount; i++)
		{
			CHK_HR(pXMLNodeList->get_item(i, &pXMLNode));
			// do something based on node
			CHK_HR(pXMLNode->get_nodeName(&nodeName));
			printf("Node %i -> Name: %ls. \n", i, nodeName);
			if (wcscmp(nodeName, L"Interfaces") == 0)
			{
				CHK_HR(pXMLNode->get_childNodes(&pXMLChildNodeList));
				long childCount;
				CHK_HR(pXMLChildNodeList->get_length(&childCount));
				for (int j = 0; j < childCount; j++)
				{
					CHK_HR(pXMLChildNodeList->get_item(j, &pXMLChildNode));
					CHK_HR(pXMLChildNode->get_nodeName(&childNodeName));
					printf("Response -> Node %ls -> ChildNode -> %ls. \n", nodeName, childNodeName);

					if (wcscmp(childNodeName, L"Interface") == 0)
					{
						printf("Located an interface. \n");

						//TM
						CHK_HR(pXMLChildNode->get_attributes(&pXMLInterfaceAttributes));
						CHK_HR(pXMLInterfaceAttributes->getNamedItem(L"IsPrimary", &primaryAttribute));
						CHK_HR(primaryAttribute->get_text(&primary));

						printf("primary flag:%S\n", primary);

						if (wcscmp(primary, L"true") == 0)
						{
							printf("***found primary interface***\n");
							break;
						}
					}
			
					SAFE_RELEASE(pXMLChildNode);
					SysFreeString(childNodeName);
					childNodeName = nullptr;
				}

				CHK_HR(pXMLChildNode->get_childNodes(&pXMLIPSubnetList));
				long subnetCount;
				CHK_HR(pXMLIPSubnetList->get_length(&subnetCount));
				for (int k = 0; k < 1; k++)
				{
					CHK_HR(pXMLIPSubnetList->get_item(k, &pXMLIPSubnetNode));
					CHK_HR(pXMLIPSubnetNode->get_nodeName(&ipSubnetNodeName));
					if (wcscmp(ipSubnetNodeName, L"IPSubnet") == 0)
					{
						CHK_HR(pXMLIPSubnetNode->get_attributes(&pXMLSubnetAttributes));

						CHK_HR(pXMLSubnetAttributes->getNamedItem(L"Prefix", &prefixAttribute));
						CHK_HR(prefixAttribute->get_text(&subnetAddress));
						int wlen = wcslen(subnetAddress) + 1;
						*subnetIp = (char *)malloc(wlen);
						wcstombs(*subnetIp, subnetAddress, wlen);
						printf("Subnet ip=%s\n", *subnetIp);
						SAFE_RELEASE(prefixAttribute);

						CHK_HR(pXMLIPSubnetNode->get_childNodes(&pXMLIPAddressList));
						long ipAddressCount;
						CHK_HR(pXMLIPAddressList->get_length(&ipAddressCount));
						for (int l = 0; l < ipAddressCount; l++)
						{
							CHK_HR(pXMLIPAddressList->get_item(l, &pXMLIPAddressNode));
							CHK_HR(pXMLIPAddressNode->get_nodeName(&ipAddressNodeName));
							if (wcscmp(ipAddressNodeName, L"IPAddress") == 0)
							{
								CHK_HR(pXMLIPAddressNode->get_attributes(&pXMLIPAddressAttributes));
								CHK_HR(pXMLIPAddressAttributes->getNamedItem(L"IsPrimary", &ipAttribute));
								CHK_HR(ipAttribute->get_text(&ipType));
								printf("isPrimary==%S ", ipType);
								SAFE_RELEASE(ipAttribute);

								if (wcscmp(ipType, L"true") == 0)
								{
									CHK_HR(pXMLIPAddressNode->get_attributes(&pXMLIPAddressAttributes));
									CHK_HR(pXMLIPAddressAttributes->getNamedItem(L"Address", &ipAttribute));
									CHK_HR(ipAttribute->get_text(&ipAddress));
									int wlen = wcslen(ipAddress) + 1;
									*gatewayIp = (char *)malloc(wlen);
									wcstombs(*gatewayIp, ipAddress, wlen);
									printf("Gateway ip ch=%s\n", *gatewayIp);

									SAFE_RELEASE(ipAttribute);
									break;
								}
							}
						}
					}
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
		//printf("Failed to load DOM from stocks.xml. %S\n", bstrErr);
		printf("Failed to load/parse the xml document from string %s. Error: %ls\n", xmlResponse, bstrErr);
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
	SysFreeString(programName);
	SysFreeString(ipAddress);
	SysFreeString(ipType);

	VariantClear(&varFileName);
}








void CountOrRetrieveSecondaryIps(char *xmlResponse, int countOnly, unsigned int *count, unsigned int *ips)
{
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

	//TM
	IXMLDOMNamedNodeMap*  pXMLInterfaceAttributes = nullptr;
	IXMLDOMNode *primaryAttribute = nullptr;
	BSTR primary = nullptr;
	//TM

	IXMLDOMNode *ipAttribute = nullptr;

	BSTR bstrXML = nullptr;
	BSTR bstrErr = nullptr;
	BSTR nodeName = nullptr;
	BSTR childNodeName = nullptr;
	BSTR ipSubnetNodeName = nullptr;
	BSTR ipAddressNodeName = nullptr;
	BSTR programName = nullptr;
	BSTR ipAddress = nullptr;
	BSTR ipType = nullptr;
	VARIANT_BOOL varStatus = 0;
	VARIANT varFileName;
	VariantInit(&varFileName);
	wchar_t *entryPoint = nullptr;
	CreateAndInitDOM(&pXMLDom);
	printf("Going to load xml from string->\n %s\n", xmlResponse);
	CComBSTR combstr(xmlResponse);
	CHK_HR(pXMLDom->loadXML(combstr, &varStatus));
	if (varStatus == VARIANT_TRUE)
	{
		CHK_HR(pXMLDom->get_xml(&bstrXML));
		// printf("XML DOM loaded from stocks.xml:\n%S\n", bstrXML);
		printf("%S\n\n", bstrXML);
		
		CHK_HR(pXMLDom->get_childNodes(&pXMLNodeList));
		long lCount;
		pXMLNodeList->get_length(&lCount);
		printf("Retrieved %d nodes from response. \n", lCount);
		for (int i = 0; i < lCount; i++)
		{
			CHK_HR(pXMLNodeList->get_item(i, &pXMLNode));
			// do something based on node
			CHK_HR(pXMLNode->get_nodeName(&nodeName));
			printf("Node %i -> Name: %ls. \n", i, nodeName);
			if (wcscmp(nodeName, L"Interfaces") == 0)
			{
				CHK_HR(pXMLNode->get_childNodes(&pXMLChildNodeList));
				long childCount;
				CHK_HR(pXMLChildNodeList->get_length(&childCount));
				for (int j = 0; j < childCount; j++)
				{
					CHK_HR(pXMLChildNodeList->get_item(j, &pXMLChildNode));
					CHK_HR(pXMLChildNode->get_nodeName(&childNodeName));
					printf("Response -> Node %ls -> ChildNode -> %ls. \n", nodeName, childNodeName);

				if (wcscmp(childNodeName, L"Interface") == 0)
					{
						printf("Located an interface. \n");
						
						CHK_HR(pXMLChildNode->get_childNodes(&pXMLIPSubnetList));
						long subnetCount;
						CHK_HR(pXMLIPSubnetList->get_length(&subnetCount));
						for (int k = 0; k < subnetCount; k++)
						{
							CHK_HR(pXMLIPSubnetList->get_item(k, &pXMLIPSubnetNode));
							CHK_HR(pXMLIPSubnetNode->get_nodeName(&ipSubnetNodeName));
							if (wcscmp(ipSubnetNodeName, L"IPSubnet") == 0)
							{
								//TM
								printf("Located an ip subnet. \n");
								//CHK_HR(pXMLChildNode->get_attributes(&pXMLSubnetAttributes));
								//TM

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
										printf("ip=%S ", ipAddress);
										SAFE_RELEASE(ipAttribute);
										
										CHK_HR(pXMLIPAddressAttributes->getNamedItem(L"IsPrimary", &ipAttribute));
										CHK_HR(ipAttribute->get_text(&ipType));
										printf("isPrimary==%S ", ipType);
										SAFE_RELEASE(ipAttribute);

										char *ipStr = _com_util::ConvertBSTRToString(ipAddress);
										char *primary = _com_util::ConvertBSTRToString(ipType);;
										if(strncmp(primary, "false", 5) == 0)
										{
											int ipInt = ConvertToIPAddress((unsigned char *)ipStr);
											if(!countOnly)
											{
												ips[*count] = ipInt;
											}
											(*count) = (*count) + 1;
											printf(" int=%d \n", ipInt);
										}
										else
										{
											printf("\n");
										}
										SysFreeString(ipAddress);
										SysFreeString(ipType);
										delete[] ipStr;
										delete[] primary;

										//printf(_bstr_t(varValue));
										CHK_HR(pXMLIPAddressNode->get_text(&programName));
										//printf("IP Address is [%S] \n", programName);
										if ((entryPoint) != nullptr)
										{
											delete[](entryPoint);
										}
										size_t len = wcsnlen(programName, MAX_PATH) + 1;
										(entryPoint) = new wchar_t[len];
										wcscpy_s((entryPoint), len, programName);
										SysFreeString(programName);
										programName = nullptr;
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
		//printf("Failed to load DOM from stocks.xml. %S\n", bstrErr);
		printf("Failed to load/parse the xml document from string %s. Error: %ls\n", xmlResponse, bstrErr);		
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
	SysFreeString(programName);
	SysFreeString(ipAddress);
	SysFreeString(ipType);

	VariantClear(&varFileName);
}

int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips)
{
	unsigned int ipCount = 0;
	HRESULT hr = CoInitialize(nullptr);
	if (SUCCEEDED(hr))
	{
		try
		{
			CountOrRetrieveSecondaryIps(xmlResponse, countOnly, &ipCount, ips);
		}
		catch (...)
		{
			CoUninitialize();
			throw;
		}
		CoUninitialize();
	}
	return ipCount;
}

void ParseSubnetGwIp(char *xmlResponse, char **subnetIp, char **gwIp)
{
	unsigned int ipCount = 0;
	HRESULT hr = CoInitialize(nullptr);
	if (SUCCEEDED(hr))
	{
		try
		{
			RetrieveSubnetGatewayIp(xmlResponse, subnetIp, gwIp);
		}
		catch (...)
		{
			CoUninitialize();
			throw;
		}
		CoUninitialize();
	}
}
#endif