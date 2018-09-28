// Windows Internal Libraries (wil)
//
// Usage Guidelines:
// https://microsoft.sharepoint.com/teams/osg_development/Shared%20Documents/Windows%20Error%20Handling%20Helpers.docx?web=1
//
// wil Usage Guidelines:
// https://microsoft.sharepoint.com/teams/osg_development/Shared%20Documents/Windows%20Internal%20Libraries%20for%20C++%20Usage%20Guide.docx?web=1
//
// wil Discussion Alias (wildisc):
// http://idwebelements/GroupManagement.aspx?Group=wildisc&Operation=join  (one-click join)
//
//! @file
//! Windows Error Handling Helpers: standard error handling mechanisms across return codes, fail fast, exceptions and logging

#pragma once

#include <Windows.h>
#include <strsafe.h>
#include <malloc.h>     // malloc / free used for internal buffer management
#include <intrin.h>     // provides the _ReturnAddress() intrinsic
#include "Common.h"
#if defined(WIL_ENABLE_EXCEPTIONS) && !defined(WIL_SUPPRESS_NEW)
#include <new>          // provides std::bad_alloc in the windows and public CRT headers
#endif

#ifndef STATUS_SUCCESS
#define STATUS_SUCCESS              ((NTSTATUS)0x00000000L)
#endif
#ifndef STATUS_UNSUCCESSFUL
#define STATUS_UNSUCCESSFUL         ((NTSTATUS)0xC0000001L)
#endif
#if !defined(_NTDEF_)
typedef _Return_type_success_(return >= 0) LONG NTSTATUS;
#endif

#pragma warning(push)
#pragma warning(disable:4714 6262)    // __forceinline not honored, stack size

//*****************************************************************************
// Behavioral setup (error handling macro configuration)
//*****************************************************************************
// Set any of the following macros to the values given below before including Result.h to
// control the error handling macro's trade-offs between diagnostics and performance

// RESULT_DIAGNOSTICS_LEVEL
// This define controls the level of diagnostic instrumentation that is built into the binary as a
// byproduct of using the macros.  The amount of diagnostic instrumentation that is supplied is
// a trade-off between diagnosibility of issues and code size and performance.  The modes are:
//      0   - No diagnostics, smallest & fastest (subject to tail-merge)
//      1   - No diagnostics, unique call sites for each macro (defeat's tail-merge)
//      2   - Line number
//      3   - Line number + source filename
//      4   - Line number + source filename + function name
//      5   - Line number + source filename + function name + code within the macro
// By default, mode 3 is used in free builds and mode 5 is used in checked builds.  Note that the
// _ReturnAddress() will always be available through all modes when possible.

// RESULT_INCLUDE_CALLER_RETURNADDRESS
// This controls whether or not the _ReturnAddress() of the function that includes the macro will
// be reported to telemetry.  Note that this is in addition to the _ReturnAddress() of the actual
// macro position (which is always reported).  The values are:
//      0   - The address is not included
//      1   - The address is included
// The default value is '1'.

// RESULT_INLINE_ERROR_TESTS
// For conditional macros (other than RETURN_XXX), this controls whether branches will be evaluated
// within the call containing the macro or will be forced into the function called by the macros.
// Pushing branching into the called function reduces code size and the number of unique branches
// evaluated, but increases the instruction count executed per macro.
//      0   - Branching will not happen inline to the macros
//      1   - Branching is pushed into the calling function via __forceinline
// The default value is '1'.  Note that XXX_MSG functions are always effectively mode '0' due to the
// compiler's unwillingness to inline var-arg functions.

// RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST
// RESULT_INCLUDE_CALLER_RETURNADDRESS_FAIL_FAST
// RESULT_INLINE_ERROR_TESTS_FAIL_FAST
// These defines are identical to those above in form/function, but only applicable to fail fast error
// handling allowing a process to have different diagnostic information and performance characteristics
// for fail fast than for other error handling given the different reporting infrastructure (Watson
// vs Telemetry).

// Setup the debug behavior
#ifndef RESULT_DEBUG
#if (DBG || defined(DEBUG) || defined(_DEBUG)) && !defined(NDEBUG)
#define RESULT_DEBUG
#endif
#endif

// Set the default diagnostic mode
// Note that RESULT_DEBUG_INFO and RESULT_SUPPRESS_DEBUG_INFO are older deprecated models of controlling mode
#ifndef RESULT_DIAGNOSTICS_LEVEL
#if (defined(RESULT_DEBUG) || defined(RESULT_DEBUG_INFO)) && !defined(RESULT_SUPPRESS_DEBUG_INFO)
#define RESULT_DIAGNOSTICS_LEVEL 5
#else
#define RESULT_DIAGNOSTICS_LEVEL 3
#endif
#endif
#ifndef RESULT_INCLUDE_CALLER_RETURNADDRESS
#define RESULT_INCLUDE_CALLER_RETURNADDRESS 1
#endif
#ifndef RESULT_INLINE_ERROR_TESTS
#define RESULT_INLINE_ERROR_TESTS 1
#endif
#ifndef RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST
#define RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST RESULT_DIAGNOSTICS_LEVEL
#endif
#ifndef RESULT_INCLUDE_CALLER_RETURNADDRESS_FAIL_FAST
#define RESULT_INCLUDE_CALLER_RETURNADDRESS_FAIL_FAST RESULT_INCLUDE_CALLER_RETURNADDRESS
#endif
#ifndef RESULT_INLINE_ERROR_TESTS_FAIL_FAST
#define RESULT_INLINE_ERROR_TESTS_FAIL_FAST RESULT_INLINE_ERROR_TESTS
#endif


//*****************************************************************************
// Win32 specific error macros
//*****************************************************************************

#define FAILED_WIN32(win32err)                              ((win32err) != 0)
#define SUCCEEDED_WIN32(win32err)                           ((win32err) == 0)


//*****************************************************************************
// NT_STATUS specific error macros
//*****************************************************************************

#define FAILED_NTSTATUS(status)                             (((NTSTATUS)(status)) < 0)
#define SUCCEEDED_NTSTATUS(status)                          (((NTSTATUS)(status)) >= 0)


//*****************************************************************************
// Testing helpers - redefine to run unit tests against fail fast
//*****************************************************************************

#ifndef RESULT_NORETURN
#define RESULT_NORETURN                                     __declspec(noreturn)
#endif
#ifndef RESULT_NORETURN_NULL
#define RESULT_NORETURN_NULL                                _Ret_notnull_
#endif
#ifndef RESULT_RAISE_FAST_FAIL_EXCEPTION
#define RESULT_RAISE_FAST_FAIL_EXCEPTION                    __fastfail(FAST_FAIL_FATAL_APP_EXIT)
#endif


//*****************************************************************************
// Helpers to setup the macros and functions used below... do not directly use.
//*****************************************************************************

/// @cond
#define __R_DIAGNOSTICS(diagnostics)                        diagnostics.returnAddress, diagnostics.line, diagnostics.file, nullptr, nullptr
#define __R_DIAGNOSTICS_RA(diagnostics, address)            diagnostics.returnAddress, diagnostics.line, diagnostics.file, nullptr, nullptr, address
#define __R_FN_PARAMS_FULL                                  _In_opt_ void* callerReturnAddress, unsigned int lineNumber, _In_opt_ PCSTR fileName, _In_opt_ PCSTR functionName, _In_opt_ PCSTR code, void* returnAddress
#define __R_FN_LOCALS_FULL_RA                               void* callerReturnAddress = nullptr; unsigned int lineNumber = 0; PCSTR fileName = nullptr; PCSTR functionName = nullptr; PCSTR code = nullptr; void* returnAddress = _ReturnAddress();
#define __R_ENABLE_IF_IS_CLASS(ptrType)                     typename wistd::enable_if_t<wistd::is_class<ptrType>::value, void*> = (void*)0
#define __R_ENABLE_IF_IS_NOT_CLASS(ptrType)                 typename wistd::enable_if_t<!wistd::is_class<ptrType>::value, void*> = (void*)0
// NOTE: This BEGINs the common macro handling (__R_ prefix) for non-fail fast handled cases
//       This entire section will be repeated below for fail fast (__RFF_ prefix).
#define __R_COMMA ,
#define __R_FN_CALL_FULL                                    callerReturnAddress, lineNumber, fileName, functionName, code, returnAddress
#define __R_FN_CALL_FULL_RA                                 callerReturnAddress, lineNumber, fileName, functionName, code, _ReturnAddress()
// The following macros assemble the varying amount of data we want to collect from the macros, treating it uniformly
#if (RESULT_DIAGNOSTICS_LEVEL >= 2)  // line number
#define __R_IF_LINE(term) term
#define __R_IF_NOT_LINE(term)
#define __R_IF_COMMA ,
#define __R_LINE_VALUE __LINE__
#else
#define __R_IF_LINE(term)
#define __R_IF_NOT_LINE(term) term
#define __R_IF_COMMA
#define __R_LINE_VALUE 0
#endif
#if (RESULT_DIAGNOSTICS_LEVEL >= 3) // line number + file name
#define __R_IF_FILE(term) term
#define __R_IF_NOT_FILE(term)
#define __R_FILE_VALUE __FILE__
#else
#define __R_IF_FILE(term)
#define __R_IF_NOT_FILE(term) term
#define __R_FILE_VALUE nullptr
#endif
#if (RESULT_DIAGNOSTICS_LEVEL >= 4) // line number + file name + function name
#define __R_IF_FUNCTION(term) term
#define __R_IF_NOT_FUNCTION(term)
#else
#define __R_IF_FUNCTION(term)
#define __R_IF_NOT_FUNCTION(term) term
#endif
#if (RESULT_DIAGNOSTICS_LEVEL >= 5) // line number + file name + function name + macro code
#define __R_IF_CODE(term) term
#define __R_IF_NOT_CODE(term)
#else
#define __R_IF_CODE(term)
#define __R_IF_NOT_CODE(term) term
#endif
#if (RESULT_INCLUDE_CALLER_RETURNADDRESS == 1)
#define __R_IF_CALLERADDRESS(term) term
#define __R_IF_NOT_CALLERADDRESS(term)
#define __R_CALLERADDRESS_VALUE _ReturnAddress()
#else
#define __R_IF_CALLERADDRESS(term)
#define __R_IF_NOT_CALLERADDRESS(term) term
#define __R_CALLERADDRESS_VALUE nullptr
#endif
#if (RESULT_INCLUDE_CALLER_RETURNADDRESS == 1) || (RESULT_DIAGNOSTICS_LEVEL >= 2)
#define __R_IF_TRAIL_COMMA ,
#else
#define __R_IF_TRAIL_COMMA
#endif
// Assemble the varying amounts of data into a single macro
#define __R_INFO_ONLY(CODE)                                 __R_IF_CALLERADDRESS(_ReturnAddress() __R_IF_COMMA) __R_IF_LINE(__R_LINE_VALUE) __R_IF_FILE(__R_COMMA __R_FILE_VALUE) __R_IF_FUNCTION(__R_COMMA __FUNCTION__) __R_IF_CODE(__R_COMMA CODE)
#define __R_INFO(CODE)                                      __R_INFO_ONLY(CODE) __R_IF_TRAIL_COMMA
#define __R_INFO_NOFILE_ONLY(CODE)                          __R_IF_CALLERADDRESS(_ReturnAddress() __R_IF_COMMA) __R_IF_LINE(__R_LINE_VALUE) __R_IF_FILE(__R_COMMA "wil") __R_IF_FUNCTION(__R_COMMA __FUNCTION__) __R_IF_CODE(__R_COMMA CODE)
#define __R_INFO_NOFILE(CODE)                               __R_INFO_NOFILE_ONLY(CODE) __R_IF_TRAIL_COMMA
#define __R_FN_PARAMS_ONLY                                  __R_IF_CALLERADDRESS(void* callerReturnAddress __R_IF_COMMA) __R_IF_LINE(unsigned int lineNumber) __R_IF_FILE(__R_COMMA _In_opt_ PCSTR fileName) __R_IF_FUNCTION(__R_COMMA _In_opt_ PCSTR functionName) __R_IF_CODE(__R_COMMA _In_opt_ PCSTR code)
#define __R_FN_PARAMS                                       __R_FN_PARAMS_ONLY __R_IF_TRAIL_COMMA
#define __R_FN_CALL_ONLY                                    __R_IF_CALLERADDRESS(callerReturnAddress __R_IF_COMMA) __R_IF_LINE(lineNumber) __R_IF_FILE(__R_COMMA fileName) __R_IF_FUNCTION(__R_COMMA functionName) __R_IF_CODE(__R_COMMA code)
#define __R_FN_CALL                                         __R_FN_CALL_ONLY __R_IF_TRAIL_COMMA
#define __R_FN_LOCALS                                       __R_IF_NOT_CALLERADDRESS(void* callerReturnAddress = nullptr;) __R_IF_NOT_LINE(unsigned int lineNumber = 0;) __R_IF_NOT_FILE(PCSTR fileName = nullptr;) __R_IF_NOT_FUNCTION(PCSTR functionName = nullptr;) __R_IF_NOT_CODE(PCSTR code = nullptr;)
#define __R_FN_LOCALS_RA                                    __R_IF_NOT_CALLERADDRESS(void* callerReturnAddress = nullptr;) __R_IF_NOT_LINE(unsigned int lineNumber = 0;) __R_IF_NOT_FILE(PCSTR fileName = nullptr;) __R_IF_NOT_FUNCTION(PCSTR functionName = nullptr;) __R_IF_NOT_CODE(PCSTR code = nullptr;) void* returnAddress = _ReturnAddress();
#define __R_FN_UNREFERENCED                                 __R_IF_CALLERADDRESS(callerReturnAddress;) __R_IF_LINE(lineNumber;) __R_IF_FILE(fileName;) __R_IF_FUNCTION(functionName;) __R_IF_CODE(code;)
// 1) Direct Methods
//      * Called Directly by Macros
//      * Always noinline
//      * May be template-driven to create unique call sites if (RESULT_DIAGNOSTICS_LEVEL == 1)
#if (RESULT_DIAGNOSTICS_LEVEL == 1)
#define __R_DIRECT_METHOD(RetType, MethodName)              template <unsigned int optimizerCounter> inline __declspec(noinline) RetType MethodName
#define __R_DIRECT_NORET_METHOD(RetType, MethodName)        template <unsigned int optimizerCounter> inline __declspec(noinline) RESULT_NORETURN RetType MethodName
#else
#define __R_DIRECT_METHOD(RetType, MethodName)              inline __declspec(noinline) RetType MethodName
#define __R_DIRECT_NORET_METHOD(RetType, MethodName)        inline __declspec(noinline) RESULT_NORETURN RetType MethodName
#endif
#define __R_DIRECT_FN_PARAMS                                __R_FN_PARAMS
#define __R_DIRECT_FN_PARAMS_ONLY                           __R_FN_PARAMS_ONLY
#define __R_DIRECT_FN_CALL                                  __R_FN_CALL_FULL_RA __R_COMMA
#define __R_DIRECT_FN_CALL_ONLY                             __R_FN_CALL_FULL_RA
// 2) Internal Methods
//      * Only called by Conditional routines
//      * 'inline' when (RESULT_INLINE_ERROR_TESTS = 0 and RESULT_DIAGNOSTICS_LEVEL != 1), otherwise noinline (directly called by code when branching is forceinlined)
//      * May be template-driven to create unique call sites if (RESULT_DIAGNOSTICS_LEVEL == 1 and RESULT_INLINE_ERROR_TESTS = 1)
#if (RESULT_DIAGNOSTICS_LEVEL == 1)
#define __R_INTERNAL_NOINLINE_METHOD(MethodName)            inline __declspec(noinline) void MethodName
#define __R_INTERNAL_NOINLINE_NORET_METHOD(MethodName)      inline __declspec(noinline) RESULT_NORETURN void MethodName
#define __R_INTERNAL_INLINE_METHOD(MethodName)              template <unsigned int optimizerCounter> inline __declspec(noinline) void MethodName
#define __R_INTERNAL_INLINE_NORET_METHOD(MethodName)        template <unsigned int optimizerCounter> inline __declspec(noinline) RESULT_NORETURN void MethodName
#define __R_CALL_INTERNAL_INLINE_METHOD(MethodName)         MethodName <optimizerCounter>
#else
#define __R_INTERNAL_NOINLINE_METHOD(MethodName)            inline void MethodName
#define __R_INTERNAL_NOINLINE_NORET_METHOD(MethodName)      inline RESULT_NORETURN void MethodName
#define __R_INTERNAL_INLINE_METHOD(MethodName)              inline __declspec(noinline) void MethodName
#define __R_INTERNAL_INLINE_NORET_METHOD(MethodName)        inline __declspec(noinline) RESULT_NORETURN void MethodName
#define __R_CALL_INTERNAL_INLINE_METHOD(MethodName)         MethodName
#endif
#define __R_CALL_INTERNAL_NOINLINE_METHOD(MethodName)       MethodName
#define __R_INTERNAL_NOINLINE_FN_PARAMS                     __R_FN_PARAMS void* returnAddress __R_COMMA
#define __R_INTERNAL_NOINLINE_FN_PARAMS_ONLY                __R_FN_PARAMS void* returnAddress
#define __R_INTERNAL_NOINLINE_FN_CALL                       __R_FN_CALL_FULL __R_COMMA
#define __R_INTERNAL_NOINLINE_FN_CALL_ONLY                  __R_FN_CALL_FULL
#define __R_INTERNAL_INLINE_FN_PARAMS                       __R_FN_PARAMS
#define __R_INTERNAL_INLINE_FN_PARAMS_ONLY                  __R_FN_PARAMS_ONLY
#define __R_INTERNAL_INLINE_FN_CALL                         __R_FN_CALL_FULL_RA __R_COMMA
#define __R_INTERNAL_INLINE_FN_CALL_ONLY                    __R_FN_CALL_FULL_RA
#if (RESULT_INLINE_ERROR_TESTS == 0)
#define __R_INTERNAL_METHOD                                 __R_INTERNAL_NOINLINE_METHOD
#define __R_INTERNAL_NORET_METHOD                           __R_INTERNAL_NOINLINE_NORET_METHOD
#define __R_CALL_INTERNAL_METHOD                            __R_CALL_INTERNAL_NOINLINE_METHOD
#define __R_INTERNAL_FN_PARAMS                              __R_INTERNAL_NOINLINE_FN_PARAMS
#define __R_INTERNAL_FN_PARAMS_ONLY                         __R_INTERNAL_NOINLINE_FN_PARAMS_ONLY
#define __R_INTERNAL_FN_CALL                                __R_INTERNAL_NOINLINE_FN_CALL
#define __R_INTERNAL_FN_CALL_ONLY                           __R_INTERNAL_NOINLINE_FN_CALL_ONLY
#else
#define __R_INTERNAL_METHOD                                 __R_INTERNAL_INLINE_METHOD
#define __R_INTERNAL_NORET_METHOD                           __R_INTERNAL_INLINE_NORET_METHOD
#define __R_CALL_INTERNAL_METHOD                            __R_CALL_INTERNAL_INLINE_METHOD
#define __R_INTERNAL_FN_PARAMS                              __R_INTERNAL_INLINE_FN_PARAMS
#define __R_INTERNAL_FN_PARAMS_ONLY                         __R_INTERNAL_INLINE_FN_PARAMS_ONLY
#define __R_INTERNAL_FN_CALL                                __R_INTERNAL_INLINE_FN_CALL
#define __R_INTERNAL_FN_CALL_ONLY                           __R_INTERNAL_INLINE_FN_CALL_ONLY
#endif
// 3) Conditional Methods
//      * Called Directly by Macros
//      * May be noinline or __forceinline depending upon (RESULT_INLINE_ERROR_TESTS)
//      * May be template-driven to create unique call sites if (RESULT_DIAGNOSTICS_LEVEL == 1)
#if (RESULT_DIAGNOSTICS_LEVEL == 1)
#define __R_CONDITIONAL_NOINLINE_METHOD(RetType, MethodName)            template <unsigned int optimizerCounter> inline __declspec(noinline) RetType MethodName
#define __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RetType, MethodName)   inline __declspec(noinline) RetType MethodName
#define __R_CONDITIONAL_INLINE_METHOD(RetType, MethodName)              template <unsigned int optimizerCounter> __forceinline RetType MethodName
#define __R_CONDITIONAL_INLINE_TEMPLATE_METHOD(RetType, MethodName)     __forceinline RetType MethodName
#define __R_CONDITIONAL_PARTIAL_TEMPLATE                                unsigned int optimizerCounter __R_COMMA
#else
#define __R_CONDITIONAL_NOINLINE_METHOD(RetType, MethodName)            inline __declspec(noinline) RetType MethodName
#define __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RetType, MethodName)   inline __declspec(noinline) RetType MethodName
#define __R_CONDITIONAL_INLINE_METHOD(RetType, MethodName)              __forceinline RetType MethodName
#define __R_CONDITIONAL_INLINE_TEMPLATE_METHOD(RetType, MethodName)     __forceinline RetType MethodName
#define __R_CONDITIONAL_PARTIAL_TEMPLATE
#endif
#define __R_CONDITIONAL_NOINLINE_FN_CALL                    __R_FN_CALL _ReturnAddress() __R_COMMA
#define __R_CONDITIONAL_NOINLINE_FN_CALL_ONLY               __R_FN_CALL _ReturnAddress()
#define __R_CONDITIONAL_INLINE_FN_CALL                      __R_FN_CALL
#define __R_CONDITIONAL_INLINE_FN_CALL_ONLY                 __R_FN_CALL_ONLY
#if (RESULT_INLINE_ERROR_TESTS == 0)
#define __R_CONDITIONAL_METHOD                              __R_CONDITIONAL_NOINLINE_METHOD
#define __R_CONDITIONAL_TEMPLATE_METHOD                     __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD
#define __R_CONDITIONAL_FN_CALL                             __R_CONDITIONAL_NOINLINE_FN_CALL
#define __R_CONDITIONAL_FN_CALL_ONLY                        __R_CONDITIONAL_NOINLINE_FN_CALL_ONLY
#else
#define __R_CONDITIONAL_METHOD                              __R_CONDITIONAL_INLINE_METHOD
#define __R_CONDITIONAL_TEMPLATE_METHOD                     __R_CONDITIONAL_INLINE_TEMPLATE_METHOD
#define __R_CONDITIONAL_FN_CALL                             __R_CONDITIONAL_INLINE_FN_CALL
#define __R_CONDITIONAL_FN_CALL_ONLY                        __R_CONDITIONAL_INLINE_FN_CALL_ONLY
#endif
#define __R_CONDITIONAL_FN_PARAMS                           __R_FN_PARAMS
#define __R_CONDITIONAL_FN_PARAMS_ONLY                      __R_FN_PARAMS_ONLY
// Macro call-site helpers
#define __R_NS_ASSEMBLE2(ri, rd)                            in##ri##diag##rd                // Differing internal namespaces eliminate ODR violations between modes
#define __R_NS_ASSEMBLE(ri, rd)                             __R_NS_ASSEMBLE2(ri, rd)
#define __R_NS_NAME                                         __R_NS_ASSEMBLE(RESULT_INLINE_ERROR_TESTS, RESULT_DIAGNOSTICS_LEVEL)
#define __R_NS wil::details::__R_NS_NAME
#if (RESULT_DIAGNOSTICS_LEVEL == 1)
#define __R_FN(MethodName)                                  __R_NS:: MethodName <__COUNTER__>
#else
#define __R_FN(MethodName)                                  __R_NS:: MethodName
#endif
// NOTE: This ENDs the common macro handling (__R_ prefix) for non-fail fast handled cases
//       This entire section is repeated below for fail fast (__RFF_ prefix).  For ease of editing this section, the
//       process is to copy/paste, and search and replace (__R_ -> __RFF_), (RESULT_DIAGNOSTICS_LEVEL -> RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST),
//       (RESULT_INLINE_ERROR_TESTS -> RESULT_INLINE_ERROR_TESTS_FAIL_FAST) and (RESULT_INCLUDE_CALLER_RETURNADDRESS -> RESULT_INCLUDE_CALLER_RETURNADDRESS_FAIL_FAST)
#define __RFF_COMMA ,
#define __RFF_FN_CALL_FULL                                    callerReturnAddress, lineNumber, fileName, functionName, code, returnAddress
#define __RFF_FN_CALL_FULL_RA                                 callerReturnAddress, lineNumber, fileName, functionName, code, _ReturnAddress()
// The following macros assemble the varying amount of data we want to collect from the macros, treating it uniformly
#if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST >= 2)  // line number
#define __RFF_IF_LINE(term) term
#define __RFF_IF_NOT_LINE(term)
#define __RFF_IF_COMMA ,
#else
#define __RFF_IF_LINE(term)
#define __RFF_IF_NOT_LINE(term) term
#define __RFF_IF_COMMA
#endif
#if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST >= 3) // line number + file name
#define __RFF_IF_FILE(term) term
#define __RFF_IF_NOT_FILE(term)
#else
#define __RFF_IF_FILE(term)
#define __RFF_IF_NOT_FILE(term) term
#endif
#if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST >= 4) // line number + file name + function name
#define __RFF_IF_FUNCTION(term) term
#define __RFF_IF_NOT_FUNCTION(term)
#else
#define __RFF_IF_FUNCTION(term)
#define __RFF_IF_NOT_FUNCTION(term) term
#endif
#if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST >= 5) // line number + file name + function name + macro code
#define __RFF_IF_CODE(term) term
#define __RFF_IF_NOT_CODE(term)
#else
#define __RFF_IF_CODE(term)
#define __RFF_IF_NOT_CODE(term) term
#endif
#if (RESULT_INCLUDE_CALLER_RETURNADDRESS_FAIL_FAST == 1)
#define __RFF_IF_CALLERADDRESS(term) term
#define __RFF_IF_NOT_CALLERADDRESS(term)
#else
#define __RFF_IF_CALLERADDRESS(term)
#define __RFF_IF_NOT_CALLERADDRESS(term) term
#endif
#if (RESULT_INCLUDE_CALLER_RETURNADDRESS_FAIL_FAST == 1) || (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST >= 2)
#define __RFF_IF_TRAIL_COMMA ,
#else
#define __RFF_IF_TRAIL_COMMA
#endif
// Assemble the varying amounts of data into a single macro
#define __RFF_INFO_ONLY(CODE)                                 __RFF_IF_CALLERADDRESS(_ReturnAddress() __RFF_IF_COMMA) __RFF_IF_LINE(__R_LINE_VALUE) __RFF_IF_FILE(__RFF_COMMA __R_FILE_VALUE) __RFF_IF_FUNCTION(__RFF_COMMA __FUNCTION__) __RFF_IF_CODE(__RFF_COMMA CODE)
#define __RFF_INFO(CODE)                                      __RFF_INFO_ONLY(CODE) __RFF_IF_TRAIL_COMMA
#define __RFF_INFO_NOFILE_ONLY(CODE)                          __RFF_IF_CALLERADDRESS(_ReturnAddress() __RFF_IF_COMMA) __RFF_IF_LINE(__R_LINE_VALUE) __RFF_IF_FILE(__RFF_COMMA "wil") __RFF_IF_FUNCTION(__RFF_COMMA __FUNCTION__) __RFF_IF_CODE(__RFF_COMMA CODE)
#define __RFF_INFO_NOFILE(CODE)                               __RFF_INFO_NOFILE_ONLY(CODE) __RFF_IF_TRAIL_COMMA
#define __RFF_FN_PARAMS_ONLY                                  __RFF_IF_CALLERADDRESS(void* callerReturnAddress __RFF_IF_COMMA) __RFF_IF_LINE(unsigned int lineNumber) __RFF_IF_FILE(__RFF_COMMA _In_opt_ PCSTR fileName) __RFF_IF_FUNCTION(__RFF_COMMA _In_opt_ PCSTR functionName) __RFF_IF_CODE(__RFF_COMMA _In_opt_ PCSTR code)
#define __RFF_FN_PARAMS                                       __RFF_FN_PARAMS_ONLY __RFF_IF_TRAIL_COMMA
#define __RFF_FN_CALL_ONLY                                    __RFF_IF_CALLERADDRESS(callerReturnAddress __RFF_IF_COMMA) __RFF_IF_LINE(lineNumber) __RFF_IF_FILE(__RFF_COMMA fileName) __RFF_IF_FUNCTION(__RFF_COMMA functionName) __RFF_IF_CODE(__RFF_COMMA code)
#define __RFF_FN_CALL                                         __RFF_FN_CALL_ONLY __RFF_IF_TRAIL_COMMA
#define __RFF_FN_LOCALS                                       __RFF_IF_NOT_CALLERADDRESS(void* callerReturnAddress = nullptr;) __RFF_IF_NOT_LINE(unsigned int lineNumber = 0;) __RFF_IF_NOT_FILE(PCSTR fileName = nullptr;) __RFF_IF_NOT_FUNCTION(PCSTR functionName = nullptr;) __RFF_IF_NOT_CODE(PCSTR code = nullptr;)
#define __RFF_FN_UNREFERENCED                                 __RFF_IF_CALLERADDRESS(callerReturnAddress;) __RFF_IF_LINE(lineNumber;) __RFF_IF_FILE(fileName;) __RFF_IF_FUNCTION(functionName;) __RFF_IF_CODE(code;)
// 1) Direct Methods
//      * Called Directly by Macros
//      * Always noinline
//      * May be template-driven to create unique call sites if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST == 1)
#if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST == 1)
#define __RFF_DIRECT_METHOD(RetType, MethodName)              template <unsigned int optimizerCounter> inline __declspec(noinline) RetType MethodName
#define __RFF_DIRECT_NORET_METHOD(RetType, MethodName)        template <unsigned int optimizerCounter> inline __declspec(noinline) RESULT_NORETURN RetType MethodName
#else
#define __RFF_DIRECT_METHOD(RetType, MethodName)              inline __declspec(noinline) RetType MethodName
#define __RFF_DIRECT_NORET_METHOD(RetType, MethodName)        inline __declspec(noinline) RESULT_NORETURN RetType MethodName
#endif
#define __RFF_DIRECT_FN_PARAMS                                __RFF_FN_PARAMS
#define __RFF_DIRECT_FN_PARAMS_ONLY                           __RFF_FN_PARAMS_ONLY
#define __RFF_DIRECT_FN_CALL                                  __RFF_FN_CALL_FULL_RA __RFF_COMMA
#define __RFF_DIRECT_FN_CALL_ONLY                             __RFF_FN_CALL_FULL_RA
// 2) Internal Methods
//      * Only called by Conditional routines
//      * 'inline' when (RESULT_INLINE_ERROR_TESTS_FAIL_FAST = 0 and RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST != 1), otherwise noinline (directly called by code when branching is forceinlined)
//      * May be template-driven to create unique call sites if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST == 1 and RESULT_INLINE_ERROR_TESTS_FAIL_FAST = 1)
#if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST == 1)
#define __RFF_INTERNAL_NOINLINE_METHOD(MethodName)            inline __declspec(noinline) void MethodName
#define __RFF_INTERNAL_NOINLINE_NORET_METHOD(MethodName)      inline __declspec(noinline) RESULT_NORETURN void MethodName
#define __RFF_INTERNAL_INLINE_METHOD(MethodName)              template <unsigned int optimizerCounter> inline __declspec(noinline) void MethodName
#define __RFF_INTERNAL_INLINE_NORET_METHOD(MethodName)        template <unsigned int optimizerCounter> inline __declspec(noinline) RESULT_NORETURN void MethodName
#define __RFF_CALL_INTERNAL_INLINE_METHOD(MethodName)         MethodName <optimizerCounter>
#else
#define __RFF_INTERNAL_NOINLINE_METHOD(MethodName)            inline void MethodName
#define __RFF_INTERNAL_NOINLINE_NORET_METHOD(MethodName)      inline RESULT_NORETURN void MethodName
#define __RFF_INTERNAL_INLINE_METHOD(MethodName)              inline __declspec(noinline) void MethodName
#define __RFF_INTERNAL_INLINE_NORET_METHOD(MethodName)        inline __declspec(noinline) RESULT_NORETURN void MethodName
#define __RFF_CALL_INTERNAL_INLINE_METHOD(MethodName)         MethodName
#endif
#define __RFF_CALL_INTERNAL_NOINLINE_METHOD(MethodName)       MethodName
#define __RFF_INTERNAL_NOINLINE_FN_PARAMS                     __RFF_FN_PARAMS void* returnAddress __RFF_COMMA
#define __RFF_INTERNAL_NOINLINE_FN_PARAMS_ONLY                __RFF_FN_PARAMS void* returnAddress
#define __RFF_INTERNAL_NOINLINE_FN_CALL                       __RFF_FN_CALL_FULL __RFF_COMMA
#define __RFF_INTERNAL_NOINLINE_FN_CALL_ONLY                  __RFF_FN_CALL_FULL
#define __RFF_INTERNAL_INLINE_FN_PARAMS                       __RFF_FN_PARAMS
#define __RFF_INTERNAL_INLINE_FN_PARAMS_ONLY                  __RFF_FN_PARAMS_ONLY
#define __RFF_INTERNAL_INLINE_FN_CALL                         __RFF_FN_CALL_FULL_RA __RFF_COMMA
#define __RFF_INTERNAL_INLINE_FN_CALL_ONLY                    __RFF_FN_CALL_FULL_RA
#if (RESULT_INLINE_ERROR_TESTS_FAIL_FAST == 0)
#define __RFF_INTERNAL_METHOD                                 __RFF_INTERNAL_NOINLINE_METHOD
#define __RFF_INTERNAL_NORET_METHOD                           __RFF_INTERNAL_NOINLINE_NORET_METHOD
#define __RFF_CALL_INTERNAL_METHOD                            __RFF_CALL_INTERNAL_NOINLINE_METHOD
#define __RFF_INTERNAL_FN_PARAMS                              __RFF_INTERNAL_NOINLINE_FN_PARAMS
#define __RFF_INTERNAL_FN_PARAMS_ONLY                         __RFF_INTERNAL_NOINLINE_FN_PARAMS_ONLY
#define __RFF_INTERNAL_FN_CALL                                __RFF_INTERNAL_NOINLINE_FN_CALL
#define __RFF_INTERNAL_FN_CALL_ONLY                           __RFF_INTERNAL_NOINLINE_FN_CALL_ONLY
#else
#define __RFF_INTERNAL_METHOD                                 __RFF_INTERNAL_INLINE_METHOD
#define __RFF_INTERNAL_NORET_METHOD                           __RFF_INTERNAL_INLINE_NORET_METHOD
#define __RFF_CALL_INTERNAL_METHOD                            __RFF_CALL_INTERNAL_INLINE_METHOD
#define __RFF_INTERNAL_FN_PARAMS                              __RFF_INTERNAL_INLINE_FN_PARAMS
#define __RFF_INTERNAL_FN_PARAMS_ONLY                         __RFF_INTERNAL_INLINE_FN_PARAMS_ONLY
#define __RFF_INTERNAL_FN_CALL                                __RFF_INTERNAL_INLINE_FN_CALL
#define __RFF_INTERNAL_FN_CALL_ONLY                           __RFF_INTERNAL_INLINE_FN_CALL_ONLY
#endif
// 3) Conditional Methods
//      * Called Directly by Macros
//      * May be noinline or __forceinline depending upon (RESULT_INLINE_ERROR_TESTS_FAIL_FAST)
//      * May be template-driven to create unique call sites if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST == 1)
#if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST == 1)
#define __RFF_CONDITIONAL_NOINLINE_METHOD(RetType, MethodName)            template <unsigned int optimizerCounter> inline __declspec(noinline) RetType MethodName
#define __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RetType, MethodName)   inline __declspec(noinline) RetType MethodName
#define __RFF_CONDITIONAL_INLINE_METHOD(RetType, MethodName)              template <unsigned int optimizerCounter> __forceinline RetType MethodName
#define __RFF_CONDITIONAL_INLINE_TEMPLATE_METHOD(RetType, MethodName)     __forceinline RetType MethodName
#define __RFF_CONDITIONAL_PARTIAL_TEMPLATE                                unsigned int optimizerCounter __RFF_COMMA
#else
#define __RFF_CONDITIONAL_NOINLINE_METHOD(RetType, MethodName)            inline __declspec(noinline) RetType MethodName
#define __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RetType, MethodName)   inline __declspec(noinline) RetType MethodName
#define __RFF_CONDITIONAL_INLINE_METHOD(RetType, MethodName)              __forceinline RetType MethodName
#define __RFF_CONDITIONAL_INLINE_TEMPLATE_METHOD(RetType, MethodName)     __forceinline RetType MethodName
#define __RFF_CONDITIONAL_PARTIAL_TEMPLATE
#endif
#define __RFF_CONDITIONAL_NOINLINE_FN_CALL                    __RFF_FN_CALL _ReturnAddress() __RFF_COMMA
#define __RFF_CONDITIONAL_NOINLINE_FN_CALL_ONLY               __RFF_FN_CALL _ReturnAddress()
#define __RFF_CONDITIONAL_INLINE_FN_CALL                      __RFF_FN_CALL
#define __RFF_CONDITIONAL_INLINE_FN_CALL_ONLY                 __RFF_FN_CALL_ONLY
#if (RESULT_INLINE_ERROR_TESTS_FAIL_FAST == 0)
#define __RFF_CONDITIONAL_METHOD                              __RFF_CONDITIONAL_NOINLINE_METHOD
#define __RFF_CONDITIONAL_TEMPLATE_METHOD                     __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD
#define __RFF_CONDITIONAL_FN_CALL                             __RFF_CONDITIONAL_NOINLINE_FN_CALL
#define __RFF_CONDITIONAL_FN_CALL_ONLY                        __RFF_CONDITIONAL_NOINLINE_FN_CALL_ONLY
#else
#define __RFF_CONDITIONAL_METHOD                              __RFF_CONDITIONAL_INLINE_METHOD
#define __RFF_CONDITIONAL_TEMPLATE_METHOD                     __RFF_CONDITIONAL_INLINE_TEMPLATE_METHOD
#define __RFF_CONDITIONAL_FN_CALL                             __RFF_CONDITIONAL_INLINE_FN_CALL
#define __RFF_CONDITIONAL_FN_CALL_ONLY                        __RFF_CONDITIONAL_INLINE_FN_CALL_ONLY
#endif
#define __RFF_CONDITIONAL_FN_PARAMS                           __RFF_FN_PARAMS
#define __RFF_CONDITIONAL_FN_PARAMS_ONLY                      __RFF_FN_PARAMS_ONLY
// Macro call-site helpers
#define __RFF_NS_ASSEMBLE2(ri, rd)                            in##ri##diag##rd                // Differing internal namespaces eliminate ODR violations between modes
#define __RFF_NS_ASSEMBLE(ri, rd)                             __RFF_NS_ASSEMBLE2(ri, rd)
#define __RFF_NS_NAME                                         __RFF_NS_ASSEMBLE(RESULT_INLINE_ERROR_TESTS_FAIL_FAST, RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST)
#define __RFF_NS wil::details::__RFF_NS_NAME
#if (RESULT_DIAGNOSTICS_LEVEL_FAIL_FAST == 1)
#define __RFF_FN(MethodName)                                  __RFF_NS:: MethodName <__COUNTER__>
#else
#define __RFF_FN(MethodName)                                  __RFF_NS:: MethodName
#endif
// end-of-repeated fail-fast handling macros
// Helpers for return macros
#define __RETURN_HR_MSG(hr, str, fmt, ...)                  do { HRESULT __hr = (hr); if (FAILED(__hr)) { __R_FN(Return_HrMsg)(__R_INFO(str) __hr, fmt, __VA_ARGS__); } return __hr; } while (0, 0)
#define __RETURN_HR_MSG_FAIL(hr, str, fmt, ...)             do { HRESULT __hr = (hr); __R_FN(Return_HrMsg)(__R_INFO(str) __hr, fmt, __VA_ARGS__); return __hr; } while (0, 0)
#define __RETURN_WIN32_MSG(err, str, fmt, ...)              do { DWORD __err = (err); if (FAILED_WIN32(__err)) { return __R_FN(Return_Win32Msg)(__R_INFO(str) __err, fmt, __VA_ARGS__); } return S_OK; } while (0, 0)
#define __RETURN_WIN32_MSG_FAIL(err, str, fmt, ...)         do { DWORD __err = (err); return __R_FN(Return_Win32Msg)(__R_INFO(str) __err, fmt, __VA_ARGS__); } while (0, 0)
#define __RETURN_GLE_MSG_FAIL(str, fmt, ...)                return __R_FN(Return_GetLastErrorMsg)(__R_INFO(str) fmt, __VA_ARGS__)
#define __RETURN_NTSTATUS_MSG(status, str, fmt, ...)        do { NTSTATUS __status = (status); if(FAILED_NTSTATUS(__status)) { return __R_FN(Return_NtStatusMsg)(__R_INFO(str) __status, fmt, __VA_ARGS__); } return S_OK; } while (0, 0)
#define __RETURN_NTSTATUS_MSG_FAIL(status, str, fmt, ...)   do { NTSTATUS __status = (status); return __R_FN(Return_NtStatusMsg)(__R_INFO(str) __status, fmt, __VA_ARGS__); } while (0, 0)
#define __RETURN_HR(hr, str)                                do { HRESULT __hr = (hr); if (FAILED(__hr)) { __R_FN(Return_Hr)(__R_INFO(str) __hr); } return __hr; } while (0, 0)
#define __RETURN_HR_NOFILE(hr, str)                         do { HRESULT __hr = (hr); if (FAILED(__hr)) { __R_FN(Return_Hr)(__R_INFO_NOFILE(str) __hr); } return __hr; } while (0, 0)
#define __RETURN_HR_FAIL(hr, str)                           do { HRESULT __hr = (hr); __R_FN(Return_Hr)(__R_INFO(str) __hr); return __hr; } while (0, 0)
#define __RETURN_HR_FAIL_NOFILE(hr, str)                    do { HRESULT __hr = (hr); __R_FN(Return_Hr)(__R_INFO_NOFILE(str) __hr); return __hr; } while (0, 0)
#define __RETURN_WIN32(err, str)                            do { DWORD __err = (err); if (FAILED_WIN32(__err)) { return __R_FN(Return_Win32)(__R_INFO(str) __err); } return S_OK; } while (0, 0)
#define __RETURN_WIN32_FAIL(err, str)                       do { DWORD __err = (err); return __R_FN(Return_Win32)(__R_INFO(str) __err); } while (0, 0)
#define __RETURN_GLE_FAIL(str)                              return __R_FN(Return_GetLastError)(__R_INFO_ONLY(str))
#define __RETURN_GLE_FAIL_NOFILE(str)                       return __R_FN(Return_GetLastError)(__R_INFO_NOFILE_ONLY(str))
#define __RETURN_NTSTATUS(status, str)                      do { NTSTATUS __status = (status); if(FAILED_NTSTATUS(__status)) { return __R_FN(Return_NtStatus)(__R_INFO(str) __status); } return S_OK; } while (0, 0)
#define __RETURN_NTSTATUS_FAIL(status, str)                 do { NTSTATUS __status = (status); return __R_FN(Return_NtStatus)(__R_INFO(str) __status); } while (0, 0)
#if defined(_PREFAST_)
#define __WI_ANALYSIS_ASSUME(_exp)                          _Analysis_assume_(_exp)
#else
#ifdef RESULT_DEBUG
#define __WI_ANALYSIS_ASSUME(_exp)                          ((void) 0)
#else
#define __WI_ANALYSIS_ASSUME(_exp)                          __noop(_exp)
#endif
#endif // _PREFAST_
/// @endcond


