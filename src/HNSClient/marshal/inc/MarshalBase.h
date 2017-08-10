/**
\copyright Copyright (c) Microsoft Corporation

\file MarshalBase.h

\brief Contains the base infrastructure for marshaling and unmarshaling
static-typed values to and from JSON or XML repository objects.

\author John Starks (jostarks)
\date Apr 23, 2015

Last Updated: 
Sushant Sharma (susharm) 06-April-2017

*/

/**
\mainpage

The marshal package provides methods to marshal C++ objects to and from various
serialization formats. Currently, JSON (in the form of Casablanca web::json::value objects)
and XML (in the form of IXmlReader/IXmlWriter objects) are supported.

See these primary APIs:

- Marshal::ToJson()

- Marshal::FromJson()

- Marshal::ToDeserializedJson()

- Marshal::FromDeserializedJson()

- Marshal::ToXml()

- Marshal::FromXml()

*/

#pragma once

#include <windows.h>
#include <vector>
#include <map>
#include <array>
#include <ntassert.h>
#include "vmoptional.h"

namespace Marshal
{

	/// A list of types. Part of the metadata for a class that can be marshaled.
	template <class... Types> struct TypeList;

	/// A type with marshaling modifiers.
	template <class Type, class... Modifiers> struct ModifiedType {};

	/// The names of marshalable enumeration elements. Used when they are marshaled as
	/// strings.
	struct EnumData
	{
		const PCWSTR* Names;
		const PCWSTR* SortedNames;
		const UINT32* SortedNamesToValues;
		UINT32 Count;
	};

	/// Flags controlling the marshal process.
	enum class MarshalFlags
	{
		None = 0,
	};

	DEFINE_ENUM_FLAG_OPERATORS(MarshalFlags);

	/// Flags controlling the unmarshal process.
	enum class UnmarshalFlags
	{
		None = 0,

		/// In JSON, treat a non-array value as the same as a single array
		/// if necessary to succeed.
		FuzzyArrayMatch = 1 << 0,

		/// In XML, treat attribute and element names as case insensitive.
		/// In JSON, treat the object keys in insensitive manner
		CaseInsensitive = 1 << 1,

		// Fail if there is data present that is not in the schema.
		Strict = 1 << 2,
	};

	DEFINE_ENUM_FLAG_OPERATORS(UnmarshalFlags);

	/// A set of errors that can occur during unmarshaling due to invalid input.
	enum class UnmarshalError
	{
		NoError,
		ExpectedSignedInteger,
		ExpectedUnsignedInteger,
		ExpectedFloat,
		NumberOutOfRange,
		ExpectedObject,
		ExpectedArray,
		ExpectedString,
		UnknownEnumValue,
		ExpectedBoolean,
		InvalidGuidValue,
		MissingRequiredField,
		InvalidBase64,
		InvalidJson,
		InvalidMacAddress,
		InvalidXml,
		DuplicateField,
		UnknownField,

		// Add new elements above this
		Count
	};

	/// Options controlling the marshaling and unmarshaling of an individual field.
	enum class FieldOption : UINT32
	{
		None = 0,
		OmitEmpty = 1u << 0,
		Required = 1u << 1,
		XmlAttribute = 1u << 2,
		XmlValue = 1u << 3,
	};

	DEFINE_ENUM_FLAG_OPERATORS(FieldOption);

	/// Information about a field in an object: its name, its offset within
	/// the object, and any options that should be considering while marshaling
	/// or unmarshaling the field.
	struct FieldData
	{
		PCWSTR Name;
		UINT32 Offset;
		FieldOption Options;
		UINT64 DefaultValue;
	};

	namespace Details
	{
		/// Generates an array of elements of type DataType with an element
		/// for each type in Types. This is used to generate the table of function
		/// pointers for marshaling and unmarshaling objects.
		template <class T, class DataType, template <class, class...> class Transformer>
		struct ObjectTypeInfo;

		template <class DataType, template <class, class...> class Transformer>
		struct ObjectTypeInfo<TypeList<>, DataType, Transformer>
		{
			static std::array<const DataType*, 0> Value;
		};

		template <class DataType, template <class, class...> class Transformer>
		std::array<const DataType*, 0> ObjectTypeInfo<TypeList<>, DataType, Transformer>::Value;

