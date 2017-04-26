/**
\copyright Copyright (c) Microsoft Corporation

\file MarshalJson.h

\brief Contains the infrastructure for marshaling and unmarshaling
static-typed values to and from JSON.

The main entry points in this file are Marshal::ToJson() and Marshal::FromJson().

\author John Starks (jostarks)
\date Apr 23, 2015
*/

#pragma once

#include "MarshalBase.h"
#include "../../cpprest/json.h"
#include <map>
#include <vector>
#include <array>

namespace Marshal
{

	namespace json = web::json;

	namespace Details
	{
		/// Function pointers for converting a C++ value to and from a JSON value.
		struct JsonTypeData
		{
			json::value(*ToJson)(const void* Value, const MarshalContext& Context);
			bool(*FromJson)(const json::value& Value, void* Result, const UnmarshalContext& Context);
		};

		/// Wrapper around ToJsonValue to use as a function pointer.
		template <class T, class... Modifiers>
		json::value
			ToJsonVoid(
				_In_ const void* Value,
				_In_ const MarshalContext& Context
			)
		{
			return ToJsonValue(*static_cast<const T*>(Value), Context, DefaultHelper<Modifiers>()...);
		}

		/// Wrapper around FromJsonValue to use as a function pointer.
		template <class T, class... Modifiers>
		bool
			FromJsonVoid(
				_In_ const json::value& Value,
				_Inout_ void* Result,
				_In_ const UnmarshalContext& Context
			)
		{
			return FromJsonValue(Value, static_cast<T*>(Result), Context, DefaultHelper<Modifiers>()...);
		}

		/// Helper class to use to generate a table of function pointers for
		/// converting to/from JSON.
		template <class T, class... Modifiers>
		struct JsonTypeInfo
		{
			static const JsonTypeData Value;
		};

		template <class T, class... Modifiers>
		const JsonTypeData JsonTypeInfo<T, Modifiers...>::Value = { ToJsonVoid<T, Modifiers...>, FromJsonVoid<T, Modifiers...> };

		json::object::const_iterator
			JsonObjectFindCaseInsensitive(
				_In_ const json::object& Object,
				_In_ PCWSTR Key
			);

		json::value
			ObjectToJson(
				_In_ const void* Value,
				_In_ const MarshalContext& Context,
				_In_ ArrayView<FieldData> Fields,
				_In_ ArrayView<const BaseTypeData*> Types,
				_In_ ArrayView<const JsonTypeData*> JsonTypes
			);

		bool
			ObjectFromJson(
				_In_ const json::value& Value,
				_Inout_ void* Result,
				_In_ const UnmarshalContext& Context,
				_In_ ArrayView<FieldData> Fields,
				_In_ ArrayView<const BaseTypeData*> Types,
				_In_ ArrayView<const JsonTypeData*> JsonTypes
			);

		json::value
			ArrayToJson(
				_In_reads_bytes_(Count * Size) const void* Value,
				_In_ size_t Count,
				_In_ const MarshalContext& Context,
				_In_ const BaseTypeData* Type,
				_In_ const JsonTypeData* JsonType,
				_In_ size_t Size
			);

		bool
			ArrayFromJson(
				_In_reads_(Count) const json::value* Values,
				_Inout_updates_bytes_(Count * Size) void* Result,
				_In_ size_t Count,
				_In_ const UnmarshalContext& Context,
				_In_ const BaseTypeData* Type,
				_In_ const JsonTypeData* JsonType,
				_In_ size_t Size
			);

		std::pair<bool, std::vector<std::pair<std::wstring, UnmarshalError>>>
			ParseJson(
				_In_ const std::wstring& String,
				_Inout_ json::value* Value
			);

		/// Helper class to convert from a JSON value to a simple value type.
		/// This is done as a series of conversion operators so that this type
		/// can be passed to FromSimpleValue without knowing which type is
		/// expected (see FromJsonGeneric()).
		struct JsonToSimpleType
		{
			const json::value& Value;

			JsonToSimpleType() = delete;
			JsonToSimpleType(const JsonToSimpleType&) = delete;
			JsonToSimpleType& operator=(const JsonToSimpleType&) = delete;

			operator std::wstring();
			operator UINT64();
			operator INT64();
			operator bool();
			operator double();
		};

		/// Converts a non-marshalable object to a JSON value by calling into
		/// one of the ToSimpleValue() overrides. If you see an error here,
		/// it's probably because there is no ToSimpleValue override for your type.
		template <class T, class... Modifiers>
		json::value
			ToJsonGeneric(
				_In_ const T& Value,
				_In_ const MarshalContext& Context,
				std::false_type, /* CanMarshalObject<T> */
				Modifiers... ModifierValues
			)
		{
			return json::value(ToSimpleValue(Value, Context, ModifierValues...));
		}