//*****************************************************************************
// Fail fast helpers (for use only internally to WIL)
//*****************************************************************************

/// @cond
#define __FAIL_FAST_ASSERT__(condition)                         do { if (!(condition)) { __RFF_FN(FailFast_Unexpected)(__RFF_INFO_ONLY(#condition)); } } while (0, 0)
#define __FAIL_FAST_IMMEDIATE_ASSERT__(condition)               do { if (!(condition)) { RESULT_RAISE_FAST_FAIL_EXCEPTION; } } while (0, 0)
#define __FAIL_FAST_ASSERT_WIN32_BOOL_FALSE__(condition)        __RFF_FN(FailFast_IfWin32BoolFalse)(__RFF_INFO(#condition) wil::verify_BOOL(condition))
/// @endcond


//*****************************************************************************
// Macros for returning failures as HRESULTs
//*****************************************************************************

// Always returns a known result (HRESULT) - always logs failures
#define RETURN_HR(hr)                                           __RETURN_HR(wil::verify_hresult(hr), #hr)
#define RETURN_LAST_ERROR()                                     __RETURN_GLE_FAIL(nullptr)
#define RETURN_WIN32(win32err)                                  __RETURN_WIN32(win32err, #win32err)
#define RETURN_NTSTATUS(status)                                 __RETURN_NTSTATUS(status, #status)

// Conditionally returns failures (HRESULT) - always logs failures
#define RETURN_IF_FAILED(hr)                                    do { HRESULT __hrRet = wil::verify_hresult(hr); if (FAILED(__hrRet)) { __RETURN_HR_FAIL(__hrRet, #hr); }} while (0, 0)
#define RETURN_IF_WIN32_BOOL_FALSE(win32BOOL)                   do { BOOL __boolRet = wil::verify_BOOL(win32BOOL); if (!__boolRet) { __RETURN_GLE_FAIL(#win32BOOL); }} while (0, 0)
#define RETURN_IF_WIN32_ERROR(win32err)                         do { DWORD __errRet = (win32err); if (FAILED_WIN32(__errRet)) { __RETURN_WIN32_FAIL(__errRet, #win32err); }} while (0, 0)
#define RETURN_IF_HANDLE_INVALID(handle)                        do { HANDLE __hRet = (handle); if (__hRet == INVALID_HANDLE_VALUE) { __RETURN_GLE_FAIL(#handle); }} while (0, 0)
#define RETURN_IF_HANDLE_NULL(handle)                           do { HANDLE __hRet = (handle); if (__hRet == nullptr) { __RETURN_GLE_FAIL(#handle); }} while (0, 0)
#define RETURN_IF_NULL_ALLOC(ptr)                               do { if ((ptr) == nullptr) { __RETURN_HR_FAIL(E_OUTOFMEMORY, #ptr); }} while (0, 0)
#define RETURN_HR_IF(hr, condition)                             do { if (wil::verify_bool(condition)) { __RETURN_HR(wil::verify_hresult(hr), #condition); }} while (0, 0)
#define RETURN_HR_IF_FALSE(hr, condition)                       RETURN_HR_IF(hr, !(wil::verify_bool(condition)))
#define RETURN_HR_IF_NULL(hr, ptr)                              do { if ((ptr) == nullptr) { __RETURN_HR(wil::verify_hresult(hr), #ptr); }} while (0, 0)
#define RETURN_LAST_ERROR_IF(condition)                         do { if (wil::verify_bool(condition)) { __RETURN_GLE_FAIL(#condition); }} while (0, 0)
#define RETURN_LAST_ERROR_IF_FALSE(condition)                   RETURN_LAST_ERROR_IF(!(wil::verify_bool(condition)))
#define RETURN_LAST_ERROR_IF_NULL(ptr)                          do { if ((ptr) == nullptr) { __RETURN_GLE_FAIL(#ptr); }} while (0, 0)
#define RETURN_IF_NTSTATUS_FAILED(status)                       do { NTSTATUS __statusRet = (status); if (FAILED_NTSTATUS(__statusRet)) { __RETURN_NTSTATUS_FAIL(__statusRet, #status); }} while (0, 0)

// Always returns a known failure (HRESULT) - always logs a var-arg message on failure
#define RETURN_HR_MSG(hr, fmt, ...)                             __RETURN_HR_MSG(wil::verify_hresult(hr), #hr, fmt, __VA_ARGS__)
#define RETURN_LAST_ERROR_MSG(fmt, ...)                         __RETURN_GLE_MSG_FAIL(nullptr, fmt, __VA_ARGS__)
#define RETURN_WIN32_MSG(win32err, fmt, ...)                    __RETURN_WIN32_MSG(win32err, #win32err, fmt, __VA_ARGS__)
#define RETURN_NTSTATUS_MSG(status, fmt, ...)                   __RETURN_NTSTATUS_MSG(status, #status, fmt, __VA_ARGS__)
    
// Conditionally returns failures (HRESULT) - always logs a var-arg message on failure
#define RETURN_IF_FAILED_MSG(hr, fmt, ...)                      do { auto __hrRet = wil::verify_hresult(hr); if (FAILED(__hrRet)) { __RETURN_HR_MSG_FAIL(__hrRet, #hr, fmt, __VA_ARGS__); }} while (0, 0)
#define RETURN_IF_WIN32_BOOL_FALSE_MSG(win32BOOL, fmt, ...)     do { if (!wil::verify_BOOL(win32BOOL)) { __RETURN_GLE_MSG_FAIL(#win32BOOL, fmt, __VA_ARGS__); }} while (0, 0)
#define RETURN_IF_WIN32_ERROR_MSG(win32err, fmt, ...)           do { auto __errRet = (win32err); if (FAILED_WIN32(__errRet)) { __RETURN_WIN32_MSG_FAIL(__errRet, #win32err, fmt, __VA_ARGS__); }} while (0, 0)
#define RETURN_IF_HANDLE_INVALID_MSG(handle, fmt, ...)          do { HANDLE __hRet = (handle); if (__hRet == INVALID_HANDLE_VALUE) { __RETURN_GLE_MSG_FAIL(#handle, fmt, __VA_ARGS__); }} while (0, 0)
#define RETURN_IF_HANDLE_NULL_MSG(handle, fmt, ...)             do { HANDLE __hRet = (handle); if (__hRet == nullptr) { __RETURN_GLE_MSG_FAIL(#handle, fmt, __VA_ARGS__); }} while (0, 0)
#define RETURN_IF_NULL_ALLOC_MSG(ptr, fmt, ...)                 do { if ((ptr) == nullptr) { __RETURN_HR_MSG_FAIL(E_OUTOFMEMORY, #ptr, fmt, __VA_ARGS__); }} while (0, 0)
#define RETURN_HR_IF_MSG(hr, condition, fmt, ...)               do { if (wil::verify_bool(condition)) { __RETURN_HR_MSG(wil::verify_hresult(hr), #condition, fmt, __VA_ARGS__); }} while (0, 0)
#define RETURN_HR_IF_FALSE_MSG(hr, condition, fmt, ...)         RETURN_HR_IF_MSG(hr, !(wil::verify_bool(condition)), fmt, __VA_ARGS__)
#define RETURN_HR_IF_NULL_MSG(hr, ptr, fmt, ...)                do { if ((ptr) == nullptr) { __RETURN_HR_MSG(wil::verify_hresult(hr), #ptr, fmt, __VA_ARGS__); }} while (0, 0)
#define RETURN_LAST_ERROR_IF_MSG(condition, fmt, ...)           do { if (wil::verify_bool(condition)) { __RETURN_GLE_MSG_FAIL(#condition, fmt, __VA_ARGS__); }} while (0, 0)
#define RETURN_LAST_ERROR_IF_FALSE_MSG(condition, fmt, ...)     RETURN_LAST_ERROR_IF_MSG(!(wil::verify_bool(condition)), fmt, __VA_ARGS__)
#define RETURN_LAST_ERROR_IF_NULL_MSG(ptr, fmt, ...)            do { if ((ptr) == nullptr) { __RETURN_GLE_MSG_FAIL(#ptr, fmt, __VA_ARGS__); }} while (0, 0)
#define RETURN_IF_NTSTATUS_FAILED_MSG(status, fmt, ...)         do { NTSTATUS __statusRet = (status); if (FAILED_NTSTATUS(__statusRet)) { __RETURN_NTSTATUS_MSG_FAIL(__statusRet, #status, fmt, __VA_ARGS__); }} while (0, 0)

// Conditionally returns failures (HRESULT) - use for failures that are expected in common use - failures are not logged - macros are only for control flow pattern
#define RETURN_IF_FAILED_EXPECTED(hr)                           do { auto __hrRet = wil::verify_hresult(hr); if (FAILED(__hrRet)) { return __hrRet; }} while (0, 0)
#define RETURN_IF_WIN32_BOOL_FALSE_EXPECTED(win32BOOL)          do { if (!wil::verify_BOOL(win32BOOL)) { return wil::details::GetLastErrorFailHr(); }} while (0, 0)
#define RETURN_IF_WIN32_ERROR_EXPECTED(win32err)                do { auto __errRet = (win32err); if (FAILED_WIN32(__errRet)) { return HRESULT_FROM_WIN32(__errRet); }} while (0, 0)
#define RETURN_IF_HANDLE_INVALID_EXPECTED(handle)               do { HANDLE __hRet = (handle); if (__hRet == INVALID_HANDLE_VALUE) { return wil::details::GetLastErrorFailHr(); }} while (0, 0)
#define RETURN_IF_HANDLE_NULL_EXPECTED(handle)                  do { HANDLE __hRet = (handle); if (__hRet == nullptr) { return wil::details::GetLastErrorFailHr(); }} while (0, 0)
#define RETURN_IF_NULL_ALLOC_EXPECTED(ptr)                      do { if ((ptr) == nullptr) { return E_OUTOFMEMORY; }} while (0, 0)
#define RETURN_HR_IF_EXPECTED(hr, condition)                    do { if (wil::verify_bool(condition)) { return wil::verify_hresult(hr); }} while (0, 0)
#define RETURN_HR_IF_FALSE_EXPECTED(hr, condition)              RETURN_HR_IF_EXPECTED(hr, !(wil::verify_bool(condition)))
#define RETURN_HR_IF_NULL_EXPECTED(hr, ptr)                     do { if ((ptr) == nullptr) { return wil::verify_hresult(hr); }} while (0, 0)
#define RETURN_LAST_ERROR_IF_EXPECTED(condition)                do { if (wil::verify_bool(condition)) { return wil::details::GetLastErrorFailHr(); }} while (0, 0)
#define RETURN_LAST_ERROR_IF_FALSE_EXPECTED(condition)          RETURN_LAST_ERROR_IF_EXPECTED(!(wil::verify_bool(condition)))
#define RETURN_LAST_ERROR_IF_NULL_EXPECTED(ptr)                 do { if ((ptr) == nullptr) { return wil::details::GetLastErrorFailHr(); }} while (0, 0)
#define RETURN_IF_NTSTATUS_FAILED_EXPECTED(status)              do { auto __statusRet = (status); if (FAILED_NTSTATUS(__statusRet)) { return wil::details::NtStatusToHr(__statusRet); }} while (0, 0)


//*****************************************************************************
// Macros for logging failures (ignore or pass-through)
//*****************************************************************************

// Always logs a known failure
#define LOG_HR(hr)                                              __R_FN(Log_Hr)(__R_INFO(#hr) wil::verify_hresult(hr))
#define LOG_LAST_ERROR()                                        __R_FN(Log_GetLastError)(__R_INFO_ONLY(nullptr))
#define LOG_WIN32(win32err)                                     __R_FN(Log_Win32)(__R_INFO(#win32err) win32err)
#define LOG_NTSTATUS(status)                                    __R_FN(Log_NtStatus)(__R_INFO(#status) status)

// Conditionally logs failures - returns parameter value
#define LOG_IF_FAILED(hr)                                       __R_FN(Log_IfFailed)(__R_INFO(#hr) wil::verify_hresult(hr))
#define LOG_IF_WIN32_BOOL_FALSE(win32BOOL)                      __R_FN(Log_IfWin32BoolFalse)(__R_INFO(#win32BOOL) wil::verify_BOOL(win32BOOL))
#define LOG_IF_WIN32_ERROR(win32err)                            __R_FN(Log_IfWin32Error)(__R_INFO(#win32err) win32err)
#define LOG_IF_HANDLE_INVALID(handle)                           __R_FN(Log_IfHandleInvalid)(__R_INFO(#handle) handle)
#define LOG_IF_HANDLE_NULL(handle)                              __R_FN(Log_IfHandleNull)(__R_INFO(#handle) handle)
#define LOG_IF_NULL_ALLOC(ptr)                                  __R_FN(Log_IfNullAlloc)(__R_INFO(#ptr) ptr)
#define LOG_HR_IF(hr, condition)                                __R_FN(Log_HrIf)(__R_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition))
#define LOG_HR_IF_FALSE(hr, condition)                          __R_FN(Log_HrIfFalse)(__R_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition))
#define LOG_HR_IF_NULL(hr, ptr)                                 __R_FN(Log_HrIfNull)(__R_INFO(#ptr) wil::verify_hresult(hr), ptr)
#define LOG_LAST_ERROR_IF(condition)                            __R_FN(Log_GetLastErrorIf)(__R_INFO(#condition) wil::verify_bool(condition))
#define LOG_LAST_ERROR_IF_FALSE(condition)                      __R_FN(Log_GetLastErrorIfFalse)(__R_INFO(#condition) wil::verify_bool(condition))
#define LOG_LAST_ERROR_IF_NULL(ptr)                             __R_FN(Log_GetLastErrorIfNull)(__R_INFO(#ptr) ptr)
#define LOG_IF_NTSTATUS_FAILED(status)                          __R_FN(Log_IfNtStatusFailed)(__R_INFO(#status) status)

// Alternatives for SUCCEEDED(hr) and FAILED(hr) that conditionally log failures
#define SUCCEEDED_LOG(hr)                                       SUCCEEDED(LOG_IF_FAILED(hr))
#define FAILED_LOG(hr)                                          FAILED(LOG_IF_FAILED(hr))
#define SUCCEEDED_WIN32_LOG(win32err)                           SUCCEEDED_WIN32(LOG_IF_WIN32_ERROR(win32err))
#define FAILED_WIN32_LOG(win32err)                              FAILED_WIN32(LOG_IF_WIN32_ERROR(win32err))
#define SUCCEEDED_NTSTATUS_LOG(status)                          SUCCEEDED_NTSTATUS(LOG_IF_NTSTATUS_FAILED(status))
#define FAILED_NTSTATUS_LOG(status)                             FAILED_NTSTATUS(LOG_IF_NTSTATUS_FAILED(status))

// Alternatives for NT_SUCCESS(x) that conditionally logs failures
#define NT_SUCCESS_LOG(status)                                  NT_SUCCESS(LOG_IF_NTSTATUS_FAILED(status))

// Always logs a known failure - logs a var-arg message on failure
#define LOG_HR_MSG(hr, fmt, ...)                                __R_FN(Log_HrMsg)(__R_INFO(#hr) wil::verify_hresult(hr), fmt, __VA_ARGS__)
#define LOG_LAST_ERROR_MSG(fmt, ...)                            __R_FN(Log_GetLastErrorMsg)(__R_INFO(nullptr) fmt, __VA_ARGS__)
#define LOG_WIN32_MSG(win32err, fmt, ...)                       __R_FN(Log_Win32Msg)(__R_INFO(#win32err) win32err, fmt, __VA_ARGS__)
#define LOG_NTSTATUS_MSG(status, fmt, ...)                      __R_FN(Log_NtStatusMsg)(__R_INFO(#status) status, fmt, __VA_ARGS__)

// Conditionally logs failures - returns parameter value - logs a var-arg message on failure
#define LOG_IF_FAILED_MSG(hr, fmt, ...)                         __R_FN(Log_IfFailedMsg)(__R_INFO(#hr) wil::verify_hresult(hr), fmt, __VA_ARGS__)
#define LOG_IF_WIN32_BOOL_FALSE_MSG(win32BOOL, fmt, ...)        __R_FN(Log_IfWin32BoolFalseMsg)(__R_INFO(#win32BOOL) wil::verify_BOOL(win32BOOL), fmt, __VA_ARGS__)
#define LOG_IF_WIN32_ERROR_MSG(win32err, fmt, ...)              __R_FN(Log_IfWin32ErrorMsg)(__R_INFO(#win32err) win32err, fmt, __VA_ARGS__)
#define LOG_IF_HANDLE_INVALID_MSG(handle, fmt, ...)             __R_FN(Log_IfHandleInvalidMsg)(__R_INFO(#handle) handle, fmt, __VA_ARGS__)
#define LOG_IF_HANDLE_NULL_MSG(handle, fmt, ...)                __R_FN(Log_IfHandleNullMsg)(__R_INFO(#handle) handle, fmt, __VA_ARGS__)
#define LOG_IF_NULL_ALLOC_MSG(ptr, fmt, ...)                    __R_FN(Log_IfNullAllocMsg)(__R_INFO(#ptr) ptr, fmt, __VA_ARGS__)
#define LOG_HR_IF_MSG(hr, condition, fmt, ...)                  __R_FN(Log_HrIfMsg)(__R_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition), fmt, __VA_ARGS__)
#define LOG_HR_IF_FALSE_MSG(hr, condition, fmt, ...)            __R_FN(Log_HrIfFalseMsg)(__R_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition), fmt, __VA_ARGS__)
#define LOG_HR_IF_NULL_MSG(hr, ptr, fmt, ...)                   __R_FN(Log_HrIfNullMsg)(__R_INFO(#ptr) wil::verify_hresult(hr), ptr, fmt, __VA_ARGS__)
#define LOG_LAST_ERROR_IF_MSG(condition, fmt, ...)              __R_FN(Log_GetLastErrorIfMsg)(__R_INFO(#condition) wil::verify_bool(condition), fmt, __VA_ARGS__)
#define LOG_LAST_ERROR_IF_FALSE_MSG(condition, fmt, ...)        __R_FN(Log_GetLastErrorIfFalseMsg)(__R_INFO(#condition) wil::verify_bool(condition), fmt, __VA_ARGS__)
#define LOG_LAST_ERROR_IF_NULL_MSG(ptr, fmt, ...)               __R_FN(Log_GetLastErrorIfNullMsg)(__R_INFO(#ptr) ptr, fmt, __VA_ARGS__)
#define LOG_IF_NTSTATUS_FAILED_MSG(status, fmt, ...)            __R_FN(Log_IfNtStatusFailedMsg)(__R_INFO(#status) status, fmt, __VA_ARGS__)


//*****************************************************************************
// Macros to fail fast the process on failures
//*****************************************************************************

