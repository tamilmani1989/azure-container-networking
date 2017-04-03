// ***********************************************************************
// <copyright file="ovPluginException.cpp" company="Microsoft">
//     Copyright (c) Microsoft Corporation. All rights reserved.
// </copyright>
// <summary></summary>
// ***********************************************************************
#include "ovPluginException.h"
#include <TraceLoggingProvider.h>
#include <Evntrace.h>

// {D0E4BC17-34C7-43fc-9A72-D89A59D6979A}
TRACELOGGING_DEFINE_PROVIDER(g_ovPluginLoggingProvider, "Microsoft.Azure.Networking.OverlayPlugin", (0xd0e4bc17, 0x34c7, 0x43fc, 0x9a, 0x72, 0xd8, 0x9a, 0x59, 0xd6, 0x97, 0x9a));

using namespace Microsoft::Azure::Networking::OverlayPlugin;

////////////////////////////////////////////////////////////////////////////////
// HostException::Throw
////////////////////////////////////////////////////////////////////////////////
_Analysis_noreturn_
__declspec(noreturn)
void __cdecl PluginException::Throw(_In_ HRESULT err, _In_z_ _Printf_format_string_ LPCWSTR format, ...)
{
    va_list list;
    va_start(list, format);

    std::wstring str;

    try
    {
#pragma warning(suppress: 26018) /* no buffer overflow */
        str = FormatV(format, list);
    }
    catch (...)
    {
        va_end(list);
        throw;
    }

    va_end(list);

    throw PluginException(err, std::move(str));
}
////////////////////////////////////////////////////////////////////////////////
// PluginException::FormatV
////////////////////////////////////////////////////////////////////////////////
std::wstring __cdecl PluginException::FormatV(_In_z_ _Printf_format_string_ LPCWSTR format, _In_ va_list list)
{
    std::wstring ret;

#pragma warning(suppress: 26018) /* format in StringUtils::Format has the same annotation as in _vscwprintf, no buffer overflow */
    int len = _vscwprintf(format, list);

    if (len < 0)
    {
        throw PluginException(E_INVALIDARG, L"Unable to format string");
    }

    ret.resize(static_cast<size_t>(len + 1));

    len = _vsnwprintf_s(const_cast<LPWSTR>(ret.data()), ret.size(), ret.size(), format, list);

    if (len < 0)
    {
        throw PluginException(E_FAIL, L"Unable to format string");
    }

    ret.resize(static_cast<size_t>(len));

    return ret;
}
////////////////////////////////////////////////////////////////////////////////
// HostException::ReportException
////////////////////////////////////////////////////////////////////////////////
HRESULT PluginException::LogException() throw()
{
    HRESULT hr;

    try
    {
        throw;
    }
    catch (const PluginException& hostEx)
    {
        hr = hostEx.GetError();

        TraceLoggingWrite(g_ovPluginLoggingProvider,
            "Error",
            TraceLoggingLevel(TRACE_LEVEL_ERROR),
            TraceLoggingValue(hostEx.GetMessage().c_str(), "message"),
            TraceLoggingHResult(hr, "error"));
    }
    catch (const std::bad_alloc&)
    {
        hr = E_OUTOFMEMORY;

        TraceLoggingWrite(g_ovPluginLoggingProvider,
            "Error",
            TraceLoggingLevel(TRACE_LEVEL_CRITICAL),
            TraceLoggingValue("Out of memory", "message"),
            TraceLoggingHResult(hr, "error"));
    }
    catch (const std::exception& stdEx)
    {
        hr = E_FAIL;

        TraceLoggingWrite(g_ovPluginLoggingProvider,
            "Error",
            TraceLoggingLevel(TRACE_LEVEL_ERROR),
            TraceLoggingValue(stdEx.what(), "message"),
            TraceLoggingHResult(hr, "error"));
    }
    catch (...)
    {
        hr = E_FAIL;

        TraceLoggingWrite(g_ovPluginLoggingProvider, 
            "UnexpectedException", 
            TraceLoggingLevel(TRACE_LEVEL_CRITICAL));
    }

    return hr;
}