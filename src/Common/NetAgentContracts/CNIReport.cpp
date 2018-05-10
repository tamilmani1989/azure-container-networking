/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    CNIReport.cpp

Abstract:

    Implementation of CNI report functions.

Environment:

    User Mode - Win32

--*/
//
// ------------------------------------------------------------------- Includes
//
#include "CNIReport.h"
#include "Logging.h"	
using namespace std;
using namespace web;
using namespace http;
using namespace json;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;


void
CNIReport::InitializeCNIReport(
    json::object& JsonObject
)
/*++

Routine Description:

    Initialize the CNI report with the Json object received from request.

Arguments:

    JsonObject = Json object received as part of the request from the VM.

Return Value:

    None.

--*/
{
    SetMembersFromJson(JsonObject);
}

void
CNIReport::InitializeCNIReportFieldIdMap()
/*++

Routine Description:

    Initialize the report field FieldName to key map.

Arguments:

    None.
    
Return Value:

    None.

--*/
{
    PluginReport::InitializePluginReportFieldIdMap();
    CNIReportFieldIdMap[STARTFLAG_STR] = CNIReportKey::StartFlag;
    CNIReportFieldIdMap[CNI_SUCCEEDED_STR] = CNIReportKey::CniSucceeded;
    CNIReportFieldIdMap[VNET_ADDRESS_SPACE_STR] = CNIReportKey::VnetAddressSpace;
    CNIReportFieldIdMap[INTERFACE_DETAILS_STR] = CNIReportKey::InterfaceDetails;
    CNIReportFieldIdMap[INTERFACE_TYPE_STR] = CNIReportKey::InterfaceType;
    CNIReportFieldIdMap[SUBNET_STR] = CNIReportKey::Subnet;
    CNIReportFieldIdMap[PRIMARY_CA_STR] = CNIReportKey::PrimaryCA;
    CNIReportFieldIdMap[MAC_STR] = CNIReportKey::MAC;
    CNIReportFieldIdMap[CA_TOTAL_COUNT_STR] = CNIReportKey::SecondaryCATotalCount;
    CNIReportFieldIdMap[CA_USED_COUNT_STR] = CNIReportKey::SecondaryCAUsedCount;
    CNIReportFieldIdMap[NAME_STR] = CNIReportKey::IfName;
    CNIReportFieldIdMap[ERROR_MESSAGE_STR] = CNIReportKey::IfErrorMessage;
    CNIReportFieldIdMap[BRIDGE_DETAILS_STR] = CNIReportKey::BridgeDetails;
}

void InterfaceInfo::SetInterfaceDetailsFields(
    wstring FieldName,
    json::value JsonValue
)
/*++

Routine Description:

    Sets InterfaceDetails Class fields

Arguments:

    FieldName - Name of the field.

    JsonValue - Json value to be associated with the field.

Return Value:

    None.

--*/
{
    switch (CNIReportFieldIdMap[FieldName])
    {
    case CNIReportKey::InterfaceType:
        InterfaceType.assign(JsonValue.as_string());
        break;

    case CNIReportKey::Subnet:
        Subnet.assign(JsonValue.as_string());
        break;

    case CNIReportKey::PrimaryCA:
        PrimaryCA.assign(JsonValue.as_string());
        break;

    case CNIReportKey::MAC:
        MAC.assign(JsonValue.as_string());
        break;

    case CNIReportKey::IfName:
        Name.assign(JsonValue.as_string());
        break;

    case CNIReportKey::SecondaryCATotalCount:
        SecondaryCATotalCount = JsonValue.as_integer();
        break;

    case CNIReportKey::SecondaryCAUsedCount:
        SecondaryCAUsedCount = JsonValue.as_integer();
        break;

    case CNIReportKey::IfErrorMessage:
        ErrorMessage.assign(JsonValue.as_string());
        break;

    default:
        LOGINFOW(L"InterfaceDetails Unknown key:{%s}\n", FieldName.c_str());
    }
}

void 
CNIReport::SetCNIReportField(
    wstring FieldName, 
    json::value JsonValue
)
/*++

Routine Description:

    Set JsonValue in report field.

Arguments:

    FieldName - Name of the field.

    JsonValue - Json value to be associated with the field.

Return Value:

    None.

--*/
{

    if (!FindFieldFromBaseClass(FieldName, JsonValue))
    {
        switch (CNIReportFieldIdMap[FieldName])
        {
        case CNIReportKey::StartFlag:
            StartFlag = JsonValue.as_bool();
            break;
        
        case CNIReportKey::CniSucceeded:
            CniSucceeded = JsonValue.as_bool();
            break;
        
        case CNIReportKey::VnetAddressSpace:
        {
            break;
        }

        case CNIReportKey::InterfaceDetails:
        {
            json::object JsonObject = JsonValue.as_object();

            for (auto const & e : JsonObject)
            {
                const wstring propertyName = e.first;
                const json::value propertyValue = e.second;

                InterfaceDetails.SetInterfaceDetailsFields(propertyName, propertyValue);
            }
            break;
        }

        case CNIReportKey::BridgeDetails:
            break;

        default:
            LOGINFOW(L"Unknown key:{%s}", FieldName.c_str());
        }
    }
}

void 
CNIReport::SetMembersFromJson(
    json::object & JsonObject)
/*++

Routine Description:

    Set members of the CNI report with the report received from VM.

Arguments:

    JsonObject - Json CNI report object.
    
Return Value:

    None.

--*/
{
    for (auto const & e : JsonObject)
    {
        const wstring propertyName = e.first;
        const json::value propertyValue = e.second;

        SetCNIReportField(propertyName, propertyValue);
    }
}


void
CNIReport::PrintCNIReport()
/*++

Routine Description:

    Print each member of CNI Report

Arguments:
    
    None.

Return Value:

    None.

--*/
{
    LOGINFOW(L"StartFlag %d", StartFlag);
    LOGINFOW(L"CniSucceeded %d", CniSucceeded);

    PluginReport::PrintPluginReport();
    
    LOGINFOW(L"InterfaceDetails{Type %s PrimaryCA %s Subnet %s MAC %s Name %s CATotalCount %d CAUsedCount %d ErrMsg %s}",
        InterfaceDetails.InterfaceType.c_str(), InterfaceDetails.PrimaryCA.c_str(), InterfaceDetails.Subnet.c_str(), InterfaceDetails.MAC.c_str(), 
        InterfaceDetails.Name.c_str(),InterfaceDetails.SecondaryCATotalCount, InterfaceDetails.SecondaryCAUsedCount, InterfaceDetails.ErrorMessage.c_str());
}