// Always fail fast a known failure
#define FAIL_FAST_HR(hr)                                        __RFF_FN(FailFast_Hr)(__RFF_INFO(#hr) wil::verify_hresult(hr))
#define FAIL_FAST_LAST_ERROR()                                  __RFF_FN(FailFast_GetLastError)(__RFF_INFO_ONLY(nullptr))
#define FAIL_FAST_WIN32(win32err)                               __RFF_FN(FailFast_Win32)(__RFF_INFO(#win32err) win32err)
#define FAIL_FAST_NTSTATUS(status)                              __RFF_FN(FailFast_NtStatus)(__RFF_INFO(#status) status)
    
// Conditionally fail fast failures - returns parameter value
#define FAIL_FAST_IF_FAILED(hr)                                 __RFF_FN(FailFast_IfFailed)(__RFF_INFO(#hr) wil::verify_hresult(hr))
#define FAIL_FAST_IF_WIN32_BOOL_FALSE(win32BOOL)                __RFF_FN(FailFast_IfWin32BoolFalse)(__RFF_INFO(#win32BOOL) wil::verify_BOOL(win32BOOL))
#define FAIL_FAST_IF_WIN32_ERROR(win32err)                      __RFF_FN(FailFast_IfWin32Error)(__RFF_INFO(#win32err) win32err)
#define FAIL_FAST_IF_HANDLE_INVALID(handle)                     __RFF_FN(FailFast_IfHandleInvalid)(__RFF_INFO(#handle) handle)
#define FAIL_FAST_IF_HANDLE_NULL(handle)                        __RFF_FN(FailFast_IfHandleNull)(__RFF_INFO(#handle) handle)
#define FAIL_FAST_IF_NULL_ALLOC(ptr)                            __RFF_FN(FailFast_IfNullAlloc)(__RFF_INFO(#ptr) ptr)
#define FAIL_FAST_HR_IF(hr, condition)                          __RFF_FN(FailFast_HrIf)(__RFF_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition))
#define FAIL_FAST_HR_IF_FALSE(hr, condition)                    __RFF_FN(FailFast_HrIfFalse)(__RFF_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition))
#define FAIL_FAST_HR_IF_NULL(hr, ptr)                           __RFF_FN(FailFast_HrIfNull)(__RFF_INFO(#ptr) wil::verify_hresult(hr), ptr)
#define FAIL_FAST_LAST_ERROR_IF(condition)                      __RFF_FN(FailFast_GetLastErrorIf)(__RFF_INFO(#condition) wil::verify_bool(condition))
#define FAIL_FAST_LAST_ERROR_IF_FALSE(condition)                __RFF_FN(FailFast_GetLastErrorIfFalse)(__RFF_INFO(#condition) wil::verify_bool(condition))
#define FAIL_FAST_LAST_ERROR_IF_NULL(ptr)                       __RFF_FN(FailFast_GetLastErrorIfNull)(__RFF_INFO(#ptr) ptr)
#define FAIL_FAST_IF_NTSTATUS_FAILED(status)                    __RFF_FN(FailFast_IfNtStatusFailed)(__RFF_INFO(#status) status)

// Always fail fast a known failure - fail fast a var-arg message on failure
#define FAIL_FAST_HR_MSG(hr, fmt, ...)                          __RFF_FN(FailFast_HrMsg)(__RFF_INFO(#hr) wil::verify_hresult(hr), fmt, __VA_ARGS__)
#define FAIL_FAST_LAST_ERROR_MSG(fmt, ...)                      __RFF_FN(FailFast_GetLastErrorMsg)(__RFF_INFO(nullptr) fmt, __VA_ARGS__)
#define FAIL_FAST_WIN32_MSG(win32err, fmt, ...)                 __RFF_FN(FailFast_Win32Msg)(__RFF_INFO(#win32err) win32err, fmt, __VA_ARGS__)
#define FAIL_FAST_NTSTATUS_MSG(status, fmt, ...)                __RFF_FN(FailFast_NtStatusMsg)(__RFF_INFO(#status) status, fmt, __VA_ARGS__)

// Conditionally fail fast failures - returns parameter value - fail fast a var-arg message on failure
#define FAIL_FAST_IF_FAILED_MSG(hr, fmt, ...)                   __RFF_FN(FailFast_IfFailedMsg)(__RFF_INFO(#hr) wil::verify_hresult(hr), fmt, __VA_ARGS__)
#define FAIL_FAST_IF_WIN32_BOOL_FALSE_MSG(win32BOOL, fmt, ...)  __RFF_FN(FailFast_IfWin32BoolFalseMsg)(__RFF_INFO(#win32BOOL) wil::verify_BOOL(win32BOOL), fmt, __VA_ARGS__)
#define FAIL_FAST_IF_WIN32_ERROR_MSG(win32err, fmt, ...)        __RFF_FN(FailFast_IfWin32ErrorMsg)(__RFF_INFO(#win32err) win32err, fmt, __VA_ARGS__)
#define FAIL_FAST_IF_HANDLE_INVALID_MSG(handle, fmt, ...)       __RFF_FN(FailFast_IfHandleInvalidMsg)(__RFF_INFO(#handle) handle, fmt, __VA_ARGS__)
#define FAIL_FAST_IF_HANDLE_NULL_MSG(handle, fmt, ...)          __RFF_FN(FailFast_IfHandleNullMsg)(__RFF_INFO(#handle) handle, fmt, __VA_ARGS__)
#define FAIL_FAST_IF_NULL_ALLOC_MSG(ptr, fmt, ...)              __RFF_FN(FailFast_IfNullAllocMsg)(__RFF_INFO(#ptr) ptr, fmt, __VA_ARGS__)
#define FAIL_FAST_HR_IF_MSG(hr, condition, fmt, ...)            __RFF_FN(FailFast_HrIfMsg)(__RFF_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition), fmt, __VA_ARGS__)
#define FAIL_FAST_HR_IF_FALSE_MSG(hr, condition, fmt, ...)      __RFF_FN(FailFast_HrIfFalseMsg)(__RFF_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition), fmt, __VA_ARGS__)
#define FAIL_FAST_HR_IF_NULL_MSG(hr, ptr, fmt, ...)             __RFF_FN(FailFast_HrIfNullMsg)(__RFF_INFO(#ptr) wil::verify_hresult(hr), ptr, fmt, __VA_ARGS__)
#define FAIL_FAST_LAST_ERROR_IF_MSG(condition, fmt, ...)        __RFF_FN(FailFast_GetLastErrorIfMsg)(__RFF_INFO(#condition) wil::verify_bool(condition), fmt, __VA_ARGS__)
#define FAIL_FAST_LAST_ERROR_IF_FALSE_MSG(condition, fmt, ...)  __RFF_FN(FailFast_GetLastErrorIfFalseMsg)(__RFF_INFO(#condition) wil::verify_bool(condition), fmt, __VA_ARGS__)
#define FAIL_FAST_LAST_ERROR_IF_NULL_MSG(ptr, fmt, ...)         __RFF_FN(FailFast_GetLastErrorIfNullMsg)(__RFF_INFO(#ptr) ptr, fmt, __VA_ARGS__)
#define FAIL_FAST_IF_NTSTATUS_FAILED_MSG(status, fmt, ...)      __RFF_FN(FailFast_IfNtStatusFailedMsg)(__RFF_INFO(#status) status, fmt, __VA_ARGS__)

// Always fail fast a known failure
#ifndef FAIL_FAST
#define FAIL_FAST()                                             __RFF_FN(FailFast_Unexpected)(__RFF_INFO_ONLY(nullptr))
#endif

// Conditionally fail fast failures - returns parameter value
#define FAIL_FAST_IF(condition)                                 __RFF_FN(FailFast_If)(__RFF_INFO(#condition) wil::verify_bool(condition))
#define FAIL_FAST_IF_FALSE(condition)                           __RFF_FN(FailFast_IfFalse)(__RFF_INFO(#condition) wil::verify_bool(condition))
#define FAIL_FAST_IF_NULL(ptr)                                  __RFF_FN(FailFast_IfNull)(__RFF_INFO(#ptr) ptr)

// Always fail fast a known failure - fail fast a var-arg message on failure
#define FAIL_FAST_MSG(fmt, ...)                                 __RFF_FN(FailFast_UnexpectedMsg)(__RFF_INFO(nullptr) fmt, __VA_ARGS__)

// Conditionally fail fast failures - returns parameter value - fail fast a var-arg message on failure
#define FAIL_FAST_IF_MSG(condition, fmt, ...)                   __RFF_FN(FailFast_IfMsg)(__RFF_INFO(#condition) wil::verify_bool(condition), fmt, __VA_ARGS__)
#define FAIL_FAST_IF_FALSE_MSG(condition, fmt, ...)             __RFF_FN(FailFast_IfFalseMsg)(__RFF_INFO(#condition) wil::verify_bool(condition), fmt, __VA_ARGS__)
#define FAIL_FAST_IF_NULL_MSG(ptr, fmt, ...)                    __RFF_FN(FailFast_IfNullMsg)(__RFF_INFO(#ptr) ptr, fmt, __VA_ARGS__)

// Immediate fail fast (no telemetry - use rarely / only when *already* in an undefined state)
#define FAIL_FAST_IMMEDIATE()                                   __RFF_FN(FailFastImmediate_Unexpected)()

// Conditional immediate fail fast (no telemetry - use rarely / only when *already* in an undefined state)
#define FAIL_FAST_IMMEDIATE_IF_FAILED(hr)                       __RFF_FN(FailFastImmediate_IfFailed)(wil::verify_hresult(hr))
#define FAIL_FAST_IMMEDIATE_IF(condition)                       __RFF_FN(FailFastImmediate_If)(wil::verify_bool(condition))
#define FAIL_FAST_IMMEDIATE_IF_FALSE(condition)                 __RFF_FN(FailFastImmediate_IfFalse)(wil::verify_bool(condition))
#define FAIL_FAST_IMMEDIATE_IF_NULL(ptr)                        __RFF_FN(FailFastImmediate_IfNull)(ptr)
#define FAIL_FAST_IMMEDIATE_IF_NTSTATUS_FAILED(status)          __RFF_FN(FailFastImmediate_IfNtStatusFailed)(status)

// Specializations
#define FAIL_FAST_IMMEDIATE_IF_IN_LOADER_CALLOUT()              do { if (wil::details::g_pfnFailFastInLoaderCallout != nullptr) { wil::details::g_pfnFailFastInLoaderCallout(); } } while (0, 0)


//*****************************************************************************
// Macros to throw exceptions on failure
//*****************************************************************************

#ifdef WIL_ENABLE_EXCEPTIONS

// Always throw a known failure
#define THROW_HR(hr)                                            __R_FN(Throw_Hr)(__R_INFO(#hr) wil::verify_hresult(hr))
#define THROW_LAST_ERROR()                                      __R_FN(Throw_GetLastError)(__R_INFO_ONLY(nullptr))
#define THROW_WIN32(win32err)                                   __R_FN(Throw_Win32)(__R_INFO(#win32err) win32err)
#define THROW_EXCEPTION(exception)                              wil::details::ReportFailure_CustomException(__R_INFO(#exception) exception)
#define THROW_NTSTATUS(status)                                  __R_FN(Throw_NtStatus)(__R_INFO(#status) status)

// Conditionally throw failures - returns parameter value
#define THROW_IF_FAILED(hr)                                     __R_FN(Throw_IfFailed)(__R_INFO(#hr) wil::verify_hresult(hr))
#define THROW_IF_WIN32_BOOL_FALSE(win32BOOL)                    __R_FN(Throw_IfWin32BoolFalse)(__R_INFO(#win32BOOL) wil::verify_BOOL(win32BOOL))
#define THROW_IF_WIN32_ERROR(win32err)                          __R_FN(Throw_IfWin32Error)(__R_INFO(#win32err) win32err)
#define THROW_IF_HANDLE_INVALID(handle)                         __R_FN(Throw_IfHandleInvalid)(__R_INFO(#handle) handle)
#define THROW_IF_HANDLE_NULL(handle)                            __R_FN(Throw_IfHandleNull)(__R_INFO(#handle) handle)
#define THROW_IF_NULL_ALLOC(ptr)                                __R_FN(Throw_IfNullAlloc)(__R_INFO(#ptr) ptr)
#define THROW_HR_IF(hr, condition)                              __R_FN(Throw_HrIf)(__R_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition))
#define THROW_HR_IF_FALSE(hr, condition)                        __R_FN(Throw_HrIfFalse)(__R_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition))
#define THROW_HR_IF_NULL(hr, ptr)                               __R_FN(Throw_HrIfNull)(__R_INFO(#ptr) wil::verify_hresult(hr), ptr)
#define THROW_LAST_ERROR_IF(condition)                          __R_FN(Throw_GetLastErrorIf)(__R_INFO(#condition) wil::verify_bool(condition))
#define THROW_LAST_ERROR_IF_FALSE(condition)                    __R_FN(Throw_GetLastErrorIfFalse)(__R_INFO(#condition) wil::verify_bool(condition))
#define THROW_LAST_ERROR_IF_NULL(ptr)                           __R_FN(Throw_GetLastErrorIfNull)(__R_INFO(#ptr) ptr)
#define THROW_IF_NTSTATUS_FAILED(status)                        __R_FN(Throw_IfNtStatusFailed)(__R_INFO(#status) status)

// Always throw a known failure - throw a var-arg message on failure
#define THROW_HR_MSG(hr, fmt, ...)                              __R_FN(Throw_HrMsg)(__R_INFO(#hr) wil::verify_hresult(hr), fmt, __VA_ARGS__)
#define THROW_LAST_ERROR_MSG(fmt, ...)                          __R_FN(Throw_GetLastErrorMsg)(__R_INFO(nullptr) fmt, __VA_ARGS__)
#define THROW_WIN32_MSG(win32err, fmt, ...)                     __R_FN(Throw_Win32Msg)(__R_INFO(#win32err) win32err, fmt, __VA_ARGS__)
#define THROW_EXCEPTION_MSG(exception, fmt, ...)                wil::details::ReportFailure_CustomExceptionMsg(__R_INFO(#exception) exception, fmt, __VA_ARGS__)
#define THROW_NTSTATUS_MSG(status, fmt, ...)                    __R_FN(Throw_NtStatusMsg)(__R_INFO(#status) status, fmt, __VA_ARGS__)

// Conditionally throw failures - returns parameter value - throw a var-arg message on failure
#define THROW_IF_FAILED_MSG(hr, fmt, ...)                       __R_FN(Throw_IfFailedMsg)(__R_INFO(#hr) wil::verify_hresult(hr), fmt, __VA_ARGS__)
#define THROW_IF_WIN32_BOOL_FALSE_MSG(win32BOOL, fmt, ...)      __R_FN(Throw_IfWin32BoolFalseMsg)(__R_INFO(#win32BOOL) wil::verify_BOOL(win32BOOL), fmt, __VA_ARGS__)
#define THROW_IF_WIN32_ERROR_MSG(win32err, fmt, ...)            __R_FN(Throw_IfWin32ErrorMsg)(__R_INFO(#win32err) win32err, fmt, __VA_ARGS__)
#define THROW_IF_HANDLE_INVALID_MSG(handle, fmt, ...)           __R_FN(Throw_IfHandleInvalidMsg)(__R_INFO(#handle) handle, fmt, __VA_ARGS__)
#define THROW_IF_HANDLE_NULL_MSG(handle, fmt, ...)              __R_FN(Throw_IfHandleNullMsg)(__R_INFO(#handle) handle, fmt, __VA_ARGS__)
#define THROW_IF_NULL_ALLOC_MSG(ptr, fmt, ...)                  __R_FN(Throw_IfNullAllocMsg)(__R_INFO(#ptr) ptr, fmt, __VA_ARGS__)
#define THROW_HR_IF_MSG(hr, condition, fmt, ...)                __R_FN(Throw_HrIfMsg)(__R_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition), fmt, __VA_ARGS__)
#define THROW_HR_IF_FALSE_MSG(hr, condition, fmt, ...)          __R_FN(Throw_HrIfFalseMsg)(__R_INFO(#condition) wil::verify_hresult(hr), wil::verify_bool(condition), fmt, __VA_ARGS__)
#define THROW_HR_IF_NULL_MSG(hr, ptr, fmt, ...)                 __R_FN(Throw_HrIfNullMsg)(__R_INFO(#ptr) wil::verify_hresult(hr), ptr, fmt, __VA_ARGS__)
#define THROW_LAST_ERROR_IF_MSG(condition, fmt, ...)            __R_FN(Throw_GetLastErrorIfMsg)(__R_INFO(#condition) wil::verify_bool(condition), fmt, __VA_ARGS__)
#define THROW_LAST_ERROR_IF_FALSE_MSG(condition, fmt, ...)      __R_FN(Throw_GetLastErrorIfFalseMsg)(__R_INFO(#condition) wil::verify_bool(condition), fmt, __VA_ARGS__)
#define THROW_LAST_ERROR_IF_NULL_MSG(ptr, fmt, ...)             __R_FN(Throw_GetLastErrorIfNullMsg)(__R_INFO(#ptr) ptr, fmt, __VA_ARGS__)
#define THROW_IF_NTSTATUS_FAILED_MSG(status, fmt, ...)          __R_FN(Throw_IfNtStatusFailedMsg)(__R_INFO(#status) status, fmt, __VA_ARGS__)


//*****************************************************************************
// Macros to catch and convert exceptions on failure
//*****************************************************************************

// Use these macros *within* a catch (...) block to handle exceptions
#define RETURN_CAUGHT_EXCEPTION()                               return __R_FN(Return_CaughtException)(__R_INFO_ONLY(nullptr))
#define RETURN_CAUGHT_EXCEPTION_MSG(fmt, ...)                   return __R_FN(Return_CaughtExceptionMsg)(__R_INFO(nullptr) fmt, __VA_ARGS__)
#define RETURN_CAUGHT_EXCEPTION_EXPECTED()                      return wil::ResultFromCaughtException()
#define LOG_CAUGHT_EXCEPTION()                                  __R_FN(Log_CaughtException)(__R_INFO_ONLY(nullptr))
#define LOG_CAUGHT_EXCEPTION_MSG(fmt, ...)                      __R_FN(Log_CaughtExceptionMsg)(__R_INFO(nullptr) fmt, __VA_ARGS__)
#define FAIL_FAST_CAUGHT_EXCEPTION()                            __R_FN(FailFast_CaughtException)(__R_INFO_ONLY(nullptr))
#define FAIL_FAST_CAUGHT_EXCEPTION_MSG(fmt, ...)                __R_FN(FailFast_CaughtExceptionMsg)(__R_INFO(nullptr) fmt, __VA_ARGS__)
#define THROW_NORMALIZED_CAUGHT_EXCEPTION()                     __R_FN(Throw_CaughtException)(__R_INFO_ONLY(nullptr))
#define THROW_NORMALIZED_CAUGHT_EXCEPTION_MSG(fmt, ...)         __R_FN(Throw_CaughtExceptionMsg)(__R_INFO(nullptr) fmt, __VA_ARGS__)

// Use these macros in place of a catch block to handle exceptions
#define CATCH_RETURN()                                          catch (...) { RETURN_CAUGHT_EXCEPTION(); }
#define CATCH_RETURN_MSG(fmt, ...)                              catch (...) { RETURN_CAUGHT_EXCEPTION_MSG(fmt, __VA_ARGS__); }
#define CATCH_RETURN_EXPECTED()                                 catch (...) { RETURN_CAUGHT_EXCEPTION_EXPECTED(); }
#define CATCH_LOG()                                             catch (...) { LOG_CAUGHT_EXCEPTION(); }
#define CATCH_LOG_MSG(fmt, ...)                                 catch (...) { LOG_CAUGHT_EXCEPTION_MSG(fmt, __VA_ARGS__); }
#define CATCH_FAIL_FAST()                                       catch (...) { FAIL_FAST_CAUGHT_EXCEPTION(); }
#define CATCH_FAIL_FAST_MSG(fmt, ...)                           catch (...) { FAIL_FAST_CAUGHT_EXCEPTION_MSG(fmt, __VA_ARGS__); }
#define CATCH_THROW_NORMALIZED()                                catch (...) { THROW_NORMALIZED_CAUGHT_EXCEPTION(); }
#define CATCH_THROW_NORMALIZED_MSG(fmt, ...)                    catch (...) { THROW_NORMALIZED_CAUGHT_EXCEPTION_MSG(fmt, __VA_ARGS__); }

#endif  // WIL_ENABLE_EXCEPTIONS

// Use this macro to supply diagnostics information to wil::ResultFromException
#define WI_DIAGNOSTICS_INFO                                     wil::DiagnosticsInfo(__R_CALLERADDRESS_VALUE, __R_LINE_VALUE, __R_FILE_VALUE)
#define WI_DIAGNOSTICS_NAME(name)                               wil::DiagnosticsInfo(__R_CALLERADDRESS_VALUE, __R_LINE_VALUE, __R_FILE_VALUE, name)



//*****************************************************************************
// Assert Macros
//*****************************************************************************

#ifdef RESULT_DEBUG
#define WI_ASSERT(condition)                                (__WI_ANALYSIS_ASSUME(condition), ((!(condition)) ? (__annotation(L"Debug", L"AssertFail", L#condition), DbgRaiseAssertionFailure(), false) : true))
#define WI_ASSERT_MSG(condition, msg)                       (__WI_ANALYSIS_ASSUME(condition), ((!(condition)) ? (__annotation(L"Debug", L"AssertFail", L##msg), DbgRaiseAssertionFailure(), false) : true))
#define WI_ASSERT_NOASSUME                                  WI_ASSERT
#define WI_ASSERT_MSG_NOASSUME                              WI_ASSERT_MSG
#define WI_VERIFY                                           WI_ASSERT
#define WI_VERIFY_MSG                                       WI_ASSERT_MSG
#define WI_VERIFY_SUCCEEDED(condition)                      WI_ASSERT(SUCCEEDED(condition))
#else
#define WI_ASSERT(condition)                                (__WI_ANALYSIS_ASSUME(condition), 0)
#define WI_ASSERT_MSG(condition, msg)                       (__WI_ANALYSIS_ASSUME(condition), 0)
#define WI_ASSERT_NOASSUME(condition)                       ((void) 0)
#define WI_ASSERT_MSG_NOASSUME(condition, msg)              ((void) 0)
#define WI_VERIFY(condition)                                (__WI_ANALYSIS_ASSUME(condition), ((condition) ? true : false))
#define WI_VERIFY_MSG(condition, msg)                       (__WI_ANALYSIS_ASSUME(condition), ((condition) ? true : false))
#define WI_VERIFY_SUCCEEDED(condition)                      (__WI_ANALYSIS_ASSUME(SUCCEEDED(condition)), ((SUCCEEDED(condition)) ? true : false))
#endif // RESULT_DEBUG


//*****************************************************************************
// Usage Error Macros
//*****************************************************************************

#ifndef WI_USAGE_ASSERT_STOP
#define WI_USAGE_ASSERT_STOP(condition)                     WI_ASSERT(condition)
#endif
#ifdef RESULT_DEBUG
#define WI_USAGE_ERROR(msg, ...)                            do { LOG_HR_MSG(HRESULT_FROM_WIN32(ERROR_ASSERTION_FAILURE), msg, __VA_ARGS__); WI_USAGE_ASSERT_STOP(false); } while (0, 0)
#define WI_USAGE_ERROR_FORWARD(msg, ...)                    do { ReportFailure_ReplaceMsg(__R_FN_CALL_FULL, FailureType::Log, HRESULT_FROM_WIN32(ERROR_ASSERTION_FAILURE), msg, __VA_ARGS__); WI_USAGE_ASSERT_STOP(false); } while (0, 0)
#else
#define WI_USAGE_ERROR(msg, ...)                            do { LOG_HR(HRESULT_FROM_WIN32(ERROR_ASSERTION_FAILURE)); WI_USAGE_ASSERT_STOP(false); } while (0, 0)
#define WI_USAGE_ERROR_FORWARD(msg, ...)                    do { ReportFailure_Hr(__R_FN_CALL_FULL, FailureType::Log, HRESULT_FROM_WIN32(ERROR_ASSERTION_FAILURE)); WI_USAGE_ASSERT_STOP(false); } while (0, 0)
#endif
#define WI_USAGE_VERIFY(condition, msg, ...)                do { auto __passed = wil::verify_bool(condition); if (!__passed) { WI_USAGE_ERROR(msg, __VA_ARGS__); }} while (0, 0)
#define WI_USAGE_VERIFY_FORWARD(condition, msg, ...)        do { auto __passed = wil::verify_bool(condition); if (!__passed) { WI_USAGE_ERROR_FORWARD(msg, __VA_ARGS__); }} while (0, 0)
#ifdef RESULT_DEBUG
#define WI_USAGE_ASSERT(condition, msg, ...)                WI_USAGE_VERIFY(condition, msg, __VA_ARGS__)
#else
#define WI_USAGE_ASSERT(condition, msg, ...)
#endif

//*****************************************************************************
// Internal Error Macros - DO NOT USE - these are for internal WIL use only to reduce sizes of binaries that use WIL
//*****************************************************************************
#ifdef RESULT_DEBUG
#define __WIL_PRIVATE_RETURN_IF_FAILED(hr)                   RETURN_IF_FAILED(hr)
#define __WIL_PRIVATE_RETURN_HR_IF(hr, cond)                 RETURN_HR_IF(hr, cond)
#define __WIL_PRIVATE_RETURN_LAST_ERROR_IF(cond)             RETURN_LAST_ERROR_IF(cond)
#define __WIL_PRIVATE_RETURN_IF_WIN32_BOOL_FALSE(win32BOOL)  RETURN_IF_WIN32_BOOL_FALSE(win32BOOL)
#define __WIL_PRIVATE_RETURN_LAST_ERROR_IF_NULL(ptr)         RETURN_LAST_ERROR_IF_NULL(ptr)
#define __WIL_PRIVATE_RETURN_IF_NULL_ALLOC(ptr)              RETURN_IF_NULL_ALLOC(ptr)
#define __WIL_PRIVATE_RETURN_LAST_ERROR()                    RETURN_LAST_ERROR()
#define __WIL_PRIVATE_FAIL_FAST_HR_IF(hr, condition)         FAIL_FAST_HR_IF(hr, condition)
#define __WIL_PRIVATE_FAIL_FAST_HR(hr)                       FAIL_FAST_HR(hr)
#define __WIL_PRIVATE_LOG_HR(hr)                             LOG_HR(hr)
#else
#define __WIL_PRIVATE_RETURN_IF_FAILED(hr)                   do { HRESULT __hrRet = wil::verify_hresult(hr); if (FAILED(__hrRet)) { __RETURN_HR_FAIL_NOFILE(__hrRet, #hr); }} while (0, 0)
#define __WIL_PRIVATE_RETURN_HR_IF(hr, cond)                 do { if (wil::verify_bool(cond)) { __RETURN_HR_NOFILE(wil::verify_hresult(hr), #cond); }} while (0, 0)
#define __WIL_PRIVATE_RETURN_LAST_ERROR_IF(cond)             do { if (wil::verify_bool(cond)) { __RETURN_GLE_FAIL_NOFILE(#cond); }} while (0, 0)
#define __WIL_PRIVATE_RETURN_IF_WIN32_BOOL_FALSE(win32BOOL)  do { BOOL __boolRet = wil::verify_BOOL(win32BOOL); if (!__boolRet) { __RETURN_GLE_FAIL_NOFILE(#win32BOOL); }} while (0, 0)
#define __WIL_PRIVATE_RETURN_LAST_ERROR_IF_NULL(ptr)         do { if ((ptr) == nullptr) { __RETURN_GLE_FAIL_NOFILE(#ptr); }} while (0, 0)
#define __WIL_PRIVATE_RETURN_IF_NULL_ALLOC(ptr)              do { if ((ptr) == nullptr) { __RETURN_HR_FAIL_NOFILE(E_OUTOFMEMORY, #ptr); }} while (0, 0)
#define __WIL_PRIVATE_RETURN_LAST_ERROR()                    __RETURN_GLE_FAIL_NOFILE(nullptr)
#define __WIL_PRIVATE_FAIL_FAST_HR_IF(hr, condition)         __RFF_FN(FailFast_HrIf)(__RFF_INFO_NOFILE(#condition) wil::verify_hresult(hr), wil::verify_bool(condition))
#define __WIL_PRIVATE_FAIL_FAST_HR(hr)                       __RFF_FN(FailFast_Hr)(__RFF_INFO_NOFILE(#hr) wil::verify_hresult(hr))
#define __WIL_PRIVATE_LOG_HR(hr)                             __R_FN(Log_Hr)(__R_INFO_NOFILE(#hr) wil::verify_hresult(hr))
#endif

// [deprecated] Old macros names -- to be removed -- do not use
#define RETURN_HR_IF_TRUE           RETURN_HR_IF
#define RETURN_LAST_ERROR_IF_TRUE   RETURN_LAST_ERROR_IF
#define RETURN_HR_IF_TRUE_LOG       RETURN_HR_IF_LOG
#define RETURN_HR_IF_TRUE_MSG       RETURN_HR_IF_MSG
#define RETURN_HR_IF_TRUE_EXPECTED  RETURN_HR_IF_EXPECTED
#define LOG_HR_IF_TRUE              LOG_HR_IF
#define LOG_HR_IF_TRUE_MSG          LOG_HR_IF_MSG
#define LOG_LAST_ERROR_IF_TRUE      LOG_LAST_ERROR_IF
#define FAIL_FAST_HR_IF_TRUE        FAIL_FAST_HR_IF
#define RETURN_HR_LOG               RETURN_HR
#define RETURN_LAST_ERROR_LOG       RETURN_LAST_ERROR
#define RETURN_WIN32_LOG            RETURN_WIN32
#define RETURN_NTSTATUS_LOG         RETURN_NTSTATUS
#define RETURN_IF_FAILED_LOG            RETURN_IF_FAILED
#define RETURN_IF_WIN32_BOOL_FALSE_LOG  RETURN_IF_WIN32_BOOL_FALSE
#define RETURN_IF_WIN32_ERROR_LOG       RETURN_IF_WIN32_ERROR
#define RETURN_IF_HANDLE_INVALID_LOG    RETURN_IF_HANDLE_INVALID
#define RETURN_IF_HANDLE_NULL_LOG       RETURN_IF_HANDLE_NULL
#define RETURN_IF_NULL_ALLOC_LOG        RETURN_IF_NULL_ALLOC
#define RETURN_HR_IF_LOG                RETURN_HR_IF
#define RETURN_HR_IF_FALSE_LOG          RETURN_HR_IF_FALSE
#define RETURN_HR_IF_NULL_LOG           RETURN_HR_IF_NULL
#define RETURN_LAST_ERROR_IF_LOG        RETURN_LAST_ERROR_IF
#define RETURN_LAST_ERROR_IF_FALSE_LOG  RETURN_LAST_ERROR_IF_FALSE
#define RETURN_LAST_ERROR_IF_NULL_LOG   RETURN_LAST_ERROR_IF_NULL
#define RETURN_IF_NTSTATUS_FAILED_LOG   RETURN_IF_NTSTATUS_FAILED
#ifdef WIL_ENABLE_EXCEPTIONS
#define THROW_HR_IF_TRUE            THROW_HR_IF
#define THROW_LAST_ERROR_IF_TRUE    THROW_LAST_ERROR_IF
#define THROW_HR_IF_TRUE_MSG        THROW_HR_IF_MSG
#endif  // WIL_ENABLE_EXCEPTIONS

