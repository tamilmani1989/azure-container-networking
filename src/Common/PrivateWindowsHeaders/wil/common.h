// Windows Internal Libraries (wil)
//
// wil Usage Guidelines:
// https://microsoft.sharepoint.com/teams/osg_development/Shared%20Documents/Windows%20Internal%20Libraries%20for%20C++%20Usage%20Guide.docx?web=1
//
// wil Discussion Alias (wildisc):
// http://idwebelements/GroupManagement.aspx?Group=wildisc&Operation=join  (one-click join)
//
//! @file
//! Dependency-free straight c++ helpers, macros and type traits.
//! This file is always included implicitly by all wil headers.  It contains helpers that are broadly applicable without
//! requiring any dependencies.  Including this file naturally includes wistd_type_traits.h to supply functionality such
//! as `wistd::move()` to exception-free code.

#pragma once
#ifndef __cplusplus
#error This file is designed for C++ consumers only
#endif
#pragma warning(push)
#pragma warning(disable:4714)    // __forceinline not honored

// DO NOT add *any* further includes to this file -- there should be no dependencies from its usage
#include "wistd_type_traits.h"


// Some SAL remapping / decoration to better support Doxygen.  Macros that look like function calls can
// confuse Doxygen when they are used to decorate a function or variable.  We simplify some of these to
// basic macros without the function for common use cases.
/// @cond
#define _Success_return_ _Success_(return)
#define _Success_true_ _Success_(true)
#define __declspec_noinline_ __declspec(noinline)
#define __declspec_selectany_ __declspec(selectany)
/// @endcond

#if defined(_CPPUNWIND) && !defined(WIL_SUPPRESS_EXCEPTIONS)
//! This define is automatically set when exceptions are enabled within wil.
//! It is automatically defined when your code is compiled with exceptions enabled (via checking for the built-in
//! _CPPUNWIND flag) unless you explicitly define WIL_SUPPRESS_EXCEPTIONS ahead of including your first wil
//! header.  All exception-based WIL methods and classes are included behind:
//! ~~~~
//! #ifdef WIL_ENABLE_EXCEPTIONS
//! // code
//! #endif
//! ~~~~
//! This enables exception-free code to directly include WIL headers without worrying about exception-based
//! routines suddenly becoming available.
#define WIL_ENABLE_EXCEPTIONS
#endif
/// @endcond

/// @cond
#if defined(WIL_EXCEPTION_MODE)
static_assert(WIL_EXCEPTION_MODE <= 2, "Invalid exception mode");
#elif !defined(WIL_LOCK_EXCEPTION_MODE)
#define WIL_EXCEPTION_MODE 0            // default, can link exception-based and non-exception based libraries together
#pragma detect_mismatch("ODR_violation_WIL_EXCEPTION_MODE_mismatch", "0")
#elif defined(WIL_ENABLE_EXCEPTIONS)
#define WIL_EXCEPTION_MODE 1            // new code optimization:  ONLY support linking libraries together that have exceptions enabled
#pragma detect_mismatch("ODR_violation_WIL_EXCEPTION_MODE_mismatch", "1")
#else
#define WIL_EXCEPTION_MODE 2            // old code optimization:  ONLY support linking libraries that are NOT using exceptions
#pragma detect_mismatch("ODR_violation_WIL_EXCEPTION_MODE_mismatch", "2")
#endif

#if WIL_EXCEPTION_MODE == 1 && !defined(WIL_ENABLE_EXCEPTIONS)
#error Must enable exceptions when WIL_EXCEPTION_MODE == 1
#endif

// block for documentation only
#if defined(WIL_DOXYGEN)
/** This define can be explicitly set to disable exception usage within wil.
Normally this define is never needed as the WIL_ENABLE_EXCEPTIONS macro is enabled automatically by looking
at _CPPUNWIND.  If your code compiles with exceptions enabled, but does not want to enable the exception-based
classes and methods from WIL, define this macro ahead of including the first WIL header. */
#define WIL_SUPPRESS_EXCEPTIONS

/** This define can be explicitly set to lock the process exception mode to WIL_ENABLE_EXCEPTIONS.
Locking the exception mode provides optimizations to exception barriers, staging hooks and DLL load costs as it eliminates the need to
do copy-on-write initialization of various funciton pointers and the necessary indirection that's done within WIL to avoid ODR violations
when linking libraries together with different exception handling semantics. */
#define WIL_LOCK_EXCEPTION_MODE

/** This define explicit sets the exception mode for the process to control optimizations.
Three exception modes are avialable:
0)  This is the default.  This enables a binary to link both exception-based and non-exception based libraries together that
    use WIL.  This adds overhead to exception barriers, DLL copy on write pages and indirection through function pointers to avoid ODR
    violations when linking libraries together with different exception handling semantics.
1)  Prefer this setting when it can be used.  This locks the binary to only supporting libraries which were built with exceptions enabled.
2)  This locks the binary to libraries built without exceptions. */
#define WIL_EXCEPTION_MODE
#endif


//! @defgroup macrobuilding Macro Composition
//! The following macros are building blocks primarily intended for authoring other macros.
//! @{

//! Re-state a macro value (indirection for composition)
#define WI_FLATTEN(...)                     __VA_ARGS__

/// @cond
#define __WI_PASTE_imp(a, b)                a##b
/// @endcond
//! This macro is for use in other macros to paste two tokens together, such as a constant and the __LINE__ macro.
#define WI_PASTE(a, b)                      __WI_PASTE_imp(a, b)

/// @cond
#define __WI_ARGS_COUNT1(A0, A1, A2, A3, A4, A5, A6, A7, A8, A9, A10, A11, A12, A13, A14, A15, A16, A17, A18, A19, A20, A21, A22, A23, A24, A25, A26, A27, A28, A29, \
                         A30, A31, A32, A33, A34, A35, A36, A37, A38, A39, A40, A41, A42, A43, A44, A45, A46, A47, A48, A49, A50, A51, A52, A53, A54, A55, A56, A57, A58, A59, \
                         A60, A61, A62, A63, A64, A65, A66, A67, A68, A69, A70, A71, A72, A73, A74, A75, A76, A77, A78, A79, A80, A81, A82, A83, A84, A85, A86, A87, A88, A89, \
                         A90, A91, A92, A93, A94, A95, A96, A97, A98, A99, count, ...) count
