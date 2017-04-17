/**
\copyright Copyright (c) Microsoft Corporation

\file MarshalBase.cpp

\brief Contains the base infrastructure for marshaling and unmarshaling
static-typed values to and from JSON or VM repository objects.

\author John Starks (jostarks)
\date Apr 23, 2015
*/

#include "inc\MarshalBase.h"
#include <string>
#include <algorithm>
#include <rpc.h>
#include <objbase.h>

namespace Marshal
{
	namespace Details
	{
		UnmarshalRootContext::UnmarshalRootContext(UnmarshalFlags Flags)
			: m_Flags(Flags)
		{
		}

		/// Records an error during unmarshaling.
		/// \param Key The position in the object being unmarshaled that had the error
		/// \param Error The reason for the error
		void
			UnmarshalRootContext::ReportError(
				_In_ PCWSTR Key,
				_In_ UnmarshalError Error
			)
		{
			m_Errors.emplace_back(Key, Error);
		}

		/// Gets the errors that occurred during unmarshaling.
		///
		/// This is destructive because it transfers ownership of the errors to the caller.
		std::vector<std::pair<std::wstring, UnmarshalError>>
			UnmarshalRootContext::TakeErrors()
		{
			return std::move(m_Errors);
		}

		/// Gets the flags that were specified when initiating the unmarshal request.
		UnmarshalFlags
			UnmarshalRootContext::GetFlags() const
		{
			return m_Flags;
		}

		UnmarshalIndexContext::UnmarshalIndexContext(
			_In_ const UnmarshalContext& Parent,
			_In_ size_t Index
		)
			: UnmarshalContext(Parent),
			m_Index(Index)
		{
		}

		/// Appends a string representation of the current array index to the key that's used
		/// when reporting errors.
		void
			UnmarshalIndexContext::GetKeySuffix(
				_Inout_ std::wstring& Key
			) const
		{
			Key += L'[';
			Key += std::to_wstring(m_Index);
			Key += L']';
		}

		UnmarshalFieldContext::UnmarshalFieldContext(
			_In_ const UnmarshalContext& Parent,
			_In_ PCWSTR Name
		)
			: UnmarshalContext(Parent),
			m_Name(Name)
		{
		}

		/// Appends a string representation of the current field name to the key that's used
		/// when reporting errors.
		void
			UnmarshalFieldContext::GetKeySuffix(
				_Inout_ std::wstring& Key
			) const
		{
			Key += L'.';
			Key += m_Name;
		}

		UnmarshalSuffixContext::UnmarshalSuffixContext(
			_In_ const UnmarshalContext& Parent,
			_In_ PCWSTR Suffix
		)
			: UnmarshalContext(Parent),
			m_Suffix(Suffix)
		{
		}

		/// Appends a string representation of the current field name to the key that's used
		/// when reporting errors.
		void
			UnmarshalSuffixContext::GetKeySuffix(
				_Inout_ std::wstring& Key
			) const
		{
			Key += m_Suffix;
		}

		/// Converts an enumeration value to a string.
		/// \param Value The enumeration value (already casted to an integer).
		/// \param Data Metadata about the enumeration.
		std::wstring
			EnumToSimpleValue(
				_In_ UINT32 Value,
				_In_ const EnumData& Data
			)
		{
			if (Value > Data.Count)
			{
				__fastfail(FAST_FAIL_INVALID_ARG);
			}

			return Data.Names[Value];
		}

		/// Converts a string to an enumeration value.
		/// \param Value The string representation of the enum value.
		/// \param Result Returns the resulting enumeration value.
		/// \param Data Metadata about the enumeration.
		///
		/// \return false if there was no matching value.
		bool
			EnumFromSimpleValue(
				_In_ const std::wstring& Value,
				_Inout_ UINT32* Result,
				_In_ const UnmarshalContext& Context,
				_In_ const EnumData& Data
			)
		{
			auto begin = Data.SortedNames;
			auto end = Data.SortedNames + Data.Count;
			auto iter = std::lower_bound(begin,
				end,
				Value.c_str(),
				[](PCWSTR Name, PCWSTR Value)
			{

				return _wcsicmp(Name, Value) < 0;
			});

			if (iter == end || _wcsicmp(*iter, Value.c_str()) != 0)
			{
				Context.ReportError(UnmarshalError::UnknownEnumValue);
				return false;
			}

			*Result = Data.SortedNamesToValues[iter - begin];
			return true;
		}

		/// Sets each field of a marshalable object to its default values.
		void
			SetObjectFieldDefaults(
				_In_ void* Value,
				_In_ ArrayView<FieldData> Fields,
				_In_ ArrayView<const BaseTypeData*> Types
			)
		{
			size_t index = 0;
			for (const auto& field : Fields)
			{
				Types[index]->SetDefault(static_cast<char*>(Value) + field.Offset, field.DefaultValue);
				index++;
			}
		}
	}