#ifndef WIL_SUPPRESS_PRIVATE_API_USE
// Forward declaration
extern "C" DECLSPEC_IMPORT VOID NTAPI LdrFastFailInLoaderCallout(VOID);
#endif

namespace wil
{
    // Indicates the kind of message / failure type that was used to produce a given error
    enum class FailureType
    {
        Exception,          // THROW_...
        Return,             // RETURN_..._LOG or RETURN_..._MSG
        Log,                // LOG_...
        FailFast            // FAIL_FAST_...
    };

    /** Use with functions and macros that allow customizing which kinds of exceptions are handled.
    This is used with methods like wil::ResultFromException and wil::ResultFromExceptionDebug. */
    enum class SupportedExceptions
    {
        Default,        //!< [Default] all well known exceptions (honors g_fResultFailFastUnknownExceptions).
        Known,          //!< [Known] all well known exceptions (including std::exception).
        All,            //!< [All] all exceptions, known or otherwise.
        None,           //!< [None] no exceptions at all, an exception will fail-fast where thrown.
        Thrown,         //!< [Thrown] exceptions thrown by wil only (Platform::Exception^ or ResultException).
        ThrownOrAlloc   //!< [ThrownOrAlloc] exceptions thrown by wil (Platform::Exception^ or ResultException) or std::bad_alloc.
    };

    // Represents the call context information about a given failure
    // No constructors, destructors or virtual members should be contained within
    struct CallContextInfo
    {
        long contextId;                         // incrementing ID for this call context (unique across an individual module load within process)
        PCSTR contextName;                      // the explicit name given to this context
        PCWSTR contextMessage;                  // [optional] Message that can be associated with the call context
    };

    // Represents all context information about a given failure
    // No constructors, destructors or virtual members should be contained within
    struct FailureInfo
    {
        FailureType type;
        HRESULT hr;
        long failureId;                         // incrementing ID for this specific failure (unique across an individual module load within process)
        PCWSTR pszMessage;                      // Message is only present for _MSG logging (it's the Sprintf message)
        DWORD threadId;                         // the thread this failure was originally encountered on
        PCSTR pszCode;                          // [debug only] Capture code from the macro
        PCSTR pszFunction;                      // [debug only] The function name
        PCSTR pszFile;
        unsigned int uLineNumber;
        int cFailureCount;                      // How many failures of 'type' have been reported in this module so far
        PCSTR pszCallContext;                   // General breakdown of the call context stack that generated this failure
        CallContextInfo callContextOriginating; // The outermost (first seen) call context
        CallContextInfo callContextCurrent;     // The most recently seen call context
        PCSTR pszModule;                        // The module where the failure originated
        void* returnAddress;                    // The return address to the point that called the macro
        void* callerReturnAddress;              // The return address of the function that includes the macro
    };

    //! Created automatically from using WI_DIAGNOSTICS_INFO to provide diagnostics to functions.
    //! Note that typically wil hides diagnostics from users under the covers by passing them automatically to functions as 
    //! parameters hidden behind a macro.  In some cases, the user needs to directly supply these, so this class provides
    //! the mechanism for that.  We only use this for user-passed content as it can't be directly controlled by RESULT_DIAGNOSTICS_LEVEL
    //! to ensure there are no ODR violations (though that variable still controls what parameters within this structure would be available).
    struct DiagnosticsInfo
    {
        void* returnAddress = nullptr;
        PCSTR file = nullptr;
        PCSTR name = nullptr;
        unsigned short line = 0;

        DiagnosticsInfo() = default;

        __forceinline DiagnosticsInfo(void* returnAddress_, unsigned short line_, PCSTR file_) :
            returnAddress(returnAddress_),
            file(file_),
            line(line_)
        {
        }

        __forceinline DiagnosticsInfo(void* returnAddress_, unsigned short line_, PCSTR file_, PCSTR name_) :
            returnAddress(returnAddress_),
            file(file_),
            name(name_),
            line(line_)
        {
        }
    };

    enum class ErrorReturn
    {
        Auto,
        None
    };

    // [optionally] Plug in error logging
    // Note:  This callback is deprecated.  Please use SetResultTelemetryFallback for telemetry or
    // SetResultLoggingCallback for observation.
    extern "C" __declspec(selectany) void(__stdcall *g_pfnResultLoggingCallback)(_Inout_ wil::FailureInfo *pFailure, _Inout_updates_opt_z_(cchDebugMessage) PWSTR pszDebugMessage, _Pre_satisfies_(cchDebugMessage > 0) size_t cchDebugMessage) WI_NOEXCEPT = nullptr;

    // [optional]
    // This can be explicitly set to control whether or not error messages will be output to OutputDebugString.  It can also
    // be set directly from within the debugger to force console logging for debugging purposes.
    #ifdef RESULT_DEBUG
    __declspec(selectany) bool g_fResultOutputDebugString = true;
    #else
    __declspec(selectany) bool g_fResultOutputDebugString = false;
    #endif

    /** Allows code-based control over writing failures to OutputDebugString
    When set, this method is queried on each failure write to determine whether or not the text-based
    failure information should be generated and written out. This provides a default behavior for
    logging output.
    ~~~~
    wil::g_pfnShouldOutputDebugString = [] { return IsDebuggerPresent() ? true : false; };
    ~~~~
    */
    __declspec(selectany) bool(__stdcall *g_pfnShouldOutputDebugString)() WI_NOEXCEPT = nullptr;

    // [optionally] Plug in additional exception-type support (return S_OK when *unable* to remap the exception)
    __declspec(selectany) HRESULT(__stdcall *g_pfnResultFromCaughtException)() WI_NOEXCEPT = nullptr;

    // [optionally] Use to configure fast fail of unknown exceptions (turn them off).
    __declspec(selectany) bool g_fResultFailFastUnknownExceptions = true;

    // [optionally] Set to false to a configure all THROW_XXX macros in C++/CX to throw ResultException rather than Platform::Exception^
    __declspec(selectany) bool g_fResultThrowPlatformException = true;

    // [optionally] Set to false to a configure all CATCH_ and CAUGHT_ macros to NOT support (fail-fast) std::exception based exceptions (other than std::bad_alloc and wil::ResultException)
    __declspec(selectany) bool g_fResultSupportStdException = true;

    /// @cond
    namespace details
    {
        // True if g_pfnResultLoggingCallback is set (allows cutting off backwards compat calls to the function)
        __declspec(selectany) bool g_resultMessageCallbackSet = false;

        _Success_(true) _Ret_range_(dest, destEnd)
        inline PWSTR LogStringPrintf(_Out_writes_to_ptr_(destEnd) _Always_(_Post_z_) PWSTR dest, _Pre_satisfies_(destEnd >= dest) PCWSTR destEnd, _In_ _Printf_format_string_ PCWSTR format, ...)
        {
            va_list argList;
            va_start(argList, format);
            StringCchVPrintfW(dest, (destEnd - dest), format, argList);
            return (destEnd == dest) ? dest : (dest + wcslen(dest));
        }
    }
    /// @endcond

    // This call generates the default logging string that makes its way to OutputDebugString for
    // any particular failure.  This string is also used to associate a failure with a PlatformException^ which
    // only allows a single string to be associated with the exception.
    inline HRESULT GetFailureLogString(_Out_writes_(cchDest) _Always_(_Post_z_) PWSTR pszDest, _Pre_satisfies_(cchDest > 0) _In_ size_t cchDest, _In_ FailureInfo const &failure) WI_NOEXCEPT
    {
        // This function was lenient to empty strings at one point and some callers became dependent on this beahvior
        if ((cchDest == 0) || (pszDest == nullptr))
        {
            return S_OK;
        }

        pszDest[0] = L'\0';

        // Call the logging callback (if present) to allow them to generate the debug string that will be pushed to the console
        // or the platform exception object if the caller desires it.
        if ((g_pfnResultLoggingCallback != nullptr) && details::g_resultMessageCallbackSet)
        {
            // older-form callback was a non-const FailureInfo*; conceptually this is const as callers should not be modifying
            g_pfnResultLoggingCallback(const_cast<FailureInfo*>(&failure), pszDest, cchDest);
        }

        // The callback only optionally needs to supply the debug string -- if the callback didn't populate it, yet we still want
        // it for OutputDebugString or exception message, then generate the default string.
        if (pszDest[0] == L'\0')
        {
            PCSTR pszType = "";
            switch (failure.type)
            {
            case FailureType::Exception:
                pszType = "Exception";
                break;
            case FailureType::Return:
                pszType = "ReturnHr";
                break;
            case FailureType::Log:
                pszType = "LogHr";
                break;
            case FailureType::FailFast:
                pszType = "FailFast";
                break;
            }

            wchar_t szErrorText[256];
            szErrorText[0] = L'\0';
            FormatMessageW(FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS, nullptr, failure.hr, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), szErrorText, ARRAYSIZE(szErrorText), nullptr);

            // %FILENAME(%LINE): %TYPE(%count) tid(%threadid) %HRESULT %SystemMessage
            //     %Caller_MSG [%CODE(%FUNCTION)]

            PWSTR dest = pszDest;
            PCWSTR destEnd = (pszDest + cchDest);

            if (failure.pszFile != nullptr)
            {
                dest = details::LogStringPrintf(dest, destEnd, L"%hs(%d)\\%hs!%p: ", failure.pszFile, failure.uLineNumber, failure.pszModule, failure.returnAddress);
            }
            else
            {
                dest = details::LogStringPrintf(dest, destEnd, L"%hs!%p: ", failure.pszModule, failure.returnAddress);
            }

            if (failure.callerReturnAddress != nullptr)
            {
                dest = details::LogStringPrintf(dest, destEnd, L"(caller: %p) ", failure.callerReturnAddress);
            }

            dest = details::LogStringPrintf(dest, destEnd, L"%hs(%d) tid(%x) %08X %ws", pszType, failure.cFailureCount, ::GetCurrentThreadId(), failure.hr, szErrorText);

            if ((failure.pszMessage != nullptr) || (failure.pszCallContext != nullptr) || (failure.pszFunction != nullptr))
            {
                dest = details::LogStringPrintf(dest, destEnd, L"    ");
                if (failure.pszMessage != nullptr)
                {
                    dest = details::LogStringPrintf(dest, destEnd, L"Msg:[%ws] ", failure.pszMessage);
                }
                if (failure.pszCallContext != nullptr)
                {
                    dest = details::LogStringPrintf(dest, destEnd, L"CallContext:[%hs] ", failure.pszCallContext);
                }

                if (failure.pszCode != nullptr)
                {
                    dest = details::LogStringPrintf(dest, destEnd, L"[%hs(%hs)]\n", failure.pszFunction, failure.pszCode);
                }
                else if (failure.pszFunction != nullptr)
                {
                    dest = details::LogStringPrintf(dest, destEnd, L"[%hs]\n", failure.pszFunction);
                }
                else
                {
                    dest = details::LogStringPrintf(dest, destEnd, L"\n");
                }
            }
        }

