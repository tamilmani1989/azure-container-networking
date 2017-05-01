/****************************** Module Header ******************************\
* Module Name:  ReponseParser.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Parser for nm agent response
*
\***************************************************************************/
#ifdef _WIN32
#include <msxml6.h>
#include <stdio.h>
#include "atlbase.h"
#include "comutil.h"
#include <ws2tcpip.h>
#include "../../Logging/Logging.h"
#define DllExport   __declspec( dllexport )  

namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {

// Macro that calls a COM method returning HRESULT value.
#define CHK_HR(stmt)        do { hr=(stmt); if (FAILED(hr)) goto CleanUp; } while(0)

// Macro to verify memory allcation.
#define CHK_ALLOC(p)        do { if (!(p)) { hr = E_OUTOFMEMORY; goto CleanUp; } } while(0)

// Macro that releases a COM object if not NULL.
#define SAFE_RELEASE(p)     do { if ((p)) { (p)->Release(); (p) = nullptr; } } while(0)


size_t msgSize = 2048;

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
		sprintf_s(logMsg, msgSize, "Uable to convert %s from string to ip adress.", ipAddressinStringFormat);
		AquaLogging::Log(err, logMsg);
	}
	return 0;
}

// TODO: Check for memory leaks..
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
	sprintf_s(logMsg, msgSize, "Going to load xml from string->\n %s\n", xmlResponse);
	AquaLogging::Log(verbose, logMsg);
	CComBSTR combstr(xmlResponse);
	CHK_HR(pXMLDom->loadXML(combstr, &varStatus));
	if (varStatus == VARIANT_TRUE)
	{
		CHK_HR(pXMLDom->get_xml(&bstrXML));
		sprintf_s(logMsg, msgSize, "%S\n\n", bstrXML);
		AquaLogging::Log(verbose, logMsg);

		CHK_HR(pXMLDom->get_childNodes(&pXMLNodeList));
		long lCount;
		pXMLNodeList->get_length(&lCount);
		sprintf_s(logMsg, msgSize, "Retrieved %d nodes from response. \n", lCount);
		AquaLogging::Log(verbose, logMsg);
		for (int i = 0; i < lCount; i++)
		{
			CHK_HR(pXMLNodeList->get_item(i, &pXMLNode));
			// do something based on node
			CHK_HR(pXMLNode->get_nodeName(&nodeName));
			sprintf_s(logMsg, msgSize, "Node %i -> Name: %ls. \n", i, nodeName);
			AquaLogging::Log(verbose, logMsg);
			if (wcscmp(nodeName, L"Interfaces") == 0)
			{
				CHK_HR(pXMLNode->get_childNodes(&pXMLChildNodeList));
				long childCount;
				CHK_HR(pXMLChildNodeList->get_length(&childCount));
				for (int j = 0; j < childCount; j++)
				{
					CHK_HR(pXMLChildNodeList->get_item(j, &pXMLChildNode));
					CHK_HR(pXMLChildNode->get_nodeName(&childNodeName));
					sprintf_s(logMsg, msgSize, "Response -> Node %ls -> ChildNode -> %ls. \n", nodeName, childNodeName);
					AquaLogging::Log(verbose, logMsg);

					if (wcscmp(childNodeName, L"Interface") == 0)
					{
						CHK_HR(pXMLChildNode->get_attributes(&pXMLInterfaceAttributes));
						CHK_HR(pXMLInterfaceAttributes->getNamedItem(L"IsPrimary", &primaryAttribute));
						CHK_HR(primaryAttribute->get_text(&primary));
						SAFE_RELEASE(primaryAttribute);
						SAFE_RELEASE(pXMLInterfaceAttributes);
						sprintf_s(logMsg, msgSize, "primary flag:%S\n", primary);
						AquaLogging::Log(verbose, logMsg);
						if (wcscmp(primary, L"true") == 0)
						{
							sprintf_s(logMsg, msgSize, "***Found primary interface***\n");
							AquaLogging::Log(verbose, logMsg);
							SysFreeString(primary);
							primary = nullptr;
							SysFreeString(childNodeName);
							childNodeName = nullptr;

							break; // remember to free up the memory
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
					sprintf_s(logMsg, msgSize, "Subnet ip=%s\n", *subnetIp);
					AquaLogging::Log(verbose, logMsg);

					SysFreeString(subnetAddress);
					subnetAddress = nullptr;
					SAFE_RELEASE(prefixAttribute);
					SAFE_RELEASE(pXMLSubnetAttributes);

					char *subTemp, *gwTemp;
					size_t len = strlen(*subnetIp);
					sprintf_s(logMsg, msgSize, "len:%d\n", (int) len);
					AquaLogging::Log(verbose, logMsg);
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
					sprintf_s(logMsg, msgSize, "gateway Ip=%s\n", gwTemp);
					AquaLogging::Log(verbose, logMsg);
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
			//SAFE_RELEASE(pXMLIPSubnetNode);
		}

	}
	else
	{
		// Failed to load xml, get last parsing error
		CHK_HR(pXMLDom->get_parseError(&pXMLErr));
		CHK_HR(pXMLErr->get_reason(&bstrErr));
		sprintf_s(logMsg, msgSize, "Failed to load/parse the xml document from string %s. Error: %ls\n", xmlResponse, bstrErr);
		AquaLogging::Log(err, logMsg);
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
	sprintf_s(logMsg, msgSize, "Going to load xml from string->\n %s\n", xmlResponse);
	AquaLogging::Log(verbose, logMsg);
	CComBSTR combstr(xmlResponse);
	CHK_HR(pXMLDom->loadXML(combstr, &varStatus));
	if (varStatus == VARIANT_TRUE)
	{
		CHK_HR(pXMLDom->get_xml(&bstrXML));
		sprintf_s(logMsg, msgSize, "%S\n\n", bstrXML);
		AquaLogging::Log(verbose, logMsg);
		
		CHK_HR(pXMLDom->get_childNodes(&pXMLNodeList));
		long lCount;
		pXMLNodeList->get_length(&lCount);
		sprintf_s(logMsg, msgSize, "Retrieved %d nodes from response. \n", lCount);
		AquaLogging::Log(verbose, logMsg);
		for (int i = 0; i < lCount; i++)
		{
			CHK_HR(pXMLNodeList->get_item(i, &pXMLNode));
			// do something based on node
			CHK_HR(pXMLNode->get_nodeName(&nodeName));
			sprintf_s(logMsg, msgSize, "Node %i -> Name: %ls. \n", i, nodeName);
			AquaLogging::Log(verbose, logMsg);
			if (wcscmp(nodeName, L"Interfaces") == 0)
			{
				CHK_HR(pXMLNode->get_childNodes(&pXMLChildNodeList));
				long childCount;
				CHK_HR(pXMLChildNodeList->get_length(&childCount));
				for (int j = 0; j < childCount; j++)
				{
					CHK_HR(pXMLChildNodeList->get_item(j, &pXMLChildNode));
					CHK_HR(pXMLChildNode->get_nodeName(&childNodeName));
					sprintf_s(logMsg, msgSize, "Response -> Node %ls -> ChildNode -> %ls. \n", nodeName, childNodeName);
					AquaLogging::Log(verbose, logMsg);

					if (wcscmp(childNodeName, L"Interface") == 0)
					{
						sprintf_s(logMsg, msgSize, "Located an interface. \n");
						AquaLogging::Log(verbose, logMsg);
						
						CHK_HR(pXMLChildNode->get_childNodes(&pXMLIPSubnetList));
						long subnetCount;
						CHK_HR(pXMLIPSubnetList->get_length(&subnetCount));
						for (int k = 0; k < subnetCount; k++)
						{
							CHK_HR(pXMLIPSubnetList->get_item(k, &pXMLIPSubnetNode));
							CHK_HR(pXMLIPSubnetNode->get_nodeName(&ipSubnetNodeName));
							if (wcscmp(ipSubnetNodeName, L"IPSubnet") == 0)
							{
								sprintf_s(logMsg, msgSize, "Located an ip subnet. \n");
								AquaLogging::Log(verbose, logMsg);
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
										sprintf_s(logMsg, msgSize, "ip=%S ", ipAddress);
										AquaLogging::Log(verbose, logMsg);
										SAFE_RELEASE(ipAttribute);
										
										CHK_HR(pXMLIPAddressAttributes->getNamedItem(L"IsPrimary", &ipAttribute));
										CHK_HR(ipAttribute->get_text(&ipType));
										sprintf_s(logMsg, msgSize, "isPrimary==%S ", ipType);
										AquaLogging::Log(verbose, logMsg);
										SAFE_RELEASE(ipAttribute);

										char *ipStr = _com_util::ConvertBSTRToString(ipAddress);
										char *primary = _com_util::ConvertBSTRToString(ipType);;
										if(strncmp(primary, "false", 5) == 0)
										{
											unsigned int ipInt = ConvertToIPAddress((unsigned char *)ipStr);
											if(!countOnly)
											{
												ips[*count] = ipInt;
											}
											(*count) = (*count) + 1;
											sprintf_s(logMsg, msgSize, " int=%d \n", ipInt);
											AquaLogging::Log(verbose, logMsg);
										}
										else
										{
											sprintf_s(logMsg, msgSize, "\n");
											AquaLogging::Log(verbose, logMsg);
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
		sprintf_s(logMsg, msgSize, "Failed to load/parse the xml document from string %s. Error: %ls\n", xmlResponse, bstrErr);
		AquaLogging::Log(err, logMsg);
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
}

DllExport int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips)
{
	unsigned int ipCount = 0;
	bool shouldCoUnInitialize = true;
	HRESULT hr = CoInitializeEx(nullptr, COINIT_MULTITHREADED);
	
	if(hr == RPC_E_CHANGED_MODE) // this is considered as failed call, and only successfull calls should have couninitiallize
	{
		sprintf_s(logMsg, msgSize, "CoInitializeEx returned RPC_E_CHANGED_MODE. code: 0x%X", hr);
		AquaLogging::Log(warn, logMsg);
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
			sprintf_s(logMsg, msgSize, "Exception in CountOrRetrieveSecondaryIps");
			AquaLogging::Log(err, logMsg);
		}

		if (shouldCoUnInitialize)
		{
			CoUninitialize();
		}
	}
	else
	{
		sprintf_s(logMsg, msgSize, "Coinitialize failed with code 0x%X", hr);
		AquaLogging::Log(err, logMsg);
	}
	return ipCount;
}

void ParseSubnetAndGatewayIp(char *xmlResponse, char **subnetIp, char **gatewayIp)
{
	bool shouldCoUnInitialize = true;
	HRESULT hr = CoInitializeEx(nullptr, COINIT_MULTITHREADED);

	if (hr == RPC_E_CHANGED_MODE) // this is considered as failed call, and only successfull calls should have couninitiallize
	{
		sprintf_s(logMsg, msgSize, "CoInitializeEx returned RPC_E_CHANGED_MODE. code: 0x%X", hr);
		AquaLogging::Log(warn, logMsg);
		shouldCoUnInitialize = false;
	}

	if (hr != E_INVALIDARG &&
		hr != E_OUTOFMEMORY &&
		hr != E_UNEXPECTED)
	{
		try
		{
			RetrieveSubnetGatewayIp(xmlResponse, subnetIp, gatewayIp);
		}
		catch (...)
		{
			sprintf_s(logMsg, msgSize, "Exception in RetrieveSubnetGatewayIp");
			AquaLogging::Log(err, logMsg);
		}

		if (shouldCoUnInitialize)
		{
			CoUninitialize();
		}
	}
	else
	{
		sprintf_s(logMsg, msgSize, "Coinitialize failed with code 0x%X\n", hr);
		AquaLogging::Log(err, logMsg);
	}
}
} } } } }
#endif