		template <class... Types, class DataType, template <class, class...> class Transformer>
		struct ObjectTypeInfo<TypeList<Types...>, DataType, Transformer>
		{
			static std::array<const DataType*, sizeof...(Types)> Value;
		};

		template <template <class, class...> class Transformer, class T>
		struct TransformerHelper
		{
			using Type = Transformer<T>;
		};

		/// Special case for ModifiedType<T>, which allows modifiers to be added
		/// to the type to get different marshaling routines.
		template <template <class, class...> class Transformer, class T, class... Modifiers>
		struct TransformerHelper<Transformer, ModifiedType<T, Modifiers...>>
		{
			using Type = Transformer<T, Modifiers...>;
		};

		template <class... Types, class DataType, template <class, class...> class Transformer>
		std::array<const DataType*, sizeof...(Types)> ObjectTypeInfo<TypeList<Types...>, DataType, Transformer>::Value =
		{
			&TransformerHelper<Transformer, Types>::Type::Value...
		};

		/// A table of function pointers and an associated generator for determining
		/// if an instance of a type is its default value and for setting it to its
		/// default.
		///
		/// Usually used with ObjectTypeInfo.
		struct BaseTypeData
		{
			bool(*IsDefault)(const void* Value, UINT64 DefaultValue);
			void(*SetDefault)(void* Result, UINT64 DefaultValue);
		};

		template <class T, class... Parameters>
		struct BaseTypeInfo
		{
			static const Details::BaseTypeData Value;
		};

		template <class T, class... Parameters>
		const Details::BaseTypeData BaseTypeInfo<T, Parameters...>::Value = { Details::IsDefaultVoid<T>, Details::SetDefaultVoid<T> };

		/// A class used to store the unmarshal flags and to capture errors that
		/// occur during unmarshaling.
		class UnmarshalRootContext
		{
		public:
			explicit
				UnmarshalRootContext(UnmarshalFlags Flags = UnmarshalFlags::None);

			void
				ReportError(
					_In_ PCWSTR Key,
					_In_ UnmarshalError Error
				);

			std::vector<std::pair<std::wstring, UnmarshalError>> TakeErrors();
			UnmarshalFlags GetFlags() const;

		private:
			std::vector<std::pair<std::wstring, UnmarshalError>> m_Errors;
			UnmarshalFlags m_Flags;
		};

		/// A simple class that provides STL-style container access to an
		/// array.
		///
		/// FUTURE-jostarks: replace with std::array_view once available.
		template <class T>
		class ArrayView
		{
		public:
			ArrayView()
				: m_Begin(nullptr), m_End(nullptr)
			{
			}

			template <size_t n>
			ArrayView(const T(&Array)[n])
				: m_Begin(Array), m_End(Array + n)
			{
			}

			template <size_t n>
			ArrayView(const std::array<T, n>& Array)
				: m_Begin(&*Array.begin()), m_End(&*Array.end())
			{
			}

			const T* begin() { return m_Begin; }
			const T* end() { return m_End; }
			const T& operator[](size_t Index) { return *(begin() + Index); }
			size_t size() const { return m_End - m_Begin; }

		private:
			const T* m_Begin;
			const T* m_End;
		};

		struct NoDefaultValue {};
	}

	/// Converts FieldData::DefaultValue to the appropriate type for type T.
	/// Defaults to returning Details::NoDefaultValue, meaning that the
	/// DefaultValue field is not used.
	template <class T, class = void>
	struct DefaultValueConverter
	{
		static
			Details::NoDefaultValue
			FromTableValue(
				_In_ UINT64 DefaultValue
			)
		{
			NT_ASSERT(DefaultValue == 0);
			UNREFERENCED_PARAMETER(DefaultValue);

			return{};
		};
	};

	/// Specialization for integral types and enums.
	template <class T>
	struct DefaultValueConverter<T, std::enable_if_t<std::is_integral<T>::value || std::is_enum<T>::value>>
	{
		static
			T
			FromTableValue(
				_In_ UINT64 DefaultValue
			)
		{
			return static_cast<T>(DefaultValue);
		}

		static
			UINT64
			ToTableValue(
				_In_ T DefaultValue
			)
		{
			return static_cast<UINT64>(DefaultValue);
		}
	};

	/// Specialization for bool.
	template <>
	struct DefaultValueConverter<bool, void>
	{
		static
			bool
			FromTableValue(
				_In_ UINT64 DefaultValue
			)
		{
			return DefaultValue != 0;
		}

