// Windows Internal Libraries (wil)
// Note: do not include this file directly, include "wil\Functional.h"
//
// wil Usage Guidelines:
// https://microsoft.sharepoint.com/teams/osg_development/Shared%20Documents/Windows%20Internal%20Libraries%20for%20C++%20Usage%20Guide.docx?web=1
//
// wil Discussion Alias (wildisc):
// http://idwebelements/GroupManagement.aspx?Group=wildisc&Operation=join  (one-click join)

#pragma once

// STL common functionality
//
// Some aspects of STL are core language concepts that should be used from all C++ code, regardless
// of whether exceptions are enabled in the component.  Common library code that expects to be used
// from exception-free components want these concepts, but including <type_traits> directly introduces
// friction as it requires components not using STL to declare their STL version.  Doing so creates
// ambiguity around whether STL use is safe in a particular component and implicitly brings in
// a long list of headers (including <new>) which can create further ambiguity around throwing new
// support (some routines pulled in may expect it).  Secondarily, pulling in these headers also has
// the potential to create naming conflicts or other implied dependencies.
//
// To promote the use of these core language concepts outside of STL-based binaries, this file is
// selectively pulling those concepts *directly* from corresponding STL headers.  The corresponding
// "std::" namespace STL functions and types should be preferred over these in code that is bound to
// STL.  The implementation and naming of all functions are taken directly from STL, instead using 
// "wistd" (Windows Internal std) as the namespace.
//
// Routines in this namespace should always be considered a reflection of the *current* STL implementation
// of those routines.  Updates from STL should be taken, but no "bugs" should be fixed here.
//
// New, exception-based code should not use this namespace, but instead should prefer the std:: implementation.
// Only code that is not exception-based and libraries that expect to be utilized across both exception
// and non-exception based code should utilize this functionality.

#pragma warning(push)
#pragma warning(disable: 4100 4180 4244 4521 4522)

/// @cond
namespace wistd     // ("Windows Internal" std)
{
    // wistd::function
    //
    // All of the code below is in direct support of wistd::function.  This class is identical to std::function
    // with the following exceptions:
    //
    // 1) It never allocates and is safe to use from exception-free code (custom allocators are not supported)
    // 2) It's slightly bigger on the stack (60 bytes, rather than 24 for 32bit)
    // 3) There is an explicit static-assert if a lambda becomes too large to hold in the internal buffer (rather than an allocation)

    /// @cond
    namespace details
    {
        // minimally necessary featured no-throw allocator for wistd::function
        template <class _Ty>
        class function_allocator
        {
        public:
        	typedef _Ty *pointer;
        	typedef size_t size_type;

            template<class _Other>
            struct rebind
            {
                typedef function_allocator<_Other> other;
            };

            function_allocator() = default;

	        template<class _Other>
		    function_allocator(const function_allocator<_Other>&) WI_NOEXCEPT
		    {
		    }

            void deallocate(pointer, size_type)
            {
                __fastfail(FAST_FAIL_FATAL_APP_EXIT);   // see allocate() below
            }

            pointer allocate(size_type)
            {
                // wistd::function<> is built not to allocate, but instead use a small in-class buffer of 10 pointers to contain the allocated
                // function pointer and capture clause.  If this __fastfail fires, it means that that size constraint has somehow escaped the
                // static_assert<> below.  (Indicating a bug in wistd::function<>.)
                __fastfail(FAST_FAIL_FATAL_APP_EXIT);
                return nullptr;
            }

            template<class _Uty>
            void destroy(_Uty *_Ptr)
            {
                _Ptr->~_Uty();
            }
        };
    }
    /// @endcond

    // Implementation detail:
    // Brief overview of the process followed for bringing <function> into wistd:
    // Pull in appropriate code:
    //      Part of <functional> (leave behind std::bind)
    //      Most of <xrefwrap>
    //      Part of <xstddef> (the various type and variadic macros)
    // Renames:
    //      _STD        -> wistd::
    //      _NOEXCEPT   -> WI_NOEXCEPT
    //      ALL MACROS  -> Prefix with __WI
    // Removal:
    //      type_info, target(), target_type
    //      all methods supporting a custom allocator
    // Changes:
    //      _Xbad_function_call() -> make fail fast, rather than throw
    //      make bool cast explicit
    //      change default allocator to function_allocator<>
    //      increase _Space buffer to 12 pointers and static_assert that sizeof callable wrapper is within that (remove exceptions)

    // __WI_VARIADIC_EXPAND* MACROS  
  
 #if !defined(__WI_VARIADIC_MAX)  
  #define __WI_VARIADIC_MAX	5  
  
 #elif __WI_VARIADIC_MAX < 5 || 10 < __WI_VARIADIC_MAX  
  #error __WI_VARIADIC_MAX must be between 5 and 10, inclusive  
 #endif /* !defined(__WI_VARIADIC_MAX) */  
  
 #if __WI_VARIADIC_MAX == 5  
#define __WI_VARIADIC_EXPAND_2X	__WI_VARIADIC_EXPAND_25  
#define __WI_VARIADIC_EXPAND_ALT_0X	__WI_VARIADIC_EXPAND_ALT_05  
#define __WI_VARIADIC_EXPAND_P1_2X	__WI_VARIADIC_EXPAND_P1_25  
#define __WI_VARIADIC_EXPAND_0_1X(FUNC) \
    __WI_VARIADIC_EXPAND_XX_15(FUNC, __WI_VARIADIC_EXPAND_0)
#define __WI_VARIADIC_EXPAND_XX_1X(FUNC, EXPAND) \
    __WI_VARIADIC_EXPAND_XX_15(FUNC, EXPAND)  
#define __WI_VARIADIC_EXPAND_1X_1D(FUNC) \
    __WI_VARIADIC_EXPAND_15_1D(FUNC)  
  
#define __WI_PAD_LIST0	__WI_PAD_LIST0_5  
#define __WI_PAD_LIST1	__WI_PAD_LIST0_4  
#define __WI_PAD_LIST2	__WI_PAD_LIST0_3  
#define __WI_PAD_LIST3	__WI_PAD_LIST0_2  
#define __WI_PAD_LIST4	__WI_PAD_LIST0_1  
#define __WI_PAD_LIST5	__WI_PAD_LIST0_0  
  
 #elif __WI_VARIADIC_MAX == 6  
#define __WI_VARIADIC_EXPAND_2X	__WI_VARIADIC_EXPAND_26  
#define __WI_VARIADIC_EXPAND_ALT_0X	__WI_VARIADIC_EXPAND_ALT_06  
#define __WI_VARIADIC_EXPAND_P1_2X	__WI_VARIADIC_EXPAND_P1_26  
#define __WI_VARIADIC_EXPAND_0_1X(FUNC) \  
    __WI_VARIADIC_EXPAND_XX_16(FUNC, __WI_VARIADIC_EXPAND_0)  
#define __WI_VARIADIC_EXPAND_XX_1X(FUNC, EXPAND) \  
    __WI_VARIADIC_EXPAND_XX_16(FUNC, EXPAND)  
#define __WI_VARIADIC_EXPAND_1X_1D(FUNC) \  
    __WI_VARIADIC_EXPAND_16_1D(FUNC)  
  
#define __WI_PAD_LIST0 __WI_PAD_LIST0_6  
#define __WI_PAD_LIST1 __WI_PAD_LIST0_5  
#define __WI_PAD_LIST2 __WI_PAD_LIST0_4  
#define __WI_PAD_LIST3 __WI_PAD_LIST0_3  
#define __WI_PAD_LIST4 __WI_PAD_LIST0_2  
#define __WI_PAD_LIST5 __WI_PAD_LIST0_1  
#define __WI_PAD_LIST6 __WI_PAD_LIST0_0  
  
 #elif __WI_VARIADIC_MAX == 7  
#define __WI_VARIADIC_EXPAND_2X	__WI_VARIADIC_EXPAND_27  
#define __WI_VARIADIC_EXPAND_ALT_0X	__WI_VARIADIC_EXPAND_ALT_07  
#define __WI_VARIADIC_EXPAND_P1_2X	__WI_VARIADIC_EXPAND_P1_27  
#define __WI_VARIADIC_EXPAND_0_1X(FUNC) \  
    __WI_VARIADIC_EXPAND_XX_17(FUNC, __WI_VARIADIC_EXPAND_0)  
#define __WI_VARIADIC_EXPAND_XX_1X(FUNC, EXPAND) \  
    __WI_VARIADIC_EXPAND_XX_17(FUNC, EXPAND)  
#define __WI_VARIADIC_EXPAND_1X_1D(FUNC) \  
    __WI_VARIADIC_EXPAND_17_1D(FUNC)  
  
#define __WI_PAD_LIST0 __WI_PAD_LIST0_7  
#define __WI_PAD_LIST1 __WI_PAD_LIST0_6  
#define __WI_PAD_LIST2 __WI_PAD_LIST0_5  
#define __WI_PAD_LIST3 __WI_PAD_LIST0_4  
#define __WI_PAD_LIST4 __WI_PAD_LIST0_3  
#define __WI_PAD_LIST5 __WI_PAD_LIST0_2  
#define __WI_PAD_LIST6 __WI_PAD_LIST0_1  
#define __WI_PAD_LIST7 __WI_PAD_LIST0_0  
  
 #elif __WI_VARIADIC_MAX == 8  
#define __WI_VARIADIC_EXPAND_2X	__WI_VARIADIC_EXPAND_28  
#define __WI_VARIADIC_EXPAND_ALT_0X	__WI_VARIADIC_EXPAND_ALT_08  
#define __WI_VARIADIC_EXPAND_P1_2X	__WI_VARIADIC_EXPAND_P1_28  
#define __WI_VARIADIC_EXPAND_0_1X(FUNC) \  
    __WI_VARIADIC_EXPAND_XX_18(FUNC, __WI_VARIADIC_EXPAND_0)  
#define __WI_VARIADIC_EXPAND_XX_1X(FUNC, EXPAND) \  
    __WI_VARIADIC_EXPAND_XX_18(FUNC, EXPAND)  
#define __WI_VARIADIC_EXPAND_1X_1D(FUNC) \  
    __WI_VARIADIC_EXPAND_18_1D(FUNC)  
  
#define __WI_PAD_LIST0 __WI_PAD_LIST0_8  
#define __WI_PAD_LIST1 __WI_PAD_LIST0_7  
#define __WI_PAD_LIST2 __WI_PAD_LIST0_6  
#define __WI_PAD_LIST3 __WI_PAD_LIST0_5  
#define __WI_PAD_LIST4 __WI_PAD_LIST0_4  
#define __WI_PAD_LIST5 __WI_PAD_LIST0_3  
#define __WI_PAD_LIST6 __WI_PAD_LIST0_2  
#define __WI_PAD_LIST7 __WI_PAD_LIST0_1  
#define __WI_PAD_LIST8 __WI_PAD_LIST0_0  
  
 #elif __WI_VARIADIC_MAX == 9  
#define __WI_VARIADIC_EXPAND_2X	__WI_VARIADIC_EXPAND_29  
#define __WI_VARIADIC_EXPAND_ALT_0X	__WI_VARIADIC_EXPAND_ALT_09  
#define __WI_VARIADIC_EXPAND_P1_2X	__WI_VARIADIC_EXPAND_P1_29  
#define __WI_VARIADIC_EXPAND_0_1X(FUNC) \  
    __WI_VARIADIC_EXPAND_XX_19(FUNC, __WI_VARIADIC_EXPAND_0)  
#define __WI_VARIADIC_EXPAND_XX_1X(FUNC, EXPAND) \  
    __WI_VARIADIC_EXPAND_XX_19(FUNC, EXPAND)  
#define __WI_VARIADIC_EXPAND_1X_1D(FUNC) \  
    __WI_VARIADIC_EXPAND_19_1D(FUNC)  
  
#define __WI_PAD_LIST0 __WI_PAD_LIST0_9  
#define __WI_PAD_LIST1 __WI_PAD_LIST0_8  
#define __WI_PAD_LIST2 __WI_PAD_LIST0_7  
#define __WI_PAD_LIST3 __WI_PAD_LIST0_6  
#define __WI_PAD_LIST4 __WI_PAD_LIST0_5  
#define __WI_PAD_LIST5 __WI_PAD_LIST0_4  
#define __WI_PAD_LIST6 __WI_PAD_LIST0_3  
#define __WI_PAD_LIST7 __WI_PAD_LIST0_2  
#define __WI_PAD_LIST8 __WI_PAD_LIST0_1  
#define __WI_PAD_LIST9 __WI_PAD_LIST0_0  
  
