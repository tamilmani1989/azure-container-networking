// Windows Internal Libraries (wil)
// Note: do not include this file directly, include "wil\Common.h"
//
// wil Usage Guidelines:
// https://microsoft.sharepoint.com/teams/osg_development/Shared%20Documents/Windows%20Internal%20Libraries%20for%20C++%20Usage%20Guide.docx?web=1
//
// wil Discussion Alias (wildisc):
// http://idwebelements/GroupManagement.aspx?Group=wildisc&Operation=join  (one-click join)
//
//! @file
//! Selective type traits from STL's <type_traits> header under the `wistd` namespace.


#pragma once

// DO NOT add *any* includes to this file -- there should be no dependencies from its usage

// This macro declares the intent to be 'noexcept'.  Code written with this macro should be written
// with the expectation that it will be switched to 'noexcept' once widely available.  The implication
// here is that the compiler will enforce termination if an exception is actually thrown from within
// the tagged functions AND that the compiler will presume no exceptions are thrown from the function
// when optimizing call sites.  For now, 'throw()' only gives the optimization.  'noexcept' will add the
// termination.

#define WI_NOEXCEPT throw()

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
// Additional templates or functions should be added on an as-needed basis.
//
// New, exception-based code should not use this namespace, but instead should prefer the std:: implementation.
// Only code that is not exception-based and libraries that expect to be utilized across both exception
// and non-exception based code should utilize this functionality.

/// @cond
namespace wistd     // ("Windows Internal" std)
{
#define __WI_COMMA	,	/* for commas in macro parameters */
  
	// TEMPLATE CLASS unary_function
template<class _T1,
	class _Ret>
	struct unary_function;


	// TEMPLATE CLASS binary_function
template<class _T1,
	class _T2,
	class _Ret>
	struct binary_function;


    // TEMPLATE CLASS integral_constant
template<class _Ty,
    _Ty _Val>
    struct integral_constant
    { // convenient template for integral constant types
    static const _Ty value = _Val;

    typedef _Ty value_type;
    typedef integral_constant<_Ty, _Val> type;

    operator value_type() const
        { // return stored value
        return (value);
        }
    };

typedef integral_constant<bool, true> true_type;
typedef integral_constant<bool, false> false_type;


    // TEMPLATE CLASS _Cat_base
template<bool>
    struct _Cat_base
        : false_type
    {	// base class for type predicates
    };

template<>
    struct _Cat_base<true>
        : true_type
    {	// base class for type predicates
    };

  #define __WI_IS_BASE_OF(_Base, _Der)	\
    : _Cat_base<__is_base_of(_Base, _Der)>
  #define __WI_IS_CONVERTIBLE(_From, _To)	\
    : _Cat_base<__is_convertible_to(_From, _To)>
  #define __WI_IS_ABSTRACT(_Ty)	\
    : _Cat_base<__is_abstract(_Ty)>
  #define __WI_IS_EMPTY(_Ty)	\
    : _Cat_base<__is_empty(_Ty)>
  #define  __WI_IS_CLASS(_Ty)   \
    : _Cat_base<__is_class(_Ty)>
  #define __WI_HAS_TRIVIAL_DESTRUCTOR(_Ty)  \
    : _Cat_base<__has_trivial_destructor(_Ty)>


    typedef decltype(__nullptr) nullptr_t;


    // TEMPLATE remove_reference
template<class _Ty>
    struct remove_reference
    { // remove reference
    typedef _Ty type;
    };

template<class _Ty>
    struct remove_reference<_Ty&>
    { // remove reference
    typedef _Ty type;
    };

template<class _Ty>
    struct remove_reference<_Ty&&>
    { // remove rvalue reference
    typedef _Ty type;
    };


    // TEMPLATE CLASS remove_pointer
template<class _Ty>
	struct remove_pointer
	{	// remove pointer
	typedef _Ty type;
	};

template<class _Ty>
	struct remove_pointer<_Ty *>
	{	// remove pointer
	typedef _Ty type;
	};

template<class _Ty>
	struct remove_pointer<_Ty *const>
	{	// remove pointer
	typedef _Ty type;
	};

template<class _Ty>
	struct remove_pointer<_Ty *volatile>
	{	// remove pointer
	typedef _Ty type;
	};

template<class _Ty>
	struct remove_pointer<_Ty *const volatile>
	{	// remove pointer
	typedef _Ty type;
	};


    // TEMPLATE FUNCTION move
template<class _Ty> inline
    typename remove_reference<_Ty>::type&&
        move(_Ty&& _Arg) WI_NOEXCEPT
    { // forward _Arg as movable
    return ((typename remove_reference<_Ty>::type&&)_Arg);
    }


