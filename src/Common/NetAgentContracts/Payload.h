/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

    Payload.h

Abstract:

    Header to define the Reports Payload functions.

Environment:

    User Mode - Win32

--*/

#pragma once

#include "DNCReport.h"
#include "CNIReport.h"
#include "NPMReport.h"
#include "CNSReport.h"

#define DNC_REPORTS L"DNCReports"
#define CNI_REPORTS L"CNIReports"
#define NPM_REPORTS L"NPMReports"
#define CNS_REPORTS L"CNSReports"

//
// Enums for fields in Reports Payload
//
namespace PayloadEnum
{
    enum PayloadKey
    {
        DNCReports,
        CNIReports,
        NPMReports,
        CNSReports
    };
}

static map<wstring, PayloadEnum::PayloadKey> PayloadFieldIdMap;

// Reports Payload
class Payload : public Microsoft::Azure::Networking::Aqua::ClientLibrary::Json<Payload>{

public:

    Payload(json::object& JsonObject);

    // DNCReports to be processed
    vector<DNCReport> DNCReports;

    // CNIReports to be processed
    vector<CNIReport> CNIReports;

    // NPMReports to be processed
    vector<NPMReport> NPMReports;

    // CNSReports to be processed
    vector<CNSReport> CNSReports;

    static void InitializeFieldIdMap();

    void SetFields(wstring FieldName, json::value JsonValue);

    void Print();
};