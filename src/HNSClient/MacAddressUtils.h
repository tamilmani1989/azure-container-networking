/*++

Copyright (c) Microsoft Corporation

Module Name:

MacAddressUtils.h

Abstract:

Contains a set of helper methods for manipulating MAC addresses.

Author:

Attilio Mainetti    (attiliom)  01-September-2009

Last Updated:

Sushant Sharma (susharm) 06-April-2017

--*/

#pragma once

#include <windows.h>
#include <string>
#include "marshal/inc/MarshalBase.h"
#include <wil/result.h>

// Disable warning that array initialization works in VC++ now.
// For some reason, this doesn't work if pushed/popped, but it's a
// useless, annoying warning anyway.
#pragma warning(disable: 4351)

namespace MacAddressUtils
{

	struct MacAddress
	{
		BYTE Value[6]{};
	};

	inline
		bool
		operator==(
			const MacAddress& Left,
			const MacAddress& Right
			)
		/*++

		Routine Description:

		Checks if two MAC addresses are the same.

		Arguments:

		Left - The left one.

		Right - The right one.

		Return Value:

		If they're the same.

		--*/
	{
		return Left.Value[0] == Right.Value[0] &&
			Left.Value[1] == Right.Value[1] &&
			Left.Value[2] == Right.Value[2] &&
			Left.Value[3] == Right.Value[3] &&
			Left.Value[4] == Right.Value[4] &&
			Left.Value[5] == Right.Value[5];
	}

	namespace Details
	{
		/// Converts an ASCII value known to be a hex digit to an integer value.
		inline
			UINT8
			ToNibble(
				_In_ char HexDigit
			)
		{
			// Clearing bit 0x20 will turn a-f to A-F.
			return (HexDigit >= '0' && HexDigit <= '9')
				? (HexDigit - '0')
				: ((HexDigit & ~0x20) - 'A' + 10);
		}

		enum
		{
			UnseparatedMacAddressBufferSize = _countof(L"xxxxxxxxxxxx"),
			SeparatedMacAddressBufferSize = _countof(L"xx-xx-xx-xx-xx-xx"),
		};

		/// Writes a MAC address to a string without separators. The string buffer
		/// must be large enough for UnseparatedMacAddressBufferSize characters.
		inline
			size_t
			WriteUnseparatedMacAddressString(
				_In_reads_(6) const BYTE* MacAddress,
				_Out_writes_(UnseparatedMacAddressBufferSize) WCHAR* String
			)
		{
			swprintf_s(String, UnseparatedMacAddressBufferSize,
				L"%02X%02X%02X%02X%02X%02X",
				MacAddress[0],
				MacAddress[1],
				MacAddress[2],
				MacAddress[3],
				MacAddress[4],
				MacAddress[5]);

			return UnseparatedMacAddressBufferSize - 1;
		}

		/// Writes a MAC address to a string with separators. The string buffer
		/// must be large enough for SeparatedMacAddressBufferSize characters.
		inline
			size_t
			WriteSeparatedMacAddressString(
				_In_reads_(6) const BYTE* MacAddress,
				_Out_writes_(SeparatedMacAddressBufferSize) WCHAR* String,
				_In_ WCHAR Separator
			)
		{
			swprintf_s(String, SeparatedMacAddressBufferSize,
				L"%02X%c%02X%c%02X%c%02X%c%02X%c%02X",
				MacAddress[0],
				Separator,
				MacAddress[1],
				Separator,
				MacAddress[2],
				Separator,
				MacAddress[3],
				Separator,
				MacAddress[4],
				Separator,
				MacAddress[5]);

			return SeparatedMacAddressBufferSize - 1;
		}

		/// Parses a MAC address string. If Separator is non-zero, then it is
		/// the separator character to look for; otherwise the string must be
		/// unseparated. If the separator character is 0xFFFF, then any character
		/// may be used as a separator character.
		inline
			bool
			ParseMacAddressString(
				_In_ PCWSTR String,
				_Out_writes_all_(6) PBYTE Value,
				_In_ WCHAR Separator
			)
		{
			MacAddress tempMacAddress;
			size_t curPos = 0;
			int octet;
			for (octet = 0; octet < 6; ++octet)
			{
				if (octet != 0 && Separator != 0)
				{
					if (String[curPos] != Separator && Separator != 0xFFFF)
					{
						break;
					}

					++curPos;
				}

				if (!iswxdigit(String[curPos]) || !iswxdigit(String[curPos + 1]))
				{
					break;
				}

				tempMacAddress.Value[octet] = ToNibble((char)String[curPos]) * 16u +
					ToNibble((char)String[curPos + 1]);

				curPos += 2;
			}

			if (octet != 6 || String[curPos] != '\0')
			{
				return false;
			}

			memcpy(Value, tempMacAddress.Value, sizeof(tempMacAddress.Value));
			return true;
		}
	}