#define __WI_ARGS_COUNT0(...) WI_FLATTEN(__WI_ARGS_COUNT1(__VA_ARGS__, 99, 98, 97, 96, 95, 94, 93, 92, 91, 90, 89, 88, 87, 86, 85, 84, 83, 82, 81, 80, \
                         79, 78, 77, 76, 75, 74, 73, 72, 71, 70, 69, 68, 67, 66, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 50,  49, 48, 47, 46, 45, 44, 43, 42, 41, 40, \
                         39, 38, 37, 36, 35, 34, 33, 32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0))
#define __WI_ARGS_COUNT_PREFIX(...) 0, __VA_ARGS__
/// @endcond
//! This variadic macro returns the number of arguments passed to it (up to 99).
#define WI_ARGS_COUNT(...) __WI_ARGS_COUNT0(__WI_ARGS_COUNT_PREFIX(__VA_ARGS__))

/// @cond
#define __WI_FOR_imp0( fn)
#define __WI_FOR_imp1( fn, arg, ...) __WI_FOR_impN( 0, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp2( fn, arg, ...) __WI_FOR_impN( 1, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp3( fn, arg, ...) __WI_FOR_impN( 2, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp4( fn, arg, ...) __WI_FOR_impN( 3, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp5( fn, arg, ...) __WI_FOR_impN( 4, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp6( fn, arg, ...) __WI_FOR_impN( 5, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp7( fn, arg, ...) __WI_FOR_impN( 6, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp8( fn, arg, ...) __WI_FOR_impN( 7, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp9( fn, arg, ...) __WI_FOR_impN( 8, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp10(fn, arg, ...) __WI_FOR_impN( 9, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp11(fn, arg, ...) __WI_FOR_impN(10, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp12(fn, arg, ...) __WI_FOR_impN(11, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp13(fn, arg, ...) __WI_FOR_impN(12, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp14(fn, arg, ...) __WI_FOR_impN(13, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp15(fn, arg, ...) __WI_FOR_impN(14, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp16(fn, arg, ...) __WI_FOR_impN(15, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp17(fn, arg, ...) __WI_FOR_impN(16, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp18(fn, arg, ...) __WI_FOR_impN(17, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp19(fn, arg, ...) __WI_FOR_impN(18, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp20(fn, arg, ...) __WI_FOR_impN(19, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp21(fn, arg, ...) __WI_FOR_impN(20, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp22(fn, arg, ...) __WI_FOR_impN(21, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp23(fn, arg, ...) __WI_FOR_impN(22, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp24(fn, arg, ...) __WI_FOR_impN(23, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp25(fn, arg, ...) __WI_FOR_impN(24, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp26(fn, arg, ...) __WI_FOR_impN(25, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp27(fn, arg, ...) __WI_FOR_impN(26, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp28(fn, arg, ...) __WI_FOR_impN(27, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp29(fn, arg, ...) __WI_FOR_impN(28, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp30(fn, arg, ...) __WI_FOR_impN(29, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp31(fn, arg, ...) __WI_FOR_impN(30, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp32(fn, arg, ...) __WI_FOR_impN(31, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp33(fn, arg, ...) __WI_FOR_impN(32, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp34(fn, arg, ...) __WI_FOR_impN(33, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp35(fn, arg, ...) __WI_FOR_impN(34, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp36(fn, arg, ...) __WI_FOR_impN(35, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp37(fn, arg, ...) __WI_FOR_impN(36, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp38(fn, arg, ...) __WI_FOR_impN(37, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp39(fn, arg, ...) __WI_FOR_impN(38, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp40(fn, arg, ...) __WI_FOR_impN(39, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp41(fn, arg, ...) __WI_FOR_impN(40, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp42(fn, arg, ...) __WI_FOR_impN(41, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp43(fn, arg, ...) __WI_FOR_impN(42, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp44(fn, arg, ...) __WI_FOR_impN(43, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp45(fn, arg, ...) __WI_FOR_impN(44, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp46(fn, arg, ...) __WI_FOR_impN(45, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp47(fn, arg, ...) __WI_FOR_impN(46, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp48(fn, arg, ...) __WI_FOR_impN(47, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp49(fn, arg, ...) __WI_FOR_impN(48, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp50(fn, arg, ...) __WI_FOR_impN(49, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp51(fn, arg, ...) __WI_FOR_impN(50, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp52(fn, arg, ...) __WI_FOR_impN(51, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp53(fn, arg, ...) __WI_FOR_impN(52, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp54(fn, arg, ...) __WI_FOR_impN(53, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp55(fn, arg, ...) __WI_FOR_impN(54, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp56(fn, arg, ...) __WI_FOR_impN(55, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp57(fn, arg, ...) __WI_FOR_impN(56, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp58(fn, arg, ...) __WI_FOR_impN(57, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp59(fn, arg, ...) __WI_FOR_impN(58, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp60(fn, arg, ...) __WI_FOR_impN(59, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp61(fn, arg, ...) __WI_FOR_impN(60, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp62(fn, arg, ...) __WI_FOR_impN(61, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp63(fn, arg, ...) __WI_FOR_impN(62, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp64(fn, arg, ...) __WI_FOR_impN(63, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp65(fn, arg, ...) __WI_FOR_impN(64, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp66(fn, arg, ...) __WI_FOR_impN(65, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp67(fn, arg, ...) __WI_FOR_impN(66, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp68(fn, arg, ...) __WI_FOR_impN(67, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp69(fn, arg, ...) __WI_FOR_impN(68, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp70(fn, arg, ...) __WI_FOR_impN(69, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp71(fn, arg, ...) __WI_FOR_impN(70, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp72(fn, arg, ...) __WI_FOR_impN(71, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp73(fn, arg, ...) __WI_FOR_impN(72, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp74(fn, arg, ...) __WI_FOR_impN(73, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp75(fn, arg, ...) __WI_FOR_impN(74, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp76(fn, arg, ...) __WI_FOR_impN(75, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp77(fn, arg, ...) __WI_FOR_impN(76, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp78(fn, arg, ...) __WI_FOR_impN(77, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp79(fn, arg, ...) __WI_FOR_impN(78, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp80(fn, arg, ...) __WI_FOR_impN(79, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp81(fn, arg, ...) __WI_FOR_impN(80, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp82(fn, arg, ...) __WI_FOR_impN(81, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp83(fn, arg, ...) __WI_FOR_impN(82, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp84(fn, arg, ...) __WI_FOR_impN(83, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp85(fn, arg, ...) __WI_FOR_impN(84, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp86(fn, arg, ...) __WI_FOR_impN(85, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp87(fn, arg, ...) __WI_FOR_impN(86, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp88(fn, arg, ...) __WI_FOR_impN(87, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp89(fn, arg, ...) __WI_FOR_impN(88, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp90(fn, arg, ...) __WI_FOR_impN(89, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp91(fn, arg, ...) __WI_FOR_impN(90, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp92(fn, arg, ...) __WI_FOR_impN(91, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp93(fn, arg, ...) __WI_FOR_impN(92, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp94(fn, arg, ...) __WI_FOR_impN(93, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp95(fn, arg, ...) __WI_FOR_impN(94, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp96(fn, arg, ...) __WI_FOR_impN(95, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp97(fn, arg, ...) __WI_FOR_impN(96, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp98(fn, arg, ...) __WI_FOR_impN(97, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_imp99(fn, arg, ...) __WI_FOR_impN(98, fn, arg, fn, __VA_ARGS__)
#define __WI_FOR_impN(n, fn, arg, ...) \
    fn(arg) \
    WI_PASTE(__WI_FOR_imp, n) WI_FLATTEN((__VA_ARGS__))
