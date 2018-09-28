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
#define LOCATION_STR L"location"
#define VMNAME_STR L"name"
#define OFFER_STR L"offer"
#define OSTYPE_STR L"osType"
#define PLACEMENT_GROUPID_STR L"placementGroupId"
#define PLATFORM_FAULT_DOMAIN_STR L"platformFaultDomain"
#define PLATFORM_UPDATE_DOMAIN_STR L"platformUpdateDomain"
#define PUBLISHER_STR L"publisher"
#define RESOURCE_GROUP_NAME_STR L"resourceGroupName"
#define SKU_STR L"sku"
#define SUBSCRIPTIONID_STR L"subscriptionId"
#define TAGS_STR L"tags"
#define OSVERSION_STR L"version"
#define KERNEL_VERSION_STR L"KernelVersion"
#define VMID_STR L"vmId"
#define VMSIZE_STR L"vmSize"

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