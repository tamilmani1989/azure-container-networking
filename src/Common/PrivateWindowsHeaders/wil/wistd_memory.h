// Windows Internal Libraries (wil)
// Note: do not include this file directly, include "wil\Resource.h"
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
// from exception-free components want these concepts, but including STL headers directly introduces
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

/// @cond
namespace wistd     // ("Windows Internal" std)
{
    // Available Types and Routines:
    //
    // wistd::unique_ptr<T>     - used to hold an allocated object (traditional smart pointer)

    // TEMPLATE CLASS default_delete
template<class _Ty>
    struct default_delete
    {	// default deleter for unique_ptr
    typedef default_delete<_Ty> _Myt;

    default_delete() WI_NOEXCEPT
        {	// default construct
        }

    template<class _Ty2>
        default_delete(const default_delete<_Ty2>&,
            typename enable_if<is_convertible<_Ty2 *, _Ty *>::value,
                void>::type ** = 0) WI_NOEXCEPT
        {	// construct from another default_delete
        }

    void operator()(_Ty *_Ptr) const WI_NOEXCEPT
        {	// delete a pointer
        static_assert(0 < sizeof (_Ty),
            "can't delete an incomplete type");
        delete _Ptr;
        }
    };

template<class _Ty>
    struct default_delete<_Ty[]>
    {	// default deleter for unique_ptr to array of unknown size
    typedef default_delete<_Ty> _Myt;

    default_delete() WI_NOEXCEPT
        {	// default construct
        }

    template<class _Other>
        void operator()(_Other *) const;    //not defined

    void operator()(_Ty *_Ptr) const WI_NOEXCEPT
        {	// delete a pointer
        static_assert(0 < sizeof (_Ty),
            "can't delete an incomplete type");
        delete[] _Ptr;
        }
    };

        // TEMPLATE STRUCT _Get_deleter_pointer_type
template<class _Val,
    class _Ty>
    struct _Get_deleter_pointer_type
    __WI_GET_TYPE_OR_DEFAULT(pointer,
        _Val *);

    // TEMPLATE CLASS _Unique_ptr_base
template<class _Ty,
    class _Dx,
    bool _Empty_deleter>
    class _Unique_ptr_base
    {	// stores pointer and deleter
public:
    typedef typename remove_reference<_Dx>::type _Dx_noref;
    typedef typename _Get_deleter_pointer_type<_Ty, _Dx_noref>::type pointer;

    _Unique_ptr_base(pointer _Ptr, _Dx _Dt)
        : _Myptr(_Ptr), _Mydel(_Dt)
        {	// construct with pointer and deleter
        }

    _Unique_ptr_base(pointer _Ptr)
        : _Myptr(_Ptr)
        {	// construct with pointer and deleter
        }

    template<class _Ptr2,
        class _Dx2>
        _Unique_ptr_base(_Ptr2 _Ptr, _Dx2 _Dt)
        : _Myptr(_Ptr), _Mydel(_Dt)
        {	// construct with compatible pointer and deleter
        }

    template<class _Ptr2>
        _Unique_ptr_base(_Ptr2 _Ptr)
        : _Myptr(_Ptr)
        {	// construct with compatible pointer and deleter
        }

    _Dx_noref& get_deleter()
        {	// return reference to deleter
        return (_Mydel);
        }

    const _Dx_noref& get_deleter() const
        {	// return const reference to deleter
        return (_Mydel);
        }

    pointer _Myptr;	// the managed pointer
    _Dx _Mydel;		// the deleter
    };

template<class _Ty,
    class _Dx>
    class _Unique_ptr_base<_Ty, _Dx, true>
        : public _Dx
    {	// store pointer and empty deleter
public:
    typedef _Dx _Mybase;
    typedef typename remove_reference<_Dx>::type _Dx_noref;
    typedef typename _Get_deleter_pointer_type<_Ty, _Dx_noref>::type pointer;

    _Unique_ptr_base(pointer _Ptr, _Dx _Dt) WI_NOEXCEPT
        : _Myptr(_Ptr), _Mybase(_Dt)
        {	// construct with pointer and deleter
        }

    _Unique_ptr_base(pointer _Ptr) WI_NOEXCEPT
        : _Myptr(_Ptr)
        {	// construct with pointer and deleter
        }

    template<class _Ptr2,
        class _Dx2>
        _Unique_ptr_base(_Ptr2 _Ptr, _Dx2 _Dt) WI_NOEXCEPT
        : _Myptr(_Ptr), _Mybase(_Dt)
        {	// construct with compatible pointer and deleter
        }

    template<class _Ptr2>
        _Unique_ptr_base(_Ptr2 _Ptr) WI_NOEXCEPT
        : _Myptr(_Ptr)
        {	// construct with compatible pointer and deleter
        }

    _Dx_noref& get_deleter() WI_NOEXCEPT
        {	// return reference to deleter
        return (*this);
        }

    const _Dx_noref& get_deleter() const WI_NOEXCEPT
        {	// return const reference to deleter
        return (*this);
        }

    pointer _Myptr;	// the managed pointer
    };

