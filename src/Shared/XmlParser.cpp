/****************************** Module Header ******************************\
* Module Name:  XmlParser.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* HElper to parse the xml files.
*
\***************************************************************************/

#include <XmlParser.h>
#include "Utilities.h"
#include <atlbase.h>

// Macro that calls a COM method returning HRESULT value.
#define CHK_HR(stmt)        do { hr=(stmt); if (FAILED(hr)) goto CleanUp; } while(0)

// Macro to verify memory allcation.
#define CHK_ALLOC(p)        do { if (!(p)) { hr = E_OUTOFMEMORY; goto CleanUp; } } while(0)

// Macro that releases a COM object if not NULL.
#define SAFE_RELEASE(p)     do { if ((p)) { (p)->Release(); (p) = nullptr; } } while(0)\

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

void RetrieveEntryPointFromManifest(wchar_t* xmlFilePath, PluginMetadata *pluginData)
/*++

Routine Description:
	This function parses XML file and populates pluginData object

Arguments:
	xmlFilePath - xml file path
	pluginData - Object contains information about plugin

Return Value:

--*/
{
	HRESULT hr = S_OK;
	IXMLDOMDocument *pXMLDom = nullptr;
	IXMLDOMParseError *pXMLErr = nullptr;
	IXMLDOMNodeList* pXMLNodeList = nullptr;
	IXMLDOMNode* pXMLNode = nullptr;
	IXMLDOMNode* pXMLChildNode = nullptr;
	IXMLDOMNodeList* pXMLChildNodeList = nullptr;
	IXMLDOMNodeList* pXMLEntryPointList = nullptr;
	IXMLDOMNodeList* pXMLEntryPointChildList = nullptr;
	IXMLDOMNode* pXMLEntryPointNode = nullptr;
	IXMLDOMNode* pXMLEntryPointChildNode = nullptr;

	BSTR bstrXML = nullptr;
	BSTR bstrErr = nullptr;
	BSTR nodeName = nullptr;
	BSTR childNodeName = nullptr;
	BSTR entryPointNodeName = nullptr;
	BSTR entryPointChildNodeName = nullptr;
	BSTR programName = nullptr;
	BSTR restartOnExitTemp = nullptr;
	BSTR blockRoleTemp = nullptr;
	VARIANT_BOOL varStatus;
	VARIANT varFileName;
	VariantInit(&varFileName);

	CHK_HR(CreateAndInitDOM(&pXMLDom));
	wchar_t szMessage[MAX_PATH];
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
		L"Going to load xml from %s\n", xmlFilePath);
	AquariusUtilities::WriteEventLogEntry(szMessage);
	// XML file name to load
	CHK_HR(VariantFromString(xmlFilePath, varFileName));
	CHK_HR(pXMLDom->load(varFileName, &varStatus));

	if (varStatus == VARIANT_TRUE)
	{
		CHK_HR(pXMLDom->get_xml(&bstrXML));
		// printf("XML DOM loaded from stocks.xml:\n%S\n", bstrXML);
		AquariusUtilities::WriteEventLogEntry(bstrXML);

		CHK_HR(pXMLDom->get_childNodes(&pXMLNodeList));
		long lCount;
		pXMLNodeList->get_length(&lCount);
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
			L"Retrieved %d nodes from %s. \n", lCount, xmlFilePath);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		for(int i = 0; i < lCount; i++)
		{
			CHK_HR(pXMLNodeList->get_item(i, &pXMLNode));
			// do something based on node
			CHK_HR(pXMLNode->get_nodeName(&nodeName));
			StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
				L"Node %i -> Name: %s. \n", i, nodeName);
			AquariusUtilities::WriteEventLogEntry(szMessage);
		
			if(wcscmp(nodeName, L"PluginManifest") == 0)
			{
				StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
					L"Located plugin manifest in %s. \n", xmlFilePath);
				AquariusUtilities::WriteEventLogEntry(szMessage);
				
				IXMLDOMNamedNodeMap*  pXMLNamedNodeMap;
				IXMLDOMNode *pIXMLDOMNode = NULL;  
				VARIANT varValue;
				pXMLNode->get_attributes(&pXMLNamedNodeMap);
				HRESULT hr = pXMLNamedNodeMap->getNamedItem(L"Name", &pIXMLDOMNode);
				
				if(SUCCEEDED(hr) && pIXMLDOMNode)  
				{  
					pIXMLDOMNode->get_nodeValue(&varValue);  
					pIXMLDOMNode->Release();  
					pIXMLDOMNode = NULL;  
					
					if (pluginData->name != nullptr) 
					{
						StringCchPrintf(szMessage, ARRAYSIZE(szMessage),
							L"Going to Release name %s", pluginData->name);
						AquariusUtilities::WriteEventLogEntry(szMessage);
						delete[] pluginData->name;
						pluginData->name = nullptr;
					}
					size_t len = wcslen(varValue.bstrVal)+1;
					pluginData->name = new wchar_t[len];
					wcscpy_s(pluginData->name, len, varValue.bstrVal);

					StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
						L"Plugin Name:%s\n", pluginData->name);
					AquariusUtilities::WriteEventLogEntry(szMessage);
				}  
				else
				{
					StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
						L"Plugin Name retrieval failed %x", hr);
					AquariusUtilities::WriteEventLogEntry(szMessage);					
				}

				pXMLNamedNodeMap->Release();  
				pXMLNamedNodeMap = NULL; 

				CHK_HR(pXMLNode->get_childNodes(&pXMLChildNodeList));
				long childCount;
				CHK_HR(pXMLChildNodeList->get_length(&childCount));
				
				for (int j = 0; j < childCount; j++)
				{
					CHK_HR(pXMLChildNodeList->get_item(j, &pXMLChildNode));
					CHK_HR(pXMLChildNode->get_nodeName(&childNodeName));
					StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
						L"File %s -> Node %s -> ChildNode -> %s. \n", 
						xmlFilePath, nodeName, childNodeName);
					AquariusUtilities::WriteEventLogEntry(szMessage);

					if ((wcscmp(childNodeName, L"BlockRoleStartOnUnhealthy") == 0))
					{
						CHK_HR(pXMLChildNode->get_text(&blockRoleTemp));
						StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
							L"BlockRoleStartOnUnhealthy in %s is %s \n", xmlFilePath, blockRoleTemp);
						AquariusUtilities::WriteEventLogEntry(szMessage);
						pluginData->blockRole = 0;
						if(wcscmp(blockRoleTemp, L"True")==0)
						{
							pluginData->blockRole = 1;
						}
						SysFreeString(blockRoleTemp);
						blockRoleTemp = nullptr;

					}
					else if (wcscmp(childNodeName, L"CodePackage") == 0)
					{						
						StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
							L"Located first code package %s. \n", xmlFilePath);
						AquariusUtilities::WriteEventLogEntry(szMessage);
						CHK_HR(pXMLChildNode->get_childNodes(&pXMLEntryPointList));
						long entryPointCount;
						CHK_HR(pXMLEntryPointList->get_length(&entryPointCount));
						
						for (int k = 0; k < entryPointCount; k++)
						{
							CHK_HR(pXMLEntryPointList->get_item(k, &pXMLEntryPointNode));
							CHK_HR(pXMLEntryPointNode->get_nodeName(&entryPointNodeName));
							
							if(wcscmp(entryPointNodeName, L"EntryPoint") == 0)
							{
								StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
									L"Located an entry point for code package in %s. \n", 
									xmlFilePath);
								AquariusUtilities::WriteEventLogEntry(szMessage);
								CHK_HR(pXMLEntryPointNode->get_childNodes(&pXMLEntryPointChildList));
								long entryPointChildrenCount;
								CHK_HR(pXMLEntryPointChildList->get_length(&entryPointChildrenCount));
								
								for (int l = 0; l<entryPointChildrenCount; l++)
								{
									StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
										L"Located a program for entry point in %s.... \n", xmlFilePath);
									AquariusUtilities::WriteEventLogEntry(szMessage);
									CHK_HR(pXMLEntryPointChildList->get_item(l, &pXMLEntryPointChildNode));
									CHK_HR(pXMLEntryPointChildNode->get_nodeName(&entryPointChildNodeName));
									
									StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
										L"childname of entrypoint %s. \n", entryPointChildNodeName);
									AquariusUtilities::WriteEventLogEntry(szMessage);

									if(wcscmp(entryPointChildNodeName, L"Program") == 0)
									{
										CHK_HR(pXMLEntryPointChildNode->get_text(&programName));
										StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
											L"Program name in %s is %s.. \n", xmlFilePath, programName);
										AquariusUtilities::WriteEventLogEntry(szMessage);
										
										if( pluginData->entryPoint != nullptr)
										{
											delete[] pluginData->entryPoint;
										}

										size_t len = wcsnlen(programName, MAX_PATH) + 1;
										pluginData->entryPoint = new wchar_t[len];
                                        wcscpy_s(pluginData->entryPoint, len, programName);
										SysFreeString(programName);
										programName = nullptr;
									}
									else if(wcscmp(entryPointChildNodeName, L"RestartOnExit") == 0)
									{
										CHK_HR(pXMLEntryPointChildNode->get_text(&restartOnExitTemp));
										StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
											L"Restart Status in %s is %s \n", xmlFilePath, restartOnExitTemp);
										AquariusUtilities::WriteEventLogEntry(szMessage);
										pluginData->restartOnExit = 1;
										
										if(wcscmp(restartOnExitTemp, L"False")==0)
										{
											pluginData->restartOnExit = 0;
										}

										SysFreeString(restartOnExitTemp);
										restartOnExitTemp = nullptr;
									}

									else if(wcscmp(entryPointChildNodeName, L"Arguments") == 0)
									{
										if( pluginData->arguments != nullptr)
										{
											SysFreeString(pluginData->arguments);
											pluginData->arguments=nullptr;
										}

										CHK_HR(pXMLEntryPointChildNode->get_text(&pluginData->arguments));
										StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
											L"Arguments %s \n", xmlFilePath, pluginData->arguments);
										AquariusUtilities::WriteEventLogEntry(szMessage);
									}

									SAFE_RELEASE(pXMLEntryPointChildNode);
									SysFreeString(entryPointChildNodeName);
									entryPointChildNodeName = nullptr;
								}

								SAFE_RELEASE(pXMLEntryPointChildList);
							}

							SAFE_RELEASE(pXMLEntryPointNode);
							SysFreeString(entryPointNodeName);
							entryPointNodeName = nullptr;
						}

						SAFE_RELEASE(pXMLEntryPointList);
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
		
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), 
			L"Failed to load/parse the xml document from location: %s. Error: %S\n", 
			xmlFilePath, bstrErr);
		AquariusUtilities::WriteEventLogEntry(szMessage);
	}