		/// Converts a non-marshalable object from a JSON value by calling into
		/// one of the FromSimpleValue() overrides. If you see an error here,
		/// it's probably because there is no FromSimpleValue override for your
		/// type.
		template <class T, class... Modifiers>
		bool
			FromJsonGeneric(
				_In_ const json::value& Value,
				_Inout_ T* Result,
				_In_ const UnmarshalContext& Context,
				std::false_type, /* CanMarshalObject<T> */
				Modifiers... ModifierValues
			)
		{
			JsonToSimpleType converter{ Value };

			// Since the converter uses implicit conversion operators, there is
			// no way to extract an error value explicitly. Therefore, any error
			// in the conversion is thrown as an UnmarshalError and must be
			// caught and reported as normal.
			try
			{
				return FromSimpleValue(converter, Result, Context, ModifierValues...);
			}
			catch (UnmarshalError Error)
			{
				Context.ReportError(Error);
				return false;
			}
		}

		/// Convert a marshalable object to JSON representation.
		template <class T>
		json::value
			ToJsonGeneric(
				_In_ const T& Value,
				_In_ const MarshalContext& Context,
				std::true_type /* CanMarshalObject<T> */
			)
		{
			return ObjectToJson(std::addressof(Value),
				Context,
				T::FieldData,
				ObjectTypeInfo<T::FieldTypes, BaseTypeData, BaseTypeInfo>::Value,
				ObjectTypeInfo<T::FieldTypes, JsonTypeData, JsonTypeInfo>::Value);
		}

		/// Convert a marshalable object from JSON representation.
		template <class T>
		bool
			FromJsonGeneric(
				_In_ const json::value& Value,
				_Inout_ T* Result,
				_In_ const UnmarshalContext& Context,
				std::true_type /* CanMarshalObject<T> */
			)
		{
			return ObjectFromJson(Value,
				Result,
				Context,
				T::FieldData,
				ObjectTypeInfo<T::FieldTypes, BaseTypeData, BaseTypeInfo>::Value,
				ObjectTypeInfo<T::FieldTypes, JsonTypeData, JsonTypeInfo>::Value);
		}
	}

	/// Convert a value to JSON. This is the "base" version of this function that
	/// handles marshalable objects and types that have a ToSimpleValue/FromSimpleValue
	/// override. For other cases, there are other overrides of this function.
	template <class T, class... Modifiers>
	json::value
		ToJsonValue(
			_In_ const T& Value,
			_In_ const MarshalContext& Context,
			Modifiers... ModifierValues
		)
	{
		return Details::ToJsonGeneric(Value, Context, decltype(CanMarshalObject(Value))(), ModifierValues...);
	}