    // TEMPLATE FUNCTION forward
template<class _Ty> inline
    _Ty&& forward(typename remove_reference<_Ty>::type& _Arg) WI_NOEXCEPT
    { // forward an lvalue
    return (static_cast<_Ty&&>(_Arg));
    }

template<class _Ty> inline
    _Ty&& forward(typename remove_reference<_Ty>::type&& _Arg) WI_NOEXCEPT
    { // forward anything
    static_assert(!is_lvalue_reference<_Ty>::value, "bad forward call");
    return (static_cast<_Ty&&>(_Arg));
    }

        // TEMPLATE CLASS add_rvalue_reference
        template<class _Ty>
    struct add_rvalue_reference
    {	// add rvalue reference
        typedef typename remove_reference<_Ty>::type&& type;
    };

    template<class _Ty>
    struct add_rvalue_reference<_Ty&>
    {	// add rvalue reference to reference
        typedef _Ty& type;
    };

    template<>
    struct add_rvalue_reference<void>
    {	// add reference
        typedef void type;
    };

    template<>
    struct add_rvalue_reference<const void>
    {	// add reference
        typedef const void type;
    };

    template<>
    struct add_rvalue_reference<volatile void>
    {	// add reference
        typedef volatile void type;
    };

    template<>
    struct add_rvalue_reference<const volatile void>
    {	// add reference
        typedef const volatile void type;
    };

    // TEMPLATE FUNCTION declval
    template<class _Ty>
    typename add_rvalue_reference<_Ty>::type
        declval(int = 0) WI_NOEXCEPT;

    // TEMPLATE CLASS is_array
template<class _Ty>
    struct is_array
        : false_type
    {	// determine whether _Ty is an array
    };

template<class _Ty, size_t _Nx>
    struct is_array<_Ty[_Nx]>
        : true_type
    {	// determine whether _Ty is an array
    };

template<class _Ty>
    struct is_array<_Ty[]>
        : true_type
    {	// determine whether _Ty is an array
    };


    // TEMPLATE CLASS is_lvalue_reference
template<class _Ty>
    struct is_lvalue_reference
        : false_type
    { // determine whether _Ty is an lvalue reference
    };

template<class _Ty>
    struct is_lvalue_reference<_Ty&>
        : true_type
    { // determine whether _Ty is an lvalue reference
    };


    // TEMPLATE CLASS is_rvalue_reference
template<class _Ty>
    struct is_rvalue_reference
        : false_type
    { // determine whether _Ty is an rvalue reference
    };

template<class _Ty>
    struct is_rvalue_reference<_Ty&&>
        : true_type
    { // determine whether _Ty is an rvalue reference
    };


    // TEMPLATE CLASS is_reference
template<class _Ty>
    struct is_reference
        : _Cat_base<is_lvalue_reference<_Ty>::value
        || is_rvalue_reference<_Ty>::value>
    {	// determine whether _Ty is a reference
    };


    // TEMPLATE CLASS is_same
template<class _Ty1, class _Ty2>
    struct is_same
        : false_type
    {	// determine whether _Ty1 and _Ty2 are the same type
    };

template<class _Ty1>
    struct is_same<_Ty1, _Ty1>
        : true_type
    {	// determine whether _Ty1 and _Ty2 are the same type
    };


    // TEMPLATE CLASS extent
template<class _Ty, unsigned int _Nx>
    struct _Extent
        : integral_constant<size_t, 0>
    {	// determine extent of dimension _Nx of array _Ty
    };

template<class _Ty, unsigned int _Ix>
    struct _Extent<_Ty[_Ix], 0>
        : integral_constant<size_t, _Ix>
    {	// determine extent of dimension _Nx of array _Ty
    };

template<class _Ty, unsigned int _Nx, unsigned int _Ix>
    struct _Extent<_Ty[_Ix], _Nx>
        : _Extent<_Ty, _Nx - 1>
    {	// determine extent of dimension _Nx of array _Ty
    };

template<class _Ty, unsigned int _Nx>
    struct _Extent<_Ty[], _Nx>
        : _Extent<_Ty, _Nx - 1>
    {	// determine extent of dimension _Nx of array _Ty
    };

template<class _Ty, unsigned int _Nx = 0>
    struct extent
        : _Extent<_Ty, _Nx>
    {	// determine extent of dimension _Nx of array _Ty
    };


    // TEMPLATE CLASS is_base_of
template<class _Base, class _Der>
    struct is_base_of __WI_IS_BASE_OF(_Base, _Der)
    {	// determine whether _Base is a base of or the same as _Der
    };


