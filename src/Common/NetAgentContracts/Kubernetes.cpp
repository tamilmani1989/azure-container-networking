/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

	Kubernetes.cpp

Abstract:

	 Implement Kubernetes related functions.

Environment:

	User Mode - Win32

--*/

#include "Kubernetes.h"
#include "Logging.h"

using namespace std;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

ClusterState::ClusterState()
/*++

Routine Description:

	Default constructor for ClusterState.

Arguments:

	None.

Return Value:

	None.

--*/
{
	InitializeFieldIdMap();
}

ClusterState::ClusterState(json::object & JsonObject)
/*++

Routine Description:

	Initialize ClusterState from JsonObject received from request.

Arguments:

	JsonObject = Json object received as part of the request from the VM.

Return Value:

	None.

--*/
{
	InitializeFieldIdMap();
	SetMembersFromJson(JsonObject);
}

void ClusterState::InitializeFieldIdMap()
/*++

Routine Description:

	Initialize the report field FieldName to key map.

Arguments:

	None.

Return Value:

	None.

--*/
{
	ClusterStateFieldIdMap[POD_COUNT_STR] = ClusterStateEnum::ClusterStateKey::PodCount;
	ClusterStateFieldIdMap[NS_COUNT_STR] = ClusterStateEnum::ClusterStateKey::NsCount;
	ClusterStateFieldIdMap[NWPOLICY_COUNT_STR] = ClusterStateEnum::ClusterStateKey::NwPolicyCount;
}

void ClusterState::SetFields(wstring Name, json::value Value)
/*++

Routine Description:

	Sets ClusterState fields.

Arguments:

	Name - Name of the field.

	Value - Json value to be associated with the field.

Return Value:

	None.

--*/
{
	// Prevent memory exhaustion attack, since we have a [] operator
	// in the switch statement below.
	if (ClusterStateFieldIdMap.find(Name) == ClusterStateFieldIdMap.end())
	{
		wprintf(L"Unknown key:{%s}", Name.c_str());
		return;
	}

	switch (ClusterStateFieldIdMap[Name])
	{
	case ClusterStateEnum::ClusterStateKey::PodCount:
		PodCount = Value.as_integer();
		break;

	case ClusterStateEnum::ClusterStateKey::NsCount:
		NsCount = Value.as_integer();
		break;

	case ClusterStateEnum::ClusterStateKey::NwPolicyCount:
		NwPolicyCount = Value.as_integer();
		break;
	}
}

void ClusterState::Print()
/*++

Routine Description:

	Print each member of ClusterState.

Arguments:

	None.

Return Value:

	None.

--*/
{
	wstring s = L"ClusterState: {";
	s += L"PodCount: " + to_wstring(PodCount) + L"\n";
	s += L"NsCount: " + to_wstring(NsCount) + L"\n";
	s += L"NwPolicyCount" + to_wstring(NwPolicyCount) + L"\n";

	LOGINFOW(L"%s}", s.c_str())
}
