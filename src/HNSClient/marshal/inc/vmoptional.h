/*++

Copyright (c) Microsoft Corporation

Module Name:

vmoptional.h

Abstract:

This file contains the optional class. It is modeled after
std::experimental::optional, and can be replaced once this class
has been standardized.

It behaves a little differently in that it does not avoid constructing
the underlying object to the default state when the optional value is
"null"/not engaged. This should not be a problem for our use of it.

See http://www.open-std.org/jtc1/sc22/wg21/docs/papers/2012/n3406.html.

Author:

John Starks (jostarks) 31-Jan-2015

--*/

#pragma once

#include <type_traits>
#include <windef.h>

namespace vmstd
{

	struct nullopt_t {};

	template <class T>
	struct optional
	{
	public:
		// construct an empty value
		optional() = default;

		// copy
		optional(const optional& Other) :
			m_Value(Other.m_Value), m_IsSet(Other.m_IsSet)
		{}

		// move, clearing the the rvalue
		optional(optional&& Other) :
			m_Value(std::move(Other.m_Value)), m_IsSet(Other.m_IsSet)
		{
			Other.m_IsSet = false;
		}

		optional& operator=(const optional& Other)
		{
			if (Other.m_IsSet)
			{
				m_Value = Other.m_Value;
				m_IsSet = true;
			}
			else
			{
				m_Value = {};
				m_IsSet = false;
			}

			return *this;
		}

		optional& operator=(optional&& Other)
		{
			if (Other.m_IsSet)
			{
				m_Value = std::move(Other.m_Value);
				m_IsSet = true;
			}
			else
			{
				m_Value = {};
				m_IsSet = false;
			}

			return *this;
		}

		~optional() = default;

		// initialize with a value
		optional(const T& Value) : m_Value(Value), m_IsSet(true)
		{
		}

		optional(T&& Value) : m_Value(std::move(Value)), m_IsSet(true)
		{
		}

		// assign a value
		optional& operator=(const T& Value)
		{
			m_Value = Value;
			m_IsSet = true;
			return *this;
		}

		optional& operator=(T&& Value)
		{
			m_Value = std::move(Value);
			m_IsSet = true;
			return *this;
		}

		// skip this generic forwarder for integer types to avoid spurious
		// truncation warnings
		template <class U, class = std::enable_if<!std::is_integral<T>::value && std::is_same<typename std::remove_reference<U>::type, T>::value>::type>
		optional& operator=(U&& Value)
		{
			m_Value = std::forward<U>(Value);
			m_IsSet = true;
			return *this;
		}

		// clear to the empty state
		optional& operator=(nullopt_t)
		{
			m_IsSet = false;
			m_Value = {};
			return *this;
		}

		// determine whether a value is present
		explicit operator bool() const
		{
			return m_IsSet;
		}

		// get a pointer to the value; it must be present
		const T* operator->() const
		{
			return &CheckedGet();
		}

		// get a pointer to the value; it must be present
		T* operator->()
		{
			return &CheckedGet();
		}

		// get a reference to the value; it must be present
		const T& operator*() const
		{
			return CheckedGet();
		}

		// get a reference to the value; it must be present
		T& operator*()
		{
			return CheckedGet();
		}

		// get a copy of the value if it's present, otherwise get the passed in value
		template <class U>
		T value_or(U&& Value) const
		{
			if (m_IsSet)
			{
				return m_Value;
			}
			else
			{
				return std::forward<U>(Value);
			}
		}

	private:
		T m_Value = {};
		bool m_IsSet = false;

		// get the value, terminating the process if it is not present
		T& CheckedGet()
		{
			if (!m_IsSet)
			{
				__fastfail(FAST_FAIL_INVALID_ARG);
			}

			return m_Value;
		}

		const T& CheckedGet() const
		{
			if (!m_IsSet)
			{
				__fastfail(FAST_FAIL_INVALID_ARG);
			}

			return m_Value;
		}
	};

	// set an optional to this value to clear the optional
	static constexpr nullopt_t nullopt{};

	// comparisons
	template <class T>
	bool
		operator==(
			_In_ const optional<T>& Left,
			_In_ const optional<T>& Right
			)
	{
		return (!Left && !Right) || (Left && Right && *Left == *Right);
	}


	template <class T>
	bool
		operator!=(
			_In_ const optional<T>& Left,
			_In_ const optional<T>& Right
			)
	{
		return !(Left == Right);
	}


	// comparisons to check for empty values
	template <class T>
	bool
		operator==(
			_In_ const optional<T>& Left,
			_In_ nullopt_t
			)
	{
		return !Left;
	}


	template <class T>
	bool
		operator==(
			_In_ nullopt_t,
			_In_ const optional<T>& Right
			)
	{
		return !Right;
	}


	template <class T>
	bool
		operator!=(
			_In_ const optional<T>& Left,
			_In_ nullopt_t
			)
	{
		return !!Left;
	}


	template <class T>
	bool
		operator!=(
			_In_ nullopt_t,
			_In_ const optional<T>& Right
			)
	{
		return !!Right;
	}

}

