#pragma once
#include <windows.h>
#include <ntassert.h>

#include <NetSetupApi.h>
#include <NetSetupCxx.h>
#include <utility>
#include <algorithm>

#ifndef WIN_ASSERT_NOTREACHED
#  define WIN_ASSERT_NOTREACHED() WIN_ASSERT(FALSE)
#endif

#ifndef NOTHING
#  define NOTHING
#endif

static const GUID NULL_GUID = {};

#pragma warning(push)
#pragma warning(disable : 4127) // constant in conditional

namespace NetSetup2
{
	std::vector<BYTE> details::StringListToMultiString(_In_ std::vector<std::wstring> const & value)
	{
		std::vector<BYTE> data;

		size_t size = sizeof(WCHAR);
		for (auto const &str : value)
			size += (str.size() + 1) * sizeof(WCHAR);
		data.reserve(size);

		for (auto const &str : value)
		{
			if (str.empty())
			{
				NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
					"A NETSETUPPROP_TYPE_STRING_LIST should not contain an empty string.");
			}

			auto p = reinterpret_cast<BYTE const*>(str.c_str());
			auto len = (str.size() + 1) * sizeof(WCHAR);

			data.insert(data.end(), p, p + len);
		}

		WCHAR empty = L'\0';
		data.insert(
			data.end(),
			reinterpret_cast<BYTE*>(&empty),
			reinterpret_cast<BYTE*>(&empty + 1));