    // TEMPLATE CLASS is_abstract
template<class _Ty>
    struct is_abstract __WI_IS_ABSTRACT(_Ty)
    {	// determine whether _Ty is an abstract class
    };

    
    // TEMPLATE CLASS is_class
template<class _Ty>
    struct is_class __WI_IS_CLASS(_Ty)
    {	// determine whether _Ty is a class
    };

    
    // TEMPLATE CLASS is_empty
template<class _Ty>
    struct is_empty __WI_IS_EMPTY(_Ty)
    {	// determine whether _Ty is an empty class
    };


    // TEMPLATE CLASS is_convertible

template<class _From, class _To>
    struct is_convertible
        __WI_IS_CONVERTIBLE(_From, _To)
    {	// determine whether _From is convertible to _To
    };

template<class _From, class _To>
using is_convertible_t = typename is_convertible<_From, _To>::type;



    // TEMPLATE CLASS remove_const
template<class _Ty>
    struct remove_const
    {	// remove top level const qualifier
    typedef _Ty type;
    };

template<class _Ty>
    struct remove_const<const _Ty>
    {	// remove top level const qualifier
    typedef _Ty type;
    };

template<class _Ty>
    struct remove_const<const _Ty[]>
    {	// remove top level const qualifier
    typedef _Ty type[];
    };

template<class _Ty, unsigned int _Nx>
    struct remove_const<const _Ty[_Nx]>
    {	// remove top level const qualifier
    typedef _Ty type[_Nx];
    };


    // TEMPLATE CLASS remove_volatile
template<class _Ty>
    struct remove_volatile
    {	// remove top level volatile qualifier
    typedef _Ty type;
    };

template<class _Ty>
    struct remove_volatile<volatile _Ty>
    {	// remove top level volatile qualifier
    typedef _Ty type;
    };

template<class _Ty>
    struct remove_volatile<volatile _Ty[]>
    {	// remove top level volatile qualifier
    typedef _Ty type[];
    };

template<class _Ty, unsigned int _Nx>
    struct remove_volatile<volatile _Ty[_Nx]>
    {	// remove top level volatile qualifier
    typedef _Ty type[_Nx];
    };


    // TEMPLATE CLASS remove_cv
template<class _Ty>
    struct remove_cv
    {	// remove top level const and volatile qualifiers
    typedef typename remove_const<typename remove_volatile<_Ty>::type>::type
        type;
    };


    // TEMPLATE CLASS enable_if
template<bool _Test,
    class _Ty = void>
    struct enable_if
    {	// type is undefined for assumed !_Test
    };

template<class _Ty>
    struct enable_if<true, _Ty>
    {	// type is _Ty for _Test
    typedef _Ty type;
    };

template<bool _Test,
    class _Ty = void>
    using enable_if_t = typename enable_if<_Test, _Ty>::type;


    // TEMPLATE FUNCTION addressof
template<class _Ty> inline
    _Ty *addressof(_Ty& _Val) WI_NOEXCEPT
    {	// return address of _Val
    return (reinterpret_cast<_Ty *>(
        (&const_cast<char&>(
        reinterpret_cast<const volatile char&>(_Val)))));
    }

    // TEMPLATE CLASS is_trivially_destructible
template<class _Ty>
    struct is_trivially_destructible
        __WI_HAS_TRIVIAL_DESTRUCTOR(_Ty)
    {   // determine whether _Ty has a trivial destructor
    };

    // TEMPLATE CLASS has_trivial_destructor -- retained
template<class _Ty>
    struct has_trivial_destructor

        : is_trivially_destructible<_Ty>::type

    {   // determine whether _Ty has a trivial destructor
    };

    // TYPE TESTING MACROS
struct _Wrap_int
    {	// wraps int so that int argument is favored over _Wrap_int
    _Wrap_int(int)
        {	// do nothing
        }
    };

template<class _Ty>
    struct _Identity
    {	// map _Ty to type unchanged, without operator()
    typedef _Ty type;
    };

#define __WI_GET_TYPE_OR_DEFAULT(TYPE, DEFAULT) \
    { \
    template<class _Uty> \
        static auto _Fn(int) \
            -> _Identity<typename _Uty::TYPE>; \
 \
    template<class _Uty> \
        static auto _Fn(_Wrap_int) \
            -> _Identity<DEFAULT>; \
 \
    typedef decltype(_Fn<_Ty>(0)) _Decltype; \
    typedef typename _Decltype::type type; \
    }

#define __WI_HAS_TYPES(TYPE1, TYPE2, TYPE3) \
    { \
    template<class _Uty> \
        static auto _Fn(int, \
            _Identity<typename _Uty::TYPE1> * = 0, \
            _Identity<typename _Uty::TYPE2> * = 0, \
            _Identity<typename _Uty::TYPE3> * = 0) \
            -> true_type; \
 \
    template<class _Uty> \
        static auto _Fn(_Wrap_int) \
            -> false_type; \
 \
    typedef decltype(_Fn<_Ty>(0)) type; \
    }

#define __WI_HAS_ONE_TYPE(TYPE) \
    __WI_HAS_TYPES(TYPE, TYPE, TYPE)