	/// Converts a MAC address from binary format to string format in the form
	/// xxxxxxxxxxxx.
	inline
		VOID
		MacAddressToMacAddressString(
			_In_reads_(6) const BYTE*          MacAddress,
			_Out_               std::wstring&  FormattedMacAddress
		)
	{
		FormattedMacAddress.resize(Details::UnseparatedMacAddressBufferSize);
		FormattedMacAddress.resize(Details::WriteUnseparatedMacAddressString(MacAddress, &FormattedMacAddress[0]));
	}

	/// Converts a MAC address string format to binary format.
	/// If SeparatedFormat is TRUE, the format is xx-xx-xx-xx-xx. Otherwise, it
	/// is xxxxxxxxxxxx.
	inline
		VOID
		MacAddressStringToMacAddress(
			_In_                LPCWSTR MacAddressString,
			_Out_writes_all_(6) PBYTE   MacAddress,
			_In_                BOOL    SeparatedFormat = FALSE
		)
	{
		THROW_HR_IF_FALSE(E_INVALIDARG,
			Details::ParseMacAddressString(MacAddressString, MacAddress, SeparatedFormat ? '-' : 0));
	}

	/// Converts a MAC address formatted string format to simple MAC address.
	/// E.g. XX-XX-XX-XX-XX-XX is converted to XXXXXXXXXXXX
	inline
		VOID
		RemoveSeparatorFromMacAddressString(
			_In_                         LPCWSTR MacAddressSeparatedString,
			_In_                         UINT32  MacAddressSize,
			_Out_writes_(MacAddressSize) LPWSTR  MacAddressValue
		)
	{
		if (MacAddressSize != Details::UnseparatedMacAddressBufferSize)
		{
			THROW_HR(E_INVALIDARG);
		}

		// Parse the passed-in MAC address, ignoring the separator characters entirely.
		MacAddress tempMacAddress;
		THROW_HR_IF_FALSE(E_INVALIDARG,
			Details::ParseMacAddressString(MacAddressSeparatedString, tempMacAddress.Value, 0xFFFF));

		Details::WriteUnseparatedMacAddressString(tempMacAddress.Value, MacAddressValue);
	}

	/// Converts a MAC address string format to include the separator. That is,
	/// converts in place xxxxxxxxxxxx to xx-xx-xx-xx-xx-xx, where - is whatever
	/// separator is provided by the caller.
	inline
		VOID
		AddSeparatorToMacAddressString(
			_Inout_ std::wstring&       FormattedMacAddress,
			_In_    WCHAR               Separator
		)
	{
		BYTE macAddress[6];
		THROW_HR_IF_FALSE(E_INVALIDARG,
			Details::ParseMacAddressString(FormattedMacAddress.c_str(), macAddress, 0));

		FormattedMacAddress.resize(Details::SeparatedMacAddressBufferSize);
		FormattedMacAddress.resize(Details::WriteSeparatedMacAddressString(macAddress, &FormattedMacAddress[0], Separator));
	}

	/// Converts a MAC address to a string in the form xx-xx-xx-xx-xx-xx.
	inline
		std::wstring
		ToSimpleValue(
			_In_ const MacAddress& Value,
			_In_ const Marshal::MarshalContext&
		)
	{
		std::wstring result(Details::SeparatedMacAddressBufferSize, L'\0');
		result.resize(Details::WriteSeparatedMacAddressString(Value.Value, &result[0], '-'));
		return result;
	}

	/// Converts a MAC address from a string in the form xx-xx-xx-xx-xx-xx.
	inline
		bool
		FromSimpleValue(
			_In_ const std::wstring& Value,
			_Out_ MacAddress* Result,
			_In_ const Marshal::UnmarshalContext& Context
		)
	{
		if (!Details::ParseMacAddressString(Value.c_str(), Result->Value, '-'))
		{
			Context.ReportError(Marshal::UnmarshalError::InvalidMacAddress);
			return false;
		}

		return true;
	}

}

