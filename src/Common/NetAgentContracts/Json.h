/*++

Copyright (c) 2018 Microsoft Corporation

Module Name:

	Json.h

Abstract:

	Header to define the abstract Json class.

Environment:

	User Mode - Win32

--*/

#pragma once

#include <string>
#include <cpprest/json.h>

using namespace std;
using namespace web;

// Generic abstract json class.
// All derived classes need to implement declared method in this class.
namespace Microsoft {namespace Azure {namespace Networking {namespace Aqua {namespace ClientLibrary {

template<class T>
class Json {
public:

	static void InitializeFields() { T::InitializeFieldIdMap(); };

	void SetMembersFromJson(json::object &JsonObject);

	virtual void SetFields(
		wstring Name,
		json::value Value
	) = 0;

	virtual void Print() = 0;

	inline static const wstring ConvertBoolToWstring(const bool b) { return b ? L"true" : L"false"; };
};

}
template<class T>
void ClientLibrary::Json<T>::SetMembersFromJson(json::object & JsonObject)
/*++

Routine Description:

	Set members of the Metadata object with the report received from VM.

Arguments:

	JsonObject - Json Metadata object.

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
}}}}