        // TEMPLATE STRUCT _Has_result_type
template<class _Ty>
    struct _Has_result_type
        __WI_HAS_ONE_TYPE(result_type);


 #ifdef _M_IX86

  #ifdef _M_CEE
#define __WI_NON_MEMBER_CALL(FUNC, CONST_OPT) \
    FUNC(__cdecl, CONST_OPT) \
    FUNC(__stdcall, CONST_OPT) \
    FUNC(__clrcall, CONST_OPT)
#define __WI_MEMBER_CALL(FUNC, CONST_OPT, CV_OPT) \
    FUNC(__thiscall, CONST_OPT, CV_OPT) \
    FUNC(__cdecl, CONST_OPT, CV_OPT) \
    FUNC(__stdcall, CONST_OPT, CV_OPT) \
    FUNC(__clrcall, CONST_OPT, CV_OPT)

  #else /* _M_CEE */
#define __WI_NON_MEMBER_CALL(FUNC, CONST_OPT) \
    FUNC(__cdecl, CONST_OPT) \
    FUNC(__stdcall, CONST_OPT) \
    FUNC(__fastcall, CONST_OPT)
#define __WI_MEMBER_CALL(FUNC, CONST_OPT, CV_OPT) \
    FUNC(__thiscall, CONST_OPT, CV_OPT) \
    FUNC(__cdecl, CONST_OPT, CV_OPT) \
    FUNC(__stdcall, CONST_OPT, CV_OPT) \
    FUNC(__fastcall, CONST_OPT, CV_OPT)
  #endif /* _M_CEE */

 #else /* _M_IX86 */

  #ifdef _M_CEE
#define __WI_NON_MEMBER_CALL(FUNC, CONST_OPT) \
    FUNC(__cdecl, CONST_OPT) \
    FUNC(__clrcall, CONST_OPT)
#define __WI_MEMBER_CALL(FUNC, CONST_OPT, CV_OPT) \
    FUNC(__cdecl, CONST_OPT, CV_OPT) \
    FUNC(__clrcall, CONST_OPT, CV_OPT)

  #else /* _M_CEE */
#define __WI_NON_MEMBER_CALL(FUNC, CONST_OPT) \
    FUNC(__cdecl, CONST_OPT)
#define __WI_MEMBER_CALL(FUNC, CONST_OPT, CV_OPT) \
    FUNC(__cdecl, CONST_OPT, CV_OPT)
  #endif /* _M_CEE */
 #endif /* _M_IX86 */

#define __WI_NON_MEMBER_CALL_CONST(FUNC) \
    __WI_NON_MEMBER_CALL(FUNC, ) \
    __WI_NON_MEMBER_CALL(FUNC, const)

#define __WI_MEMBER_CALL_CV(FUNC, CONST_OPT) \
    __WI_MEMBER_CALL(FUNC, CONST_OPT, ) \
    __WI_MEMBER_CALL(FUNC, CONST_OPT, const) \
    __WI_MEMBER_CALL(FUNC, CONST_OPT, volatile) \
    __WI_MEMBER_CALL(FUNC, CONST_OPT, const volatile)

#define __WI_MEMBER_CALL_CONST_CV(FUNC) \
    __WI_MEMBER_CALL_CV(FUNC, ) \
    __WI_MEMBER_CALL_CV(FUNC, const)

#define __WI_EMPTY_ARGUMENT		/* for empty macro argument */

#define __WI_CLASS_DEFINE_CV(CLASS) \
    CLASS(__WI_EMPTY_ARGUMENT) \
    CLASS(const) \
    CLASS(volatile) \
    CLASS(const volatile)


        // TYPE DEFINITIONS

template<bool,
    class _Ty1,
    class _Ty2>
    struct _If
    {	// type is _Ty2 for assumed false
    typedef _Ty2 type;
    };

template<class _Ty1,
    class _Ty2>
    struct _If<true, _Ty1, _Ty2>
    {	// type is _Ty1 for assumed true
    typedef _Ty1 type;
    };

template<class _Ty>
    struct _Always_false
    {	// false value that probably won't be optimized away
    static const bool value = false;
    };