CleanUp:
	SAFE_RELEASE(pXMLDom);
	SAFE_RELEASE(pXMLErr);
	SAFE_RELEASE(pXMLChildNode);
	SAFE_RELEASE(pXMLChildNodeList);
	SAFE_RELEASE(pXMLEntryPointNode);
	SAFE_RELEASE(pXMLEntryPointChildNode);
	SAFE_RELEASE(pXMLEntryPointChildList);
	SAFE_RELEASE(pXMLEntryPointList);
	SAFE_RELEASE(pXMLNode);
	SAFE_RELEASE(pXMLNodeList);

	SysFreeString(bstrXML);
	SysFreeString(bstrErr);	
	SysFreeString(nodeName);	
	SysFreeString(childNodeName);
	SysFreeString(entryPointNodeName);
	SysFreeString(entryPointChildNodeName);
	SysFreeString(programName);
	SysFreeString(restartOnExitTemp);
	SysFreeString(blockRoleTemp);

	VariantClear(&varFileName);
}

bool IsValidEntryPoint(wchar_t **entryPoint)
{
	wchar_t *ep = (*entryPoint);
	if(ep != nullptr)
	{
		if(wcsstr(ep, L".."))
		{
			return 0;
		}
		return 1;
	}
	return 0;
}

int XmlParser::ParseManifest(wchar_t* xmlFilePath, PluginMetadata *pluginData)
{
	HRESULT hr = CoInitialize(nullptr);
	
	if (SUCCEEDED(hr))
	{
		try 
		{
			RetrieveEntryPointFromManifest(xmlFilePath, pluginData);
		}
		catch(...)
		{
			CoUninitialize();
			throw;
		}
	
		CoUninitialize();
	
		if (!IsValidEntryPoint(&pluginData->entryPoint))
		{
			wchar_t errMsg[256];
			StringCchPrintf(errMsg, ARRAYSIZE(errMsg),
				L"Invalid entry point %s Relative/empty paths not allowed in entry point.", pluginData->entryPoint);
			AquariusUtilities::WriteEventLogEntry(errMsg);
			throw errMsg;
		}
	
		return 1;
	}

	return 0;
}