		static
			UINT64
			ToTableValue(
				_In_ bool DefaultValue
			)
		{
			return DefaultValue;
		}
	};

	/// Specialization for double.
	template <>
	struct DefaultValueConverter<double, void>
	{
		static
			double
			FromTableValue(
				_In_ UINT64 DefaultValue
			)
		{
			double asDouble;
			memcpy(&asDouble, &DefaultValue, sizeof(asDouble));
			return asDouble;
		}

		static
			UINT64
			ToTableValue(
				_In_ double DefaultValue
			)
		{
			UINT64 asUint64;
			memcpy(&asUint64, &DefaultValue, sizeof(asUint64));
			return asUint64;
		}
	};

	/// Specialization for string (uses PCWSTR).
	template <>
	struct DefaultValueConverter<std::wstring, void>
	{
		static
			PCWSTR
			FromTableValue(
				_In_ UINT64 DefaultValue
			)
		{
			return reinterpret_cast<PCWSTR>(static_cast<ULONG_PTR>(DefaultValue));
		}

		static
			UINT64
			ToTableValue(
				_In_ PCWSTR DefaultValue
			)
		{
			return reinterpret_cast<ULONG_PTR>(DefaultValue);
		}
	};

	/// A currently-empty class passed to every marshaling function.
	class MarshalContext
	{
	};

	/// Context used during unmarshling to track errors.
	class UnmarshalContext
	{
	public:
		UnmarshalContext(
			_In_ Details::UnmarshalRootContext& RootContext
		);

		void ReportError(
			_In_ UnmarshalError Error
		) const;

		UnmarshalFlags GetFlags() const;

	protected:
		explicit
			UnmarshalContext(
				_In_ const UnmarshalContext& Parent
			);

		virtual void GetKeySuffix(
			_Inout_ std::wstring&
		) const;

	private:
		void
			GetKey(
				_Inout_ std::wstring& Key
			) const;

		Details::UnmarshalRootContext* m_Root = nullptr;
		const UnmarshalContext* m_Parent = nullptr;
	};

	namespace Details
	{
		/// This function exists to work around a VC++ bug with parameter pack expansion.
		/// In the future, replace uses of this with T().
		template <class T>
		T DefaultHelper()
		{
			return T();
		}

		/// Wrapper around IsDefault that is specialized for whether there is
		/// a default value type.
		template <class T, class U>
		bool
			IsDefaultHelper(
				_In_ const T& Value,
				_In_ U&& DefaultValue
			)
		{
			return IsDefault(Value, std::forward<U>(DefaultValue));
		}

		template <class T>
		bool
			IsDefaultHelper(
				_In_ const T& Value,
				_In_ NoDefaultValue
			)
		{
			return IsDefault(Value);
		}

		/// Wrapper around SetDefault that is specialized for whether there is
		/// a default value type.
		template <class T, class U>
		void
			SetDefaultHelper(
				_Inout_ T* Result,
				_In_ U&& DefaultValue
			)
		{
			SetDefault(Result, std::forward<U>(DefaultValue));
		}

		template <class T>
		void
			SetDefaultHelper(
				_Inout_ T* Result,
				_In_ NoDefaultValue
			)
		{
			SetDefault(Result);
		}

		/// Wrapper around IsDefaultHelper that converts the UINT64 default value
		/// into the appropriate type.
		template <class T>
		bool
			IsDefaultVoid(
				_In_ const void* Value,
				_In_ UINT64 DefaultValue
			)
		{
			return IsDefaultHelper(*static_cast<const T*>(Value), DefaultValueConverter<T>::FromTableValue(DefaultValue));
		}

		/// Wrapper around SetDefaultHelper that converts the UINT64 default value
		/// into the appropriate type.
		template <class T>
		void
			SetDefaultVoid(
				_In_ void* Result,
				_In_ UINT64 DefaultValue
			)
		{
			SetDefaultHelper(static_cast<T*>(Result), DefaultValueConverter<T>::FromTableValue(DefaultValue));
		}

		/// SetDefault implementation for marshalable objects.
		template <class T>
		void SetDefaultGeneric(
			_Inout_ T* Result,
			std::true_type /* CanMarshalObject<T> */
		)
		{
			SetObjectFieldDefaults(Result,
				T::FieldData,
				ObjectTypeInfo<T::FieldTypes, BaseTypeData, BaseTypeInfo>::Value);
		}