	// TEMPLATE CLASS remove_extent
template<class _Ty>
	struct remove_extent
	{	// remove array extent
	typedef _Ty type;
	};

template<class _Ty, unsigned int _Ix>
	struct remove_extent<_Ty[_Ix]>
	{	// remove array extent
	typedef _Ty type;
	};

template<class _Ty>
	struct remove_extent<_Ty[]>
	{	// remove array extent
	typedef _Ty type;
	};


    // TEMPLATE CLASS add_reference -- retained
template<class _Ty>
    struct add_reference
    {	// add reference
    typedef _Ty& type;
    };

#define __WI_ADD_REFERENCE_VOID(CV_OPT) \
template<> \
    struct add_reference<CV_OPT void> \
    {	/* add reference */ \
    typedef CV_OPT void type; \
    };

__WI_CLASS_DEFINE_CV(__WI_ADD_REFERENCE_VOID)
#undef __WI_ADD_REFERENCE_VOID


template<class _Ty>
	struct _Is_funptr
		: false_type
	{	// base class for function pointer predicates
	};

template<class _Ty>
	struct _Is_memfunptr
		: false_type
	{	// base class for member function pointer predicates
	};

#define __WIL_CLASS_IS_FUNPTR( \
	TEMPLATE_LIST, PADDING_LIST, LIST, COMMA, X1, X2, X3, X4) \
template<class _Ret COMMA LIST(_CLASS_TYPE)> \
	struct _Is_funptr<_Ret (*)(LIST(_TYPE))> \
		: true_type \
	{	/* base class for function pointer predicates */ \
	}; \
template<class _Ret, \
	class _Arg0 COMMA LIST(_CLASS_TYPE)> \
	struct _Is_memfunptr<_Ret (_Arg0::*)(LIST(_TYPE))> \
		: true_type \
	{	/* base class for member function pointer predicates */ \
	}; \
template<class _Ret, \
	class _Arg0 COMMA LIST(_CLASS_TYPE)> \
	struct _Is_memfunptr<_Ret (_Arg0::*)(LIST(_TYPE)) const> \
		: true_type \
	{	/* base class for member function pointer predicates */ \
	}; \
template<class _Ret, \
	class _Arg0 COMMA LIST(_CLASS_TYPE)> \
	struct _Is_memfunptr<_Ret (_Arg0::*)(LIST(_TYPE)) volatile> \
		: true_type \
	{	/* base class for member function pointer predicates */ \
	}; \
template<class _Ret, \
	class _Arg0 COMMA LIST(_CLASS_TYPE)> \
	struct _Is_memfunptr<_Ret (_Arg0::*)(LIST(_TYPE)) const volatile> \
		: true_type \
	{	/* base class for member function pointer predicates */ \
	}; \
template<class _Ret COMMA LIST(_CLASS_TYPE)> \
	struct _Is_funptr<_Ret (*)(LIST(_TYPE)...)> \
		: true_type \
	{	/* base class for function pointer predicates */ \
	}; \
template<class _Ret, \
	class _Arg0 COMMA LIST(_CLASS_TYPE)> \
	struct _Is_memfunptr<_Ret (_Arg0::*)(LIST(_TYPE)...)> \
		: true_type \
	{	/* base class for member function pointer predicates */ \
	}; \
template<class _Ret, \
	class _Arg0 COMMA LIST(_CLASS_TYPE)> \
	struct _Is_memfunptr<_Ret (_Arg0::*)(LIST(_TYPE)...) const> \
		: true_type \
	{	/* base class for member function pointer predicates */ \
	}; \
template<class _Ret, \
	class _Arg0 COMMA LIST(_CLASS_TYPE)> \
	struct _Is_memfunptr<_Ret (_Arg0::*)(LIST(_TYPE)...) volatile> \
		: true_type \
	{	/* base class for member function pointer predicates */ \
	}; \
template<class _Ret, \
	class _Arg0 COMMA LIST(_CLASS_TYPE)> \
	struct _Is_memfunptr<_Ret (_Arg0::*)(LIST(_TYPE)...) const volatile> \
		: true_type \
	{	/* base class for member function pointer predicates */ \
	};


    // TEMPLATE CLASS is_member_object_pointer
template<class _Ty>
	struct _Is_member_object_pointer
		: false_type
	{	// determine whether _Ty is a pointer to member object
	};