#define __WI_FOR_imp(n, fnAndArgs)  WI_PASTE(__WI_FOR_imp, n) fnAndArgs
/// @endcond
//! Iterates through each of the given arguments invoking the specified macro against each one.
#define WI_FOREACH(fn, ...) __WI_FOR_imp(WI_ARGS_COUNT(__VA_ARGS__), (fn, __VA_ARGS__))

//! Dispatches a single macro name to separate macros based on the number of arguments passed to it.
#define WI_MACRO_DISPATCH(name, ...) WI_PASTE(WI_PASTE(name, WI_ARGS_COUNT(__VA_ARGS__)), (__VA_ARGS__))

//! @} // Macro composition helpers


//! @defgroup bitwise Bitwise Inspection and Manipulation
//! Bitwise helpers to improve readability and reduce the error rate of bitwise operations.
//! Several macros have been constructed to assist with bitwise inspection and manipulation.  These macros exist
//! for two primary purposes:
//!
//! 1. To improve the readability of bitwise comparisons and manipulation.
//!
//!    The macro names are the more concise, readable form of what's being done and do not require that any flags
//!    or variables be specified multiple times for the comparisons.
//!
//! 2. To reduce the error rate associated with bitwise operations.
//!
//!    The readability improvements naturally lend themselves to this by cutting down the number of concepts.
//!    Using `WI_IS_FLAG_SET(var, MyEnum::Flag)` rather than `((var & MyEnum::Flag) == MyEnum::Flag)` removes the comparison 
//!    operator and repetition in the flag value.
//!
//!    Additionally, these macros separate single flag operations (which tend to be the most common) from multi-flag 
//!    operations so that compile-time errors are generated for bitwise operations which are likely incorrect,
//!    such as:  `WI_IS_FLAG_SET(var, MyEnum::None)` or `WI_IS_FLAG_SET(var, MyEnum::ValidMask)`.
//! 
//! Note that the single flag helpers should be used when a compile-time constant single flag is being manipulated.  These
//! helpers provide compile-time errors on misuse and should be preferred over the multi-flag helpers.  The multi-flag helpers
//! should be used when multiple flags are being used simultaneously or when the flag values are not compile-time constants.
//!
//! Common example usage (manipulation of flag variables):
//! ~~~~
//! WI_SET_FLAG(m_flags, MyFlags::Foo);                                 // Set a single flag in the given variable
//! WI_SET_ALL_FLAGS(m_flags, MyFlags::Foo | MyFlags::Bar);             // Set one or more flags
//! WI_CLEAR_FLAG_IF(m_flags, MyFlags::Bar, isBarClosed);               // Conditionally clear a single flag based upon a bool
//! WI_CLEAR_ALL_FLAGS(m_flags, MyFlags::Foo | MyFlags::Bar);           // Clear one or more flags from the given variable
//! WI_TOGGLE_FLAG(m_flags, MyFlags::Foo);                              // Toggle (change to the opposite value) a single flag
//! WI_UPDATE_FLAG(m_flags, MyFlags::Bar, isBarClosed);                 // Sets or Clears a single flag from the given variable based upon a bool value
//! WI_UPDATE_FLAGS_IN_MASK(m_flags, flagsMask, newFlagValues);         // Sets or Clears the flags in flagsMask to the masked values from newFlagValues
//! ~~~~
//! Common example usage (inspection of flag variables):
//! ~~~~
//! if (WI_IS_FLAG_SET(m_flags, MyFlags::Foo))                          // Is a single flag set in the given variable?
//! if (WI_IS_ANY_FLAG_SET(m_flags, MyFlags::Foo | MyFlags::Bar))       // Is at least one flag from the given mask set?
//! if (WI_ARE_ALL_FLAGS_CLEAR(m_flags, MyFlags::Foo | MyFlags::Bar))   // Are all flags in the given list clear?
//! if (WI_IS_SINGLE_FLAG_SET(m_flags))                                 // Is *exactly* one flag set in the given variable?
//! ~~~~
//! @{

// block for documentation only
#ifdef WIL_DOXYGEN

//! Allows assigning a set of flags to a compile-time constant bitwise or combination of enum class flag.
//! Enum flags are typically constructed by DEFINE_ENUM_FLAG_OPERATORS which declares runtime functions to combine
//! flag values.  This function doesn't allow assigning a set of flags to a compile-time constant.  This
//! macro can be used to rectify that by allowing enum class flags to be combined (with a comma separating them
//! instead of the '|' operator).
//! ~~~~
//! const EnumType flags = WI_COMPILETIME_COMBINE_FLAGS(EnumType::Value1, EnumType::Value2);
//! ~~~~
#define WI_COMPILETIME_COMBINE_FLAGS(...)

