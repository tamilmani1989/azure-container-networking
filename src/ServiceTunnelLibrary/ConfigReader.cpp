/*++

Copyright (c) 2008  Microsoft Corporation

Module Name:

ConfigReader.cpp

Abstract:

Implements methods to retrieve values from xml file.

--*/
#include "ConfigReader.h"
#include "Logging.h"

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

HRESULT ConfigReader::GetInstanceId(
	__in XmlParserHelper* XmlParserHelper,
	std::wstring& InstanceName)
	/*++

	Routine Description:

	Helper to retrieve the instance name from the RCF.

	Arguments:

	XmlParserHelper - XmlParserHelepr object which has the RCF file loaded.

	InstanceName - Out paramter to hold the instance name retrieved from RCF.

	Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

	--*/
{
	HRESULT hr = S_OK;
	std::set<std::wstring> instanceValues;
	std::wstring incarnationNodePath;
	incarnationNodePath.append(XPATH_ROOT_START_TAG);
	incarnationNodePath.append(INCARNATION_NODE_NAME);

	hr = XmlParserHelper->QueryNodeAttributeValues(
		(LPWSTR)incarnationNodePath.c_str(),
		INSTANCE_ATTRIBUTE_NAME,
		instanceValues);
	if (hr != S_OK)
	{
		LOGERRW(L"GetInstanceId failed with error [%0x]. NodeQuery:[%s], Attribute[%s]", hr, incarnationNodePath.c_str(), INSTANCE_ATTRIBUTE_NAME);
		goto Cleanup;
	}

	LOGINFOW(
		L"NodeQuery:%s, Attribute:%s, Count:%llu.",
		incarnationNodePath.c_str(),
		INSTANCE_ATTRIBUTE_NAME,
		instanceValues.size());

	if (instanceValues.size() == 0)
	{
		hr = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
		if (hr != S_OK)
		{
			LOGERRW(L"GetInstanceId: Failed to find attribute NodeQuery:[%s], Attribute[%s]", incarnationNodePath.c_str(), INSTANCE_ATTRIBUTE_NAME);
			goto Cleanup;
		}
	}

	if (instanceValues.size() > 1)
	{
		hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
		if (hr != S_OK)
		{
			LOGERRW(L"GetInstanceId: Failed more than one attribute value. NodeQuery:[%s], Attribute[%s]", incarnationNodePath.c_str(), INSTANCE_ATTRIBUTE_NAME);
			goto Cleanup;
		}
	}

	InstanceName.clear();
	InstanceName.assign(*instanceValues.begin());

Cleanup:

	return hr;
}

HRESULT ConfigReader::GetInstanceAddress(
	__in XmlParserHelper* XmlParserHelper,
	__in std::wstring InstanceName,
	__inout std::wstring& InstanceAddress)
	/*++

	Routine Description:

	Helper to retrieve the instance address from the RCF.

	Arguments:

	XmlParserHelper - XmlParserHelper object which has the RCF file loaded.

	InstanceName - Instance name whose address to be retrieved from RCF.

	InstanceAddress - Out parameter to hold the instance address.

	Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

	--*/
{
	HRESULT hr = S_OK;
	std::set<std::wstring> instanceValues;
	std::wstring instanceAddressNodePath;
	instanceAddressNodePath.append(XPATH_ROOT_START_TAG);
	instanceAddressNodePath.append(INSTANCE_NODE_NAME);
	instanceAddressNodePath.append(L"[@");
	instanceAddressNodePath.append(ID_ATRRIBUTE_NAME);
	instanceAddressNodePath.append(L"='");
	instanceAddressNodePath.append(InstanceName);
	instanceAddressNodePath.append(L"']");

	hr = XmlParserHelper->QueryNodeAttributeValues(
		(LPWSTR)instanceAddressNodePath.c_str(),
		ADDRESS_ATTRIBUTE_NAME,
		instanceValues);
	if (hr != S_OK)
	{
		LOGERRW(L"GetInstanceAddress failed with error [%0x]. NodeQuery:[%s], Attribute[%s]", hr, instanceAddressNodePath.c_str(), ADDRESS_ATTRIBUTE_NAME);
		goto Cleanup;
	}

	LOGINFOW(
		L"NodeQuery:%s, Attribute:%s, Count:%llu.",
		instanceAddressNodePath.c_str(),
		ADDRESS_ATTRIBUTE_NAME,
		instanceValues.size());

	if (instanceValues.size() == 0)
	{
		hr = HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
		if (hr != S_OK)
		{
			LOGERRW(L"GetInstanceAddress: Failed to find attribute NodeQuery:[%s], Attribute[%s]", instanceAddressNodePath.c_str(), ADDRESS_ATTRIBUTE_NAME);
			goto Cleanup;
		}
	}

	if (instanceValues.size() > 1)
	{
		hr = HRESULT_FROM_WIN32(ERROR_INVALID_DATA);
		if (hr != S_OK)
		{
			LOGERRW(L"GetInstanceAddress: Failed more than one attribute value. NodeQuery:[%s], Attribute[%s]", instanceAddressNodePath.c_str(), ADDRESS_ATTRIBUTE_NAME);
			goto Cleanup;
		}
	}

	InstanceAddress.clear();
	InstanceAddress.assign(*instanceValues.begin());

Cleanup:

	return hr;
}