		return data;
	}

	Property::Property(_In_ NETSETUPPROPERTY const &source) :
		m_key(source.CompKey.Key),
		m_type(source.Type),
		m_data(source.Buffer, source.Buffer + source.BufferSize)
	{

	}

	Property::Property(_In_ Property const &source) :
		m_key(source.m_key),
		m_type(source.m_type),
		m_data(source.m_data)
	{

	}

	Property::Property(_In_ Property &&source) :
		m_key(source.m_key),
		m_type(source.m_type),
		m_data(std::move(source.m_data))
	{
		source.m_key = {};
		source.m_type = NETSETUPPROP_TYPE_EMPTY;
	}

	Property &
		Property::operator=(
			_In_ Property const &source)
	{
		m_key = source.m_key;
		m_type = source.m_type;
		m_data = source.m_data;
		return *this;
	}

	Property &
		Property::operator=(
			_In_ Property &&source)
	{
		m_key = source.m_key;
		m_type = source.m_type;
		m_data = std::move(source.m_data);

		source.m_key = {};
		source.SetEmpty();

		return *this;
	}

	bool
		Property::operator==(_In_ Property const & other) const
	{
		if (this->m_type != other.m_type)
		{
			return false;
		}

		return this->m_data == other.m_data;
	}

	bool
		Property::operator!=(_In_ Property const & other) const
	{
		return !(*this == other);
	}

	bool
		Property::operator<(_In_ Property const & right) const
	{
		auto & left = *this;

		if (left.m_type != right.m_type)
		{
			return left.m_type < right.m_type;
		}

		switch (m_type)
		{
		case NETSETUPPROP_TYPE_AGGREGATE:
		case NETSETUPPROP_TYPE_BINARY:
			return left.GetByteArray() < right.GetByteArray();
		case NETSETUPPROP_TYPE_OBJECT_ID_ARRAY:
			// case NETSETUPPROP_TYPE_BIND_PATH: // same as OBJECT_ID_ARRAY
			return left.GetObjectIdArray() < right.GetObjectIdArray();
		case NETSETUPPROP_TYPE_BOOLEAN:
			return left.GetBoolean() < right.GetBoolean();
		case NETSETUPPROP_TYPE_BOOLEAN_ARRAY:
			return left.GetBooleanArray() < right.GetBooleanArray();
		case NETSETUPPROP_TYPE_BYTE:
			return left.GetByte() < right.GetByte();
		case NETSETUPPROP_TYPE_EMPTY:
			return false;
		case NETSETUPPROP_TYPE_GUID:
			return left.m_data < right.m_data;
		case NETSETUPPROP_TYPE_GUID_ARRAY:
			return left.m_data < right.m_data;
		case NETSETUPPROP_TYPE_NULL:
			return false;
		case NETSETUPPROP_TYPE_OBJECT_ID:
			return left.GetObjectId() < right.GetObjectId();
		case NETSETUPPROP_TYPE_PROPERTY_KEY:
			return left.GetValueAsPropertyKey() < right.GetValueAsPropertyKey();
		case NETSETUPPROP_TYPE_PROPERTY_KEY_ARRAY:
			return left.GetValueAsPropertyKeyArray() < right.GetValueAsPropertyKeyArray();
		case NETSETUPPROP_TYPE_STRING:
		case NETSETUPPROP_TYPE_STRING_INDIRECT:
			return left.GetString() < right.GetString();
		case NETSETUPPROP_TYPE_STRING_LIST:
			return left.GetStringArray() < right.GetStringArray();
		case NETSETUPPROP_TYPE_UINT32:
			return left.GetUint32() < right.GetUint32();
		case NETSETUPPROP_TYPE_UINT32_ARRAY:
			return left.GetUint32Array() < right.GetUint32Array();
		case NETSETUPPROP_TYPE_UINT64:
			return left.GetUint64() < right.GetUint64();
		case NETSETUPPROP_TYPE_UINT64_ARRAY:
			return left.GetUint64Array() < right.GetUint64Array();
		default:
			WIN_ASSERT_NOTREACHED();
			return false;
		}
	}

	bool
		Property::operator<=(_In_ Property const & right) const
	{
		return (*this < right) || (*this == right);
	}

	bool
		Property::operator>(_In_ Property const & right) const
	{
		return right < *this;
	}

	bool
		Property::operator>=(_In_ Property const & right) const
	{
		return (*this > right) || (*this == right);
	}

	bool
		Property::IsReadOnly(_In_ NetSetup2::details::TransactionBase & transaction) const
	{
		return transaction.ReflectOnProperty(GetKey()).get_IsReadOnly();
	}

	bool
		Property::EqualsDefaultValue(_In_ NetSetup2::details::TransactionBase & transaction) const
	{
		if (auto defaultValue = transaction.ReflectOnProperty(GetKey()).tryget_DefaultValue())
		{
			if (*defaultValue == GetRawData())
			{
				return true;
			}
		}

		return false;
	}

	NETSETUPPROPKEY const &
		Property::GetKey() const
	{
		return m_key;
	}

	NETSETUPPROPTYPE
		Property::GetType() const
	{
		return m_type;
	}

	std::vector<BYTE> const &
		Property::GetRawData() const
	{
		return m_data;
	}

	void
		Property::SetRawData(_In_ NETSETUPPROPTYPE type, _In_ std::vector<BYTE> const &data)
	{
		m_type = type;
		m_data = data;
	}

	void
		Property::SetRawData(_In_ NETSETUPPROPTYPE type, _Inout_ std::vector<BYTE> &&data)
	{
		m_type = type;
		m_data = std::move(data);
	}

	void
		Property::SetRawData(_In_ NETSETUPPROPTYPE type, _In_reads_(cbData) BYTE const *data, _In_ ULONG cbData)
	{
		m_type = type;
		m_data.assign(data, data + cbData);
	}

	void
		Property::Store(
			_Out_ NETSETUPPROPERTY &destination) const
	{
		destination.CompKey.Key = m_key;
		destination.CompKey.Store = NETSETUPPROP_STORE_SYSTEM;
		destination.CompKey.LocaleName = nullptr;
		destination.Type = m_type;
		destination.BufferSize = static_cast<ULONG>(m_data.size());

		if (m_data.empty())
		{
			destination.Buffer = nullptr;
		}
		else
		{
			destination.Buffer = const_cast<BYTE*>(m_data.data());
		}
	}

	void
		Property::Load(
			_In_ NETSETUPPROPERTY const &source)
	{
		m_key = source.CompKey.Key;
		m_type = source.Type;

		if (source.BufferSize > 0)
		{
			m_data.assign(source.Buffer, source.Buffer + source.BufferSize);
		}
		else
		{
			m_data.clear();
		}
	}

	bool
		Property::IsEmpty() const
	{
		return m_type == NETSETUPPROP_TYPE_EMPTY;
	}

	void
		Property::SetEmpty()
	{
		m_type = NETSETUPPROP_TYPE_EMPTY;
		m_data.clear();
	}

	bool
		Property::IsNull() const
	{
		return m_type == NETSETUPPROP_TYPE_NULL;
	}

	void
		Property::SetNull()
	{
		m_type = NETSETUPPROP_TYPE_NULL;
		m_data.clear();
	}

	bool
		Property::GetBoolean() const
	{
		if (m_type != NETSETUPPROP_TYPE_BOOLEAN)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_BOOLEAN", m_type);
		}

		if (m_data.size() != 1)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_BOOLEAN", m_data.size());
		}

		switch (m_data[0])
		{
		case 0:
			return false;
		case 1:
			return true;
		default:
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid payload %u for NETSETUPPROP_TYPE_BOOLEAN", m_data[0]);
		}
	}

	bool
		Property::GetBooleanOrDefault() const
	{
		if (IsEmpty())
			return false;
		return GetBoolean();
	}

	void
		Property::SetBoolean(_In_ bool value)
	{
		m_type = NETSETUPPROP_TYPE_BOOLEAN;
		m_data.resize(1);
		m_data[0] = value ? 1 : 0;
	}

	std::vector<bool>
		Property::GetBooleanArray() const
	{
		if (m_type != NETSETUPPROP_TYPE_BOOLEAN_ARRAY)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_BOOLEAN_ARRAY", m_type);
		}

		if (m_data.size() == 0)
		{
			return std::vector<bool>();
		}

		std::vector<bool> result;
		result.reserve(m_data.size());

		for (auto value : m_data)
		{
			result.push_back(!!value);
		}

		return result;
	}

	std::vector<bool>
		Property::GetBooleanArrayOrDefault() const
	{
		if (IsEmpty())
			return{};
		return GetBooleanArray();
	}

	void
		Property::SetBooleanArray(_In_ std::vector<bool> const &value)
	{
		m_type = NETSETUPPROP_TYPE_BOOLEAN_ARRAY;
		m_data.resize(value.size());
		for (size_t i = 0; i < value.size(); i++)
			m_data[i] = value[i];
	}

	BYTE
		Property::GetByte() const
	{
		if (m_type != NETSETUPPROP_TYPE_BYTE)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_BYTE", m_type);
		}

		if (m_data.size() != sizeof(BYTE))
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_BYTE", m_data.size());
		}

		return m_data[0];
	}

	BYTE
		Property::GetByteOrDefault() const
	{
		if (IsEmpty())
			return 0;
		return GetByte();
	}

	void
		Property::SetByte(_In_ BYTE value)
	{
		m_type = NETSETUPPROP_TYPE_BYTE;
		m_data.assign(&value, &value + 1);
	}

	std::vector<BYTE> const &
		Property::GetByteArray() const
	{
		if (m_type != NETSETUPPROP_TYPE_BINARY)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_BINARY", m_type);
		}

		return m_data;
	}

	std::vector<BYTE> const
		Property::GetByteArrayOrDefault() const
	{
		if (IsEmpty())
			return{};
		return GetByteArray();
	}

	void
		Property::SetByteArray(_In_ std::vector<BYTE> const &value)
	{
		m_type = NETSETUPPROP_TYPE_BINARY;
		m_data = value;
	}

	void
		Property::SetByteArray(_In_ std::vector<BYTE> &&value)
	{
		m_type = NETSETUPPROP_TYPE_BINARY;
		m_data = std::move(value);
	}

	ULONG
		Property::GetUint32() const
	{
		if (m_type != NETSETUPPROP_TYPE_UINT32)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_UINT32", m_type);
		}

		if (m_data.size() != sizeof(ULONG))
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_UINT32", m_data.size());
		}

		return *reinterpret_cast<ULONG const*>(m_data.data());
	}

	ULONG
		Property::GetUint32OrDefault() const
	{
		if (IsEmpty())
			return 0;
		return GetUint32();
	}

	void
		Property::SetUint32(_In_ ULONG value)
	{
		m_type = NETSETUPPROP_TYPE_UINT32;
		m_data.assign((BYTE*)&value, (BYTE*)(&value + 1));
	}

	std::vector<ULONG>
		Property::GetUint32Array() const
	{
		if (m_type != NETSETUPPROP_TYPE_UINT32_ARRAY)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_UINT32_ARRAY", m_type);
		}

		if (m_data.size() == 0)
		{
			return std::vector<ULONG>();
		}

		if (m_data.size() % sizeof(ULONG) != 0)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_UINT32_ARRAY", m_data.size());
		}

		return std::vector<ULONG>((ULONG*)&*m_data.begin(), (ULONG*)&*m_data.end());
	}

	std::vector<ULONG>
		Property::GetUint32ArrayOrDefault() const
	{
		if (IsEmpty())
			return{};
		return GetUint32Array();
	}

	void
		Property::SetUint32Array(_In_ std::vector<ULONG> const &value)
	{
		m_type = NETSETUPPROP_TYPE_UINT32_ARRAY;
		m_data.assign(
			(BYTE const*)value.data(),
			(BYTE const*)value.data() + value.size() * sizeof(ULONG));
	}

	ULONG64
		Property::GetUint64() const
	{
		if (m_type != NETSETUPPROP_TYPE_UINT64)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_UINT64", m_type);
		}

		if (m_data.size() != sizeof(ULONG64))
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_UINT64", m_data.size());
		}

		return *reinterpret_cast<ULONG64 UNALIGNED const*>(m_data.data());
	}

	ULONG64
		Property::GetUint64OrDefault() const
	{
		if (IsEmpty())
			return 0;
		return GetUint64();
	}

	void
		Property::SetUint64(_In_ ULONG64 value)
	{
		m_type = NETSETUPPROP_TYPE_UINT64;
		m_data.assign((BYTE*)&value, (BYTE*)(&value + 1));
	}

	std::vector<ULONG64>
		Property::GetUint64Array() const
	{
		if (m_type != NETSETUPPROP_TYPE_UINT64_ARRAY)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_UINT64_ARRAY", m_type);
		}

		if (m_data.size() == 0)
		{
			return std::vector<ULONG64>();
		}

		if (m_data.size() % sizeof(ULONG64) != 0)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_UINT64_ARRAY", m_data.size());
		}

		return std::vector<ULONG64>((ULONG64*)&*m_data.begin(), (ULONG64*)&*m_data.end());
	}

	std::vector<ULONG64>
		Property::GetUint64ArrayOrDefault() const
	{
		if (IsEmpty())
			return{};
		return GetUint64Array();
	}

	void
		Property::SetUint64Array(_In_ std::vector<ULONG64> const &value)
	{
		m_type = NETSETUPPROP_TYPE_UINT64_ARRAY;
		m_data.assign(
			(BYTE const*)value.data(),
			(BYTE const*)value.data() + value.size() * sizeof(ULONG64));
	}

	GUID
		Property::GetGuid() const
	{
		if (m_type != NETSETUPPROP_TYPE_GUID)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_GUID", m_type);
		}

		if (m_data.size() != sizeof(GUID))
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_GUID", m_data.size());
		}

		return *reinterpret_cast<GUID const*>(&m_data[0]);
	}

	GUID
		Property::GetGuidOrDefault() const
	{
		if (IsEmpty())
			return NULL_GUID;
		return GetGuid();
	}

	void
		Property::SetGuid(_In_ GUID const &value)
	{
		m_type = NETSETUPPROP_TYPE_GUID;
		m_data.resize(sizeof(GUID));
		m_data.assign((BYTE const*)&value, (BYTE const*)&value + sizeof(value));
	}

	std::vector<GUID>
		Property::GetGuidArray() const
	{
		if (m_type != NETSETUPPROP_TYPE_GUID_ARRAY)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_GUID_ARRAY", m_type);
		}

		if (m_data.size() == 0)
		{
			return std::vector<GUID>();
		}

		if (m_data.size() % sizeof(GUID) != 0)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_GUID_ARRAY", m_data.size());
		}

		return std::vector<GUID>((GUID*)&*m_data.begin(), (GUID*)&*m_data.end());
	}

	std::vector<GUID>
		Property::GetGuidArrayOrDefault() const
	{
		if (IsEmpty())
			return{};
		return GetGuidArray();
	}

	void
		Property::SetGuidArray(_In_ std::vector<GUID> const &value)
	{
		m_type = NETSETUPPROP_TYPE_GUID_ARRAY;
		m_data.assign(
			(BYTE const*)value.data(),
			(BYTE const*)value.data() + value.size() * sizeof(GUID));
	}

	std::wstring
		Property::GetString() const
	{

		if (m_type != NETSETUPPROP_TYPE_STRING && m_type != NETSETUPPROP_TYPE_STRING_INDIRECT)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_STRING", m_type);
		}

		if (m_data.empty())
		{
			return{};
		}

		if (m_data.size() % sizeof(WCHAR) != 0)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_STRING", m_data.size());
		}

		if (m_data[m_data.size() - 2] != 0 || m_data[m_data.size() - 1] != 0)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE, "String is not null-terminated");
		}

		PCWSTR begin = (PCWSTR)&m_data[0];
		PCWSTR end = (PCWSTR)(begin + m_data.size() / sizeof(WCHAR));

		return{ begin, end - 1 };
	}

	std::wstring
		Property::GetStringOrDefault() const
	{
		if (IsEmpty())
			return{};
		return GetString();
	}

	void
		Property::SetString(_In_opt_ PCWSTR value)
	{
		m_type = NETSETUPPROP_TYPE_STRING;

		if (!value)
			value = L"";

		m_data.assign(
			(BYTE const*)value,
			(BYTE const*)(value + wcslen(value) + 1));
	}

	void
		Property::SetString(_In_ std::wstring const &value)
	{
		m_type = NETSETUPPROP_TYPE_STRING;
		m_data.assign(
			(BYTE const*)value.c_str(),
			(BYTE const*)(value.c_str() + value.size() + 1));
	}

	void
		Property::SetIndirectString(_In_opt_ PCWSTR value)
	{
		m_type = NETSETUPPROP_TYPE_STRING_INDIRECT;

		if (!value)
			value = L"";

		m_data.assign(
			(BYTE const*)value,
			(BYTE const*)(value + wcslen(value) + 1));
	}

	void
		Property::SetIndirectString(_In_ std::wstring const &value)
	{
		m_type = NETSETUPPROP_TYPE_STRING_INDIRECT;
		m_data.assign(
			(BYTE const*)value.c_str(),
			(BYTE const*)(value.c_str() + value.size() + 1));
	}

	std::vector<std::wstring>
		Property::GetStringArray() const
	{
		std::vector<std::wstring> result;

		if (m_type == NETSETUPPROP_TYPE_EMPTY)
		{
			return result;
		}

		if (m_type != NETSETUPPROP_TYPE_STRING_LIST)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_STRING_LIST", m_type);
		}

		if (m_data.empty())
		{
			return result;
		}

		if (m_data.size() % sizeof(WCHAR) != 0)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_STRING_LIST", m_data.size());
		}

		PCWSTR begin = (PCWSTR)&m_data[0];
		PCWSTR end = (PCWSTR)(begin + m_data.size() / sizeof(WCHAR));

		PCWSTR str = begin;

		while (*str)
		{
			PCWSTR cur = str;
			while (*cur)
			{
				++cur;

				if (cur >= end)
				{
					NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
						"Property key is missing null-terminator for NETSETUPPROP_TYPE_STRING_LIST");
				}
			}

			result.push_back(std::wstring(str, cur));

			str = cur + 1;

			if (str >= end)
			{
				NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
					"Property key is missing final null-terminator for NETSETUPPROP_TYPE_STRING_LIST");
			}
		}

		return result;
	}

	std::vector<std::wstring>
		Property::GetStringArrayOrDefault() const
	{
		if (IsEmpty())
			return{};
		return GetStringArray();
	}

	void
		Property::SetStringArray(_In_ std::vector<std::wstring> const &value)
	{
		m_type = NETSETUPPROP_TYPE_STRING_LIST;
		m_data = details::StringListToMultiString(value);
	}

	NETSETUP_OBJECT_ID
		Property::GetObjectId() const
	{
		if (m_type != NETSETUPPROP_TYPE_OBJECT_ID)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_OBJECT_ID", m_type);
		}

		if (m_data.size() != sizeof(NETSETUP_OBJECT_ID))
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_OBJECT_ID", m_data.size());
		}

		return *reinterpret_cast<NETSETUP_OBJECT_ID const*>(&m_data[0]);
	}

	NETSETUP_OBJECT_ID
		Property::GetObjectIdOrDefault() const
	{
		if (IsEmpty())
			return{};
		return GetObjectId();
	}

	void
		Property::SetObjectId(_In_ NETSETUP_OBJECT_ID const &value)
	{
		m_type = NETSETUPPROP_TYPE_OBJECT_ID;
		m_data.assign(
			(BYTE const*)&value,
			(BYTE const*)(&value + 1));
	}

	std::vector<NETSETUP_OBJECT_ID>
		Property::GetObjectIdArray() const
	{
		if (m_type != NETSETUPPROP_TYPE_BIND_PATH)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_BIND_PATH", m_type);
		}

		if (m_data.size() == 0)
		{
			return std::vector<NETSETUP_OBJECT_ID>();
		}

		if (m_data.size() % sizeof(NETSETUP_OBJECT_ID) != 0)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_BIND_PATH", m_data.size());
		}

		return std::vector<NETSETUP_OBJECT_ID>(
			(NETSETUP_OBJECT_ID*)&*m_data.begin(),
			(NETSETUP_OBJECT_ID*)&*m_data.end());
	}

	std::vector<NETSETUP_OBJECT_ID>
		Property::GetObjectIdArrayOrDefault() const
	{
		if (IsEmpty())
			return{};
		return GetObjectIdArray();
	}

	void
		Property::SetObjectIdArray(_In_ std::vector<NETSETUP_OBJECT_ID> const &value)
	{
		m_type = NETSETUPPROP_TYPE_BIND_PATH;
		m_data.assign(
			(BYTE const*)value.data(),
			(BYTE const*)(value.data() + value.size()));
	}

	NETSETUPPROPKEY
		Property::GetValueAsPropertyKey() const
	{
		if (m_type != NETSETUPPROP_TYPE_PROPERTY_KEY)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_PROPERTY_KEY", m_type);
		}

		if (m_data.size() != sizeof(NETSETUPPROPKEY))
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_PROPERTY_KEY", m_data.size());
		}

		return *reinterpret_cast<NETSETUPPROPKEY const*>(&m_data[0]);
	}

	void
		Property::SetValueAsPropertyKey(_In_ NETSETUPPROPKEY const &value)
	{
		m_type = NETSETUPPROP_TYPE_PROPERTY_KEY;
		m_data.assign(
			(BYTE const*)&value,
			(BYTE const*)(&value + 1));
	}

	std::vector<NETSETUPPROPKEY>
		Property::GetValueAsPropertyKeyArray() const
	{
		if (m_type != NETSETUPPROP_TYPE_PROPERTY_KEY_ARRAY)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_DATATYPE_MISMATCH), STORE,
				"Property key has type %u; expected NETSETUPPROP_TYPE_PROPERTY_KEY_ARRAY", m_type);
		}

		if (m_data.size() == 0)
		{
			return std::vector<NETSETUPPROPKEY>();
		}

		if (m_data.size() % sizeof(NETSETUPPROPKEY) != 0)
		{
			NETSETUP_THROW_EX(HRESULT_FROM_WIN32(ERROR_INVALID_DATA), STORE,
				"Property key has invalid length %Iu for NETSETUPPROP_TYPE_PROPERTY_KEY_ARRAY", m_data.size());
		}

		return std::vector<NETSETUPPROPKEY>(
			(NETSETUPPROPKEY*)&*m_data.begin(),
			(NETSETUPPROPKEY*)&*m_data.end());
	}

	std::vector<NETSETUPPROPKEY>
		Property::GetValueAsPropertyKeyArrayOrDefault() const
	{
		if (IsEmpty())
			return{};
		return GetValueAsPropertyKeyArray();
	}

	void
		Property::SetValueAsPropertyKeyArray(_In_ std::vector<NETSETUPPROPKEY> const &value)
	{
		m_type = NETSETUPPROP_TYPE_PROPERTY_KEY_ARRAY;
		m_data.assign(
			(BYTE const*)value.data(),
			(BYTE const*)(value.data() + value.size()));
	}

	PropertyArray::PropertyArray(
		_In_ NETSETUP_OBJECT const & object) :
		PropertyArray(object.NumProperties, object.Properties)
	{
		NOTHING;
	}

	PropertyArray::PropertyArray(
		_In_ ULONG NumProperties,
		_In_reads_(NumProperties) NETSETUPPROPERTY const *Properties)
	{
		reserve(NumProperties);

		for (ULONG i = 0; i < NumProperties; i++)
			emplace_back(Properties[i]);
	}

	Property &
		PropertyArray::GetProperty(
			_In_ NETSETUPPROPKEY const & key)
	{
		auto iter = std::find_if(begin(), end(), [=](Property const &prop) {
			return prop.GetKey() == key;
		});

		if (iter == end())
			NETSETUP_THROW(HRESULT_FROM_WIN32(ERROR_NO_DATA));

		return *iter;
	}

	Property const &
		PropertyArray::GetProperty(
			_In_ NETSETUPPROPKEY const & key) const
	{
		auto iter = std::find_if(begin(), end(), [=](Property const &prop) {
			return prop.GetKey() == key;
		});

		if (iter == end())
			NETSETUP_THROW(HRESULT_FROM_WIN32(ERROR_NO_DATA));

		return *iter;
	}

	details::Optional<Property &>
		PropertyArray::TryGetProperty(
			_In_ NETSETUPPROPKEY const & key)
	{
		auto iter = std::find_if(begin(), end(), [=](Property const &prop) {
			return prop.GetKey() == key;
		});

		if (iter == end())
			return{};

		return{ *iter };
	}

	details::Optional<Property const&>
		PropertyArray::TryGetProperty(
			_In_ NETSETUPPROPKEY const & key) const
	{
		auto iter = std::find_if(begin(), end(), [=](Property const &prop) {
			return prop.GetKey() == key;
		});

		if (iter == end())
			return{};

		return{ *iter };
	}

	bool
		PropertyArray::HasKey(
			_In_ NETSETUPPROPKEY const & key) const
	{
		return this->TryGetProperty(key);
	}

	GUID
		ObjectCreationTemplate::Create(
			_In_ details::TransactionBase &transaction,
			_In_ NETSETUP_OBJECT_TYPE type)
	{
		std::vector<NETSETUPPROPERTY> properties(m_initialProperties.size());

		for (size_t i = 0; i < m_initialProperties.size(); i++)
			m_initialProperties[i].Store(properties[i]);

		GUID id;
		HRESULT hr = NetSetupCreateObject(
			transaction,
			type,
			static_cast<ULONG>(properties.size()),
			properties.data(),
			&id);
		IFFAILED_THROW(hr);

		return id;
	}

	Property
		ObjectCreationTemplate::GetProperty(_In_ NETSETUPPROPKEY const & key)
	{
		auto property = m_initialProperties.TryGetProperty(key);
		return property ? *property : Property{};
	}

	Property
		ObjectCreationTemplate::GetProperty(_In_ NETSETUPPROPKEY const & key) const
	{
		auto property = m_initialProperties.TryGetProperty(key);
		return property ? *property : Property{ key };
	}

	void
		ObjectCreationTemplate::DeleteProperty(_In_ NETSETUPPROPKEY const &key)
	{
		auto end = std::remove_if(m_initialProperties.begin(), m_initialProperties.end(), [&](Property const &prop)
		{
			return prop.GetKey() == key;
		});

		m_initialProperties.erase(end, m_initialProperties.end());
	}

	void
		ObjectCreationTemplate::SetProperty(_In_ Property const &property)
	{
		if (property.IsEmpty())
		{
			DeleteProperty(property.GetKey());
		}
		else
		{
			if (auto existing = m_initialProperties.TryGetProperty(property.GetKey()))
				*existing = property;
			else
				m_initialProperties.emplace_back(property);
		}
	}

	void
		ObjectCreationTemplate::SetProperty(_In_ Property &&property)
	{
		if (property.IsEmpty())
		{
			DeleteProperty(property.GetKey());
		}
		else
		{
			if (auto existing = m_initialProperties.TryGetProperty(property.GetKey()))
				*existing = std::move(property);
			else
				m_initialProperties.emplace_back(std::move(property));
		}
	}

	void
		ObjectCreationTemplate::WritePropertiesToExistingObject(_In_ ActiveObject &target) const
	{
		for (auto &property : m_initialProperties)
		{
			auto existingValue = target.GetProperty(property.GetKey());
			if (existingValue != property)
				target.SetProperty(property);
		}
	}

	PropertyArray &
		ObjectCreationTemplate::properties()
	{
		return m_initialProperties;
	}

	ActiveObject::ActiveObject(
		_In_ details::TransactionBase &transaction,
		_In_ NETSETUP_OBJECT_TYPE type,
		_In_ GUID const &id) :
		m_transaction(transaction),
		m_type(type),
		m_id(id)
	{

	}

	ActiveObject::ActiveObject(
		_In_ details::TransactionBase &transaction,
		_In_ NETSETUP_OBJECT_ID const &id) :
		m_transaction(transaction),
		m_type(id.ObjectType),
		m_id(id.ObjectId)
	{

	}

	ActiveObject::ActiveObject(
		_In_ details::TransactionBase &transaction,
		_In_ NETSETUP_OBJECT const &object) :
		m_transaction(transaction),
		m_type(object.ObjectType),
		m_id(object.ObjectId)
	{

	}

	void
		ActiveObject::Delete()
	{
		HRESULT hr = NetSetupDeleteObject(
			m_transaction,
			m_type,
			m_id);
		IFFAILED_THROW(hr);
	}

	std::vector<NETSETUPPROPKEY>
		ActiveObject::GetPropertyKeys() const
	{

		ULONG numReturned, numTotal;
		IFFAILED_THROW(NetSetupGetObjectPropertyKeys(m_transaction, m_type, m_id, nullptr, 0, &numReturned, &numTotal));

		std::vector<NETSETUPPROPKEY> keys(numTotal);

		IFFAILED_THROW(NetSetupGetObjectPropertyKeys(m_transaction, m_type, m_id, keys.data(), (ULONG)keys.size(), &numReturned, &numTotal));

		return keys;
	}


	NETSETUPPROPTYPE
		ActiveObject::GetPropertyType(_In_ NETSETUPPROPKEY const & key) const
	{
		return GetProperty(key).GetType();
	}

	Property
		ActiveObject::GetProperty(_In_ NETSETUPPROPKEY const &key, ULONG QueryFlags) const
	{
		NETSETUPPROPCOMPKEY comp = { key };
		NETSETUPPROPERTY const *result = nullptr;
		ULONG numReturned;
		HRESULT hr = NetSetupGetObjectProperties(
			m_transaction,
			m_type,
			m_id,
			QueryFlags,
			1, &comp,
			&numReturned, &result);
		IFFAILED_THROW(hr);

		details::AutoFreeProperties afp(numReturned, result);

		if (!NETSETUP_VERIFY(numReturned == 1))
		{
			NETSETUP_THROW_EX(E_UNEXPECTED, STORE,
				"NetSetupGetObjectProperties returned fewer properties than requested");
		}

		return Property{ result[0] };
	}

	Property
		ActiveObject::GetProperty(_In_ NETSETUPPROPKEY const &key) const
	{
		return GetProperty(key, 0);
	}

	Property
		ActiveObject::GetPropertyExpanded(_In_ NETSETUPPROPKEY const & key) const
	{
		return GetProperty(key, NETSETUP_RESOLVE_INDIRECT_STRINGS);
	}

	std::vector<Property>
		ActiveObject::GetProperties(_In_ std::vector<NETSETUPPROPKEY> const &keys, ULONG QueryFlags) const
	{
		if (keys.empty())
			return{};

		std::vector<NETSETUPPROPCOMPKEY> comp;
		comp.reserve(keys.size());
		for (auto &key : keys)
			comp.push_back({ key });

		NETSETUPPROPERTY const *result = nullptr;
		ULONG numReturned;
		HRESULT hr = NetSetupGetObjectProperties(
			m_transaction,
			m_type,
			m_id,
			QueryFlags,
			static_cast<ULONG>(comp.size()), comp.data(),
			&numReturned, &result);
		IFFAILED_THROW(hr);

		details::AutoFreeProperties afp(numReturned, result);

		if (!NETSETUP_VERIFY(numReturned == keys.size()))
		{
			NETSETUP_THROW_EX(E_UNEXPECTED, STORE,
				"NetSetupGetObjectProperties returned fewer properties than requested");
		}

		std::vector<Property> properties;
		properties.reserve(numReturned);
		for (ULONG i = 0; i < numReturned; ++i)
			properties.emplace_back(result[i]);

		return properties;
	}

	void
		ActiveObject::SetProperty(_In_ Property&& property)
	{
		this->SetPropertyUniversal(std::move(property));
	}

	void
		ActiveObject::SetProperty(_In_ Property const &property)
	{
		this->SetPropertyUniversal(property);
	}

	void
		ActiveObject::SetRawProperty(_In_ NETSETUPPROPERTY const & property)
	{
		if (m_queue)
		{
			m_queue->emplace_back(property);
		}
		else
		{
			auto hr = NetSetupSetObjectProperties(
				m_transaction,
				m_type,
				m_id,
				1ul,
				&property);
			IFFAILED_THROW(hr);
		}
	}

	void
		ActiveObject::QueueProperties(_In_ size_t sizeHint)
	{
		NETSETUP_ASSERT(!m_queue);

		m_queue.reset(new std::vector<Property>());
		m_queue->reserve(sizeHint);
	}

	void
		ActiveObject::SetProperties(_In_ std::vector<Property> const & properties)
	{
		if (m_queue)
		{
			m_queue->reserve(m_queue->size() + properties.size());

			for (auto & property : properties)
			{
				m_queue->emplace_back(property);
			}
		}
		else
		{
			SetMultipleProperties(properties);
		}
	}

	void
		ActiveObject::SetProperties(_In_ std::vector<Property> && properties)
	{
		if (m_queue)
		{
			m_queue->reserve(m_queue->size() + properties.size());

			for (auto&& property : properties)
			{
				m_queue->emplace_back(std::move(property));
			}
		}
		else
		{
			SetMultipleProperties(std::move(properties));
		}
	}

	void
		ActiveObject::FlushQueuedProperties()
	{
		ClearQueue clears(*this);

		if (m_queue)
		{
			SetMultipleProperties(*m_queue);
		}
	}

	void
		ActiveObject::CancelQueuedProperties()
	{
		ClearQueue clears(*this);
	}

	bool
		ActiveObject::Matches(NETSETUPPROP_FILTER_EXPRESSION const * begin, NETSETUPPROP_FILTER_EXPRESSION const * end) const
	{
		return m_transaction.TryMatchObjectById(NETSETUP_OBJECT_ID{ this->m_type, this->m_id }, begin, end);
	}

	void
		ActiveObject::SetMultipleProperties(std::vector<Property> const & properties)
	{
		std::vector<NETSETUPPROPERTY> outputProperties;
		outputProperties.reserve(properties.size());

		for (auto & property : properties)
		{
			NETSETUPPROPERTY* storage;

			auto existingProperty = std::find_if(outputProperties.begin(), outputProperties.end(), [&property](NETSETUPPROPERTY const & prop) {
				return prop.CompKey.Key == property.GetKey();
			});

			if (existingProperty == outputProperties.end())
			{
				outputProperties.emplace_back();
				storage = &outputProperties.back();
			}
			else
			{
				storage = &(*existingProperty);
			}

			property.Store(*storage);
		}

		HRESULT hr = NetSetupSetObjectProperties(
			static_cast<HNETSETUP>(m_transaction),
			m_type,
			m_id,
			static_cast<ULONG>(outputProperties.size()),
			outputProperties.data());
		IFFAILED_THROW(hr);
	}

	void
		Mux::AddMember(
			_In_ NetworkInterface &networkInterface) const
	{
		auto mux = networkInterface.TryGetMuxAbove();
		if (mux)
		{
			if (mux->GetObjectId().ObjectId == this->GetObjectId().ObjectId)
				NetSetup2::Exception::throw_MemberAlreadyInThisMux(HNETSETUP(m_transaction), networkInterface.GetObjectId().ObjectId, this->GetObjectId().ObjectId);
			else
				NetSetup2::Exception::throw_MemberAlreadyInAnotherMux(HNETSETUP(m_transaction), networkInterface.GetObjectId().ObjectId, this->GetObjectId().ObjectId, mux->GetObjectId().ObjectId);
		}

		networkInterface.set_MemberOfMuxId(GetObjectUnique());
	}

	void
		Mux::RemoveMember(
			_In_ NetworkInterface &networkInterface) const
	{
		auto above = networkInterface.TryGetMuxAbove();
		if (!NETSETUP_VERIFY(above) || !NETSETUP_VERIFY(*above == *this))
			NETSETUP_THROW(HRESULT_FROM_WIN32(ERROR_NOT_OWNER));

		networkInterface.clear_MemberOfMuxId();
	}

	std::vector<std::unique_ptr<NetworkInterface>>
		Mux::EnumerateMembers() const
	{
		auto id = GetObjectUnique();

		NETSETUPPROP_FILTER_EXPRESSION filters[] =
		{
			{ NETSETUPPROP_OPERATOR_EQUALS,{
				{ NETSETUPPKEY_MUX_MemberOfMuxId, },
				NETSETUPPROP_TYPE_GUID,
				sizeof(id),
				reinterpret_cast<BYTE*>(&id),
			} },
		};

		return m_transaction.GetAllNetworkInterfaces(filters);
	}

	NetworkInterfaceTemplate
		Mux::CreateVirtualInterface() const
	{
		NetworkInterfaceTemplate nic;
		nic.set_VirtualInterfaceOfMuxId(GetObjectUnique());
		return nic;
	}

	std::vector<std::unique_ptr<NetworkInterface>>
		Mux::EnumerateVirtualInterfaces() const
	{
		auto id = GetObjectUnique();

		NETSETUPPROP_FILTER_EXPRESSION filters[] =
		{
			{ NETSETUPPROP_OPERATOR_EQUALS,{
				{ NETSETUPPKEY_MUX_VirtualInterfaceOfMuxId, },
				NETSETUPPROP_TYPE_GUID,
				sizeof(id),
				reinterpret_cast<BYTE*>(&id),
			} },
		};

		return m_transaction.GetAllNetworkInterfaces(filters);
	}

	details::Optional<Mux>
		NetworkInterface::TryGetMuxAbove() const
	{
		auto muxId = tryget_MemberOfMuxId();
		if (!muxId)
			return{};

		return m_transaction.GetMuxById(*muxId);
	}

	details::Optional<Mux>
		NetworkInterface::TryGetMuxBelow() const
	{
		auto muxId = tryget_VirtualInterfaceOfMuxId();
		if (!muxId)
			return{};

		return m_transaction.GetMuxById(*muxId);
	}

	namespace details
	{

		TransactionBase::TransactionBase(
			_In_ HNETSETUP handle) :
			m_netsetup(handle)
		{

		}

		TransactionBase::TransactionBase(
			_In_ TransactionBase && rhs) :
			m_netsetup(rhs.m_netsetup)
		{
			rhs.m_netsetup = nullptr;
		}

		void
			TransactionBase::Validate()
		{
			HRESULT hr = NetSetupValidateTransaction(GetHandle());
			IFFAILED_THROW(hr);
		}

		void
			TransactionBase::Commit()
		{
			HRESULT hr = NetSetupCommit(GetHandle());
			IFFAILED_THROW(hr);
		}

		void
			TransactionBase::Rollback()
		{
			NetSetupRollback(GetHandle());
		}

		std::vector<std::unique_ptr<BindingPath>>
			TransactionBase::GetAllBindingPathsBetween(ActiveObject const & lower, ActiveObject const & higher)
		{
			auto idLow = lower.GetObjectId();
			auto idHigh = higher.GetObjectId();

			NETSETUPPROP_FILTER_EXPRESSION filters[] =
			{
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Binding_Lowest, },
					NETSETUPPROP_TYPE_OBJECT_ID,
					sizeof(idLow),
					(BYTE*)&idLow,
				} },
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Binding_Highest },
					NETSETUPPROP_TYPE_OBJECT_ID,
					sizeof(idHigh),
					(BYTE*)&idHigh,
				} },
			};

			return GetAllBindingPaths(filters);
		}

		bool
			TransactionBase::IsBindable(ActiveObject const &lower, ActiveObject const &higher)
		{
			return !GetAllBindingPathsBetween(lower, higher).empty();
		}

		std::vector<std::unique_ptr<BindingPath>>
			TransactionBase::GetAllEnabledBindingPathsBetween(ActiveObject const & lower, ActiveObject const & higher)
		{
			auto idLow = lower.GetObjectId();
			auto idHigh = higher.GetObjectId();
			BOOLEAN enabled = TRUE;

			NETSETUPPROP_FILTER_EXPRESSION filters[] =
			{
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Binding_Lowest, },
					NETSETUPPROP_TYPE_OBJECT_ID,
					sizeof(idLow),
					(BYTE*)&idLow,
				} },
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Binding_Highest },
					NETSETUPPROP_TYPE_OBJECT_ID,
					sizeof(idHigh),
					(BYTE*)&idHigh,
				} },
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Binding_IsEnabled },
					NETSETUPPROP_TYPE_BOOLEAN,
					sizeof(BOOLEAN),
					(BYTE*)&enabled,
				} },
			};

			return GetAllBindingPaths(filters);
		}

		bool
			TransactionBase::IsBindingEnabled(ActiveObject const &lower, ActiveObject const &higher)
		{
			return !GetAllEnabledBindingPathsBetween(lower, higher).empty();
		}

		static
			details::AutoFreeObjects
			GetObjectWithFilterInner(
				_In_ TransactionBase const &transaction,
				_In_ NETSETUP_OBJECT_TYPE type,
				_In_opt_ GUID const *id,
				_In_ ULONG numFilterExpressions,
				_In_reads_opt_(numFilterExpressions) NETSETUPPROP_FILTER_EXPRESSION const *filter)
		{
			ULONG numObjects;
			NETSETUP_OBJECT const *objects = nullptr;
			HRESULT hr = NetSetupGetObjects(
				transaction,
				type,
				id,
				0,
				0, nullptr,
				numFilterExpressions, filter,
				&numObjects, &objects);
			IFFAILED_THROW(hr);
			NETSETUP_ASSERT(numObjects < 2);

			return{ numObjects, objects };
		}


		template<typename T>
		T
			GetObjectWithFilter(
				_In_ TransactionBase const &transaction,
				_In_ NETSETUP_OBJECT_TYPE type,
				_In_opt_ GUID const *id,
				_In_ ULONG numFilterExpressions,
				_In_reads_opt_(numFilterExpressions) NETSETUPPROP_FILTER_EXPRESSION const *filter)
		{
			auto const &afo = GetObjectWithFilterInner(transaction, type, id, numFilterExpressions, filter);

			if (afo.m_num == 0)
				NETSETUP_THROW(HRESULT_FROM_WIN32(ERROR_NOT_FOUND));

			return{ const_cast<TransactionBase &>(transaction), type, afo.m_obj[0].ObjectId };
		}

		template<typename T>
		details::Optional<T>
			TryGetObjectWithFilter(
				_In_ TransactionBase const &transaction,
				_In_ NETSETUP_OBJECT_TYPE type,
				_In_opt_ GUID const *id,
				_In_ ULONG numFilterExpressions,
				_In_reads_opt_(numFilterExpressions) NETSETUPPROP_FILTER_EXPRESSION const *filter)
		{
			auto const &afo = GetObjectWithFilterInner(transaction, type, id, numFilterExpressions, filter);

			if (afo.m_num == 0)
				return{};

			return{ T{ const_cast<TransactionBase &>(transaction), type, afo.m_obj[0].ObjectId } };
		}

		template<typename T>
		T
			GetObjectByIdInner(
				_In_ TransactionBase const &transaction,
				_In_ NETSETUP_OBJECT_TYPE type,
				_In_ GUID const &id)
		{
			return GetObjectWithFilter<T>(transaction, type, &id, 0, nullptr);
		}

		template<typename T>
		details::Optional<T>
			TryGetObjectByIdInner(
				_In_ TransactionBase const &transaction,
				_In_ NETSETUP_OBJECT_TYPE type,
				_In_ GUID const &id)
		{
			return TryGetObjectWithFilter<T>(transaction, type, &id, 0, nullptr);
		}

		template<typename T>
		T
			GetObjectByIdentifierInner(
				_In_ TransactionBase const &transaction,
				_In_ NETSETUP_OBJECT_TYPE type,
				_In_ std::wstring const &identifier)
		{
			NETSETUPPROP_FILTER_EXPRESSION filter =
			{ NETSETUPPROP_OPERATOR_EQUALS_IGNORE_CASE,{
				{ NETSETUPPKEY_Driver_Identifier, },
				NETSETUPPROP_TYPE_STRING,
				static_cast<ULONG>((1 + identifier.size()) * sizeof(WCHAR)),
				const_cast<BYTE*>(reinterpret_cast<BYTE const*>(identifier.c_str())),
			} };

			return GetObjectWithFilter<T>(transaction, type, nullptr, 1, &filter);
		}

		template<typename T>
		details::Optional<T>
			TryGetObjectByIdentifierInner(
				_In_ TransactionBase const &transaction,
				_In_ NETSETUP_OBJECT_TYPE type,
				_In_ std::wstring const &identifier)
		{
			NETSETUPPROP_FILTER_EXPRESSION filter =
			{ NETSETUPPROP_OPERATOR_EQUALS_IGNORE_CASE,{
				{ NETSETUPPKEY_Driver_Identifier, },
				NETSETUPPROP_TYPE_STRING,
				static_cast<ULONG>((1 + identifier.size()) * sizeof(WCHAR)),
				const_cast<BYTE*>(reinterpret_cast<BYTE const*>(identifier.c_str())),
			} };

			return TryGetObjectWithFilter<T>(transaction, type, nullptr, 1, &filter);
		}

		NetworkInterface
			TransactionBase::GetNetworkInterfaceById(
				_In_ GUID const &id)
		{
			return GetObjectByIdInner<NetworkInterface>(*this, NetSetupObjectTypeInterface, id);
		}

		details::Optional<NetworkInterface>
			TransactionBase::TryGetNetworkInterfaceById(
				_In_ GUID const &id)
		{
			return TryGetObjectByIdInner<NetworkInterface>(*this, NetSetupObjectTypeInterface, id);
		}

		NetworkInterface
			TransactionBase::GetNetworkInterfaceByIfAlias(_In_ std::wstring const &ifAlias)
		{
			NETSETUPPROP_FILTER_EXPRESSION filter =
			{ NETSETUPPROP_OPERATOR_EQUALS_IGNORE_CASE,{
				{ NETSETUPPKEY_Interface_IfAlias, },
				NETSETUPPROP_TYPE_STRING,
				static_cast<ULONG>((1 + ifAlias.size()) * sizeof(WCHAR)),
				const_cast<BYTE*>(reinterpret_cast<BYTE const*>(ifAlias.c_str())),
			} };

			return GetObjectWithFilter<NetworkInterface>(*this, NetSetupObjectTypeInterface, nullptr, 1, &filter);
		}

		details::Optional<NetworkInterface>
			TransactionBase::TryGetNetworkInterfaceByIfAlias(_In_ std::wstring const &ifAlias)
		{
			NETSETUPPROP_FILTER_EXPRESSION filter =
			{ NETSETUPPROP_OPERATOR_EQUALS_IGNORE_CASE,{
				{ NETSETUPPKEY_Interface_IfAlias, },
				NETSETUPPROP_TYPE_STRING,
				static_cast<ULONG>((1 + ifAlias.size()) * sizeof(WCHAR)),
				const_cast<BYTE*>(reinterpret_cast<BYTE const*>(ifAlias.c_str())),
			} };

			return TryGetObjectWithFilter<NetworkInterface>(*this, NetSetupObjectTypeInterface, nullptr, 1, &filter);
		}

		ProtocolDriver
			TransactionBase::GetProtocolDriverById(_In_ GUID const &id)
		{
			return GetObjectByIdInner<ProtocolDriver>(*this, NetSetupObjectTypeProtocolDriver, id);
		}

		details::Optional<ProtocolDriver>
			TransactionBase::TryGetProtocolDriverById(_In_ GUID const &id)
		{
			return TryGetObjectByIdInner<ProtocolDriver>(*this, NetSetupObjectTypeProtocolDriver, id);
		}

		ProtocolDriver
			TransactionBase::GetProtocolDriverByName(_In_ std::wstring const &name)
		{
			return GetObjectByIdentifierInner<ProtocolDriver>(*this, NetSetupObjectTypeProtocolDriver, name);
		}

		details::Optional<ProtocolDriver>
			TransactionBase::TryGetProtocolDriverByName(_In_ std::wstring const &name)
		{
			return TryGetObjectByIdentifierInner<ProtocolDriver>(*this, NetSetupObjectTypeProtocolDriver, name);
		}

		FilterDriver
			TransactionBase::GetFilterDriverById(_In_ GUID const &id)
		{
			return GetObjectByIdInner<FilterDriver>(*this, NetSetupObjectTypeFilterDriver, id);
		}

		details::Optional<FilterDriver>
			TransactionBase::TryGetFilterDriverById(_In_ GUID const &id)
		{
			return TryGetObjectByIdInner<FilterDriver>(*this, NetSetupObjectTypeFilterDriver, id);
		}

		FilterDriver
			TransactionBase::GetFilterDriverByName(_In_ std::wstring const &name)
		{
			return GetObjectByIdentifierInner<FilterDriver>(*this, NetSetupObjectTypeFilterDriver, name);
		}

		details::Optional<FilterDriver>
			TransactionBase::TryGetFilterDriverByName(_In_ std::wstring const &name)
		{
			return TryGetObjectByIdentifierInner<FilterDriver>(*this, NetSetupObjectTypeFilterDriver, name);
		}

		ClientDriver
			TransactionBase::GetClientDriverById(_In_ GUID const &id)
		{
			return GetObjectByIdInner<ClientDriver>(*this, NetSetupObjectTypeClientDriver, id);
		}

		details::Optional<ClientDriver>
			TransactionBase::TryGetClientDriverById(_In_ GUID const &id)
		{
			return TryGetObjectByIdInner<ClientDriver>(*this, NetSetupObjectTypeClientDriver, id);
		}

		ClientDriver
			TransactionBase::GetClientDriverByName(_In_ std::wstring const &name)
		{
			return GetObjectByIdentifierInner<ClientDriver>(*this, NetSetupObjectTypeClientDriver, name);
		}

		details::Optional<ClientDriver>
			TransactionBase::TryGetClientDriverByName(_In_ std::wstring const &name)
		{
			return TryGetObjectByIdentifierInner<ClientDriver>(*this, NetSetupObjectTypeClientDriver, name);
		}

		ServiceDriver
			TransactionBase::GetServiceDriverById(_In_ GUID const &id)
		{
			return GetObjectByIdInner<ServiceDriver>(*this, NetSetupObjectTypeServiceDriver, id);
		}

		details::Optional<ServiceDriver>
			TransactionBase::TryGetServiceDriverById(_In_ GUID const &id)
		{
			return TryGetObjectByIdInner<ServiceDriver>(*this, NetSetupObjectTypeServiceDriver, id);
		}

		ServiceDriver
			TransactionBase::GetServiceDriverByName(_In_ std::wstring const &name)
		{
			return GetObjectByIdentifierInner<ServiceDriver>(*this, NetSetupObjectTypeServiceDriver, name);
		}

		details::Optional<ServiceDriver>
			TransactionBase::TryGetServiceDriverByName(_In_ std::wstring const &name)
		{
			return TryGetObjectByIdentifierInner<ServiceDriver>(*this, NetSetupObjectTypeServiceDriver, name);
		}

		ReflectedProperty
			TransactionBase::ReflectOnProperty(_In_ NETSETUPPROPKEY const &key)
		{
			return ReflectedProperty(*this, NETSETUP_REFLECTED_PROPERTY_FROM_KEY(&key), key.fmtid);

			//return GetObjectByIdInner<ReflectedProperty>(*this, , key.fmtid);
		}

		BindingPath
			TransactionBase::GetBindingPathById(_In_ GUID const &id)
		{
			return GetObjectByIdInner<BindingPath>(*this, NetSetupObjectTypeBindingPath, id);
		}

		details::Optional<BindingPath>
			TransactionBase::TryGetBindingPathById(_In_ GUID const &id)
		{
			return TryGetObjectByIdInner<BindingPath>(*this, NetSetupObjectTypeBindingPath, id);
		}

		BindingPath
			TransactionBase::GetBindingPathBetween(
				_In_ ActiveObject const &lower,
				_In_ ActiveObject const &higher)
		{
			auto path = TryGetBindingPathBetween(lower, higher);
			if (!path)
				NETSETUP_THROW(HRESULT_FROM_WIN32(ERROR_NOT_FOUND));
			return std::move(*path);
		}

		details::Optional<BindingPath>
			TransactionBase::TryGetBindingPathBetween(
				_In_ ActiveObject const &lower,
				_In_ ActiveObject const &higher)
		{
			auto idLow = lower.GetObjectId();
			auto idHigh = higher.GetObjectId();

			NETSETUPPROP_FILTER_EXPRESSION filters[] =
			{
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Binding_Lowest, },
					NETSETUPPROP_TYPE_OBJECT_ID,
					sizeof(idLow),
					(BYTE*)&idLow,
				} },
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Binding_Highest },
					NETSETUPPROP_TYPE_OBJECT_ID,
					sizeof(idHigh),
					(BYTE*)&idHigh,
				} },
			};

			return TryGetObjectWithFilter<BindingPath>(*this, NetSetupObjectTypeBindingPath, nullptr, RTL_NUMBER_OF(filters), filters);
		}

		std::vector<std::unique_ptr<BindingPath>>
			TransactionBase::GetAllBindingPathsAbove(ActiveObject const & lower)
		{
			auto idLow = lower.GetObjectId();

			NETSETUPPROP_FILTER_EXPRESSION filter[] =
			{
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Binding_Lowest, },
					NETSETUPPROP_TYPE_OBJECT_ID,
					sizeof(idLow),
					(BYTE*)&idLow,
				} },
			};

			return GetAllBindingPaths(filter);
		}

		std::vector<std::unique_ptr<BindingPath>>
			TransactionBase::GetAllBindingPathsBelow(ActiveObject const & upper)
		{
			auto idHigh = upper.GetObjectId();

			NETSETUPPROP_FILTER_EXPRESSION filter[] =
			{
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Binding_Highest },
					NETSETUPPROP_TYPE_OBJECT_ID,
					sizeof(idHigh),
					(BYTE*)&idHigh,
				} },
			};

			return GetAllBindingPaths(filter);
		}

		details::Optional<BindingPath>
			TransactionBase::TryFindBindingPath(std::vector<NETSETUP_OBJECT_ID> const & path)
		{
			NETSETUPPROP_FILTER_EXPRESSION filter[] =
			{
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Binding_Path },
					NETSETUPPROP_TYPE_OBJECT_ID_ARRAY,
					static_cast<ULONG>(path.size() * sizeof(NETSETUP_OBJECT_ID)),
					(BYTE*)&path[0]
				} },
			};

			auto paths = GetAllBindingPaths(filter);
			if (paths.size() == 0)
			{
				return{};
			}

			WIN_VERIFY(paths.size() == 1);

			return *paths[0];
		}

		Mux
			TransactionBase::GetMuxById(_In_ GUID const &id)
		{
			return GetObjectByIdInner<Mux>(*this, NetSetupObjectTypeMux, id);
		}

		details::Optional<Mux>
			TransactionBase::TryGetMuxById(_In_ GUID const &id)
		{
			return TryGetObjectByIdInner<Mux>(*this, NetSetupObjectTypeMux, id);
		}

		BindRule
			TransactionBase::GetBindRuleById(_In_ GUID const &id)
		{
			return GetObjectByIdInner<BindRule>(*this, NetSetupObjectTypeBindRule, id);
		}

		details::Optional<BindRule>
			TransactionBase::TryGetBindRuleById(_In_ GUID const &id)
		{
			return TryGetObjectByIdInner<BindRule>(*this, NetSetupObjectTypeBindRule, id);
		}

		ReflectedProperty
			TransactionBase::ReflectOnProperty(_In_ std::wstring const &name)
		{
			auto results = TryReflectOnProperty(name);
			if (!results)
				NETSETUP_THROW(HRESULT_FROM_WIN32(ERROR_NOT_FOUND));
			return std::move(*results);
		}

		ActiveObject
			TransactionBase::GetObjectById(_In_ NETSETUP_OBJECT_ID const &id)
		{
			return GetObjectByIdInner<ActiveObject>(*this, id.ObjectType, id.ObjectId);
		}

		ActiveObject
			TransactionBase::GetObjectById(_In_ NETSETUP_OBJECT_TYPE type, _In_ GUID const &id)
		{
			return GetObjectByIdInner<ActiveObject>(*this, type, id);
		}

		details::Optional<ActiveObject>
			TransactionBase::TryGetObjectById(_In_ NETSETUP_OBJECT_ID const &id)
		{
			return TryGetObjectByIdInner<ActiveObject>(*this, id.ObjectType, id.ObjectId);
		}

		details::Optional<ActiveObject>
			TransactionBase::TryGetObjectById(_In_ NETSETUP_OBJECT_TYPE type, _In_ GUID const &id)
		{
			return TryGetObjectByIdInner<ActiveObject>(*this, type, id);
		}

		details::Optional<ActiveObject>
			TransactionBase::TryMatchObjectById(
				_In_ NETSETUP_OBJECT_ID const & id,
				_In_ NETSETUPPROP_FILTER_EXPRESSION const * begin,
				_In_ NETSETUPPROP_FILTER_EXPRESSION const * end)
		{
			return TryGetObjectWithFilter<ActiveObject>(*this, id.ObjectType, &id.ObjectId, static_cast<ULONG>(end - begin), begin);
		}

		details::Optional<ReflectedProperty>
			TransactionBase::TryReflectOnProperty(_In_ std::wstring const &name)
		{
			NETSETUPPROP_FILTER_EXPRESSION filters[] =
			{
				{ NETSETUPPROP_OPERATOR_EQUALS,{
					{ NETSETUPPKEY_Reflection_PropertyName, },
					NETSETUPPROP_TYPE_STRING,
					static_cast<ULONG>((name.size() + 1) * sizeof(WCHAR)),
					const_cast<BYTE*>(reinterpret_cast<BYTE const*>(name.c_str())),
				} },
			};

			auto results = GetAllObjectsInner<ReflectedProperty>(NetSetupObjectTypeReflectedProperty, nullptr, 0, filters, RTL_NUMBER_OF(filters));
			NETSETUP_ASSERT(results.size() < 2);

			if (results.size() == 0)
				return{};

			return{ *results[0] };
		}

	}

	// static 
	bool
		Transaction::IsOnlineApiAvailable()
	{
		HNETSETUP netsetup = nullptr;
		HRESULT hr = NetSetupInitialize(nullptr, &netsetup);
		if (hr == E_NOTIMPL)
			return false;

		IFFAILED_THROW(hr);

		NetSetupClose(netsetup);
		return true;
	}


	Transaction::Transaction(
		_In_opt_ NETSETUP_ENVIRONMENT *environment)
	{
		HRESULT hr = NetSetupInitialize(environment, &m_netsetup);
		IFFAILED_THROW(hr);
	}

	Transaction::Transaction(
		_In_ HNETSETUP &&handle) :
		TransactionBase(handle)
	{
		handle = nullptr;
	}

	Transaction::~Transaction()
	{
		if (m_netsetup)
		{
			NetSetupClose(m_netsetup);
			m_netsetup = nullptr;
		}
	}

	namespace Aggregate
	{
		namespace details
		{

			size_t
				GetByteCount_STRING_LIST_helper(std::vector<std::wstring> const &value)
			{
				size_t cch = 1;
				for (auto &s : value)
				{
					NETSETUP_ASSERT(!value.empty());
					cch += s.size() + 1;
				}
				return cch * sizeof(WCHAR);
			}

			std::vector<BYTE>
				GetFlatStringArray(std::vector<std::wstring> const &value)
			{
				auto cb = GetByteCount_STRING_LIST_helper(value);
				std::vector<BYTE> result(cb);
				auto p = reinterpret_cast<PWSTR>(result.data());

				for (auto &s : value)
				{
					auto c = s.size() + 1;
					memcpy(p, s.c_str(), c * sizeof(WCHAR));
					p += c;
				}
				*p = L'\0';

				return result;
			}

		}
	}

	namespace Aggregate
	{
		namespace details
		{

			Property
				GetTransactionProperty(
					_In_ HNETSETUP transaction,
					_In_ NETSETUPPROPKEY const &key)
			{
				NETSETUPPROPCOMPKEY comp = { key };
				NETSETUPPROPERTY const *result = nullptr;
				ULONG numReturned;
				HRESULT hr = NetSetupGetObjectProperties(
					transaction,
					NetSetupObjectTypeTransaction,
					NULL_GUID,
					0,
					1, &comp,
					&numReturned, &result);
				IFFAILED_THROW(hr);

				::NetSetup2::details::AutoFreeProperties afp(numReturned, result);

				if (!NETSETUP_VERIFY(numReturned == 1))
				{
					NETSETUP_THROW_EX(E_UNEXPECTED, STORE,
						"NetSetupGetObjectProperties returned fewer properties than requested");
				}

				return Property{ result[0] };
			}


		}
	}

}

#pragma warning(pop)