        // Explicitly choosing to return success in the event of truncation... Current callers
        // depend upon it or it would be eliminated.
        return S_OK;
    }

    /// @cond
    namespace details
    {
        #ifdef WIL_SUPPRESS_PRIVATE_API_USE
        #pragma detect_mismatch("ODR_violation_WIL_SUPPRESS_PRIVATE_API_USE_mismatch", "1")
        #else
        #pragma detect_mismatch("ODR_violation_WIL_SUPPRESS_PRIVATE_API_USE_mismatch", "0")
        #endif

        //! Interface used to wrap up code (generally a lambda or other functor) to run in an exception-managed context where
        //! exceptions or errors can be observed and logged.
        struct IFunctor
        {
            virtual HRESULT Run() = 0;
        };

        //! Used to provide custom behavior when an exception is encountered while executing IFunctor
        struct IFunctorHost
        {
            virtual HRESULT Run(IFunctor& functor) = 0;
            virtual HRESULT ExceptionThrown(void* returnAddress) = 0;
        };

        // Fallback telemetry provider callback (set with wil::SetResultTelemetryFallback)
        __declspec(selectany) void(__stdcall *g_pfnTelemetryCallback)(bool alreadyReported, wil::FailureInfo const &failure) WI_NOEXCEPT = nullptr;

        // Result.h plug-in (WIL use only)
        __declspec(selectany) void(__stdcall *g_pfnGetContextAndNotifyFailure)(_Inout_ FailureInfo *pFailure, _Out_writes_(callContextStringLength) _Post_z_ PSTR callContextString, _Pre_satisfies_(callContextStringLength > 0) size_t callContextStringLength) WI_NOEXCEPT = nullptr;

        // Observe all errors flowing through the system with this callback (set with wil::SetResultLoggingCallback); use with custom logging
        __declspec(selectany) void(__stdcall *g_pfnLoggingCallback)(wil::FailureInfo const &failure) WI_NOEXCEPT = nullptr;

        // Desktop/System Only:  Module fetch function (automatically setup)
        __declspec(selectany) PCSTR(__stdcall *g_pfnGetModuleName)() WI_NOEXCEPT = nullptr;

        // Desktop/System Only:  Retrieve address offset and modulename
        __declspec(selectany) bool(__stdcall *g_pfnGetModuleInformation)(void* address, _Out_opt_ unsigned int* addressOffset, _Out_writes_bytes_opt_(size) char* name, size_t size) WI_NOEXCEPT = nullptr;

        // Desktop/System Only:  Private module load fail fast function (automatically setup)
        __declspec(selectany) void(__stdcall *g_pfnFailFastInLoaderCallout)() WI_NOEXCEPT = nullptr;

        // Desktop/System Only:  Private module load convert NtStatus to HResult (automatically setup)
        __declspec(selectany) ULONG(__stdcall *g_pfnRtlNtStatusToDosErrorNoTeb)(NTSTATUS) WI_NOEXCEPT = nullptr;

        // Private API to determine whether or not termination is happening
        __declspec(selectany) BOOLEAN(__stdcall *g_pfnRtlDllShutdownInProgress)() WI_NOEXCEPT = nullptr;
        __declspec(selectany) bool g_processShutdownInProgress = false;

        // Exception-based compiled additions
        __declspec(selectany) HRESULT(__stdcall *g_pfnRunFunctorWithExceptionFilter)(IFunctor& functor, IFunctorHost& host, void* returnAddress) = nullptr;
        __declspec(selectany) void(__stdcall *g_pfnRethrow)() = nullptr;
        __declspec(selectany) void(__stdcall *g_pfnThrowResultException)(const FailureInfo& failure) = nullptr;
        extern "C" __declspec(selectany) HRESULT(__stdcall *g_pfnResultFromCaughtExceptionInternal)(_Out_writes_opt_(debugStringChars) PWSTR debugString, _When_(debugString != nullptr, _Pre_satisfies_(debugStringChars > 0)) size_t debugStringChars, _Out_ bool* isNormalized) WI_NOEXCEPT = nullptr;

        // C++/cx compiled additions
        extern "C" __declspec(selectany) void(__stdcall *g_pfnThrowPlatformException)(FailureInfo const &failure, PCWSTR debugString) = nullptr;
        extern "C" __declspec(selectany) _Always_(_Post_satisfies_(return < 0)) HRESULT(__stdcall *g_pfnResultFromCaughtException_WinRt)(_Inout_updates_opt_(debugStringChars) PWSTR debugString, _When_(debugString != nullptr, _Pre_satisfies_(debugStringChars > 0)) size_t debugStringChars, _Out_ bool* isNormalized) WI_NOEXCEPT = nullptr;
        __declspec(selectany) _Always_(_Post_satisfies_(return < 0)) HRESULT(__stdcall *g_pfnResultFromKnownExceptions_WinRt)(const DiagnosticsInfo& diagnostics, void* returnAddress, SupportedExceptions supported, IFunctor& functor) WI_NOEXCEPT = nullptr;

        enum class ReportFailureOptions
        {
            None                    = 0x00,
            ForcePlatformException  = 0x01,
            SuppressAction          = 0x02,
            MayRethrow              = 0x04,
        };
        DEFINE_ENUM_FLAG_OPERATORS(ReportFailureOptions);

        template <typename TFunctor>
        using functor_return_type = decltype((*static_cast<TFunctor*>(nullptr))());

        template <typename TFunctor>
        struct functor_wrapper_void : public IFunctor
        {
            TFunctor&& functor;
            functor_wrapper_void(TFunctor&& functor_) : functor(wistd::forward<TFunctor>(functor_)) { }
            HRESULT Run() override
            {
                functor();
                return S_OK;
            }
        };

        template <typename TFunctor>
        struct functor_wrapper_HRESULT : public IFunctor
        {
            TFunctor&& functor;
            functor_wrapper_HRESULT(TFunctor& functor_) : functor(wistd::forward<TFunctor>(functor_)) { }
            HRESULT Run() override
            {
                return functor();
            }
        };

        template <typename TFunctor, typename TReturn>
        struct functor_wrapper_other : public IFunctor
        {
            TFunctor&& functor;
            TReturn& retVal;
            functor_wrapper_other(TFunctor& functor_, TReturn& retval_) : functor(wistd::forward<TFunctor>(functor_)), retVal(retval_) { }
            HRESULT Run() override
            {
                retVal = functor();
                return S_OK;
            }
        };

        struct tag_return_void : public wistd::integral_constant<size_t, 0>
        {
            template <typename TFunctor>
            using functor_wrapper = functor_wrapper_void<TFunctor>;
        };

        struct tag_return_HRESULT : public wistd::integral_constant<size_t, 1>
        {
            template <typename TFunctor>
            using functor_wrapper = functor_wrapper_HRESULT<TFunctor>;
        };

        struct tag_return_other : public wistd::integral_constant<size_t, 2>
        {
            template <typename TFunctor, typename TReturn>
            using functor_wrapper = functor_wrapper_other<TFunctor, TReturn>;
        };

        // type-trait to help discover the return type of a functor for tag/dispatch.

        template <ErrorReturn errorReturn, typename T>
        struct return_type
        {
            typedef tag_return_other type;
        };

        template <>
        struct return_type<ErrorReturn::Auto, HRESULT>
        {
            typedef tag_return_HRESULT type;
        };

        template <>
        struct return_type<ErrorReturn::Auto, void>
        {
            typedef tag_return_void type;
        };

        template <>
        struct return_type<ErrorReturn::None, void>
        {
            typedef tag_return_void type;
        };

        template <ErrorReturn errorReturn, typename Functor>
        using functor_tag = typename return_type<errorReturn, functor_return_type<Functor>>::type;

        // Forward declarations to enable use of fail fast and reporting internally...
        namespace __R_NS_NAME
        {
            __R_DIRECT_METHOD(HRESULT, Log_Hr)(__R_DIRECT_FN_PARAMS HRESULT hr) WI_NOEXCEPT;
            __R_DIRECT_METHOD(HRESULT, Log_HrMsg)(__R_DIRECT_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT;
            __R_DIRECT_METHOD(DWORD, Log_Win32Msg)(__R_DIRECT_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT;
        }
        namespace __RFF_NS_NAME
        {
            __RFF_DIRECT_NORET_METHOD(void, FailFast_Unexpected)(__RFF_DIRECT_FN_PARAMS_ONLY) WI_NOEXCEPT;
            __RFF_CONDITIONAL_METHOD(bool, FailFast_If)(__RFF_CONDITIONAL_FN_PARAMS bool condition) WI_NOEXCEPT;
            __RFF_CONDITIONAL_METHOD(bool, FailFast_HrIf)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition) WI_NOEXCEPT;
            __RFF_CONDITIONAL_METHOD(bool, FailFast_IfFalse)(__RFF_CONDITIONAL_FN_PARAMS bool condition) WI_NOEXCEPT;
            __RFF_CONDITIONAL_METHOD(bool, FailFastImmediate_If)(bool condition) WI_NOEXCEPT;
        }

        inline void LogFailure(__R_FN_PARAMS_FULL, FailureType type, HRESULT hr, _In_opt_ PCWSTR message,
                               bool fWantDebugString, _Out_writes_(debugStringSizeChars) _Post_z_ PWSTR debugString, _Pre_satisfies_(debugStringSizeChars > 0) size_t debugStringSizeChars,
                               _Out_writes_(callContextStringSizeChars) _Post_z_ PSTR callContextString, _Pre_satisfies_(callContextStringSizeChars > 0) size_t callContextStringSizeChars,
                               _Out_ FailureInfo *failure) WI_NOEXCEPT;
        __declspec(noinline) inline void ReportFailure(__R_FN_PARAMS_FULL, FailureType type, HRESULT hr, _In_opt_ PCWSTR message = nullptr, ReportFailureOptions options = ReportFailureOptions::None);
        inline void ReportFailure_ReplaceMsg(__R_FN_PARAMS_FULL, FailureType type, HRESULT hr, _Printf_format_string_ PCSTR formatString, ...);
        __declspec(noinline) inline void ReportFailure_Hr(__R_FN_PARAMS_FULL, FailureType type, HRESULT hr);
        __declspec(noinline) inline HRESULT ReportFailure_CaughtException(__R_FN_PARAMS_FULL, FailureType type, SupportedExceptions supported = SupportedExceptions::Default);

        // A simple ref-counted buffer class.  The interface is very similar to shared_ptr<>, only it manages
        // an allocated buffer (malloc) and maintains the size.

        class shared_buffer
        {
        public:
            shared_buffer() WI_NOEXCEPT : m_pCopy(nullptr), m_size(0)
            {
            }

            shared_buffer(shared_buffer const &other) WI_NOEXCEPT : m_pCopy(nullptr), m_size(0)
            {
                assign(other.m_pCopy, other.m_size);
            }

            shared_buffer(shared_buffer &&other) WI_NOEXCEPT :
                m_pCopy(other.m_pCopy),
                m_size(other.m_size)
            {
                other.m_pCopy = nullptr;
                other.m_size = 0;
            }

            ~shared_buffer() WI_NOEXCEPT
            {
                reset();
            }

            shared_buffer& operator=(shared_buffer const &other) WI_NOEXCEPT
            {
                assign(other.m_pCopy, other.m_size);
                return *this;
            }

            shared_buffer& operator=(shared_buffer &&other) WI_NOEXCEPT
            {
                reset();
                m_pCopy = other.m_pCopy;
                m_size = other.m_size;
                other.m_pCopy = nullptr;
                other.m_size = 0;
                return *this;
            }

            void reset() WI_NOEXCEPT
            {
                if (m_pCopy != nullptr)
                {
                    if (0 == ::InterlockedDecrementRelease(m_pCopy))
                    {
                        free(m_pCopy);
                    }
                    m_pCopy = nullptr;
                    m_size = 0;
                }
            }

            bool create(_In_reads_bytes_opt_(cbData) void const *pData, size_t cbData) WI_NOEXCEPT
            {
                if (cbData == 0)
                {
                    reset();
                    return true;
                }

                long *pCopyRefCount = reinterpret_cast<long *>(malloc(sizeof(long)+cbData));
                if (pCopyRefCount == nullptr)
                {
                    return false;
                }

                *pCopyRefCount = 0;
                if (pData != nullptr)
                {
                    memcpy_s(pCopyRefCount + 1, cbData, pData, cbData); // +1 to advance past sizeof(long) counter
                }
                assign(pCopyRefCount, cbData);
                return true;
            }

            bool create(size_t cbData) WI_NOEXCEPT
            {
                return create(nullptr, cbData);
            }

            void* get(_Out_opt_ size_t *pSize = nullptr) const WI_NOEXCEPT
            {
                if (pSize != nullptr)
                {
                    *pSize = m_size;
                }
                return (m_pCopy == nullptr) ? nullptr : (m_pCopy + 1);
            }

            size_t size() const WI_NOEXCEPT
            {
                return m_size;
            }

            explicit operator bool() const WI_NOEXCEPT
            {
                return (m_pCopy != nullptr);
            }

            bool unique() const WI_NOEXCEPT
            {
                return ((m_pCopy != nullptr) && (*m_pCopy == 1));
            }

        private:
            long *m_pCopy;      // pointer to allocation: refcount + data
            size_t m_size;      // size of the data from m_pCopy

            void assign(_In_opt_ long *pCopy, size_t cbSize) WI_NOEXCEPT
            {
                reset();
                if (pCopy != nullptr)
                {
                    m_pCopy = pCopy;
                    m_size = cbSize;
                    ::InterlockedIncrementNoFence(m_pCopy);
                }
            }
        };

        inline shared_buffer make_shared_buffer_nothrow(_In_reads_bytes_opt_(countBytes) void *pData, size_t countBytes) WI_NOEXCEPT
        {
            shared_buffer buffer;
            buffer.create(pData, countBytes);
            return buffer;
        }

        inline shared_buffer make_shared_buffer_nothrow(size_t countBytes) WI_NOEXCEPT
        {
            shared_buffer buffer;
            buffer.create(countBytes);
            return buffer;
        }

        // A small mimic of the STL shared_ptr class, but unlike shared_ptr, a pointer is not attached to the class, but is
        // always simply contained within (it cannot be attached or detached).

        template <typename object_t>
        class shared_object
        {
        public:
            shared_object() WI_NOEXCEPT : m_pCopy(nullptr)
            {
            }

            shared_object(shared_object const &other) WI_NOEXCEPT :
                m_pCopy(other.m_pCopy)
            {
                    if (m_pCopy != nullptr)
                    {
                        ::InterlockedIncrementNoFence(&m_pCopy->m_refCount);
                    }
                }

            shared_object(shared_object &&other) WI_NOEXCEPT :
            m_pCopy(other.m_pCopy)
            {
                other.m_pCopy = nullptr;
            }

            ~shared_object() WI_NOEXCEPT
            {
                reset();
            }

            shared_object& operator=(shared_object const &other) WI_NOEXCEPT
            {
                reset();
                m_pCopy = other.m_pCopy;
                if (m_pCopy != nullptr)
                {
                    ::InterlockedIncrementNoFence(&m_pCopy->m_refCount);
                }
                return *this;
            }

            shared_object& operator=(shared_object &&other) WI_NOEXCEPT
            {
                reset();
                m_pCopy = other.m_pCopy;
                other.m_pCopy = nullptr;
                return *this;
            }

            void reset() WI_NOEXCEPT
            {
                if (m_pCopy != nullptr)
                {
                    if (0 == ::InterlockedDecrementRelease(&m_pCopy->m_refCount))
                    {
                        delete m_pCopy;
                    }
                    m_pCopy = nullptr;
                }
            }

            bool create()
            {
                RefAndObject *pObject = new(std::nothrow) RefAndObject();
                if (pObject == nullptr)
                {
                    return false;
                }
                reset();
                m_pCopy = pObject;
                return true;
            }

            template <typename param_t>
            bool create(param_t &&param1)
            {
                RefAndObject *pObject = new(std::nothrow) RefAndObject(wistd::forward<param_t>(param1));
                if (pObject == nullptr)
                {
                    return false;
                }
                reset();
                m_pCopy = pObject;
                return true;
            }

            object_t* get() const WI_NOEXCEPT
            {
                return (m_pCopy == nullptr) ? nullptr : &m_pCopy->m_object;
            }

            explicit operator bool() const WI_NOEXCEPT
            {
                return (m_pCopy != nullptr);
            }

            bool unique() const WI_NOEXCEPT
            {
                return ((m_pCopy != nullptr) && (m_pCopy->m_refCount == 1));
            }

            object_t *operator->() const WI_NOEXCEPT
            {
                return get();
            }

        private:
            struct RefAndObject
            {
                long m_refCount;
                object_t m_object;

                RefAndObject() :
                    m_refCount(1),
                    m_object()
                {
                }

                template <typename param_t>
                RefAndObject(param_t &&param1) :
                    m_refCount(1),
                    m_object(wistd::forward<param_t>(param1))
                {
                }
            };

            RefAndObject *m_pCopy;
        };

        // The following functions are basically the same, but are kept separated to:
        // 1) Provide a unique count and last error code per-type
        // 2) Avoid merging the types to allow easy debugging (breakpoints, conditional breakpoints based
        //      upon count of errors from a particular type, etc)

        __declspec(noinline) inline int RecordException(HRESULT hr) WI_NOEXCEPT
        {
            static HRESULT volatile s_hrErrorLast = S_OK;
            static long volatile s_cErrorCount = 0;
            s_hrErrorLast = hr;
            return ::InterlockedIncrementNoFence(&s_cErrorCount);
        }

        __declspec(noinline) inline int RecordReturn(HRESULT hr) WI_NOEXCEPT
        {
            static HRESULT volatile s_hrErrorLast = S_OK;
            static long volatile s_cErrorCount = 0;
            s_hrErrorLast = hr;
            return ::InterlockedIncrementNoFence(&s_cErrorCount);
        }

        __declspec(noinline) inline int RecordLog(HRESULT hr) WI_NOEXCEPT
        {
            static HRESULT volatile s_hrErrorLast = S_OK;
            static long volatile s_cErrorCount = 0;
            s_hrErrorLast = hr;
            return ::InterlockedIncrementNoFence(&s_cErrorCount);
        }

        __declspec(noinline) inline int RecordFailFast(HRESULT hr) WI_NOEXCEPT
        {
            static HRESULT volatile s_hrErrorLast = S_OK;
            s_hrErrorLast = hr;
            return 1;
        }

#if WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP | WINAPI_PARTITION_SYSTEM)
        inline bool __stdcall GetModuleInformation(_In_opt_ void* address, _Out_opt_ unsigned int* addressOffset, _Out_writes_bytes_opt_(size) char* name, size_t size) WI_NOEXCEPT
        {
            HMODULE hModule = nullptr;
            if (address && !GetModuleHandleExW(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS | GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT, reinterpret_cast<PCWSTR>(address), &hModule))
            {
                return false;
            }
            if (addressOffset)
            {
                *addressOffset = address ? static_cast<unsigned int>(static_cast<unsigned char*>(address) - reinterpret_cast<unsigned char *>(hModule)) : 0;
            }
            if (name)
            {
                char module[MAX_PATH];
                if (!GetModuleFileNameA(hModule, module, ARRAYSIZE(module)))
                {
                    return false;
                }

                PCSTR start = module + strlen(module);
                while ((start > module) && (*(start - 1) != '\\'))
                {
                    start--;
                }
                StringCchCopyA(name, size, start);
            }
            return true;
        }

        inline PCSTR __stdcall GetCurrentModuleName() WI_NOEXCEPT
        {
            static char s_szModule[64] = {};
            static volatile bool s_fModuleValid = false;
            if (!s_fModuleValid)    // Races are acceptable
            {
                GetModuleInformation(&RecordFailFast, nullptr, s_szModule, ARRAYSIZE(s_szModule));
                s_fModuleValid = true;
            }
            return s_szModule;
        }

#ifndef WIL_SUPPRESS_PRIVATE_API_USE
        inline HMODULE GetNTDLLModuleHandle() WI_NOEXCEPT
        {
            static HMODULE s_hmod = 0;
            if (s_hmod == 0)
            {
                s_hmod = LoadLibraryExW(L"ntdll.dll", NULL, LOAD_LIBRARY_SEARCH_SYSTEM32);
            }
            return s_hmod;
        }

        inline void __stdcall FailFastInLoaderCallout() WI_NOEXCEPT
        {
            // GetProcAddress is used here since we may be linking against an ntdll that doesn't contain this function
            // e.g. if this header is used on an OS older than Threshold (which may be the case for IE)
            auto pfn = reinterpret_cast<decltype(&::LdrFastFailInLoaderCallout)>(GetProcAddress(GetNTDLLModuleHandle(), "LdrFastFailInLoaderCallout"));
            if (pfn != nullptr)
            {
                pfn();  // don't do anything non-trivial from DllMain, fail fast.
            }
        }

        inline ULONG __stdcall RtlNtStatusToDosErrorNoTeb(_In_ NTSTATUS status) WI_NOEXCEPT
        {
            static decltype(RtlNtStatusToDosErrorNoTeb) *s_pfnRtlNtStatusToDosErrorNoTeb = nullptr;
            if (s_pfnRtlNtStatusToDosErrorNoTeb == nullptr)
            {
                s_pfnRtlNtStatusToDosErrorNoTeb = reinterpret_cast<decltype(RtlNtStatusToDosErrorNoTeb)*>(GetProcAddress(GetNTDLLModuleHandle(), "RtlNtStatusToDosErrorNoTeb"));
            }
            return s_pfnRtlNtStatusToDosErrorNoTeb ? s_pfnRtlNtStatusToDosErrorNoTeb(status) : 0;
        }

        inline BOOLEAN __stdcall RtlDllShutdownInProgress() WI_NOEXCEPT
        {
            static decltype(RtlDllShutdownInProgress) *s_pfnRtlDllShutdownInProgress = nullptr;
            if (s_pfnRtlDllShutdownInProgress == nullptr)
            {
                s_pfnRtlDllShutdownInProgress = reinterpret_cast<decltype(RtlDllShutdownInProgress)*>(GetProcAddress(GetNTDLLModuleHandle(), "RtlDllShutdownInProgress"));
            }
            return s_pfnRtlDllShutdownInProgress ? s_pfnRtlDllShutdownInProgress() : FALSE;
        }
#endif
#endif  // WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP | WINAPI_PARTITION_SYSTEM)

        inline bool __stdcall GetModuleInformationFromAddress(_In_opt_ void* address, _Out_opt_ unsigned int* addressOffset, _Out_writes_bytes_opt_(size) char* buffer, size_t size) WI_NOEXCEPT
        {
            if (size > 0)
            {
                *buffer = '\0';
            }
            if (addressOffset)
            {
                *addressOffset = 0;
            }
            if (g_pfnGetModuleInformation)
            {
                return g_pfnGetModuleInformation(address, addressOffset, buffer, size);
            }
            return false;
        }

        __declspec(noinline) inline HRESULT NtStatusToHr(NTSTATUS status) WI_NOEXCEPT
        {
            // The following conversions are the only known incorrect mappings in RtlNtStatusToDosErrorNoTeb
            if (SUCCEEDED_NTSTATUS(status))
            {
                // All successful status codes have only one hresult equivalent, S_OK
                return S_OK;
            }
            if (status == STATUS_NO_MEMORY)
            {
                // RtlNtStatusToDosErrorNoTeb maps STATUS_NO_MEMORY to the less popular of two Win32 no memory error codes resulting in an unexpected mapping
                return E_OUTOFMEMORY;
            }

            if (g_pfnRtlNtStatusToDosErrorNoTeb != nullptr)
            {
                DWORD err = g_pfnRtlNtStatusToDosErrorNoTeb(status);
                if (err != 0)
                {
                    return HRESULT_FROM_WIN32(err);
                }
            }

            return HRESULT_FROM_NT(status);
        }

        // The following set of functions all differ only based upon number of arguments.  They are unified in their handling
        // of data from each of the various error-handling types (fast fail, exceptions, etc.).

        inline DWORD GetLastErrorFail(__R_FN_PARAMS_FULL) WI_NOEXCEPT
        {
            __R_FN_UNREFERENCED;
            auto err = ::GetLastError();
            if (SUCCEEDED_WIN32(err))
            {
                // This function should only be called when GetLastError() is set to a FAILURE.
                // If you hit this assert (or are reviewing this failure telemetry), then there are one of three issues:
                //  1) Your code is using a macro (such as RETURN_IF_WIN32_BOOL_FALSE()) on a function that does not actually
                //      set the last error (consult MSDN).
                //  2) Your macro check against the error is not immediately after the API call.  Pushing it later can result
                //      in another API call between the previous one and the check resetting the last error.
                //  3) The API you're calling has a bug in it and does not accurately set the last error (there are a few
                //      examples here, such as SendMessageTimeout() that don't accurately set the last error).  For these,
                //      please send mail to 'wildisc' when found and work-around with win32errorhelpers.

                WI_USAGE_ERROR_FORWARD("CALLER BUG: Macro usage error detected.  GetLastError() does not have an error.");
                return ERROR_ASSERTION_FAILURE;
            }
            return err;
        }

        inline HRESULT GetLastErrorFailHr(__R_FN_PARAMS_FULL) WI_NOEXCEPT
        {
            return HRESULT_FROM_WIN32(GetLastErrorFail(__R_FN_CALL_FULL));
        }

        inline __declspec(noinline) HRESULT GetLastErrorFailHr() WI_NOEXCEPT
        {
            __R_FN_LOCALS_FULL_RA;
            return GetLastErrorFailHr(__R_FN_CALL_FULL);
        }

        inline void PrintLoggingMessage(_Out_writes_(cchDest) _Post_z_ PWSTR pszDest, _Pre_satisfies_(cchDest > 0) size_t cchDest, _In_opt_ _Printf_format_string_ PCSTR formatString, _In_opt_ va_list argList) WI_NOEXCEPT
        {
            if (formatString == nullptr)
            {
                pszDest[0] = L'\0';
            }
            else
            {
                wchar_t szFormatWide[2048];
                StringCchPrintfW(szFormatWide, ARRAYSIZE(szFormatWide), L"%hs", formatString);
                StringCchVPrintfW(pszDest, cchDest, szFormatWide, argList);
            }
        }

#pragma warning(push)
#pragma warning(disable:__WARNING_RETURNING_BAD_RESULT)
        // NOTE: The following two functions are unfortunate copies of strsafe.h functions that have been copied to reduce the friction associated with using
        // Result.h and ResultException.h in a build that does not have WINAPI_PARTITION_DESKTOP defined (where these are conditionally enabled).

        STRSAFEWORKERAPI WilStringLengthWorkerA(_In_reads_or_z_(cchMax) STRSAFE_PCNZCH psz, _In_ _In_range_(<= , STRSAFE_MAX_CCH) size_t cchMax, _Out_opt_ _Deref_out_range_(< , cchMax) _Deref_out_range_(<= , _String_length_(psz)) size_t* pcchLength)
        {
            HRESULT hr = S_OK;
            size_t cchOriginalMax = cchMax;
            while (cchMax && (*psz != '\0'))
            {
                psz++;
                cchMax--;
            }
            if (cchMax == 0)
            {
                // the string is longer than cchMax
                hr = STRSAFE_E_INVALID_PARAMETER;
            }
            if (pcchLength)
            {
                if (SUCCEEDED(hr))
                {
                    *pcchLength = cchOriginalMax - cchMax;
                }
                else
                {
                    *pcchLength = 0;
                }
            }
            return hr;
        }

        _Must_inspect_result_ STRSAFEAPI StringCchLengthA(_In_reads_or_z_(cchMax) STRSAFE_PCNZCH psz, _In_ _In_range_(1, STRSAFE_MAX_CCH) size_t cchMax, _Out_opt_ _Deref_out_range_(<, cchMax) _Deref_out_range_(<= , _String_length_(psz)) size_t* pcchLength)
        {
            HRESULT hr;
            if ((psz == NULL) || (cchMax > STRSAFE_MAX_CCH))
            {
                hr = STRSAFE_E_INVALID_PARAMETER;
            }
            else
            {
                hr = WilStringLengthWorkerA(psz, cchMax, pcchLength);
            }
            if (FAILED(hr) && pcchLength)
            {
                *pcchLength = 0;
            }
            return hr;
        }
#pragma warning(pop)

        #pragma warning(push)
        #pragma warning(disable : 4100) // Unused parameter (pszDest)
        _Post_satisfies_(cchDest > 0 && cchDest <= cchMax) STRSAFEWORKERAPI WilStringValidateDestA(_In_reads_opt_(cchDest) STRSAFE_PCNZCH pszDest, _In_ size_t cchDest, _In_ const size_t cchMax)
        {
            HRESULT hr = S_OK;
            if ((cchDest == 0) || (cchDest > cchMax))
            {
                hr = STRSAFE_E_INVALID_PARAMETER;
            }
            return hr;
        }
        #pragma warning(pop)

        STRSAFEWORKERAPI WilStringVPrintfWorkerA(_Out_writes_(cchDest) _Always_(_Post_z_) STRSAFE_LPSTR pszDest, _In_ _In_range_(1, STRSAFE_MAX_CCH) size_t cchDest, _Always_(_Out_opt_ _Deref_out_range_(<=, cchDest - 1)) size_t* pcchNewDestLength, _In_ _Printf_format_string_ STRSAFE_LPCSTR pszFormat, _In_ va_list argList)
        {
            HRESULT hr = S_OK;
            int iRet;
            size_t cchMax;
            size_t cchNewDestLength = 0;

            // leave the last space for the null terminator
            cchMax = cchDest - 1;
#undef STRSAFE_USE_SECURE_CRT
#define STRSAFE_USE_SECURE_CRT 1
        #if (STRSAFE_USE_SECURE_CRT == 1) && !defined(STRSAFE_LIB_IMPL)
            iRet = _vsnprintf_s(pszDest, cchDest, cchMax, pszFormat, argList);
        #else
        #pragma warning(push)
        #pragma warning(disable: __WARNING_BANNED_API_USAGE)// "STRSAFE not included"
            iRet = _vsnprintf(pszDest, cchMax, pszFormat, argList);
        #pragma warning(pop)
        #endif
            // ASSERT((iRet < 0) || (((size_t)iRet) <= cchMax));

            if ((iRet < 0) || (((size_t)iRet) > cchMax))
            {
                // need to null terminate the string
                pszDest += cchMax;
                *pszDest = '\0';

                cchNewDestLength = cchMax;

                // we have truncated pszDest
                hr = STRSAFE_E_INSUFFICIENT_BUFFER;
            }
            else if (((size_t)iRet) == cchMax)
            {
                // need to null terminate the string
                pszDest += cchMax;
                *pszDest = '\0';

                cchNewDestLength = cchMax;
            }
            else
            {
                cchNewDestLength = (size_t)iRet;
            }

            if (pcchNewDestLength)
            {
                *pcchNewDestLength = cchNewDestLength;
            }

            return hr;
        }

        STRSAFEAPI StringCchPrintfA( _Out_writes_(cchDest) _Always_(_Post_z_) STRSAFE_LPSTR pszDest, _In_ size_t cchDest, _In_ _Printf_format_string_ STRSAFE_LPCSTR pszFormat, ...)
        {
            HRESULT hr;
            hr = wil::details::WilStringValidateDestA(pszDest, cchDest, STRSAFE_MAX_CCH);
            if (SUCCEEDED(hr))
            {
                va_list argList;
                va_start(argList, pszFormat);
                hr = wil::details::WilStringVPrintfWorkerA(pszDest, cchDest, NULL, pszFormat, argList);
                va_end(argList);
            }
            else if (cchDest > 0)
            {
                *pszDest = '\0';
            }
            return hr;
        }

        _Ret_range_(sizeof(char), (psz == nullptr) ? sizeof(char) : (_String_length_(psz) + sizeof(char)))
        inline size_t ResultStringSize(_In_opt_ PCSTR psz)
            { return (psz == nullptr) ? sizeof(char) : (strlen(psz) + sizeof(char)); }

        _Ret_range_(sizeof(wchar_t), (psz == nullptr) ? sizeof(wchar_t) : ((_String_length_(psz) + 1) * sizeof(wchar_t)))
        inline size_t ResultStringSize(_In_opt_ PCWSTR psz) 
            { return (psz == nullptr) ? sizeof(wchar_t) : (wcslen(psz) + 1) * sizeof(wchar_t); }

        template<typename TString>
        _Ret_range_(pStart, pEnd) inline unsigned char *WriteResultString(_Out_writes_to_ptr_opt_(pEnd) unsigned char *pStart, _In_opt_ unsigned char *pEnd, _In_opt_ TString pszString, _Out_opt_ TString *ppszBufferString = nullptr)
        {
            decltype(pszString[0]) const cZero = 0;
            TString const pszLocal = (pszString == nullptr) ? &cZero : pszString;
            size_t const cchWant = ResultStringSize(pszLocal) / sizeof(cZero);
            size_t const cchMax = static_cast<size_t>(pEnd - pStart) / sizeof(cZero);
            size_t const cchCopy = (cchWant < cchMax) ? cchWant : cchMax;
            memcpy_s(pStart, cchMax * sizeof(cZero), pszLocal, cchCopy * sizeof(cZero));
            wil::AssignToOptParam(ppszBufferString, (cchCopy > 1) ? reinterpret_cast<TString>(pStart) : nullptr);
            if ((cchCopy < cchWant) && (cchCopy > 0))
            {
                auto pZero = pStart + ((cchCopy - 1) * sizeof(cZero));
                ::ZeroMemory(pZero, sizeof(cZero));
            }
            return (pStart + (cchCopy * sizeof(cZero)));
        }

        _Ret_range_(0, (cchMax > 0) ? cchMax - 1 : 0) inline size_t UntrustedStringLength(_In_ PCSTR psz, _In_ size_t cchMax)    { size_t cbLength; return SUCCEEDED(wil::details::StringCchLengthA(psz, cchMax, &cbLength)) ? cbLength : 0; }
        _Ret_range_(0, (cchMax > 0) ? cchMax - 1 : 0) inline size_t UntrustedStringLength(_In_ PCWSTR psz, _In_ size_t cchMax)   { size_t cbLength; return SUCCEEDED(::StringCchLengthW(psz, cchMax, &cbLength)) ? cbLength : 0; }

        template<typename TString>
        _Ret_range_(pStart, pEnd) inline unsigned char *GetResultString(_In_reads_to_ptr_opt_(pEnd) unsigned char *pStart, _In_opt_ unsigned char *pEnd, _Out_ TString *ppszBufferString)
        {
            size_t cchLen = UntrustedStringLength(reinterpret_cast<TString>(pStart), (pEnd - pStart) / sizeof((*ppszBufferString)[0]));
            *ppszBufferString = (cchLen > 0) ? reinterpret_cast<TString>(pStart) : nullptr;
            auto pReturn = min(pEnd, pStart + ((cchLen + 1) * sizeof((*ppszBufferString)[0])));
            __analysis_assume((pReturn >= pStart) && (pReturn <= pEnd));
            return pReturn;
        }


        template <typename TLambda>
        class ScopeExitFn
        {
        public:
            explicit ScopeExitFn(TLambda&& exitFunctor) WI_NOEXCEPT :
                m_exitFunctor(wistd::move(exitFunctor)),
                m_shouldRun(true)
            {
                static_assert(!wistd::is_lvalue_reference<TLambda>::value && !wistd::is_rvalue_reference<TLambda>::value,
                              "ScopeExit should only be directly used with lambdas");
            }

            ScopeExitFn(ScopeExitFn&& other) WI_NOEXCEPT :
                m_exitFunctor(wistd::move(other.m_exitFunctor)),
                m_shouldRun(other.m_shouldRun)
            {
                other.m_shouldRun = false;
            }

            ~ScopeExitFn() WI_NOEXCEPT
            {
                operator()();
            }

            void operator()() WI_NOEXCEPT
            {
                if (m_shouldRun)
                {
                    m_shouldRun = false;
                    m_exitFunctor();
                }
            }

            void Dismiss() WI_NOEXCEPT
            {
                m_shouldRun = false;
            }

            ScopeExitFn(ScopeExitFn const &) = delete;
            ScopeExitFn & operator=(ScopeExitFn const &) = delete;

        private:
            TLambda m_exitFunctor;
            bool m_shouldRun;
        };

#ifdef WIL_ENABLE_EXCEPTIONS
        enum class GuardType : unsigned char
        {
            Ignore,
            Log,
            FailFast
        };

        template <typename TLambda>
        class ScopeExitFnGuard
        {
        public:
            explicit ScopeExitFnGuard(GuardType type, TLambda&& exitFunctor) WI_NOEXCEPT :
                m_exitFunctor(wistd::move(exitFunctor)),
                m_type(type)
            {
                static_assert(!wistd::is_lvalue_reference<TLambda>::value && !wistd::is_rvalue_reference<TLambda>::value,
                              "ScopeExit should only be directly used with lambdas");
            }

            ScopeExitFnGuard(ScopeExitFnGuard&& other) WI_NOEXCEPT :
                m_exitFunctor(wistd::move(other.m_exitFunctor)),
                m_type(other.m_type),
                m_shouldRun(other.m_shouldRun)
            {
                other.m_shouldRun = false;
            }

            ~ScopeExitFnGuard() WI_NOEXCEPT
            {
                operator()();
            }

            void operator()() WI_NOEXCEPT
            {
                try
                {
                    RunWithoutGuard();
                }
                catch (...)
                {
                    if (m_type == GuardType::FailFast)
                    {
                        __WIL_PRIVATE_FAIL_FAST_HR(wil::ResultFromCaughtException());
                    }
                    else if (m_type == GuardType::Log)
                    {
                        __WIL_PRIVATE_LOG_HR(wil::ResultFromCaughtException());
                    }
                    // GuardType::Ignore = no-op
                }
            }

            void RunWithoutGuard()
            {
                if (m_shouldRun)
                {
                    m_shouldRun = false;
                    m_exitFunctor();
                }
            }

            void Dismiss() WI_NOEXCEPT
            {
                m_shouldRun = false;
            }

            ScopeExitFnGuard(const ScopeExitFnGuard&) = delete;
            ScopeExitFnGuard& operator=(const ScopeExitFnGuard&) = delete;

        private:
            TLambda m_exitFunctor;
            GuardType m_type;
            bool m_shouldRun = true;
        };
#endif // WIL_ENABLE_EXCEPTIONS
    } // details namespace
    /// @endcond

    //*****************************************************************************
    // WIL result handling initializers
    //
    // Generally, callers do not need to manually initialize WIL. This header creates
    // the appropriate .CRT init section pieces through global objects to ensure that
    // WilInitialize... is called before DllMain or main().
    //
    // Certain binaries do not link with the CRT or do not support .CRT-section based
    // initializers. Those binaries must link only with other static libraries that
    // also set RESULT_SUPPRESS_STATIC_INITIALIZERS to ensure no .CRT inits are left,
    // and they should call one of the WilInitialize_ResultMacros_??? methods during
    // their initialization phase.  Skipping this initialization path is OK as well,
    // but results in a slightly degraded experience with result reporting.
    //
    // Calling WilInitialize_ResultMacros_DesktopOrSystem_SuppressPrivateApiUse provides:
    // - The name of the current module in wil::FailureInfo::pszModule
    // - The name of the returning-to module during wil\staging.h failures
    //
    // Calling WilInitialize_ResultMacros_DesktopOrSystem additionally provides:
    // - FAIL_FAST_IMMEDIATE_IF_IN_LOADER_CALLOUT enforcement
    // - Higher fidelity mapping of NTSTATUS->HRESULT for RETURN_IF_NTSTATUS*
    // - wil::ProcessShutdownInProgress returns 'true' during process shutdown (false when not called or set)
    //*****************************************************************************

#if WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP | WINAPI_PARTITION_SYSTEM)
    //! Call this method to initialize WIL manually in a module where RESULT_SUPPRESS_STATIC_INITIALIZERS is required. WIL will 
    //! only use publicly documented APIs.
    inline void WilInitialize_ResultMacros_DesktopOrSystem_SuppressPrivateApiUse()
    {
        details::g_pfnGetModuleName        = details::GetCurrentModuleName;
        details::g_pfnGetModuleInformation = details::GetModuleInformation;
    }

#ifndef WIL_SUPPRESS_PRIVATE_API_USE
    //! Call this method to initialize WIL manually in a module where RESULT_SUPPRESS_STATIC_INITIALIZERS is required. WIL will
    //! use internal methods to provide additional diagnostic information & behavior.
    inline void WilInitialize_ResultMacros_DesktopOrSystem()
    {
        WilInitialize_ResultMacros_DesktopOrSystem_SuppressPrivateApiUse();
        details::g_pfnFailFastInLoaderCallout    = details::FailFastInLoaderCallout;
        details::g_pfnRtlNtStatusToDosErrorNoTeb = details::RtlNtStatusToDosErrorNoTeb;
        details::g_pfnRtlDllShutdownInProgress   = details::RtlDllShutdownInProgress;
    }
#endif // WIL_SUPPRESS_PRIVATE_API_USE

    /// @cond
    namespace details
    {
#ifndef RESULT_SUPPRESS_STATIC_INITIALIZERS
#ifdef WIL_SUPPRESS_PRIVATE_API_USE
        WI_HEADER_INITITALIZATION_FUNCTION(WilInitialize_ResultMacros_DesktopOrSystem_SuppressPrivateApiUse, []
        {
            ::wil::WilInitialize_ResultMacros_DesktopOrSystem_SuppressPrivateApiUse();
            return 1;
        });
#else
        WI_HEADER_INITITALIZATION_FUNCTION(WilInitialize_ResultMacros_DesktopOrSystem, []
        {
            ::wil::WilInitialize_ResultMacros_DesktopOrSystem();
            return 1;
        });
#endif // WIL_SUPPRESS_PRIVATE_API_USE
#endif // RESULT_SUPPRESS_STATIC_INITIALIZERS
    }
    /// @endcond
    
#endif // WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP | WINAPI_PARTITION_SYSTEM)

    //*****************************************************************************
    // Public Error Handling Helpers
    //*****************************************************************************

    //! Call this method to determine if process shutdown is in progress (allows avoiding work during dll unload).
    inline bool ProcessShutdownInProgress()
    {
        return (details::g_processShutdownInProgress || (details::g_pfnRtlDllShutdownInProgress ? details::g_pfnRtlDllShutdownInProgress() : false));
    }

    /** Use this object to wrap an object that wants to prevent its destructor from being run when the process is shutting down.
    Upon process shutdown a method (ProcessShutdown()) is called that must be implemented on the object, otherwise the destructor is
    called as is typical. */
    template<class T>
    class shutdown_aware_object
    {
    public:
        shutdown_aware_object()
        {
            void* var = &m_raw;
            ::new(var) T();
        }

        ~shutdown_aware_object()
        {
            if (ProcessShutdownInProgress())
            {
                get().ProcessShutdown();
            }
            else
            {
                (&get())->~T();
            }
        }

        //! Retrieves a reference to the contained object
        T& get() WI_NOEXCEPT
        {
            return *reinterpret_cast<T*>(&m_raw);
        }

    private:
        unsigned char m_raw[sizeof(T)];
    };

    /** Forward your DLLMain to this function so that WIL can have visibility into whether a DLL unload is because
    of termination or normal unload.  Note that when private API usage is enabled, WIL attempts to make this
    determination on its own without this callback.  Suppressing private APIs requires use of this. */
    inline void DLLMain(HINSTANCE, DWORD reason, _In_opt_ LPVOID reserved)
    {
        if (!details::g_processShutdownInProgress)
        {
            if ((reason == DLL_PROCESS_DETACH) && (reserved != nullptr))
            {
                details::g_processShutdownInProgress = true;
            }
        }
    }

    // [optionally] Plug in fallback telemetry reporting
    // Normally, the callback is owned by including ResultLogging.h in the including module.  Alternatively a module
    // could re-route fallback telemetry to any ONE specific provider by calling this method.
    inline void SetResultTelemetryFallback(_In_opt_ decltype(details::g_pfnTelemetryCallback) callbackFunction)
    {
        // Only ONE telemetry provider can own the fallback telemetry callback.
        __FAIL_FAST_IMMEDIATE_ASSERT__((details::g_pfnTelemetryCallback == nullptr) || (callbackFunction == nullptr));
        details::g_pfnTelemetryCallback = callbackFunction;
    }

    // [optionally] Plug in result logging (do not use for telemetry)
    // This provides the ability for a module to hook all failures flowing through the system for inspection
    // and/or logging.
    inline void SetResultLoggingCallback(_In_opt_ decltype(details::g_pfnLoggingCallback) callbackFunction)
    {
        // Only ONE function can own the result logging callback
        __FAIL_FAST_IMMEDIATE_ASSERT__((details::g_pfnLoggingCallback == nullptr) || (callbackFunction == nullptr));
        details::g_pfnLoggingCallback = callbackFunction;
    }

    // [optionally] Plug in custom result messages
    // There are some purposes that require translating the full information that is known about a failure
    // into a message to be logged (either through the console for debugging OR as the message attached
    // to a Platform::Exception^).  This callback allows a module to format the string itself away from the
    // default.
    inline void SetResultMessageCallback(_In_opt_ decltype(wil::g_pfnResultLoggingCallback) callbackFunction)
    {
        // Only ONE function can own the result message callback
        __FAIL_FAST_IMMEDIATE_ASSERT__((g_pfnResultLoggingCallback == nullptr) || (callbackFunction == nullptr));
        details::g_resultMessageCallbackSet = true;
        g_pfnResultLoggingCallback = callbackFunction;
    }

    // [optionally] Plug in exception remapping
    // A module can plug a callback in using this function to setup custom exception handling to allow any
    // exception type to be converted into an HRESULT from exception barriers.
    inline void SetResultFromCaughtExceptionCallback(_In_opt_ decltype(wil::g_pfnResultFromCaughtException) callbackFunction)
    {
        // Only ONE function can own the exception conversion
        __FAIL_FAST_IMMEDIATE_ASSERT__((g_pfnResultFromCaughtException == nullptr) || (callbackFunction == nullptr));
        g_pfnResultFromCaughtException = callbackFunction;
    }


    // A RAII wrapper around the storage of a FailureInfo struct (which is normally meant to be consumed
    // on the stack or from the caller).  The storage of FailureInfo needs to copy some data internally
    // for lifetime purposes.

    class StoredFailureInfo
    {
    public:
        StoredFailureInfo() WI_NOEXCEPT
        {
            ::ZeroMemory(&m_failureInfo, sizeof(m_failureInfo));
        }

        StoredFailureInfo(FailureInfo const &other) WI_NOEXCEPT
        {
            SetFailureInfo(other);
        }

        FailureInfo const & GetFailureInfo() const WI_NOEXCEPT
        {
            return m_failureInfo;
        }

        void SetFailureInfo(FailureInfo const &failure) WI_NOEXCEPT
        {
            m_failureInfo = failure;

            size_t const cbNeed = details::ResultStringSize(failure.pszMessage) +
                                  details::ResultStringSize(failure.pszCode) +
                                  details::ResultStringSize(failure.pszFunction) +
                                  details::ResultStringSize(failure.pszFile) +
                                  details::ResultStringSize(failure.pszCallContext) +
                                  details::ResultStringSize(failure.pszModule) +
                                  details::ResultStringSize(failure.callContextCurrent.contextName) +
                                  details::ResultStringSize(failure.callContextCurrent.contextMessage) +
                                  details::ResultStringSize(failure.callContextOriginating.contextName) +
                                  details::ResultStringSize(failure.callContextOriginating.contextMessage);

            if (!m_spStrings.unique() || (m_spStrings.size() < cbNeed))
            {
                m_spStrings.reset();
                m_spStrings.create(cbNeed);
            }

            size_t cbAlloc;
            unsigned char *pBuffer = static_cast<unsigned char *>(m_spStrings.get(&cbAlloc));
            unsigned char *pBufferEnd = (pBuffer != nullptr) ? pBuffer + cbAlloc : nullptr;

            pBuffer = details::WriteResultString(pBuffer, pBufferEnd, failure.pszMessage, &m_failureInfo.pszMessage);
            pBuffer = details::WriteResultString(pBuffer, pBufferEnd, failure.pszCode, &m_failureInfo.pszCode);
            pBuffer = details::WriteResultString(pBuffer, pBufferEnd, failure.pszFunction, &m_failureInfo.pszFunction);
            pBuffer = details::WriteResultString(pBuffer, pBufferEnd, failure.pszFile, &m_failureInfo.pszFile);
            pBuffer = details::WriteResultString(pBuffer, pBufferEnd, failure.pszCallContext, &m_failureInfo.pszCallContext);
            pBuffer = details::WriteResultString(pBuffer, pBufferEnd, failure.pszModule, &m_failureInfo.pszModule);
            pBuffer = details::WriteResultString(pBuffer, pBufferEnd, failure.callContextCurrent.contextName, &m_failureInfo.callContextCurrent.contextName);
            pBuffer = details::WriteResultString(pBuffer, pBufferEnd, failure.callContextCurrent.contextMessage, &m_failureInfo.callContextCurrent.contextMessage);
            pBuffer = details::WriteResultString(pBuffer, pBufferEnd, failure.callContextOriginating.contextName, &m_failureInfo.callContextOriginating.contextName);
            details::WriteResultString(pBuffer, pBufferEnd, failure.callContextOriginating.contextMessage, &m_failureInfo.callContextOriginating.contextMessage);
        }

        // Relies upon generated copy constructor and assignment operator

    protected:
        FailureInfo m_failureInfo;
        details::shared_buffer m_spStrings;
    };

#if defined(WIL_ENABLE_EXCEPTIONS) || defined(WIL_FORCE_INCLUDE_RESULT_EXCEPTION)

    //! This is WIL's default exception class thrown from all THROW_XXX macros (outside of c++/cx).
    //! This class stores all of the FailureInfo context that is available when the exception is thrown.  It's also caught by
    //! exception guards for automatic conversion to HRESULT.
    //!
    //! In c++/cx, Platform::Exception^ is used instead of this class (unless @ref wil::g_fResultThrowPlatformException has been changed).
    class ResultException : public std::exception
    {
    public:
        //! Constructs a new ResultException from an existing FailureInfo.
        ResultException(const FailureInfo& failure) WI_NOEXCEPT :
            m_failure(failure)
        {
        }

        //! Constructs a new exception type from a given HRESULT (use only for constructing custom exception types).
        ResultException(_Pre_satisfies_(hr < 0) HRESULT hr) WI_NOEXCEPT :
            m_failure(CustomExceptionFailureInfo(hr))
        {
        }

        //! Returns the failed HRESULT that this exception represents.
        _Always_(_Post_satisfies_(return < 0)) HRESULT GetErrorCode() const WI_NOEXCEPT
        {
            HRESULT const hr = m_failure.GetFailureInfo().hr;
            __analysis_assume(hr < 0);
            return hr;
        }

        //! Get a reference to the stored FailureInfo.
        FailureInfo const & GetFailureInfo() const WI_NOEXCEPT
        {
            return m_failure.GetFailureInfo();
        }

        //! Sets the stored FailureInfo (use primarily only when constructing custom exception types).
        void SetFailureInfo(FailureInfo const &failure) WI_NOEXCEPT
        {
            m_failure.SetFailureInfo(failure);
        }

        //! Provides a string representing the FailureInfo from this exception.
        inline const char * __CLR_OR_THIS_CALL what() const override
        {
            if (!m_what)
            {
                wchar_t message[2048];
                GetFailureLogString(message, ARRAYSIZE(message), m_failure.GetFailureInfo());
                
                char messageA[1024];
                wil::details::StringCchPrintfA(messageA, ARRAYSIZE(messageA), "%ws", message);
                m_what.create(messageA, strlen(messageA) + sizeof(*messageA));
            }
            return static_cast<const char *>(m_what.get());
        }

        // Relies upon auto-generated copy constructor and assignment operator
    protected:
        StoredFailureInfo m_failure;                //!< The failure information for this exception
        mutable details::shared_buffer m_what;      //!< The on-demand generated what() string

        //! Use to produce a custom FailureInfo from an HRESULT (use only when constructing custom exception types).
        static FailureInfo CustomExceptionFailureInfo(HRESULT hr) WI_NOEXCEPT
        {
            FailureInfo fi = {};
            fi.type = FailureType::Exception;
            fi.hr = hr;
            return fi;
        }
    };
#endif

    // The ScopeExit function accepts a lambda and returns an object meant to be captured in
    // an auto variable.  That lambda will be run when the object goes out of scope.  For most
    // begin/end pairings RAII techniques should be used instead, but occasionally there are
    // uncommon cases where RAII techniques are impractical (ensuring a file is deleted after
    // a test routine, etc).  Usage involves:
    //
    //      FunctionNeedingCleanup();       // start with a call to establish some state that requires cleanup
    //      auto fnCleanup = ScopeExit([&]
    //      {
    //          // write cleanup code here
    //      });
    //
    // When fnCleanup goes out of scope it will run its lambda.  Optionally, if the cleanup is
    // no long needed, it can be dismissed (so that it is not run):
    //
    //      fnCleanup.Dismiss();
    //
    // If the cleanup is needed ahead of going out of scope it can be explicitly run:
    //
    //      fnCleanup();
    //
    // Rules:
    //      * Do not call code in the lambda that may throw (use one of the below variants for that)
    //      * Always capture by '&' so that exception-based code cannot throw as a byproduct of construction.
    //      * Prefer RAII constructs over using this (never use this to free a handle, for example, use an RAII class)

    template <typename TLambda>
    inline wil::details::ScopeExitFn<TLambda> ScopeExit(TLambda&& exitFunctor) WI_NOEXCEPT
    {
        return wil::details::ScopeExitFn<TLambda>(wistd::forward<TLambda>(exitFunctor));
    }


    //*****************************************************************************
    // Public Helpers that catch -- mostly only enabled when exceptions are enabled
    //*****************************************************************************

    // ResultFromCaughtException is a function that is meant to be called from within a catch(...) block.  Internally
    // it re-throws and catches the exception to convert it to an HRESULT.  If an exception is of an unrecognized type
    // the function will fail fast.
    //
    // try
    // {
    //     // Code
    // }
    // catch (...)
    // {
    //     hr = wil::ResultFromCaughtException();
    // }
    _Always_(_Post_satisfies_(return < 0))
    __declspec(noinline) inline HRESULT ResultFromCaughtException() WI_NOEXCEPT
    {
        bool isNormalized = false;
        HRESULT hr = S_OK;
        if (details::g_pfnResultFromCaughtExceptionInternal)
        {
            hr = details::g_pfnResultFromCaughtExceptionInternal(nullptr, 0, &isNormalized);
        }
        if (FAILED(hr))
        {
            return hr;
        }

        // Caller bug: an unknown exception was thrown
        __WIL_PRIVATE_FAIL_FAST_HR_IF(HRESULT_FROM_WIN32(ERROR_UNHANDLED_EXCEPTION), g_fResultFailFastUnknownExceptions);
        return HRESULT_FROM_WIN32(ERROR_UNHANDLED_EXCEPTION);
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    // The ScopeExitXXX functions work identically to ScopeExit but add an exception guard for the lambda.  The different
    // variants control what happens if an exception were actually thrown from the cleanup lambda.

    // The ScopeExitNoExcept version should eventually be removed in favor of the simpler ScopeExit() once noexcept is
    // fully implemented to provide fail fast handling of exceptions from the regular scope exit class.
    template <typename TLambda>
    inline wil::details::ScopeExitFnGuard<TLambda> ScopeExitNoExcept(TLambda&& exitFunctor) WI_NOEXCEPT
    {
        return wil::details::ScopeExitFnGuard<TLambda>(wil::details::GuardType::FailFast, wistd::forward<TLambda>(exitFunctor));
    }

    template <typename TLambda>
    inline wil::details::ScopeExitFnGuard<TLambda> ScopeExitLog(TLambda&& exitFunctor) WI_NOEXCEPT
    {
        return wil::details::ScopeExitFnGuard<TLambda>(wil::details::GuardType::Log, wistd::forward<TLambda>(exitFunctor));
    }

    template <typename TLambda>
    inline wil::details::ScopeExitFnGuard<TLambda> ScopeExitIgnore(TLambda&& exitFunctor) WI_NOEXCEPT
    {
        return wil::details::ScopeExitFnGuard<TLambda>(wil::details::GuardType::Ignore, wistd::forward<TLambda>(exitFunctor));
    }


    //! A lambda-based exception guard that can vary the supported exception types.
    //! This function accepts a lambda and diagnostics information as its parameters and executes that lambda
    //! under a try/catch(...) block.  All exceptions are caught and the function reports the exception information
    //! and diagnostics to telemetry on failure.  An HRESULT is returned that maps to the exception.
    //!
    //! Note that an overload exists that does not report failures to telemetry at all.  This version should be preferred
    //! to that version.  Also note that neither of these versions are preferred over using try catch blocks to accomplish
    //! the same thing as they will be more efficient.
    //!
    //! See @ref page_exception_guards for more information and examples on exception guards.
    //! ~~~~
    //! return wil::ResultFromException(WI_DIAGNOSTICS_INFO, [&]
    //! {
    //!     // exception-based code
    //!     // telemetry is reported with full exception information
    //! });
    //! ~~~~
    //! @param diagnostics  Always pass WI_DIAGNOSTICS_INFO as the first parameter
    //! @param supported    What kind of exceptions you want to support
    //! @param functor      A lambda that accepts no parameters; any return value is ignored
    //! @return             S_OK on success (no exception thrown) or an error based upon the exception thrown 
    template <typename Functor>
    __forceinline HRESULT ResultFromException(const DiagnosticsInfo& diagnostics, SupportedExceptions supported, Functor&& functor) WI_NOEXCEPT
    {
        static_assert(details::functor_tag<ErrorReturn::None, Functor>::value != details::tag_return_other::value, "Functor must return void or HRESULT");
        details::functor_tag<ErrorReturn::None, Functor>::functor_wrapper<Functor> functorObject(wistd::forward<Functor>(functor));

        return wil::details::ResultFromException(diagnostics, supported, functorObject);
    }

    //! A lambda-based exception guard.
    //! This overload uses SupportedExceptions::Known by default.  See @ref ResultFromException for more detailed information.
    template <typename Functor>
    __forceinline HRESULT ResultFromException(const DiagnosticsInfo& diagnostics, Functor&& functor) WI_NOEXCEPT
    {
        return ResultFromException(diagnostics, SupportedExceptions::Known, wistd::forward<Functor>(functor));
    }

    //! A lambda-based exception guard that does not report failures to telemetry.
    //! This function accepts a lambda as it's only parameter and executes that lambda under a try/catch(...) block.
    //! All exceptions are caught and the function returns an HRESULT mapping to the exception.
    //!
    //! This version (taking only a lambda) does not report failures to telemetry.  An overload with the same name
    //! can be utilized by passing `WI_DIAGNOSTICS_INFO` as the first parameter and the lambda as the second parameter
    //! to report failure information to telemetry.
    //!
    //! See @ref page_exception_guards for more information and examples on exception guards.
    //! ~~~~
    //! hr = wil::ResultFromException([&]
    //! {
    //!     // exception-based code
    //!     // the conversion of exception to HRESULT doesn't report telemetry
    //! });
    //! 
    //! hr = wil::ResultFromException(WI_DIAGNOSTICS_INFO, [&]
    //! {
    //!     // exception-based code
    //!     // telemetry is reported with full exception information
    //! });
    //! ~~~~
    //! @param functor  A lambda that accepts no parameters; any return value is ignored
    //! @return         S_OK on success (no exception thrown) or an error based upon the exception thrown 
    template <typename Functor>
    inline HRESULT ResultFromException(Functor&& functor) WI_NOEXCEPT try
    {
        static_assert(details::functor_tag<ErrorReturn::None, Functor>::value == details::tag_return_void::value, "Functor must return void");
        details::functor_tag<ErrorReturn::None, Functor>::functor_wrapper<Functor> functorObject(wistd::forward<Functor>(functor));

        functorObject.Run();
        return S_OK;
    }
    catch (...)
    {
        return ResultFromCaughtException();
    }


    //! A lambda-based exception guard that can identify the origin of unknown exceptions and can vary the supported exception types.
    //! Functionally this is nearly identical to the corresponding @ref ResultFromException function with the exception
    //! that it utilizes structured exception handling internally to be able to terminate at the point where a unknown
    //! exception is thrown, rather than after that unknown exception has been unwound.  Though less efficient, this leads
    //! to a better debugging experience when analyzing unknown exceptions.
    //!
    //! For example:
    //! ~~~~
    //! hr = wil::ResultFromExceptionDebug(WI_DIAGNOSTICS_INFO, [&]
    //! {
    //!     FunctionWhichMayThrow();
    //! });
    //! ~~~~
    //! Assume FunctionWhichMayThrow() has a bug in it where it accidentally does a `throw E_INVALIDARG;`.  This ends up
    //! throwing a `long` as an exception object which is not what the caller intended.  The normal @ref ResultFromException
    //! would fail-fast when this is encountered, but it would do so AFTER FunctionWhichMayThrow() is already off of the
    //! stack and has been unwound.  Because SEH is used for ResultFromExceptionDebug, the fail-fast occurs with everything
    //! leading up to and including the `throw INVALIDARG;` still on the stack (and easily debuggable).
    //!
    //! The penalty paid for using this, however, is efficiency.  It's far less efficient as a general pattern than either
    //! using ResultFromException directly or especially using try with CATCH_ macros directly.  Still it's helpful to deploy
    //! selectively to isolate issues a component may be having with unknown/unhandled exceptions.
    //!
    //! The ability to vary the SupportedExceptions that this routine provides adds the ability to track down unexpected
    //! exceptions not falling into the supported category easily through fail-fast.  For example, by not supporting any
    //! exception, you can use this function to quickly add an exception guard that will fail-fast any exception at the point
    //! the exception occurs (the throw) in a codepath where the origination of unknown exceptions need to be tracked down.
    //!
    //! Also see @ref ResultFromExceptionDebugNoStdException.  It functions almost identically, but also will fail-fast and stop
    //! on std::exception based exceptions (but not Platform::Exception^ or wil::ResultException).  Using this can help isolate
    //! where an unexpected exception is being generated from.
    //! @param diagnostics  Always pass WI_DIAGNOSTICS_INFO as the first parameter
    //! @param supported    What kind of exceptions you want to support
    //! @param functor      A lambda that accepts no parameters; any return value is ignored
    //! @return             S_OK on success (no exception thrown) or an error based upon the exception thrown 
    template <typename Functor>
    __forceinline HRESULT ResultFromExceptionDebug(const DiagnosticsInfo& diagnostics, SupportedExceptions supported, Functor&& functor) WI_NOEXCEPT
    {
        static_assert(details::functor_tag<ErrorReturn::None, Functor>::value == details::tag_return_void::value, "Functor must return void");
        details::functor_tag<ErrorReturn::None, Functor>::functor_wrapper<Functor> functorObject(wistd::forward<Functor>(functor));

        return wil::details::ResultFromExceptionDebug(diagnostics, supported, functorObject);
    }

    //! A lambda-based exception guard that can identify the origin of unknown exceptions.
    //! This overload uses SupportedExceptions::Known by default.  See @ref ResultFromExceptionDebug for more detailed information.
    template <typename Functor>
    __forceinline HRESULT ResultFromExceptionDebug(const DiagnosticsInfo& diagnostics, Functor&& functor) WI_NOEXCEPT
    {
        static_assert(details::functor_tag<ErrorReturn::None, Functor>::value == details::tag_return_void::value, "Functor must return void");
        details::functor_tag<ErrorReturn::None, Functor>::functor_wrapper<Functor> functorObject(wistd::forward<Functor>(functor));

        return wil::details::ResultFromExceptionDebug(diagnostics, SupportedExceptions::Known, functorObject);
    }

    //! A fail-fast based exception guard.
    //! Technically this is an overload of @ref ResultFromExceptionDebug that uses SupportedExceptions::None by default.  Any uncaught
    //! exception that makes it back to this guard would result in a fail-fast at the point the exception is thrown.
    template <typename Functor>
    __forceinline void FailFastException(const DiagnosticsInfo& diagnostics, Functor&& functor) WI_NOEXCEPT
    {
        static_assert(details::functor_tag<ErrorReturn::None, Functor>::value == details::tag_return_void::value, "Functor must return void");
        details::functor_tag<ErrorReturn::None, Functor>::functor_wrapper<Functor> functorObject(wistd::forward<Functor>(functor));

        wil::details::ResultFromExceptionDebug(diagnostics, SupportedExceptions::None, functorObject);
    }

#endif // WIL_ENABLE_EXCEPTIONS

    //! Identical to 'throw;', but can be called from error-code neutral code to rethrow in code that *may* be running under an exception context
    inline void RethrowCaughtException()
    {
        // We always want to rethrow the exception under normal circumstances.  Ordinarily, we could actually guarantee
        // this as we should be able to rethrow if we caught an exception, but if we got here in the middle of running
        // dynamic initializers, then it's possible that we haven't yet setup the rethrow function pointer, thus the
        // runtime check without the noreturn annotation.

        if (details::g_pfnRethrow)
        {
            details::g_pfnRethrow();
        }
    }

    //! Identical to 'throw ResultException(failure);', but can be referenced from error-code neutral code
    inline void ThrowResultException(const FailureInfo& failure)
    {
        if (details::g_pfnThrowResultException)
        {
            details::g_pfnThrowResultException(failure);
        }
    }

    //! @cond
    namespace details
    {
#ifdef WIL_ENABLE_EXCEPTIONS
        //*****************************************************************************
        // Private helpers to catch and propagate exceptions
        //*****************************************************************************

        __declspec(noreturn) inline void TerminateAndReportError(_In_opt_ PEXCEPTION_POINTERS)
        {
            // This is an intentional fail-fast that was caught by an exception guard with WIL.  Look back up the callstack to determine
            // the source of the actual exception being thrown.  The exception guard used by the calling code did not expect this
            // exception type to be thrown or is specifically requesting fail-fast for this class of exception.

            RESULT_RAISE_FAST_FAIL_EXCEPTION;
        }

        inline void MaybeGetExceptionString(const ResultException& exception, _Out_writes_opt_(debugStringChars) PWSTR debugString, _When_(debugString != nullptr, _Pre_satisfies_(debugStringChars > 0)) size_t debugStringChars)
        {
            if (debugString)
            {
                GetFailureLogString(debugString, debugStringChars, exception.GetFailureInfo());
            }
        }

        inline void MaybeGetExceptionString(const std::exception& exception, _Out_writes_opt_(debugStringChars) PWSTR debugString, _When_(debugString != nullptr, _Pre_satisfies_(debugStringChars > 0)) size_t debugStringChars)
        {
            if (debugString)
            {
                StringCchPrintfW(debugString, debugStringChars, L"std::exception: %hs", exception.what());
            }
        }

        inline HRESULT ResultFromKnownException(const ResultException& exception, const DiagnosticsInfo& diagnostics, void* returnAddress)
        {
            wchar_t message[2048];
            message[0] = L'\0';
            MaybeGetExceptionString(exception, message, ARRAYSIZE(message));
            auto hr = exception.GetErrorCode();
            wil::details::ReportFailure(__R_DIAGNOSTICS_RA(diagnostics, returnAddress), FailureType::Log, hr, message);
            return hr;
        }

        inline HRESULT ResultFromKnownException(const std::bad_alloc& exception, const DiagnosticsInfo& diagnostics, void* returnAddress)
        {
            wchar_t message[2048];
            message[0] = L'\0';
            MaybeGetExceptionString(exception, message, ARRAYSIZE(message));
            auto hr = E_OUTOFMEMORY;
            wil::details::ReportFailure(__R_DIAGNOSTICS_RA(diagnostics, returnAddress), FailureType::Log, hr, message);
            return hr;
        }

        inline HRESULT ResultFromKnownException(const std::exception& exception, const DiagnosticsInfo& diagnostics, void* returnAddress)
        {
            wchar_t message[2048];
            message[0] = L'\0';
            MaybeGetExceptionString(exception, message, ARRAYSIZE(message));
            auto hr = HRESULT_FROM_WIN32(ERROR_UNHANDLED_EXCEPTION);
            wil::details::ReportFailure(__R_DIAGNOSTICS_RA(diagnostics, returnAddress), FailureType::Log, hr, message);
            return hr;
        }

        inline HRESULT RecognizeCaughtExceptionFromCallback(_Inout_updates_opt_(debugStringChars) PWSTR debugString, _When_(debugString != nullptr, _Pre_satisfies_(debugStringChars > 0)) size_t debugStringChars)
        {
            HRESULT hr = g_pfnResultFromCaughtException();

            // If we still don't know the error -- or we would like to get the debug string for the error (if possible) we
            // rethrow and catch std::exception.

            if (SUCCEEDED(hr) || debugString)
            {
                try
                {
                    throw;
                }
                catch (std::exception& exception)
                {
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    if (SUCCEEDED(hr))
                    {
                        hr = HRESULT_FROM_WIN32(ERROR_UNHANDLED_EXCEPTION);
                    }
                }
                catch (...)
                {
                }
            }

            return hr;
        }

#ifdef __cplusplus_winrt
        inline Platform::String^ GetPlatformExceptionMessage(Platform::Exception^ exception)
        {
            struct RawExceptionData_Partial
            {
                PCWSTR description;
                PCWSTR restrictedErrorString;
            };

            auto exceptionPtr = reinterpret_cast<void*>(static_cast<::Platform::Object^>(exception));
            auto exceptionInfoPtr = reinterpret_cast<ULONG_PTR*>(exceptionPtr) - 1;
            auto partial = reinterpret_cast<RawExceptionData_Partial*>(*exceptionInfoPtr);

            Platform::String^ message = exception->Message;

            PCWSTR errorString = partial->restrictedErrorString;
            PCWSTR messageString = reinterpret_cast<PCWSTR>(message ? message->Data() : nullptr);

            // An old Platform::Exception^ bug that did not actually expose the error string out of the exception
            // message.  We do it by hand here if the message associated with the strong does not contain the
            // message that was originally attached to the string (in the fixed version it will).

            if ((errorString && *errorString && messageString) &&
                (wcsstr(messageString, errorString) == nullptr))
            {
                return ref new Platform::String(reinterpret_cast<_Null_terminated_ const __wchar_t *>(errorString));
            }
            return message;
        }

        inline void MaybeGetExceptionString(_In_ Platform::Exception^ exception, _Out_writes_opt_(debugStringChars) PWSTR debugString, _When_(debugString != nullptr, _Pre_satisfies_(debugStringChars > 0)) size_t debugStringChars)
        {
            if (debugString)
            {
                auto message = GetPlatformExceptionMessage(exception);
                auto messageString = !message ? L"(null Message)" : reinterpret_cast<PCWSTR>(message->Data());
                StringCchPrintfW(debugString, debugStringChars, L"Platform::Exception^: %ws", messageString);
            }
        }

        inline HRESULT ResultFromKnownException(Platform::Exception^ exception, const DiagnosticsInfo& diagnostics, void* returnAddress)
        {
            wchar_t message[2048];
            message[0] = L'\0';
            MaybeGetExceptionString(exception, message, ARRAYSIZE(message));
            auto hr = exception->HResult;
            wil::details::ReportFailure(__R_DIAGNOSTICS_RA(diagnostics, returnAddress), FailureType::Log, hr, message);
            return hr;
        }

        inline HRESULT __stdcall ResultFromCaughtException_WinRt(_Inout_updates_opt_(debugStringChars) PWSTR debugString, _When_(debugString != nullptr, _Pre_satisfies_(debugStringChars > 0)) size_t debugStringChars, _Inout_ bool* isNormalized) WI_NOEXCEPT
        {
            if (g_pfnResultFromCaughtException)
            {
                try
                {
                    throw;
                }
                catch (const ResultException& exception)
                {
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return exception.GetErrorCode();
                }
                catch (Platform::Exception^ exception)
                {
                    *isNormalized = true;
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return exception->HResult;
                }
                catch (const std::bad_alloc& exception)
                {
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return E_OUTOFMEMORY;
                }
                catch (...)
                {
                    auto hr = RecognizeCaughtExceptionFromCallback(debugString, debugStringChars);
                    if (FAILED(hr))
                    {
                        return hr;
                    }
                }
            }
            else
            {
                try
                {
                    throw;
                }
                catch (const ResultException& exception)
                {
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return exception.GetErrorCode();
                }
                catch (Platform::Exception^ exception)
                {
                    *isNormalized = true;
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return exception->HResult;
                }
                catch (const std::bad_alloc& exception)
                {
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return E_OUTOFMEMORY;
                }
                catch (std::exception& exception)
                {
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return HRESULT_FROM_WIN32(ERROR_UNHANDLED_EXCEPTION);
                }
                catch (...)
                {
                }
            }

            // Tell the caller that we were unable to map the exception by succeeding...
            return S_OK;
        }

        // WinRT supporting version to execute a functor and catch known exceptions.
        inline HRESULT __stdcall ResultFromKnownExceptions_WinRt(const DiagnosticsInfo& diagnostics, void* returnAddress, SupportedExceptions supported, IFunctor& functor)
        {
            WI_ASSERT(supported != SupportedExceptions::Default);

            switch (supported)
            {
            case SupportedExceptions::Known:
                try
                {
                    return functor.Run();
                }
                catch (const ResultException& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                catch (Platform::Exception^ exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                catch (const std::bad_alloc& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                catch (std::exception& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                break;

            case SupportedExceptions::ThrownOrAlloc:
                try
                {
                    return functor.Run();
                }
                catch (const ResultException& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                catch (Platform::Exception^ exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                catch (const std::bad_alloc& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                break;

            case SupportedExceptions::Thrown:
                try
                {
                    return functor.Run();
                }
                catch (const ResultException& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                catch (Platform::Exception^ exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                break;
            }

            WI_ASSERT(false);
            return S_OK;
        }

        inline void __stdcall ThrowPlatformException(FailureInfo const &failure, LPCWSTR debugString)
        {
            throw Platform::Exception::CreateException(failure.hr, ref new Platform::String(reinterpret_cast<_Null_terminated_ const __wchar_t *>(debugString)));
        }

#if !defined(RESULT_SUPPRESS_STATIC_INITIALIZERS)
        WI_HEADER_INITITALIZATION_FUNCTION(InitializeWinRt, []
        {
            g_pfnResultFromCaughtException_WinRt = ResultFromCaughtException_WinRt;
            g_pfnResultFromKnownExceptions_WinRt = ResultFromKnownExceptions_WinRt;
            g_pfnThrowPlatformException = ThrowPlatformException;
            return 1;
        });
#endif
#endif

        inline void __stdcall Rethrow()
        {
            throw;
        }

        inline void __stdcall ThrowResultExceptionInternal(const FailureInfo& failure)
        {
            throw ResultException(failure);
        }

        __declspec(noinline) inline HRESULT __stdcall ResultFromCaughtExceptionInternal(_Out_writes_opt_(debugStringChars) PWSTR debugString, _When_(debugString != nullptr, _Pre_satisfies_(debugStringChars > 0)) size_t debugStringChars, _Out_ bool* isNormalized) WI_NOEXCEPT
        {
            if (debugString)
            {
                *debugString = L'\0';
            }
            *isNormalized = false;

            if (details::g_pfnResultFromCaughtException_WinRt != nullptr)
            {
                return details::g_pfnResultFromCaughtException_WinRt(debugString, debugStringChars, isNormalized);
            }

            if (g_pfnResultFromCaughtException)
            {
                try
                {
                    throw;
                }
                catch (const ResultException& exception)
                {
                    *isNormalized = true;
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return exception.GetErrorCode();
                }
                catch (const std::bad_alloc& exception)
                {
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return E_OUTOFMEMORY;
                }
                catch (...)
                {
                    auto hr = RecognizeCaughtExceptionFromCallback(debugString, debugStringChars);
                    if (FAILED(hr))
                    {
                        return hr;
                    }
                }
            }
            else
            {
                try
                {
                    throw;
                }
                catch (const ResultException& exception)
                {
                    *isNormalized = true;
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return exception.GetErrorCode();
                }
                catch (const std::bad_alloc& exception)
                {
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return E_OUTOFMEMORY;
                }
                catch (std::exception& exception)
                {
                    MaybeGetExceptionString(exception, debugString, debugStringChars);
                    return HRESULT_FROM_WIN32(ERROR_UNHANDLED_EXCEPTION);
                }
                catch (...)
                {
                }
            }

            // Tell the caller that we were unable to map the exception by succeeding...
            return S_OK;
        }

        // Runs the given functor, converting any exceptions of the supported types that are known to HRESULTs and returning
        // that HRESULT.  Does NOT attempt to catch unknown exceptions (which propagate).  Primarily used by SEH exception
        // handling techniques to stop at the point the exception is thrown.
        inline HRESULT ResultFromKnownExceptions(const DiagnosticsInfo& diagnostics, void* returnAddress, SupportedExceptions supported, IFunctor& functor)
        {
            if (supported == SupportedExceptions::Default)
            {
                supported = g_fResultSupportStdException ? SupportedExceptions::Known : SupportedExceptions::ThrownOrAlloc;
            }

            if ((details::g_pfnResultFromKnownExceptions_WinRt != nullptr) &&
                ((supported == SupportedExceptions::Known) || (supported == SupportedExceptions::Thrown) || (supported == SupportedExceptions::ThrownOrAlloc)))
            {
                return details::g_pfnResultFromKnownExceptions_WinRt(diagnostics, returnAddress, supported, functor);
            }

            switch (supported)
            {
            case SupportedExceptions::Known:
                try
                {
                    return functor.Run();
                }
                catch (const ResultException& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                catch (const std::bad_alloc& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                catch (std::exception& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }

            case SupportedExceptions::ThrownOrAlloc:
                try
                {
                    return functor.Run();
                }
                catch (const ResultException& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }
                catch (const std::bad_alloc& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }

            case SupportedExceptions::Thrown:
                try
                {
                    return functor.Run();
                }
                catch (const ResultException& exception)
                {
                    return ResultFromKnownException(exception, diagnostics, returnAddress);
                }

            case SupportedExceptions::All:
                try
                {
                    return functor.Run();
                }
                catch (...)
                {
                    return wil::details::ReportFailure_CaughtException(__R_DIAGNOSTICS_RA(diagnostics, returnAddress), FailureType::Log, supported);
                }

            case SupportedExceptions::None:
                return functor.Run();
            }

            WI_ASSERT(false);
            return S_OK;
        }

        inline HRESULT ResultFromExceptionSeh(const DiagnosticsInfo& diagnostics, void* returnAddress, SupportedExceptions supported, IFunctor& functor) WI_NOEXCEPT
        {
            __try
            {
                return wil::details::ResultFromKnownExceptions(diagnostics, returnAddress, supported, functor);
            }
            __except (wil::details::TerminateAndReportError(GetExceptionInformation()), EXCEPTION_CONTINUE_SEARCH)
            {
            }
        }

        __declspec(noinline) inline HRESULT ResultFromException(const DiagnosticsInfo& diagnostics, SupportedExceptions supported, IFunctor& functor) WI_NOEXCEPT
        {
#ifdef RESULT_DEBUG
            // We can't do debug SEH handling if the caller also wants a shot at mapping the exceptions
            // themselves or if the caller doesn't want to fail-fast unknown exceptions
            if ((g_pfnResultFromCaughtException == nullptr) && g_fResultFailFastUnknownExceptions)
            {
                return wil::details::ResultFromExceptionSeh(diagnostics, _ReturnAddress(), supported, functor);
            }
#endif
            try
            {
                return functor.Run();
            }
            catch (...)
            {
                return wil::details::ReportFailure_CaughtException(__R_DIAGNOSTICS(diagnostics), _ReturnAddress(), FailureType::Log, supported);
            }
        }

        __declspec(noinline) inline HRESULT ResultFromExceptionDebug(const DiagnosticsInfo& diagnostics, SupportedExceptions supported, IFunctor& functor) WI_NOEXCEPT
        {
            return wil::details::ResultFromExceptionSeh(diagnostics, _ReturnAddress(), supported, functor);
        }

        // Exception guard -- catch exceptions and log them (or handle them with a custom callback)
        // WARNING: may throw an exception...
        inline HRESULT __stdcall RunFunctorWithExceptionFilter(IFunctor& functor, IFunctorHost& host, void* returnAddress)
        {
            try
            {
                return host.Run(functor);
            }
            catch (...)
            {
                // Note that the host may choose to re-throw, throw a normalized exception, return S_OK and eat the exception or
                // return the remapped failure.
                return host.ExceptionThrown(returnAddress);
            }
        }

        WI_HEADER_INITITALIZATION_FUNCTION(InitializeResultExceptions, []
        {
            g_pfnRunFunctorWithExceptionFilter = RunFunctorWithExceptionFilter;
            g_pfnRethrow = Rethrow;
            g_pfnThrowResultException = ThrowResultExceptionInternal;
            g_pfnResultFromCaughtExceptionInternal = ResultFromCaughtExceptionInternal;
            return 1;
        });

#endif  // WIL_ENABLE_EXCEPTIONS

        // Exception guard -- catch exceptions and log them (or handle them with a custom callback)
        // WARNING: may throw an exception...
        inline __declspec(noinline) HRESULT RunFunctor(IFunctor& functor, IFunctorHost& host)
        {
            if (g_pfnRunFunctorWithExceptionFilter)
            {
                return g_pfnRunFunctorWithExceptionFilter(functor, host, _ReturnAddress());
            }

            return host.Run(functor);
        }


        //*****************************************************************************
        // Shared Reporting -- all reporting macros bubble up through this codepath
        //*****************************************************************************

        inline void LogFailure(__R_FN_PARAMS_FULL, FailureType type, HRESULT hr, _In_opt_ PCWSTR message,
            bool fWantDebugString, _Out_writes_(debugStringSizeChars) _Post_z_ PWSTR debugString, _Pre_satisfies_(debugStringSizeChars > 0) size_t debugStringSizeChars,
            _Out_writes_(callContextStringSizeChars) _Post_z_ PSTR callContextString, _Pre_satisfies_(callContextStringSizeChars > 0) size_t callContextStringSizeChars,
            _Out_ FailureInfo *failure) WI_NOEXCEPT
        {
            debugString[0] = L'\0';
            callContextString[0] = L'\0';

            static long volatile s_failureId = 0;

            int failureCount = 0;
            switch (type)
            {
            case FailureType::Exception:
                failureCount = RecordException(hr);
                break;
            case FailureType::Return:
                failureCount = RecordReturn(hr);
                break;
            case FailureType::Log:
                if (SUCCEEDED(hr))
                {
                    // If you hit this assert (or are reviewing this failure telemetry), then most likely you are trying to log success
                    // using one of the WIL macros.  Example:
                    //      LOG_HR(S_OK);
                    // Instead, use one of the forms that conditionally logs based upon the error condition:
                    //      LOG_IF_FAILED(hr);

                    WI_USAGE_ERROR_FORWARD("CALLER BUG: Macro usage error detected.  Do not LOG_XXX success.");
                    hr = HRESULT_FROM_WIN32(ERROR_ASSERTION_FAILURE);
                }
                failureCount = RecordLog(hr);
                break;
            case FailureType::FailFast:
                failureCount = RecordFailFast(hr);
                break;
            };

            failure->type = type;
            failure->hr = hr;
            failure->failureId = ::InterlockedIncrementNoFence(&s_failureId);
            failure->pszMessage = ((message != nullptr) && (message[0] != L'\0')) ? message : nullptr;
            failure->threadId = ::GetCurrentThreadId();
            failure->pszFile = fileName;
            failure->uLineNumber = lineNumber;
            failure->cFailureCount = failureCount;
            failure->pszCode = code;
            failure->pszFunction = functionName;
            failure->returnAddress = returnAddress;
            failure->callerReturnAddress = callerReturnAddress;
            failure->pszCallContext = nullptr;
            ::ZeroMemory(&failure->callContextCurrent, sizeof(failure->callContextCurrent));
            ::ZeroMemory(&failure->callContextOriginating, sizeof(failure->callContextOriginating));
            failure->pszModule = (g_pfnGetModuleName != nullptr) ? g_pfnGetModuleName() : nullptr;

            // Completes filling out failure, notifies thread-based callbacks and the telemetry callback
            if (details::g_pfnGetContextAndNotifyFailure)
            {
                details::g_pfnGetContextAndNotifyFailure(failure, callContextString, callContextStringSizeChars);
            }

            // Allow hooks to inspect the failure before acting upon it
            if (details::g_pfnLoggingCallback)
            {
                details::g_pfnLoggingCallback(*failure);
            }

            if (SUCCEEDED(failure->hr))
            {
                // Caller bug: Leaking a success code into a failure-only function
                FAIL_FAST_IMMEDIATE_IF(type != FailureType::FailFast);
                failure->hr = E_UNEXPECTED;
            }

            // We log to OutputDebugString if:
            // * Someone set g_fResultOutputDebugString to true (by the calling module or in the debugger)
            // * Neither a telemetry nor logging callback is set
            // * Someone set s g_pfnShouldOutputDebugString which returned true.
            bool const fUseOutputDebugString = (g_fResultOutputDebugString
                || ((g_pfnTelemetryCallback == nullptr) && (g_pfnResultLoggingCallback == nullptr))
                || ((g_pfnShouldOutputDebugString != nullptr) && g_pfnShouldOutputDebugString())
                );

            // We need to generate the logging message if:
            // * We're logging to OutputDebugString
            // * OR the caller asked us to (generally for attaching to a C++/CX exception)
            if (fWantDebugString || fUseOutputDebugString)
            {
                // Call the logging callback (if present) to allow them to generate the debug string that will be pushed to the console
                // or the platform exception object if the caller desires it.
                if ((g_pfnResultLoggingCallback != nullptr) && !g_resultMessageCallbackSet)
                {
                    g_pfnResultLoggingCallback(failure, debugString, debugStringSizeChars);
                }

                // The callback only optionally needs to supply the debug string -- if the callback didn't populate it, yet we still want
                // it for OutputDebugString or exception message, then generate the default string.
                if (debugString[0] == L'\0')
                {
                    GetFailureLogString(debugString, debugStringSizeChars, *failure);
                }

                if (fUseOutputDebugString)
                {
                    ::OutputDebugStringW(debugString);
                }
            }
            else
            {
                // [deprecated behavior]
                // This callback was at one point *always* called for all failures, so we continue to call it for failures even when we don't
                // need to generate the debug string information (when the callback was supplied directly).  We can avoid this if the caller
                // used the explicit function (through g_resultMessageCallbackSet)
                if ((g_pfnResultLoggingCallback != nullptr) && !g_resultMessageCallbackSet)
                {
                    g_pfnResultLoggingCallback(failure, nullptr, 0);
                }
            }
        }

        inline __declspec(noinline) void ReportFailure(__R_FN_PARAMS_FULL, FailureType type, HRESULT hr, _In_opt_ PCWSTR message, ReportFailureOptions options)
        {
            bool needPlatformException = ((type == FailureType::Exception) &&
                WI_IS_FLAG_CLEAR(options, ReportFailureOptions::MayRethrow) &&
                (g_pfnThrowPlatformException != nullptr) &&
                (g_fResultThrowPlatformException || WI_IS_FLAG_SET(options, ReportFailureOptions::ForcePlatformException)));

            FailureInfo failure;
            wchar_t debugString[2048];
            char callContextString[1024];

            LogFailure(__R_FN_CALL_FULL, type, hr, message, needPlatformException,
                debugString, ARRAYSIZE(debugString), callContextString, ARRAYSIZE(callContextString), &failure);

            if (WI_IS_FLAG_CLEAR(options, ReportFailureOptions::SuppressAction))
            {
                if (type == FailureType::FailFast)
                {
                    // This is an explicit fail fast - examine the callstack to determine the precise reason for this failure
                    RESULT_RAISE_FAST_FAIL_EXCEPTION;
                }
                else if (type == FailureType::Exception)
                {
                    if (needPlatformException)
                    {
                        g_pfnThrowPlatformException(failure, debugString);
                    }

                    if (WI_IS_FLAG_SET(options, ReportFailureOptions::MayRethrow))
                    {
                        RethrowCaughtException();
                    }

                    ThrowResultException(failure);
                    
                    // Wil was instructed to throw, but doesn't have any capability to do so (global function pointers are not setup)
                    RESULT_RAISE_FAST_FAIL_EXCEPTION;
                }
            }
        }

        inline HRESULT ReportFailure_CaughtExceptionCommon(__R_FN_PARAMS_FULL, FailureType type, _Inout_updates_(debugStringChars) PWSTR debugString, _Pre_satisfies_(debugStringChars > 0) size_t debugStringChars, SupportedExceptions supported)
        {
            bool isNormalized = false;
            auto length = wcslen(debugString);
            WI_ASSERT(length < debugStringChars);
            HRESULT hr = S_OK;
            if (details::g_pfnResultFromCaughtExceptionInternal)
            {
                hr = details::g_pfnResultFromCaughtExceptionInternal(debugString + length, debugStringChars - length, &isNormalized);
            }

            const bool known = (FAILED(hr));
            if (!known)
            {
                hr = HRESULT_FROM_WIN32(ERROR_UNHANDLED_EXCEPTION);
            }

            if ((supported == SupportedExceptions::None) ||
                ((supported == SupportedExceptions::Known) && !known) ||
                ((supported == SupportedExceptions::Thrown) && !isNormalized) ||
                ((supported == SupportedExceptions::Default) && !known && g_fResultFailFastUnknownExceptions))
            {
                // By default WIL will issue a fail fast for unrecognized exception types.  Wil recognizes any std::exception or wil::ResultException based
                // types and Platform::Exception^, so there aren't too many valid exception types which could cause this.  Those that are valid, should be handled 
                // by remapping the exception callback.  Those that are not valid should be found and fixed (meaningless accidents like 'throw hr;').
                // The caller may also be requesting non-default behavior to fail-fast more frequently (primarily for debugging unknown exceptions).

                type = FailureType::FailFast;
            }

            ReportFailureOptions options = ReportFailureOptions::ForcePlatformException;
            WI_SET_FLAG_IF(options, ReportFailureOptions::MayRethrow, isNormalized);
            ReportFailure(__R_FN_CALL_FULL, type, hr, debugString, options);
            return hr;
        }

        inline void ReportFailure_Msg(__R_FN_PARAMS_FULL, FailureType type, HRESULT hr, _Printf_format_string_ PCSTR formatString, va_list argList)
        {
            wchar_t message[2048];
            PrintLoggingMessage(message, ARRAYSIZE(message), formatString, argList);
            ReportFailure(__R_FN_CALL_FULL, type, hr, message);
        }

        inline void ReportFailure_ReplaceMsg(__R_FN_PARAMS_FULL, FailureType type, HRESULT hr, _Printf_format_string_ PCSTR formatString, ...)
        {
            va_list argList;
            va_start(argList, formatString);
            ReportFailure_Msg(__R_FN_CALL_FULL, type, hr, formatString, argList);
        }

        __declspec(noinline) inline void ReportFailure_Hr(__R_FN_PARAMS_FULL, FailureType type, HRESULT hr)
        {
            ReportFailure(__R_FN_CALL_FULL, type, hr);
        }

        __declspec(noinline) inline HRESULT ReportFailure_Win32(__R_FN_PARAMS_FULL, FailureType type, DWORD err)
        {
            auto hr = HRESULT_FROM_WIN32(err);
            ReportFailure(__R_FN_CALL_FULL, type, hr);
            return hr;
        }

        __declspec(noinline) inline DWORD ReportFailure_GetLastError(__R_FN_PARAMS_FULL, FailureType type)
        {
            auto err = GetLastErrorFail(__R_FN_CALL_FULL);
            ReportFailure(__R_FN_CALL_FULL, type, HRESULT_FROM_WIN32(err));
            return err;
        }

        __declspec(noinline) inline HRESULT ReportFailure_GetLastErrorHr(__R_FN_PARAMS_FULL, FailureType type)
        {
            auto hr = GetLastErrorFailHr(__R_FN_CALL_FULL);
            ReportFailure(__R_FN_CALL_FULL, type, hr);
            return hr;
        }

        __declspec(noinline) inline HRESULT ReportFailure_NtStatus(__R_FN_PARAMS_FULL, FailureType type, NTSTATUS status)
        {
            auto hr = wil::details::NtStatusToHr(status);
            ReportFailure(__R_FN_CALL_FULL, type, hr);
            return hr;
        }

        __declspec(noinline) inline HRESULT ReportFailure_CaughtException(__R_FN_PARAMS_FULL, FailureType type, SupportedExceptions supported)
        {
            wchar_t message[2048];
            message[0] = L'\0';
            return ReportFailure_CaughtExceptionCommon(__R_FN_CALL_FULL, type, message, ARRAYSIZE(message), supported);
        }

        __declspec(noinline) inline void ReportFailure_HrMsg(__R_FN_PARAMS_FULL, FailureType type, HRESULT hr, _Printf_format_string_ PCSTR formatString, va_list argList)
        {
            ReportFailure_Msg(__R_FN_CALL_FULL, type, hr, formatString, argList);
        }

        __declspec(noinline) inline HRESULT ReportFailure_Win32Msg(__R_FN_PARAMS_FULL, FailureType type, DWORD err, _Printf_format_string_ PCSTR formatString, va_list argList)
        {
            auto hr = HRESULT_FROM_WIN32(err);
            ReportFailure_Msg(__R_FN_CALL_FULL, type, HRESULT_FROM_WIN32(err), formatString, argList);
            return hr;
        }

        __declspec(noinline) inline DWORD ReportFailure_GetLastErrorMsg(__R_FN_PARAMS_FULL, FailureType type, _Printf_format_string_ PCSTR formatString, va_list argList)
        {
            auto err = GetLastErrorFail(__R_FN_CALL_FULL);
            ReportFailure_Msg(__R_FN_CALL_FULL, type, HRESULT_FROM_WIN32(err), formatString, argList);
            return err;
        }

        __declspec(noinline) inline HRESULT ReportFailure_GetLastErrorHrMsg(__R_FN_PARAMS_FULL, FailureType type, _Printf_format_string_ PCSTR formatString, va_list argList)
        {
            auto hr = GetLastErrorFailHr(__R_FN_CALL_FULL);
            ReportFailure_Msg(__R_FN_CALL_FULL, type, hr, formatString, argList);
            return hr;
        }

        __declspec(noinline) inline HRESULT ReportFailure_NtStatusMsg(__R_FN_PARAMS_FULL, FailureType type, NTSTATUS status, _Printf_format_string_ PCSTR formatString, va_list argList)
        {
            auto hr = wil::details::NtStatusToHr(status);
            ReportFailure_Msg(__R_FN_CALL_FULL, type, hr, formatString, argList);
            return hr;
        }

        __declspec(noinline) inline HRESULT ReportFailure_CaughtExceptionMsg(__R_FN_PARAMS_FULL, FailureType type, _Printf_format_string_ PCSTR formatString, va_list argList)
        {
            // Pre-populate the buffer with our message, the exception message will be added to it...
            wchar_t message[2048];
            PrintLoggingMessage(message, ARRAYSIZE(message), formatString, argList);
            StringCchCatW(message, ARRAYSIZE(message), L" -- ");
            return ReportFailure_CaughtExceptionCommon(__R_FN_CALL_FULL, type, message, ARRAYSIZE(message), SupportedExceptions::Default);
        }


        //*****************************************************************************
        // Support for throwing custom exception types
        //*****************************************************************************

#ifdef WIL_ENABLE_EXCEPTIONS
        inline HRESULT GetErrorCode(_In_ ResultException &exception) WI_NOEXCEPT
        {
            return exception.GetErrorCode();
        }

        inline void SetFailureInfo(_In_ FailureInfo const &failure, _Inout_ ResultException &exception) WI_NOEXCEPT
        {
            return exception.SetFailureInfo(failure);
        }

#ifdef __cplusplus_winrt
        inline HRESULT GetErrorCode(_In_ Platform::Exception^ exception) WI_NOEXCEPT
        {
            return exception->HResult;
        }

        inline void SetFailureInfo(_In_ FailureInfo const &, _Inout_ Platform::Exception^ exception) WI_NOEXCEPT
        {
            // no-op -- once a PlatformException^ is created, we can't modify the message, but this function must
            // exist to distinguish this from ResultException
        }
#endif

        template <typename T>
        __declspec(noreturn) inline void ReportFailure_CustomExceptionHelper(_Inout_ T &exception, __R_FN_PARAMS_FULL, _In_opt_ PCWSTR message = nullptr)
        {
            // When seeing the error: "cannot convert parameter 1 from 'XXX' to 'wil::ResultException &'"
            // Custom exceptions must be based upon either ResultException or Platform::Exception^ to be used with ResultException.h.
            // This compilation error indicates an attempt to throw an incompatible exception type.
            const HRESULT hr = GetErrorCode(exception);

            FailureInfo failure;
            wchar_t debugString[2048];
            char callContextString[1024];

            LogFailure(__R_FN_CALL_FULL, FailureType::Exception, hr, message, false,     // false = does not need debug string
                       debugString, ARRAYSIZE(debugString), callContextString, ARRAYSIZE(callContextString), &failure);

            // push the failure info context into the custom exception class
            SetFailureInfo(failure, exception);

            throw exception;
        }

        template <typename T>
        __declspec(noreturn, noinline) inline void ReportFailure_CustomException(__R_FN_PARAMS _In_ T exception)
        {
            __R_FN_LOCALS_RA;
            ReportFailure_CustomExceptionHelper(exception, __R_FN_CALL_FULL);
        }

        template <typename T>
        __declspec(noreturn, noinline) inline void ReportFailure_CustomExceptionMsg(__R_FN_PARAMS _In_ T exception, _In_ _Printf_format_string_ PCSTR formatString, ...)
        {
            va_list argList;
            va_start(argList, formatString);
            wchar_t message[2048];
            PrintLoggingMessage(message, ARRAYSIZE(message), formatString, argList);

            __R_FN_LOCALS_RA;
            ReportFailure_CustomExceptionHelper(exception, __R_FN_CALL_FULL, message);
        }
#endif

        namespace __R_NS_NAME
        {
            //*****************************************************************************
            // Return Macros
            //*****************************************************************************

            __R_DIRECT_METHOD(void, Return_Hr)(__R_DIRECT_FN_PARAMS HRESULT hr) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Hr(__R_DIRECT_FN_CALL FailureType::Return, hr);
            }

            __R_DIRECT_METHOD(HRESULT, Return_Win32)(__R_DIRECT_FN_PARAMS DWORD err) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                return wil::details::ReportFailure_Win32(__R_DIRECT_FN_CALL FailureType::Return, err);
            }

            __R_DIRECT_METHOD(HRESULT, Return_GetLastError)(__R_DIRECT_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                return wil::details::ReportFailure_GetLastErrorHr(__R_DIRECT_FN_CALL FailureType::Return);
            }

            __R_DIRECT_METHOD(HRESULT, Return_NtStatus)(__R_DIRECT_FN_PARAMS NTSTATUS status) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                return wil::details::ReportFailure_NtStatus(__R_DIRECT_FN_CALL FailureType::Return, status);
            }

#ifdef WIL_ENABLE_EXCEPTIONS
            __R_DIRECT_METHOD(HRESULT, Return_CaughtException)(__R_DIRECT_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                return wil::details::ReportFailure_CaughtException(__R_DIRECT_FN_CALL FailureType::Return);
            }
#endif

            __R_DIRECT_METHOD(void, Return_HrMsg)(__R_DIRECT_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__R_DIRECT_FN_CALL FailureType::Return, hr, formatString, argList);
            }

            __R_DIRECT_METHOD(HRESULT, Return_Win32Msg)(__R_DIRECT_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                return wil::details::ReportFailure_Win32Msg(__R_DIRECT_FN_CALL FailureType::Return, err, formatString, argList);
            }

            __R_DIRECT_METHOD(HRESULT, Return_GetLastErrorMsg)(__R_DIRECT_FN_PARAMS _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                return wil::details::ReportFailure_GetLastErrorHrMsg(__R_DIRECT_FN_CALL FailureType::Return, formatString, argList);
            }

            __R_DIRECT_METHOD(HRESULT, Return_NtStatusMsg)(__R_DIRECT_FN_PARAMS NTSTATUS status, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                return wil::details::ReportFailure_NtStatusMsg(__R_DIRECT_FN_CALL FailureType::Return, status, formatString, argList);
            }

#ifdef WIL_ENABLE_EXCEPTIONS
            __R_DIRECT_METHOD(HRESULT, Return_CaughtExceptionMsg)(__R_DIRECT_FN_PARAMS _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                return wil::details::ReportFailure_CaughtExceptionMsg(__R_DIRECT_FN_CALL FailureType::Return, formatString, argList);
            }
#endif

            //*****************************************************************************
            // Log Macros
            //*****************************************************************************

            __R_DIRECT_METHOD(HRESULT, Log_Hr)(__R_DIRECT_FN_PARAMS HRESULT hr) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Hr(__R_DIRECT_FN_CALL FailureType::Log, hr);
                return hr;
            }

            __R_DIRECT_METHOD(DWORD, Log_Win32)(__R_DIRECT_FN_PARAMS DWORD err) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Win32(__R_DIRECT_FN_CALL FailureType::Log, err);
                return err;
            }

            __R_DIRECT_METHOD(DWORD, Log_GetLastError)(__R_DIRECT_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                return wil::details::ReportFailure_GetLastError(__R_DIRECT_FN_CALL FailureType::Log);
            }

            __R_DIRECT_METHOD(NTSTATUS, Log_NtStatus)(__R_DIRECT_FN_PARAMS NTSTATUS status) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_NtStatus(__R_DIRECT_FN_CALL FailureType::Log, status);
                return status;
            }

#ifdef WIL_ENABLE_EXCEPTIONS
            __R_DIRECT_METHOD(HRESULT, Log_CaughtException)(__R_DIRECT_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                return wil::details::ReportFailure_CaughtException(__R_DIRECT_FN_CALL FailureType::Log);
            }
#endif

            __R_INTERNAL_METHOD(_Log_Hr)(__R_INTERNAL_FN_PARAMS HRESULT hr) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Hr(__R_INTERNAL_FN_CALL FailureType::Log, hr);
            }

            __R_INTERNAL_METHOD(_Log_GetLastError)(__R_INTERNAL_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_GetLastError(__R_INTERNAL_FN_CALL FailureType::Log);
            }

            __R_INTERNAL_METHOD(_Log_Win32)(__R_INTERNAL_FN_PARAMS DWORD err) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Win32(__R_INTERNAL_FN_CALL FailureType::Log, err);
            }

            __R_INTERNAL_METHOD(_Log_NullAlloc)(__R_INTERNAL_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Hr(__R_INTERNAL_FN_CALL FailureType::Log, E_OUTOFMEMORY);
            }

            __R_INTERNAL_METHOD(_Log_NtStatus)(__R_INTERNAL_FN_PARAMS NTSTATUS status) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_NtStatus(__R_INTERNAL_FN_CALL FailureType::Log, status);
            }

            __R_CONDITIONAL_METHOD(HRESULT, Log_IfFailed)(__R_CONDITIONAL_FN_PARAMS HRESULT hr) WI_NOEXCEPT
            {
                if (FAILED(hr))
                {
                    __R_CALL_INTERNAL_METHOD(_Log_Hr)(__R_CONDITIONAL_FN_CALL hr);
                }
                return hr;
            }

            __R_CONDITIONAL_METHOD(BOOL, Log_IfWin32BoolFalse)(__R_CONDITIONAL_FN_PARAMS BOOL ret) WI_NOEXCEPT
            {
                if (!ret)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return ret;
            }

            __R_CONDITIONAL_METHOD(DWORD, Log_IfWin32Error)(__R_CONDITIONAL_FN_PARAMS DWORD err) WI_NOEXCEPT
            {
                if (FAILED_WIN32(err))
                {
                    __R_CALL_INTERNAL_METHOD(_Log_Win32)(__R_CONDITIONAL_FN_CALL err);
                }
                return err;
            }

            __R_CONDITIONAL_METHOD(HANDLE, Log_IfHandleInvalid)(__R_CONDITIONAL_FN_PARAMS HANDLE handle) WI_NOEXCEPT
            {
                if (handle == INVALID_HANDLE_VALUE)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return handle;
            }

            __R_CONDITIONAL_METHOD(HANDLE, Log_IfHandleNull)(__R_CONDITIONAL_FN_PARAMS HANDLE handle) WI_NOEXCEPT
            {
                if (handle == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return handle;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(PointerT, Log_IfNullAlloc)(__R_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_NullAlloc)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(void, Log_IfNullAlloc)(__R_CONDITIONAL_FN_PARAMS const PointerT& pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_NullAlloc)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
            }

            __R_CONDITIONAL_METHOD(bool, Log_HrIf)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition) WI_NOEXCEPT
            {
                if (condition)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_Hr)(__R_CONDITIONAL_FN_CALL hr);
                }
                return condition;
            }

            __R_CONDITIONAL_METHOD(bool, Log_HrIfFalse)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition) WI_NOEXCEPT
            {
                if (!condition)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_Hr)(__R_CONDITIONAL_FN_CALL hr);
                }
                return condition;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(PointerT, Log_HrIfNull)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, _Pre_maybenull_ PointerT pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_Hr)(__R_CONDITIONAL_FN_CALL hr);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(void, Log_HrIfNull)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, _In_opt_ const PointerT& pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_Hr)(__R_CONDITIONAL_FN_CALL hr);
                }
            }

            __R_CONDITIONAL_METHOD(bool, Log_GetLastErrorIf)(__R_CONDITIONAL_FN_PARAMS bool condition) WI_NOEXCEPT
            {
                if (condition)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return condition;
            }

            __R_CONDITIONAL_METHOD(bool, Log_GetLastErrorIfFalse)(__R_CONDITIONAL_FN_PARAMS bool condition) WI_NOEXCEPT
            {
                if (!condition)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return condition;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(PointerT, Log_GetLastErrorIfNull)(__R_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(void, Log_GetLastErrorIfNull)(__R_CONDITIONAL_FN_PARAMS _In_opt_ const PointerT& pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Log_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
            }

            __R_CONDITIONAL_METHOD(NTSTATUS, Log_IfNtStatusFailed)(__R_CONDITIONAL_FN_PARAMS NTSTATUS status) WI_NOEXCEPT
            {
                if (FAILED_NTSTATUS(status))
                {
                    __R_CALL_INTERNAL_METHOD(_Log_NtStatus)(__R_CONDITIONAL_FN_CALL status);
                }
                return status;
            }

            __R_DIRECT_METHOD(HRESULT, Log_HrMsg)(__R_DIRECT_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__R_DIRECT_FN_CALL FailureType::Log, hr, formatString, argList);
                return hr;
            }

            __R_DIRECT_METHOD(DWORD, Log_Win32Msg)(__R_DIRECT_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                wil::details::ReportFailure_Win32Msg(__R_DIRECT_FN_CALL FailureType::Log, err, formatString, argList);
                return err;
            }

            __R_DIRECT_METHOD(DWORD, Log_GetLastErrorMsg)(__R_DIRECT_FN_PARAMS _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                return wil::details::ReportFailure_GetLastErrorMsg(__R_DIRECT_FN_CALL FailureType::Log, formatString, argList);
            }
            
            __R_DIRECT_METHOD(NTSTATUS, Log_NtStatusMsg)(__R_DIRECT_FN_PARAMS NTSTATUS status, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                wil::details::ReportFailure_NtStatusMsg(__R_DIRECT_FN_CALL FailureType::Log, status, formatString, argList);
                return status;
            }

#ifdef WIL_ENABLE_EXCEPTIONS
            __R_DIRECT_METHOD(HRESULT, Log_CaughtExceptionMsg)(__R_DIRECT_FN_PARAMS _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                return wil::details::ReportFailure_CaughtExceptionMsg(__R_DIRECT_FN_CALL FailureType::Log, formatString, argList);
            }
#endif            

            __R_INTERNAL_NOINLINE_METHOD(_Log_HrMsg)(__R_INTERNAL_NOINLINE_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__R_INTERNAL_NOINLINE_FN_CALL FailureType::Log, hr, formatString, argList);
            }

            __R_INTERNAL_NOINLINE_METHOD(_Log_GetLastErrorMsg)(__R_INTERNAL_NOINLINE_FN_PARAMS _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_GetLastErrorMsg(__R_INTERNAL_NOINLINE_FN_CALL FailureType::Log, formatString, argList);
            }

            __R_INTERNAL_NOINLINE_METHOD(_Log_Win32Msg)(__R_INTERNAL_NOINLINE_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Win32Msg(__R_INTERNAL_NOINLINE_FN_CALL FailureType::Log, err, formatString, argList);
            }

            __R_INTERNAL_NOINLINE_METHOD(_Log_NullAllocMsg)(__R_INTERNAL_NOINLINE_FN_PARAMS _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__R_INTERNAL_NOINLINE_FN_CALL FailureType::Log, E_OUTOFMEMORY, formatString, argList);
            }

            __R_INTERNAL_NOINLINE_METHOD(_Log_NtStatusMsg)(__R_INTERNAL_NOINLINE_FN_PARAMS NTSTATUS status, _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_NtStatusMsg(__R_INTERNAL_NOINLINE_FN_CALL FailureType::Log, status, formatString, argList);
            }

            __R_CONDITIONAL_NOINLINE_METHOD(HRESULT, Log_IfFailedMsg)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (FAILED(hr))
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_HrMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return hr;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(BOOL, Log_IfWin32BoolFalseMsg)(__R_CONDITIONAL_FN_PARAMS BOOL ret, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (!ret)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return ret;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(DWORD, Log_IfWin32ErrorMsg)(__R_CONDITIONAL_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (FAILED_WIN32(err))
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_Win32Msg)(__R_CONDITIONAL_NOINLINE_FN_CALL err, formatString, argList);
                }
                return err;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(HANDLE, Log_IfHandleInvalidMsg)(__R_CONDITIONAL_FN_PARAMS HANDLE handle, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (handle == INVALID_HANDLE_VALUE)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return handle;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(HANDLE, Log_IfHandleNullMsg)(__R_CONDITIONAL_FN_PARAMS HANDLE handle, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (handle == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return handle;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(PointerT, Log_IfNullAllocMsg)(__R_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_NullAllocMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL_ONLY, formatString, argList);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(void, Log_IfNullAllocMsg)(__R_CONDITIONAL_FN_PARAMS const PointerT& pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_NullAllocMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL_ONLY, formatString, argList);
                }
            }

            __R_CONDITIONAL_NOINLINE_METHOD(bool, Log_HrIfMsg)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_HrMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return condition;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(bool, Log_HrIfFalseMsg)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (!condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_HrMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return condition;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(PointerT, Log_HrIfNullMsg)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, _Pre_maybenull_ PointerT pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_HrMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(void, Log_HrIfNullMsg)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, _In_opt_ const PointerT& pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_HrMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
            }

            __R_CONDITIONAL_NOINLINE_METHOD(bool, Log_GetLastErrorIfMsg)(__R_CONDITIONAL_FN_PARAMS bool condition, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return condition;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(bool, Log_GetLastErrorIfFalseMsg)(__R_CONDITIONAL_FN_PARAMS bool condition, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (!condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return condition;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(PointerT, Log_GetLastErrorIfNullMsg)(__R_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(void, Log_GetLastErrorIfNullMsg)(__R_CONDITIONAL_FN_PARAMS _In_opt_ const PointerT& pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
            }

            __R_CONDITIONAL_NOINLINE_METHOD(NTSTATUS, Log_IfNtStatusFailedMsg)(__R_CONDITIONAL_FN_PARAMS NTSTATUS status, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (FAILED_NTSTATUS(status))
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Log_NtStatusMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL status, formatString, argList);
                }
                return status;
            }
        } // namespace __R_NS_NAME

        namespace __RFF_NS_NAME
        {
            //*****************************************************************************
            // FailFast Macros
            //*****************************************************************************

            __RFF_DIRECT_NORET_METHOD(void, FailFast_Hr)(__RFF_DIRECT_FN_PARAMS HRESULT hr) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_Hr(__RFF_DIRECT_FN_CALL FailureType::FailFast, hr);
            }

            __RFF_DIRECT_NORET_METHOD(void, FailFast_Win32)(__RFF_DIRECT_FN_PARAMS DWORD err) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_Win32(__RFF_DIRECT_FN_CALL FailureType::FailFast, err);
            }

            __RFF_DIRECT_NORET_METHOD(void, FailFast_GetLastError)(__RFF_DIRECT_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_GetLastError(__RFF_DIRECT_FN_CALL FailureType::FailFast);
            }

            __RFF_DIRECT_NORET_METHOD(void, FailFast_NtStatus)(__RFF_DIRECT_FN_PARAMS NTSTATUS status) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_NtStatus(__RFF_DIRECT_FN_CALL FailureType::FailFast, status);
            }

