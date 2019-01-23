/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

    GetInterfaceForContainerResponse.cpp

Abstract:

    Implementations of GetInterfaceForContainerResponse functions.

Environment:

    User Mode - Win32

--*/

#include "GetInterfaceForContainerResponse.h"
#include "Logging.h"

using namespace std;
using namespace web;
using namespace http;
using namespace json;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

GetInterfaceForContainerResponse::GetInterfaceForContainerResponse(json::object& JsonObject)
/*++

Routine Description:

    Initialize the GetInterfaceForContainerResponse with the Json object received from request.

Arguments:

    JsonObject = Json object received as part of the request from the VM.

Return Value:

    None.

--*/
{
    InitializeFieldIdMap();
    SetMembersFromJson(JsonObject);
}

void GetInterfaceForContainerResponse::InitializeFieldIdMap()
/*++

Routine Description:

    Initialize the GetInterfaceForContainerResponse field FieldName to key map.

Arguments:

    None.

Return Value:

    None.

--*/
{
    GetInterfaceForContainerResponseFieldIdMap[RESPONSE] = GetInterfaceForContainerResponseEnum::ResponseKey::Response;
    GetInterfaceForContainerResponseFieldIdMap[NETWORK_INTERFACE] = GetInterfaceForContainerResponseEnum::ResponseKey::NetworkInterface;
    GetInterfaceForContainerResponseFieldIdMap[CNET_ADDRESS_SPACE] = GetInterfaceForContainerResponseEnum::ResponseKey::CnetAddressSpace;
    GetInterfaceForContainerResponseFieldIdMap[DNS_SERVERS] = GetInterfaceForContainerResponseEnum::ResponseKey::DNSServers;
    GetInterfaceForContainerResponseFieldIdMap[NETWORK_CONTAINER_VERSION] = GetInterfaceForContainerResponseEnum::ResponseKey::NetworkContainerVersion;
}

void GetInterfaceForContainerResponse::SetFields(wstring Name, json::value Value)
/*++

Routine Description:

    Sets GetInterfaceForContainerResponse fields.

Arguments:

    Name - Name of the field.

    Value - Json value to be associated with the field.

Return Value:

    None.

--*/
{
    // Prevent memory exhaustion attack, since we have a [] operator
	// in the switch statement below.
	if (GetInterfaceForContainerResponseFieldIdMap.find(Name) != GetInterfaceForContainerResponseFieldIdMap.end())
	{
        switch (GetInterfaceForContainerResponseFieldIdMap[Name])
        {
        case GetInterfaceForContainerResponseEnum::ResponseKey::Response:
            Response = Response::Response(Value.as_object());

            break;
        case GetInterfaceForContainerResponseEnum::ResponseKey::NetworkInterface:
            NetworkInterface = NetworkInterface::NetworkInterface(Value.as_object());

            break;
        case GetInterfaceForContainerResponseEnum::ResponseKey::CnetAddressSpace:
            {
                auto tmpArr = Value.as_array();
                for ( auto iter = tmpArr.begin(); iter != tmpArr.end(); ++iter ) {
                    CnetAddressSpace.push_back(IPSubnet(iter->as_object()));
                }
            }

            break;
        case GetInterfaceForContainerResponseEnum::ResponseKey::DNSServers:
            {
                auto tmpArr = Value.as_array();
                for ( auto iter = tmpArr.begin(); iter != tmpArr.end(); ++iter ) {
                    DNSServers.push_back(iter->as_string());
                }
            }

            break;
        case GetInterfaceForContainerResponseEnum::ResponseKey::NetworkContainerVersion:
            NetworkContainerVersion = Value.as_string();

            break;
        }
	}
}

Response::Response()
/*++

Routine Description:

    Default Response.

Return Value:

    None.

--*/
{
    InitializeFieldIdMap();
}

Response::Response(json::object& JsonObject)
/*++

Routine Description:

    Initialize the Response with the Json object received from GetInterfaceForContainerResponse.

Arguments:

    JsonObject = Json object received as part of the request from the VM.

Return Value:

    None.

--*/
{
    InitializeFieldIdMap();
    SetMembersFromJson(JsonObject);
}

void Response::InitializeFieldIdMap()
/*++

Routine Description:

    Initialize the Response field FieldName to key map.

Arguments:

    None.

Return Value:

    None.

--*/
{
    GetInterfaceForContainerResponseFieldIdMap[RETURN_CODE] = GetInterfaceForContainerResponseEnum::ResponseKey::ReturnCode;
    GetInterfaceForContainerResponseFieldIdMap[MESSAGE] = GetInterfaceForContainerResponseEnum::ResponseKey::Message;
}

