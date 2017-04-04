// ***********************************************************************
// <copyright file="ovPluginException.h" company="Microsoft">
//     Copyright (c) Microsoft Corporation. All rights reserved.
// </copyright>
// <summary></summary>
// ***********************************************************************
#pragma once

#ifndef __PC_EXCEPTION_H__
#define __PC_EXCEPTION_H__

#include <Windows.h>
#include <comdef.h>
#include <stdexcept>
#include <TraceLoggingProvider.h>
#include <Evntrace.h>
#include <wil\Com.h>
#include "hnsPluginApi.h"

// tracing
TRACELOGGING_DECLARE_PROVIDER(g_ovPluginLoggingProvider);

namespace Microsoft { namespace Azure { namespace Networking { namespace OverlayPlugin {

    typedef wil::unique_cotaskmem_string PCStringPtr;
    typedef wil::com_ptr<IHNSObjectList> HNSObjListPtr;
    typedef wil::com_ptr<IHNSObject> HNSObjPtr;
    typedef wil::com_ptr<IHNSProvider> HNSProviderPtr;

    #define TraceLoggingFuncLine                     TraceLoggingString( __FUNCTION__, "Function" ), TraceLoggingUInt32( __LINE__, "Line" )

    #define PC_THROW_IF_FAILED_LOG(exp) \
        do{ \
            HRESULT _Hr_ = exp; \
            if( FAILED( _Hr_ ) ) { PC_THROW_FAILED_LOG(_Hr_); } \
        } while(0)

    #define PC_THROW_FAILED_LOG(_Hr_) \
        TraceLoggingWrite( g_ovPluginLoggingProvider, \
                           "Exit function", \
                           TraceLoggingLevel( TRACE_LEVEL_ERROR ), \
                           TraceLoggingString( __FUNCTION__, "Function" ), TraceLoggingUInt32( __LINE__, "Line" ), \
                           TraceLoggingHResult(_Hr_, "error")); _com_error err(_Hr_); PluginException::Throw(_Hr_, L"%s", L"Unsupported COM Error")

    #define TRACE_ENTER() \
        TraceLoggingWrite( g_ovPluginLoggingProvider, \
                           "Enter function", \
                           TraceLoggingLevel( TRACE_LEVEL_VERBOSE ), \
                           TraceLoggingFuncLine )

    #define TRACE_EXIT() \
        TraceLoggingWrite( g_ovPluginLoggingProvider, \
                           "Exit function", \
                           TraceLoggingLevel( TRACE_LEVEL_VERBOSE ), \
                           TraceLoggingFuncLine, \
                           TraceLoggingHResult(S_OK, "success"));

    /// <summary>Provides base class for private cloud plugin exceptions.</summary>
    class PluginException : public std::runtime_error
    {
    public:
        /// <summary>Initializes a new instance.<summary>
        /// <param name='err'>HRESULT associated with error.</param>
        /// <param name='msg'>Message associated with error.</param>
        explicit PluginException(_In_ HRESULT err, _In_ const std::wstring& msg) :
            std::runtime_error(nullptr),
            m_hr(err),
            m_msg(msg)
        {
        }
        /// <summary>Initializes a new instance.<summary>
        /// <param name='err'>HRESULT associated with error.</param>
        /// <param name='msg'>Message associated with error.</param>
        PluginException(_In_ HRESULT err, _Inout_ std::wstring&& msg) :
            std::runtime_error(nullptr),
            m_hr(err),
            m_msg(std::move(msg))
        {
        }
        /// <summary>Throws a new instance initializes with last Win32 error and message.</summary>
        _Analysis_noreturn_
        __declspec(noreturn)
        static void ThrowLastError(_In_ const std::wstring& msg)
        {
            DWORD err = ::GetLastError();
            throw PluginException(HRESULT_FROM_WIN32(err), msg);
        }
        /// <summary>Formats message and throws exception.</summary>
        _Analysis_noreturn_
        __declspec(noreturn)
        static void __cdecl Throw(_In_ HRESULT err, _In_z_ _Printf_format_string_ LPCWSTR format, ...);
        /// <summary>Gets message.</summary>
        const std::wstring& GetMessage() const throw()
        {
            return m_msg;
        }
        /// <summary>Gets HRESULT.</summary>
        HRESULT GetError() const throw()
        {
            return m_hr;
        }
        /// <summary>Reports exception to trace log.</summary>
        /// <remarks>Function re-throws exception currently being handled. Call it only from catch block.</remarks>
        static HRESULT LogException() throw();
    private:
        static std::wstring __cdecl PluginException::FormatV(_In_z_ _Printf_format_string_ LPCWSTR format, _In_ va_list list);
        /// <summary>Error code.</summary>
        HRESULT m_hr;
        /// <summary>Message.</summary>
        std::wstring m_msg;
    };
}}}}

#endif /* __PC_EXCEPTION_H__ */
#pragma once
