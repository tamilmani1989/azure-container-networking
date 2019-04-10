/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

    Payload.cpp

Abstract:

    Implementations of Reports Payload functions.

Environment:

    User Mode - Win32

--*/

#include "Payload.h"
#include "Logging.h"

using namespace std;
using namespace web;
using namespace http;
using namespace json;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

Payload::Payload(json::object& JsonObject)
/*++

Routine Description:

    Initialize the Payload with the Json object received from request.

Arguments:

    JsonObject = Json object received as part of the request from the VM.

Return Value:

    None.

--*/
{
    SetMembersFromJson(JsonObject);
}

void Payload::InitializeFieldIdMap()
/*++

Routine Description:

    Initialize the payload field FieldName to key map.

Arguments:

    None.

Return Value:

    None.

--*/
{
    PayloadFieldIdMap[DNC_REPORTS] = PayloadEnum::PayloadKey::DNCReports;
    PayloadFieldIdMap[CNI_REPORTS] = PayloadEnum::PayloadKey::CNIReports;
    PayloadFieldIdMap[NPM_REPORTS] = PayloadEnum::PayloadKey::NPMReports;
    PayloadFieldIdMap[CNS_REPORTS] = PayloadEnum::PayloadKey::CNSReports;
}

void Payload::SetFields(wstring Name, json::value Value)
/*++

Routine Description:

    Sets Payload fields.

Arguments:

    Name - Name of the field.

    Value - Json value to be associated with the field.

Return Value:

    None.

--*/
{
    if (Value == NULL)
    {
        LOGINFOW(L"value is null. Don't parse it");
        return;
    }

    auto tmpArr = Value.as_array();

    // Prevent memory exhaustion attack, since we have a [] operator
    // in the switch statement below.
    if (PayloadFieldIdMap.find(Name) == PayloadFieldIdMap.end())
    {
        LOGINFOW(L"Unknown key:{%s}", Name.c_str());
        return;
    }

    switch (PayloadFieldIdMap[Name])
    {
    case PayloadEnum::PayloadKey::DNCReports:
        for ( auto iter = tmpArr.begin(); iter != tmpArr.end(); ++iter ) {
            DNCReports.push_back(DNCReport::DNCReport(iter->as_object()));
        }

        break;
    case PayloadEnum::PayloadKey::CNIReports:
        for ( auto iter = tmpArr.begin(); iter != tmpArr.end(); ++iter ) {
            CNIReports.push_back(CNIReport::CNIReport(iter->as_object()));
        }

        break;
    case PayloadEnum::PayloadKey::NPMReports:
        for ( auto iter = tmpArr.begin(); iter != tmpArr.end(); ++iter ) {
            NPMReports.push_back(NPMReport::NPMReport(iter->as_object()));
        }

        break;
    case PayloadEnum::PayloadKey::CNSReports:
        for ( auto iter = tmpArr.begin(); iter != tmpArr.end(); ++iter ) {
            CNSReports.push_back(CNSReport::CNSReport(iter->as_object()));
        }

        break;
    }
}

void Payload::Print()
/*++

Routine Description:

    Print each member of Payload.

Arguments:

    None.

Return Value:

    None.

--*/
{
    for ( auto iter = DNCReports.begin(); iter != DNCReports.end(); ++iter ) iter->Print();
    for ( auto iter = CNIReports.begin(); iter != CNIReports.end(); ++iter ) iter->Print();
    for ( auto iter = NPMReports.begin(); iter != NPMReports.end(); ++iter ) iter->Print();
    for ( auto iter = CNSReports.begin(); iter != CNSReports.end(); ++iter ) iter->Print();
}