//! Validates that exactly ONE bit is set in compile-time constant `flag` or a compilation error is produced.
#define WI_STATIC_ASSERT_SINGLE_BIT_SET(flag)

//! @name Bitwise modification macros
//! @{

//! Set multiple bitflags specified by `flags` in the variable `var`.
#define WI_SET_ALL_FLAGS(var, flags)
//! Set a single compile-time constant `flag` in the variable `var`.
#define WI_SET_FLAG(var, flag)
//! Conditionally sets a single compile-time constant `flag` in the variable `var` only if `condition` is true.
#define WI_SET_FLAG_IF(var, flag, condition)
//! Conditionally sets a single compile-time constant `flag` in the variable `var` only if `condition` is false.
#define WI_SET_FLAG_IF_FALSE(var, flag, condition)

//! Clear multiple bitflags specified by `flags` from the variable `var`.
#define WI_CLEAR_ALL_FLAGS(var, flags)
//! Clear a single compile-time constant `flag` from the variable `var`.
#define WI_CLEAR_FLAG(var, flag)
//! Conditionally clear a single compile-time constant `flag` in the variable `var` only if `condition` is true.
#define WI_CLEAR_FLAG_IF(var, flag, condition)
//! Conditionally clear a single compile-time constant `flag` in the variable `var` only if `condition` is false.
#define WI_CLEAR_FLAG_IF_FALSE(var, flag, condition)

//! Changes a single compile-time constant `flag` in the variable `var` to be set if `isFlagSet` is true or cleared if `isFlagSet` is false.
#define WI_UPDATE_FLAG(var, flag, isFlagSet)
//! Changes only the flags specified by `flagsMask` in the variable `var` to match the corresponding flags in `newFlags`.
#define WI_UPDATE_FLAGS_IN_MASK(var, flagsMask, newFlags)

//! Toggles (XOR the value) of multiple bitflags specified by `flags` in the variable `var`.
#define WI_TOGGLE_ALL_FLAGS(var, flags)
//! Toggles (XOR the value) of a single compile-time constant `flag` in the variable `var`.
#define WI_TOGGLE_FLAG(var, flag)
//! @}      // bitwise modification helpers

//! @name Bitwise inspection macros
//! @{

//! Evaluates as true if every bitflag specified in `flags` is set within `val`.
#define WI_ARE_ALL_FLAGS_SET(val, flags)
//! Evaluates as true if one or more bitflags specified in `flags` are set within `val`.
#define WI_IS_ANY_FLAG_SET(val, flags)
//! Evaluates as true if a single compile-time constant `flag` is set within `val`.
#define WI_IS_FLAG_SET(val, flag)

//! Evaluates as true if every bitflag specified in `flags` is clear within `val`.
#define WI_ARE_ALL_FLAGS_CLEAR(val, flags)
//! Evaluates as true if one or more bitflags specified in `flags` are clear within `val`.
#define WI_IS_ANY_FLAG_CLEAR(val, flags)
//! Evaluates as true if a single compile-time constant `flag` is clear within `val`.
#define WI_IS_FLAG_CLEAR(val, flag)

//! Evaluates as true if exactly one bit (any bit) is set within `val`.
#define WI_IS_SINGLE_FLAG_SET(val)
//! Evaluates as true if exactly one bit from within the specified `mask` is set within `val`.
#define WI_IS_SINGLE_FLAG_SET_IN_MASK(val, mask)
//! Evaluates as true if exactly one bit (any bit) is set within `val` or if there are no bits set within `val`.
#define WI_IS_CLEAR_OR_SINGLE_FLAG_SET(val)
//! Evaluates as true if exactly one bit from within the specified `mask` is set within `val` or if there are no bits from `mask` set within `val`.
#define WI_IS_CLEAR_OR_SINGLE_FLAG_SET_IN_MASK(val, mask)
//! @}

//! This define can be explicitly set to enable usage of PascalCase() macro names for bitwise operations.
//! Defining WIL_SUPPORT_BITOPERATION_PASCAL_NAMES ahead of including a WIL header will enable the PascalCase name variants
//! for all of the bitwise test and manipulation macros.  For example, when this define is set, you can use `SetFlag(var, flag)`
//! in addition to the standard WIL macro name `WI_SET_FLAG(var, flag)`.
//!
//! These names are not enabled by default given the likelihood of a macro name like 'SetFlag' conflicting with existing code.
//! Traditionally shell has enabled these fairly natural names through the cstock header, so they are provided as an option for teams 
//! to enable.
#define WIL_SUPPORT_BITOPERATION_PASCAL_NAMES

//! @name PascalNames for bitwise helper macros
//! These are not enabled by default (see @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES for more information).
//! @{

//! Alias for @ref WI_SET_ALL_FLAGS when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define SetAllFlags(var, flags)
//! Alias for @ref WI_SET_FLAG when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define SetFlag(var, flag)
//! Alias for @ref WI_SET_FLAG_IF when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define SetFlagIf(var, flag, condition)
//! Alias for @ref WI_SET_FLAG_IF_FALSE when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define SetFlagIfFalse(var, flag, condition)

//! Alias for @ref WI_CLEAR_ALL_FLAGS when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define ClearAllFlags(var, flags)
//! Alias for @ref WI_CLEAR_FLAG when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define ClearFlag(var, flag)
//! Alias for @ref WI_CLEAR_FLAG_IF when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define ClearFlagIf(var, flag, condition)
//! Alias for @ref WI_CLEAR_FLAG_IF_FALSE when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define ClearFlagIfFalse(var, flag, condition)

//! Alias for @ref WI_UPDATE_FLAG when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define UpdateFlag(var, flag, isFlagSet)
//! Alias for @ref WI_UPDATE_FLAGS_IN_MASK when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define UpdateFlagsInMask(var, flagsMask, newFlags)