template<class _Ty1, class _Ty2>
	struct _Is_member_object_pointer<_Ty1 _Ty2::*>
		: _Cat_base<!_Is_memfunptr<_Ty1 _Ty2::*>::value>
	{	// determine whether _Ty is a pointer to member object
	};

template<class _Ty>
	struct is_member_object_pointer
		: _Is_member_object_pointer<typename remove_cv<_Ty>::type>
	{	// determine whether _Ty is a pointer to member object
	};


	// TEMPLATE CLASS is_member_function_pointer
template<class _Ty>
	struct is_member_function_pointer
		: _Cat_base<_Is_memfunptr<typename remove_cv<_Ty>::type>::value>
	{	// determine whether _Ty is a pointer to member function
	};


	// TEMPLATE CLASS is_pointer
template<class _Ty>
	struct _Is_pointer
		: false_type
	{	// determine whether _Ty is a pointer
	};

template<class _Ty>
	struct _Is_pointer<_Ty *>
		: _Cat_base<!is_member_object_pointer<_Ty *>::value
		&& !is_member_function_pointer<_Ty *>::value>
	{	// determine whether _Ty is a pointer
	};

template<class _Ty>
	struct is_pointer
		: _Is_pointer<typename remove_cv<_Ty>::type>
	{	// determine whether _Ty is a pointer
	};


        //	FUNCTIONAL STUFF (from <functional>)
		// TEMPLATE STRUCT unary_function
template<class _Arg,
	class _Result>
	struct unary_function
	{	// base class for unary functions
	typedef _Arg argument_type;
	typedef _Result result_type;
	};

		// TEMPLATE STRUCT binary_function
template<class _Arg1,
	class _Arg2,
	class _Result>
	struct binary_function
	{	// base class for binary functions
	typedef _Arg1 first_argument_type;
	typedef _Arg2 second_argument_type;
	typedef _Result result_type;
	};


	// TEMPLATE CLASS is_function
template<class _Ty>
	struct is_function
		: _Cat_base<_Is_funptr<typename remove_cv<_Ty>::type *>::value>
	{	// determine whether _Ty is a function
	};

template<class _Ty>
	struct is_function<_Ty&>
		: false_type
	{	// determine whether _Ty is a function
	};

template<class _Ty>
	struct is_function<_Ty&&>
		: false_type
	{	// determine whether _Ty is a function
	};

	// TEMPLATE CLASS _Is_integral
template<class _Ty>
	struct _Is_integral
		: false_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<bool>
		: true_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<char>
		: true_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<unsigned char>
		: true_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<signed char>
		: true_type
	{	// determine whether _Ty is integral
	};

 #ifdef _NATIVE_WCHAR_T_DEFINED
template<>
	struct _Is_integral<wchar_t>
		: true_type
	{	// determine whether _Ty is integral
	};
 #endif /* _NATIVE_WCHAR_T_DEFINED */

template<>
	struct _Is_integral<unsigned short>
		: true_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<signed short>
		: true_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<unsigned int>
		: true_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<signed int>
		: true_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<unsigned long>
		: true_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<signed long>
		: true_type
	{	// determine whether _Ty is integral
	};

 #if _HAS_CHAR16_T_LANGUAGE_SUPPORT
template<>
	struct _Is_integral<char16_t>
		: true_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<char32_t>
		: true_type
	{	// determine whether _Ty is integral
	};
 #endif /* _HAS_CHAR16_T_LANGUAGE_SUPPORT */

 #ifdef _LONGLONG
template<>
	struct _Is_integral<_LONGLONG>
		: true_type
	{	// determine whether _Ty is integral
	};

template<>
	struct _Is_integral<_ULONGLONG>
		: true_type
	{	// determine whether _Ty is integral
	};
 #endif /* _LONGLONG */

	// TEMPLATE CLASS is_integral
template<class _Ty>
	struct is_integral
		: _Is_integral<typename remove_cv<_Ty>::type>
	{	// determine whether _Ty is integral
	};
	// TEMPLATE CLASS _Is_floating_point
template<class _Ty>
	struct _Is_floating_point
		: false_type
	{	// determine whether _Ty is floating point
	};

template<>
	struct _Is_floating_point<float>
		: true_type
	{	// determine whether _Ty is floating point
	};

template<>
	struct _Is_floating_point<double>
		: true_type
	{	// determine whether _Ty is floating point
	};

template<>
	struct _Is_floating_point<long double>
		: true_type
	{	// determine whether _Ty is floating point
	};

	// TEMPLATE CLASS is_floating_point
template<class _Ty>
	struct is_floating_point
		: _Is_floating_point<typename remove_cv<_Ty>::type>
	{	// determine whether _Ty is floating point
	};	// TEMPLATE CLASS add_pointer

    // TEMPLATE CLASS is_arithmetic