	UnmarshalContext::UnmarshalContext(
		_In_ Details::UnmarshalRootContext& RootContext
	)
		: m_Root(&RootContext)
	{
	}

	/// Reports an error at the current unmarshaling position.
	void
		UnmarshalContext::ReportError(
			_In_ UnmarshalError Error
		) const
	{
		std::wstring key;
		GetKey(key);
		m_Root->ReportError(key.c_str(), Error);
	}

	/// Gets the flags that were specified when initiating the unmarshal request.
	UnmarshalFlags
		UnmarshalContext::GetFlags() const
	{
		return m_Root->GetFlags();
	}

	UnmarshalContext::UnmarshalContext(

		_In_ const UnmarshalContext& Parent
	)
		: m_Root(Parent.m_Root),
		m_Parent(&Parent)
	{
	}

	void
		UnmarshalContext::GetKeySuffix(
			_Inout_ std::wstring&
		) const
	{
	}

	/// Gets a string that identifies the current position in the object being unmarshaled.
	void
		UnmarshalContext::GetKey(
			_Inout_ std::wstring& Key
		) const
	{
		if (m_Parent != nullptr)
		{
			m_Parent->GetKey(Key);
			GetKeySuffix(Key);
		}
		else
		{
			Key = L"$";
		}
	}

	bool
		ToSimpleValue(
			_In_ bool Value,
			_In_ const MarshalContext&
		)
	{
		return Value;
	}

	bool
		FromSimpleValue(
			_In_ bool Value,
			_Inout_ bool* Result,
			_In_ const UnmarshalContext&
		)
	{
		*Result = Value;
		return true;
	}

	std::wstring
		ToSimpleValue(
			_In_ const std::wstring& Value,
			_In_ const MarshalContext&
		)
	{
		return Value;
	}

	bool
		FromSimpleValue(
			_In_ const std::wstring& Value,
			_Inout_ std::wstring* Result,
			_In_ const UnmarshalContext&
		)
	{
		*Result = Value;
		return true;
	}

	bool
		IsDefault(
			_In_ const std::wstring& Value,
			_In_opt_ PCWSTR DefaultValue
		)
	{
		return DefaultValue != nullptr ? (Value == DefaultValue) : Value.empty();
	}

	void
		SetDefault(
			_Inout_ std::wstring* Result,
			_In_opt_ PCWSTR DefaultValue
		)
	{
		if (DefaultValue != nullptr)
		{
			*Result = DefaultValue;
		}
		else
		{
			Result->clear();
		}
	}

	double
		ToSimpleValue(
			_In_ double Value,
			_In_ const MarshalContext&
		)
	{
		return Value;
	}

	bool
		FromSimpleValue(
			_In_ double Value,
			_Inout_ double* Result,
			_In_ const UnmarshalContext&
		)
	{
		*Result = Value;
		return true;
	}

	// Converts a GUID to a string, optionally surrounding the result with curly braces (COM format).
	std::wstring
		ToSimpleValueGuid(
			_In_ REFGUID Value,
			_In_ bool AddBraces
		)
	{
		WCHAR buffer[_countof(L"{xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}")];
		swprintf_s(buffer,
			AddBraces ? L"{%08lx-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x}" :
			L"%08lx-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x",
			Value.Data1, Value.Data2, Value.Data3, Value.Data4[0],
			Value.Data4[1], Value.Data4[2], Value.Data4[3], Value.Data4[4],
			Value.Data4[5], Value.Data4[6], Value.Data4[7]);

		return buffer;
	}

	/// Converts a GUID to a string in xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx format.
	std::wstring
		ToSimpleValue(
			_In_ REFGUID Value,
			_In_ const MarshalContext&
		)
	{
		return ToSimpleValueGuid(Value, false);
	}
	
	/// Converts a GUID from a string in xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx format.
	bool 
		FromSimpleValue(
			_In_ const std::wstring& Value,
			_Inout_ GUID* Result,
			_In_ const UnmarshalContext& Context)
	{
		if (UuidFromStringW((RPC_WSTR)Value.c_str(), Result) != RPC_S_OK)
		{
			Context.ReportError(UnmarshalError::InvalidGuidValue);
			return false;
		}

		return true;
	}


	/// Converts a GUID to a string in {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx} format.
	std::wstring
		ToSimpleValue(
			_In_ REFGUID Value,
			_In_ const MarshalContext&,
			IncludeBraces
		)
	{
		return ToSimpleValueGuid(Value, true);
	}

	/// Converts a GUID from a string in {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx} format.
	bool
		FromSimpleValue(
			_In_ const std::wstring& Value,
			_Inout_ GUID* Result,
			_In_ const UnmarshalContext& Context,
			IncludeBraces
		)
	{
		if (IIDFromString(Value.c_str(), Result) != S_OK)
		{
			Context.ReportError(UnmarshalError::InvalidGuidValue);
			return false;
		}

		return true;
	}

}