//! Alias for @ref WI_TOGGLE_ALL_FLAGS when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define ToggleAllFlags(var, flags)
//! Alias for @ref WI_TOGGLE_FLAG when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define ToggleFlag(var, flag)

//! Alias for @ref WI_ARE_ALL_FLAGS_SET when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define AreAllFlagsSet(val, flags)
//! Alias for @ref WI_IS_ANY_FLAG_SET when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define IsAnyFlagSet(val, flags)
//! Alias for @ref WI_IS_FLAG_SET when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define IsFlagSet(val, flag)

//! Alias for @ref WI_ARE_ALL_FLAGS_CLEAR when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define AreAllFlagsClear(val, flags)
//! Alias for @ref WI_IS_ANY_FLAG_CLEAR when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define IsAnyFlagClear(val, flags)
//! Alias for @ref WI_IS_FLAG_CLEAR when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define IsFlagClear(val, flag)

//! Alias for @ref WI_IS_SINGLE_FLAG_SET when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define IsSingleFlagSet(val)
//! Alias for @ref WI_IS_SINGLE_FLAG_SET_IN_MASK when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define IsSingleFlagSetInMask(val, mask)
//! Alias for @ref WI_IS_CLEAR_OR_SINGLE_FLAG_SET when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define IsClearOrSingleFlagSet(val)
//! Alias for @ref WI_IS_CLEAR_OR_SINGLE_FLAG_SET_IN_MASK when @ref WIL_SUPPORT_BITOPERATION_PASCAL_NAMES is specified.
#define IsClearOrSingleFlagSetInMask(val, mask)
//! @}  // Alternate PascalNames for bitwise helpers

#else // WIL_DOXYGEN

#define __WI_COMPILETIME_COMBINE_FLAGS1(a1) (a1)
#define __WI_COMPILETIME_COMBINE_FLAGS2(a1, a2) static_cast<decltype((a1) | (a2))>( \
    static_cast<::wil::integral_from_enum<decltype(a1)>>(a1) | \
    static_cast<::wil::integral_from_enum<decltype(a2)>>(a2))
#define __WI_COMPILETIME_COMBINE_FLAGS3(a1, a2, a3) static_cast<decltype((a1) | (a2) | (a3))>( \
    static_cast<::wil::integral_from_enum<decltype(a1)>>(a1) | \
    static_cast<::wil::integral_from_enum<decltype(a2)>>(a2) | \
    static_cast<::wil::integral_from_enum<decltype(a3)>>(a3))
#define __WI_COMPILETIME_COMBINE_FLAGS4(a1, a2, a3, a4) static_cast<decltype((a1) | (a2) | (a3) | (a4))>( \
    static_cast<::wil::integral_from_enum<decltype(a1)>>(a1) | \
    static_cast<::wil::integral_from_enum<decltype(a2)>>(a2) | \
    static_cast<::wil::integral_from_enum<decltype(a3)>>(a3) | \
    static_cast<::wil::integral_from_enum<decltype(a4)>>(a4))
#define __WI_COMPILETIME_COMBINE_FLAGS5(a1, a2, a3, a4, a5) static_cast<decltype((a1) | (a2) | (a3) | (a4) | (a5))>( \
    static_cast<::wil::integral_from_enum<decltype(a1)>>(a1) | \
    static_cast<::wil::integral_from_enum<decltype(a2)>>(a2) | \
    static_cast<::wil::integral_from_enum<decltype(a3)>>(a3) | \
    static_cast<::wil::integral_from_enum<decltype(a4)>>(a4) | \
    static_cast<::wil::integral_from_enum<decltype(a5)>>(a5))

#define WI_COMPILETIME_COMBINE_FLAGS(...)                   WI_MACRO_DISPATCH(__WI_COMPILETIME_COMBINE_FLAGS, __VA_ARGS__)

#define WI_STATIC_ASSERT_SINGLE_BIT_SET(flag)               static_cast<decltype(flag)>(::wil::details::verify_single_flag_helper<static_cast<unsigned long long>(flag)>::value)

#define  WI_ENUM_VALUE(val)                                 static_cast<::wil::integral_from_enum<decltype(val)>>(val)

// Macros to manipulate flag bitmasks

#define WI_SET_ALL_FLAGS(var, flags)                        ((var) |= (flags))
#define WI_SET_FLAG(var, flag)                              WI_SET_ALL_FLAGS(var, WI_STATIC_ASSERT_SINGLE_BIT_SET(flag))
#define WI_SET_FLAG_IF(var, flag, condition)                do { if (wil::verify_bool(condition)) { WI_SET_FLAG(var, flag); } } while (0, 0)
#define WI_SET_FLAG_IF_FALSE(var, flag, condition)          do { if (!(wil::verify_bool(condition))) { WI_SET_FLAG(var, flag); } } while (0, 0)

#define WI_CLEAR_ALL_FLAGS(var, flags)                      ((var) &= ~(flags))
#define WI_CLEAR_FLAG(var, flag)                            WI_CLEAR_ALL_FLAGS(var, WI_STATIC_ASSERT_SINGLE_BIT_SET(flag))
#define WI_CLEAR_FLAG_IF(var, flag, condition)              do { if (wil::verify_bool(condition)) { WI_CLEAR_FLAG(var, flag); } } while (0, 0)
#define WI_CLEAR_FLAG_IF_FALSE(var, flag, condition)        do { if (!(wil::verify_bool(condition))) { WI_CLEAR_FLAG(var, flag); } } while (0, 0)

#define WI_UPDATE_FLAG(var, flag, isFlagSet)                (wil::verify_bool(isFlagSet) ? WI_SET_FLAG(var, flag) : WI_CLEAR_FLAG(var, flag))
#define WI_UPDATE_FLAGS_IN_MASK(var, flagsMask, newFlags)   wil::details::UpdateFlagsInMaskHelper(var, flagsMask, newFlags)

#define WI_TOGGLE_FLAG(var, flag)                           WI_TOGGLE_ALL_FLAGS(var, WI_STATIC_ASSERT_SINGLE_BIT_SET(flag))
#define WI_TOGGLE_ALL_FLAGS(var, flags)                     ((var) ^= (flags))

// Macros to inspect flag bitmasks