template<class _Ty>
	struct is_arithmetic
		: _Cat_base<is_integral<_Ty>::value
		|| is_floating_point<_Ty>::value>
	{	// determine whether _Ty is an arithmetic type
	};

template<class _Ty>
	struct add_pointer
	{	// add pointer
	typedef typename remove_reference<_Ty>::type *type;
	};


    // TEMPLATE CLASS decay
template<class _Ty>
	struct decay
	{	// determines decayed version of _Ty
	typedef typename remove_reference<_Ty>::type _Ty1;

	typedef typename _If<is_array<_Ty1>::value,
		typename remove_extent<_Ty1>::type *,
		typename _If<is_function<_Ty1>::value,
			typename add_pointer<_Ty1>::type,
			typename remove_cv<_Ty1>::type>::type>::type type;
	};

#define __WIL_NOEXCEPT_OP(x)

    // TEMPLATE CLASS is_void
template<class _Ty>
	struct _Is_void
		: false_type
	{	// determine whether _Ty is void
	};

template<>
	struct _Is_void<void>
		: true_type
	{	// determine whether _Ty is void
	};

template<class _Ty>
	struct is_void
		: _Is_void<typename remove_cv<_Ty>::type>
	{	// determine whether _Ty is void
	};

		// TEMPLATE CLASS pointer_traits
template<class _Ty>
	struct pointer_traits;

template<class _Ty>
	struct _Get_first_parameter
	{	// get _Ty::element_type
	typedef typename _Ty::element_type type;
	};

		// TEMPLATE STRUCT _Replace_first_parameter
template<class _Newfirst,
	class _Ty>
	struct _Replace_first_parameter
	{	// get _Ty::element_type
	typedef typename _Ty::template rebind<_Newfirst>::other type;
	};

		// TEMPLATE STRUCT _Get_element_type
template<class _Ty>
	struct _Get_element_type
	__WI_GET_TYPE_OR_DEFAULT(element_type,
		typename _Get_first_parameter<_Ty>::type);

		// TEMPLATE STRUCT _Get_ptr_difference_type
template<class _Ty>
	struct _Get_ptr_difference_type
	__WI_GET_TYPE_OR_DEFAULT(difference_type,
		ptrdiff_t);

		// TEMPLATE STRUCT _Get_rebind_type
template<class _Ty,
	class _Other>
	struct _Get_rebind_type
	__WI_GET_TYPE_OR_DEFAULT(template rebind<_Other>::other,
		typename _Replace_first_parameter<_Other __WI_COMMA _Uty>::type);

		// TEMPLATE CLASS pointer_traits
template<class _Ty>
	struct pointer_traits
	{	// defines traits for arbitrary pointers
	typedef pointer_traits<_Ty> other;

	typedef typename _Get_element_type<_Ty>::type element_type;
	typedef _Ty pointer;
	typedef typename _Get_ptr_difference_type<_Ty>::type difference_type;

	template<class _Other>
		struct rebind
		{	// converts X<element_type> to X<_Other>
		typedef typename _Get_rebind_type<_Ty, _Other>::type other;
		};

	static pointer pointer_to(element_type& _Val)
		{	// convert raw reference to pointer
		return (_Ty::pointer_to(_Val));
		}
	};

		// TEMPLATE CLASS pointer_traits<_Ty *>
template<class _Ty>
	struct pointer_traits<_Ty *>
	{	// defines traits for raw pointers
	typedef pointer_traits<_Ty *> other;

	typedef _Ty element_type;
	typedef _Ty *pointer;
	typedef ptrdiff_t difference_type;

	template<class _Other>
		struct rebind
		{	// converts to a pointer to _Other
		typedef _Other *other;
		};

	typedef typename _If<is_void<_Ty>::value,
		char&,
		typename add_reference<_Ty>::type>::type _Reftype;

	static pointer pointer_to(_Reftype _Val)
		{	// convert raw reference to pointer
		return (wistd::addressof(_Val));
		}
	};

		// TEMPLATE FUNCTION iter_swap_wil (from <xutility>)
template<class _Ty> inline
	void swap_wil(_Ty&, _Ty&)
		__WIL_NOEXCEPT_OP(is_nothrow_move_constructible<_Ty>::value
			&& is_nothrow_move_assignable<_Ty>::value);

template<class _FwdIt1,
	class _FwdIt2> inline
	void iter_swap_wil(_FwdIt1 _Left, _FwdIt2 _Right)
	{	// swap *_Left and *_Right
	swap_wil(*_Left, *_Right);
	}

		// TEMPLATE FUNCTION _Move_wil