#ifdef WIL_ENABLE_EXCEPTIONS
            __RFF_DIRECT_NORET_METHOD(void, FailFast_CaughtException)(__RFF_DIRECT_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_CaughtException(__RFF_DIRECT_FN_CALL FailureType::FailFast);
            }
#endif

            __RFF_INTERNAL_NORET_METHOD(_FailFast_Hr)(__RFF_INTERNAL_FN_PARAMS HRESULT hr) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_Hr(__RFF_INTERNAL_FN_CALL FailureType::FailFast, hr);
            }

            __RFF_INTERNAL_NORET_METHOD(_FailFast_GetLastError)(__RFF_INTERNAL_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_GetLastError(__RFF_INTERNAL_FN_CALL FailureType::FailFast);
            }

            __RFF_INTERNAL_NORET_METHOD(_FailFast_Win32)(__RFF_INTERNAL_FN_PARAMS DWORD err) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_Win32(__RFF_INTERNAL_FN_CALL FailureType::FailFast, err);
            }

            __RFF_INTERNAL_NORET_METHOD(_FailFast_NullAlloc)(__RFF_INTERNAL_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_Hr(__RFF_INTERNAL_FN_CALL FailureType::FailFast, E_OUTOFMEMORY);
            }

            __RFF_INTERNAL_NORET_METHOD(_FailFast_NtStatus)(__RFF_INTERNAL_FN_PARAMS NTSTATUS status) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_NtStatus(__RFF_INTERNAL_FN_CALL FailureType::FailFast, status);
            }

            __RFF_CONDITIONAL_METHOD(HRESULT, FailFast_IfFailed)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr) WI_NOEXCEPT
            {
                if (FAILED(hr))
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_Hr)(__RFF_CONDITIONAL_FN_CALL hr);
                }
                return hr;
            }

            __RFF_CONDITIONAL_METHOD(BOOL, FailFast_IfWin32BoolFalse)(__RFF_CONDITIONAL_FN_PARAMS BOOL ret) WI_NOEXCEPT
            {
                if (!ret)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_GetLastError)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
                return ret;
            }

            __RFF_CONDITIONAL_METHOD(DWORD, FailFast_IfWin32Error)(__RFF_CONDITIONAL_FN_PARAMS DWORD err) WI_NOEXCEPT
            {
                if (FAILED_WIN32(err))
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_Win32)(__RFF_CONDITIONAL_FN_CALL err);
                }
                return err;
            }

            __RFF_CONDITIONAL_METHOD(HANDLE, FailFast_IfHandleInvalid)(__RFF_CONDITIONAL_FN_PARAMS HANDLE handle) WI_NOEXCEPT
            {
                if (handle == INVALID_HANDLE_VALUE)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_GetLastError)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
                return handle;
            }

            __RFF_CONDITIONAL_METHOD(RESULT_NORETURN_NULL HANDLE, FailFast_IfHandleNull)(__RFF_CONDITIONAL_FN_PARAMS HANDLE handle) WI_NOEXCEPT
            {
                if (handle == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_GetLastError)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
                return handle;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __RFF_CONDITIONAL_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, FailFast_IfNullAlloc)(__RFF_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_NullAlloc)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
                return pointer;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __RFF_CONDITIONAL_TEMPLATE_METHOD(void, FailFast_IfNullAlloc)(__RFF_CONDITIONAL_FN_PARAMS const PointerT& pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_NullAlloc)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
            }

            __RFF_CONDITIONAL_METHOD(bool, FailFast_HrIf)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition) WI_NOEXCEPT
            {
                if (condition)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_Hr)(__RFF_CONDITIONAL_FN_CALL hr);
                }
                return condition;
            }

            __RFF_CONDITIONAL_METHOD(bool, FailFast_HrIfFalse)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition) WI_NOEXCEPT
            {
                if (!condition)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_Hr)(__RFF_CONDITIONAL_FN_CALL hr);
                }
                return condition;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __RFF_CONDITIONAL_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, FailFast_HrIfNull)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr, _Pre_maybenull_ PointerT pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_Hr)(__RFF_CONDITIONAL_FN_CALL hr);
                }
                return pointer;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __RFF_CONDITIONAL_TEMPLATE_METHOD(void, FailFast_HrIfNull)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr, _In_opt_ const PointerT& pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_Hr)(__RFF_CONDITIONAL_FN_CALL hr);
                }
            }

            __RFF_CONDITIONAL_METHOD(bool, FailFast_GetLastErrorIf)(__RFF_CONDITIONAL_FN_PARAMS bool condition) WI_NOEXCEPT
            {
                if (condition)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_GetLastError)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
                return condition;
            }

            __RFF_CONDITIONAL_METHOD(bool, FailFast_GetLastErrorIfFalse)(__RFF_CONDITIONAL_FN_PARAMS bool condition) WI_NOEXCEPT
            {
                if (!condition)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_GetLastError)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
                return condition;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __RFF_CONDITIONAL_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, FailFast_GetLastErrorIfNull)(__RFF_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_GetLastError)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
                return pointer;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __RFF_CONDITIONAL_TEMPLATE_METHOD(void, FailFast_GetLastErrorIfNull)(__RFF_CONDITIONAL_FN_PARAMS _In_opt_ const PointerT& pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_GetLastError)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
            }

            __RFF_CONDITIONAL_METHOD(NTSTATUS, FailFast_IfNtStatusFailed)(__RFF_CONDITIONAL_FN_PARAMS NTSTATUS status) WI_NOEXCEPT
            {
                if (FAILED_NTSTATUS(status))
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_NtStatus)(__RFF_CONDITIONAL_FN_CALL status);
                }
                return status;
            }

            __RFF_DIRECT_NORET_METHOD(void, FailFast_HrMsg)(__RFF_DIRECT_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__RFF_DIRECT_FN_CALL FailureType::FailFast, hr, formatString, argList);
            }

            __RFF_DIRECT_NORET_METHOD(void, FailFast_Win32Msg)(__RFF_DIRECT_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_Win32Msg(__RFF_DIRECT_FN_CALL FailureType::FailFast, err, formatString, argList);
            }

            __RFF_DIRECT_NORET_METHOD(void, FailFast_GetLastErrorMsg)(__RFF_DIRECT_FN_PARAMS _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_GetLastErrorMsg(__RFF_DIRECT_FN_CALL FailureType::FailFast, formatString, argList);
            }

            __RFF_DIRECT_NORET_METHOD(void, FailFast_NtStatusMsg)(__RFF_DIRECT_FN_PARAMS NTSTATUS status, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_NtStatusMsg(__RFF_DIRECT_FN_CALL FailureType::FailFast, status, formatString, argList);
            }