HRESULT ConfigReader::GetServiceTunnelIpv6Addresses(
	__in XmlParserHelper* XmlParserHelper,
	__in std::wstring InstanceName,
	__inout std::set<std::wstring>& ServiceTunnelIpv6Addresses)
	/*++

	Routine Description:

	Helper to retrieve the service tunnel Ipv6 addresses from the RCF.

	Arguments:

	XmlParserHelper - XmlParserHelper object which has the RCF file loaded.

	InstanceName - Instance name whose address to be retrieved from RCF.

	InstanceAddress - Out parameter to hold the service tunnel Ipv6 addresses.

	Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

	--*/
{
	HRESULT hr = S_OK;
	std::set<std::wstring> instanceValues;
	std::wstring serviceTunnelAddressNodePath;
	std::set<std::wstring>::iterator instanceValuesIter;

	serviceTunnelAddressNodePath.append(XPATH_ROOT_START_TAG);
	serviceTunnelAddressNodePath.append(INSTANCE_NODE_NAME);
	serviceTunnelAddressNodePath.append(XmlParserHelper->PrepareAttributeQuery(ID_ATRRIBUTE_NAME, InstanceName));
	serviceTunnelAddressNodePath.append(XPATH_PATH_SEPARATOR_TAG);
	serviceTunnelAddressNodePath.append(L"InputEndpoints");
	serviceTunnelAddressNodePath.append(XPATH_PATH_SEPARATOR_TAG);
	serviceTunnelAddressNodePath.append(ENDPOINT_NODE_NAME);

	hr = XmlParserHelper->QueryNodeAttributeValues(
		serviceTunnelAddressNodePath.c_str(),
		SERVICE_TUNNEL_IPV6_ATTRIBUTE_NAME,
		instanceValues);
	if (hr != S_OK)
	{
		LOGERRW(L"GetServiceTunnelIpv6Addresses failed with error [%0x]. NodeQuery:[%s], Attribute[%s]", hr, serviceTunnelAddressNodePath.c_str(), ADDRESS_ATTRIBUTE_NAME);
		goto Cleanup;
	}

	LOGINFOW(
		L"NodeQuery:%s, Attribute:%s, Count:%llu.",
		serviceTunnelAddressNodePath.c_str(),
		ADDRESS_ATTRIBUTE_NAME,
		instanceValues.size());

	ServiceTunnelIpv6Addresses.clear();
	for (instanceValuesIter = instanceValues.begin(); instanceValuesIter != instanceValues.end(); ++instanceValuesIter)
	{
		ServiceTunnelIpv6Addresses.insert(*instanceValuesIter);
	}

Cleanup:

	return hr;
}