    // TEMPLATE CLASS unique_ptr SCALAR
template<class _Ty,
    class _Dx = default_delete<_Ty>>
    class unique_ptr
        : private _Unique_ptr_base<_Ty, _Dx,
            is_empty<_Dx>::value
                || is_same<default_delete<_Ty>, _Dx>::value>
    {	// non-copyable pointer to an object
public:
    typedef unique_ptr<_Ty, _Dx> _Myt;
    typedef _Unique_ptr_base<_Ty, _Dx,
        is_empty<_Dx>::value
            || is_same<default_delete<_Ty>, _Dx>::value> _Mybase;
    typedef typename _Mybase::pointer pointer;
    typedef _Ty element_type;
    typedef _Dx deleter_type;

    using _Mybase::get_deleter;

    unique_ptr() WI_NOEXCEPT
        : _Mybase(pointer())
        {	// default construct
        static_assert(!is_pointer<_Dx>::value,
            "unique_ptr constructed with null deleter pointer");
        }

    unique_ptr(nullptr_t) WI_NOEXCEPT
        : _Mybase(pointer())
        {	// null pointer construct
        static_assert(!is_pointer<_Dx>::value,
            "unique_ptr constructed with null deleter pointer");
        }

    _Myt& operator=(nullptr_t) WI_NOEXCEPT
        {	// assign a null pointer
        reset();
        return (*this);
        }

    explicit unique_ptr(pointer _Ptr) WI_NOEXCEPT
        : _Mybase(_Ptr)
        {	// construct with pointer
        static_assert(!is_pointer<_Dx>::value,
            "unique_ptr constructed with null deleter pointer");
        }

    unique_ptr(pointer _Ptr,
        typename _If<is_reference<_Dx>::value, _Dx,
            const typename remove_reference<_Dx>::type&>::type _Dt) WI_NOEXCEPT
        : _Mybase(_Ptr, _Dt)
        {	// construct with pointer and (maybe const) deleter&
        }

    unique_ptr(pointer _Ptr,
        typename remove_reference<_Dx>::type&& _Dt) WI_NOEXCEPT
        : _Mybase(_Ptr, wistd::move(_Dt))
        {	// construct by moving deleter
        static_assert(!is_reference<_Dx>::value,
            "unique_ptr constructed with reference to rvalue deleter");
        }

    unique_ptr(unique_ptr&& _Right) WI_NOEXCEPT
        : _Mybase(_Right.release(),
            wistd::forward<_Dx>(_Right.get_deleter()))
        {	// construct by moving _Right
        }

    template<class _Ty2,
        class _Dx2>
        unique_ptr(unique_ptr<_Ty2, _Dx2>&& _Right,
            typename enable_if<!is_array<_Ty2>::value
                && is_convertible<typename unique_ptr<_Ty2, _Dx2>::pointer,
                    pointer>::value
                && (is_reference<_Dx>::value && is_same<_Dx, _Dx2>::value
                    || !is_reference<_Dx>::value
                        && is_convertible<_Dx2, _Dx>::value),
                void>::type ** = 0) WI_NOEXCEPT
            : _Mybase(_Right.release(),
                wistd::forward<_Dx2>(_Right.get_deleter()))
        {	// construct by moving _Right
        }

    template<class _Ty2,
        class _Dx2>
        typename enable_if<!is_array<_Ty2>::value
            && is_convertible<typename unique_ptr<_Ty2, _Dx2>::pointer,
                pointer>::value,
            _Myt&>::type
        operator=(unique_ptr<_Ty2, _Dx2>&& _Right) WI_NOEXCEPT
        {	// assign by moving _Right
        reset(_Right.release());
        this->get_deleter() = wistd::forward<_Dx2>(_Right.get_deleter());
        return (*this);
        }

    _Myt& operator=(_Myt&& _Right) WI_NOEXCEPT
        {	// assign by moving _Right
        if (this != &_Right)
            {	// different, do the move
            reset(_Right.release());
            this->get_deleter() = wistd::forward<_Dx>(_Right.get_deleter());
            }
        return (*this);
        }

    void swap(_Myt& _Right) WI_NOEXCEPT
        {	// swap elements
        _Swap_adl_wil(this->_Myptr, _Right._Myptr);
        _Swap_adl_wil(this->get_deleter(),
            _Right.get_deleter());
        }

    ~unique_ptr() WI_NOEXCEPT
        {	// destroy the object
        _Delete();
        }

    typename add_reference<_Ty>::type operator*() const
        {	// return reference to object
        return (*this->_Myptr);
        }

    pointer operator->() const WI_NOEXCEPT
        {	// return pointer to class object
        return (wistd::pointer_traits<pointer>::pointer_to(**this));
        }

    pointer get() const WI_NOEXCEPT
        {	// return pointer to object
        return (this->_Myptr);
        }

    explicit operator bool() const WI_NOEXCEPT
        {	// test for non-null pointer
        return (this->_Myptr != pointer());
        }

    pointer release() WI_NOEXCEPT
        {	// yield ownership of pointer
        pointer _Ans = this->_Myptr;
        this->_Myptr = pointer();
        return (_Ans);
        }

    void reset(pointer _Ptr = pointer()) WI_NOEXCEPT
        {	// establish new pointer
        if (_Ptr != this->_Myptr)
            {	// different pointer, delete old and reassign
            _Delete();
            this->_Myptr = _Ptr;
            }
        }

private:
    void _Delete()
        {	// delete the pointer
        if (this->_Myptr != pointer())
            this->get_deleter()(this->_Myptr);
        }

    unique_ptr(const _Myt&);    // not defined
    _Myt& operator=(const _Myt&);   // not defined
    };

