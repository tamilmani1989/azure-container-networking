/**
\copyright Copyright (c) Microsoft Corporation

\file MarshalJson.cpp

\brief Contains the infrastructure for marshaling and unmarshaling
static-typed values to and from JSON.

\author John Starks (jostarks)
\date Apr 23, 2015

Last Updated: 
Sushant Sharma (susharm) 06-April-2017
*/

#include "inc\MarshalJson.h"
#include <wincrypt.h>
#include <wil/result.h>

namespace Marshal
{

	namespace Details
	{
		/// Find a field within a JSON object.
		///
		/// \param Object The object to lookup the key in.
		/// \param Key The key we are searching for.
		/// \param CaseInsensitiveComparison Perform case insensitive key comparison.
		///
		/// \return An iterator into the object, or Object.end() if no such object was found.
		json::object::const_iterator
			JsonFindObject(
				_In_ const json::object& Object,
				_In_ PCWSTR Key,
				_In_ const bool CaseInsensitiveComparison
			)
		{

			auto iter = std::lower_bound(Object.begin(),
				Object.end(),
				Key,
				[](const std::pair<std::wstring, json::value>& Value, const wchar_t* Key)
			{
				return Value.first < Key;
			});

			if (iter != Object.end() && iter->first == Key)
			{
				return iter;
			}
			else if (!CaseInsensitiveComparison)
			{
				return Object.end();
			}

			// Do a slow-path case insensitive comparison if the flag is specified
			return std::find_if(Object.begin(),
				Object.end(),
				[Key](const std::pair<std::wstring, json::value>& Value)
			{
				return _wcsicmp(Value.first.c_str(), Key) == 0;
			});
		}

		/// Convert a JSON value to a string, throwing on type mismatch.
		JsonToSimpleType::operator std::wstring()
		{
			if (!Value.is_string())
			{
				throw UnmarshalError::ExpectedString;
			}

			return Value.as_string();
		}

		/// Convert a JSON value to an unsigned integer, throwing on type mismatch.
		JsonToSimpleType::operator UINT64()
		{
			if (!Value.is_number())
			{
				throw UnmarshalError::ExpectedUnsignedInteger;
			}

			auto number = Value.as_number();
			if (!number.is_uint64())
			{
				throw UnmarshalError::ExpectedUnsignedInteger;
			}

			return number.to_uint64();
		}

		/// Convert a JSON value to a signed integer, throwing on type mismatch.
		JsonToSimpleType::operator INT64()
		{
			if (!Value.is_number())
			{
				throw UnmarshalError::ExpectedSignedInteger;
			}

			auto number = Value.as_number();
			if (!number.is_int64())
			{
				throw UnmarshalError::ExpectedSignedInteger;
			}

			return number.to_int64();
		}

		/// Convert a JSON value to a Boolean, throwing on type mismatch.
		JsonToSimpleType::operator bool()
		{
			if (!Value.is_boolean())
			{
				throw UnmarshalError::ExpectedBoolean;
			}

			return Value.as_bool();
		}

		/// Convert a JSON value to a floating point value, throwing on type mismatch.
		JsonToSimpleType::operator double()
		{
			if (!Value.is_number())
			{
				throw UnmarshalError::ExpectedFloat;
			}

			return Value.as_number().to_double();
		}

		/// Convert a marshalable object to a JSON value.
		///
		/// \param Value The object to convert.
		/// \param Context The marshal context.
		/// \param Fields An array of field information (name, offset).
		/// \param Types An array of basic type functions (IsDefault, SetDefault).
		/// \param JsonTypes An array of JSON type functions (ToJson, FromJson).
		///
		/// \return Returns a JSON value.
		json::value
			ObjectToJson(
				_In_ const void* Value,
				_In_ const MarshalContext& Context,
				_In_ ArrayView<FieldData> Fields,
				_In_ ArrayView<const BaseTypeData*> Types,
				_In_ ArrayView<const JsonTypeData*> JsonTypes
			)
		{
			std::vector<std::pair<std::wstring, json::value>> object;
			size_t index = 0;
			for (const auto& field : Fields)
			{
				const void* fieldValue = static_cast<const char*>(Value) + field.Offset;
				if ((field.Options & FieldOption::OmitEmpty) != FieldOption::OmitEmpty ||
					!Types[index]->IsDefault(fieldValue, field.DefaultValue))
				{
					object.emplace_back(field.Name, JsonTypes[index]->ToJson(fieldValue, Context));

					// If OmitEmpty was specified on an object type that ended up with no
					// fields, then remove the object. This is easier to detect and correct after the
					// fact than examine the object (recursively) beforehand.
					if ((field.Options & FieldOption::OmitEmpty) == FieldOption::OmitEmpty &&
						object.back().second.is_object() && object.back().second.as_object().empty())
					{
						object.pop_back();
					}
				}

				index++;
			}

			return json::value::object(std::move(object));
		}

