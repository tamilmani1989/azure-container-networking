/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

Metadata.h

Abstract:

Header to define the metadata of reports.

Environment:

User Mode - Win32

--*/

#pragma once

#include <string>
#include <map>
#include <cpprest/json.h>
#include "Json.h"

using namespace std;
using namespace web;

#define METADATA_STR L"compute"
#define Location_STR L"Location"
#define VMNAME_STR L"VMName"
#define OFFER_STR L"Offer"
#define OSTYPE_STR L"OSType"
#define PLACEMENT_GROUPID_STR L"PlacementGroupID"
#define PLATFORM_FAULT_DOMAIN_STR L"PlatformFaultDomain"
#define PLATFORM_UPDATE_DOMAIN_STR L"PlatformUpdateDomain"
#define PUBLISHER_STR L"Publisher"
#define RESOURCE_GROUP_NAME_STR L"ResourceGroupName"
#define SKU_STR L"Sku"
#define SUBSCRIPTIONID_STR L"SubscriptionID"
#define TAGS_STR L"Tags"
#define OSVERSION_STR L"OSVersion"
#define KERNEL_VERSION_STR L"KernelVersion"
#define VMID_STR L"VMID"
#define VMSIZE_STR L"VMSize"

//
// Enums for fields in Metadata class.
//
namespace MetadataEnum
{
	enum MetadataKey
	{
		Location,
		VMName,
		Offer,
		OSType,
		PlacementGroupID,
		PlatformFaultDomain,
		PlatformUpdateDomain,
		Publisher,
		ResourceGroupName,
		Sku,
		SubscriptionID,
		Tags,
		OSVersion,
		KernelVersion,
		VMID,
		VMSize
	};
}

static map<wstring, MetadataEnum::MetadataKey> MetadataFieldIdMap;

class Metadata : public Microsoft::Azure::Networking::Aqua::ClientLibrary::Json<Metadata>
{

public:

	// Default constructor.
	Metadata();

	// Metadata Constructor from JsonObject.
	Metadata(json::object& JsonObject);

	// Azure Region the plugin is running in.
	wstring Location;

	// Name of the VM.
	wstring VMName;

	// Offer information for the VM image. 
	// This value is only present for images deployed from Azure image gallery.
	wstring Offer;

	// Linux or Windows.
	wstring OSType;

	// Placement Group of the virtual machine scale set.
	wstring PlacementGroupID;

	// Fault domain the plugin is running in.
	wstring PlatformFaultDomain;

	// Update domain the plugin is running in.
	wstring PlatformUpdateDomain;

	// Publisher of the VM image.
	wstring Publisher;

	// Resource group for the plugin.
	wstring ResourceGroupName;

	// Specific SKU for the VM image.
	wstring Sku;

	// Azure subscription for the Virtual Machine.
	wstring SubscriptionID;

	// Tags for the Virtual Machine.
	wstring Tags;

	// Version of the VM image.
	wstring OSVersion;

	// Kernel version of the VM image.
	wstring KernelVersion;

	// Unique identifier for the VM.
	wstring VMID;

	// Size of the VM.
	wstring VMSize;

	// Initialize metadata fields.
	static void InitializeFieldIdMap();

	// Fill metadata member variables.
	void SetFields(wstring FieldName, json::value JsonValue);

	// Print metadata.
	void Print();
};