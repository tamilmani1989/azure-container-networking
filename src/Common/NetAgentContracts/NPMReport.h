/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

	NPMReport.h

Abstract:

	Header to define the Azure Network Policy Manager report functions.

Environment:

	User Mode - Win32

--*/

#pragma once

#include <string>
#include <map>
#include <cpprest/json.h>
#include "PluginReport.h"
#include "Metadata.h"
#include "Kubernetes.h"

using namespace std;
using namespace web;

#define IS_NEW_INSTANCE_STR L"IsNewInstance"
#define CLUSTERID L"ClusterID"
#define NODENAME_STR L"NodeName"
#define INSTANCE_NAME_STR L"InstanceName"
#define NPM_VERSION_STR L"NpmVersion"
#define KUBERNETES_VERSION_STR L"KubernetesVersion"
#define ERROR_MESSAGE_STR L"ErrorMessage"
#define EVENT_MESSAGE_STR L"EventMessage"
#define UPTIME_STR L"UpTime"
//
// Enums for fields in the NPM report.
//
namespace NPMEnum 
{
	enum NPMReportKey 
	{
		IsNewInstance,
		ClusterID,
		NodeName,
		InstanceName,
		NpmVersion,
		KubernetesVersion,
		ErrorMessage,
		EventMessage,
		UpTime,
		ClusterState,
		Metadata
	};
}

static map<std::wstring, NPMEnum::NPMReportKey> NPMReportFieldIdMap;

// NPM Report
class NPMReport : public PluginReport {

public:

	NPMReport(json::object& JsonObject);

	// Flag to identify number of VMs running Azure NPM.
	bool IsNewInstance;

	// Unique identifier for the cluster.
	wstring ClusterId;

	// Name of the Node.
	wstring NodeName;

	// Name of the Azure NPM Instance.
	wstring InstanceName;

	// Version of Azure NPM.
	wstring NpmVersion;

	// Version of Kubernetes.
	wstring KubernetesVersion;

	// Error message.
	wstring ErrorMessage;

	// Event message.
	wstring EventMessage;

	// The Azure NPM instance's running time.
	wstring UpTime;

	// State of the kubernetes cluster.
	ClusterState ClusterState;

	// Metadata for Azure NPM report.
	Metadata Metadata;

	static void InitializeFieldIdMap();

	void SetFields(
		wstring FieldName,
		json::value Value
	);

	void Print();
};