    // TEMPLATE CLASS unique_ptr ARRAY
template<class _Ty,
    class _Dx>
    class unique_ptr<_Ty[], _Dx>
        : private _Unique_ptr_base<_Ty, _Dx,
            is_empty<_Dx>::value
                || is_same<default_delete<_Ty[]>, _Dx>::value>
    {	// non-copyable pointer to an array object
public:
    typedef unique_ptr<_Ty[], _Dx> _Myt;
    typedef _Unique_ptr_base<_Ty, _Dx,
        is_empty<_Dx>::value
            || is_same<default_delete<_Ty[]>, _Dx>::value> _Mybase;
    typedef typename _Mybase::pointer pointer;
    typedef _Ty element_type;
    typedef _Dx deleter_type;

    using _Mybase::get_deleter;

    unique_ptr() WI_NOEXCEPT
        : _Mybase(pointer())
        {	// default construct
        static_assert(!is_pointer<_Dx>::value,
            "unique_ptr constructed with null deleter pointer");
        }

    explicit unique_ptr(pointer _Ptr) WI_NOEXCEPT
        : _Mybase(_Ptr)
        {	// construct with pointer
        static_assert(!is_pointer<_Dx>::value,
            "unique_ptr constructed with null deleter pointer");
        }

    unique_ptr(pointer _Ptr,
        typename _If<is_reference<_Dx>::value, _Dx,
            const typename remove_reference<_Dx>::type&>::type _Dt) WI_NOEXCEPT
        : _Mybase(_Ptr, _Dt)
        {	// construct with pointer and (maybe const) deleter&
        }

    unique_ptr(pointer _Ptr,
        typename remove_reference<_Dx>::type&& _Dt) WI_NOEXCEPT
        : _Mybase(_Ptr, wistd::move(_Dt))
        {	// construct by moving deleter
        static_assert(!is_reference<_Dx>::value,
            "unique_ptr constructed with reference to rvalue deleter");
        }

    unique_ptr(unique_ptr&& _Right) WI_NOEXCEPT
        : _Mybase(_Right.release(),
            wistd::forward<_Dx>(_Right.get_deleter()))
        {	// construct by moving _Right
        }

private:
    template<class _Ty2,
        class _Dx2>
        unique_ptr(unique_ptr<_Ty2, _Dx2>&& _Right);	// not defined

    template<class _Ty2,
        class _Dx2>
        _Myt& operator=(unique_ptr<_Ty2, _Dx2>&& _Right);	// not defined

public:
    _Myt& operator=(_Myt&& _Right) WI_NOEXCEPT
        {	// assign by moving _Right
        if (this != &_Right)
            {	// different, do the swap
            reset(_Right.release());
            this->get_deleter() = wistd::move(_Right.get_deleter());
            }
        return (*this);
        }

    unique_ptr(nullptr_t) WI_NOEXCEPT
        : _Mybase(pointer())
        {	// null pointer construct
        static_assert(!is_pointer<_Dx>::value,
            "unique_ptr constructed with null deleter pointer");
        }

    _Myt& operator=(nullptr_t) WI_NOEXCEPT
        {	// assign a null pointer
        reset();
        return (*this);
        }

    void reset(nullptr_t) WI_NOEXCEPT
        {	// establish new null pointer
        if (this->_Myptr != 0)
            {	// different pointer, delete old and reassign
            _Delete();
            this->_Myptr = 0;
            }
        }

    void swap(_Myt& _Right) WI_NOEXCEPT
        {	// swap elements
        _Swap_adl_wil(this->_Myptr, _Right._Myptr);
        _Swap_adl_wil(this->get_deleter(), _Right.get_deleter());
        }

    ~unique_ptr() WI_NOEXCEPT
        {	// destroy the object
        _Delete();
        }

    typename add_reference<_Ty>::type operator[](size_t _Idx) const
        {	// return reference to object
        return (this->_Myptr[_Idx]);
        }

    pointer get() const WI_NOEXCEPT
        {	// return pointer to object
        return (this->_Myptr);
        }

    explicit operator bool() const WI_NOEXCEPT
        {	// test for non-null pointer
        return (this->_Myptr != pointer());
        }

    pointer release() WI_NOEXCEPT
        {	// yield ownership of pointer
        pointer _Ans = this->_Myptr;
        this->_Myptr = pointer();
        return (_Ans);
        }

    void reset(pointer _Ptr = pointer()) WI_NOEXCEPT
        {	// establish new pointer
        if (_Ptr != this->_Myptr)
            {	// different pointer, delete old and reassign
            _Delete();
            this->_Myptr = _Ptr;
            }
        }

private:
    template<class _Ptr2>
        explicit unique_ptr(_Ptr2); // not defined

    template<class _Ptr2,
        class _Dx2>
        unique_ptr(_Ptr2, _Dx2); // not defined

    unique_ptr(const _Myt&); // not defined
    template<class _Ty2,
        class _Dx2>
        unique_ptr(const unique_ptr<_Ty2, _Dx2>&);  // not defined

    _Myt& operator=(const _Myt&); // not defined
    template<class _Ty2,
        class _Dx2>
        _Myt& operator=(const unique_ptr<_Ty2, _Dx2>&); //not defined

    template<class _Ptr2>
        void reset(_Ptr2); // not defined

    void _Delete()
        {	// delete the pointer
        this->get_deleter()(this->_Myptr);
        }
    };

template<class _Ty,
    class _Dx>
    void swap(unique_ptr<_Ty, _Dx>& _Left,
        unique_ptr<_Ty, _Dx>& _Right) WI_NOEXCEPT
    {	// swap _Left with _Right
    _Left.swap(_Right);
    }

template<class _Ty1,
    class _Dx1,
    class _Ty2,
    class _Dx2>
    bool operator==(const unique_ptr<_Ty1, _Dx1>& _Left,
        const unique_ptr<_Ty2, _Dx2>& _Right)
    {	// test if unique_ptr _Left equals _Right
    return (_Left.get() == _Right.get());
    }

template<class _Ty1,
    class _Dx1,
    class _Ty2,
    class _Dx2>
    bool operator!=(const unique_ptr<_Ty1, _Dx1>& _Left,
        const unique_ptr<_Ty2, _Dx2>& _Right)
    {	// test if unique_ptr _Left doesn't equal _Right
    return (!(_Left == _Right));
    }

template<class _Ty1,
    class _Dx1,
    class _Ty2,
    class _Dx2>
    bool operator<(const unique_ptr<_Ty1, _Dx1>& _Left,
        const unique_ptr<_Ty2, _Dx2>& _Right)
    {	// test if unique_ptr _Left precedes _Right
    return (less<decltype(_Always_false<_Ty1>::value
        ? _Left.get() : _Right.get())>()(
            _Left.get(), _Right.get()));
    }

template<class _Ty1,
    class _Dx1,
    class _Ty2,
    class _Dx2>
    bool operator>=(const unique_ptr<_Ty1, _Dx1>& _Left,
        const unique_ptr<_Ty2, _Dx2>& _Right)
    {	// test if unique_ptr _Left doesn't precede _Right
    return (!(_Left < _Right));
    }

template<class _Ty1,
    class _Dx1,
    class _Ty2,
    class _Dx2>
    bool operator>(const unique_ptr<_Ty1, _Dx1>& _Left,
        const unique_ptr<_Ty2, _Dx2>& _Right)
    {	// test if unique_ptr _Right precedes _Left
    return (_Right < _Left);
    }

template<class _Ty1,
    class _Dx1,
    class _Ty2,
    class _Dx2>
    bool operator<=(const unique_ptr<_Ty1, _Dx1>& _Left,
        const unique_ptr<_Ty2, _Dx2>& _Right)
    {	// test if unique_ptr _Right doesn't precede _Left
    return (!(_Right < _Left));
    }

template<class _Ty,
    class _Dx>
    bool operator==(const unique_ptr<_Ty, _Dx>& _Left,
        nullptr_t) WI_NOEXCEPT
    {	// test if unique_ptr == nullptr
    return (!_Left);
    }

template<class _Ty,
    class _Dx>
    bool operator==(nullptr_t,
        const unique_ptr<_Ty, _Dx>& _Right) WI_NOEXCEPT
    {	// test if nullptr == unique_ptr
    return (!_Right);
    }

template<class _Ty,
    class _Dx>
    bool operator!=(const unique_ptr<_Ty, _Dx>& _Left,
        nullptr_t _Right) WI_NOEXCEPT
    {	// test if unique_ptr != nullptr
    return (!(_Left == _Right));
    }

template<class _Ty,
    class _Dx>
    bool operator!=(nullptr_t _Left,
        const unique_ptr<_Ty, _Dx>& _Right) WI_NOEXCEPT
    {	// test if nullptr != unique_ptr
    return (!(_Left == _Right));
    }

template<class _Ty,
    class _Dx>
    bool operator<(const unique_ptr<_Ty, _Dx>& _Left,
        nullptr_t _Right)
    {	// test if unique_ptr < nullptr
    typedef typename unique_ptr<_Ty, _Dx>::pointer _Ptr;
    return (less<_Ptr>()(_Left.get(), _Right));
    }

template<class _Ty,
    class _Dx>
    bool operator<(nullptr_t _Left,
        const unique_ptr<_Ty, _Dx>& _Right)
    {	// test if nullptr < unique_ptr
    typedef typename unique_ptr<_Ty, _Dx>::pointer _Ptr;
    return (less<_Ptr>()(_Left, _Right.get()));
    }

template<class _Ty,
    class _Dx>
    bool operator>=(const unique_ptr<_Ty, _Dx>& _Left,
        nullptr_t _Right)
    {	// test if unique_ptr >= nullptr
    return (!(_Left < _Right));
    }

template<class _Ty,
    class _Dx>
    bool operator>=(nullptr_t _Left,
        const unique_ptr<_Ty, _Dx>& _Right)
    {	// test if nullptr >= unique_ptr
    return (!(_Left < _Right));
    }

template<class _Ty,
    class _Dx>
    bool operator>(const unique_ptr<_Ty, _Dx>& _Left,
        nullptr_t _Right)
    {	// test if unique_ptr > nullptr
    return (_Right < _Left);
    }

template<class _Ty,
    class _Dx>
    bool operator>(nullptr_t _Left,
        const unique_ptr<_Ty, _Dx>& _Right)
    {	// test if nullptr > unique_ptr
    return (_Right < _Left);
    }

template<class _Ty,
    class _Dx>
    bool operator<=(const unique_ptr<_Ty, _Dx>& _Left,
        nullptr_t _Right)
    {	// test if unique_ptr <= nullptr
    return (!(_Right < _Left));
    }

template<class _Ty,
    class _Dx>
    bool operator<=(nullptr_t _Left,
        const unique_ptr<_Ty, _Dx>& _Right)
    {	// test if nullptr <= unique_ptr
    return (!(_Right < _Left));
    }

} // wistd
/// @endcond