#define WI_ARE_ALL_FLAGS_SET(val, flags)                    wil::details::AreAllFlagsSetHelper(val, flags)
#define WI_IS_ANY_FLAG_SET(val, flags)                      (static_cast<decltype((val) & (flags))>(WI_ENUM_VALUE(val) & WI_ENUM_VALUE(flags)) != static_cast<decltype((val) & (flags))>(0))
#define WI_IS_FLAG_SET(val, flag)                           WI_IS_ANY_FLAG_SET(val, WI_STATIC_ASSERT_SINGLE_BIT_SET(flag))

#define WI_ARE_ALL_FLAGS_CLEAR(val, flags)                  (static_cast<decltype((val) & (flags))>(WI_ENUM_VALUE(val) & WI_ENUM_VALUE(flags)) == static_cast<decltype((val) & (flags))>(0))
#define WI_IS_ANY_FLAG_CLEAR(val, flags)                    (!wil::details::AreAllFlagsSetHelper(val, flags))
#define WI_IS_FLAG_CLEAR(val, flag)                         WI_ARE_ALL_FLAGS_CLEAR(val, WI_STATIC_ASSERT_SINGLE_BIT_SET(flag))

#define WI_IS_SINGLE_FLAG_SET(val)                          wil::details::IsSingleFlagSetHelper(val)
#define WI_IS_SINGLE_FLAG_SET_IN_MASK(val, mask)            wil::details::IsSingleFlagSetHelper((val) & (mask))
#define WI_IS_CLEAR_OR_SINGLE_FLAG_SET(val)                 wil::details::IsClearOrSingleFlagSetHelper(val)
#define WI_IS_CLEAR_OR_SINGLE_FLAG_SET_IN_MASK(val, mask)   wil::details::IsClearOrSingleFlagSetHelper((val) & (mask))

#if defined(WIL_SUPPORT_BITOPERATION_PASCAL_NAMES) || defined(SUPPORT_BITOPERATION_PASCAL_NAMES)

#define SetAllFlags(var, flags)                             WI_SET_ALL_FLAGS(var, flags)
#define SetFlag(var, flag)                                  WI_SET_FLAG(var, flag)
#define SetFlagIf(var, flag, condition)                     WI_SET_FLAG_IF(var, flag, condition)
#define SetFlagIfFalse(var, flag, condition)                WI_SET_FLAG_IF_FALSE(var, flag, condition)
#define ClearAllFlags(var, flags)                           WI_CLEAR_ALL_FLAGS(var, flags)
#define ClearFlag(var, flag)                                WI_CLEAR_FLAG(var, flag)
#define ClearFlagIf(var, flag, condition)                   WI_CLEAR_FLAG_IF(var, flag, condition)
#define ClearFlagIfFalse(var, flag, condition)              WI_CLEAR_FLAG_IF_FALSE(var, flag, condition)
#define UpdateFlag(var, flag, isFlagSet)                    WI_UPDATE_FLAG(var, flag, isFlagSet)
#define UpdateFlagsInMask(var, flagsMask, newFlags)         WI_UPDATE_FLAGS_IN_MASK(var, flagsMask, newFlags)
#define ToggleAllFlags(var, flags)                          WI_TOGGLE_ALL_FLAGS(var, flags)
#define ToggleFlag(var, flag)                               WI_TOGGLE_FLAG(var, flag)
#define AreAllFlagsSet(val, flags)                          WI_ARE_ALL_FLAGS_SET(val, flags)
#define IsAnyFlagSet(val, flags)                            WI_IS_ANY_FLAG_SET(val, flags)
#define IsFlagSet(val, flag)                                WI_IS_FLAG_SET(val, flag)
#define AreAllFlagsClear(val, flags)                        WI_ARE_ALL_FLAGS_CLEAR(val, flags)
#define IsAnyFlagClear(val, flags)                          WI_IS_ANY_FLAG_CLEAR(val, flags)
#define IsFlagClear(val, flag)                              WI_IS_FLAG_CLEAR(val, flag)
#define IsSingleFlagSet (val)                               WI_IS_SINGLE_FLAG_SET(val)
#define IsSingleFlagSetInMask(val, mask)                    WI_IS_SINGLE_FLAG_SET_IN_MASK(val, mask)
#define IsClearOrSingleFlagSet(val)                         WI_IS_CLEAR_OR_SINGLE_FLAG_SET(val)
#define IsClearOrSingleFlagSetInMask(val, mask)             WI_IS_CLEAR_OR_SINGLE_FLAG_SET_IN_MASK(val, mask)

#endif // WIL_SUPPORT_BITOPERATION_PASCAL_NAMES
#endif // !WIL_DOXYGEN

//! @deprecated Will be removed in the future.
#define WI_SET_FLAG_IF_TRUE                                 WI_SET_FLAG_IF

//! @}  // end bitfields group