 #elif __WI_VARIADIC_MAX == 10  
#define __WI_VARIADIC_EXPAND_2X	__WI_VARIADIC_EXPAND_2A  
#define __WI_VARIADIC_EXPAND_ALT_0X	__WI_VARIADIC_EXPAND_ALT_0A  
#define __WI_VARIADIC_EXPAND_P1_2X	__WI_VARIADIC_EXPAND_P1_2A  
#define __WI_VARIADIC_EXPAND_0_1X(FUNC) \  
    __WI_VARIADIC_EXPAND_XX_1A(FUNC, __WI_VARIADIC_EXPAND_0)  
#define __WI_VARIADIC_EXPAND_XX_1X(FUNC, EXPAND) \  
    __WI_VARIADIC_EXPAND_XX_1A(FUNC, EXPAND)  
#define __WI_VARIADIC_EXPAND_1X_1D(FUNC) \  
    __WI_VARIADIC_EXPAND_1A_1D(FUNC)  
  
#define __WI_PAD_LIST0 __WI_PAD_LIST0_10  
#define __WI_PAD_LIST1 __WI_PAD_LIST0_9  
#define __WI_PAD_LIST2 __WI_PAD_LIST0_8  
#define __WI_PAD_LIST3 __WI_PAD_LIST0_7  
#define __WI_PAD_LIST4 __WI_PAD_LIST0_6  
#define __WI_PAD_LIST5 __WI_PAD_LIST0_5  
#define __WI_PAD_LIST6 __WI_PAD_LIST0_4  
#define __WI_PAD_LIST7 __WI_PAD_LIST0_3  
#define __WI_PAD_LIST8 __WI_PAD_LIST0_2  
#define __WI_PAD_LIST9 __WI_PAD_LIST0_1  
#define __WI_PAD_LIST10 __WI_PAD_LIST0_0
 #endif /* __WI_VARIADIC_MAX */  
  
// call FUNC(TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4)  
#define __WI_VARIADIC_EXPAND_0(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST0, __WI_PAD_LIST0, __WI_RAW_LIST0, , X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_1(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_2(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_3(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST3, __WI_PAD_LIST3, __WI_RAW_LIST3, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_4(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST4, __WI_PAD_LIST4, __WI_RAW_LIST4, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_5(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST5, __WI_PAD_LIST5, __WI_RAW_LIST5, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_6(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST6, __WI_PAD_LIST6, __WI_RAW_LIST6, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_7(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST7, __WI_PAD_LIST7, __WI_RAW_LIST7, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_8(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST8, __WI_PAD_LIST8, __WI_RAW_LIST8, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_9(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST9, __WI_PAD_LIST9, __WI_RAW_LIST9, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_10(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST10, __WI_PAD_LIST10, __WI_RAW_LIST10, __WI_COMMA, X1, X2, X3, X4)  
  
// alternate form for NxN expansion  
#define __WI_VARIADIC_EXPAND_ALT_0(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST0, __WI_PAD_LIST0, __WI_RAW_LIST0, , X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_1(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_2(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_3(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST3, __WI_PAD_LIST3, __WI_RAW_LIST3, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_4(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST4, __WI_PAD_LIST4, __WI_RAW_LIST4, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_5(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST5, __WI_PAD_LIST5, __WI_RAW_LIST5, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_6(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST6, __WI_PAD_LIST6, __WI_RAW_LIST6, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_7(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST7, __WI_PAD_LIST7, __WI_RAW_LIST7, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_8(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST8, __WI_PAD_LIST8, __WI_RAW_LIST8, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_9(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST9, __WI_PAD_LIST9, __WI_RAW_LIST9, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_A(FUNC, X1, X2, X3, X4) \
FUNC(__WI_TEM_LIST10, __WI_PAD_LIST10, __WI_RAW_LIST10, __WI_COMMA, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_25(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_2(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_3(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_4(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_5(FUNC, X1, X2, X3, X4)
  
#define __WI_VARIADIC_EXPAND_26(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_25(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_6(FUNC, X1, X2, X3, X4)
  
#define __WI_VARIADIC_EXPAND_27(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_26(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_7(FUNC, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_28(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_27(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_8(FUNC, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_29(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_28(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_9(FUNC, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_2A(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_29(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_10(FUNC, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_05(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_0(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_1(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_2(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_3(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_4(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_5(FUNC, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_06(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_05(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_6(FUNC, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_07(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_06(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_7(FUNC, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_08(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_07(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_8(FUNC, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_09(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_08(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_9(FUNC, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_ALT_0A(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_09(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_ALT_A(FUNC, X1, X2, X3, X4)  
  
    // for 0-X args  
#define __WI_VARIADIC_EXPAND_0X(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_0(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_1X(FUNC, X1, X2, X3, X4)  
  
#define __WI_VARIADIC_EXPAND_1X(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_1(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_2X(FUNC, X1, X2, X3, X4)  
  
    // for extra list, one element shorter  
#define __WI_VARIADIC_EXPAND_P1_0(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_0(FUNC, X1, X2, __WI_RAW_LIST0, ) \
  
#define __WI_VARIADIC_EXPAND_P1_1(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_1(FUNC, X1, X2, __WI_RAW_LIST0, ) \
  
#define __WI_VARIADIC_EXPAND_P1_25(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_2(FUNC, X1, X2, __WI_RAW_LIST1, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_3(FUNC, X1, X2, __WI_RAW_LIST2, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_4(FUNC, X1, X2, __WI_RAW_LIST3, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_5(FUNC, X1, X2, __WI_RAW_LIST4, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_P1_26(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_P1_25(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_6(FUNC, X1, X2, __WI_RAW_LIST5, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_P1_27(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_P1_26(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_7(FUNC, X1, X2, __WI_RAW_LIST6, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_P1_28(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_P1_27(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_8(FUNC, X1, X2, __WI_RAW_LIST7, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_P1_29(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_P1_28(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_9(FUNC, X1, X2, __WI_RAW_LIST8, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_P1_2A(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_P1_29(FUNC, X1, X2, X3, X4) \
    __WI_VARIADIC_EXPAND_10(FUNC, X1, X2, __WI_RAW_LIST9, __WI_COMMA)  
  
    // VARIADIC DOUBLE LOOPS  
// call FUNC(TEMPLATE_LIST1, PADDING_LIST1, LIST1, COMMA1,  
// TEMPLATE_LIST2, PADDING_LIST2, LIST2, COMMA2)  
  
    // for (0-X, 0) args  
#define __WI_VARIADIC_EXPAND_0X_0(FUNC) \
    __WI_VARIADIC_EXPAND_0X(FUNC, __WI_TEM_LIST0, __WI_PAD_LIST0, __WI_RAW_LIST0, ) \
  
    // for (??, 1-X) args  
#define __WI_VARIADIC_EXPAND_XX_11(FUNC, EXPAND) \
    EXPAND(FUNC, __WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_XX_12(FUNC, EXPAND) \
    __WI_VARIADIC_EXPAND_XX_11(FUNC, EXPAND) \
    EXPAND(FUNC, __WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_XX_13(FUNC, EXPAND) \
    __WI_VARIADIC_EXPAND_XX_12(FUNC, EXPAND) \
    EXPAND(FUNC, __WI_TEM_LIST3, __WI_PAD_LIST3, __WI_RAW_LIST3, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_XX_14(FUNC, EXPAND) \
    __WI_VARIADIC_EXPAND_XX_13(FUNC, EXPAND) \
    EXPAND(FUNC, __WI_TEM_LIST4, __WI_PAD_LIST4, __WI_RAW_LIST4, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_XX_15(FUNC, EXPAND) \
    __WI_VARIADIC_EXPAND_XX_14(FUNC, EXPAND) \
    EXPAND(FUNC, __WI_TEM_LIST5, __WI_PAD_LIST5, __WI_RAW_LIST5, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_XX_16(FUNC, EXPAND) \
    __WI_VARIADIC_EXPAND_XX_15(FUNC, EXPAND) \
    EXPAND(FUNC, __WI_TEM_LIST6, __WI_PAD_LIST6, __WI_RAW_LIST6, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_XX_17(FUNC, EXPAND) \
    __WI_VARIADIC_EXPAND_XX_16(FUNC, EXPAND) \
    EXPAND(FUNC, __WI_TEM_LIST7, __WI_PAD_LIST7, __WI_RAW_LIST7, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_XX_18(FUNC, EXPAND) \
    __WI_VARIADIC_EXPAND_XX_17(FUNC, EXPAND) \
    EXPAND(FUNC, __WI_TEM_LIST8, __WI_PAD_LIST8, __WI_RAW_LIST8, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_XX_19(FUNC, EXPAND) \
    __WI_VARIADIC_EXPAND_XX_18(FUNC, EXPAND) \
    EXPAND(FUNC, __WI_TEM_LIST9, __WI_PAD_LIST9, __WI_RAW_LIST9, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_XX_1A(FUNC, EXPAND) \
    __WI_VARIADIC_EXPAND_XX_19(FUNC, EXPAND) \
    EXPAND(FUNC, __WI_TEM_LIST10, __WI_PAD_LIST10, __WI_RAW_LIST10, __WI_COMMA)  
  
    // for (n-X, n-X) args  
#define __WI_VARIADIC_EXPAND_0X_0X(FUNC) \
    __WI_VARIADIC_EXPAND_0X_0(FUNC) \
    __WI_VARIADIC_EXPAND_0_1X(FUNC) \
    __WI_VARIADIC_EXPAND_1X_1X(FUNC)  
  
#define __WI_VARIADIC_EXPAND_1X_1X(FUNC) \
    __WI_VARIADIC_EXPAND_XX_1X(FUNC, __WI_VARIADIC_EXPAND_1X)  
  
    // template lists for functions with no leading parameter  
#define __WI_TEM_LIST0(MAP)  
  
#define __WI_TEM_LIST1(MAP) \
    template<MAP(0)>  
  
#define __WI_TEM_LIST2(MAP) \
    template<MAP(0) __WI_COMMA MAP(1)>  
  
#define __WI_TEM_LIST3(MAP) \
    template<MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2)>  
  
#define __WI_TEM_LIST4(MAP) \
    template<MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3)>  
  
#define __WI_TEM_LIST5(MAP) \
    template<MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4)>  
  
#define __WI_TEM_LIST6(MAP) \
    template<MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4) __WI_COMMA MAP(5)>  
  
#define __WI_TEM_LIST7(MAP) \
    template<MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6)>  
  
#define __WI_TEM_LIST8(MAP) \
    template<MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6) __WI_COMMA MAP(7)>  
  
#define __WI_TEM_LIST9(MAP) \
    template<MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6) __WI_COMMA MAP(7) \
    __WI_COMMA MAP(8)>  
  
#define __WI_TEM_LIST10(MAP) \
    template<MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6) __WI_COMMA MAP(7) \
    __WI_COMMA MAP(8) __WI_COMMA MAP(9)>  
  
    // diagonal, for (1-Y, 1-Z) args, Y+Z <= X  
#define __WI_VARIADIC_EXPAND_15_1D(FUNC) \
    __WI_VARIADIC_EXPAND_1(FUNC, __WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_1(FUNC, __WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_1(FUNC, __WI_TEM_LIST3, __WI_PAD_LIST3, __WI_RAW_LIST3, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_1(FUNC, __WI_TEM_LIST4, __WI_PAD_LIST4, __WI_RAW_LIST4, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_2(FUNC, __WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_2(FUNC, __WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_2(FUNC, __WI_TEM_LIST3, __WI_PAD_LIST3, __WI_RAW_LIST3, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_3(FUNC, __WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_3(FUNC, __WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_4(FUNC, __WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_16_1D(FUNC) \
    __WI_VARIADIC_EXPAND_15_1D(FUNC) \
    __WI_VARIADIC_EXPAND_1(FUNC, __WI_TEM_LIST5, __WI_PAD_LIST5, __WI_RAW_LIST5, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_2(FUNC, __WI_TEM_LIST4, __WI_PAD_LIST4, __WI_RAW_LIST4, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_3(FUNC, __WI_TEM_LIST3, __WI_PAD_LIST3, __WI_RAW_LIST3, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_4(FUNC, __WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_5(FUNC, __WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_17_1D(FUNC) \
    __WI_VARIADIC_EXPAND_16_1D(FUNC) \
    __WI_VARIADIC_EXPAND_1(FUNC, __WI_TEM_LIST6, __WI_PAD_LIST6, __WI_RAW_LIST6, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_2(FUNC, __WI_TEM_LIST5, __WI_PAD_LIST5, __WI_RAW_LIST5, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_3(FUNC, __WI_TEM_LIST4, __WI_PAD_LIST4, __WI_RAW_LIST4, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_4(FUNC, __WI_TEM_LIST3, __WI_PAD_LIST3, __WI_RAW_LIST3, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_5(FUNC, __WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_6(FUNC, __WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_18_1D(FUNC) \
    __WI_VARIADIC_EXPAND_17_1D(FUNC) \
    __WI_VARIADIC_EXPAND_1(FUNC, __WI_TEM_LIST7, __WI_PAD_LIST7, __WI_RAW_LIST7, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_2(FUNC, __WI_TEM_LIST6, __WI_PAD_LIST6, __WI_RAW_LIST6, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_3(FUNC, __WI_TEM_LIST5, __WI_PAD_LIST5, __WI_RAW_LIST5, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_4(FUNC, __WI_TEM_LIST4, __WI_PAD_LIST4, __WI_RAW_LIST4, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_5(FUNC, __WI_TEM_LIST3, __WI_PAD_LIST3, __WI_RAW_LIST3, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_6(FUNC, __WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_7(FUNC, __WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_19_1D(FUNC) \
    __WI_VARIADIC_EXPAND_18_1D(FUNC) \
    __WI_VARIADIC_EXPAND_1(FUNC, __WI_TEM_LIST8, __WI_PAD_LIST8, __WI_RAW_LIST8, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_2(FUNC, __WI_TEM_LIST7, __WI_PAD_LIST7, __WI_RAW_LIST7, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_3(FUNC, __WI_TEM_LIST6, __WI_PAD_LIST6, __WI_RAW_LIST6, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_4(FUNC, __WI_TEM_LIST5, __WI_PAD_LIST5, __WI_RAW_LIST5, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_5(FUNC, __WI_TEM_LIST4, __WI_PAD_LIST4, __WI_RAW_LIST4, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_6(FUNC, __WI_TEM_LIST3, __WI_PAD_LIST3, __WI_RAW_LIST3, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_7(FUNC, __WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_8(FUNC, __WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA)  
  
#define __WI_VARIADIC_EXPAND_1A_1D(FUNC) \
    __WI_VARIADIC_EXPAND_19_1D(FUNC) \
    __WI_VARIADIC_EXPAND_1(FUNC, __WI_TEM_LIST9, __WI_PAD_LIST9, __WI_RAW_LIST9, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_2(FUNC, __WI_TEM_LIST8, __WI_PAD_LIST8, __WI_RAW_LIST8, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_3(FUNC, __WI_TEM_LIST7, __WI_PAD_LIST7, __WI_RAW_LIST7, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_4(FUNC, __WI_TEM_LIST6, __WI_PAD_LIST6, __WI_RAW_LIST6, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_5(FUNC, __WI_TEM_LIST5, __WI_PAD_LIST5, __WI_RAW_LIST5, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_6(FUNC, __WI_TEM_LIST4, __WI_PAD_LIST4, __WI_RAW_LIST4, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_7(FUNC, __WI_TEM_LIST3, __WI_PAD_LIST3, __WI_RAW_LIST3, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_8(FUNC, __WI_TEM_LIST2, __WI_PAD_LIST2, __WI_RAW_LIST2, __WI_COMMA) \
    __WI_VARIADIC_EXPAND_9(FUNC, __WI_TEM_LIST1, __WI_PAD_LIST1, __WI_RAW_LIST1, __WI_COMMA)  
  
    // padding lists  
#define __WI_PAD_LIST0_0(MAP) \
    MAP(0)  
  
#define __WI_PAD_LIST0_1(MAP) \
    MAP(0) __WI_COMMA MAP(1)  
  
#define __WI_PAD_LIST0_2(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2)  
  
#define __WI_PAD_LIST0_3(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3)  
  
#define __WI_PAD_LIST0_4(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) __WI_COMMA \
    MAP(4)  
  
#define __WI_PAD_LIST0_5(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) __WI_COMMA \
    MAP(4) __WI_COMMA MAP(5)  
  
#define __WI_PAD_LIST0_6(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) __WI_COMMA \
    MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6)  
  
#define __WI_PAD_LIST0_7(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) __WI_COMMA \
    MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6) __WI_COMMA MAP(7)  
  
#define __WI_PAD_LIST0_8(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) __WI_COMMA \
    MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6) __WI_COMMA MAP(7) __WI_COMMA \
    MAP(8)  
  
#define __WI_PAD_LIST0_9(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) __WI_COMMA \
    MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6) __WI_COMMA MAP(7) __WI_COMMA \
    MAP(8) __WI_COMMA MAP(9)  
  
#define __WI_PAD_LIST0_10(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) __WI_COMMA \
    MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6) __WI_COMMA MAP(7) __WI_COMMA \
    MAP(8) __WI_COMMA MAP(9) __WI_COMMA MAP(10)  
  
    // plain lists  
#define __WI_RAW_LIST0(MAP)  
  
#define __WI_RAW_LIST1(MAP) \
    MAP(0)  
  
#define __WI_RAW_LIST2(MAP) \
    MAP(0) __WI_COMMA MAP(1)  
  
#define __WI_RAW_LIST3(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2)  
  
#define __WI_RAW_LIST4(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3)  
  
#define __WI_RAW_LIST5(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4)  
  
#define __WI_RAW_LIST6(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4) __WI_COMMA MAP(5)  
  
#define __WI_RAW_LIST7(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6)  
  
#define __WI_RAW_LIST8(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6) __WI_COMMA MAP(7)  
  
#define __WI_RAW_LIST9(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6) __WI_COMMA MAP(7) \
    __WI_COMMA MAP(8)  
  
#define __WI_RAW_LIST10(MAP) \
    MAP(0) __WI_COMMA MAP(1) __WI_COMMA MAP(2) __WI_COMMA MAP(3) \
    __WI_COMMA MAP(4) __WI_COMMA MAP(5) __WI_COMMA MAP(6) __WI_COMMA MAP(7) \
    __WI_COMMA MAP(8) __WI_COMMA MAP(9)  
  
    // variant calling sequences  
  
 #if defined(_M_IX86)  
  
 #if defined(_M_CEE)  
#define __WI_VARIADIC_CALL_OPT_X1(FUNC, X1, X2, X3, X4, \
    CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __stdcall, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __clrcall, X6, X7, X8)  
  
 #else /* defined(_M_CEE) */  
#define __WI_VARIADIC_CALL_OPT_X1(FUNC, X1, X2, X3, X4, \
    CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __stdcall, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __fastcall, X6, X7, X8)  
 #endif /* defined(_M_CEE) */  
  
#else /* defined (_M_IX86) */  
  
 #if defined(_M_CEE)  
#define __WI_VARIADIC_CALL_OPT_X1(FUNC, X1, X2, X3, X4, \
    CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __clrcall, X6, X7, X8)  
  
 #else /* defined(_M_CEE) */  
#define __WI_VARIADIC_CALL_OPT_X1(FUNC, X1, X2, X3, X4, \
    CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, CALL_OPT, X6, X7, X8)  
 #endif /* defined(_M_CEE) */  
 #endif /* defined (_M_IX86) */  
  
 #if defined(_M_IX86)  
  
 #if defined(_M_CEE)  
#define __WI_VARIADIC_CALL_OPT_X2(FUNC, X1, X2, X3, X4, \
    CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __cdecl, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __stdcall, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __clrcall, X6, X7, X8)  
  
 #else /* defined(_M_CEE) */  
#define __WI_VARIADIC_CALL_OPT_X2(FUNC, X1, X2, X3, X4, \
    CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __cdecl, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __stdcall, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __fastcall, X6, X7, X8)  
 #endif /* defined(_M_CEE) */  
  
#else /* defined (_M_IX86) */  
  
 #if defined(_M_CEE)  
#define __WI_VARIADIC_CALL_OPT_X2(FUNC, X1, X2, X3, X4, \
    CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, __clrcall, X6, X7, X8)  
  
 #else /* defined(_M_CEE) */  
#define __WI_VARIADIC_CALL_OPT_X2(FUNC, X1, X2, X3, X4, \
    CALL_OPT, X6, X7, X8) \
        FUNC(X1, X2, X3, X4, CALL_OPT, X6, X7, X8)  
 #endif /* defined(_M_CEE) */  
 #endif /* defined (_M_IX86) */  
  
    // MAP functions  
#define __WI_VAR_VAL(NUM)	\
    _V ## NUM  
#define __WI_VAR_TYPE(NUM)	\
    _V ## NUM ## _t  
#define __WI_CLASS_TYPE(NUM)	\
    class __WI_VAR_TYPE(NUM)  
  
#define __WI_VAR_VALX(NUM)	\
    _Vx ## NUM  
#define __WI_VAR_TYPEX(NUM)	\
    _Vx ## NUM ## _t  
#define __WI_CLASS_TYPEX(NUM)	\
    class __WI_VAR_TYPEX(NUM)  
  
#define __WI_CLASS_TYPE_TYPEX(NUM)	\
    class __WI_VAR_TYPE(NUM) __WI_COMMA class __WI_VAR_TYPEX(NUM)  
  
#define __WI_CLASS_NIL(NUM)	\
    class = _Nil  
#define __WI_CLASS_TYPE_NIL(NUM)	\
    class __WI_VAR_TYPE(NUM) = _Nil  
#define __WI_NIL_PAD(NUM)	\
    _Nil  
  
#define __WI_ARG(NUM) \
    __WI_VAR_VAL(NUM)  
#define __WI_ARGX(NUM) \
    __WI_VAR_VALX(NUM)  
#define __WI_DECLVAL(NUM)	\
    wistd::declval<__WI_VAR_TYPE(NUM)>()  
#define __WI_DECAY_COPY_FORWARD_ARG(NUM)	\
    _Decay_copy(wistd::forward<__WI_TYPE(NUM)>(__WI_VAR_VAL(NUM)))  
#define __WI_REMOVE_REF(NUM)	\
    typename remove_reference<__WI_TYPE(NUM)>::type&  
#define __WI_UNREFWRAP_TYPE(NUM)	\
    typename _Unrefwrap<__WI_TYPE(NUM)>::type  
  
#define __WI_ELEMENT_ARG(NUM)	\
    wistd::get<NUM>(_Tpl1)  
#define __WI_FORWARD_ARG(NUM)	\
    wistd::forward<__WI_VAR_TYPE(NUM)>(__WI_VAR_VAL(NUM))  
#define __WI_FORWARD_ELEMENT_ARG(NUM)	\
    wistd::get<NUM>(wistd::forward<decltype(_Tpl1)>(_Tpl1))  
  
#define __WI_ELEMENT_ARGX(NUM)	\
    wistd::get<NUM>(_Tpl2)  
#define __WI_FORWARD_ARGX(NUM)	\
    wistd::forward<__WI_VAR_TYPEX(NUM)>(__WI_VAR_VALX(NUM))  
#define __WI_FORWARD_ELEMENT_ARGX(NUM)	\
    wistd::get<NUM>(wistd::forward<decltype(_Tpl2)>(_Tpl2))  
  
#define __WI_TYPE(NUM) \
    __WI_VAR_TYPE(NUM)  
#define __WI_TYPE_REF(NUM)	\
    __WI_VAR_TYPE(NUM)&  
#define __WI_TYPE_REFREF(NUM)	\
    __WI_VAR_TYPE(NUM)&&  
#define __WI_CONST_TYPE_REF(NUM)	\
    const __WI_TYPE(NUM)&  
#define __WI_TYPE_ARG(NUM) \
    __WI_VAR_TYPE(NUM) __WI_VAR_VAL(NUM)  
#define __WI_TYPE_REF_ARG(NUM)	\
    __WI_TYPE_REF(NUM) __WI_VAR_VAL(NUM)  
#define __WI_TYPE_REFREF_ARG(NUM)	\
    __WI_TYPE_REFREF(NUM) __WI_VAR_VAL(NUM)  
#define __WI_CONST_TYPE_REF_ARG(NUM)	\
    const __WI_TYPE(NUM)& __WI_VAR_VAL(NUM)  
  
#define __WI_TYPEX(NUM) \
    __WI_VAR_TYPEX(NUM)  
#define __WI_TYPEX_REF(NUM)	\
    __WI_VAR_TYPEX(NUM)&  
#define __WI_TYPEX_REFREF(NUM)	\
    __WI_VAR_TYPEX(NUM)&&  
#define __WI_CONST_TYPEX_REF(NUM)	\
    const __WI_TYPEX(NUM)&  
#define __WI_TYPEX_ARG(NUM) \
    __WI_VAR_TYPEX(NUM) __WI_VAR_VALX(NUM)  
#define __WI_TYPEX_REF_ARG(NUM)	\
    __WI_TYPEX_REF(NUM) __WI_VAR_VALX(NUM)  
#define __WI_TYPEX_REFREF_ARG(NUM)	\
    __WI_TYPEX_REFREF(NUM) __WI_VAR_VALX(NUM)  
#define __WI_CONST_TYPEX_REF_ARG(NUM)	\
    const __WI_TYPEX(NUM)& __WI_VAR_VALX(NUM)  
#define __WI_DECAY_TYPEX(NUM) \
    typename decay<__WI_TYPEX(NUM)>::type  
  
// NB: Need an extra class to avoid specializing initial template declaration  
#define __WI_MAX_CLASS_LIST	\
    __WI_PAD_LIST0(__WI_CLASS_TYPE_NIL) __WI_COMMA class = _Nil  
#define __WI_MAX_NIL__WI_CLASS_LIST	\
    __WI_PAD_LIST0(__WI_CLASS_NIL) __WI_COMMA class = _Nil  
#define __WI_MAX_NIL_LIST	\
    __WI_PAD_LIST0(__WI_NIL_PAD) __WI_COMMA _Nil  
#define __WI_MAX_ALIST \
    __WI_PAD_LIST0(__WI_TYPE)  
 

// PARTIAL CONTENTS OF: <xtr1common>
// Only what was necessary for wistd::function

    // STRUCT _Nil
struct _Nil
    {	// empty struct, for unused argument types
    };
static _Nil _Nil_obj;


    // TEMPLATE STRUCT _Copy_cv
template<class _Tgt,
    class _Src>
    struct _Copy_cv
    {	// plain version
    typedef typename remove_reference<_Tgt>::type _Tgtx;
    typedef _Tgtx& type;
    };

template<class _Tgt,
    class _Src>
    struct _Copy_cv<_Tgt, const _Src>
    {	// const version
    typedef typename remove_reference<_Tgt>::type _Tgtx;
    typedef const _Tgtx& type;
    };

template<class _Tgt,
    class _Src>
    struct _Copy_cv<_Tgt, volatile _Src>
    {	// volatile version
    typedef typename remove_reference<_Tgt>::type _Tgtx;
    typedef volatile _Tgtx& type;
    };

template<class _Tgt,
    class _Src>
    struct _Copy_cv<_Tgt, const volatile _Src>
    {	// const volatile version
    typedef typename remove_reference<_Tgt>::type _Tgtx;
    typedef const volatile _Tgtx& type;
    };

template<class _Tgt,
    class _Src>
    struct _Copy_cv<_Tgt, _Src&>
    {	// remove reference from second argument
    typedef typename _Copy_cv<_Tgt, _Src>::type type;
    };

// CONTENTS OF: <xrefwrap>
// Most of xrefwrap has been pulled in

template<class _Ty>
    class reference_wrapper;

template<class, class = _Nil, class = _Nil, class = _Nil, __WI_MAX_CLASS_LIST>
    struct _Fun_class_base;

template<class _Ret,
    class _Farg0>
    struct _Fun_class_base<_Ret, _Farg0, _Nil, _Nil, __WI_MAX_NIL_LIST>
        : public unary_function<_Farg0, _Ret>
    {	// base if one argument
    typedef _Farg0 _Arg0;
    };

template<class _Ret,
    class _Farg0,
    class _Farg1>
    struct _Fun_class_base<_Ret, _Farg0, _Farg1, _Nil, __WI_MAX_NIL_LIST>
        : public binary_function<_Farg0, _Farg1, _Ret>
    {	// base if two arguments
    typedef _Farg0 _Arg0;
    };

#define __WI_CLASS_FUN__WI_CLASS_BASE( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
template<class _Ret, \
    class _Farg0, \
    class _Farg1, \
    class _Farg2 __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Fun_class_base<_Ret, _Farg0, _Farg1, _Farg2 \
        __WI_COMMA LIST(__WI_TYPE), PADDING_LIST(__WI_NIL_PAD)> \
    {	/* base if more than two arguments */ \
    typedef _Farg0 _Arg0; \
    };

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_FUN__WI_CLASS_BASE, , , , )
#undef __WI_CLASS_FUN__WI_CLASS_BASE

// IMPLEMENT result_of
        // TEMPLATE STRUCT _Get_callable_type
template<class _Fty>
    struct _Get_callable_type
    {	// get typeof _Fty() or _Fty
    template<class _Uty>
        static auto _Fn(int)
            -> typename decltype(wistd::declval<_Uty>()());

    template<class _Uty>
        static auto _Fn(_Wrap_int)
            -> _Fty;

    typedef decltype(_Fn<_Fty>(0)) type;
    };

template<bool,
    class _Fty, class = _Nil, __WI_MAX_CLASS_LIST>
    struct _Result_type;

template<class _Fty>
    struct _Result_type<false, _Fty, _Nil, __WI_MAX_NIL_LIST>
    {	// handle zero argument case
    typedef typename _Get_callable_type<_Fty>::type type;
    };

#define __WI_CLASS_RESULT_TYPE( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
template<class _Fty __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Result_type<true, _Fty \
        __WI_COMMA LIST(__WI_TYPE), PADDING_LIST(__WI_NIL_PAD), _Nil> \
    {	/* define return type for UDT with nested result_type */ \
    typedef typename _Fty::result_type type; \
    }; \
template<class _Fty, \
    class _Xarg0 __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Result_type<false, _Fty, _Xarg0 \
        __WI_COMMA LIST(__WI_TYPE), PADDING_LIST(__WI_NIL_PAD)> \
    {	/* define return type for UDT with no nested result_type */ \
    typedef decltype(wistd::declval<_Fty>()( \
        wistd::declval<_Xarg0>() __WI_COMMA LIST(__WI_DECLVAL))) type; \
    };

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_RESULT_TYPE, , , , )
#undef __WI_CLASS_RESULT_TYPE

template<class _Fty, __WI_MAX_CLASS_LIST>
    struct _Result_ofx;

template<class _Fty, __WI_MAX_CLASS_LIST,
    class _Obj = _Nil,
    class _Xarg0 = _Nil>
    struct _Result_of
    {	// finds result of calling _Fty with optional arguments
    typedef typename _Result_ofx<_Fty, __WI_MAX_ALIST>::type type;
    };

#define __WI_CLASS_RESULT_OFX( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
template<class _Fty __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Result_ofx<_Fty __WI_COMMA LIST(__WI_TYPE), PADDING_LIST(__WI_NIL_PAD)> \
    {	/* UDT */ \
    static const bool value = _Has_result_type<_Fty>::type::value; \
    typedef typename _Result_type< \
        _Result_ofx<_Fty __WI_COMMA LIST(__WI_TYPE)>::value, \
        _Fty __WI_COMMA LIST(__WI_TYPE)>::type type; \
    };

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_RESULT_OFX, , , , )
#undef __WI_CLASS_RESULT_OFX

#define __WI_CLASS_RESULT_OF_PF( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, X2, X3, X4) \
template<class _Ret __WI_COMMA LIST(__WI_CLASS_TYPE) __WI_COMMA LIST(__WI_CLASS_TYPEX)> \
    struct _Result_of<_Ret (CALL_OPT &)(LIST(__WI_TYPE)) \
        __WI_COMMA LIST(__WI_TYPEX), PADDING_LIST(__WI_NIL_PAD), _Nil, _Nil> \
    {	/* template to determine result of call operation */ \
        /* on ptr to function */ \
    typedef _Ret type; \
    }; \
template<class _Ret __WI_COMMA LIST(__WI_CLASS_TYPE) __WI_COMMA LIST(__WI_CLASS_TYPEX)> \
    struct _Result_of<_Ret (CALL_OPT *)(LIST(__WI_TYPE)) \
        __WI_COMMA LIST(__WI_TYPEX), PADDING_LIST(__WI_NIL_PAD), _Nil, _Nil> \
    {	/* template to determine result of call operation */ \
        /* on ptr to function */ \
    typedef _Ret type; \
    }; \
template<class _Ret __WI_COMMA LIST(__WI_CLASS_TYPE) __WI_COMMA LIST(__WI_CLASS_TYPEX)> \
    struct _Result_of<_Ret (CALL_OPT * const)(LIST(__WI_TYPE)) \
        __WI_COMMA LIST(__WI_TYPEX), PADDING_LIST(__WI_NIL_PAD), _Nil, _Nil> \
    {	/* template to determine result of call operation */ \
        /* on ptr to function */ \
    typedef _Ret type; \
    };

#define __WI_CLASS_RESULT_OF_PF_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_VARIADIC_CALL_OPT_X1(__WI_CLASS_RESULT_OF_PF, \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, \
            __cdecl, X2, X3, X4)

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_RESULT_OF_PF_OPT_0X, , , , )

#undef __WI_CLASS_RESULT_OF_PF_OPT_0X
#undef __WI_CLASS_RESULT_OF_PF

template<class _Ret,
    class _Ty,
    class _Obj>
    struct _Result_of<_Ret _Ty::*, _Obj, _Nil, __WI_MAX_NIL_LIST>
    {	// template to determine result of call operation
        // on pointer to member data
    typedef _Ret& type;
    };

template<class _Ret,
    class _Ty,
    class _Obj>
    struct _Result_of<_Ret _Ty::* const, _Obj, _Nil, __WI_MAX_NIL_LIST>
    {	// template to determine result of call operation
        // on pointer to member data
    typedef _Ret& type;
    };

template<class _Ret,
    class _Ty,
    class _Obj>
    struct _Result_of<_Ret _Ty::*, const _Obj, _Nil, __WI_MAX_NIL_LIST>
    {	// template to determine result of call operation
        // on pointer to member data
    typedef const _Ret& type;
    };

template<class _Ret,
    class _Ty,
    class _Obj>
    struct _Result_of<_Ret _Ty::* const, const _Obj, _Nil, __WI_MAX_NIL_LIST>
    {	// template to determine result of call operation
        // on pointer to member data
    typedef const _Ret& type;
    };

#define __WI_CLASS_RESULT_OF_PMF( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, CV_OPT, X3, X4) \
template<class _Ret, \
    class _Ty, \
    class _Tyx __WI_COMMA LIST(__WI_CLASS_TYPE) __WI_COMMA LIST(__WI_CLASS_TYPEX)> \
    struct _Result_of<_Ret (CALL_OPT _Ty::*)(LIST(__WI_TYPE)) CV_OPT, \
        _Tyx& __WI_COMMA LIST(__WI_TYPEX), PADDING_LIST(__WI_NIL_PAD), _Nil> \
    {	/* template to determine result of call operation */ \
        /* on pointer to member function */ \
    typedef _Ret type; \
    }; \
template<class _Ret, \
    class _Ty, \
    class _Tyx __WI_COMMA LIST(__WI_CLASS_TYPE) __WI_COMMA LIST(__WI_CLASS_TYPEX)> \
    struct _Result_of<_Ret (CALL_OPT _Ty::* const)(LIST(__WI_TYPE)) CV_OPT, \
        _Tyx& __WI_COMMA LIST(__WI_TYPEX), PADDING_LIST(__WI_NIL_PAD), _Nil> \
    {	/* template to determine result of call operation */ \
        /* on const pointer to member function */ \
    typedef _Ret type; \
    };

#define __WI_CLASS_RESULT_OF_PMF_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_VARIADIC_CALL_OPT_X2(__WI_CLASS_RESULT_OF_PMF, \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, \
            __thiscall, X2, X3, X4)

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_RESULT_OF_PMF_OPT_0X, , , , )
__WI_VARIADIC_EXPAND_0X(__WI_CLASS_RESULT_OF_PMF_OPT_0X, , const, , )
__WI_VARIADIC_EXPAND_0X(__WI_CLASS_RESULT_OF_PMF_OPT_0X, , volatile, , )
__WI_VARIADIC_EXPAND_0X(__WI_CLASS_RESULT_OF_PMF_OPT_0X, , const volatile, , )

#undef __WI_CLASS_RESULT_OF_PMF_OPT_0X
#undef __WI_CLASS_RESULT_OF_PMF

template<class _Fty>
    struct _Result_of0;

#define __WI_CLASS_RESULT_OF_FUN( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, X2, X3, X4) \
template<class _Fty __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Result_of0<_Fty CALL_OPT (LIST(__WI_TYPE))> \
    {	/* generic result_of */ \
    typedef typename _Result_of<_Fty \
        __WI_COMMA LIST(__WI_REMOVE_REF), PADDING_LIST(__WI_NIL_PAD), \
        _Nil, _Nil>::type type; \
    };

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_RESULT_OF_FUN, , , , )
#undef __WI_CLASS_RESULT_OF_FUN

    // TEMPLATE STRUCT result_of
template<class _Fty>
    struct result_of
    {	// template to determine result of call operation
    typedef typename _Result_of0<_Fty>::type type;
    };

// SUPPORT CLASSES FOR CALL WRAPPERS

    // TEMPLATE STRUCT _Pmd_caller
template<class _Ret,
    class _Arg0>
    struct _Pmd_caller
    {	// bind object and pointer to member data
    template<class _Pmd,
        class _Farg0>
        static _Ret _Call_pmd(_Pmd _Pm, _Farg0&& _Fx0, true_type)
        {	// apply to object
        return ((_Ret)(_Fx0.*_Pm));
        }

    template<class _Pmd,
        class _Farg0>
        static _Ret _Call_pmd(_Pmd _Pm, _Farg0&& _Fx0, false_type)
        {	// apply to (possibly smart) pointer
        return ((*_Fx0).*_Pm);
        }

    template<class _Pmd,
        class _Farg0>
        static _Ret _Apply_pmd(_Pmd _Pm, _Farg0&& _Fx0)
        {	// apply to object
        typedef typename remove_reference<_Arg0>::type _Arg0_bare;
        typedef typename remove_reference<_Farg0>::type _Farg0_bare;
        typedef _Cat_base<is_same<_Arg0_bare, _Farg0_bare>::value
            || is_base_of<_Arg0_bare, _Farg0_bare>::value
                && is_same<typename add_reference<_Farg0_bare>::type,
                    _Farg0>::value> _Is_obj;

        return (_Call_pmd<_Pmd, _Farg0&&>(_Pm,
            wistd::forward<_Farg0>(_Fx0), _Is_obj()));
        }
    };

    // TEMPLATE STRUCT _Callable_base
template<class _Ty,
    bool _Indirect>
    struct _Callable_base;

template<class _Ty>
    struct _Callable_base<_Ty, false>
    {	// base types for callable object wrappers
    enum {_EEN_INDIRECT = 0};	// helper for expression evaluator
    typedef _Ty _MyTy;
    typedef const _Ty& _MyCnstTy;

    _Callable_base(const _Callable_base& _Right)
        : _Object(_Right._Object)
        {	// copy construct
        }

    _Callable_base(_Callable_base&& _Right)
        : _Object(wistd::forward<_Ty>(_Right._Object))
        {	// move construct
        }

    template<class _Ty2>
        _Callable_base(_Ty2&& _Val)
        : _Object(wistd::forward<_Ty2>(_Val))
        {	// construct by forwarding
        }

    const _Ty& _Get() const
        {	// return reference to stored object
        return (_Object);
        }

    _Ty& _Get()
        {	// return reference to stored object
        return (_Object);
        }

private:
    _Callable_base& operator=(const _Callable_base&);

    _Ty _Object;
};

template<class _Ty>
    struct _Callable_base<_Ty, true>
    {	// base types for callable object wrappers holding references
        // (used by reference_wrapper)
    enum {_EEN_INDIRECT = 1};	// helper for expression evaluator
    typedef _Ty _MyTy;
    typedef _Ty& _MyCnstTy;

    _Callable_base(_Ty& _Val)
        : _Ptr(wistd::addressof(_Val))
        {	// construct
        }

    _MyCnstTy _Get() const
        {	// return reference to stored object
        return (*_Ptr);
        }

    _Ty& _Get()
        {	// return reference to stored object
        return (*_Ptr);
        }

    void _Reset(_Ty& _Val)
        {	// reseat reference
        _Ptr = wistd::addressof(_Val);
        }

private:
    _Ty *_Ptr;
};

    // TEMPLATE STRUCT _Callable_pmd
template<class _Ty,
    class _Memty,
    bool _Indirect = false>
    struct _Callable_pmd
        : _Callable_base<_Ty, _Indirect>
    {	// wrap pointer to member data
    _Callable_pmd(const _Callable_pmd& _Right)
        : _Callable_base<_Ty, _Indirect>(_Right._Get())
        {	// construct
        }

    _Callable_pmd(_Ty& _Val)
        : _Callable_base<_Ty, _Indirect>(_Val)
        {	// construct
        }

    template<class _Ret,
        class _Arg0>
        _Ret _ApplyX(_Arg0&& _A0) const
        {	// apply
        return (_Pmd_caller<_Ret, _Memty>::
            _Apply_pmd(this->_Get(), wistd::forward<_Arg0>(_A0)));
        }
    };

    // TEMPLATE STRUCT _Callable_obj
template<class _Ty,
    bool _Indirect = false>
    struct _Callable_obj
        : _Callable_base<_Ty, _Indirect>
    {	// wrap function object
    typedef _Callable_base<_Ty, _Indirect> _Mybase;

    _Callable_obj(_Callable_obj& _Right)
        : _Mybase(_Right._Get())
        {	// copy construct
        }

    _Callable_obj(_Callable_obj&& _Right)
        : _Mybase(wistd::forward<_Ty>(_Right._Get()))
        {	// move construct
        }

    template<class _Ty2>
        _Callable_obj(_Ty2&& _Val)
        : _Mybase(wistd::forward<_Ty2>(_Val))
        {	// construct
        }

#define __WI_APPLYX_CALLOBJ( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    template<class _Ret __WI_COMMA LIST(__WI_CLASS_TYPE)> \
        _Ret _ApplyX(LIST(__WI_TYPE_REFREF_ARG)) \
        { /* apply to UDT object */ \
        return (this->_Get()(LIST(__WI_FORWARD_ARG))); \
        } \
    template<class _Ret __WI_COMMA LIST(__WI_CLASS_TYPE)> \
        _Ret _ApplyX(LIST(__WI_TYPE_REFREF_ARG)) const \
        { /* apply to UDT object */ \
        return (this->_Get()(LIST(__WI_FORWARD_ARG))); \
        }

__WI_VARIADIC_EXPAND_0X(__WI_APPLYX_CALLOBJ, , , , )
#undef __WI_APPLYX_CALLOBJ
    };

    // TEMPLATE STRUCT _Pmf_caller
template<class _Ret,
    class _Arg0>
    struct _Pmf_caller
    {	// bind object and pointer to member function
    typedef _Arg0 _Funobj;

// apply to pointer to member function
#define __WI_PMF_CALLER_CALL_PMF( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, CV_OPT, X3, X4) \
    template<class _Pmf, \
        class _Farg0 __WI_COMMA LIST(__WI_CLASS_TYPE)> \
        static _Ret _Call_pmf(_Pmf _Pm, _Farg0 _Fx0, true_type \
            __WI_COMMA LIST(__WI_TYPE_REFREF_ARG)) \
        {	/* apply to object */ \
        typedef typename _Copy_cv<_Arg0, _Farg0>::type \
            _Funobj_cv; \
        return (((_Funobj_cv)_Fx0.*_Pm)(LIST(__WI_FORWARD_ARG))); \
        } \
    template<class _Pmf, \
        class _Farg0 __WI_COMMA LIST(__WI_CLASS_TYPE)> \
        static _Ret _Call_pmf(_Pmf _Pm, _Farg0&& _Fx0, false_type \
            __WI_COMMA LIST(__WI_TYPE_REFREF_ARG)) \
        {	/* apply to (possibly smart) pointer */ \
        return (((*_Fx0).*_Pm)(LIST(__WI_FORWARD_ARG))); \
        }

__WI_VARIADIC_EXPAND_0X(__WI_PMF_CALLER_CALL_PMF, , , , )
#undef __WI_PMF_CALLER_CALL_PMF

#define __WI_PMF_CALLER_APPLY_PMF( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, CV_OPT, X3, X4) \
    template<class _Pmf, \
        class _Farg0 __WI_COMMA LIST(__WI_CLASS_TYPE)>\
        static _Ret _Apply_pmf(_Pmf _Pm, \
            _Farg0&& _Fx0 __WI_COMMA LIST(__WI_TYPE_REFREF_ARG)) \
        {	/* apply to object */ \
        typedef typename remove_reference<_Arg0>::type _Arg0_bare0; \
        typedef typename remove_cv<_Arg0_bare0>::type _Arg0_bare; \
        typedef typename remove_reference<_Farg0>::type _Farg0_bare; \
        typedef _Cat_base<is_same<_Arg0_bare, _Farg0_bare>::value \
            || is_base_of<_Arg0_bare, _Farg0_bare>::value \
                && is_same<typename add_reference<_Farg0_bare>::type, \
                    _Farg0>::value> _Is_obj; \
        return (_Call_pmf<_Pmf, _Farg0&& __WI_COMMA LIST(__WI_TYPE_REFREF)>( \
            _Pm, wistd::forward<_Farg0>(_Fx0), _Is_obj() \
                __WI_COMMA LIST(__WI_FORWARD_ARG))); \
        }

__WI_VARIADIC_EXPAND_0X(__WI_PMF_CALLER_APPLY_PMF, , , , )
#undef __WI_PMF_CALLER_APPLY_PMF
    };

    // TEMPLATE STRUCT _Callable_pmf
template<class _Ty,
    class _Memty,
    bool _Indirect = false>
    struct _Callable_pmf
        : _Callable_base<_Ty, _Indirect>
    {	// wrap pointer to member function
    _Callable_pmf(const _Callable_pmf& _Right)
        : _Callable_base<_Ty, _Indirect>(_Right._Get())
        {	// construct
        }

    _Callable_pmf(_Ty& _Val)
        : _Callable_base<_Ty, _Indirect>(_Val)
        {	// construct
        }

#define __WI_CALLABLE_PMF_APPLYX( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, CV_OPT, X3, X4) \
    template<class _Ret, \
        class _Xarg0 __WI_COMMA LIST(__WI_CLASS_TYPE)> \
        _Ret _ApplyX(_Xarg0&& _Arg0 __WI_COMMA LIST(__WI_TYPE_REFREF_ARG)) const \
        { /* call pointer to member function */ \
        return (_Pmf_caller<_Ret, _Memty>:: \
            _Apply_pmf(this->_Get(), \
                wistd::forward<_Xarg0>(_Arg0) __WI_COMMA LIST(__WI_FORWARD_ARG))); \
        }

__WI_VARIADIC_EXPAND_0X(__WI_CALLABLE_PMF_APPLYX, , , , )
#undef __WI_CALLABLE_PMF_APPLYX
    };

    // TEMPLATE STRUCT _Callable_fun
template<class _Ty,
    bool _Indirect = false>
    struct _Callable_fun
        : _Callable_base<_Ty, _Indirect>
    {	// wrap pointer to function
    _Callable_fun(const _Callable_fun& _Right)
        : _Callable_base<_Ty, _Indirect>(_Right._Get())
        {	// construct
        }

    _Callable_fun(_Ty& _Val)
        : _Callable_base<_Ty, _Indirect>(_Val)
        {	// construct
        }

#define __WI_CALLABLE_FUN_APPLYX( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, CV_OPT, X3, X4) \
    template<class _Ret __WI_COMMA LIST(__WI_CLASS_TYPE)> \
        _Ret _ApplyX(LIST(__WI_TYPE_REFREF_ARG)) const \
        { /* call pointer to function */ \
        return (this->_Get()(LIST(__WI_FORWARD_ARG))); \
        }

__WI_VARIADIC_EXPAND_0X(__WI_CALLABLE_FUN_APPLYX, , , , )
#undef __WI_CALLABLE_FUN_APPLYX
    };

    // TEMPLATE STRUCT _Call_wrapper_base
template<class _Callable>
    struct _Call_wrapper_base
    {	// wrap callable object
    typedef typename _Callable::_MyTy _MyTy;
    typedef typename _Callable::_MyCnstTy _MyCnstTy;

    _Call_wrapper_base(_MyTy& _Val)
        : _Callee(_Val)
        {	// construct
        }

    void _Reset(_MyTy& _Val)
        {	// reset
        _Callee._Reset(_Val);
        }

    _MyCnstTy _Get() const
        {	// get
        return (_Callee._Get());
        }

    _MyCnstTy _Get()
        {	// get
        return (_Callee._Get());
        }

    _Callable _Callee;
    };

    // TEMPLATE STRUCT _Call_wrapper
template<class _Callable,
    bool _Is_abstract = false>
    struct _Call_wrapper
    : _Call_wrapper_base<_Callable>
    {	// wrap callable object
    typedef _Call_wrapper_base<_Callable> _Mybase;

    _Call_wrapper(typename _Call_wrapper_base<_Callable>::_MyTy& _Val)
        : _Call_wrapper_base<_Callable>(_Val)
        {	// construct
        }

#define __WI_CALL_WRAPPER_OP( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    TEMPLATE_LIST(__WI_CLASS_TYPE) \
        typename result_of< \
            typename _Callable::_MyTy(LIST(__WI_TYPE))>::type \
            operator()(LIST(__WI_TYPE_REFREF_ARG)) const \
        { /* call target object */ \
        typedef typename result_of< \
            typename _Callable::_MyTy(LIST(__WI_TYPE))>::type _Ret; \
        return (this->_Callee.template _ApplyX<_Ret>(LIST(__WI_FORWARD_ARG))); \
        }

__WI_VARIADIC_EXPAND_0X(__WI_CALL_WRAPPER_OP, , , , )
#undef __WI_CALL_WRAPPER_OP
    };

template<class _Callable>
    struct _Call_wrapper<_Callable, true>
    : _Call_wrapper_base<_Callable>
    {	// wrap abstract callable object
    typedef _Call_wrapper_base<_Callable> _Mybase;

    _Call_wrapper(typename _Call_wrapper_base<_Callable>::_MyTy& _Val)
        : _Call_wrapper_base<_Callable>(_Val)
        {	// construct
        }
    };

        // TEMPLATE STRUCT _Has_result_and_arg_type
template<class _Ty>
    struct _Has_result_and_arg_type
        __WI_HAS_TYPES(argument_type, result_type, result_type);

        // TEMPLATE STRUCT _Has_result_and_2arg_type
template<class _Ty>
    struct _Has_result_and_2arg_type
        __WI_HAS_TYPES(first_argument_type, second_argument_type, result_type);

    // TEMPLATE CLASS _Refwrap_result0
template<class _Ty,
    bool>
    struct _Refwrap_result0
    {	// define result_type when target object defines it
    typedef typename _Ty::result_type result_type;
    };

template<class _Ty>
    struct _Refwrap_result0<_Ty, false>
    {	// no result_type when not defined by target object
    };

// TEMPLATE CLASS _Refwrap_result1_helper
template<class _Ty,
    bool>
    struct _Refwrap_result1_helper
        : _Refwrap_result0<_Ty, _Has_result_type<_Ty>::type::value>
    {	// select _Refwrap_result0 specialization
    };

template<class _Ty>
    struct _Refwrap_result1_helper<_Ty, true>
        : unary_function<typename _Ty::argument_type,
            typename _Ty::result_type>
    {	// derive from unary_function
    };

    // TEMPLATE CLASS _Refwrap_result1
template<class _Ty,
    bool>
    struct _Refwrap_result1
        : _Refwrap_result0<_Ty, _Has_result_type<_Ty>::type::value>
    {	// select base for type without typedefs for result and one argument
    };

template<class _Ty>
    struct _Refwrap_result1<_Ty, true>
        : _Refwrap_result1_helper<_Ty,
            is_base_of<unary_function<
                typename _Ty::argument_type,
                typename _Ty::result_type>, _Ty>::value>
    {	// select base for type with typedefs for result and one argument
    };

    // TEMPLATE CLASS _Refwrap_result2_helper
template<class _Ty,
    bool>
    struct _Refwrap_result2_helper
        : _Refwrap_result1<_Ty, _Has_result_and_arg_type<_Ty>::type::value>
    {	// select base
    };

template<class _Ty>
    struct _Refwrap_result2_helper<_Ty, true>
        : binary_function<typename _Ty::first_argument_type,
            typename _Ty::second_argument_type,
            typename _Ty::result_type>,
        _Refwrap_result1<_Ty, _Has_result_and_arg_type<_Ty>::type::value>
    {	// base for type derived from binary_function
    };

    // TEMPLATE CLASS _Refwrap_result2
template<class _Ty,
    bool>
    struct _Refwrap_result2
        : _Refwrap_result1<_Ty, _Has_result_and_arg_type<_Ty>::type::value>
    {	// select base for type without typedefs for result and two arguments
    };

template<class _Ty>
    struct _Refwrap_result2<_Ty, true>
        : _Refwrap_result2_helper<_Ty,
            is_base_of<binary_function<
                typename _Ty::first_argument_type,
                typename _Ty::second_argument_type,
                typename _Ty::result_type>, _Ty>::value>
    {	// select base for type with typedefs for result and two arguments
    };

    // TEMPLATE CLASS _Refwrap_impl
template<class _Ty>
    struct _Refwrap_impl
        : _Call_wrapper<_Callable_obj<_Ty, true>,
            is_abstract<_Ty>::value>,
            _Refwrap_result2<_Ty, _Has_result_and_2arg_type<_Ty>::type::value>
    {	// reference_wrapper implementation for UDT
    _Refwrap_impl(_Ty& _Val)
        : _Call_wrapper<_Callable_obj<_Ty, true>,
            is_abstract<_Ty>::value>(_Val)
        {	// construct
        }
    };

template<class _Rx,
    class _Arg0>
    struct _Refwrap_impl<_Rx _Arg0::*>
        : _Call_wrapper<_Callable_pmd<_Rx _Arg0::*const, _Arg0, false> >
    {	// reference_wrapper implementation for pointer to member data
    typedef _Rx _Arg0::*const _Fty;
    typedef _Rx result_type;

    _Refwrap_impl(_Fty _Val)
        : _Call_wrapper<_Callable_pmd<_Fty, _Arg0, false> >(_Val)
        {	// construct
        }
    };

template<class _Rx,
    class _Arg0>
    struct _Refwrap_impl<_Rx _Arg0::*const>
        : _Call_wrapper<_Callable_pmd<_Rx _Arg0::*, _Arg0, false> >
    {	// reference_wrapper implementation for const pointer to member data
    typedef _Rx _Arg0::*_Fty;
    typedef _Rx result_type;
    _Refwrap_impl(_Fty _Val)
        : _Call_wrapper<_Callable_pmd<_Fty, _Arg0, false> >(_Val)
        {	// construct
        }
    };

    // TEMPLATE CLASS _Refwrap_impl
    // TEMPLATE CLASS _Refwrap_impl FOR FUNCTIONS, POINTERS TO FUNCTIONS
#define __WI_CLASS_REFWRAP_IMPL0( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, X2, X3, X4) \
template<class _Rx __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Refwrap_impl<_Rx CALL_OPT (LIST(__WI_TYPE))> \
        : _Call_wrapper<_Callable_fun< \
                _Rx(CALL_OPT *)(LIST(__WI_TYPE)), false> >, \
            _Fun_class_base<_Rx __WI_COMMA LIST(__WI_TYPE), \
                PADDING_LIST(__WI_NIL_PAD), _Nil, _Nil, _Nil> \
    {	/* implement for pointer to function */ \
    typedef _Rx(CALL_OPT *_Fty)(LIST(__WI_TYPE)); \
    typedef _Rx result_type; \
    _Refwrap_impl(_Fty _Val) \
        : _Call_wrapper<_Callable_fun<_Fty, false> >(_Val) \
        {	/* construct */ \
        } \
    }; \
template<class _Rx __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Refwrap_impl<_Rx(CALL_OPT *)(LIST(__WI_TYPE))> \
        : _Call_wrapper<_Callable_fun< \
                _Rx(CALL_OPT *)(LIST(__WI_TYPE)), true> >, \
            _Fun_class_base<_Rx __WI_COMMA LIST(__WI_TYPE), \
                PADDING_LIST(__WI_NIL_PAD), _Nil, _Nil, _Nil> \
    {	/* implement for pointer to function */ \
    typedef _Rx(CALL_OPT *_Fty)(LIST(__WI_TYPE)); \
    typedef _Rx result_type; \
    _Refwrap_impl(_Fty& _Val) \
        : _Call_wrapper<_Callable_fun<_Fty, true> >(_Val) \
        {	/* construct */ \
        } \
    }; \
template<class _Rx __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Refwrap_impl<_Rx(CALL_OPT * const)(LIST(__WI_TYPE))> \
        : _Call_wrapper<_Callable_fun< \
                _Rx(CALL_OPT * const)(LIST(__WI_TYPE)), true> >, \
            _Fun_class_base<_Rx __WI_COMMA LIST(__WI_TYPE), \
                PADDING_LIST(__WI_NIL_PAD), _Nil, _Nil, _Nil> \
    {	/* implement for pointer to function */ \
    typedef _Rx(CALL_OPT * const _Fty)(LIST(__WI_TYPE)); \
    typedef _Rx result_type; \
    _Refwrap_impl(_Fty& _Val) \
        : _Call_wrapper<_Callable_fun<_Fty, true> >(_Val) \
        {	/* construct */ \
        } \
    };

#define __WI_CLASS_REFWRAP_IMPL0_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_VARIADIC_CALL_OPT_X1(__WI_CLASS_REFWRAP_IMPL0, \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, \
            __cdecl, X2, X3, X4)

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_REFWRAP_IMPL0_OPT_0X, , , , )
#undef __WI_CLASS_REFWRAP_IMPL0_OPT_0X
#undef __WI_CLASS_REFWRAP_IMPL0

    // TEMPLATE CLASS _Refwrap_impl FOR POINTERS TO MEMBER FUNCTIONS
#define __WI_CLASS_REFWRAP_IMPL1( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, CV_OPT, X3, X4) \
template<class _Rx, \
    class _Arg0 __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Refwrap_impl<_Rx(CALL_OPT _Arg0::*)(LIST(__WI_TYPE)) CV_OPT> \
        : _Call_wrapper<_Callable_pmf< \
            _Rx(CALL_OPT _Arg0::*)(LIST(__WI_TYPE)) CV_OPT, _Arg0, true> >, \
                _Fun_class_base<_Rx, _Arg0 __WI_COMMA LIST(__WI_TYPE)> \
    {	/* implement for pointer to member function */ \
    typedef _Rx(CALL_OPT _Arg0::* _Fty)(LIST(__WI_TYPE)); \
    typedef _Rx result_type; \
    _Refwrap_impl(_Fty& _Val) \
        : _Call_wrapper<_Callable_pmf<_Fty, _Arg0, true> >(_Val) \
        {	/* construct */ \
        } \
    }; \
template<class _Rx, \
    class _Arg0 __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Refwrap_impl<_Rx(CALL_OPT _Arg0::* const)(LIST(__WI_TYPE)) CV_OPT> \
        : _Call_wrapper<_Callable_pmf< \
            _Rx(CALL_OPT _Arg0::* const)(LIST(__WI_TYPE)) CV_OPT, _Arg0, true> >, \
                _Fun_class_base<_Rx, _Arg0 __WI_COMMA LIST(__WI_TYPE)> \
    {	/* implement for pointer to member function */ \
    typedef _Rx(CALL_OPT _Arg0::* const _Fty)(LIST(__WI_TYPE)) CV_OPT; \
    typedef _Rx result_type; \
    _Refwrap_impl(_Fty& _Val) \
        : _Call_wrapper<_Callable_pmf<_Fty, _Arg0, true> >(_Val) \
        {	/* construct */ \
        } \
    };

#define __WI_CLASS_REFWRAP_IMPL1_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_VARIADIC_CALL_OPT_X1(__WI_CLASS_RESULT_OF_PMF, \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, \
            __cdecl, X2, X3, X4)

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_REFWRAP_IMPL1, , , , )
__WI_VARIADIC_EXPAND_0X(__WI_CLASS_REFWRAP_IMPL1, , const, , )
__WI_VARIADIC_EXPAND_0X(__WI_CLASS_REFWRAP_IMPL1, , volatile, , )
__WI_VARIADIC_EXPAND_0X(__WI_CLASS_REFWRAP_IMPL1, , const volatile, , )

#undef __WI_CLASS_REFWRAP_IMPL1_OPT_0X
#undef __WI_CLASS_REFWRAP_IMPL1

    // TEMPLATE CLASS reference_wrapper
template<class _Ty>
    class reference_wrapper
    : public _Refwrap_impl<_Ty>
    {	// stand-in for an assignable reference
public:
    typedef _Refwrap_impl<_Ty> _MyBase;
    typedef _Ty type;

    reference_wrapper(_Ty& _Val) WI_NOEXCEPT
        : _MyBase(_Val)
        {	// construct
        }

    operator _Ty&() const WI_NOEXCEPT
        {	// return reference
        return (this->_Get());
        }

    _Ty& get() const WI_NOEXCEPT
        {	// return reference
        return (this->_Get());
        }
    };

    // TEMPLATE FUNCTIONS ref AND cref
template<class _Ty>
    reference_wrapper<_Ty>
        ref(_Ty& _Val) WI_NOEXCEPT
    {	// create reference_wrapper<_Ty> object
    return (reference_wrapper<_Ty>(_Val));
    }

template<class _Ty>
    void ref(const _Ty&& _Val);	// not defined

template<class _Ty>
    reference_wrapper<_Ty>
        ref(reference_wrapper<_Ty> _Val) WI_NOEXCEPT
    {	// create reference_wrapper<_Ty> object
    return (_Val);
    }

template<class _Ty>
    reference_wrapper<const _Ty>
        cref(const _Ty& _Val) WI_NOEXCEPT
    {	// create reference_wrapper<const _Ty> object
    return (reference_wrapper<const _Ty>(_Val));
    }

//template<class _Ty>
//	void cref(const _Ty&& _Val);	// not defined

template<class _Ty>
    reference_wrapper<const _Ty>
        cref(reference_wrapper<const _Ty> _Val) WI_NOEXCEPT
    {	// create reference_wrapper<const _Ty> object
    return (_Val);
    }

// CONTENTS OF: <functional>
// All of the std::function work, leave behind std::bind

// IMPLEMENT mem_fn
    // TEMPLATE FUNCTION mem_fn
template<class _Rx,
    class _Arg0>
    _Call_wrapper<_Callable_pmd<_Rx _Arg0::*const, _Arg0> >
        mem_fn(_Rx _Arg0::*const _Pmd)
    {	// return data object wrapper
    return (_Call_wrapper<_Callable_pmd<_Rx _Arg0::*const, _Arg0> >(_Pmd));
    }

    // TEMPLATE CLASS _Mem_fn_wrap
template<class _Rx,
    class _Pmf,
    class _Arg0, __WI_MAX_CLASS_LIST>
    class _Mem_fn_wrap;

template<class _Rx,
    class _Pmf,
    class _Arg0>
    class _Mem_fn_wrap<_Rx, _Pmf, _Arg0>
        : public _Call_wrapper<_Callable_pmf<_Pmf, _Arg0> >,
            public _Fun_class_base<_Rx, _Arg0>
    {	// wrap pointer to member function, one argument
public:
    typedef _Rx result_type;
    typedef _Arg0 *argument_type;

    _Mem_fn_wrap(_Pmf _Fx)
        : _Call_wrapper<_Callable_pmf<_Pmf, _Arg0> >(_Fx)
        {	// construct
        }
    };

#define __WI_CLASS_MEM_FN_WRAP( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
template<class _Rx, \
    class _Pmf, \
    class _Arg0, \
    class _Arg1 __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    class _Mem_fn_wrap<_Rx, _Pmf, _Arg0, _Arg1 __WI_COMMA LIST(__WI_TYPE), \
        PADDING_LIST(__WI_NIL_PAD)> \
        : public _Call_wrapper<_Callable_pmf<_Pmf, _Arg0> >, \
            public _Fun_class_base<_Rx, _Arg0, _Arg1 __WI_COMMA LIST(__WI_TYPE)> \
    {	/* wrap pointer to member function, two or more arguments */ \
public: \
    typedef _Rx result_type; \
    typedef _Arg0 *first_argument_type; \
    typedef _Arg1 second_argument_type; \
    _Mem_fn_wrap(_Pmf _Fx) \
        : _Call_wrapper<_Callable_pmf<_Pmf, _Arg0> >(_Fx) \
        {	/* construct */ \
        } \
    };

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_MEM_FN_WRAP, , , , )
#undef __WI_CLASS_MEM_FN_WRAP

    // TEMPLATE FUNCTION mem_fn, pointer to member function
#define __WI_MEM_FN_IMPL( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, CV_OPT, X3, X4) \
template<class _Rx, \
    class _Arg0 __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    _Mem_fn_wrap<_Rx, _Rx(_Arg0::*)(LIST(__WI_TYPE)) CV_OPT CALL_OPT, \
        CV_OPT _Arg0 __WI_COMMA LIST(__WI_TYPE)> \
            mem_fn(_Rx(_Arg0::*_Pm)(LIST(__WI_TYPE)) CV_OPT CALL_OPT) \
    {	/* bind to pointer to member function */ \
    return (_Mem_fn_wrap<_Rx, _Rx(_Arg0::*)(LIST(__WI_TYPE)) CV_OPT CALL_OPT, \
        CV_OPT _Arg0 __WI_COMMA LIST(__WI_TYPE)>(_Pm)); \
    }

#define __WI_VARIADIC_MEM_FN_IMPL(FUNC, CALL_OPT) \
__WI_VARIADIC_EXPAND_0X(FUNC, CALL_OPT, , , ) \
__WI_VARIADIC_EXPAND_0X(FUNC, CALL_OPT, const, , ) \
__WI_VARIADIC_EXPAND_0X(FUNC, CALL_OPT, volatile, , ) \
__WI_VARIADIC_EXPAND_0X(FUNC, CALL_OPT, const volatile, , )

__WI_VARIADIC_MEM_FN_IMPL(__WI_MEM_FN_IMPL, )

#undef __WI_VARIADIC_MEM_FN_IMPL
#undef __WI_MEM_FN_IMPL

// IMPLEMENT function

__declspec(noreturn)
inline void _Xbad_function_call()
{
    // Attempting to call through a wistd::function<> object that is not currently assigned
    __fastfail(FAST_FAIL_FATAL_APP_EXIT);
}

    // TEMPLATE CLASS _Func_base
template<class _Rx,
    __WI_MAX_CLASS_LIST>
    class _Func_base;

#define __WI_CLASS_FUNC_BASE( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
template<class _Rx __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    class _Func_base<_Rx __WI_COMMA LIST(__WI_TYPE), PADDING_LIST(__WI_NIL_PAD)> \
    {	/* abstract base for implementation types */ \
public: \
    typedef _Func_base<_Rx __WI_COMMA LIST(__WI_TYPE)> _Myt; \
    virtual _Myt *_Copy(void *) = 0; \
    virtual _Myt *_Move(void *) = 0; \
    virtual _Rx _Do_call(LIST(__WI_TYPE_REFREF)) = 0; \
    virtual void _Delete_this(bool) = 0; \
    virtual ~_Func_base() WI_NOEXCEPT \
        {	/* destroy the object */ \
        } \
private: \
    virtual const void *_Get() const = 0; \
    };

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_FUNC_BASE, , , , )
#undef __WI_CLASS_FUNC_BASE

    // TEMPLATE CLASS _Func_impl
template<class _Callable,
    class _Alloc,
    class _Rx,
    __WI_MAX_CLASS_LIST>
    class _Func_impl;

#define __WI_CLASS_FUNC_IMPL( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
template<class _Callable, \
    class _Alloc, \
    class _Rx __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    class _Func_impl<_Callable, _Alloc, _Rx __WI_COMMA LIST(__WI_TYPE), \
        PADDING_LIST(__WI_NIL_PAD)> \
        : public _Func_base<_Rx __WI_COMMA LIST(__WI_TYPE)> \
    {	/* derived class for specific implementation types */ \
public: \
    typedef _Func_impl<_Callable, _Alloc, _Rx __WI_COMMA LIST(__WI_TYPE)> _Myt; \
    typedef _Func_base<_Rx __WI_COMMA LIST(__WI_TYPE)> _Mybase; \
    typedef typename _Alloc::template rebind<_Func_impl>::other _Myalty; \
    _Func_impl(const _Func_impl& _Right) \
        : _Callee(_Right._Callee), \
            _Myal(_Right._Myal) \
        {	/* copy construct */ \
        } \
    _Func_impl(_Func_impl& _Right) \
        : _Callee(_Right._Callee), \
            _Myal(_Right._Myal) \
        {	/* copy construct */ \
        } \
    _Func_impl(_Func_impl&& _Right) \
        : _Callee(wistd::forward<_Callable>(_Right._Callee)), \
            _Myal(_Right._Myal) \
        {	/* move construct */ \
        } \
    _Func_impl(typename _Callable::_MyTy&& _Val, \
        const _Myalty& _Ax = _Myalty()) \
        : _Callee(wistd::forward<typename _Callable::_MyTy>(_Val)), _Myal(_Ax) \
        {	/* construct */ \
        } \
    template<class _Other> \
        _Func_impl(_Other&& _Val, \
            const _Myalty& _Ax = _Myalty()) \
        : _Callee(wistd::forward<_Other>(_Val)), _Myal(_Ax) \
        {	/* construct */ \
        } \
    virtual _Mybase *_Copy(void *_Where) \
        {	/* return clone of *this */ \
        if (_Where == 0) \
            _Where = _Myal.allocate(1); \
        ::new (_Where) _Myt(_Callee); \
        return ((_Mybase *)_Where); \
        } \
    virtual _Mybase *_Move(void *_Where) \
        {	/* return clone of *this */ \
        if (_Where == 0) \
            _Where = _Myal.allocate(1); \
        ::new (_Where) _Myt(wistd::move(_Callee)); \
        return ((_Mybase *)_Where); \
        } \
    virtual ~_Func_impl() WI_NOEXCEPT \
        {	/* destroy the object */ \
        } \
    virtual _Rx _Do_call(LIST(__WI_TYPE_REFREF_ARG)) \
        {	/* call wrapped function */ \
        return (_Callee.template _ApplyX<_Rx>( \
            LIST(__WI_FORWARD_ARG))); \
        } \
private: \
    virtual const void *_Get() const \
        {	/* return address of stored object */ \
        return (reinterpret_cast<const void*>(&_Callee._Get())); \
        } \
    virtual void _Delete_this(bool _Deallocate) \
        {	/* destroy self */ \
        _Myalty _Al = _Myal; \
        _Al.destroy(this); \
        if (_Deallocate) \
            _Al.deallocate(this, 1); \
        } \
    _Callable _Callee; \
    _Myalty _Myal; \
    };

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_FUNC_IMPL, , , , )
#undef __WI_CLASS_FUNC_IMPL

    // TEMPLATE CLASS _Func_class
template<class _Ret,
    __WI_MAX_CLASS_LIST>
    class _Func_class;

#define __WI_CLASS_FUNC__WI_CLASS_BEGIN_0X( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
template<class _Ret __WI_COMMA LIST(__WI_CLASS_TYPEX)> \
    class _Func_class<_Ret __WI_COMMA LIST(__WI_TYPEX), PADDING_LIST(__WI_NIL_PAD)> \
        : public _Fun_class_base<_Ret __WI_COMMA LIST(__WI_TYPEX)> \
    {	/* implement function template */ \
public: \
    typedef _Func_class<_Ret __WI_COMMA LIST(__WI_TYPEX)> _Myt; \
    typedef typename _Fun_class_base<_Ret __WI_COMMA LIST(__WI_TYPEX)>::_Arg0 _Arg0; \
    typedef _Func_base<_Ret __WI_COMMA LIST(__WI_TYPEX)> _Ptrt; \
    typedef _Ret result_type; \
    _Func_class() : _Impl(0) { } \
    _Ret operator()(LIST(__WI_TYPEX_ARG)) const \
        {	/* call through stored object */ \
        if (_Impl == 0) \
            _Xbad_function_call(); \
        return (_Impl->_Do_call(LIST(__WI_FORWARD_ARGX))); \
        } \
    bool _Empty() const \
        {	/* return true if no stored object */ \
        return (_Impl == 0); \
        } \
    ~_Func_class() WI_NOEXCEPT \
        {	/* destroy the object */ \
        _Tidy(); \
        } \
protected: \
    void _Reset() \
        {	/* remove stored object */ \
        _Set(0); \
        } \
        void _Reset(const _Myt& _Right) \
        {	/* copy _Right's stored object */ \
        if (_Right._Impl == 0) \
            _Set(0); \
        else if (_Right._Local()) \
            _Set(_Right._Impl->_Copy((void *)&_Space)); \
        else \
            _Set(_Right._Impl->_Copy(0)); \
        } \
    void _Resetm(_Myt&& _Right) \
        {	/* move _Right's stored object */ \
        if (_Right._Impl == 0) \
            _Set(0); \
        else if (_Right._Local()) \
            {	/* move and tidy */ \
            _Set(_Right._Impl->_Move((void *)&_Space)); \
            _Right._Tidy(); \
            } \
        else \
            {	/* steal from _Right */ \
            _Set(_Right._Impl); \
            _Right._Set(0); \
            } \
        } \
    template<class _Fty> \
        void _Reset(_Fty&& _Val) \
        {	/* store copy of _Val */ \
        _Reset_alloc(wistd::forward<_Fty>(_Val), details::function_allocator<_Myt>()); \
        } \
    template<class _Fty, \
        class _Alloc> \
        void _Reset_alloc(_Fty&& _Val, _Alloc _Ax) \
        {	/* store copy of _Val with allocator */ \
        typedef _Callable_obj<typename decay<_Fty>::type> \
            _MyWrapper; \
        typedef _Func_impl<_MyWrapper, _Alloc, \
            _Ret __WI_COMMA LIST(__WI_TYPEX)> _Myimpl; \
        _Do_alloc<_Myimpl>(wistd::forward<_Fty>(_Val), _Ax); \
        }

// CALL_OPT = (__cdecl, VC++ variants)
#define __WI_CLASS_FUNC__WI_CLASS_MIDDLE0_0X( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, X2, X3, X4) \
    template<class _Fret __WI_COMMA LIST(__WI_CLASS_TYPE)> \
        void _Reset(_Fret (CALL_OPT *const _Val)(LIST(__WI_TYPE_ARG))) \
        {	/* store copy of _Val */ \
        _Reset_alloc(_Val, details::function_allocator<_Myt>()); \
        } \
    template<class _Fret __WI_COMMA LIST(__WI_CLASS_TYPE), \
        class _Alloc> \
        void _Reset_alloc(_Fret (CALL_OPT *const _Val)(LIST(__WI_TYPE_ARG)), \
            _Alloc _Ax) \
        {	/* store copy of _Val with allocator */ \
        typedef _Callable_fun<_Fret (CALL_OPT *const)(LIST(__WI_TYPE))> \
            _MyWrapper; \
        typedef _Func_impl<_MyWrapper, _Alloc, _Ret __WI_COMMA LIST(__WI_TYPEX)> \
            _Myimpl; \
        _Do_alloc<_Myimpl>(_Val, _Ax); \
        }

#define __WI_CLASS_FUNC__WI_CLASS_MIDDLE1_1( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    template<class _Fret, \
        class _Farg0> \
        void _Reset(_Fret _Farg0::*const _Val) \
        {	/* store copy of _Val */ \
        _Reset_alloc(_Val, details::function_allocator<_Myt>()); \
        } \
    template<class _Fret, \
        class _Farg0, \
        class _Alloc> \
        void _Reset_alloc(_Fret _Farg0::*const _Val, _Alloc _Ax) \
        {	/* store copy of _Val with allocator */ \
        typedef _Callable_pmd<_Fret _Farg0::*const, _Farg0> \
            _MyWrapper; \
        typedef _Func_impl<_MyWrapper, _Alloc, _Ret, _Arg0> \
            _Myimpl; \
        _Do_alloc<_Myimpl>(_Val, _Ax); \
        }

        // CALL_OPT = (__thiscall, VC++ variants)
        // CV_OPT = {, const, volatile, const volatile}
#define __WI_CLASS_FUNC__WI_CLASS_MIDDLE2_0X( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, \
    CALL_OPT, CV_OPT, LIST_P1, COMMA_P1) \
    template<class _Fret, \
        class _Farg0 COMMA_P1 LIST_P1(__WI_CLASS_TYPE)> \
        void _Reset(_Fret(CALL_OPT _Farg0::*const _Val)(\
        LIST_P1(__WI_TYPE)) CV_OPT) \
        {	/* store copy of _Val */ \
        _Reset_alloc(_Val, details::function_allocator<_Myt>()); \
        } \
        template<class _Fret, \
        class _Farg0 COMMA_P1 LIST_P1(__WI_CLASS_TYPE), \
        class _Alloc> \
        void _Reset_alloc(_Fret(CALL_OPT _Farg0::*const _Val)(\
        LIST_P1(__WI_TYPE)) CV_OPT, _Alloc _Ax) \
        {	/* store copy of _Val */ \
        typedef _Callable_pmf< \
        _Fret(CALL_OPT _Farg0::*const)(LIST_P1(__WI_TYPE)) CV_OPT, _Farg0> \
        _MyWrapper; \
        typedef _Func_impl<_MyWrapper, _Alloc, _Ret, \
        _Farg0 COMMA_P1 LIST_P1(__WI_TYPE)> _Myimpl; \
        _Do_alloc<_Myimpl>(_Val, _Ax); \
        } 

#define __WI_CLASS_FUNC__WI_CLASS_END_0X( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    void _Tidy() \
        {	/* clean up */ \
        if (_Impl != 0) \
        {	/* destroy callable object and maybe delete it */ \
        _Impl->_Delete_this(!_Local()); \
        _Impl = 0; \
        } \
        } \
        void _Swap(_Myt& _Right) \
        {	/* swap contents with contents of _Right */ \
        if (this == &_Right) \
        ;	/* same object, do nothing */ \
        else if (!_Local() && !_Right._Local()) \
        wistd::swap_wil(_Impl, _Right._Impl);	/* just swap pointers */ \
        else \
        {	/* do three-way copy */ \
        _Myt _Temp; \
        _Temp._Resetm(wistd::forward<_Myt>(*this)); \
        _Tidy(); \
        _Resetm(wistd::forward<_Myt>(_Right)); \
        _Right._Tidy(); \
        _Right._Resetm(wistd::forward<_Myt>(_Temp)); \
        } \
        } \
private: \
    template<class _Myimpl, \
        class _Fty, \
        class _Alloc> \
        void _Do_alloc(_Fty&& _Val, \
        _Alloc _Ax) \
        {	/* store copy of _Val with allocator */ \
        void *_Vptr = 0; \
        _Myimpl *_Ptr = 0; \
        static_assert(sizeof(_Myimpl) <= sizeof(_Space), "The sizeof(wistd::function) has grown too large for the reserved buffer (10 pointers).  Refactor to reduce size of the capture." ); \
        /* small enough, allocate locally */ \
        _Vptr = &_Space; \
        _Ptr = ::new (_Vptr) _Myimpl(wistd::forward<_Fty>(_Val)); \
        _Set(_Ptr); \
        } \
    void _Set(_Ptrt *_Ptr) \
        {	/* store pointer to object */ \
        _Impl = _Ptr; \
        } \
    bool _Local() const \
        {	/* test for locally stored copy of object */ \
        return ((void *)_Impl == (void *)&_Space); \
        } \
    typedef void (*_Pfnty)(); \
    union \
        {	/* storage for small wrappers */ \
        _Pfnty _Pfn[3]; \
        void *_Pobj[3]; \
        long double _Ldbl;	/* for maximum alignment */ \
        char _Alias[3 * sizeof (void *)];	/* to permit aliasing */ \
        char _Buffer[12 * sizeof(void *)];  /* ensure stack-transfer of lambda can hold at least 10 pointers (callable wrapper takes 2) */ \
        } _Space; \
    _Ptrt *_Impl; \
    };

#define __WI_CLASS_FUNC__WI_CLASS_MIDDLE0_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_VARIADIC_CALL_OPT_X1(__WI_CLASS_FUNC__WI_CLASS_MIDDLE0_0X, \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, \
            __cdecl, , X3, X4)

#define __WI_CLASS_FUNC__WI_CLASS_MIDDLE2_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_VARIADIC_CALL_OPT_X2(__WI_CLASS_FUNC__WI_CLASS_MIDDLE2_0X, \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, \
            __thiscall, , X3, X4) \
    __WI_VARIADIC_CALL_OPT_X2(__WI_CLASS_FUNC__WI_CLASS_MIDDLE2_0X, \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, \
            __thiscall, const, X3, X4) \
    __WI_VARIADIC_CALL_OPT_X2(__WI_CLASS_FUNC__WI_CLASS_MIDDLE2_0X, \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, \
            __thiscall, volatile, X3, X4) \
    __WI_VARIADIC_CALL_OPT_X2(__WI_CLASS_FUNC__WI_CLASS_MIDDLE2_0X, \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, \
            __thiscall, const volatile, X3, X4)

#define __WI_CLASS_FUNC__WI_CLASS_0( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_BEGIN_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_MIDDLE0_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_MIDDLE2_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_END_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \

#define __WI_CLASS_FUNC__WI_CLASS_1( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_BEGIN_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_MIDDLE0_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_MIDDLE1_1( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_MIDDLE2_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_END_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4)

#define __WI_CLASS_FUNC__WI_CLASS_2X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_BEGIN_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_MIDDLE0_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_MIDDLE2_OPT_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4) \
    __WI_CLASS_FUNC__WI_CLASS_END_0X( \
        TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, X1, X2, X3, X4)

__WI_VARIADIC_EXPAND_P1_0(__WI_CLASS_FUNC__WI_CLASS_0, , , , )
__WI_VARIADIC_EXPAND_P1_1(__WI_CLASS_FUNC__WI_CLASS_1, , , , )
__WI_VARIADIC_EXPAND_P1_2X(__WI_CLASS_FUNC__WI_CLASS_2X, , , , )

#undef __WI_CLASS_FUNC__WI_CLASS_BEGIN_0X
#undef __WI_CLASS_FUNC__WI_CLASS_MIDDLE0_0X
#undef __WI_CLASS_FUNC__WI_CLASS_MIDDLE1_1
#undef __WI_CLASS_FUNC__WI_CLASS_MIDDLE2_0X
#undef __WI_CLASS_FUNC__WI_CLASS_END_0X
#undef __WI_CLASS_FUNC__WI_CLASS_MIDDLE0_OPT_0X
#undef __WI_CLASS_FUNC__WI_CLASS_MIDDLE2_OPT_0X

#undef __WI_CLASS_FUNC__WI_CLASS_0
#undef __WI_CLASS_FUNC__WI_CLASS_1
#undef __WI_CLASS_FUNC__WI_CLASS_2X

    // TEMPLATE CLASS _Get_function_impl
template<class _Tx>
    struct _Get_function_impl;

#define __WI_CLASS_GET_FUNCTION_IMPL( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, X2, X3, X4) \
template<class _Ret __WI_COMMA LIST(__WI_CLASS_TYPE)> \
    struct _Get_function_impl<_Ret CALL_OPT (LIST(__WI_TYPE))> \
    {	/* determine type from argument list */ \
    typedef _Func_class<_Ret __WI_COMMA LIST(__WI_TYPE)> type; \
    };

#define __WI_CLASS_GET_FUNCTION_IMPL_CALLS( \
    TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, CALL_OPT, X2, X3, X4) \
        __WI_VARIADIC_CALL_OPT_X1(__WI_CLASS_GET_FUNCTION_IMPL, \
            TEMPLATE_LIST, PADDING_LIST, LIST, __WI_COMMA, __cdecl, X2, X3, X4)

__WI_VARIADIC_EXPAND_0X(__WI_CLASS_GET_FUNCTION_IMPL_CALLS, , , , )
#undef __WI_CLASS_GET_FUNCTION_IMPL_CALLS
#undef __WI_CLASS_GET_FUNCTION_IMPL

    // TEMPLATE CLASS function
template<class _Fty>
    class function
        : public _Get_function_impl<_Fty>::type
    {	// wrapper for callable objects
public:
    typedef function<_Fty> _Myt;
    typedef typename _Get_function_impl<_Fty>::type _Mybase;

	function() WI_NOEXCEPT
		{	// construct empty function wrapper
		this->_Reset();
		}

	function(nullptr_t) WI_NOEXCEPT
		{	// construct empty function wrapper from null pointer
		this->_Reset();
		}

	function(const _Myt& _Right)
		{	// construct holding copy of _Right
		this->_Reset((const _Mybase&)_Right);
		}

	function(_Myt& _Right)
		{	// construct holding copy of _Right
		this->_Reset((const _Mybase&)_Right);
		}

	function(const _Myt&& _Right)
		{	// construct holding copy of _Right
		this->_Reset((const _Mybase&)_Right);
		}

	template<class _Fx>
		function(const _Fx& _Func)
		{	// construct wrapper holding copy of _Func
		this->_Reset(_Func);
		}

	template<class _Fx>
		function(reference_wrapper<_Fx> _Func)
		{	// construct wrapper holding reference to _Func
		this->_Reset(_Func);
		}

	~function() WI_NOEXCEPT
		{	// destroy the object
		this->_Tidy();
		}

	_Myt& operator=(const _Myt& _Right)
		{	// assign _Right
		if (this != &_Right)
			{	// clean up and copy
			this->_Tidy();
			this->_Reset((const _Mybase&)_Right);
			}
		return (*this);
		}

	_Myt& operator=(_Myt& _Right)
		{	// assign _Right
		if (this != &_Right)
			{	// clean up and copy
			this->_Tidy();
			this->_Reset((const _Mybase&)_Right);
			}
		return (*this);
		}

	function(_Myt&& _Right)
		{	// construct holding moved copy of _Right
		this->_Resetm(wistd::forward<_Myt>(_Right));
		}

	template<class _Fx>
		function(_Fx&& _Func)
		{	// construct wrapper holding moved _Func
		this->_Reset(wistd::forward<_Fx>(_Func));
		}

	_Myt& operator=(_Myt&& _Right)
		{	// assign by moving _Right
		if (this != &_Right)
			{	// clean up and copy
			this->_Tidy();
			this->_Resetm(wistd::forward<_Myt>(_Right));
			}
		return (*this);
		}

	template<class _Fx>
		_Myt& operator=(_Fx&& _Func)
		{	// move function object _Func
		this->_Tidy();
		this->_Reset(wistd::forward<_Fx>(_Func));
		return (*this);
		}

	function& operator=(nullptr_t)
		{	// clear function object
		this->_Tidy();
		this->_Reset();
		return (*this);
		}

	template<class _Fx>
		_Myt& operator=(reference_wrapper<_Fx> _Func) WI_NOEXCEPT
		{	// assign wrapper holding reference to _Func
		this->_Tidy();
		this->_Reset(_Func);
		return (*this);
		}

	void swap(_Myt& _Right) WI_NOEXCEPT
		{	// swap with _Right
		this->_Swap(_Right);
		}

    explicit operator bool() const WI_NOEXCEPT
    {	// test if wrapper holds null function pointer
        return (!this->_Empty());
    }

private:
    template<class _Fty2>
        void operator==(const function<_Fty2>&);	//	not defined
    template<class _Fty2>
        void operator!=(const function<_Fty2>&);	//	not defined
    };

    // TEMPLATE FUNCTION swap
template<class _Fty>
    void swap(function<_Fty>& _Left, function<_Fty>& _Right)
    {	// swap contents of _Left with contents of _Right
    _Left.swap(_Right);
    }

    // TEMPLATE NULL POINTER COMPARISONS
template<class _Fty>
    bool operator==(const function<_Fty>& _Other,
        nullptr_t) WI_NOEXCEPT
    {	// compare to null pointer
    return (!_Other);
    }

template<class _Fty>
    bool operator==(nullptr_t _Npc,
        const function<_Fty>& _Other) WI_NOEXCEPT
    {	// compare to null pointer
    return (operator==(_Other, _Npc));
    }

template<class _Fty>
    bool operator!=(const function<_Fty>& _Other,
        nullptr_t _Npc) WI_NOEXCEPT
    {	// compare to null pointer
    return (!operator==(_Other, _Npc));
    }

template<class _Fty>
    bool operator!=(nullptr_t _Npc,
        const function<_Fty>& _Other) WI_NOEXCEPT
    {	// compare to null pointer
    return (!operator==(_Other, _Npc));
    }

} // wistd
/// @endcond

#pragma warning(pop)