		/// Convert a JSON value to a marshalable object.
		///
		/// \param Value The JSON value to convert.
		/// \param Result An initialized marshalable object to update.
		/// \param Context The unmarshal context.
		/// \param Fields An array of field information (name, offset).
		/// \param Types An array of basic type functions (IsDefault, SetDefault).
		/// \param JsonTypes An array of JSON type functions (ToJson, FromJson).
		///
		/// \return false if any validation failed.
		bool
			ObjectFromJson(
				_In_ const json::value& Value,
				_Inout_ void* Result,
				_In_ const UnmarshalContext& Context,
				_In_ ArrayView<FieldData> Fields,
				_In_ ArrayView<const BaseTypeData*> Types,
				_In_ ArrayView<const JsonTypeData*> JsonTypes
			)
		{
			if (!Value.is_object())
			{
				Context.ReportError(UnmarshalError::ExpectedObject);
				return false;
			}

			const json::object& object = Value.as_object();
			bool allValid = true;
			size_t index = 0;
			for (const auto& field : Fields)
			{
				void* fieldValue = static_cast<char*>(Result) + field.Offset;
				auto entry = JsonFindObject(object, field.Name, (Context.GetFlags() & UnmarshalFlags::CaseInsensitive) == UnmarshalFlags::CaseInsensitive);
				if (entry != object.end() && !entry->second.is_null())
				{
					if (!JsonTypes[index]->FromJson(entry->second,
						fieldValue,
						Details::UnmarshalFieldContext(Context, entry->first.c_str())))
					{
						allValid = false;
					}
				}
				else if ((field.Options & FieldOption::Required) == FieldOption::Required)
				{
					Details::UnmarshalFieldContext(Context, field.Name).ReportError(UnmarshalError::MissingRequiredField);
					allValid = false;
				}
				else
				{
					Types[index]->SetDefault(fieldValue, field.DefaultValue);
				}

				index++;
			}

			return allValid;
		}

		/// Convert an array to a JSON value.
		///
		/// \param Value The array to convert.
		/// \param Count The number of elements in the array.
		/// \param Context The marshal context.
		/// \param Type Basic type functions for the element type (IsDefault, SetDefault).
		/// \param JsonType JSON type functions for the element type (ToJson, FromJson).
		/// \param Size The size of each array element, in bytes.
		///
		/// \return A JSON value.
		json::value
			ArrayToJson(
				_In_reads_bytes_(Count * Size) const void* Value,
				_In_ size_t Count,
				_In_ const MarshalContext& Context,
				_In_ const BaseTypeData* /* Type */,
				_In_ const JsonTypeData* JsonType,
				_In_ size_t Size
			)
		{
			std::vector<json::value> array;
			array.reserve(Count);
			for (size_t offset = 0; offset < Count * Size; offset += Size)
			{
				array.push_back(JsonType->ToJson(static_cast<const char*>(Value) + offset, Context));
			}

			return json::value::array(std::move(array));
		}