void Response::SetFields(wstring Name, json::value Value)
/*++

Routine Description:

    Sets Response fields.

Arguments:

    Name - Name of the field.

    Value - Json value to be associated with the field.

Return Value:

    None.

--*/
{
    // Prevent memory exhaustion attack, since we have a [] operator
	// in the switch statement below.
	if (GetInterfaceForContainerResponseFieldIdMap.find(Name) != GetInterfaceForContainerResponseFieldIdMap.end())
	{
        switch (GetInterfaceForContainerResponseFieldIdMap[Name])
        {
        case GetInterfaceForContainerResponseEnum::ResponseKey::ReturnCode:
            ReturnCode = Value.as_integer();

            break;
        case GetInterfaceForContainerResponseEnum::ResponseKey::Message:
            Message = Value.as_string();

            break;
        }
    }
}

NetworkInterface::NetworkInterface()
/*++

Routine Description:

    Default NetworkInterface.

Return Value:

    None.

--*/
{
    InitializeFieldIdMap();
}

NetworkInterface::NetworkInterface(json::object& JsonObject)
/*++

Routine Description:

    Initialize the NetworkInterface with the Json object received from GetInterfaceForContainerResponse.

Arguments:

    JsonObject = Json object received as part of the request from the VM.

Return Value:

    None.

--*/
{
    InitializeFieldIdMap();
    SetMembersFromJson(JsonObject);
}

void NetworkInterface::InitializeFieldIdMap()
/*++

Routine Description:

    Initialize the NetworkInterface field FieldName to key map.

Arguments:

    None.

Return Value:

    None.

--*/
{
    GetInterfaceForContainerResponseFieldIdMap[NAME] = GetInterfaceForContainerResponseEnum::ResponseKey::Name;
    GetInterfaceForContainerResponseFieldIdMap[IP_ADDRESS] = GetInterfaceForContainerResponseEnum::ResponseKey::IPAddressKey;
}

void NetworkInterface::SetFields(wstring Name, json::value Value)
/*++

Routine Description:

    Sets NetworkInterface fields.

Arguments:

    Name - Name of the field.

    Value - Json value to be associated with the field.

Return Value:

    None.

--*/
{
    // Prevent memory exhaustion attack, since we have a [] operator
	// in the switch statement below.
	if (GetInterfaceForContainerResponseFieldIdMap.find(Name) != GetInterfaceForContainerResponseFieldIdMap.end())
	{
        switch (GetInterfaceForContainerResponseFieldIdMap[Name])
        {
        case GetInterfaceForContainerResponseEnum::ResponseKey::Name:
            Name = Value.as_string();

            break;
        case GetInterfaceForContainerResponseEnum::ResponseKey::IPAddressKey:
            Address = Value.as_string();

            break;
        }
    }
}

IPSubnet::IPSubnet(json::object& JsonObject)
/*++

Routine Description:

    Initialize the IPSubnet with the Json object received from GetInterfaceForContainerResponse.

Arguments:

    JsonObject = Json object received as part of the request from the VM.

Return Value:

    None.

--*/
{
    InitializeFieldIdMap();
    SetMembersFromJson(JsonObject);
}

void IPSubnet::InitializeFieldIdMap()
/*++

Routine Description:

    Initialize the IPSubnet field FieldName to key map.

Arguments:

    None.

Return Value:

    None.

--*/
{
    GetInterfaceForContainerResponseFieldIdMap[IP_ADDRESS] = GetInterfaceForContainerResponseEnum::ResponseKey::IPAddressKey;
    GetInterfaceForContainerResponseFieldIdMap[PREFIX_LENGTH] = GetInterfaceForContainerResponseEnum::ResponseKey::PrefixLength;
}

void IPSubnet::SetFields(wstring Name, json::value Value)
/*++

Routine Description:

    Sets IPSubnet fields.

Arguments:

    Name - Name of the field.

    Value - Json value to be associated with the field.

Return Value:

    None.

--*/
{
    // Prevent memory exhaustion attack, since we have a [] operator
	// in the switch statement below.
	if (GetInterfaceForContainerResponseFieldIdMap.find(Name) != GetInterfaceForContainerResponseFieldIdMap.end())
	{
        switch (GetInterfaceForContainerResponseFieldIdMap[Name])
        {
        case GetInterfaceForContainerResponseEnum::ResponseKey::IPAddressKey:
            Address = Value.as_string();

            break;
        case GetInterfaceForContainerResponseEnum::ResponseKey::PrefixLength:
            Cidr = Value.as_integer() & 0xff;

            break;
        }
    }
}