#if defined(WIL_DOXYGEN)
//! This macro provides a C++ header with a guaranteed initialization function.
//! Normally, were a global object's constructor used for this purpose, the optimizer/linker might throw
//! the object away if it's unreferenced (which throws away the side-effects that the initialization function
//! was trying to achieve).  Using this macro forces linker inclusion of a variable that's initialized by the
//! provided function to elide that optimization.
//!
//! This functionality is primarily provided as a building block for header-based libraries (such as WIL)
//! to be able to layer additional functionality into other libraries by their mere inclusion.  Alternative models
//! of initialization should be used whenever they are available.
//! ~~~~
//! #if WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
//! WI_HEADER_INITITALIZATION_FUNCTION(InitializeDesktopFamilyApis, []
//! {
//!     g_pfnGetModuleName              = GetCurrentModuleName;
//!     g_pfnFailFastInLoaderCallout    = FailFastInLoaderCallout;
//!     g_pfnRtlNtStatusToDosErrorNoTeb = RtlNtStatusToDosErrorNoTeb;
//!     return 1;
//! });
//! #endif
//! ~~~~
//! The above example is used within WIL to decide whether or not the library containing WIL is allowed to use
//! desktop APIs.  Building this functionality as #IFDEFs within functions would create ODR violations, whereas
//! doing it with global function pointers and header initialization allows a runtime determination.
#define WI_HEADER_INITITALIZATION_FUNCTION(name, fn)
#elif defined(_M_IX86)
#define WI_HEADER_INITITALIZATION_FUNCTION(name, fn) \
    extern "C" { __declspec(selectany) unsigned char g_header_init_ ## name = static_cast<unsigned char>(fn()); } \
    __pragma(comment(linker, "/INCLUDE:_g_header_init_" #name))
#elif defined(_M_IA64) || defined(_M_AMD64) || defined(_M_ARM) || defined(_M_ARM64)
#define WI_HEADER_INITITALIZATION_FUNCTION(name, fn) \
    extern "C" { __declspec(selectany) unsigned char g_header_init_ ## name = static_cast<unsigned char>(fn()); } \
    __pragma(comment(linker, "/INCLUDE:g_header_init_" #name))
#else
    #error linker pragma must include g_header_init variation
#endif


//! All Windows Internal Library classes and functions are located within the "wil" namespace.
//! The 'wil' namespace is an intentionally short name as the intent is for code to be able to reference
//! the namespace directly (example: `wil::srwlock lock;`) without a using statement.  Resist adding a using
//! statement for wil to avoid introducing potential name collisions between wil and other namespaces.
namespace wil
{
    /// @cond
    namespace details
    {
        template <typename T>
        class pointer_range
        {
        public:
            pointer_range(T begin_, T end_) : m_begin(begin_), m_end(end_) {}
            T begin() const  { return m_begin; }
            T end() const    { return m_end; }
        private:
            T m_begin;
            T m_end;
        };
    }
    /// @endcond

    //! Enables using range-based for between a begin and end object pointer.
    //! ~~~~
    //! for (auto& obj : make_range(objPointerBegin, objPointerEnd))
    //! {
    //!     obj.Foo();
    //! }
    //! ~~~~
    template <typename T>
    details::pointer_range<T> make_range(T begin, T end)
    {
        return details::pointer_range<T>(begin, end);
    }

    //! Enables using range-based on a range when given the base pointer and the number of objects in the range.
    //! ~~~~
    //! for (auto& obj : make_range(objPointer, objCount))
    //! {
    //!     obj.Foo();
    //! }
    //! ~~~~
    template <typename T>
    details::pointer_range<T> make_range(T begin, size_t count)
    {
        return details::pointer_range<T>(begin, begin + count);
    }


    //! @defgroup outparam Output Parameters
    //! Improve the conciseness of assigning values to optional output parameters.
    //! @{

    //! Assign a value-type to an optional output parameter.
    //! Functions may have optional output parameters that want to receive a value calculated
    //! as part of the function.  Using this routine allows removal of many `if (param != nullptr)` blocks when
    //! commonly dealing with out parameters in routines.
    //! ~~~~
    //! void PerformTask(_Out_opt_ TaskResult* pResult = nullptr)
    //! {
    //!     TaskResult result = TaskResult::None;
    //!     // calculate result
    //!     wil::AssignToOptParam(pResult, result);
    //! }
    //! ~~~~
    //! @param outParam The optional out-pointer
    //! @param val The value to set in `outParam` if the out-pointer is non-null
    template <typename T>
    inline void AssignToOptParam(_Out_opt_ T *outParam, T val)
    {
        if (outParam != nullptr)
        {
            *outParam = val;
        }
    }

    //! Assign nullptr to an optional output pointer parameter.
    //! Functions may have optional output pointer parameters that want to receive an object from
    //! a function.  Using this routine provides trivial best practice initialization of an optional pointer-based
    //! output parameter on routine entry and will typically be combined with use of @ref AssignToOptParam or 
    //! @ref DetachToOptParam on routine exit.
    //! ~~~~
    //! bool HasString(_Out_opt_ PCWSTR* outString = nullptr)
    //! {
    //!     wil::AssignNullToOptParam(outString);
    //!     wil::unique_cotaskmem_string str = GetStringInternal();
    //!     bool hasString = (str != nullptr);
    //!     wil::DetachToOptParam(outString, str);
    //!     return hasString;
    //! }
    //! ~~~~
    //! @param outParam The optional out-pointer
    template <typename T>
    inline void AssignNullToOptParam(_Out_opt_ T *outParam)
    {
        if (outParam != nullptr)
        {
            *outParam = nullptr;
        }
    }

    //! @}      // end output parameter helpers


    //! Performs a logical or of the given variadic template parameters allowing indirect compile-time boolean evaluation.
    //! Example usage:
    //! ~~~~
    //! template <unsigned int... Rest>
    //! struct FeatureRequiredBy
    //! {
    //!     static const bool enabled = wil::variadic_logical_or<WilFeature<Rest>::enabled...>::value;
    //! };
    //! ~~~~
    template <bool...> struct variadic_logical_or;
    /// @cond
    template <> struct variadic_logical_or<> : wistd::false_type { };
    template <bool... Rest> struct variadic_logical_or<true, Rest...> : wistd::true_type { };
    template <bool... Rest> struct variadic_logical_or<false, Rest...> : variadic_logical_or<Rest...>::type { };
    /// @endcond

    /// @cond
    namespace details
    {
        template <unsigned long long flag>
        struct verify_single_flag_helper
        {
            static_assert((flag != 0) && ((flag & (flag - 1)) == 0), "Single flag expected, zero or multiple flags found");
            static const unsigned long long value = flag;
        };

        template <typename T>
        __forceinline bool verify_bool_helper(T const& t, wistd::true_type)
        {
            return static_cast<bool>(t);
        }

        template <typename T>
        __forceinline bool verify_bool_helper(T const& t, wistd::false_type)
        {
            static_assert((wistd::is_same<T, bool>::value), "Wrong Type: bool/BOOL/BOOLEAN/boolean expected");
        }

        template <>
        __forceinline bool verify_bool_helper<bool>(bool const& t, wistd::false_type)
        {
            return t;
        }

        template <>
        __forceinline bool verify_bool_helper<int>(int const& t, wistd::false_type)     // This supports BOOL
        {
            return (t != 0);
        }

        template <>
        __forceinline bool verify_bool_helper<unsigned char>(unsigned char const& t, wistd::false_type)     // This supports BOOLEAN and boolean
        {
            return !!t;
        }
    }
    /// @endcond


    //! @defgroup typesafety Type Validation
    //! Helpers to validate variable types to prevent accidental, but allowed type conversions.
    //! These helpers are most useful when building macros that accept a particular type.  Putting these functions around the types accepted
    //! prior to pushing that type through to a function (or using it within the macro) allows the macro to add an additional layer of type
    //! safety that would ordinarily be stripped away by C++ implicit conversions.  This system is extensively used in the error handling helper
    //! macros to validate the types given to various macro parameters.
    //! @{

    //! Verify that `val` can be evaluated as a logical bool.
    //! Other types will generate an intentional compilation error.  Allowed types for a logical bool are bool, BOOL,
    //! boolean, BOOLEAN, and classes with an explicit bool cast.
    //! @param val The logical bool expression
    //! @return A C++ bool representing the evaluation of `val`.
    template <typename T>
    __forceinline bool verify_bool(T const& val)
    {
        return details::verify_bool_helper(val, typename wistd::is_class<T>::type());
    }

    //! Verify that `val` is a Win32 BOOL value.
    //! Other types (including other logical bool expressions) will generate an intentional compilation error.  Note that this will
    //! accept any `int` value as long as that is the underlying typedef behind `BOOL`.
    //! @param val The Win32 BOOL returning expression
    //! @return A Win32 BOOL representing the evaluation of `val`.
    template <typename T>
    __forceinline int verify_BOOL(T const& val)
    {
        // Note: Written in terms of 'int' as BOOL is actually:  typedef int BOOL;
        static_assert((wistd::is_same<T, int>::value), "Wrong Type: BOOL expected");
        return val;
    }

    //! Verify that `hr` is an HRESULT value.
    //! Other types will generate an intentional compilation error.  Note that this will accept any `long` value as that is the
    //! underlying typedef behind HRESULT.
    //!
    //! Note that occasionally you might run into an HRESULT which is directly defined with a #define, such as:
    //! ~~~~
    //! #define UIA_E_NOTSUPPORTED   0x80040204  
    //! ~~~~
    //! Though this looks like an `HRESULT`, this is actually an `unsigned long` (the hex specification forces this).  When
    //! these are encountered and they are NOT in the public SDK (have not yet shipped to the public), then you should change
    //! their definition to match the manner in which `HRESULT` constants are defined in winerror.h:
    //! ~~~~
    //! #define E_NOTIMPL            _HRESULT_TYPEDEF_(0x80004001L)
    //! ~~~~
    //! When these are encountered in the public SDK, their type should not be changed and you should use a static_cast
    //! to use this value in a macro that utilizes `verify_hresult`, for example:
    //! ~~~~
    //! RETURN_HR_IF_FALSE(static_cast<HRESULT>(UIA_E_NOTSUPPORTED), (patternId == UIA_DragPatternId));
    //! ~~~~
    //! @param val The HRESULT returning expression
    //! @return An HRESULT representing the evaluation of `val`.
    template <typename T>
    _Post_satisfies_(return == hr)
    inline long verify_hresult(const T hr)
    {
        // Note: Written in terms of 'int' as HRESULT is actually:  typedef _Return_type_success_(return >= 0) long HRESULT
        static_assert(wistd::is_same<T, long>::value, "Wrong Type: HRESULT expected");
        return hr;
    }
    /// @}      // end type validation routines

    /// @cond
    // Implementation details for macros and helper functions... do not use directly.
    namespace details
    {
        // Use size-specific casts to avoid sign extending numbers -- avoid warning C4310: cast truncates constant value
        #define __WI_MAKE_UNSIGNED(val) \
            (__pragma(warning(push)) __pragma(warning(disable: 4310 4309)) (sizeof(val) == 1 ? static_cast<unsigned char>(val) : \
                                                                            sizeof(val) == 2 ? static_cast<unsigned short>(val) : \
                                                                            sizeof(val) == 4 ? static_cast<unsigned long>(val) :  \
                                                                            static_cast<unsigned long long>(val)) __pragma(warning(pop)))
        #define __WI_IS_UNSIGNED_SINGLE_FLAG_SET(val) ((val) && !((val) & ((val) - 1)))
        #define __WI_IS_SINGLE_FLAG_SET(val) __WI_IS_UNSIGNED_SINGLE_FLAG_SET(__WI_MAKE_UNSIGNED(val))

        template <typename TVal, typename TFlags>
        __forceinline bool AreAllFlagsSetHelper(TVal val, TFlags flags)
        {
            return ((val & flags) == static_cast<decltype(val & flags)>(flags));
        }

        template <typename TVal>
        __forceinline bool IsSingleFlagSetHelper(TVal val)
        {
            return __WI_IS_SINGLE_FLAG_SET(val);
        }

        template <typename TVal>
        __forceinline bool IsClearOrSingleFlagSetHelper(TVal val)
        {
            return ((val == static_cast<wistd::remove_reference<TVal>::type>(0)) || IsSingleFlagSetHelper(val));
        }

        template <typename TVal, typename TMask, typename TFlags>
        __forceinline void UpdateFlagsInMaskHelper(_Inout_ TVal& val, TMask mask, TFlags flags)
        {
            val = static_cast<wistd::remove_reference<TVal>::type>((val & ~mask) | (flags & mask));
        }

        template <long>
        struct variable_size;

        template <> 
        struct variable_size<1>
        {
            typedef unsigned char type;
        };

        template <> 
        struct variable_size<2>
        {
            typedef unsigned short type;
        };

        template <> 
        struct variable_size<4>
        {
            typedef unsigned long type;
        };

        template <> 
        struct variable_size<8>
        {
            typedef unsigned long long type;
        };

        template <typename T>
        struct variable_size_mapping
        {
            typedef typename variable_size<sizeof(T)>::type type;
        };
    } // details
    /// @endcond
    //! Defines the unsigned type of the same width (1, 2, 4, or 8 bytes) as the given type.
    //! This allows code to generically convert any enum class to it's corresponding underlying type.
    template <typename T>
    using integral_from_enum = typename details::variable_size_mapping<T>::type;

} // wil

#pragma warning(pop)