	/// Convert a value from JSON. This is the "base" version of this function that
	/// handles marshalable objects and types that have a ToSimpleValue/FromSimpleValue
	/// override. For other cases, there are other overrides of this function.
	template <class T, class... Modifiers>
	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ T* Result,
			_In_ const UnmarshalContext& Context,
			Modifiers... ModifierValues
		)
	{
		return Details::FromJsonGeneric(Value, Result, Context, decltype(CanMarshalObject(*Result))(), ModifierValues...);
	}

	/// Convert a vector<T> to JSON as an array.
	template <class T, class... Modifiers>
	json::value
		ToJsonValue(
			_In_ const std::vector<T>& Value,
			_In_ const MarshalContext& Context,
			ElementModifiers<Modifiers...>
		)
	{
		return Details::ArrayToJson(Value.data(),
			Value.size(),
			Context,
			&Details::BaseTypeInfo<T, Modifiers...>::Value,
			&Details::JsonTypeInfo<T, Modifiers...>::Value,
			sizeof(T));
	}

	/// Convert a vector<T> from JSON as an array.
	template <class T, class... Modifiers>
	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ std::vector<T>* Result,
			_In_ const UnmarshalContext& Context,
			ElementModifiers<Modifiers...>
		)
	{
		if (!Value.is_array())
		{
			// If FuzzyArrayMatch is specified, try to parse a non-array as
			// a single-element array. This is necessary to unmarshal JSON that has
			// been produced by converting from XML or a VM repository.
			if ((Context.GetFlags() & UnmarshalFlags::FuzzyArrayMatch) == UnmarshalFlags::FuzzyArrayMatch)
			{
				Result->resize(1);
				return FromJsonValue(Value, &(*Result)[0], Context);
			}
			else
			{
				Context.ReportError(UnmarshalError::ExpectedArray);
				return false;
			}
		}

		const auto& array = Value.as_array();
		size_t count = array.size();
		Result->resize(count);
		if (count > 0)
		{
			return Details::ArrayFromJson(&*array.begin(),
				Result->data(),
				count,
				Context,
				&Details::BaseTypeInfo<T, Modifiers...>::Value,
				&Details::JsonTypeInfo<T, Modifiers...>::Value,
				sizeof(T));
		}
		else
		{
			return true;
		}
	}

	/// Convert a vector<T> to JSON as an array.
	template <class T>
	json::value
		ToJsonValue(
			_In_ const std::vector<T>& Value,
			_In_ const MarshalContext& Context
		)
	{
		return ToJsonValue(Value, Context, ElementModifiers<>());
	}

	/// Convert a vector<T> from JSON as an array.
	template <class T>
	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ std::vector<T>* Result,
			_In_ const UnmarshalContext& Context
		)
	{
		return FromJsonValue(Value, Result, Context, ElementModifiers<>());
	}

	/// Convert a map<wstring, T> to JSON as an object.
	template <class T, class... Modifiers>
	json::value
		ToJsonValue(
			_In_ const std::map<std::wstring, T>& Value,
			_In_ const MarshalContext& Context,
			ElementModifiers<Modifiers...>
		)
	{
		std::vector<std::pair<std::wstring, json::value>> object;
		for (const auto& pair : Value)
		{
			object.emplace_back(pair.first, ToJsonValue(pair.second, Context, Details::DefaultHelper<Modifiers>()...));
		}

		return json::value::object(object);
	}

	/// Convert a map<wstring, T> from JSON as an object.
	template <class T, class... Modifiers>
	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ std::map<std::wstring, T>* Result,
			_In_ const UnmarshalContext& Context,
			ElementModifiers<Modifiers...>
		)
	{
		if (!Value.is_object())
		{
			Context.ReportError(UnmarshalError::ExpectedObject);
			return false;
		}

		const auto& object = Value.as_object();
		Result->clear();

		bool allValid = true;
		for (const auto& pair : object)
		{
			T parsedValue{};
			if (!pair.second.is_null())
			{
				if (!FromJsonValue(pair.second,
					&parsedValue,
					Details::UnmarshalFieldContext(Context, pair.first.c_str()),
					Details::DefaultHelper<Modifiers>()...))
				{
					allValid = false;
				}
			}

			Result->emplace(pair.first, std::move(parsedValue));
		}

		return allValid;
	}

	/// Convert a map<wstring, T> to JSON as an object.
	template <class T>
	json::value
		ToJsonValue(
			_In_ const std::map<std::wstring, T>& Value,
			_In_ const MarshalContext& Context
		)
	{
		return ToJsonValue(Value, Context, ElementModifiers<>());
	}

	/// Convert a map<wstring, T> from JSON as an object.
	template <class T>
	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ std::map<std::wstring, T>* Result,
			_In_ const UnmarshalContext& Context
		)
	{
		return FromJsonValue(Value, Result, Context, ElementModifiers<>());
	}

	/// Convert an Optional<T> to JSON.
	template <class T, class... Modifiers>
	json::value
		ToJsonValue(
			_In_ const vmstd::optional<T>& Value,
			_In_ const MarshalContext& Context,
			ElementModifiers<Modifiers...>
		)
	{
		if (!Value)
		{
			return json::value::null();
		}
		else
		{
			return ToJsonValue(*Value, Context, Details::DefaultHelper<Modifiers>()...);
		}
	}

	/// Convert an Optional<T> from JSON.
	template <class T, class... Modifiers>
	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ vmstd::optional<T>* Result,
			_In_ const UnmarshalContext& Context,
			ElementModifiers<Modifiers...>
		)
	{
		if (Value.is_null())
		{
			*Result = vmstd::nullopt;
			return true;
		}
		else
		{
			*Result = T{};
			return FromJsonValue(Value, &**Result, Context, Details::DefaultHelper<Modifiers>()...);
		}
	}

	/// Convert an Optional<T> to JSON.
	template <class T>
	json::value
		ToJsonValue(
			_In_ const vmstd::optional<T>& Value,
			_In_ const MarshalContext& Context
		)
	{
		return ToJsonValue(Value, Context, ElementModifiers<>());
	}

	/// Convert an Optional<T> from JSON.
	template <class T>
	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ vmstd::optional<T>* Result,
			_In_ const UnmarshalContext& Context
		)
	{
		return FromJsonValue(Value, Result, Context, ElementModifiers<>());
	}

	/// Marshals a value as a single-valued object.
	template <class T, class Name, class... OtherModifiers>
	json::value
		ToJsonValue(
			_In_ const T& Value,
			_In_ const MarshalContext& Context,
			SingleValueObject<Name>,
			OtherModifiers... OtherModifierValues
		)
	{
		return json::value::object({ { std::wstring(Name::Value), ToJsonValue(Value, Context, OtherModifierValues...) } });
	}

	/// Unmarshals a single-valued object to a value.
	template <class T, class Name, class... OtherModifiers>
	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ T* Result,
			_In_ const UnmarshalContext& Context,
			SingleValueObject<Name>,
			OtherModifiers...
		)
	{
		static const FieldData field[]{ { Name::Value, 0, FieldOption::Required } };
		using FieldTypes = TypeList<ModifiedType<T, OtherModifiers...>>;
		using namespace Details;
		return ObjectFromJson(Value,
			Result,
			Context,
			field,
			ObjectTypeInfo<FieldTypes, BaseTypeData, BaseTypeInfo>::Value,
			ObjectTypeInfo<FieldTypes, JsonTypeData, JsonTypeInfo>::Value);
	}

	json::value
		ToJsonValue(
			_In_ const json::value& Value,
			_In_ const MarshalContext& Context
		);

	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ json::value* Result,
			_In_ const UnmarshalContext& Context
		);

	bool
		IsDefault(
			_In_ const json::value& Value
		);

	json::value
		ToJsonValue(
			_In_ const std::vector<UINT8>& Value,
			_In_ const MarshalContext& Context
		);

	bool
		FromJsonValue(
			_In_ const json::value& Value,
			_Inout_ std::vector<UINT8>* Result,
			_In_ const UnmarshalContext& Context
		);

	/// Marshal a value to JSON representation.
	///
	/// \param Value The value to marshal.
	/// \param Flags Flags to change the behavior of the marshaling.
	/// \param ModifierValues Modifiers to change the behavior of the marshaling for
	///     this specific type. The values of these are ignored; only the type is
	///     used.
	template <class T, class... Modifiers>
	json::value
		ToDeserializedJson(
			_In_ T&& Value,
			_In_ MarshalFlags /* Flags */ = MarshalFlags::None,
			Modifiers... ModifierValues
		)
	{
		MarshalContext context;
		return ToJsonValue(std::forward<T>(Value), context, ModifierValues...);
	}

	/// Unmarshal a value from JSON representation.
	///
	/// \param Value The JSON value to unmarshal.
	/// \param Result Returns the result of the unmarshaling.
	/// \param Flags Flags to change the behavior of unmarshaling.
	/// \param ModifierValues Modifiers to change the behavior of the unmarshaling for
	///     this specific type. The values of these are ignored; only the type is
	///     used.
	///
	/// \return A pair<bool, vector<pair<wstring, UnmarshalError>>. The first
	/// value of the pair is whether the unmarshaling succeeded. The second is
	/// a list of validation errors that occured, mapping error location to error
	/// reason.
	template <class T, class... Modifiers>
	std::pair<bool, std::vector<std::pair<std::wstring, UnmarshalError>>>
		FromDeserializedJson(
			_In_ const json::value& Value,
			_Inout_ T* Result,
			_In_ UnmarshalFlags Flags = UnmarshalFlags::None,
			Modifiers... ModifierValues
		)
	{
		Details::UnmarshalRootContext context(Flags);
		bool succeeded = FromJsonValue(Value, Result, context, ModifierValues...);
		return{ succeeded, context.TakeErrors() };
	}

	/// Marshal a value to JSON string.
	///
	/// \param Value The value to marshal.
	/// \param Flags Flags to change the behavior of the marshaling.
	/// \param ModifierValues Modifiers to change the behavior of the marshaling for
	///     this specific type. The values of these are ignored; only the type is
	///     used.
	template <class T, class... Modifiers>
	std::wstring
		ToJson(
			_In_ T&& Value,
			_In_ MarshalFlags Flags = MarshalFlags::None,
			Modifiers... ModifierValues
		)
	{
		return ToDeserializedJson(std::forward<T>(Value), Flags, ModifierValues...).serialize();
	}

	/// Unmarshal a value from JSON representation.
	///
	/// \param String The JSON string value to unmarshal.
	/// \param Result Returns the result of the unmarshaling.
	/// \param Flags Flags to change the behavior of unmarshaling.
	/// \param ModifierValues Modifiers to change the behavior of the unmarshaling for
	///     this specific type. The values of these are ignored; only the type is
	///     used.
	///
	/// \return A pair<bool, vector<pair<wstring, UnmarshalError>>. The first
	/// value of the pair is whether the unmarshaling succeeded. The second is
	/// a list of validation errors that occured, mapping error location to error
	/// reason.
	template <class T, class... Modifiers>
	std::pair<bool, std::vector<std::pair<std::wstring, UnmarshalError>>>
		FromJson(
			_In_ const std::wstring& String,
			_Inout_ T* Result,
			_In_ UnmarshalFlags Flags = UnmarshalFlags::None,
			Modifiers&&... ModifierValues
		)
	{
		json::value value;
		auto result = Details::ParseJson(String, &value);
		if (result.first)
		{
			result = FromDeserializedJson(std::move(value), Result, Flags, ModifierValues...);
		}

		return result;
	}

}

