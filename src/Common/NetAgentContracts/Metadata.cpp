/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

	Metadata.cpp

Abstract:

	Implement methods of Metadata class.

Environment:

	User Mode - Win32

--*/

#include "Metadata.h"
#include "Logging.h"

using namespace std;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

Metadata::Metadata()
/*++

Routine Description:

	Default constructor for Metadata.

Arguments:

	JsonObject = Json object received as part of the request from the VM.

Return Value:

	None.

--*/
{
	InitializeFieldIdMap();
}

Metadata::Metadata(json::object& JsonObject)
/*++

Routine Description:

	Initialize Metadata with the Json object received from request.

Arguments:

	JsonObject = Json object received as part of the request from the VM.

Return Value:

	None.

--*/
{
	InitializeFieldIdMap();
	SetMembersFromJson(JsonObject);
}

void Metadata::InitializeFieldIdMap()
/*++

Routine Description:

	Initialize the report field FieldName to key map.

Arguments:

	None.

Return Value:

	None.

--*/
{
	MetadataFieldIdMap[LOCATION_STR] = MetadataEnum::MetadataKey::Location;
	MetadataFieldIdMap[VMNAME_STR] = MetadataEnum::MetadataKey::VMName;
	MetadataFieldIdMap[OFFER_STR] = MetadataEnum::MetadataKey::Offer;
	MetadataFieldIdMap[OSTYPE_STR] = MetadataEnum::MetadataKey::OSType;
	MetadataFieldIdMap[PLACEMENT_GROUPID_STR] = MetadataEnum::MetadataKey::PlacementGroupID;
	MetadataFieldIdMap[PLATFORM_FAULT_DOMAIN_STR] = MetadataEnum::MetadataKey::PlatformFaultDomain;
	MetadataFieldIdMap[PLATFORM_UPDATE_DOMAIN_STR] = MetadataEnum::MetadataKey::PlatformUpdateDomain;
	MetadataFieldIdMap[PUBLISHER_STR] = MetadataEnum::MetadataKey::Publisher;
	MetadataFieldIdMap[RESOURCE_GROUP_NAME_STR] = MetadataEnum::MetadataKey::ResourceGroupName;
	MetadataFieldIdMap[SKU_STR] = MetadataEnum::MetadataKey::Sku;
	MetadataFieldIdMap[SUBSCRIPTIONID_STR] = MetadataEnum::MetadataKey::SubscriptionID;
	MetadataFieldIdMap[TAGS_STR] = MetadataEnum::MetadataKey::Tags;
	MetadataFieldIdMap[OSVERSION_STR] = MetadataEnum::MetadataKey::OSVersion;
	MetadataFieldIdMap[KERNEL_VERSION_STR] = MetadataEnum::MetadataKey::KernelVersion;
	MetadataFieldIdMap[VMID_STR] = MetadataEnum::MetadataKey::VMID;
	MetadataFieldIdMap[VMSIZE_STR] = MetadataEnum::MetadataKey::VMSize;
}

void
Metadata::SetFields(
	wstring Name,
	json::value Value
)
/*++

Routine Description:

	Sets Metadata Class fields

Arguments:

	Name - Name of the field.

	Value - Json value to be associated with the field.

Return Value:

	None.

--*/
{
	// Prevent memory exhaustion attack, since we have a [] operator
	// in the switch statement below.
	if (MetadataFieldIdMap.find(Name) == MetadataFieldIdMap.end())
	{
		LOGERRW(L"Unknown key:{%s}", Name.c_str());
		return;
	}

	switch (MetadataFieldIdMap[Name])
	{
	case MetadataEnum::MetadataKey::Location:
		Location = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::VMName:
		VMName = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::Offer:
		Offer = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::OSType:
		OSType = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::PlacementGroupID:
		PlacementGroupID = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::PlatformFaultDomain:
		PlatformFaultDomain = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::PlatformUpdateDomain:
		PlatformUpdateDomain = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::Publisher:
		Publisher = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::ResourceGroupName:
		ResourceGroupName = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::Sku:
		Sku = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::SubscriptionID:
		SubscriptionID = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::Tags:
		Tags = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::OSVersion:
		OSVersion = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::KernelVersion:
		KernelVersion = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::VMID:
		VMID = Value.as_string();
		break;

	case MetadataEnum::MetadataKey::VMSize:
		VMSize = Value.as_string();
		break;
	}
}

void Metadata::Print()
/*++

Routine Description:

    Print each member of Metadata.

Arguments:

    None.

Return Value:

    None.

--*/
{
	wstring s = L"Metadata: {";
	s += L"Location: " + Location + L"\n";
	s += L"VMName: " + VMName + L"\n";
	s += L"Offer: " + Offer + L"\n";
	s += L"PlacementGroupID: " + PlacementGroupID + L"\n";
	s += L"PlatformFaultDomain:" + PlatformFaultDomain + L"\n";
	s += L"PlatformUpdateDomain:" + PlatformUpdateDomain + L"\n";
	s += L"Publisher:" + Publisher + L"\n";
	s += L"ResourceGroupName:" + ResourceGroupName + L"\n";
	s += L"Sku:" + Sku + L"\n";
	s += L"SubscriptionID:" + SubscriptionID + L"\n";
	s += L"Tags:" + Tags + L"\n";
	s += L"OSVersion:" + OSVersion + L"\n";
	s += L"KernelVersion:" + KernelVersion + L"\n";
	s += L"VMID:" + VMID +L"\n";
	s += L"VMSize:" + VMSize +L"\n";

	LOGINFOW(L"%s}", s.c_str());
}