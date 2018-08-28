/*++

Copyright (c) 2011  Microsoft Corporation

Module Name:

    CNIReport.cpp

Abstract:

    Implementation of CNI report functions.

Environment:

    User Mode - Win32

--*/

#include "CNIReport.h"
#include "Logging.h"

using namespace std;
using namespace web;
using namespace http;
using namespace json;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;


CNIReport::CNIReport(
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
CNIReport::InitializeFieldIdMap()
/*++

Routine Description:

    Initialize the report field FieldName to key map.

Arguments:

    None.
    
Return Value:

    None.

--*/
{
    PluginReport::InitializeFieldIdMap();

	// TODO: Remove STARTFLAG_STR in future versions. 
	// This is here for backward compatibility for now.
	CNIReportFieldIdMap[STARTFLAG_STR] = CNIEnum::CNIReportKey::IsNewInstance;
    CNIReportFieldIdMap[IS_NEW_INSTANCE_STR] = CNIEnum::CNIReportKey::IsNewInstance;
    CNIReportFieldIdMap[CNI_SUCCEEDED_STR] = CNIEnum::CNIReportKey::CniSucceeded;
    CNIReportFieldIdMap[VNET_ADDRESS_SPACE_STR] = CNIEnum::CNIReportKey::VnetAddressSpace;
    CNIReportFieldIdMap[INTERFACE_DETAILS_STR] = CNIEnum::CNIReportKey::InterfaceDetails;
    CNIReportFieldIdMap[INTERFACE_TYPE_STR] = CNIEnum::CNIReportKey::InterfaceType;
    CNIReportFieldIdMap[SUBNET_STR] = CNIEnum::CNIReportKey::Subnet;
    CNIReportFieldIdMap[PRIMARY_CA_STR] = CNIEnum::CNIReportKey::PrimaryCA;
    CNIReportFieldIdMap[MAC_STR] = CNIEnum::CNIReportKey::MAC;
    CNIReportFieldIdMap[CA_TOTAL_COUNT_STR] = CNIEnum::CNIReportKey::SecondaryCATotalCount;
    CNIReportFieldIdMap[CA_USED_COUNT_STR] = CNIEnum::CNIReportKey::SecondaryCAUsedCount;
    CNIReportFieldIdMap[NAME_STR] = CNIEnum::CNIReportKey::IfName;
    CNIReportFieldIdMap[ERROR_MESSAGE_STR] = CNIEnum::CNIReportKey::IfErrorMessage;
    CNIReportFieldIdMap[BRIDGE_DETAILS_STR] = CNIEnum::CNIReportKey::BridgeDetails;
	CNIReportFieldIdMap[METADATA_STR] = CNIEnum::CNIReportKey::Metadata;
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
    case CNIEnum::CNIReportKey::InterfaceType:
        InterfaceType.assign(JsonValue.as_string());
        break;

    case CNIEnum::CNIReportKey::Subnet:
        Subnet.assign(JsonValue.as_string());
        break;

    case CNIEnum::CNIReportKey::PrimaryCA:
        PrimaryCA.assign(JsonValue.as_string());
        break;

    case CNIEnum::CNIReportKey::MAC:
        MAC.assign(JsonValue.as_string());
        break;

    case CNIEnum::CNIReportKey::IfName:
        Name.assign(JsonValue.as_string());
        break;

    case CNIEnum::CNIReportKey::SecondaryCATotalCount:
        SecondaryCATotalCount = JsonValue.as_integer();
        break;

    case CNIEnum::CNIReportKey::SecondaryCAUsedCount:
        SecondaryCAUsedCount = JsonValue.as_integer();
        break;

    case CNIEnum::CNIReportKey::IfErrorMessage:
        ErrorMessage.assign(JsonValue.as_string());
        break;
    default:
        LOGINFOW(L"InterfaceDetails Unknown key:{%s}\n", FieldName.c_str());
    }
}

void 
CNIReport::SetFields(
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
        case CNIEnum::CNIReportKey::IsNewInstance:
            IsNewInstance = JsonValue.as_bool();
            break;
        case CNIEnum::CNIReportKey::CniSucceeded:
            CniSucceeded = JsonValue.as_bool();
            break;
        
        case CNIEnum::CNIReportKey::VnetAddressSpace:
        {
            break;
        }

        case CNIEnum::CNIReportKey::InterfaceDetails:
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

        case CNIEnum::CNIReportKey::BridgeDetails:
            break;
		case CNIEnum::CNIReportKey::Metadata:
			Metadata = Metadata::Metadata(JsonValue.as_object());
			break;

        default:
            LOGINFOW(L"Unknown key:{%s}", FieldName.c_str());
        }
    }
}

void 
CNIReport::SetMembersFromJson(json::object & JsonObject)
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

		SetFields(propertyName, propertyValue);
    }
}


void
CNIReport::Print()
/*++

Routine Description:

    Print each member of CNI Report

Arguments:
    
    None.

Return Value:

    None.

--*/
{
    LOGINFOW(L"IsNewInstance %d", IsNewInstance);
    LOGINFOW(L"CniSucceeded %d", CniSucceeded);

    PluginReport::Print();
    LOGINFOW(L"InterfaceDetails{Type %s PrimaryCA %s Subnet %s MAC %s Name %s CATotalCount %d CAUsedCount %d ErrMsg %s}",
        InterfaceDetails.InterfaceType.c_str(), InterfaceDetails.PrimaryCA.c_str(), InterfaceDetails.Subnet.c_str(), InterfaceDetails.MAC.c_str(), 
        InterfaceDetails.Name.c_str(),InterfaceDetails.SecondaryCATotalCount, InterfaceDetails.SecondaryCAUsedCount, InterfaceDetails.ErrorMessage.c_str());
	Metadata.Print();
}