#ifdef WIL_ENABLE_EXCEPTIONS
            __RFF_DIRECT_NORET_METHOD(void, FailFast_CaughtExceptionMsg)(__RFF_DIRECT_FN_PARAMS _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_CaughtExceptionMsg(__RFF_DIRECT_FN_CALL FailureType::FailFast, formatString, argList);
            }
#endif

            __RFF_INTERNAL_NOINLINE_NORET_METHOD(_FailFast_HrMsg)(__RFF_INTERNAL_NOINLINE_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__RFF_INTERNAL_NOINLINE_FN_CALL FailureType::FailFast, hr, formatString, argList);
            }

            __RFF_INTERNAL_NOINLINE_NORET_METHOD(_FailFast_GetLastErrorMsg)(__RFF_INTERNAL_NOINLINE_FN_PARAMS _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_GetLastErrorMsg(__RFF_INTERNAL_NOINLINE_FN_CALL FailureType::FailFast, formatString, argList);
            }

            __RFF_INTERNAL_NOINLINE_NORET_METHOD(_FailFast_Win32Msg)(__RFF_INTERNAL_NOINLINE_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_Win32Msg(__RFF_INTERNAL_NOINLINE_FN_CALL FailureType::FailFast, err, formatString, argList);
            }

            __RFF_INTERNAL_NOINLINE_NORET_METHOD(_FailFast_NullAllocMsg)(__RFF_INTERNAL_NOINLINE_FN_PARAMS _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__RFF_INTERNAL_NOINLINE_FN_CALL FailureType::FailFast, E_OUTOFMEMORY, formatString, argList);
            }

            __RFF_INTERNAL_NOINLINE_NORET_METHOD(_FailFast_NtStatusMsg)(__RFF_INTERNAL_NOINLINE_FN_PARAMS NTSTATUS status, _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_NtStatusMsg(__RFF_INTERNAL_NOINLINE_FN_CALL FailureType::FailFast, status, formatString, argList);
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(HRESULT, FailFast_IfFailedMsg)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (FAILED(hr))
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_HrMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return hr;
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(BOOL, FailFast_IfWin32BoolFalseMsg)(__RFF_CONDITIONAL_FN_PARAMS BOOL ret, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (!ret)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_GetLastErrorMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return ret;
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(DWORD, FailFast_IfWin32ErrorMsg)(__RFF_CONDITIONAL_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (FAILED_WIN32(err))
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_Win32Msg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL err, formatString, argList);
                }
                return err;
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(HANDLE, FailFast_IfHandleInvalidMsg)(__RFF_CONDITIONAL_FN_PARAMS HANDLE handle, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (handle == INVALID_HANDLE_VALUE)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_GetLastErrorMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return handle;
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(RESULT_NORETURN_NULL HANDLE, FailFast_IfHandleNullMsg)(__RFF_CONDITIONAL_FN_PARAMS HANDLE handle, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (handle == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_GetLastErrorMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return handle;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, FailFast_IfNullAllocMsg)(__RFF_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_NullAllocMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL_ONLY, formatString, argList);
                }
                return pointer;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(void, FailFast_IfNullAllocMsg)(__RFF_CONDITIONAL_FN_PARAMS const PointerT& pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_NullAllocMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL_ONLY, formatString, argList);
                }
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(bool, FailFast_HrIfMsg)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_HrMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return condition;
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(bool, FailFast_HrIfFalseMsg)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (!condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_HrMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return condition;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, FailFast_HrIfNullMsg)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr, _Pre_maybenull_ PointerT pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_HrMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return pointer;
            }
                
            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(void, FailFast_HrIfNullMsg)(__RFF_CONDITIONAL_FN_PARAMS HRESULT hr, _In_opt_ const PointerT& pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_HrMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(bool, FailFast_GetLastErrorIfMsg)(__RFF_CONDITIONAL_FN_PARAMS bool condition, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_GetLastErrorMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return condition;
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(bool, FailFast_GetLastErrorIfFalseMsg)(__RFF_CONDITIONAL_FN_PARAMS bool condition, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (!condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_GetLastErrorMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return condition;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, FailFast_GetLastErrorIfNullMsg)(__RFF_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_GetLastErrorMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return pointer;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(void, FailFast_GetLastErrorIfNullMsg)(__RFF_CONDITIONAL_FN_PARAMS _In_opt_ const PointerT& pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_GetLastErrorMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(NTSTATUS, FailFast_IfNtStatusFailedMsg)(__RFF_CONDITIONAL_FN_PARAMS NTSTATUS status, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (FAILED_NTSTATUS(status))
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_NtStatusMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL status, formatString, argList);
                }
                return status;
            }

            __RFF_DIRECT_NORET_METHOD(void, FailFast_Unexpected)(__RFF_DIRECT_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_Hr(__RFF_DIRECT_FN_CALL FailureType::FailFast, E_UNEXPECTED);
            }

            __RFF_INTERNAL_NORET_METHOD(_FailFast_Unexpected)(__RFF_INTERNAL_FN_PARAMS_ONLY) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_Hr(__RFF_INTERNAL_FN_CALL FailureType::FailFast, E_UNEXPECTED);
            }

            __RFF_CONDITIONAL_METHOD(bool, FailFast_If)(__RFF_CONDITIONAL_FN_PARAMS bool condition) WI_NOEXCEPT
            {
                if (condition)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_Unexpected)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
                return condition;
            }

            __RFF_CONDITIONAL_METHOD(bool, FailFast_IfFalse)(__RFF_CONDITIONAL_FN_PARAMS bool condition) WI_NOEXCEPT
            {
                if (!condition)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_Unexpected)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
                return condition;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __RFF_CONDITIONAL_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, FailFast_IfNull)(__RFF_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_Unexpected)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
                return pointer;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __RFF_CONDITIONAL_TEMPLATE_METHOD(void, FailFast_IfNull)(__RFF_CONDITIONAL_FN_PARAMS _In_opt_ const PointerT& pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFast_Unexpected)(__RFF_CONDITIONAL_FN_CALL_ONLY);
                }
            }

            __RFF_DIRECT_NORET_METHOD(void, FailFast_UnexpectedMsg)(__RFF_DIRECT_FN_PARAMS _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                va_list argList;
                va_start(argList, formatString);
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__RFF_DIRECT_FN_CALL FailureType::FailFast, E_UNEXPECTED, formatString, argList);
            }

            __RFF_INTERNAL_NOINLINE_NORET_METHOD(_FailFast_UnexpectedMsg)(__RFF_INTERNAL_NOINLINE_FN_PARAMS _Printf_format_string_ PCSTR formatString, va_list argList) WI_NOEXCEPT
            {
                __RFF_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__RFF_INTERNAL_NOINLINE_FN_CALL FailureType::FailFast, E_UNEXPECTED, formatString, argList);
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(bool, FailFast_IfMsg)(__RFF_CONDITIONAL_FN_PARAMS bool condition, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_UnexpectedMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return condition;
            }

            __RFF_CONDITIONAL_NOINLINE_METHOD(bool, FailFast_IfFalseMsg)(__RFF_CONDITIONAL_FN_PARAMS bool condition, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (!condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_UnexpectedMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return condition;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, FailFast_IfNullMsg)(__RFF_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_UnexpectedMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return pointer;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __RFF_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(void, FailFast_IfNullMsg)(__RFF_CONDITIONAL_FN_PARAMS _In_opt_ const PointerT& pointer, _Printf_format_string_ PCSTR formatString, ...) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __RFF_CALL_INTERNAL_NOINLINE_METHOD(_FailFast_UnexpectedMsg)(__RFF_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
            }

            //*****************************************************************************
            // FailFast Immediate Macros
            //*****************************************************************************

            __RFF_DIRECT_NORET_METHOD(void, FailFastImmediate_Unexpected)() WI_NOEXCEPT
            {
                RESULT_RAISE_FAST_FAIL_EXCEPTION;
            }

            __RFF_INTERNAL_NORET_METHOD(_FailFastImmediate_Unexpected)() WI_NOEXCEPT
            {
                RESULT_RAISE_FAST_FAIL_EXCEPTION;
            }

            __RFF_CONDITIONAL_METHOD(HRESULT, FailFastImmediate_IfFailed)(HRESULT hr) WI_NOEXCEPT
            {
                if (FAILED(hr))
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFastImmediate_Unexpected)();
                }
                return hr;
            }

            __RFF_CONDITIONAL_METHOD(bool, FailFastImmediate_If)(bool condition) WI_NOEXCEPT
            {
                if (condition)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFastImmediate_Unexpected)();
                }
                return condition;
            }

            __RFF_CONDITIONAL_METHOD(bool, FailFastImmediate_IfFalse)(bool condition) WI_NOEXCEPT
            {
                if (!condition)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFastImmediate_Unexpected)();
                }
                return condition;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __RFF_CONDITIONAL_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, FailFastImmediate_IfNull)(_Pre_maybenull_ PointerT pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFastImmediate_Unexpected)();
                }
                return pointer;
            }

            template <__RFF_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __RFF_CONDITIONAL_TEMPLATE_METHOD(void, FailFastImmediate_IfNull)(_In_opt_ const PointerT& pointer) WI_NOEXCEPT
            {
                if (pointer == nullptr)
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFastImmediate_Unexpected)();
                }
            }

            __RFF_CONDITIONAL_METHOD(NTSTATUS, FailFastImmediate_IfNtStatusFailed)(NTSTATUS status) WI_NOEXCEPT
            {
                if (FAILED_NTSTATUS(status))
                {
                    __RFF_CALL_INTERNAL_METHOD(_FailFastImmediate_Unexpected)();
                }
                return status;
            }
        } // namespace __RFF_NS_NAME

        namespace __R_NS_NAME
        {
            //*****************************************************************************
            // Exception Macros
            //*****************************************************************************

#ifdef WIL_ENABLE_EXCEPTIONS
            __R_DIRECT_NORET_METHOD(void, Throw_Hr)(__R_DIRECT_FN_PARAMS HRESULT hr)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Hr(__R_DIRECT_FN_CALL FailureType::Exception, hr);
            }

            __R_DIRECT_NORET_METHOD(void, Throw_Win32)(__R_DIRECT_FN_PARAMS DWORD err)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Win32(__R_DIRECT_FN_CALL FailureType::Exception, err);
            }

            __R_DIRECT_NORET_METHOD(void, Throw_GetLastError)(__R_DIRECT_FN_PARAMS_ONLY)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_GetLastError(__R_DIRECT_FN_CALL FailureType::Exception);
            }

            __R_DIRECT_NORET_METHOD(void, Throw_NtStatus)(__R_DIRECT_FN_PARAMS NTSTATUS status)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_NtStatus(__R_DIRECT_FN_CALL FailureType::Exception, status);
            }

            __R_DIRECT_NORET_METHOD(void, Throw_CaughtException)(__R_DIRECT_FN_PARAMS_ONLY)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_CaughtException(__R_DIRECT_FN_CALL FailureType::Exception);
            }

            __R_INTERNAL_NORET_METHOD(_Throw_Hr)(__R_INTERNAL_FN_PARAMS HRESULT hr)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Hr(__R_INTERNAL_FN_CALL FailureType::Exception, hr);
            }

            __R_INTERNAL_NORET_METHOD(_Throw_GetLastError)(__R_INTERNAL_FN_PARAMS_ONLY)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_GetLastError(__R_INTERNAL_FN_CALL FailureType::Exception);
            }

            __R_INTERNAL_NORET_METHOD(_Throw_Win32)(__R_INTERNAL_FN_PARAMS DWORD err)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Win32(__R_INTERNAL_FN_CALL FailureType::Exception, err);
            }

            __R_INTERNAL_NORET_METHOD(_Throw_NullAlloc)(__R_INTERNAL_FN_PARAMS_ONLY)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Hr(__R_INTERNAL_FN_CALL FailureType::Exception, E_OUTOFMEMORY);
            }

            __R_INTERNAL_NORET_METHOD(_Throw_NtStatus)(__R_INTERNAL_FN_PARAMS NTSTATUS status)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_NtStatus(__R_INTERNAL_FN_CALL FailureType::Exception, status);
            }

            __R_CONDITIONAL_METHOD(HRESULT, Throw_IfFailed)(__R_CONDITIONAL_FN_PARAMS HRESULT hr)
            {
                if (FAILED(hr))
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_Hr)(__R_CONDITIONAL_FN_CALL hr);
                }
                return hr;
            }

            __R_CONDITIONAL_METHOD(BOOL, Throw_IfWin32BoolFalse)(__R_CONDITIONAL_FN_PARAMS BOOL ret)
            {
                if (!ret)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return ret;
            }

            __R_CONDITIONAL_METHOD(DWORD, Throw_IfWin32Error)(__R_CONDITIONAL_FN_PARAMS DWORD err)
            {
                if (FAILED_WIN32(err))
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_Win32)(__R_CONDITIONAL_FN_CALL err);
                }
                return err;
            }

            __R_CONDITIONAL_METHOD(HANDLE, Throw_IfHandleInvalid)(__R_CONDITIONAL_FN_PARAMS HANDLE handle)
            {
                if (handle == INVALID_HANDLE_VALUE)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return handle;
            }

            __R_CONDITIONAL_METHOD(RESULT_NORETURN_NULL HANDLE, Throw_IfHandleNull)(__R_CONDITIONAL_FN_PARAMS HANDLE handle)
            {
                if (handle == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return handle;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, Throw_IfNullAlloc)(__R_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer)
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_NullAlloc)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(void, Throw_IfNullAlloc)(__R_CONDITIONAL_FN_PARAMS const PointerT& pointer)
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_NullAlloc)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
            }

            __R_CONDITIONAL_METHOD(bool, Throw_HrIf)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition)
            {
                if (condition)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_Hr)(__R_CONDITIONAL_FN_CALL hr);
                }
                return condition;
            }

            __R_CONDITIONAL_METHOD(bool, Throw_HrIfFalse)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition)
            {
                if (!condition)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_Hr)(__R_CONDITIONAL_FN_CALL hr);
                }
                return condition;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, Throw_HrIfNull)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, _Pre_maybenull_ PointerT pointer)
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_Hr)(__R_CONDITIONAL_FN_CALL hr);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(void, Throw_HrIfNull)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, _In_opt_ const PointerT& pointer)
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_Hr)(__R_CONDITIONAL_FN_CALL hr);
                }
            }

            __R_CONDITIONAL_METHOD(bool, Throw_GetLastErrorIf)(__R_CONDITIONAL_FN_PARAMS bool condition)
            {
                if (condition)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return condition;
            }

            __R_CONDITIONAL_METHOD(bool, Throw_GetLastErrorIfFalse)(__R_CONDITIONAL_FN_PARAMS bool condition)
            {
                if (!condition)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return condition;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, Throw_GetLastErrorIfNull)(__R_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer)
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_TEMPLATE_METHOD(void, Throw_GetLastErrorIfNull)(__R_CONDITIONAL_FN_PARAMS _In_opt_ const PointerT& pointer)
            {
                if (pointer == nullptr)
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_GetLastError)(__R_CONDITIONAL_FN_CALL_ONLY);
                }
            }

            __R_CONDITIONAL_METHOD(NTSTATUS, Throw_IfNtStatusFailed)(__R_CONDITIONAL_FN_PARAMS NTSTATUS status)
            {
                if (FAILED_NTSTATUS(status))
                {
                    __R_CALL_INTERNAL_METHOD(_Throw_NtStatus)(__R_CONDITIONAL_FN_CALL status);
                }
                return status;
            }

            __R_DIRECT_NORET_METHOD(void, Throw_HrMsg)(__R_DIRECT_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, ...)
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__R_DIRECT_FN_CALL FailureType::Exception, hr, formatString, argList);
            }

            __R_DIRECT_NORET_METHOD(void, Throw_Win32Msg)(__R_DIRECT_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, ...)
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                wil::details::ReportFailure_Win32Msg(__R_DIRECT_FN_CALL FailureType::Exception, err, formatString, argList);
            }

            __R_DIRECT_NORET_METHOD(void, Throw_GetLastErrorMsg)(__R_DIRECT_FN_PARAMS _Printf_format_string_ PCSTR formatString, ...)
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                wil::details::ReportFailure_GetLastErrorMsg(__R_DIRECT_FN_CALL FailureType::Exception, formatString, argList);
            }

            __R_DIRECT_NORET_METHOD(void, Throw_NtStatusMsg)(__R_DIRECT_FN_PARAMS NTSTATUS status, _Printf_format_string_ PCSTR formatString, ...)
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                wil::details::ReportFailure_NtStatusMsg(__R_DIRECT_FN_CALL FailureType::Exception, status, formatString, argList);
            }

            __R_DIRECT_NORET_METHOD(void, Throw_CaughtExceptionMsg)(__R_DIRECT_FN_PARAMS _Printf_format_string_ PCSTR formatString, ...)
            {
                va_list argList;
                va_start(argList, formatString);
                __R_FN_LOCALS;
                wil::details::ReportFailure_CaughtExceptionMsg(__R_DIRECT_FN_CALL FailureType::Exception, formatString, argList);
            }

            __R_INTERNAL_NOINLINE_METHOD(_Throw_HrMsg)(__R_INTERNAL_NOINLINE_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, va_list argList)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__R_INTERNAL_NOINLINE_FN_CALL FailureType::Exception, hr, formatString, argList);
            }

            __R_INTERNAL_NOINLINE_METHOD(_Throw_GetLastErrorMsg)(__R_INTERNAL_NOINLINE_FN_PARAMS _Printf_format_string_ PCSTR formatString, va_list argList)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_GetLastErrorMsg(__R_INTERNAL_NOINLINE_FN_CALL FailureType::Exception, formatString, argList);
            }

            __R_INTERNAL_NOINLINE_METHOD(_Throw_Win32Msg)(__R_INTERNAL_NOINLINE_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, va_list argList)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_Win32Msg(__R_INTERNAL_NOINLINE_FN_CALL FailureType::Exception, err, formatString, argList);
            }

            __R_INTERNAL_NOINLINE_METHOD(_Throw_NullAllocMsg)(__R_INTERNAL_NOINLINE_FN_PARAMS _Printf_format_string_ PCSTR formatString, va_list argList)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_HrMsg(__R_INTERNAL_NOINLINE_FN_CALL FailureType::Exception, E_OUTOFMEMORY, formatString, argList);
            }

            __R_INTERNAL_NOINLINE_METHOD(_Throw_NtStatusMsg)(__R_INTERNAL_NOINLINE_FN_PARAMS NTSTATUS status, _Printf_format_string_ PCSTR formatString, va_list argList)
            {
                __R_FN_LOCALS;
                wil::details::ReportFailure_NtStatusMsg(__R_INTERNAL_NOINLINE_FN_CALL FailureType::Exception, status, formatString, argList);
            }

            __R_CONDITIONAL_NOINLINE_METHOD(HRESULT, Throw_IfFailedMsg)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (FAILED(hr))
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_HrMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return hr;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(BOOL, Throw_IfWin32BoolFalseMsg)(__R_CONDITIONAL_FN_PARAMS BOOL ret, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (!ret)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return ret;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(DWORD, Throw_IfWin32ErrorMsg)(__R_CONDITIONAL_FN_PARAMS DWORD err, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (FAILED_WIN32(err))
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_Win32Msg)(__R_CONDITIONAL_NOINLINE_FN_CALL err, formatString, argList);
                }
                return err;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(HANDLE, Throw_IfHandleInvalidMsg)(__R_CONDITIONAL_FN_PARAMS HANDLE handle, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (handle == INVALID_HANDLE_VALUE)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return handle;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(RESULT_NORETURN_NULL HANDLE, Throw_IfHandleNullMsg)(__R_CONDITIONAL_FN_PARAMS HANDLE handle, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (handle == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return handle;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, Throw_IfNullAllocMsg)(__R_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_NullAllocMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL_ONLY, formatString, argList);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(void, Throw_IfNullAllocMsg)(__R_CONDITIONAL_FN_PARAMS const PointerT& pointer, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_NullAllocMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL_ONLY, formatString, argList);
                }
            }

            __R_CONDITIONAL_NOINLINE_METHOD(bool, Throw_HrIfMsg)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_HrMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return condition;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(bool, Throw_HrIfFalseMsg)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, bool condition, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (!condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_HrMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return condition;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, Throw_HrIfNullMsg)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, _Pre_maybenull_ PointerT pointer, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_HrMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(void, Throw_HrIfNullMsg)(__R_CONDITIONAL_FN_PARAMS HRESULT hr, _In_opt_ const PointerT& pointer, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_HrMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL hr, formatString, argList);
                }
            }

            __R_CONDITIONAL_NOINLINE_METHOD(bool, Throw_GetLastErrorIfMsg)(__R_CONDITIONAL_FN_PARAMS bool condition, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return condition;
            }

            __R_CONDITIONAL_NOINLINE_METHOD(bool, Throw_GetLastErrorIfFalseMsg)(__R_CONDITIONAL_FN_PARAMS bool condition, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (!condition)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return condition;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_NOT_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(RESULT_NORETURN_NULL PointerT, Throw_GetLastErrorIfNullMsg)(__R_CONDITIONAL_FN_PARAMS _Pre_maybenull_ PointerT pointer, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
                return pointer;
            }

            template <__R_CONDITIONAL_PARTIAL_TEMPLATE typename PointerT, __R_ENABLE_IF_IS_CLASS(PointerT)>
            __R_CONDITIONAL_NOINLINE_TEMPLATE_METHOD(void, Throw_GetLastErrorIfNullMsg)(__R_CONDITIONAL_FN_PARAMS _In_opt_ const PointerT& pointer, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (pointer == nullptr)
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_GetLastErrorMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL formatString, argList);
                }
            }

            __R_CONDITIONAL_NOINLINE_METHOD(NTSTATUS, Throw_IfNtStatusFailedMsg)(__R_CONDITIONAL_FN_PARAMS NTSTATUS status, _Printf_format_string_ PCSTR formatString, ...)
            {
                if (FAILED_NTSTATUS(status))
                {
                    va_list argList;
                    va_start(argList, formatString);
                    __R_CALL_INTERNAL_NOINLINE_METHOD(_Throw_NtStatusMsg)(__R_CONDITIONAL_NOINLINE_FN_CALL status, formatString, argList);
                }
                return status;
            }