template<class _Ty> inline
	typename remove_reference<_Ty>::type&&
		_Move_wil(_Ty&& _Arg) WI_NOEXCEPT
	{	// forward _Arg as movable
	return ((typename remove_reference<_Ty>::type&&)_Arg);
	}

		// TEMPLATE FUNCTION swap_wil
template<class _Ty,
	size_t _Size> inline
	void swap_wil(_Ty (&_Left)[_Size], _Ty (&_Right)[_Size])
		__WIL_NOEXCEPT_OP(__WIL_NOEXCEPT_OP(swap_wil(*_Left, *_Right)))
	{	// exchange arrays stored at _Left and _Right
	if (&_Left != &_Right)
		{	// worth swapping, swap_wil ranges
		_Ty *_First1 = _Left;
		_Ty *_Last1 = _First1 + _Size;
		_Ty *_First2 = _Right;
		for (; _First1 != _Last1; ++_First1, ++_First2)
			wistd::iter_swap_wil(_First1, _First2);
		}
	}

template<class _Ty> inline
	void swap_wil(_Ty& _Left, _Ty& _Right)
		__WIL_NOEXCEPT_OP(is_nothrow_move_constructible<_Ty>::value
			&& is_nothrow_move_assignable<_Ty>::value)
	{	// exchange values stored at _Left and _Right
	_Ty _Tmp = _Move_wil(_Left);
	_Left = _Move_wil(_Right);
	_Right = _Move_wil(_Tmp);
	}

		// TEMPLATE FUNCTION _Swap_adl_wil
template<class _Ty> inline
	void _Swap_adl_wil(_Ty& _Left, _Ty& _Right)
	{	// exchange values stored at _Left and _Right, using ADL
	swap_wil(_Left, _Right);
	}


	// TEMPLATE CLASS _Ptr_traits
template<class _Ty>
	struct _Ptr_traits
	{	// basic definition
	};

template<class _Ty>
	struct _Ptr_traits<_Ty *>
	{	// pointer properties
	static const bool _Is_const = false;
	static const bool _Is_volatile = false;
	};

template<class _Ty>
	struct _Ptr_traits<const _Ty *>
	{	// pointer to const properties
	static const bool _Is_const = true;
	static const bool _Is_volatile = false;
	};

template<class _Ty>
	struct _Ptr_traits<volatile _Ty *>
	{	// pointer to volatile properties
	static const bool _Is_const = false;
	static const bool _Is_volatile = true;
	};

template<class _Ty>
	struct _Ptr_traits<const volatile _Ty *>
	{	// pointer to const volatile properties
	static const bool _Is_const = true;
	static const bool _Is_volatile = true;
	};


// TEMPLATE CLASS is_const
template<class _Ty>
	struct is_const
		: _Cat_base<_Ptr_traits<_Ty *>::_Is_const
		&& !is_function<_Ty>::value>
	{	// determine whether _Ty is const qualified
	};

template<class _Ty, unsigned int _Nx>
	struct is_const<_Ty[_Nx]>
		: false_type
	{	// determine whether _Ty is const qualified
	};

template<class _Ty, unsigned int _Nx>
	struct is_const<const _Ty[_Nx]>
		: true_type
	{	// determine whether _Ty is const qualified
	};

template<class _Ty>
	struct is_const<_Ty&>
		: false_type
	{	// determine whether _Ty is const qualified
	};

template<class _Ty>
	struct is_const<_Ty&&>
		: false_type
	{	// determine whether _Ty is const qualified
	};

#define _IS_CONSTRUCTIBLE \
	__is_constructible
    // TEMPLATE CLASS is_constructible

	// TEMPLATE CLASS add_lvalue_reference
template<class _Ty>
	struct add_lvalue_reference
	{	// add lvalue reference
	typedef _Ty& type;
	};

	// Type modifiers
	// TEMPLATE CLASS add_const
template<class _Ty>
	struct add_const
	{	// add top level const qualifier
	typedef const _Ty type;
	};

template<class _Ty,
	class... _Args>
	struct is_constructible
		: _Cat_base<_IS_CONSTRUCTIBLE(_Ty, _Args...)>
	{	// determine whether _Ty(_Args...) is constructible
	};

	// TEMPLATE CLASS is_copy_constructible
template<class _Ty>
	struct is_copy_constructible
		: is_constructible<
			_Ty,
			typename add_lvalue_reference<
				typename add_const<_Ty>::type
			>::type
		>::type
	{	// determine whether _Ty has a copy constructor
	};

} // wistd
/// @endcond