		/// Default SetDefault implementation for non-marshalable objects and
		/// other types.
		template <class T>
		void SetDefaultGeneric(
			_Inout_ T* Result,
			std::false_type /* CanMarshalObject<T> */
		)
		{
			*Result = T{};
		}

		/// A unmarshal context for an element of an array.
		class UnmarshalIndexContext : public UnmarshalContext
		{
		public:
			UnmarshalIndexContext(
				_In_ const UnmarshalContext& Parent,
				_In_ size_t Index
			);

		protected:
			void
				GetKeySuffix(
					_Inout_ std::wstring& Key
				) const override;

		private:
			size_t m_Index = 0;
		};

		/// An unmarshal context for a field in an object.
		class UnmarshalFieldContext : public UnmarshalContext
		{
		public:
			UnmarshalFieldContext(
				_In_ const UnmarshalContext& Parent,
				_In_ PCWSTR Name
			);

		protected:
			void
				GetKeySuffix(
					_Inout_ std::wstring& Key
				) const override;

		private:
			PCWSTR m_Name = nullptr;
		};

		/// An unmarshal context for a field in an object.
		class UnmarshalSuffixContext : public UnmarshalContext
		{
		public:
			UnmarshalSuffixContext(
				_In_ const UnmarshalContext& Parent,
				_In_ PCWSTR Suffix
			);

		protected:
			void
				GetKeySuffix(
					_Inout_ std::wstring& Key
				) const override;

		private:
			PCWSTR m_Suffix = nullptr;
		};

		std::wstring
			EnumToSimpleValue(
				_In_ UINT32 Value,
				_In_ const EnumData& EnumData
			);

		bool
			EnumFromSimpleValue(
				_In_ const std::wstring& Value,
				_Inout_ UINT32* Result,
				_In_ const UnmarshalContext& Context,
				_In_ const EnumData& EnumData
			);

		void
			SetObjectFieldDefaults(
				_In_ void* Value,
				_In_ ArrayView<FieldData> Fields,
				_In_ ArrayView<const BaseTypeData*> Types
			);
	}

	/// Whether a class can be marshaled as an object. Defaults to false.
	/// Specialized for each class that provides the necessary metadata so that it
	/// can be marshaled.
	///
	/// This function is not actually defined anywhere; only the return type
	/// is used.
	template <class T>
	std::false_type CanMarshalObject(const T&);

	/// Determines whether a value is equal to the default for the type.
	template <class T>
	bool
		IsDefault(
			_In_ const T&
		)
	{
		return false;
	}

	/// Sets a value to its default.
	template <class T>
	void
		SetDefault(
			_Inout_ T* Result
		)
	{
		Details::SetDefaultGeneric(Result, decltype(CanMarshalObject(*Result))());
	}

	/// Determines whether a scalar value is equal to its default.
	template <class T>
	bool
		IsDefault(
			_In_ T Value,
			_In_ typename std::enable_if<std::is_scalar<T>::value, T>::type DefaultValue
		)
	{
		return Value == DefaultValue;
	}

	/// Sets a scalar value to its default.
	template <class T>
	void
		SetDefault(
			_Inout_ T* Result,
			_In_ typename std::enable_if<std::is_scalar<T>::value, T>::type DefaultValue
		)
	{
		*Result = DefaultValue;
	}

	/// Determines whether a vector<T> is equal to its default (empty).
	template <class T>
	bool
		IsDefault(
			_In_ const std::vector<T>& Value
		)
	{
		return Value.empty();
	}

	/// Determines whether a map<T> is equal to its default (empty).
	template <class T>
	bool
		IsDefault(
			_In_ const std::map<std::wstring, T>& Value
		)
	{
		return Value.empty();
	}

	/// Determines whether an Optional<T> is equal to its default (empty).
	template <class T>
	bool
		IsDefault(
			_In_ const vmstd::optional<T>& Value
		)
	{
		return !Value;
	}

	/// Converts an unsigned integer to a UINT64
	template <class T, typename std::enable_if<std::is_unsigned<T>::value>::type* Dummy = nullptr>
	UINT64
		ToSimpleValue(
			_In_ T Value,
			_In_ const MarshalContext&
		)
	{
		return Value;
	}