		/// Convert an array of JSON value to an array.
		///
		/// \param Values The array of JSON values to convert.
		/// \param Result An array of initialized C++ values to update with the unmarshaled result.
		/// \param Count The number of elements in the arrays.
		/// \param Context The unmarshal context.
		/// \param Type Basic type functions for the element type (IsDefault, SetDefault).
		/// \param JsonType JSON type functions for the element type (ToJson, FromJson).
		/// \param Size The size of each array element, in bytes.
		///
		/// \return false if any validation failed.
		bool
			ArrayFromJson(
				_In_reads_(Count) const json::value* Values,
				_Inout_updates_bytes_(Count * Size) void* Result,
				_In_ size_t Count,
				_In_ const UnmarshalContext& Context,
				_In_ const BaseTypeData* Type,
				_In_ const JsonTypeData* JsonType,
				_In_ size_t Size
			)
		{
			bool allValid = true;
			for (size_t i = 0; i < Count; ++i)
			{
				void* fieldValue = static_cast<char*>(Result) + i * Size;
				if (!Values[i].is_null())
				{
					if (!JsonType->FromJson(Values[i], fieldValue, Details::UnmarshalIndexContext(Context, i)))
					{
						allValid = false;
					}
				}
				else
				{
					// A null array entry gets treated as the empty/zero default value for the type.
					Type->SetDefault(fieldValue, 0);
				}
			}

			return allValid;
		}

		/// Parses a JSON string into a Casablanca json value.
		std::pair<bool, std::vector<std::pair<std::wstring, UnmarshalError>>>
			ParseJson(
				_In_ const std::wstring& String,
				_Inout_ json::value* Value
			)
		{
			try
			{
				*Value = json::value::parse(String);
				return{ true,{} };
			}
			catch (json::json_exception&)
			{
				return{ false,{ { L"$", UnmarshalError::InvalidJson } } };
			}
		}
	}

	/// "Converts" a JSON value to a JSON value.
	json::value
		ToJsonValue(
			_In_ const json::value& Value,
			_In_ const MarshalContext&
		)
	{
		return Value;
	}

	/// "Converts" a JSON value from a JSON value.
	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ json::value* Result,
			_In_ const UnmarshalContext&
		)
	{
		*Result = Value;
		return true;
	}

	/// Determines if a JSON value has its default value (null).
	bool
		IsDefault(
			_In_ const json::value& Value
		)
	{
		return Value.is_null();
	}

	/// Converts a byte vector to a Base64-encoded JSON string.
	json::value
		ToJsonValue(
			_In_ const std::vector<UINT8>& Value,
			_In_ const MarshalContext&
		)
	{
		DWORD binarySize = static_cast<DWORD>(Value.size());
		std::wstring result;

		if (binarySize > 0)
		{
			DWORD stringSize = 0;
			THROW_IF_WIN32_BOOL_FALSE(CryptBinaryToStringW(Value.data(),
				binarySize,
				CRYPT_STRING_BASE64 | CRYPT_STRING_NOCRLF,
				nullptr,
				&stringSize));

			result.resize(stringSize, 0);
			THROW_IF_WIN32_BOOL_FALSE(CryptBinaryToStringW(Value.data(),
				binarySize,
				CRYPT_STRING_BASE64 | CRYPT_STRING_NOCRLF,
				&result[0],
				&stringSize));

			// Remove the extra \0 terminator.
			result.resize(stringSize);
		}

		return json::value(std::move(result));
	}

	/// Converts a Base64-encoded JSON string to a byte vector.
	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ std::vector<UINT8>* Result,
			_In_ const UnmarshalContext& Context
		)
	{
		std::wstring str = Value.as_string();
		DWORD stringSize = static_cast<DWORD>(str.size());

		if (stringSize > 0)
		{
			DWORD binarySize = 0;
			if (!CryptStringToBinaryW(str.data(),
				stringSize,
				CRYPT_STRING_BASE64 | CRYPT_STRING_STRICT,
				nullptr,
				&binarySize,
				nullptr,
				nullptr))
			{
				DWORD error = GetLastError();
				if (error == ERROR_INVALID_DATA)
				{
					Context.ReportError(UnmarshalError::InvalidBase64);
					return false;
				}
				else
				{
					THROW_WIN32(error);
				}
			}

			Result->resize(binarySize);
			THROW_IF_WIN32_BOOL_FALSE(CryptStringToBinaryW(str.data(),
				stringSize,
				CRYPT_STRING_BASE64 | CRYPT_STRING_STRICT,
				Result->data(),
				&binarySize,
				nullptr,
				nullptr));
		}

		return true;
	}

}

