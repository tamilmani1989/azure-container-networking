/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

	Kubernetes.h

Abstract:

	Header to define the Kubernetes related functions.

Environment:

	User Mode - Win32

--*/

#include <string>
#include <map>
#include <cpprest/json.h>
#include "Logging.h"
#include "Json.h"

using namespace std;
using namespace web;

#pragma once

#define CLUSTER_STATE_STR L"ClusterState"
#define POD_COUNT_STR L"PodCount"
#define NS_COUNT_STR L"NsCount"
#define NWPOLICY_COUNT_STR L"NwPolicyCount"

namespace ClusterStateEnum 
{
	enum ClusterStateKey 
	{
		PodCount,
		NsCount,
		NwPolicyCount
	};
}

static map<std::wstring, ClusterStateEnum::ClusterStateKey> ClusterStateFieldIdMap;

// Kubernetes cluster state
class ClusterState : public Microsoft::Azure::Networking::Aqua::ClientLibrary::Json<ClusterState>
{

public:

	ClusterState();

	ClusterState(json::object &JsonObject);

	// Number of pods in the cluster.
	int PodCount;

	// Number of namespaces in the cluster.
	int NsCount;

	// Number of network policies in the cluster.
	int NwPolicyCount;

	// Initialize metadata fields.
	static void InitializeFieldIdMap();

	// Fill metadata member variables.
	void SetFields(wstring FieldName, json::value JsonValue);

	// Print metadata.
	void Print();
};