	/// Converts a UINT64 to an unsigned integer, reporting an error if this would
	/// truncate the result.
	template <class T, typename std::enable_if<std::is_unsigned<T>::value>::type* Dummy = nullptr>
	bool
		FromSimpleValue(
			_In_ UINT64 Value,
			_Inout_ T* Result,
			_In_ const UnmarshalContext& Context
		)
	{
		if (static_cast<T>(Value) != Value)
		{
			Context.ReportError(UnmarshalError::NumberOutOfRange);
			return false;
		}

		*Result = static_cast<T>(Value);
		return true;
	}

	/// Converts a signed integer to an INT64
	template <class T, typename std::enable_if<std::is_signed<T>::value>::type* Dummy = nullptr>
	INT64
		ToSimpleValue(
			_In_ T Value,
			_In_ const MarshalContext&
		)
	{
		return Value;
	}

	/// Converts an INT64 to an signed integer, reporting an error if this would
	/// truncate the result.
	template <class T, typename std::enable_if<std::is_signed<T>::value>::type* Dummy = nullptr>
	bool
		FromSimpleValue(
			_In_ INT64 Value,
			_Inout_ T* Result,
			_In_ const UnmarshalContext& Context
		)
	{
		if (static_cast<T>(Value) != Value)
		{
			Context.ReportError(UnmarshalError::NumberOutOfRange);
			return false;
		}

		*Result = static_cast<T>(Value);
		return true;
	}

	/// Converts an enumeration value to a string using enum metadata.
	template <class T, typename std::enable_if<std::is_enum<T>::value>::type* Dummy = nullptr>
	std::wstring
		ToSimpleValue(
			_In_ T Value,
			_In_ const MarshalContext&
		)
	{
		return Details::EnumToSimpleValue(static_cast<UINT32>(Value), GetEnumData(T()));
	}

	/// Converts a string to an enumeration value using enum metadata.
	template <class T, typename std::enable_if<std::is_enum<T>::value>::type* Dummy = nullptr>
	bool
		FromSimpleValue(
			_In_ const std::wstring& Value,
			_Inout_ T* Result,
			_In_ const UnmarshalContext& Context
		)
	{
		UINT32 result32;
		if (!Details::EnumFromSimpleValue(Value, &result32, Context, GetEnumData(T())))
		{
			return false;
		}

		*Result = static_cast<T>(result32);
		return true;
	}

	double
		ToSimpleValue(
			_In_ double Value,
			_In_ const MarshalContext& Context
		);

	bool
		FromSimpleValue(
			_In_ double Value,
			_Inout_ double* Result,
			_In_ const UnmarshalContext& Context
		);

	bool
		ToSimpleValue(
			_In_ bool Value,
			_In_ const MarshalContext& Context
		);

	bool
		FromSimpleValue(
			_In_ bool Value,
			_Inout_ bool* Result,
			_In_ const UnmarshalContext& Context
		);

	std::wstring
		ToSimpleValue(
			_In_ const std::wstring& Value,
			_In_ const MarshalContext& Context
		);

	bool
		FromSimpleValue(
			_In_ const std::wstring& Value,
			_Inout_ std::wstring* Result,
			_In_ const UnmarshalContext& Context
		);

	bool
		IsDefault(
			_In_ const std::wstring& Value,
			_In_opt_ PCWSTR DefaultValue
		);

	void
		SetDefault(
			_Inout_ std::wstring* Result,
			_In_opt_ PCWSTR DefaultValue
		);

	std::wstring
		ToSimpleValue(
			_In_ REFGUID Value,
			_In_ const MarshalContext& Context
		);

	bool
		FromSimpleValue(
			_In_ const std::wstring& Value,
			_Inout_ GUID* Result,
			_In_ const UnmarshalContext& Context
		);

	/// A modifier that specifies that a GUID should be unmarshaled and marshaled
	/// as a GUID string surrounded in {}, e.g. {0000000-0000-0000-0000-000000000000}
	struct IncludeBraces {};

	std::wstring
		ToSimpleValue(
			_In_ REFGUID Value,
			_In_ const MarshalContext& Context,
			IncludeBraces
		);

	bool
		FromSimpleValue(
			_In_ const std::wstring& Value,
			_Inout_ GUID* Result,
			_In_ const UnmarshalContext& Context,
			IncludeBraces
		);

	/// Modifier used to serialize as an object with a single value of the specified name.
	template <class Name>
	struct SingleValueObject {};

	/// Modifier used to specify modifiers on array entries and map values.
	template <class... Modifiers>
	struct ElementModifiers {};

}

