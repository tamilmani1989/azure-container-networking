/*++

Copyright (c) 2018  Microsoft Corporation

Module Name:

	NPMReport.cpp

Abstract:

	Implementation of Azure NPM report functions.

Environment:

	User Mode - Win32

--*/

#include "NPMReport.h"
#include "Logging.h"

using namespace std;
using namespace web;
using namespace http;
using namespace json;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

NPMReport::NPMReport(json::object& JsonObject)
/*++

Routine Description:

	Initialize the NPM report with the Json object received from request.

Arguments:

	JsonObject = Json object received as part of the request from the VM.

Return Value:

	None.

--*/
{
	SetMembersFromJson(JsonObject);
}

void
NPMReport::InitializeFieldIdMap()
/*++

Routine Description:

	Initialize the report field FieldName to key map.

Arguments:

	None.

Return Value:

	None.

--*/
{
	NPMReportFieldIdMap[IS_NEW_INSTANCE_STR] = NPMEnum::NPMReportKey::IsNewInstance;
	NPMReportFieldIdMap[CLUSTERID] = NPMEnum::NPMReportKey::ClusterID;
	NPMReportFieldIdMap[NODENAME_STR] = NPMEnum::NPMReportKey::NodeName;
	NPMReportFieldIdMap[INSTANCE_NAME_STR] = NPMEnum::NPMReportKey::InstanceName;
	NPMReportFieldIdMap[NPM_VERSION_STR] = NPMEnum::NPMReportKey::NpmVersion;
	NPMReportFieldIdMap[KUBERNETES_VERSION_STR] = NPMEnum::NPMReportKey::KubernetesVersion;
	NPMReportFieldIdMap[ERROR_MESSAGE_STR] = NPMEnum::NPMReportKey::ErrorMessage;
	NPMReportFieldIdMap[EVENT_MESSAGE_STR] = NPMEnum::NPMReportKey::EventMessage;
	NPMReportFieldIdMap[UPTIME_STR] = NPMEnum::NPMReportKey::UpTime;
	NPMReportFieldIdMap[CLUSTER_STATE_STR] = NPMEnum::NPMReportKey::ClusterState;
	NPMReportFieldIdMap[METADATA_STR] = NPMEnum::NPMReportKey::Metadata;
}

void NPMReport::SetFields(wstring Name, json::value Value)
/*++

Routine Description:

	Sets NPMReport fields.

Arguments:

	Name - Name of the field.

	Value - Json value to be associated with the field.

Return Value:

	None.

--*/
{
	// Prevent memory exhaustion attack, since we have a [] operator
	// in the switch statement below.
	if (NPMReportFieldIdMap.find(Name) == NPMReportFieldIdMap.end())
	{
		wprintf(L"Unknown key:{%s}", Name.c_str());
		return;
	}

	switch (NPMReportFieldIdMap[Name])
	{
	case NPMEnum::NPMReportKey::IsNewInstance:
		IsNewInstance = Value.as_bool();
		break;
	case NPMEnum::NPMReportKey::ClusterID:
		ClusterId = Value.as_string();
		break;
	case NPMEnum::NPMReportKey::NodeName:
		NodeName = Value.as_string();
		break;
	case NPMEnum::NPMReportKey::InstanceName:
		InstanceName = Value.as_string();
		break;
	case NPMEnum::NPMReportKey::NpmVersion:
		NpmVersion = Value.as_string();
		break;
	case NPMEnum::NPMReportKey::KubernetesVersion:
		KubernetesVersion = Value.as_string();
		break;
	case NPMEnum::NPMReportKey::ErrorMessage:
		ErrorMessage = Value.as_string();
		break;
	case NPMEnum::NPMReportKey::EventMessage:
		EventMessage = Value.as_string();
		break;
	case NPMEnum::NPMReportKey::UpTime:
		UpTime = Value.as_string();
		break;
	case NPMEnum::NPMReportKey::ClusterState:
		ClusterState = ClusterState::ClusterState(Value.as_object());
		break;
	case NPMEnum::NPMReportKey::Metadata:
		Metadata = Metadata::Metadata(Value.as_object());
		break;
	}
}

void NPMReport::Print()
/*++

Routine Description:

	Print each member of NPMReport.

Arguments:

	None.

Return Value:

	None.

--*/
{
	wstring s = L"NPMReport: {";
	s += L"IsNewInstance: " + ConvertBoolToWstring(IsNewInstance) +L"\n";
	s += L"ClusterID: " + ClusterId +L"\n";
	s += L"NODENAME: " + NodeName +L"\n";
	s += L"InstanceName: " + InstanceName +L"\n";
	s += L"NpmVersion: " + NpmVersion +L"\n";
	s += L"KubernetesVersion: " + KubernetesVersion +L"\n";
	s += L"ErrorMessage: " + ErrorMessage +L"\n";
	s += L"EventMessage: " + EventMessage +L"\n";
	s += L"UpTime: " + UpTime +L"\n";
	ClusterState.Print();
	Metadata.Print();

	LOGINFOW(L"%s}", s.c_str());
}