#endif // WIL_ENABLE_EXCEPTIONS

        }   // __R_NS_NAME namespace
    }   // details namespace
    /// @endcond
   

    //*****************************************************************************
    // Error Handling Policies to switch between error-handling style
    //*****************************************************************************
    // The following policies are used as template policies for components that can support exception, fail-fast, and
    // error-code based modes.

    // Use for classes which should return HRESULTs as their error-handling policy
    struct err_returncode_policy
    {
        typedef HRESULT result;

        __forceinline static HRESULT Win32BOOL(BOOL fReturn) { RETURN_IF_WIN32_BOOL_FALSE(fReturn); return S_OK; }
        __forceinline static HRESULT Win32Handle(HANDLE h, _Out_ HANDLE *ph) { *ph = h; RETURN_IF_HANDLE_NULL(h); return S_OK; }
        _Post_satisfies_(return == hr)
        __forceinline static HRESULT HResult(HRESULT hr) { RETURN_HR(hr); }
        __forceinline static HRESULT LastError() { RETURN_LAST_ERROR(); }
        __forceinline static HRESULT LastErrorIfFalse(bool condition) { if (!condition) { RETURN_LAST_ERROR(); } return S_OK; }
        _Post_satisfies_(return == S_OK)
        __forceinline static HRESULT OK() { return S_OK; }
    };

    // Use for classes which fail-fast on errors
    struct err_failfast_policy
    {
        typedef _Return_type_success_(true) void result;
        __forceinline static result Win32BOOL(BOOL fReturn) { FAIL_FAST_IF_WIN32_BOOL_FALSE(fReturn); }
        __forceinline static result Win32Handle(HANDLE h, _Out_ HANDLE *ph) { *ph = h; FAIL_FAST_IF_HANDLE_NULL(h); }
        _When_(FAILED(hr), _Analysis_noreturn_)
        __forceinline static result HResult(HRESULT hr) { FAIL_FAST_IF_FAILED(hr); }
        __forceinline static result LastError() { FAIL_FAST_LAST_ERROR(); }
        __forceinline static result LastErrorIfFalse(bool condition) { if (!condition) { FAIL_FAST_LAST_ERROR(); } }
        __forceinline static result OK() {}
    };

#ifdef WIL_ENABLE_EXCEPTIONS
    // Use for classes which should return through exceptions as their error-handling policy
    struct err_exception_policy
    {
        typedef _Return_type_success_(true) void result;
        __forceinline static result Win32BOOL(BOOL fReturn) { THROW_IF_WIN32_BOOL_FALSE(fReturn); }
        __forceinline static result Win32Handle(HANDLE h, _Out_ HANDLE *ph) { *ph = h; THROW_IF_HANDLE_NULL(h); }
        _When_(FAILED(hr), _Analysis_noreturn_)
        __forceinline static result HResult(HRESULT hr) { THROW_IF_FAILED(hr); }
        __forceinline static result LastError() { THROW_LAST_ERROR(); }
        __forceinline static result LastErrorIfFalse(bool condition) { if (!condition) { THROW_LAST_ERROR(); } }
        __forceinline static result OK() {}
    };
#endif

} // namespace wil

#pragma warning(pop)
