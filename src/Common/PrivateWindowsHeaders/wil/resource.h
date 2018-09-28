// Windows Internal Libraries (wil)
// Resource.h: RAII wrappers (smart pointers) and other thin usability pattern wrappers over common Windows patterns.
//
// wil Usage Guidelines:
// https://microsoft.sharepoint.com/teams/osg_development/Shared%20Documents/Windows%20Internal%20Libraries%20for%20C++%20Usage%20Guide.docx?web=1
//
// wil Discussion Alias (wildisc):
// http://idwebelements/GroupManagement.aspx?Group=wildisc&Operation=join  (one-click join)
//
// This file primarily contains the following concepts:
// * std::unique_ptr<> and std::shared_ptr<> inspired RAII wrappers over most Windows handle types
// * Thin utility methods to adapt common APIs to these RAII wrappers
// * Thin C++ usability wrappers to improve the patterns dealing with common Windows functionality (locks, etc)
// 
// General usage:
// Note that this file *depends* upon the relevant handles being wrapped to have already been included *before* including
// this file.  In support of this, it intentionally does not use #pragma once to allow other RAII wrappers to be added
// through subsequent inclusion when the #include environment is different from the first time the file was included.
//
//! @file
//! Uniform RAII (smart pointers) for managing memory, Windows handle types and locking

#include "ResultMacros.h"
#include "wistd_memory.h"
#include "wistd_functional.h"

#pragma warning(push)
#pragma warning(disable: 26135 26110)   // Missing locking annotation, Caller failing to hold lock
#pragma warning(disable:4714)           // __forceinline not honored

#ifndef __WIL_RESOURCE
#define __WIL_RESOURCE

// Forward declaration
/// @cond
namespace Microsoft
{
    namespace WRL
    {
        template <typename T>
        class ComPtr;
    }
}
/// @endcond

namespace wil
{
    /// @cond
    namespace details
    {
        typedef wistd::integral_constant<size_t, 0> pointer_access_all;             // get(), release(), addressof(), and '&' are available
        typedef wistd::integral_constant<size_t, 1> pointer_access_noaddress;       // get() and release() are available
        typedef wistd::integral_constant<size_t, 2> pointer_access_none;            // the raw pointer is not available

        template <typename pointer,                                                 // The handle type
                  typename close_fn_t,                                              // The handle close function type
                  close_fn_t close_fn,                                              //      * and function pointer
                  typename pointer_access = pointer_access_all,                     // all, noaddress or none to control pointer method access
                  typename pointer_storage = pointer,                               // The type used to store the handle (usually the same as the handle itself)
                  pointer invalid = pointer(),                                      // The invalid handle value (default ZERO value)
                  typename pointer_invalid = wistd::nullptr_t>                      // nullptr_t if the invalid handle value is compatible with nullptr, otherwise pointer
        struct resource_policy
        {
            typedef typename pointer_storage pointer_storage;
            typedef typename pointer pointer;
            typedef typename pointer_invalid pointer_invalid;
            typedef typename pointer_access pointer_access;
            __forceinline static pointer_storage invalid_value() WI_NOEXCEPT            { return invalid; }
            __forceinline static bool is_valid(pointer_storage value) WI_NOEXCEPT       { return (static_cast<pointer>(value) != static_cast<pointer>(invalid)); }
            __forceinline static void close(pointer_storage value) WI_NOEXCEPT          { close_fn(value); }
            inline static void close_reset(pointer_storage value) WI_NOEXCEPT           { auto error = ::GetLastError(); close_fn(value); ::SetLastError(error);  }
        };


        // This class provides the pointer storage behind the implementation of unique_any_t utilizing the given
        // resource_policy.  It is separate from unique_any_t to allow a type-specific specialization class to plug
        // into the inheritance chain between unique_any_t and unique_storage.  This allows classes like unique_event
        // to be a unique_any formed class, but also expose methods like SetEvent directly.

        template <typename policy>
        class unique_storage
        {
        protected:
            typedef typename policy policy;
            typedef typename policy::pointer_storage pointer_storage;
            typedef typename policy::pointer pointer;
            typedef typename unique_storage<policy> base_storage;

            unique_storage() WI_NOEXCEPT :
                m_ptr(policy::invalid_value())
            {
            }

            explicit unique_storage(pointer_storage ptr) WI_NOEXCEPT :
                m_ptr(ptr)
            {
            }

            unique_storage(unique_storage &&other) WI_NOEXCEPT :
                m_ptr(wistd::move(other.m_ptr))
            {
                other.m_ptr = policy::invalid_value();
            }

            ~unique_storage() WI_NOEXCEPT
            {
                if (policy::is_valid(m_ptr))
                {
                    policy::close(m_ptr);
                }
            }

            void replace(unique_storage &&other) WI_NOEXCEPT
            {
                reset(other.m_ptr);
                other.m_ptr = policy::invalid_value();
            }

            bool is_valid() const WI_NOEXCEPT
            {
                return policy::is_valid(m_ptr);
            }

        public:
            void reset(pointer_storage ptr = policy::invalid_value()) WI_NOEXCEPT
            {
                if (policy::is_valid(m_ptr))
                {
                    policy::close_reset(m_ptr);
                }
                m_ptr = ptr;
            }

            void reset(wistd::nullptr_t) WI_NOEXCEPT
            {
                static_assert(wistd::is_same<policy::pointer_invalid, wistd::nullptr_t>::value, "reset(nullptr): valid only for handle types using nullptr as the invalid value");
                reset();
            }

            pointer get() const WI_NOEXCEPT
            {
                return static_cast<pointer>(m_ptr);
            }

            pointer_storage release() WI_NOEXCEPT
            {
                static_assert(!wistd::is_same<policy::pointer_access, pointer_access_none>::value, "release(): the raw handle value is not available for this resource class");
                auto ptr = m_ptr;
                m_ptr = policy::invalid_value();
                return ptr;
            }

            pointer_storage *addressof() WI_NOEXCEPT
            {
                static_assert(wistd::is_same<policy::pointer_access, pointer_access_all>::value, "addressof(): the address of the raw handle is not available for this resource class");
                return &m_ptr;
            }

        private:
            pointer_storage m_ptr;
        };
    } // details
    /// @endcond


    // This class when paired with unique_storage and an optional type-specific specialization class implements
    // the same interface as STL's unique_ptr<> for resource handle types.  It is a non-copyable, yet movable class
    // supporting attach (reset), detach (release), retrieval (get()).

    template <typename storage_t>
    class unique_any_t : public storage_t
    {
    public:
        typedef typename storage_t::policy policy;
        typedef typename policy::pointer_storage pointer_storage;
        typedef typename policy::pointer pointer;

        unique_any_t(unique_any_t const &) = delete;
        unique_any_t& operator=(unique_any_t const &) = delete;

        // Note that the default constructor really shouldn't be needed (taken care of by the forwarding constructor below), but
        // the forwarding constructor causes an internal compiler error when the class is used in a C++ array.  Defining the default
        // constructor independent of the forwarding constructor removes the compiler limitation.
        unique_any_t() = default;

        // forwarding constructor: forwards all 'explicit' and multi-arg constructors to the base class
        template <typename arg1, typename... args_t>
        explicit unique_any_t(arg1 && first, args_t&&... args) :  // should not be WI_NOEXCEPT (may forward to a throwing constructor)
            storage_t(wistd::forward<arg1>(first), wistd::forward<args_t>(args)...)
        {
            static_assert(wistd::is_same<policy::pointer_access, details::pointer_access_none>::value ||
                          wistd::is_same<policy::pointer_access, details::pointer_access_all>::value ||
                          wistd::is_same<policy::pointer_access, details::pointer_access_noaddress>::value, "pointer_access policy must be a known pointer_access* integral type");
        }

        unique_any_t(wistd::nullptr_t) WI_NOEXCEPT
        {
            static_assert(wistd::is_same<policy::pointer_invalid, wistd::nullptr_t>::value, "nullptr constructor: valid only for handle types using nullptr as the invalid value");
        }

        unique_any_t(unique_any_t &&other) WI_NOEXCEPT :
            storage_t(wistd::move(other))
        {
        }

        unique_any_t& operator=(unique_any_t &&other) WI_NOEXCEPT
        {
            if (this != wistd::addressof(other))
            {
                // cast to base_storage to 'skip' calling the (optional) specialization class that provides handle-specific functionality
                replace(wistd::move(static_cast<base_storage &>(other)));
            }
            return (*this);
        }

        unique_any_t& operator=(wistd::nullptr_t) WI_NOEXCEPT
        {
            static_assert(wistd::is_same<policy::pointer_invalid, wistd::nullptr_t>::value, "nullptr assignment: valid only for handle types using nullptr as the invalid value");
            reset();
            return (*this);
        }

        void swap(unique_any_t &other) WI_NOEXCEPT
        {
            unique_any_t self(wistd::move(*this));
            operator=(wistd::move(other));
            other = wistd::move(self);
        }

        explicit operator bool() const WI_NOEXCEPT
        {
            return is_valid();
        }

        pointer_storage *operator&() WI_NOEXCEPT
        {
            static_assert(wistd::is_same<policy::pointer_access, details::pointer_access_all>::value, "operator & is not available for this handle");
            reset();
            return addressof();
        }

        pointer get() const WI_NOEXCEPT
        {
            static_assert(!wistd::is_same<policy::pointer_access, details::pointer_access_none>::value, "get(): the raw handle value is not available for this resource class");
            return storage_t::get();
        }

        // The following functions are publicly exposed by their inclusion in the unique_storage base class

        // explicit unique_any_t(pointer_storage ptr) WI_NOEXCEPT
        // void reset(pointer_storage ptr = policy::invalid_value()) WI_NOEXCEPT
        // void reset(wistd::nullptr_t) WI_NOEXCEPT
        // pointer_storage release() WI_NOEXCEPT                                        // not exposed for some resource types
        // pointer_storage *addressof() WI_NOEXCEPT                                     // not exposed for some resource types
    };

    template <typename policy>
    void swap(unique_any_t<policy>& left, unique_any_t<policy>& right) WI_NOEXCEPT
    {
        left.swap(right);
    }

    template <typename policy>
    bool operator==(const unique_any_t<policy>& left, const unique_any_t<policy>& right) WI_NOEXCEPT
    {
        return (left.get() == right.get());
    }

    template <typename policy>
    bool operator==(const unique_any_t<policy>& left, wistd::nullptr_t) WI_NOEXCEPT
    {
        static_assert(wistd::is_same<unique_any_t<policy>::policy::pointer_invalid, wistd::nullptr_t>::value, "the resource class does not use nullptr as an invalid value");
        return (left.get() == nullptr);
    }

    template <typename policy>
    bool operator==(wistd::nullptr_t, const unique_any_t<policy>& right) WI_NOEXCEPT
    {
        static_assert(wistd::is_same<unique_any_t<policy>::policy::pointer_invalid, wistd::nullptr_t>::value, "the resource class does not use nullptr as an invalid value");
        return (nullptr == right.get());
    }

    template <typename policy>
    bool operator!=(const unique_any_t<policy>& left, const unique_any_t<policy>& right) WI_NOEXCEPT
    {
        return (!(left.get() == right.get()));
    }

    template <typename policy>
    bool operator!=(const unique_any_t<policy>& left, wistd::nullptr_t) WI_NOEXCEPT
    {
        static_assert(wistd::is_same<unique_any_t<policy>::policy::pointer_invalid, wistd::nullptr_t>::value, "the resource class does not use nullptr as an invalid value");
        return (!(left.get() == nullptr));
    }

    template <typename policy>
    bool operator!=(wistd::nullptr_t, const unique_any_t<policy>& right) WI_NOEXCEPT
    {
        static_assert(wistd::is_same<unique_any_t<policy>::policy::pointer_invalid, wistd::nullptr_t>::value, "the resource class does not use nullptr as an invalid value");
        return (!(nullptr == right.get()));
    }

    template <typename policy>
    bool operator<(const unique_any_t<policy>& left, const unique_any_t<policy>& right) WI_NOEXCEPT
    {
        return (left.get() < right.get());
    }

    template <typename policy>
    bool operator>=(const unique_any_t<policy>& left, const unique_any_t<policy>& right) WI_NOEXCEPT
    {
        return (!(left < right));
    }

    template <typename policy>
    bool operator>(const unique_any_t<policy>& left, const unique_any_t<policy>& right) WI_NOEXCEPT
    {
        return (right < left);
    }

    template <typename policy>
    bool operator<=(const unique_any_t<policy>& left, const unique_any_t<policy>& right) WI_NOEXCEPT
    {
        return (!(right < left));
    }


    // unique_any provides a template alias for easily building a unique_any_t from a unique_storage class with the given
    // template parameters for resource_policy.

    template <typename pointer,                                         // The handle type
              typename close_fn_t,                                      // The handle close function type
              close_fn_t close_fn,                                      //      * and function pointer
              typename pointer_access = details::pointer_access_all,    // all, noaddress or none to control pointer method access
              typename pointer_storage = pointer,                       // The type used to store the handle (usually the same as the handle itself)
              pointer invalid = pointer(),                              // The invalid handle value (default ZERO value)
              typename pointer_invalid = wistd::nullptr_t>              // nullptr_t if the invalid handle value is compatible with nullptr, otherwise pointer
    using unique_any = unique_any_t<details::unique_storage<details::resource_policy<pointer, close_fn_t, close_fn, pointer_access, pointer_storage, invalid, pointer_invalid>>>;


    // Forward declaration...
    template <typename T, typename err_policy>
    class com_ptr_t;

    //! Type traits class that identifies the inner type of a smart pointer.
    //! Works for STL pointer types, WIL unique_any types, WIL com_ptr types and WRL ComPtr
    template <typename Ptr>
    struct smart_pointer_details
    {
        typedef typename Ptr::pointer pointer;
    };

    /// @cond
    template <typename T>
    struct smart_pointer_details<Microsoft::WRL::ComPtr<T>>
    {
        typedef T* pointer;
    };
    /// @endcond

    //! Detaches and returns the inner pointer from any smart pointer type.
    //! This function detaches pointer ownership from the given smart pointer type.  For STL and `unique_any`
    //! types this ends up calling `release()`.  For `com_ptr` it calls `detach()` and for WRL `ComPtr` it calls
    //! `Detach()`.
    //! @param ptr  A reference to the smart pointer to detach the pointer from.
    //! @return     The detached raw resource pointer.
    template <typename Ptr>
    _Check_return_ typename Ptr::pointer detach_from_smart_pointer(_Inout_ Ptr& ptr)
    {
        return ptr.release();
    }

    /// @cond
    template <typename T, typename err>
    _Check_return_ T* detach_from_smart_pointer(_Inout_ wil::com_ptr_t<T, err>& ptr)
    {
        return ptr.detach();
    }

    template <typename T>
    _Check_return_ T* detach_from_smart_pointer(_Inout_ Microsoft::WRL::ComPtr<T>& ptr)
    {
        return ptr.Detach();
    }
    /// @endcond

    //! Attaches a pointer to any compatible smart pointer type.
    //! This function gives pointer ownership to the given smart pointer type.  For STL and `unique_any`
    //! types this ends up calling `reset(ptr)`.  For com_ptr it calls `attach(ptr)` and for WRL `ComPtr` it calls
    //! `Attach(ptr)`.
    //! @param ptr  A reference to the smart pointer to attach the resource to.
    //! @param raw  The raw resource pointer to attach to the smart pointer.

    //! @cond
    template<typename T, typename err> class com_ptr_t; // forward
    namespace details
    {
        // The first two attach_to_smart_pointer() overloads are ambiguous when passed a com_ptr_t.
        // To solve that use this functions return type to elminate the reset form for com_ptr_t.
        template <typename T, typename err> wistd::false_type use_reset(wil::com_ptr_t<T, err>*) { return wistd::false_type(); }
        template <typename T> wistd::true_type use_reset(T*) { return wistd::true_type(); }
    }
    //! @endcond

    template <typename Ptr, typename EnableResetForm = wistd::enable_if_t<decltype(details::use_reset(static_cast<Ptr*>(nullptr)))::value>>
    void attach_to_smart_pointer(Ptr& ptr, typename Ptr::pointer raw)
    {
        ptr.reset(raw);
    }

    /// @cond
    template <typename T, typename err>
    void attach_to_smart_pointer(wil::com_ptr_t<T, err>& ptr, typename T* raw)
    {
        ptr.attach(raw);
    }

    template <typename T>
    void attach_to_smart_pointer(Microsoft::WRL::ComPtr<T>& ptr, typename T* raw)
    {
        ptr.Attach(raw);
    }
    /// @endcond

    //! @ingroup outparam
    //! Detach a pointer contained within a smart pointer class to an optional output pointer parameter.
    //! Functions may have optional output pointer parameters that want to receive an object from
    //! a function.  Using this routine provides a trivial nullptr test and calls the appropriate detach
    //! routine for the given smart pointer type.
    //! @see AssignNullToOptParam for an example
    //! @param outParam The optional out-pointer
    //! @param smartPtr The smart pointer class to detach and set in `outParam`.  This can be any of the well
    //!                 known smart pointer types, such as `Microsoft::WRL::ComPtr`, `wil::com_ptr`, `wil::unique*`,
    //!                 and `std::unique_ptr`.
    template <typename T, typename TSmartPointer>
    inline void DetachToOptParam(_Out_opt_ T* outParam, _Inout_ TSmartPointer&& smartPtr)
    {
        if (outParam)
        {
            *outParam = detach_from_smart_pointer(smartPtr);
        }
    }

    /// @cond
    namespace details
    {
        template <typename T>
        struct out_param_t
        {
            typedef typename wil::smart_pointer_details<T>::pointer pointer;
            T &wrapper;
            pointer pRaw;
            bool replace = true;

            out_param_t(_Inout_ T &output) :
                wrapper(output),
                pRaw(nullptr)
            {
            }

            out_param_t(out_param_t&& other) :
                wrapper(other.wrapper),
                pRaw(other.pRaw)
            {
                WI_ASSERT(other.replace);
                other.replace = false;
            }

            operator pointer*()
            {
                WI_ASSERT(replace);
                return &pRaw;
            }

            ~out_param_t()
            {
                if (replace)
                {
                    attach_to_smart_pointer(wrapper, pRaw);
                }
            }

            out_param_t(out_param_t const &other) = delete;
            out_param_t &operator=(out_param_t const &other) = delete;
        };

        template <typename Tcast, typename T>
        struct out_param_ptr_t
        {
            typedef typename wil::smart_pointer_details<T>::pointer pointer;
            T &wrapper;
            pointer pRaw;
            bool replace = true;

            out_param_ptr_t(_Inout_ T &output) :
                wrapper(output),
                pRaw(nullptr)
            {
            }

            out_param_ptr_t(out_param_ptr_t&& other) :
                wrapper(other.wrapper),
                pRaw(other.pRaw)
            {
                WI_ASSERT(other.replace);
                other.replace = false;
            }

            operator typename Tcast()
            {
                WI_ASSERT(replace);
                return reinterpret_cast<Tcast>(&pRaw);
            }

            ~out_param_ptr_t()
            {
                if (replace)
                {
                    attach_to_smart_pointer(wrapper, pRaw);
                }
            }

            out_param_ptr_t(out_param_ptr_t const &other) = delete;
            out_param_ptr_t &operator=(out_param_ptr_t const &other) = delete;
        };
    } // details
    /// @endcond

    // This class is meant to be used with smart pointers (such as unique_any, unique_ptr, and shared_any) to provide
    // a mechanism for simplifying the retrieval of an out-parameter for a class which doesn't support getting the
    // address of the pointer (like std::unique_ptr<>).  This allows writing:
    //     unique_ptr<Foo> foo;
    //     GetFoo(out_param(foo));
    // Since unique_ptr<> doesn't support the '&' operator (it doesn't expose the raw pointer) this model works for
    // doing a within-line replacement of the unique_ptr.

    template <typename T>
    details::out_param_t<T> out_param(T& p)
    {
        return details::out_param_t<T>(p);
    }

    template <typename Tcast, typename T>
    details::out_param_ptr_t<Tcast, T> out_param_ptr(T& p)
    {
        return details::out_param_ptr_t<Tcast, T>(p);
    }

    /** Use unique_struct to define an RAII type for a trivial struct that references resources that must be cleaned up.
    Unique_struct wraps a trivial struct using a custom clean up function and, optionally, custom initializer function. If no custom initialier function is defined in the template
    then ZeroMemory is used.
    Unique_struct is modeled off of std::unique_ptr. However, unique_struct inherits from the defined type instead of managing the struct through a private member variable.

    If the type you're wrapping is a system type, you can share the code by declaring it in this file (Resource.h). Send requests to wildisc.
    Otherwise, if the type is local to your project, declare it locally.
    @tparam struct_t The struct you want to manage
    @tparam close_fn_t The type of the function to clean up the struct. Takes one parameter: a pointer of struct_t. Return values are ignored.
    @tparam close_fn The function of type close_fn_t. This is called in the destructor and reset functions.
    @tparam init_fn_t Optional:The type of the function to initialize the struct.  Takes one parameter: a pointer of struct_t. Return values are ignored.
    @tparam init_fn Optional:The function of type init_fn_t. This is called in the constructor, reset, and release functions. The default is ZeroMemory to initialize the struct.
    
    Defined using the default zero memory initializer
    ~~~
    typedef wil::unique_struct<PROPVARIANT, decltype(&::PropVariantClear), ::PropVariantClear> unique_prop_variant_default_init;

    unique_prop_variant propvariant;
    SomeFunction(&propvariant);
    ~~~ 
    
    Defined using a custom initializer
    ~~~
    typedef wil::unique_struct<PROPVARIANT, decltype(&::PropVariantClear), ::PropVariantClear, decltype(&::PropVariantInit), ::PropVariantInit> unique_prop_variant;

    unique_prop_variant propvariant;
    SomeFunction(&propvariant);
    ~~~
    */
    template <typename struct_t, typename close_fn_t, close_fn_t close_fn, typename init_fn_t = wistd::nullptr_t, init_fn_t init_fn = wistd::nullptr_t()>
    class unique_struct : public struct_t
    {
    public:
        //! Initializes the managed struct using the user-provided initialization function, or ZeroMemory if no function is specified
        unique_struct()
        {
            call_init(use_default_init_fn());
        }

        //! Takes ownership of the struct by doing a shallow copy. Must explicitly be type struct_t
        explicit unique_struct(const struct_t& other) WI_NOEXCEPT :
            struct_t(other)
        {}

        //! Initializes the managed struct by taking the ownership of the other managed struct
        //! Then resets the other managed struct by calling the custom close function
        unique_struct(unique_struct&& other) WI_NOEXCEPT :
            struct_t(other.release())
        {}

        //! Resets this managed struct by calling the custom close function and takes ownership of the other managed struct
        //! Then resets the other managed struct by calling the custom close function
        unique_struct & operator=(unique_struct&& other) WI_NOEXCEPT
        {
            if (this != &other)
            {
                reset(other.release());
            }
            return *this;
        }

        //! Calls the custom close function
        ~unique_struct() WI_NOEXCEPT
        {
            close_fn(this);
        }

        void reset(const unique_struct&) = delete;

        //! Resets this managed struct by calling the custom close function and begins management of the other struct
        void reset(const struct_t& other) WI_NOEXCEPT
        {
            auto error = ::GetLastError();
            close_fn(this);
            ::SetLastError(error);
            struct_t::operator=(other);
        }

        //! Resets this managed struct by calling the custom close function
        //! Then initializes this managed struct using the user-provided initialization function, or ZeroMemory if no function is specified
        void reset() WI_NOEXCEPT
        {
            close_fn(this);
            call_init(use_default_init_fn());
        }

        void swap(struct_t&) = delete;

        //! Swaps the managed structs
        void swap(unique_struct& other) WI_NOEXCEPT
        {
            struct_t self(*this);
            struct_t::operator=(other);
            *(other.addressof()) = self;
        }

        //! Returns the managed struct
        //! Then initializes this managed struct using the user-provided initialization function, or ZeroMemory if no function is specified
        struct_t release() WI_NOEXCEPT
        {
            struct_t value(*this);
            call_init(use_default_init_fn());
            return value;
        }

        //! Returns address of the managed struct
        struct_t * addressof() WI_NOEXCEPT
        {
            return this;
        }

        //! Resets this managed struct by calling the custom close function
        //! Then initializes this managed struct using the user-provided initialization function, or ZeroMemory if no function is specified
        //! Returns address of the managed struct
        struct_t * reset_and_addressof() WI_NOEXCEPT
        {
            reset();
            return this;
        }

        unique_struct(const unique_struct&) = delete;
        unique_struct& operator=(const unique_struct&) = delete;
        unique_struct& operator=(const struct_t&) = delete;

    private:
        typedef typename wistd::is_same<init_fn_t, wistd::nullptr_t>::type use_default_init_fn;

        void call_init(wistd::true_type)
        {
            ZeroMemory(this, sizeof(*this));
        }

        void call_init(wistd::false_type)
        {
            init_fn(this);
        }
    };


    /** unique_any_array_ptr is a RAII type for managing conformant arrays that need to be freed and have elements that may need to be freed.
    The intented use for this RAII type would be to capture out params from API like IPropertyValue::GetStringArray.
    This class also maintains the size of the array, so it can iterate over the members and deallocate them before it deallocates the base array pointer.

    If the type you're wrapping is a system type, you can share the code by declaring it in this file (Resource.h). Send requests to wildisc.
    Otherwise, if the type is local to your project, declare it locally.

    @tparam ValueType: The type of array you want to manage.
    @tparam ArrayDeleter: The type of the function to clean up the array. Takes one parameter of type T[] or T*. Return values are ignored. This is called in the destructor and reset functions.
    @tparam ElementDeleter: The type of the function to clean up the array elements. Takes one parameter of type T. Return values are ignored. This is called in the destructor and reset functions.
    
    ~~~
    void GetSomeArray(_Out_ size_t*, _Out_ NOTMYTYPE**);

    struct not_my_deleter
    {
        void operator()(NOTMYTYPE p) const
        {
            destroy(p);
        }
    };

    wil::unique_any_array_ptr<NOTMYTYPE, ::CoTaskMemFree, not_my_deleter> myArray;
    GetSomeArray(myArray.size_address(), &myArray);
    ~~~
    */
    struct empty_deleter
    {
        template <typename T>
        void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T) const
        {
        }
    };

    template <typename ValueType, typename ArrayDeleter, typename ElementDeleter = empty_deleter>
    class unique_any_array_ptr
    {
    public:
        typedef ValueType value_type;
        typedef size_t size_type;
        typedef ptrdiff_t difference_type;
        typedef ValueType *pointer;
        typedef const ValueType *const_pointer;
        typedef ValueType& reference;
        typedef const ValueType& const_reference;

        typedef ValueType* iterator;
        typedef const ValueType* const_iterator;

        unique_any_array_ptr() = default;
        unique_any_array_ptr(const unique_any_array_ptr&) = delete;
        unique_any_array_ptr& operator=(const unique_any_array_ptr&) = delete;

        unique_any_array_ptr(wistd::nullptr_t) WI_NOEXCEPT
        {
        }

        unique_any_array_ptr& operator=(wistd::nullptr_t) WI_NOEXCEPT
        {
            reset();
            return *this;
        }

        unique_any_array_ptr(pointer ptr, size_t size) WI_NOEXCEPT : m_ptr(ptr), m_size(size)
        {
        }

        unique_any_array_ptr(unique_any_array_ptr&& other) WI_NOEXCEPT : m_ptr(other.m_ptr), m_size(other.m_size)
        {
            other.m_ptr = nullptr;
            other.m_size = size_type{};
        }

        unique_any_array_ptr& operator=(unique_any_array_ptr&& other) WI_NOEXCEPT
        {
            if (this != wistd::addressof(other))
            {
                reset();
                swap(other);
            }
            return *this;
        }

        ~unique_any_array_ptr() WI_NOEXCEPT
        {
            reset();
        }

        void swap(unique_any_array_ptr& other) WI_NOEXCEPT
        {
            auto ptr = m_ptr;
            auto size = m_size;
            m_ptr = other.m_ptr;
            m_size = other.m_size;
            other.m_ptr = ptr;
            other.m_size = size;
        }

        iterator begin() WI_NOEXCEPT
        {
            return (iterator(m_ptr));
        }

        const_iterator begin() const WI_NOEXCEPT
        {
            return (const_iterator(m_ptr));
        }

        iterator end() WI_NOEXCEPT
        {
            return (iterator(m_ptr + m_size));
        }

        const_iterator end() const WI_NOEXCEPT
        {
            return (const_iterator(m_ptr + m_size));
        }

        const_iterator cbegin() const WI_NOEXCEPT
        {
            return (begin());
        }

        const_iterator cend() const WI_NOEXCEPT
        {
            return (end());
        }

        size_type size() const WI_NOEXCEPT
        {
            return (m_size);
        }

        bool empty() const WI_NOEXCEPT
        {
            return (size() == size_type{});
        }

        reference operator[](size_type position)
        {
            WI_ASSERT(position < m_size);
            _Analysis_assume_(position < m_size);
            return (m_ptr[position]);
        }

        const_reference operator[](size_type position) const
        {
            WI_ASSERT(position < m_size);
            _Analysis_assume_(position < m_size);
            return (m_ptr[position]);
        }

        reference front()
        {
            WI_ASSERT(!empty());
            return (m_ptr[0]);
        }

        const_reference front() const
        {
            WI_ASSERT(!empty());
            return (m_ptr[0]);
        }

        reference back()
        {
            WI_ASSERT(!empty());
            return (m_ptr[m_size - 1]);
        }

        const_reference back() const
        {
            WI_ASSERT(!empty());
            return (m_ptr[m_size - 1]);
        }

        ValueType* data() WI_NOEXCEPT
        {
            return (m_ptr);
        }

        const ValueType* data() const WI_NOEXCEPT
        {
            return (m_ptr);
        }

        pointer get() const WI_NOEXCEPT
        {
            return m_ptr;
        }

        explicit operator bool() const WI_NOEXCEPT
        {
            return (m_ptr != pointer());
        }

        pointer release() WI_NOEXCEPT
        {
            auto result = m_ptr;
            m_ptr = nullptr;
            m_size = size_type{};
            return result;
        }

        void reset() WI_NOEXCEPT
        {
            if (m_ptr)
            {
                reset_array(ElementDeleter());
                ArrayDeleter()(m_ptr);
                m_ptr = nullptr;
                m_size = size_type{};
            }
        }

        void reset(pointer ptr, size_t size) WI_NOEXCEPT
        {
            reset();
            m_ptr = ptr;
            m_size = size;
        }

        pointer* addressof() WI_NOEXCEPT
        {
            return &m_ptr;
        }

        pointer* operator&() WI_NOEXCEPT
        {
            reset();
            return addressof();
        }

        size_type* size_address() WI_NOEXCEPT
        {
            return &m_size;
        }

        template <typename TSize>
        struct size_address_ptr
        {
            unique_any_array_ptr& wrapper;
            TSize size{};
            bool replace = true;

            size_address_ptr(_Inout_ unique_any_array_ptr& output) :
                wrapper(output)
            {
            }

            size_address_ptr(size_address_ptr&& other) :
                wrapper(other.wrapper),
                size(other.size)
            {
                WI_ASSERT(other.replace);
                other.replace = false;
            }

            operator typename TSize*()
            {
                WI_ASSERT(replace);
                return &size;
            }

            ~size_address_ptr()
            {
                if (replace)
                {
                    *wrapper.size_address() = static_cast<size_type>(size);
                }
            }

            size_address_ptr(size_address_ptr const &other) = delete;
            size_address_ptr &operator=(size_address_ptr const &other) = delete;
        };

        template <typename T>
        size_address_ptr<T> size_address() WI_NOEXCEPT
        {
            return size_address_ptr<T>(*this);
        }

    private:
        pointer m_ptr = nullptr;
        size_type m_size{};

        void reset_array(const empty_deleter&)
        {
        }

        template <typename T>
        void reset_array(const T& deleter)
        {
            for (auto& element : make_range(m_ptr, m_size))
            {
                deleter(element);
            }
        }
    };

    // forward declaration
    template <typename T, typename err_policy>
    class com_ptr_t;

    namespace details
    {
        template <typename UniqueAnyType>
        struct unique_any_array_deleter
        {
            template <typename T>
            void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T* p) const
            {
                UniqueAnyType::policy::close_reset(p);
            }
        };

        template <typename close_fn_t, close_fn_t close_fn>
        struct unique_struct_array_deleter
        {
            template <typename T>
            void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T& p) const
            {
                close_fn(&p);
            }
        };

        struct com_unknown_deleter
        {
            template <typename T>
            void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T* p) const
            {
                if (p)
                {
                    p->Release();
                }
            }
        };

        template <class T>
        struct element_traits
        {
            typedef empty_deleter deleter;
            typedef T type;
        };

        template <typename storage_t>
        struct element_traits<unique_any_t<storage_t>>
        {
            typedef typename unique_any_array_deleter<unique_any_t<storage_t>> deleter;
            typedef typename unique_any_t<storage_t>::pointer type;
        };

        template <typename T, typename err_policy>
        struct element_traits<com_ptr_t<T, err_policy>>
        {
            typedef com_unknown_deleter deleter;
            typedef T* type;
        };

        template <typename struct_t, typename close_fn_t, close_fn_t close_fn, typename init_fn_t, init_fn_t init_fn>
        struct element_traits<unique_struct<struct_t, close_fn_t, close_fn, init_fn_t, init_fn>>
        {
            typedef typename unique_struct_array_deleter<close_fn_t, close_fn> deleter;
            typedef struct_t type;
        };
    }

    template <typename T, typename ArrayDeleter>
    using unique_array_ptr = unique_any_array_ptr<typename details::element_traits<T>::type, ArrayDeleter, typename details::element_traits<T>::deleter>;

} // namespace wil
#endif // __WIL_RESOURCE


// Hash deferral function for unique_any_t
#if (defined(_UNORDERED_SET_) || defined(_UNORDERED_MAP_)) && !defined(__WIL_RESOURCE_UNIQUE_HASH)
#define __WIL_RESOURCE_UNIQUE_HASH
namespace std
{
    template <typename storage_t>
    struct hash<wil::unique_any_t<storage_t>> :
        public std::unary_function<wil::unique_any_t<storage_t>, size_t>
    {
        size_t operator()(wil::unique_any_t<storage_t> const &val) const
        {
            return (hash<wil::unique_any_t<storage_t>::pointer>()(val.get()));
        }
    };
}
#endif

// shared_any and weak_any implementation using <memory> STL header
#if defined(_MEMORY_) && defined(WIL_ENABLE_EXCEPTIONS) && !defined(WIL_RESOURCE_STL) && !defined(RESOURCE_SUPPRESS_STL)
#define WIL_RESOURCE_STL
namespace wil {

template <typename storage_t>
class weak_any;

/// @cond
namespace details
{
    // This class provides the pointer storage behind the implementation of shared_any_t utilizing the given
    // resource_policy.  It is separate from shared_any_t to allow a type-specific specialization class to plug
    // into the inheritance chain between shared_any_t and shared_storage.  This allows classes like shared_event
    // to be a shared_any formed class, but also expose methods like SetEvent directly.

    template <typename unique_t>
    class shared_storage
    {
    protected:
        typedef typename unique_t unique_t;
        typedef typename unique_t::policy policy;
        typedef typename policy::pointer_storage pointer_storage;
        typedef typename policy::pointer pointer;
        typedef typename shared_storage<unique_t> base_storage;

        shared_storage() = default;

        explicit shared_storage(pointer_storage ptr)
        {
            if (policy::is_valid(ptr))
            {
                m_ptr = std::make_shared<unique_t>(unique_t(ptr));      // unique_t on the stack to prevent leak on throw
            }
        }

        shared_storage(unique_t &&other)
        {
            if (other)
            {
                m_ptr = std::make_shared<unique_t>(wistd::move(other));
            }
        }

        shared_storage(const shared_storage &other) WI_NOEXCEPT :
            m_ptr(other.m_ptr)
        {
        }

        shared_storage& operator=(const shared_storage &other) WI_NOEXCEPT
        {
            m_ptr = other.m_ptr;
            return *this;
        }

        shared_storage(shared_storage &&other) WI_NOEXCEPT :
            m_ptr(wistd::move(other.m_ptr))
        {
        }

        shared_storage(std::shared_ptr<unique_t> const &ptr) :
            m_ptr(ptr)
        {
        }

        void replace(shared_storage &&other) WI_NOEXCEPT
        {
            m_ptr = wistd::move(other.m_ptr);
        }

        bool is_valid() const WI_NOEXCEPT
        {
            return (m_ptr && static_cast<bool>(*m_ptr));
        }

    public:
        void reset(pointer_storage ptr = policy::invalid_value())
        {
            if (policy::is_valid(ptr))
            {
                m_ptr = std::make_shared<unique_t>(unique_t(ptr));      // unique_t on the stack to prevent leak on throw
            }
            else
            {
                m_ptr = nullptr;
            }
        }

        void reset(unique_t &&other)
        {
            m_ptr = std::make_shared<unique_t>(wistd::move(other));
        }

        void reset(wistd::nullptr_t) WI_NOEXCEPT
        {
            static_assert(wistd::is_same<policy::pointer_invalid, wistd::nullptr_t>::value, "reset(nullptr): valid only for handle types using nullptr as the invalid value");
            reset();
        }

        template <typename allow_t = typename policy::pointer_access, typename wistd::enable_if<!wistd::is_same<allow_t, details::pointer_access_none>::value, int>::type = 0>
        pointer get() const WI_NOEXCEPT
        {
            return (m_ptr ? m_ptr->get() : policy::invalid_value());
        }

        template <typename allow_t = typename policy::pointer_access, typename wistd::enable_if<wistd::is_same<allow_t, details::pointer_access_all>::value, int>::type = 0>
        pointer_storage *addressof()
        {
            if (!m_ptr)
            {
                m_ptr = std::make_shared<unique_t>();
            }
            return m_ptr->addressof();
        }

        long int use_count() const WI_NOEXCEPT
        {
            return m_ptr.use_count();
        }

        bool unique() const WI_NOEXCEPT
        {
            return m_ptr.unique();
        }

    private:
        template <typename storage_t>
        friend class weak_any;

        std::shared_ptr<unique_t> m_ptr;
    };
}
/// @endcond

// This class when paired with shared_storage and an optional type-specific specialization class implements
// the same interface as STL's shared_ptr<> for resource handle types.  It is both copyable and movable, supporting
// weak references and automatic closure of the handle upon release of the last shared_any.

template <typename storage_t>
class shared_any_t : public storage_t
{
public:
    typedef typename storage_t::policy policy;
    typedef typename policy::pointer_storage pointer_storage;
    typedef typename policy::pointer pointer;
    typedef typename storage_t::unique_t unique_t;

    // default and forwarding constructor: forwards default, all 'explicit' and multi-arg constructors to the base class
    template <typename... args_t>
    explicit shared_any_t(args_t&&... args) :  // should not be WI_NOEXCEPT (may forward to a throwing constructor)
        storage_t(wistd::forward<args_t>(args)...)
    {
    }

    shared_any_t(wistd::nullptr_t) WI_NOEXCEPT
    {
        static_assert(wistd::is_same<policy::pointer_invalid, wistd::nullptr_t>::value, "nullptr constructor: valid only for handle types using nullptr as the invalid value");
    }

    shared_any_t(shared_any_t &&other) WI_NOEXCEPT :
        storage_t(wistd::move(other))
    {
    }

    shared_any_t(const shared_any_t &other) WI_NOEXCEPT :
        storage_t(other)
    {
    }

    shared_any_t& operator=(shared_any_t &&other) WI_NOEXCEPT
    {
        if (this != wistd::addressof(other))
        {
            replace(wistd::move(static_cast<base_storage &>(other)));
        }
        return (*this);
    }

    shared_any_t& operator=(const shared_any_t& other) WI_NOEXCEPT
    {
        storage_t::operator=(other);
        return (*this);
    }

    shared_any_t(unique_t &&other) :
        storage_t(wistd::move(other))
    {
    }

    shared_any_t& operator=(unique_t &&other)
    {
        reset(wistd::move(other));
        return (*this);
    }

    shared_any_t& operator=(wistd::nullptr_t) WI_NOEXCEPT
    {
        static_assert(wistd::is_same<policy::pointer_invalid, wistd::nullptr_t>::value, "nullptr assignment: valid only for handle types using nullptr as the invalid value");
        reset();
        return (*this);
    }

    void swap(shared_any_t &other) WI_NOEXCEPT
    {
        shared_any_t self(wistd::move(*this));
        operator=(wistd::move(other));
        other = wistd::move(self);
    }

    explicit operator bool() const WI_NOEXCEPT
    {
        return is_valid();
    }

    pointer_storage *operator&()
    {
        static_assert(wistd::is_same<policy::pointer_access, details::pointer_access_all>::value, "operator & is not available for this handle");
        reset();
        return addressof();
    }

    pointer get() const WI_NOEXCEPT
    {
        static_assert(!wistd::is_same<policy::pointer_access, details::pointer_access_none>::value, "get(): the raw handle value is not available for this resource class");
        return storage_t::get();
    }

    // The following functions are publicly exposed by their inclusion in the base class

    // void reset(pointer_storage ptr = policy::invalid_value()) WI_NOEXCEPT
    // void reset(wistd::nullptr_t) WI_NOEXCEPT
    // pointer_storage *addressof() WI_NOEXCEPT                                     // (note: not exposed for opaque resource types)
};

template <typename unique_t>
void swap(shared_any_t<unique_t>& left, shared_any_t<unique_t>& right) WI_NOEXCEPT
{
    left.swap(right);
}

template <typename unique_t>
bool operator==(const shared_any_t<unique_t>& left, const shared_any_t<unique_t>& right) WI_NOEXCEPT
{
    return (left.get() == right.get());
}

template <typename unique_t>
bool operator==(const shared_any_t<unique_t>& left, wistd::nullptr_t) WI_NOEXCEPT
{
    static_assert(wistd::is_same<shared_any_t<unique_t>::policy::pointer_invalid, wistd::nullptr_t>::value, "the resource class does not use nullptr as an invalid value");
    return (left.get() == nullptr);
}

template <typename unique_t>
bool operator==(wistd::nullptr_t, const shared_any_t<unique_t>& right) WI_NOEXCEPT
{
    static_assert(wistd::is_same<shared_any_t<unique_t>::policy::pointer_invalid, wistd::nullptr_t>::value, "the resource class does not use nullptr as an invalid value");
    return (nullptr == right.get());
}

template <typename unique_t>
bool operator!=(const shared_any_t<unique_t>& left, const shared_any_t<unique_t>& right) WI_NOEXCEPT
{
    return (!(left.get() == right.get()));
}

template <typename unique_t>
bool operator!=(const shared_any_t<unique_t>& left, wistd::nullptr_t) WI_NOEXCEPT
{
    static_assert(wistd::is_same<shared_any_t<unique_t>::policy::pointer_invalid, wistd::nullptr_t>::value, "the resource class does not use nullptr as an invalid value");
    return (!(left.get() == nullptr));
}

template <typename unique_t>
bool operator!=(wistd::nullptr_t, const shared_any_t<unique_t>& right) WI_NOEXCEPT
{
    static_assert(wistd::is_same<shared_any_t<unique_t>::policy::pointer_invalid, wistd::nullptr_t>::value, "the resource class does not use nullptr as an invalid value");
    return (!(nullptr == right.get()));
}

template <typename unique_t>
bool operator<(const shared_any_t<unique_t>& left, const shared_any_t<unique_t>& right) WI_NOEXCEPT
{
    return (left.get() < right.get());
}

template <typename unique_t>
bool operator>=(const shared_any_t<unique_t>& left, const shared_any_t<unique_t>& right) WI_NOEXCEPT
{
    return (!(left < right));
}

template <typename unique_t>
bool operator>(const shared_any_t<unique_t>& left, const shared_any_t<unique_t>& right) WI_NOEXCEPT
{
    return (right < left);
}

template <typename unique_t>
bool operator<=(const shared_any_t<unique_t>& left, const shared_any_t<unique_t>& right) WI_NOEXCEPT
{
    return (!(right < left));
}


// This class provides weak_ptr<> support for shared_any<>, bringing the same weak reference counting and lock() acquire semantics
// to shared_any.

template <typename shared_t>
class weak_any
{
public:
    typedef shared_t shared_t;

    weak_any() WI_NOEXCEPT
    {
    }

    weak_any(const shared_t &other) WI_NOEXCEPT :
        m_weakPtr(other.m_ptr)
    {
    }

    weak_any(const weak_any &other) WI_NOEXCEPT :
        m_weakPtr(other.m_weakPtr)
    {
    }

    weak_any& operator=(const weak_any &right) WI_NOEXCEPT
    {
        m_weakPtr = right.m_weakPtr;
        return (*this);
    }

    weak_any& operator=(const shared_t &right) WI_NOEXCEPT
    {
        m_weakPtr = right.m_ptr;
        return (*this);
    }

    void reset() WI_NOEXCEPT
    {
        m_weakPtr.reset();
    }

    void swap(weak_any &other) WI_NOEXCEPT
    {
        m_weakPtr.swap(other.m_weakPtr);
    }

    bool expired() const WI_NOEXCEPT
    {
        return m_weakPtr.expired();
    }

    shared_t lock() const WI_NOEXCEPT
    {
        return shared_t(m_weakPtr.lock());
    }

private:
    std::weak_ptr<typename shared_t::unique_t> m_weakPtr;
};

template <typename shared_t>
void swap(weak_any<shared_t>& left, weak_any<shared_t>& right) WI_NOEXCEPT
{
    left.swap(right);
}

template <typename unique_t>
using shared_any = shared_any_t<details::shared_storage<unique_t>>;

} // namespace wil
#endif


#if defined(WIL_RESOURCE_STL) && (defined(_UNORDERED_SET_) || defined(_UNORDERED_MAP_)) && !defined(__WIL_RESOURCE_SHARED_HASH)
#define __WIL_RESOURCE_SHARED_HASH
namespace std
{
    template <typename storage_t>
    struct hash<wil::shared_any_t<storage_t>> :
        public std::unary_function<wil::shared_any_t<storage_t>, size_t>
    {
        size_t operator()(wil::shared_any_t<storage_t> const &val) const
        {
            return (hash<wil::shared_any_t<storage_t>::pointer>()(val.get()));
        }
    };
}
#endif


namespace wil
{
#if defined(__NOTHROW_T_DEFINED) && !defined(__WIL__NOTHROW_T_DEFINED)
#define __WIL__NOTHROW_T_DEFINED
    /** Provides `std::make_unique()` semantics for resources allocated in a context that may not throw upon allocation failure.
    `wil::make_unique_nothrow()` is identical to `std::make_unique()` except for the following:
    * It returns `wistd::unique_ptr`, rather than `std::unique_ptr`
    * It returns an empty (null) `wistd::unique_ptr` upon allocation failure, rather than throwing an exception

    Note that `wil::make_unique_nothrow()` is not marked WI_NOEXCEPT as it may be used to create an exception-based class that may throw in its constructor.
    ~~~
    auto foo = wil::make_unique_nothrow<Foo>(fooConstructorParam1, fooConstructorParam2);
    if (foo)
    {
        foo->Bar();
    }
    ~~~
    */
    template <class _Ty, class... _Types>
    inline typename wistd::enable_if<!wistd::is_array<_Ty>::value, wistd::unique_ptr<_Ty> >::type make_unique_nothrow(_Types&&... _Args)
    {
        return (wistd::unique_ptr<_Ty>(new(std::nothrow) _Ty(wistd::forward<_Types>(_Args)...)));
    }

    /** Provides `std::make_unique()` semantics for array resources allocated in a context that may not throw upon allocation failure.
    See the overload of `wil::make_unique_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_nothrow<Foo[]>(size); // the default constructor will be called on each Foo object
    if (foos)
    {
        for (auto& elem : wil::make_range(foos.get(), size))
        {
            elem.Bar();
        }
    }
    ~~~
    */
    template <class _Ty>
    inline typename wistd::enable_if<wistd::is_array<_Ty>::value && wistd::extent<_Ty>::value == 0, wistd::unique_ptr<_Ty> >::type make_unique_nothrow(size_t _Size)
    {
        typedef typename wistd::remove_extent<_Ty>::type _Elem;
        return (wistd::unique_ptr<_Ty>(new(std::nothrow) _Elem[_Size]()));
    }

    template <class _Ty, class... _Types>
    typename wistd::enable_if<wistd::extent<_Ty>::value != 0, void>::type make_unique_nothrow(_Types&&...) = delete;

    /** Provides `std::make_unique()` semantics for resources allocated in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_unique_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_failfast<Foo>(fooConstructorParam1, fooConstructorParam2);
    foo->Bar();
    ~~~
    */
    template <class _Ty, class... _Types>
    inline typename wistd::enable_if<!wistd::is_array<_Ty>::value, wistd::unique_ptr<_Ty> >::type make_unique_failfast(_Types&&... _Args)
    {
        #pragma warning(suppress: 28193)    // temporary must be inspected (it is within the called function)
        return (wistd::unique_ptr<_Ty>(FAIL_FAST_IF_NULL_ALLOC(new(std::nothrow) _Ty(wistd::forward<_Types>(_Args)...))));
    }

    /** Provides `std::make_unique()` semantics for array resources allocated in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_unique_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_nothrow<Foo[]>(size); // the default constructor will be called on each Foo object
    for (auto& elem : wil::make_range(foos.get(), size))
    {
        elem.Bar();
    }
    ~~~
    */
    template <class _Ty>
    inline typename wistd::enable_if<wistd::is_array<_Ty>::value && wistd::extent<_Ty>::value == 0, wistd::unique_ptr<_Ty> >::type make_unique_failfast(size_t _Size)
    {
        typedef typename wistd::remove_extent<_Ty>::type _Elem;
        #pragma warning(suppress: 28193)    // temporary must be inspected (it is within the called function)
        return (wistd::unique_ptr<_Ty>(FAIL_FAST_IF_NULL_ALLOC(new(std::nothrow) _Elem[_Size]())));
    }

    template <class _Ty, class... _Types>
    typename wistd::enable_if<wistd::extent<_Ty>::value != 0, void>::type make_unique_failfast(_Types&&...) = delete;
#endif // __WIL__NOTHROW_T_DEFINED


#if defined(_WINBASE_) && !defined(__WIL_WINBASE_)
#define __WIL_WINBASE_
    /// @cond
    namespace details
    {
        inline void SetEvent(HANDLE h) WI_NOEXCEPT
        { __FAIL_FAST_ASSERT_WIN32_BOOL_FALSE__(::SetEvent(h)); }

        inline void ResetEvent(HANDLE h) WI_NOEXCEPT
        { __FAIL_FAST_ASSERT_WIN32_BOOL_FALSE__(::ResetEvent(h)); }

        inline void CloseHandle(HANDLE h) WI_NOEXCEPT
        { __FAIL_FAST_ASSERT_WIN32_BOOL_FALSE__(::CloseHandle(h)); }

        inline void ReleaseSemaphore(_In_ HANDLE h) WI_NOEXCEPT
        { __FAIL_FAST_ASSERT_WIN32_BOOL_FALSE__(::ReleaseSemaphore(h, 1, nullptr)); }

        inline void ReleaseMutex(_In_ HANDLE h) WI_NOEXCEPT
        { __FAIL_FAST_ASSERT_WIN32_BOOL_FALSE__(::ReleaseMutex(h)); }

        inline void CloseTokenLinkedToken(_In_ TOKEN_LINKED_TOKEN* linkedToken) WI_NOEXCEPT
        {
            if (linkedToken->LinkedToken && (linkedToken->LinkedToken != INVALID_HANDLE_VALUE))
            {
                __FAIL_FAST_ASSERT_WIN32_BOOL_FALSE__(::CloseHandle(linkedToken->LinkedToken));
            }
        }

        enum class PendingCallbackCancellationBehavior
        {
            Cancel,
            Wait,
            NoWait,
        };

        template <PendingCallbackCancellationBehavior cancellationBehavior>
        struct DestroyThreadPoolWait
        {
            static void Destroy(_In_ PTP_WAIT threadPoolWait) WI_NOEXCEPT
            {
                static_assert(cancellationBehavior != PendingCallbackCancellationBehavior::NoWait, "PendingCallbackCancellationBehavior::NoWait is not supported");
                ::WaitForThreadpoolWaitCallbacks(threadPoolWait, (cancellationBehavior == PendingCallbackCancellationBehavior::Cancel));
                ::CloseThreadpoolWait(threadPoolWait);
            }
        };

        template <PendingCallbackCancellationBehavior cancellationBehavior>
        struct DestroyThreadPoolWork
        {
            static void Destroy(_In_ PTP_WORK threadpoolWork) WI_NOEXCEPT
            {
                static_assert(cancellationBehavior != PendingCallbackCancellationBehavior::NoWait, "PendingCallbackCancellationBehavior::NoWait is not supported");
                ::WaitForThreadpoolWorkCallbacks(threadpoolWork, (cancellationBehavior == PendingCallbackCancellationBehavior::Cancel));
                ::CloseThreadpoolWork(threadpoolWork);
            }
        };

        // PendingCallbackCancellationBehavior::NoWait explicitly does not block waiting for
        // callbacks when destructing.  SetThreadpoolTimer(timer, nullptr, 0, 0)
        // will cancel any pending callbacks, then CloseThreadpoolTimer will
        // asynchronusly close the timer if a callback is running.
        template <PendingCallbackCancellationBehavior cancellationBehavior>
        struct DestroyThreadPoolTimer
        {
            static void Destroy(_In_ PTP_TIMER threadpoolTimer) WI_NOEXCEPT
            {
                ::SetThreadpoolTimer(threadpoolTimer, nullptr, 0, 0);
                #pragma warning(suppress:4127) // conditional expression is constant
                if (cancellationBehavior != PendingCallbackCancellationBehavior::NoWait)
                {
                    ::WaitForThreadpoolTimerCallbacks(threadpoolTimer, (cancellationBehavior == PendingCallbackCancellationBehavior::Cancel));
                }
                ::CloseThreadpoolTimer(threadpoolTimer);
            }
        };

        template <PendingCallbackCancellationBehavior cancellationBehavior>
        struct DestroyThreadPoolIo
        {
            static void Destroy(_In_ PTP_IO threadpoolIo) WI_NOEXCEPT
            {
                static_assert(cancellationBehavior != PendingCallbackCancellationBehavior::NoWait, "PendingCallbackCancellationBehavior::NoWait is not supported");
                ::WaitForThreadpoolIoCallbacks(threadpoolIo, (cancellationBehavior == PendingCallbackCancellationBehavior::Cancel));
                ::CloseThreadpoolIo(threadpoolIo);
            }
        };

        template <typename close_fn_t, close_fn_t close_fn>
        struct handle_invalid_resource_policy : resource_policy<HANDLE, close_fn_t, close_fn, details::pointer_access_all, HANDLE, INVALID_HANDLE_VALUE, HANDLE>
        {
            __forceinline static bool is_valid(pointer ptr) WI_NOEXCEPT   { return ((ptr != INVALID_HANDLE_VALUE) && (ptr != nullptr)); }
        };

        template <typename close_fn_t, close_fn_t close_fn>
        struct handle_null_resource_policy : resource_policy<HANDLE, close_fn_t, close_fn>
        {
            __forceinline static bool is_valid(pointer ptr) WI_NOEXCEPT   { return ((ptr != nullptr) && (ptr != INVALID_HANDLE_VALUE)); }
        };

        typedef resource_policy<HANDLE, decltype(&details::CloseHandle), details::CloseHandle, details::pointer_access_all> handle_resource_policy;
    }
    /// @endcond

    template <typename close_fn_t, close_fn_t close_fn>
    using unique_any_handle_invalid = unique_any_t<details::unique_storage<details::handle_invalid_resource_policy<close_fn_t, close_fn>>>;

    template <typename close_fn_t, close_fn_t close_fn>
    using unique_any_handle_null = unique_any_t<details::unique_storage<details::handle_null_resource_policy<close_fn_t, close_fn>>>;

    typedef unique_any_handle_invalid<decltype(&::CloseHandle), ::CloseHandle> unique_hfile;
    typedef unique_any_handle_null<decltype(&::CloseHandle), ::CloseHandle> unique_handle;
    typedef unique_any_handle_invalid<decltype(&::FindClose), ::FindClose> unique_hfind;
    typedef unique_any<HMODULE, decltype(&::FreeLibrary), ::FreeLibrary> unique_hmodule;

    typedef unique_struct<TOKEN_LINKED_TOKEN, decltype(&details::CloseTokenLinkedToken), details::CloseTokenLinkedToken> unique_token_linked_token;

    using unique_tool_help_snapshot = unique_hfile;

    typedef unique_any<PTP_WAIT, void(*)(PTP_WAIT), details::DestroyThreadPoolWait<details::PendingCallbackCancellationBehavior::Cancel>::Destroy> unique_threadpool_wait;
    typedef unique_any<PTP_WAIT, void(*)(PTP_WAIT), details::DestroyThreadPoolWait<details::PendingCallbackCancellationBehavior::Wait>::Destroy> unique_threadpool_wait_nocancel;
    typedef unique_any<PTP_WORK, void(*)(PTP_WORK), details::DestroyThreadPoolWork<details::PendingCallbackCancellationBehavior::Cancel>::Destroy> unique_threadpool_work;
    typedef unique_any<PTP_WORK, void(*)(PTP_WORK), details::DestroyThreadPoolWork<details::PendingCallbackCancellationBehavior::Wait>::Destroy> unique_threadpool_work_nocancel;
    typedef unique_any<PTP_TIMER, void(*)(PTP_TIMER), details::DestroyThreadPoolTimer<details::PendingCallbackCancellationBehavior::Cancel>::Destroy> unique_threadpool_timer;
    typedef unique_any<PTP_TIMER, void(*)(PTP_TIMER), details::DestroyThreadPoolTimer<details::PendingCallbackCancellationBehavior::Wait>::Destroy> unique_threadpool_timer_nocancel;
    typedef unique_any<PTP_TIMER, void(*)(PTP_TIMER), details::DestroyThreadPoolTimer<details::PendingCallbackCancellationBehavior::NoWait>::Destroy> unique_threadpool_timer_nowait;
    typedef unique_any<PTP_IO, void(*)(PTP_IO), details::DestroyThreadPoolIo<details::PendingCallbackCancellationBehavior::Cancel>::Destroy> unique_threadpool_io;
    typedef unique_any<PTP_IO, void(*)(PTP_IO), details::DestroyThreadPoolIo<details::PendingCallbackCancellationBehavior::Wait>::Destroy> unique_threadpool_io_nocancel;

#if WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)    
    typedef unique_any_handle_invalid<decltype(&::FindCloseChangeNotification), ::FindCloseChangeNotification> unique_hfind_change;
#endif

    typedef unique_any<HANDLE, decltype(&details::SetEvent), details::SetEvent, details::pointer_access_noaddress> event_set_scope_exit;
    typedef unique_any<HANDLE, decltype(&details::ResetEvent), details::ResetEvent, details::pointer_access_noaddress> event_reset_scope_exit;

    // Guarantees a SetEvent on the given event handle when the returned object goes out of scope
    // Note: call SetEvent early with the reset() method on the returned object or abort the call with the release() method
    inline event_set_scope_exit SetEvent_scope_exit(HANDLE hEvent) WI_NOEXCEPT
    {
        __FAIL_FAST_ASSERT__(hEvent != nullptr);
        return event_set_scope_exit(hEvent);
    }

    // Guarantees a ResetEvent on the given event handle when the returned object goes out of scope
    // Note: call ResetEvent early with the reset() method on the returned object or abort the call with the release() method
    inline event_reset_scope_exit ResetEvent_scope_exit(HANDLE hEvent) WI_NOEXCEPT
    {
        __FAIL_FAST_ASSERT__(hEvent != nullptr);
        return event_reset_scope_exit(hEvent);
    }

    // Checks to see if the given *manual reset* event is currently signaled.  The event must not be an auto-reset event.
    // Use when the event will only be set once (cancellation-style) or will only be reset by the polling thread
    inline bool event_is_signaled(HANDLE hEvent) WI_NOEXCEPT
    {
        auto status = ::WaitForSingleObjectEx(hEvent, 0, FALSE);
        // Fast fail will trip for wait failures, auto-reset events, or when the event is being both Set and Reset
        // from a thread other than the polling thread (use event_wait directly for those cases).
        __FAIL_FAST_ASSERT__((status == WAIT_TIMEOUT) || ((status == WAIT_OBJECT_0) && (WAIT_OBJECT_0 == ::WaitForSingleObjectEx(hEvent, 0, FALSE))));
        return (status == WAIT_OBJECT_0);
    }

    // Waits on the given handle for the specified duration
    inline bool handle_wait(HANDLE hEvent, DWORD dwMilliseconds = INFINITE) WI_NOEXCEPT
    {
        DWORD status = ::WaitForSingleObjectEx(hEvent, dwMilliseconds, FALSE);
        __FAIL_FAST_ASSERT__((status == WAIT_TIMEOUT) || (status == WAIT_OBJECT_0));
        return (status == WAIT_OBJECT_0);
    }

    enum class EventOptions
    {
        None = 0x0,
        ManualReset = 0x1,
        Signaled = 0x2
    };
    DEFINE_ENUM_FLAG_OPERATORS(EventOptions);

    template <typename storage_t, typename err_policy = err_exception_policy>
    class event_t : public storage_t
    {
    public:
        // forward all base class constructors...
        template <typename... args_t>
        explicit event_t(args_t&&... args) WI_NOEXCEPT : storage_t(wistd::forward<args_t>(args)...) {}

        // HRESULT or void error handling...
        typedef typename err_policy::result result;

        // Exception-based constructor to create an unnamed event
        event_t(EventOptions options)
        {
            static_assert(wistd::is_same<void, result>::value, "this constructor requires exceptions or fail fast; use the create method");
            create(options);
        }

        void ResetEvent() const WI_NOEXCEPT
        {
            details::ResetEvent(get());
        }

        void SetEvent() const WI_NOEXCEPT
        {
            details::SetEvent(get());
        }

        // Guarantees a SetEvent on the given event handle when the returned object goes out of scope
        // Note: call SetEvent early with the reset() method on the returned object or abort the call with the release() method
        event_set_scope_exit SetEvent_scope_exit() const WI_NOEXCEPT
        {
            return wil::SetEvent_scope_exit(get());
        }

        // Guarantees a ResetEvent on the given event handle when the returned object goes out of scope
        // Note: call ResetEvent early with the reset() method on the returned object or abort the call with the release() method
        event_reset_scope_exit ResetEvent_scope_exit() const WI_NOEXCEPT
        {
            return wil::ResetEvent_scope_exit(get());
        }

        // Checks if a *manual reset* event is currently signaled.  The event must not be an auto-reset event.
        // Use when the event will only be set once (cancellation-style) or will only be reset by the polling thread
        bool is_signaled() const WI_NOEXCEPT
        {
            return wil::event_is_signaled(get());
        }

        // Basic WaitForSingleObject on the event handle with the given timeout
        bool wait(DWORD dwMilliseconds = INFINITE) const WI_NOEXCEPT
        {
            return wil::handle_wait(get(), dwMilliseconds);
        }

        // Tries to create a named event -- returns false if unable to do so (gle may still be inspected with return=false)
        bool try_create(EventOptions options, PCWSTR name, _In_opt_ LPSECURITY_ATTRIBUTES pSecurity = nullptr, _Out_opt_ bool *pAlreadyExists = nullptr)
        {
            auto handle = ::CreateEventExW(pSecurity, name, (WI_IS_FLAG_SET(options, EventOptions::ManualReset) ? CREATE_EVENT_MANUAL_RESET : 0) | (WI_IS_FLAG_SET(options, EventOptions::Signaled) ? CREATE_EVENT_INITIAL_SET : 0), EVENT_ALL_ACCESS);
            if (handle == nullptr)
            {
                wil::AssignToOptParam(pAlreadyExists, false);
                return false;
            }
            wil::AssignToOptParam(pAlreadyExists, (::GetLastError() == ERROR_ALREADY_EXISTS));
            reset(handle);
            return true;
        }

        // Returns HRESULT for unique_event_nothrow, void with exceptions for shared_event and unique_event
        result create(EventOptions options = EventOptions::None, PCWSTR name = nullptr, _In_opt_ LPSECURITY_ATTRIBUTES pSecurity = nullptr, _Out_opt_ bool *pAlreadyExists = nullptr)
        {
            return err_policy::LastErrorIfFalse(try_create(options, name, pSecurity, pAlreadyExists));
        }

        // Tries to open the named event -- returns false if unable to do so (gle may still be inspected with return=false)
        bool try_open(_In_ PCWSTR name, DWORD desiredAccess = SYNCHRONIZE | EVENT_MODIFY_STATE, bool inheritHandle = false)
        {
            auto handle = ::OpenEventW(desiredAccess, inheritHandle, name);
            if (handle == nullptr)
            {
                return false;
            }
            reset(handle);
            return true;
        }

        // Returns HRESULT for unique_event_nothrow, void with exceptions for shared_event and unique_event
        result open(_In_ PCWSTR name, DWORD desiredAccess = SYNCHRONIZE | EVENT_MODIFY_STATE, bool inheritHandle = false)
        {
            return err_policy::LastErrorIfFalse(try_open(name, desiredAccess, inheritHandle));
        }
    };

    typedef unique_any_t<event_t<details::unique_storage<details::handle_resource_policy>, err_returncode_policy>>     unique_event_nothrow;
    typedef unique_any_t<event_t<details::unique_storage<details::handle_resource_policy>, err_failfast_policy>>       unique_event_failfast;
#ifdef WIL_ENABLE_EXCEPTIONS
    typedef unique_any_t<event_t<details::unique_storage<details::handle_resource_policy>, err_exception_policy>>      unique_event;
#endif

    typedef unique_any<HANDLE, decltype(&details::ReleaseMutex), details::ReleaseMutex, details::pointer_access_none> mutex_release_scope_exit;

    inline mutex_release_scope_exit ReleaseMutex_scope_exit(_In_ HANDLE hMutex) WI_NOEXCEPT
    {
        __FAIL_FAST_ASSERT__(hMutex != nullptr);
        return mutex_release_scope_exit(hMutex);
    }

    // For efficiency, avoid using mutexes when an srwlock or condition variable will do.
    template <typename storage_t, typename err_policy = err_exception_policy>
    class mutex_t : public storage_t
    {
    public:
        // forward all base class constructors...
        template <typename... args_t>
        explicit mutex_t(args_t&&... args) WI_NOEXCEPT : storage_t(wistd::forward<args_t>(args)...) {}

        // HRESULT or void error handling...
        typedef typename err_policy::result result;

        // Exception-based constructor to create a mutex (prefer unnamed (nullptr) for the name)
        mutex_t(_In_opt_ PCWSTR name)
        {
            static_assert(wistd::is_same<void, result>::value, "this constructor requires exceptions or fail fast; use the create method");
            create(name);
        }

        void ReleaseMutex() const WI_NOEXCEPT
        {
            details::ReleaseMutex(get());
        }

        mutex_release_scope_exit ReleaseMutex_scope_exit() const WI_NOEXCEPT
        {
            return wil::ReleaseMutex_scope_exit(get());
        }

        mutex_release_scope_exit acquire(_Out_opt_ DWORD *pStatus = nullptr, DWORD dwMilliseconds = INFINITE, BOOL bAlertable = FALSE)  const WI_NOEXCEPT
        {
            auto handle = get();
            DWORD status = ::WaitForSingleObjectEx(handle, dwMilliseconds, bAlertable);
            AssignToOptParam(pStatus, status);
            __FAIL_FAST_ASSERT__((status == WAIT_TIMEOUT) || (status == WAIT_OBJECT_0) || (status == WAIT_ABANDONED));
            return mutex_release_scope_exit(((status == WAIT_OBJECT_0) || (status == WAIT_ABANDONED)) ? handle : nullptr);
        }

        // Tries to create a named mutex -- returns false if unable to do so (gle may still be inspected with return=false)
        bool try_create(_In_opt_ PCWSTR name, DWORD dwFlags = 0, DWORD desiredAccess = MUTEX_ALL_ACCESS, _In_opt_ PSECURITY_ATTRIBUTES pMutexAttributes = nullptr)
        {
            auto handle = ::CreateMutexExW(pMutexAttributes, name, dwFlags, desiredAccess);
            if (handle == nullptr)
            {
                return false;
            }
            reset(handle);
            return true;
        }

        // Returns HRESULT for unique_mutex_nothrow, void with exceptions for shared_mutex and unique_mutex
        result create(_In_opt_ PCWSTR name = nullptr, DWORD dwFlags = 0, DWORD desiredAccess = MUTEX_ALL_ACCESS, _In_opt_ PSECURITY_ATTRIBUTES pMutexAttributes = nullptr)
        {
            return err_policy::LastErrorIfFalse(try_create(name, dwFlags, desiredAccess, pMutexAttributes));
        }

        // Tries to open a named mutex -- returns false if unable to do so (gle may still be inspected with return=false)
        bool try_open(_In_ PCWSTR name, DWORD desiredAccess = SYNCHRONIZE | MUTEX_MODIFY_STATE, bool inheritHandle = false)
        {
            auto handle = ::OpenMutexW(desiredAccess, inheritHandle, name);
            if (handle == nullptr)
            {
                return false;
            }
            reset(handle);
            return true;
        }

        // Returns HRESULT for unique_mutex_nothrow, void with exceptions for shared_mutex and unique_mutex
        result open(_In_ PCWSTR name, DWORD desiredAccess = SYNCHRONIZE | MUTEX_MODIFY_STATE, bool inheritHandle = false)
        {
            return err_policy::LastErrorIfFalse(try_open(name, desiredAccess, inheritHandle));
        }
    };

    typedef unique_any_t<mutex_t<details::unique_storage<details::handle_resource_policy>, err_returncode_policy>>     unique_mutex_nothrow;
    typedef unique_any_t<mutex_t<details::unique_storage<details::handle_resource_policy>, err_failfast_policy>>       unique_mutex_failfast;
#ifdef WIL_ENABLE_EXCEPTIONS
    typedef unique_any_t<mutex_t<details::unique_storage<details::handle_resource_policy>, err_exception_policy>>      unique_mutex;
#endif

    typedef unique_any<HANDLE, decltype(&details::ReleaseSemaphore), details::ReleaseSemaphore, details::pointer_access_none> semaphore_release_scope_exit;

    inline semaphore_release_scope_exit ReleaseSemaphore_scope_exit(_In_ HANDLE hSemaphore) WI_NOEXCEPT
    {
        __FAIL_FAST_ASSERT__(hSemaphore != nullptr);
        return semaphore_release_scope_exit(hSemaphore);
    }

    template <typename storage_t, typename err_policy = err_exception_policy>
    class semaphore_t : public storage_t
    {
    public:
        // forward all base class constructors...
        template <typename... args_t>
        explicit semaphore_t(args_t&&... args) WI_NOEXCEPT : storage_t(wistd::forward<args_t>(args)...) {}

        // HRESULT or void error handling...
        typedef typename err_policy::result result;

        // Note that for custom-constructors the type given the constructor has to match exactly as not all implicit conversions will make it through the
        // forwarding constructor.  This constructor, for example, uses 'int' instead of 'LONG' as the count to ease that particular issue (const numbers are int by default).
        explicit semaphore_t(int initialCount, int maximumCount, _In_opt_ PCWSTR name = nullptr, DWORD desiredAccess = SEMAPHORE_ALL_ACCESS, _In_opt_ PSECURITY_ATTRIBUTES pSemaphoreAttributes = nullptr)
        {
            static_assert(wistd::is_same<void, result>::value, "this constructor requires exceptions or fail fast; use the create method");
            create(initialCount, maximumCount, name, desiredAccess, pSemaphoreAttributes);
        }

        void ReleaseSemaphore(long nReleaseCount = 1, _In_opt_ long *pnPreviousCount = nullptr) WI_NOEXCEPT
        {
            long nPreviousCount = 0;
            __FAIL_FAST_ASSERT__(::ReleaseSemaphore(get(), nReleaseCount, &nPreviousCount));
            wil::AssignToOptParam(pnPreviousCount, nPreviousCount);
        }

        semaphore_release_scope_exit ReleaseSemaphore_scope_exit() WI_NOEXCEPT
        {
            return wil::ReleaseSemaphore_scope_exit(get());
        }

        semaphore_release_scope_exit acquire(_Out_opt_ DWORD *pStatus = nullptr, DWORD dwMilliseconds = INFINITE, BOOL bAlertable = FALSE) WI_NOEXCEPT
        {
            auto handle = get();
            DWORD status = ::WaitForSingleObjectEx(handle, dwMilliseconds, bAlertable);
            AssignToOptParam(pStatus, status);
            __FAIL_FAST_ASSERT__((status == WAIT_TIMEOUT) || (status == WAIT_OBJECT_0));
            return semaphore_release_scope_exit((status == WAIT_OBJECT_0) ? handle : nullptr);
        }

        // Tries to create a named event -- returns false if unable to do so (gle may still be inspected with return=false)
        bool try_create(LONG lInitialCount, LONG lMaximumCount, _In_opt_ PCWSTR name, DWORD desiredAccess = SEMAPHORE_ALL_ACCESS, _In_opt_ PSECURITY_ATTRIBUTES pSemaphoreAttributes = nullptr)
        {
            auto handle = ::CreateSemaphoreExW(pSemaphoreAttributes, lInitialCount, lMaximumCount, name, 0, desiredAccess);
            if (handle == nullptr)
            {
                return false;
            }
            reset(handle);
            return true;
        }

        // Returns HRESULT for unique_semaphore_nothrow, void with exceptions for shared_event and unique_event
        result create(LONG lInitialCount, LONG lMaximumCount, _In_opt_ PCWSTR name = nullptr, DWORD desiredAccess = SEMAPHORE_ALL_ACCESS, _In_opt_ PSECURITY_ATTRIBUTES pSemaphoreAttributes = nullptr)
        {
            return err_policy::LastErrorIfFalse(try_create(lInitialCount, lMaximumCount, name, desiredAccess, pSemaphoreAttributes));
        }

        // Tries to open the named semaphore -- returns false if unable to do so (gle may still be inspected with return=false)
        bool try_open(_In_ PCWSTR name, DWORD desiredAccess = SYNCHRONIZE | SEMAPHORE_MODIFY_STATE, bool inheritHandle = false)
        {
            auto handle = ::OpenSemaphoreW(desiredAccess, inheritHandle, name);
            if (handle == nullptr)
            {
                return false;
            }
            reset(handle);
            return true;
        }

        // Returns HRESULT for unique_semaphore_nothrow, void with exceptions for shared_semaphore and unique_semaphore
        result open(_In_ PCWSTR name, DWORD desiredAccess = SYNCHRONIZE | SEMAPHORE_MODIFY_STATE, bool inheritHandle = false)
        {
            return err_policy::LastErrorIfFalse(try_open(name, desiredAccess, inheritHandle));
        }
    };

    typedef unique_any_t<semaphore_t<details::unique_storage<details::handle_resource_policy>, err_returncode_policy>>  unique_semaphore_nothrow;
    typedef unique_any_t<semaphore_t<details::unique_storage<details::handle_resource_policy>, err_failfast_policy>>    unique_semaphore_failfast;
#ifdef WIL_ENABLE_EXCEPTIONS
    typedef unique_any_t<semaphore_t<details::unique_storage<details::handle_resource_policy>, err_exception_policy>>   unique_semaphore;
#endif

    typedef unique_any<SRWLOCK *, decltype(&::ReleaseSRWLockExclusive), ::ReleaseSRWLockExclusive, details::pointer_access_none> rwlock_release_exclusive_scope_exit;
    typedef unique_any<SRWLOCK *, decltype(&::ReleaseSRWLockShared), ::ReleaseSRWLockShared, details::pointer_access_none> rwlock_release_shared_scope_exit;

    inline rwlock_release_exclusive_scope_exit AcquireSRWLockExclusive(_Inout_ SRWLOCK *plock) WI_NOEXCEPT
    {
        ::AcquireSRWLockExclusive(plock);
        return rwlock_release_exclusive_scope_exit(plock);
    }

    inline rwlock_release_shared_scope_exit AcquireSRWLockShared(_Inout_ SRWLOCK *plock) WI_NOEXCEPT
    {
        ::AcquireSRWLockShared(plock);
        return rwlock_release_shared_scope_exit(plock);
    }

    inline rwlock_release_exclusive_scope_exit TryAcquireSRWLockExclusive(_Inout_ SRWLOCK *plock) WI_NOEXCEPT
    {
        return rwlock_release_exclusive_scope_exit(::TryAcquireSRWLockExclusive(plock) ? plock : nullptr);
    }

    inline rwlock_release_shared_scope_exit TryAcquireSRWLockShared(_Inout_ SRWLOCK *plock) WI_NOEXCEPT
    {
        return rwlock_release_shared_scope_exit(::TryAcquireSRWLockShared(plock) ? plock : nullptr);
    }

    class srwlock
    {
    public:
        srwlock(const srwlock&) = delete;
        srwlock& operator=(const srwlock&) = delete;

        srwlock() WI_NOEXCEPT :
            m_lock(SRWLOCK_INIT)
        {
        }

        rwlock_release_exclusive_scope_exit lock_exclusive() WI_NOEXCEPT
        {
            return wil::AcquireSRWLockExclusive(&m_lock);
        }

        rwlock_release_exclusive_scope_exit try_lock_exclusive() WI_NOEXCEPT
        {
            return wil::TryAcquireSRWLockExclusive(&m_lock);
        }

        rwlock_release_shared_scope_exit lock_shared() WI_NOEXCEPT
        {
            return wil::AcquireSRWLockShared(&m_lock);
        }

        rwlock_release_shared_scope_exit try_lock_shared() WI_NOEXCEPT
        {
            return wil::TryAcquireSRWLockShared(&m_lock);
        }

    private:
        SRWLOCK m_lock;
    };


    typedef unique_any<CRITICAL_SECTION *, decltype(&::LeaveCriticalSection), ::LeaveCriticalSection, details::pointer_access_none> cs_leave_scope_exit;

    inline cs_leave_scope_exit EnterCriticalSection(_Inout_ CRITICAL_SECTION *pcs) WI_NOEXCEPT
    {
        ::EnterCriticalSection(pcs);
        return cs_leave_scope_exit(pcs);
    }

    inline cs_leave_scope_exit TryEnterCriticalSection(_Inout_ CRITICAL_SECTION *pcs) WI_NOEXCEPT
    {
        return cs_leave_scope_exit(::TryEnterCriticalSection(pcs) ? pcs : nullptr);
    }

#if defined(_NTURTL_) && !defined(__WIL_NTURTL_CRITSEC_)
#define __WIL_NTURTL_CRITSEC_

    typedef unique_any<RTL_CRITICAL_SECTION*, decltype(&::RtlLeaveCriticalSection), RtlLeaveCriticalSection, details::pointer_access_none> rtlcs_leave_scope_exit;

    inline cs_leave_scope_exit RtlEnterCriticalSection(_Inout_ RTL_CRITICAL_SECTION *pcs) WI_NOEXCEPT
    {
        ::RtlEnterCriticalSection(pcs);
        return cs_leave_scope_exit(pcs);
    }

    inline cs_leave_scope_exit RtlTryEnterCriticalSection(_Inout_ RTL_CRITICAL_SECTION *pcs) WI_NOEXCEPT
    {
        return cs_leave_scope_exit(::RtlTryEnterCriticalSection(pcs) ? pcs : nullptr);
    }

#endif

    // Critical sections are worse than srwlocks in performance and memory usage (their only unique attribute
    // being recursive acquisition).  Prefer srwlocks over critical sections in new code.
    class critical_section   
    {
    public:
        critical_section(const critical_section&) = delete;
        critical_section& operator=(const critical_section&) = delete;

        critical_section(ULONG spincount = 0) WI_NOEXCEPT
        {
            // Initialization will not fail without invalid params...
            ::InitializeCriticalSectionEx(&m_cs, spincount, 0);
        }

        ~critical_section() WI_NOEXCEPT
        {
            ::DeleteCriticalSection(&m_cs);
        }

        cs_leave_scope_exit lock() WI_NOEXCEPT
        {
            return wil::EnterCriticalSection(&m_cs);
        }

        cs_leave_scope_exit try_lock() WI_NOEXCEPT
        {
            return wil::TryEnterCriticalSection(&m_cs);
        }

    private:
        CRITICAL_SECTION m_cs;
    };

    /// @cond
    namespace details
    {
        struct SecureZeroData
        {
            void *pointer;
            size_t sizeBytes;
            SecureZeroData(void *pointer_, size_t sizeBytes_ = 0) WI_NOEXCEPT { pointer = pointer_; sizeBytes = sizeBytes_; }
            operator void *() const WI_NOEXCEPT { return pointer; }
            static void Close(SecureZeroData data) WI_NOEXCEPT { ::SecureZeroMemory(data.pointer, data.sizeBytes); }
        };
    }
    /// @endcond

    typedef unique_any<void*, decltype(&details::SecureZeroData::Close), details::SecureZeroData::Close, details::pointer_access_all, details::SecureZeroData> secure_zero_memory_scope_exit;

    inline secure_zero_memory_scope_exit SecureZeroMemory_scope_exit(_In_reads_bytes_(sizeBytes) void* pSource, size_t sizeBytes)
    {
        return secure_zero_memory_scope_exit(details::SecureZeroData(pSource, sizeBytes));
    }

    inline secure_zero_memory_scope_exit SecureZeroMemory_scope_exit(_In_ PWSTR initializedString)
    {
        return SecureZeroMemory_scope_exit(static_cast<void*>(initializedString), wcslen(initializedString) * sizeof(initializedString[0]));
    }

    namespace details
    {
        inline void FreeProcessHeap(_Pre_opt_valid_ _Frees_ptr_opt_ void* p)
        {
            ::HeapFree(::GetProcessHeap(), 0, p);
        }
    }
    struct process_heap_deleter
    {
        template <typename T>
        void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T* p) const
        {
            details::FreeProcessHeap(p);
        }
    };

    // deprecated, use unique_process_heap_ptr instead as that has the correct name.
    template <typename T>
    using unique_hheap_ptr = wistd::unique_ptr<T, process_heap_deleter>;

    template <typename T>
    using unique_process_heap_ptr = wistd::unique_ptr<T, process_heap_deleter>;

    typedef unique_any<PWSTR, decltype(&details::FreeProcessHeap), details::FreeProcessHeap> unique_process_heap_string;

#endif // __WIL_WINBASE_

#if defined(__WIL_WINBASE_) && defined(__NOTHROW_T_DEFINED) && !defined(__WIL_WINBASE_NOTHROW_T_DEFINED) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_WINBASE_NOTHROW_T_DEFINED
    // unique_event_watcher, unique_event_watcher_nothrow, unique_event_watcher_failfast
    //
    // Clients must include <new> or <new.h> to enable use of this class as it uses new(std::nothrow).
    // This is to avoid the dependency on those headers that some clients can't tolerate.
    //
    // These classes makes it easy to execute a provided function when an event
    // is signaled. It will create the event handle for you, take ownership of one
    // or duplicate a handle provided. It supports the ability to signal the
    // event using SetEvent() and SetEvent_scope_exit();
    //
    // This can be used to support producer-consumer pattern
    // where a producer updates some state then signals the event when done.
    // The consumer will consume that state in the callback provided to unique_event_watcher.
    //
    // Note, multiple signals may coalesce into a single callback.
    //
    // Example use of throwing version:
    // auto globalStateWatcher = wil::make_event_watcher([]
    //     {
    //         currentState = GetGlobalState();
    //     });
    //
    // UpdateGlobalState(value);
    // globalStateWatcher.SetEvent(); // signal observers so they can update
    //
    // Example use of non-throwing version:
    // auto globalStateWatcher = wil::make_event_watcher_nothrow([]
    //     {
    //         currentState = GetGlobalState();
    //     });
    // RETURN_HR_IF_FALSE(E_OUTOFMEMORY, globalStateWatcher)
    //
    // UpdateGlobalState(value);
    // globalStateWatcher.SetEvent(); // signal observers so they can update

    /// @cond
    namespace details
    {
        struct event_watcher_state
        {
            event_watcher_state(unique_event_nothrow &&eventHandle, wistd::function<void()> &&callback)
                : m_event(wistd::move(eventHandle)), m_callback(wistd::move(callback))
            {
            }
            wistd::function<void()> m_callback;
            unique_event_nothrow m_event;
            // The thread pool must be last to ensure that the other members are valid
            // when it is destructed as it will reference them.
            // See http://osgvsowi/2224623
            unique_threadpool_wait m_threadPoolWait;
        };

        inline void delete_event_watcher_state(_In_opt_ event_watcher_state *watcherStorage) { delete watcherStorage;  }

        typedef resource_policy<event_watcher_state *, decltype(&delete_event_watcher_state), 
            delete_event_watcher_state, details::pointer_access_none> event_watcher_state_resource_policy;
    }
    /// @endcond

    template <typename storage_t, typename err_policy = err_exception_policy>
    class event_watcher_t : public storage_t
    {
    public:
        // forward all base class constructors...
        template <typename... args_t>
        explicit event_watcher_t(args_t&&... args) WI_NOEXCEPT : storage_t(wistd::forward<args_t>(args)...) {}

        // HRESULT or void error handling...
        typedef typename err_policy::result result;

        // Exception-based constructors
        template <typename err_policy>
        event_watcher_t(unique_any_t<event_t<details::unique_storage<details::handle_resource_policy>, err_policy>> &&eventHandle, wistd::function<void()> &&callback)
        {
            static_assert(wistd::is_same<void, result>::value, "this constructor requires exceptions or fail fast; use the create method");
            create(wistd::move(eventHandle), wistd::move(callback));
        }

        event_watcher_t(_In_ HANDLE eventHandle, wistd::function<void()> &&callback)
        {
            static_assert(wistd::is_same<void, result>::value, "this constructor requires exceptions or fail fast; use the create method");
            create(eventHandle, wistd::move(callback));
        }

        event_watcher_t(wistd::function<void()> &&callback)
        {
            static_assert(wistd::is_same<void, result>::value, "this constructor requires exceptions or fail fast; use the create method");
            create(wistd::move(callback));
        }

        template <typename event_err_policy>
        result create(unique_any_t<event_t<details::unique_storage<details::handle_resource_policy>, event_err_policy>> &&eventHandle,
            wistd::function<void()> &&callback)
        {
            return err_policy::HResult(create_take_hevent_ownership(eventHandle.release(), wistd::move(callback)));
        }

        // Creates the event that you will be watching.
        result create(wistd::function<void()> &&callback)
        {
            unique_event_nothrow eventHandle;
            HRESULT hr = eventHandle.create(EventOptions::ManualReset); // auto-reset is supported too.
            if (FAILED(hr))
            {
                return err_policy::HResult(hr);
            }
            return err_policy::HResult(create_take_hevent_ownership(eventHandle.release(), wistd::move(callback)));
        }

        // Input is an event handler that is duplicated into this class.
        result create(_In_ HANDLE eventHandle, wistd::function<void()> &&callback)
        {
            unique_event_nothrow ownedHandle;
            if (!DuplicateHandle(GetCurrentProcess(), eventHandle, GetCurrentProcess(), &ownedHandle, 0, FALSE, DUPLICATE_SAME_ACCESS))
            {
                return err_policy::LastError();
            }
            return err_policy::HResult(create_take_hevent_ownership(ownedHandle.release(), wistd::move(callback)));
        }

        // Provide access to the inner event and the very common SetEvent() method on it.
        unique_event_nothrow const& get_event() const WI_NOEXCEPT { return get()->m_event; }
        void SetEvent() const WI_NOEXCEPT { get()->m_event.SetEvent(); }

    private:
        // To avoid template expansion (if unique_event/unique_event_nothrow forms were used) this base
        // create function takes a raw handle and assumes its ownership, even on failure.
        HRESULT create_take_hevent_ownership(_In_ HANDLE rawHandleOwnershipTaken, wistd::function<void()> &&callback)
        {
            __FAIL_FAST_ASSERT__(rawHandleOwnershipTaken != nullptr); // invalid parameter
            unique_event_nothrow eventHandle(rawHandleOwnershipTaken);
            wistd::unique_ptr<details::event_watcher_state> watcherState(new(std::nothrow) details::event_watcher_state(wistd::move(eventHandle), wistd::move(callback)));
            RETURN_HR_IF_FALSE(E_OUTOFMEMORY, watcherState);

            watcherState->m_threadPoolWait.reset(CreateThreadpoolWait([](PTP_CALLBACK_INSTANCE, void *context, TP_WAIT *pThreadPoolWait, TP_WAIT_RESULT)
                {
                    auto pThis = static_cast<details::event_watcher_state *>(context);
                    // Manual events must be re-set to avoid missing the last notification.
                    pThis->m_event.ResetEvent();
                    // Call the client before re-arming to ensure that multiple callbacks don't
                    // run concurrently.
                    pThis->m_callback();
                    SetThreadpoolWait(pThreadPoolWait, pThis->m_event.get(), nullptr); // valid params ensure success
                }, watcherState.get(), nullptr));
            RETURN_LAST_ERROR_IF_FALSE(watcherState->m_threadPoolWait);
            reset(watcherState.release()); // no more failures after this, pass ownership
            SetThreadpoolWait(get()->m_threadPoolWait.get(), get()->m_event.get(), nullptr);
            return S_OK;
        }
    };

    typedef unique_any_t<event_watcher_t<details::unique_storage<details::event_watcher_state_resource_policy>, err_returncode_policy>> unique_event_watcher_nothrow;
    typedef unique_any_t<event_watcher_t<details::unique_storage<details::event_watcher_state_resource_policy>, err_failfast_policy>> unique_event_watcher_failfast;
    
    template <typename err_policy>
    unique_event_watcher_nothrow make_event_watcher_nothrow(unique_any_t<event_t<details::unique_storage<details::handle_resource_policy>, err_policy>> &&eventHandle, wistd::function<void()> &&callback) WI_NOEXCEPT
    {
        unique_event_watcher_nothrow watcher;
        watcher.create(wistd::move(eventHandle), wistd::move(callback));
        return watcher; // caller must test for success using if (watcher)
    }

    inline unique_event_watcher_nothrow make_event_watcher_nothrow(_In_ HANDLE eventHandle, wistd::function<void()> &&callback) WI_NOEXCEPT
    {
        unique_event_watcher_nothrow watcher;
        watcher.create(eventHandle, wistd::move(callback));
        return watcher; // caller must test for success using if (watcher)
    }

    inline unique_event_watcher_nothrow make_event_watcher_nothrow(wistd::function<void()> &&callback) WI_NOEXCEPT
    {
        unique_event_watcher_nothrow watcher;
        watcher.create(wistd::move(callback));
        return watcher; // caller must test for success using if (watcher)
    }

    template <typename err_policy>
    unique_event_watcher_failfast make_event_watcher_failfast(unique_any_t<event_t<details::unique_storage<details::handle_resource_policy>, err_policy>> &&eventHandle, wistd::function<void()> &&callback)
    {
        return unique_event_watcher_failfast(wistd::move(eventHandle), wistd::move(callback));
    }

    inline unique_event_watcher_failfast make_event_watcher_failfast(_In_ HANDLE eventHandle, wistd::function<void()> &&callback)
    {
        return unique_event_watcher_failfast(eventHandle, wistd::move(callback));
    }

    inline unique_event_watcher_failfast make_event_watcher_failfast(wistd::function<void()> &&callback)
    {
        return unique_event_watcher_failfast(wistd::move(callback));
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    typedef unique_any_t<event_watcher_t<details::unique_storage<details::event_watcher_state_resource_policy>, err_exception_policy>> unique_event_watcher;

    template <typename err_policy>
    unique_event_watcher make_event_watcher(unique_any_t<event_t<details::unique_storage<details::handle_resource_policy>, err_policy>> &&eventHandle, wistd::function<void()> &&callback)
    {
        return unique_event_watcher(wistd::move(eventHandle), wistd::move(callback));
    }

    inline unique_event_watcher make_event_watcher(_In_ HANDLE eventHandle, wistd::function<void()> &&callback)
    {
        return unique_event_watcher(eventHandle, wistd::move(callback));
    }

    inline unique_event_watcher make_event_watcher(wistd::function<void()> &&callback)
    {
        return unique_event_watcher(wistd::move(callback));
    }
#endif // WIL_ENABLE_EXCEPTIONS

#endif // __WIL_WINBASE_NOTHROW_T_DEFINED

#if defined(__WIL_WINBASE_) && !defined(__WIL_WINBASE_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_WINBASE_STL
    typedef shared_any_t<event_t<details::shared_storage<unique_event>>> shared_event;
    typedef shared_any_t<mutex_t<details::shared_storage<unique_mutex>>> shared_mutex;
    typedef shared_any_t<semaphore_t<details::shared_storage<unique_semaphore>>> shared_semaphore;
    typedef shared_any<unique_hfile> shared_hfile;
    typedef shared_any<unique_handle> shared_handle;
    typedef shared_any<unique_hfind> shared_hfind;
    typedef shared_any<unique_hmodule> shared_hmodule;

#if WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
    typedef shared_any<unique_threadpool_wait> shared_threadpool_wait;
    typedef shared_any<unique_threadpool_wait_nocancel> shared_threadpool_wait_nocancel;
    typedef shared_any<unique_threadpool_work> shared_threadpool_work;
    typedef shared_any<unique_threadpool_work_nocancel> shared_threadpool_work_nocancel;

    typedef shared_any<unique_hfind_change> shared_hfind_change;
#endif

    typedef weak_any<shared_event> weak_event;
    typedef weak_any<shared_mutex> weak_mutex;
    typedef weak_any<shared_semaphore> weak_semaphore;
    typedef weak_any<shared_hfile> weak_hfile;
    typedef weak_any<shared_handle> weak_handle;
    typedef weak_any<shared_hfind> weak_hfind;
    typedef weak_any<shared_hmodule> weak_hmodule;

#if WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
    typedef weak_any<shared_threadpool_wait> weak_threadpool_wait;
    typedef weak_any<shared_threadpool_wait_nocancel> weak_threadpool_wait_nocancel;
    typedef weak_any<shared_threadpool_work> weak_threadpool_work;
    typedef weak_any<shared_threadpool_work_nocancel> weak_threadpool_work_nocancel;

    typedef weak_any<shared_hfind_change> weak_hfind_change;
#endif

#endif // __WIL_WINBASE_STL

#if defined(__WIL_WINBASE_) && defined(__NOTHROW_T_DEFINED) && !defined(__WIL_WINBASE_NOTHROW_T_DEFINED_STL) && defined(WIL_RESOURCE_STL) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_WINBASE_NOTHROW_T_DEFINED_STL
    typedef shared_any_t<event_watcher_t<details::shared_storage<unique_event_watcher>>> shared_event_watcher;
    typedef weak_any<shared_event_watcher> weak_event_watcher;
#endif // __WIL_WINBASE_NOTHROW_T_DEFINED_STL

#if defined(_WINBASE_) && !defined(__WIL_WINBASE_DESKTOP) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_WINBASE_DESKTOP
    /// @cond
    namespace details
    {
        inline void DestroyPrivateObjectSecurity(_Pre_opt_valid_ _Frees_ptr_opt_ PSECURITY_DESCRIPTOR pObjectDescriptor) WI_NOEXCEPT
        {
            ::DestroyPrivateObjectSecurity(&pObjectDescriptor);
        }
    }
    /// @endcond

    struct hlocal_deleter
    {
        template <typename T>
        void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T* p) const
        {
            ::LocalFree(p);
        }
    };

    template <typename T>
    using unique_hlocal_ptr = wistd::unique_ptr<T, hlocal_deleter>;

    /** Provides `std::make_unique()` semantics for resources allocated with `LocalAlloc()` in a context that may not throw upon allocation failure.
    Use `wil::make_unique_hlocal_nothrow()` for resources returned from APIs that must satisfy a memory allocation contract that requires the use of `LocalAlloc()` / `LocalFree()`.
    Use `wil::make_unique_nothrow()` when `LocalAlloc()` is not required.

    Allocations are initialized with placement new and will call constructors (if present), but this does not guarantee initialization.

    Note that `wil::make_unique_hlocal_nothrow()` is not marked WI_NOEXCEPT as it may be used to create an exception-based class that may throw in its constructor.
    ~~~
    auto foo = wil::make_unique_hlocal_nothrow<Foo>();
    if (foo)
    {
        // initialize allocated Foo object as appropriate
    }
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_hlocal_ptr<T>>::type make_unique_hlocal_nothrow(Args&&... args)
    {
        static_assert(wistd::is_trivially_destructible<T>::value, "T has a destructor that won't be run when used with this function; use make_unique instead");
        unique_hlocal_ptr<T> sp(static_cast<T*>(::LocalAlloc(LMEM_FIXED, sizeof(T))));
        if (sp)
        {
            // use placement new to initialize memory from the previous allocation
            new (sp.get()) T(wistd::forward<Args>(args)...);
        }
        return sp;
    }

    /** Provides `std::make_unique()` semantics for array resources allocated with `LocalAlloc()` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_hlocal_nothrow<Foo[]>(size);
    if (foos)
    {
        for (auto& elem : wil::make_range(foos.get(), size))
        {
            // initialize allocated Foo objects as appropriate
        }
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_hlocal_ptr<T>>::type make_unique_hlocal_nothrow(size_t size)
    {
        typedef typename wistd::remove_extent<T>::type E;
        static_assert(wistd::is_trivially_destructible<E>::value, "E has a destructor that won't be run when used with this function; use make_unique instead");
        FAIL_FAST_IF_FALSE((SIZE_MAX / sizeof(E)) >= size);
        size_t allocSize = sizeof(E) * size;
        unique_hlocal_ptr<T> sp(static_cast<E*>(::LocalAlloc(LMEM_FIXED, allocSize)));
        if (sp)
        {
            // use placement new to initialize memory from the previous allocation;
            // note that array placement new cannot be used as the standard allows for operator new[]
            // to consume overhead in the allocation for internal bookkeeping
            for (auto& elem : make_range(static_cast<E*>(sp.get()), size))
            {
                new (&elem) E();
            }
        }
        return sp;
    }

    /** Provides `std::make_unique()` semantics for resources allocated with `LocalAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_hlocal_failfast<Foo>();
    // initialize allocated Foo object as appropriate
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_hlocal_ptr<T>>::type make_unique_hlocal_failfast(Args&&... args)
    {
        unique_hlocal_ptr<T> result(make_unique_hlocal_nothrow<T>(wistd::forward<Args>(args)...));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

    /** Provides `std::make_unique()` semantics for array resources allocated with `LocalAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_hlocal_failfast<Foo[]>(size);
    for (auto& elem : wil::make_range(foos.get(), size))
    {
        // initialize allocated Foo objects as appropriate
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_hlocal_ptr<T>>::type make_unique_hlocal_failfast(size_t size)
    {
        unique_hlocal_ptr<T> result(make_unique_hlocal_nothrow<T>(size));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    /** Provides `std::make_unique()` semantics for resources allocated with `LocalAlloc()`.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_hlocal<Foo>();
    // initialize allocated Foo object as appropriate
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_hlocal_ptr<T>>::type make_unique_hlocal(Args&&... args)
    {
        unique_hlocal_ptr<T> result(make_unique_hlocal_nothrow<T>(wistd::forward<Args>(args)...));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }

    /** Provides `std::make_unique()` semantics for array resources allocated with `LocalAlloc()`.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_hlocal<Foo[]>(size);
    for (auto& elem : wil::make_range(foos.get(), size))
    {
        // initialize allocated Foo objects as appropriate
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_hlocal_ptr<T>>::type make_unique_hlocal(size_t size)
    {
        unique_hlocal_ptr<T> result(make_unique_hlocal_nothrow<T>(size));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }
#endif // WIL_ENABLE_EXCEPTIONS

    typedef unique_any<HLOCAL, decltype(&::LocalFree), ::LocalFree> unique_hlocal;
    typedef unique_any<PWSTR, decltype(&::LocalFree), ::LocalFree> unique_hlocal_string;
    typedef unique_any<PSTR, decltype(&::LocalFree), ::LocalFree> unique_hlocal_ansistring;

    /** Copies a given string (up to the given length) into memory allocated with `LocalAlloc()` in a context that may not throw upon allocation failure.
    Use `wil::make_hlocal_string_nothrow()` for string resources returned from APIs that must satisfy a memory allocation contract that requires the use of `LocalAlloc()` / `LocalFree()`.
    Use `wil::make_unique_nothrow<wchar_t[]>()` and manually copy when `LocalAlloc()` is not required.
    ~~~
    auto str = wil::make_hlocal_string_nothrow(L"a string of words", 8);
    if (str)
    {
        std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    }
    ~~~
    */
    inline unique_hlocal_string make_hlocal_string_nothrow(_In_reads_opt_(length) PCWSTR source, size_t length) WI_NOEXCEPT
    {
        size_t const cb = (length + 1) * sizeof(*source);
        PWSTR result = static_cast<PWSTR>(::LocalAlloc(LMEM_FIXED, cb));
        if (result)
        {
            if (source)
            {
                memcpy(result, source, cb - sizeof(*source));
            }
            else
            {
                *result = L'\0'; // ensure null terminated in the "reserve space" use case.
            }
            result[length] = L'\0'; // ensure zero terminated
        }
        return unique_hlocal_string(result);
    }

    /** Copies a given string into memory allocated with `LocalAlloc()` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_hlocal_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_hlocal_string_nothrow(L"a string");
    if (str)
    {
        std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    }
    ~~~
    */
    inline unique_hlocal_string make_hlocal_string_nothrow(_In_ PCWSTR source) WI_NOEXCEPT
    {
        return make_hlocal_string_nothrow(source, wcslen(source));
    }

    /** Copies a given string into memory allocated with `LocalAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_hlocal_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_hlocal_string_failfast(L"a string");
    std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    ~~~
    */
    template <typename ...Args>
    inline unique_hlocal_string make_hlocal_string_failfast(Args&& ...args) WI_NOEXCEPT
    {
        unique_hlocal_string result(make_hlocal_string_nothrow(wistd::forward<Args>(args)...));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    /** Copies a given string into memory allocated with `LocalAlloc()`.
    See the overload of `wil::make_hlocal_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_hlocal_string(L"a string");
    std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    ~~~
    */
    template <typename ...Args>
    inline unique_hlocal_string make_hlocal_string(Args&& ...args)
    {
        unique_hlocal_string result(make_hlocal_string_nothrow(wistd::forward<Args>(args)...));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }
#endif // WIL_ENABLE_EXCEPTIONS

    struct hlocal_secure_deleter
    {
        template <typename T>
        void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T* p) const
        {
            if (p)
            {
                #pragma warning(suppress: 26006 26007) // LocalSize() ensures proper buffer length
                ::SecureZeroMemory(p, ::LocalSize(p)); // this is safe since LocalSize() returns 0 on failure
                ::LocalFree(p);
            }
        }
    };

    template <typename T>
    using unique_hlocal_secure_ptr = wistd::unique_ptr<T, hlocal_secure_deleter>;

    /** Provides `std::make_unique()` semantics for secure resources allocated with `LocalAlloc()` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_hlocal_secure_nothrow<Foo>();
    if (foo)
    {
        // initialize allocated Foo object as appropriate
    }
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_hlocal_secure_ptr<T>>::type make_unique_hlocal_secure_nothrow(Args&&... args)
    {
        return unique_hlocal_secure_ptr<T>(make_unique_hlocal_nothrow<T>(wistd::forward<Args>(args)...).release());
    }

    /** Provides `std::make_unique()` semantics for secure array resources allocated with `LocalAlloc()` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_hlocal_secure_nothrow<Foo[]>(size);
    if (foos)
    {
        for (auto& elem : wil::make_range(foos.get(), size))
        {
            // initialize allocated Foo objects as appropriate
        }
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_hlocal_secure_ptr<T>>::type make_unique_hlocal_secure_nothrow(size_t size)
    {
        return unique_hlocal_secure_ptr<T>(make_unique_hlocal_nothrow<T>(size).release());
    }

    /** Provides `std::make_unique()` semantics for secure resources allocated with `LocalAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_hlocal_secure_failfast<Foo>();
    // initialize allocated Foo object as appropriate
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_hlocal_secure_ptr<T>>::type make_unique_hlocal_secure_failfast(Args&&... args)
    {
        unique_hlocal_secure_ptr<T> result(make_unique_hlocal_secure_nothrow<T>(wistd::forward<Args>(args)...));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

    /** Provides `std::make_unique()` semantics for secure array resources allocated with `LocalAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_hlocal_secure_failfast<Foo[]>(size);
    for (auto& elem : wil::make_range(foos.get(), size))
    {
        // initialize allocated Foo objects as appropriate
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_hlocal_secure_ptr<T>>::type make_unique_hlocal_secure_failfast(size_t size)
    {
        unique_hlocal_secure_ptr<T> result(make_unique_hlocal_secure_nothrow<T>(size));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    /** Provides `std::make_unique()` semantics for secure resources allocated with `LocalAlloc()`.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_hlocal_secure<Foo>();
    // initialize allocated Foo object as appropriate
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_hlocal_secure_ptr<T>>::type make_unique_hlocal_secure(Args&&... args)
    {
        unique_hlocal_secure_ptr<T> result(make_unique_hlocal_secure_nothrow<T>(wistd::forward<Args>(args)...));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }

    /** Provides `std::make_unique()` semantics for secure array resources allocated with `LocalAlloc()`.
    See the overload of `wil::make_unique_hlocal_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_hlocal_secure<Foo[]>(size);
    for (auto& elem : wil::make_range(foos.get(), size))
    {
        // initialize allocated Foo objects as appropriate
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_hlocal_secure_ptr<T>>::type make_unique_hlocal_secure(size_t size)
    {
        unique_hlocal_secure_ptr<T> result(make_unique_hlocal_secure_nothrow<T>(size));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }
#endif // WIL_ENABLE_EXCEPTIONS

    typedef unique_hlocal_secure_ptr<wchar_t []> unique_hlocal_string_secure;

    /** Copies a given string into secure memory allocated with `LocalAlloc()` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_hlocal_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_hlocal_string_secure_nothrow(L"a string");
    if (str)
    {
        std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    }
    ~~~
    */
    inline unique_hlocal_string_secure make_hlocal_string_secure_nothrow(_In_ PCWSTR source) WI_NOEXCEPT
    {
        return unique_hlocal_string_secure(make_hlocal_string_nothrow(source).release());
    }

    /** Copies a given string into secure memory allocated with `LocalAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_hlocal_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_hlocal_string_secure_failfast(L"a string");
    std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    ~~~
    */
    inline unique_hlocal_string_secure make_hlocal_string_secure_failfast(_In_ PCWSTR source) WI_NOEXCEPT
    {
        unique_hlocal_string_secure result(make_hlocal_string_secure_nothrow(source));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    /** Copies a given string into secure memory allocated with `LocalAlloc()`.
    See the overload of `wil::make_hlocal_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_hlocal_string_secure(L"a string");
    std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    ~~~
    */
    inline unique_hlocal_string_secure make_hlocal_string_secure(_In_ PCWSTR source)
    {
        unique_hlocal_string_secure result(make_hlocal_string_secure_nothrow(source));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }
#endif

    struct hglobal_deleter
    {
        template <typename T>
        void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T* p) const
        {
            ::GlobalFree(p);
        }
    };
    
    template <typename T>
    using unique_hglobal_ptr = wistd::unique_ptr<T, hglobal_deleter>;

    typedef unique_any<HGLOBAL, decltype(&::GlobalFree), ::GlobalFree> unique_hglobal;
    typedef unique_any<PWSTR, decltype(&::GlobalFree), ::GlobalFree> unique_hglobal_string;
    typedef unique_any<PSTR, decltype(&::GlobalFree), ::GlobalFree> unique_hglobal_ansistring;

    /** Copies a given string (up to the given length) into memory allocated with `HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY)` in a context that may not throw upon allocation failure.
    Use `wil::make_process_heap_string_nothrow()` for string resources returned from APIs that must satisfy a memory allocation contract that requires the use of `HeapAlloc()` / `HeapFree()`.
    Use `wil::make_unique_nothrow<wchar_t[]>()` and manually copy when `HeapAlloc()` is not required.
    ~~~
    auto str = wil::make_process_heap_string_nothrow(L"a string of words", 8);
    if (str)
    {
        std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    }
    ~~~
    */
    inline unique_process_heap_string make_process_heap_string_nothrow(_In_reads_opt_(length) PCWSTR source, size_t length) WI_NOEXCEPT
    {
        size_t const cb = (length + 1) * sizeof(*source);
        PWSTR result = static_cast<PWSTR>(::HeapAlloc(::GetProcessHeap(), HEAP_ZERO_MEMORY, cb));
        if (result)
        {
            if (source)
            {
                memcpy(result, source, cb - sizeof(*source));
            }
            else
            {
                *result = L'\0'; // ensure null terminated in the "reserve space" use case.
            }
            result[length] = L'\0'; // ensure zero terminated
        }
        return unique_process_heap_string(result);
    }

    /** Copies a given string into memory allocated with `HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY)` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_process_heap_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_process_heap_string_nothrow(L"a string");
    if (str)
    {
        std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    }
    ~~~
    */
    inline unique_process_heap_string make_process_heap_string_nothrow(_In_ PCWSTR source) WI_NOEXCEPT
    {
        return make_process_heap_string_nothrow(source, wcslen(source));
    }

    /** Copies a given string into memory allocated with `HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY)` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_process_heap_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_process_heap_string_failfast(L"a string");
    std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    ~~~
    */
    template <typename ...Args>
    inline unique_process_heap_string make_process_heap_string_failfast(Args&& ...args) WI_NOEXCEPT
    {
        unique_process_heap_string result(make_process_heap_string_nothrow(wistd::forward<Args>(args)...));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    /** Copies a given string into memory allocated with `HeapAlloc()`.
    See the overload of `wil::make_process_heap_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_process_heap_string(L"a string");
    std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    ~~~
    */
    template <typename ...Args>
    inline unique_process_heap_string make_process_heap_string(Args&& ...args)
    {
        unique_process_heap_string result(make_process_heap_string_nothrow(wistd::forward<Args>(args)...));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }
#endif // WIL_ENABLE_EXCEPTIONS

    typedef unique_any_handle_null<decltype(&::HeapDestroy), ::HeapDestroy> unique_hheap;
    typedef unique_any<DWORD, decltype(&::TlsFree), ::TlsFree> unique_tls;
    typedef unique_any<PSECURITY_DESCRIPTOR, decltype(&::LocalFree), ::LocalFree> unique_hlocal_security_descriptor;
    typedef unique_any<PSECURITY_DESCRIPTOR, decltype(&details::DestroyPrivateObjectSecurity), details::DestroyPrivateObjectSecurity> unique_private_security_descriptor;
    typedef unique_any<HACCEL, decltype(&::DestroyAcceleratorTable), ::DestroyAcceleratorTable> unique_haccel;
    typedef unique_any<HCURSOR, decltype(&::DestroyCursor), ::DestroyCursor> unique_hcursor;
#if !defined(NOGDI)
    typedef unique_any<HDESK, decltype(&::CloseDesktop), ::CloseDesktop> unique_hdesk;
    typedef unique_any<HWINSTA, decltype(&::CloseWindowStation), ::CloseWindowStation> unique_hwinsta;
#endif // !defined(NOGDI)
    typedef unique_any<HWND, decltype(&::DestroyWindow), ::DestroyWindow> unique_hwnd;

    // WARNING: 'unique_security_descriptor' has been deprecated and is pending deletion!
    // Use:
    // 1. 'unique_hlocal_security_descriptor' if you need to free using 'LocalFree'.
    // 2. 'unique_private_security_descriptor' if you need to free using 'DestroyPrivateObjectSecurity'.
    typedef unique_private_security_descriptor unique_security_descriptor;

#endif
#if defined(__WIL_WINBASE_DESKTOP) && !defined(__WIL_WINBASE_DESKTOP_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_WINBASE_DESKTOP_STL
    typedef shared_any<unique_hheap> shared_hheap;
    typedef shared_any<unique_hlocal> shared_hlocal;
    typedef shared_any<unique_tls> shared_tls;
    typedef shared_any<unique_hlocal_security_descriptor> shared_hlocal_security_descriptor;
    typedef shared_any<unique_private_security_descriptor> shared_private_security_descriptor;
    typedef shared_any<unique_haccel> shared_haccel;
    typedef shared_any<unique_hcursor> shared_hcursor;
#if !defined(NOGDI)
    typedef shared_any<unique_hdesk> shared_hdesk;
    typedef shared_any<unique_hwinsta> shared_hwinsta;
#endif // !defined(NOGDI)
    typedef shared_any<unique_hwnd> shared_hwnd;

    typedef weak_any<shared_hheap> weak_hheap;
    typedef weak_any<shared_hlocal> weak_hlocal;
    typedef weak_any<shared_tls> weak_tls;
    typedef weak_any<shared_hlocal_security_descriptor> weak_hlocal_security_descriptor;
    typedef weak_any<shared_private_security_descriptor> weak_private_security_descriptor;
    typedef weak_any<shared_haccel> weak_haccel;
    typedef weak_any<shared_hcursor> weak_hcursor;
#if !defined(NOGDI)
    typedef weak_any<shared_hdesk> weak_hdesk;
    typedef weak_any<shared_hwinsta> weak_hwinsta;
#endif // !defined(NOGDI)
    typedef weak_any<shared_hwnd> weak_hwnd;
#endif // __WIL_WINBASE_DESKTOP_STL

#if defined(_COMBASEAPI_H_) && !defined(__WIL__COMBASEAPI_H_) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP) && (NTDDI_VERSION >= NTDDI_WIN8)
#define __WIL__COMBASEAPI_H_
    typedef unique_any<CO_MTA_USAGE_COOKIE, decltype(&::CoDecrementMTAUsage), ::CoDecrementMTAUsage> unique_mta_usage_cookie;
#endif // __WIL__COMBASEAPI_H_
#if defined(__WIL__COMBASEAPI_H_) && !defined(__WIL__COMBASEAPI_H__STL) && defined(WIL_RESOURCE_STL) && (NTDDI_VERSION >= NTDDI_WIN8)
#define __WIL__COMBASEAPI_H__STL
    typedef shared_any<unique_mta_usage_cookie> shared_mta_usage_cookie;
    typedef weak_any<shared_mta_usage_cookie> weak_mta_usage_cookie;
#endif // __WIL__COMBASEAPI_H__STL

#if defined(__WINSTRING_H_) && !defined(__WIL__WINSTRING_H_)
#define __WIL__WINSTRING_H_
    typedef unique_any<HSTRING, decltype(&::WindowsDeleteString), ::WindowsDeleteString> unique_hstring;
#endif // __WIL__WINSTRING_H_
#if defined(__WIL__WINSTRING_H_) && !defined(__WIL__WINSTRING_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL__WINSTRING_H_STL
    typedef shared_any<unique_hstring> shared_hstring;
    typedef weak_any<shared_hstring> weak_hstring;
#endif // __WIL__WINSTRING_H_STL


#if defined(_WINREG_) && !defined(__WIL_WINREG_) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_WINREG_
    typedef unique_any<HKEY, decltype(&::RegCloseKey), ::RegCloseKey> unique_hkey;
#endif // __WIL_WINREG_
#if defined(__WIL_WINREG_) && !defined(__WIL_WINREG_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_WINREG_STL
    typedef shared_any<unique_hkey> shared_hkey;
    typedef weak_any<shared_hkey> weak_hkey;
#endif // __WIL_WINREG_STL


#if defined(__MSGQUEUE_H__) && !defined(__WIL__MSGQUEUE_H__)
#define __WIL__MSGQUEUE_H__
    typedef unique_any<details::handle_invalid_resource_policy<decltype(&::CloseMsgQueue), ::CloseMsgQueue> unique_msg_queue;
#endif // __WIL__MSGQUEUE_H__
#if defined(__WIL__MSGQUEUE_H__) && !defined(__WIL__MSGQUEUE_H__STL) && defined(WIL_RESOURCE_STL)
#define __WIL__MSGQUEUE_H__STL
    typedef shared_any<unique_msg_queue> shared_msg_queue;
    typedef weak_any<shared_msg_queue> weak_msg_queue;
#endif // __WIL__MSGQUEUE_H__STL


#if defined(_OLEAUTO_H_) && !defined(__WIL_OLEAUTO_H_)
#define __WIL_OLEAUTO_H_
    typedef unique_any<BSTR, decltype(&::SysFreeString), ::SysFreeString> unique_bstr;

    inline wil::unique_bstr make_bstr_nothrow(PCWSTR source) WI_NOEXCEPT
    {
        return wil::unique_bstr(::SysAllocString(source));
    }

    inline wil::unique_bstr make_bstr_failfast(PCWSTR source) WI_NOEXCEPT
    {
        return wil::unique_bstr(FAIL_FAST_IF_NULL_ALLOC(::SysAllocString(source)));
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    inline wil::unique_bstr make_bstr(PCWSTR source)
    {
        wil::unique_bstr result(make_bstr_nothrow(source));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }
#endif // WIL_ENABLE_EXCEPTIONS

#endif // __WIL_OLEAUTO_H_
#if defined(__WIL_OLEAUTO_H_) && !defined(__WIL_OLEAUTO_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_OLEAUTO_H_STL
    typedef shared_any<unique_bstr> shared_bstr;
    typedef weak_any<shared_bstr> weak_bstr;
#endif // __WIL_OLEAUTO_H_STL


#if (defined(_WININET_) || defined(_DUBINET_)) && !defined(__WIL_WININET_)
#define __WIL_WININET_
    typedef unique_any<HINTERNET, decltype(&::InternetCloseHandle), ::InternetCloseHandle> unique_hinternet;
#endif // __WIL_WININET_
#if defined(__WIL_WININET_) && !defined(__WIL_WININET_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_WININET_STL
    typedef shared_any<unique_hinternet> shared_hinternet;
    typedef weak_any<shared_hinternet> weak_hinternet;
#endif // __WIL_WININET_STL


#if defined(_WINHTTPX_) && !defined(__WIL_WINHTTP_)
#define __WIL_WINHTTP_
    typedef unique_any<HINTERNET, decltype(&::WinHttpCloseHandle), ::WinHttpCloseHandle> unique_winhttp_hinternet;
#endif // __WIL_WINHTTP_
#if defined(__WIL_WINHTTP_) && !defined(__WIL_WINHTTP_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_WINHTTP_STL
    typedef shared_any<unique_winhttp_hinternet> shared_winhttp_hinternet;
    typedef weak_any<shared_winhttp_hinternet> weak_winhttp_hinternet;
#endif // __WIL_WINHTTP_STL


#if defined(_WINSOCKAPI_) && !defined(__WIL_WINSOCKAPI_) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_WINSOCKAPI_
    typedef unique_any<SOCKET, int (WINAPI*)(SOCKET), ::closesocket, details::pointer_access_all, SOCKET, INVALID_SOCKET, SOCKET> unique_socket;
#endif // __WIL_WINSOCKAPI_
#if defined(__WIL_WINSOCKAPI_) && !defined(__WIL_WINSOCKAPI_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_WINSOCKAPI_STL
    typedef shared_any<unique_socket> shared_socket;
    typedef weak_any<shared_socket> weak_socket;
#endif // __WIL_WINSOCKAPI_STL


#if defined(_WINGDI_) && !defined(__WIL_WINGDI_) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP) && !defined(NOGDI)
#define __WIL_WINGDI_
    struct WindowDC
    {
        HDC dc;
        HWND hwnd;
        WindowDC(HDC dc_, HWND hwnd_ = nullptr) WI_NOEXCEPT { dc = dc_; hwnd = hwnd_; }
        operator HDC() const WI_NOEXCEPT { return dc; }
        static void Close(WindowDC wdc) WI_NOEXCEPT { ::ReleaseDC(wdc.hwnd, wdc.dc); }
    };
    typedef unique_any<HDC, decltype(&WindowDC::Close), WindowDC::Close, details::pointer_access_all, WindowDC> unique_hdc_window;

    struct PaintDC
    {
        HWND hwnd;
        PAINTSTRUCT ps;
        PaintDC(HDC hdc = nullptr) { ::ZeroMemory(this, sizeof(*this)); ps.hdc = hdc; }
        operator HDC() const WI_NOEXCEPT { return ps.hdc; }
        static void Close(PaintDC pdc) WI_NOEXCEPT { ::EndPaint(pdc.hwnd, &pdc.ps); }
    };
    typedef unique_any<HDC, decltype(&PaintDC::Close), PaintDC::Close, details::pointer_access_all, PaintDC> unique_hdc_paint;

    struct SelectResult
    {
        HGDIOBJ hgdi;
        HDC hdc;
        SelectResult(HGDIOBJ hgdi_, HDC hdc_ = nullptr) WI_NOEXCEPT{ hgdi = hgdi_; hdc = hdc_; }
        operator HGDIOBJ() const WI_NOEXCEPT { return hgdi; }
        static void Close(SelectResult sr) WI_NOEXCEPT { ::SelectObject(sr.hdc, sr.hgdi); }
    };
    typedef unique_any<HGDIOBJ, decltype(&SelectResult::Close), SelectResult::Close, details::pointer_access_all, SelectResult> unique_select_object;

    inline unique_hdc_window GetDC(HWND hwnd) WI_NOEXCEPT
    {
        return unique_hdc_window(WindowDC(::GetDC(hwnd), hwnd));
    }

    inline unique_hdc_window GetWindowDC(HWND hwnd) WI_NOEXCEPT
    {
        return unique_hdc_window(WindowDC(::GetWindowDC(hwnd), hwnd));
    }

    inline unique_hdc_paint BeginPaint(HWND hwnd, _Out_opt_ PPAINTSTRUCT pPaintStruct = nullptr) WI_NOEXCEPT
    {
        PaintDC pdc;
        HDC hdc = ::BeginPaint(hwnd, &pdc.ps);
        wil::AssignToOptParam(pPaintStruct, pdc.ps);
        return (hdc == nullptr) ? unique_hdc_paint() : unique_hdc_paint(pdc);
    }

    inline unique_select_object SelectObject(HDC hdc, HGDIOBJ gdiobj) WI_NOEXCEPT
    {
        return unique_select_object(SelectResult(::SelectObject(hdc, gdiobj), hdc));
    }

    typedef unique_any<HGDIOBJ, decltype(&::DeleteObject), ::DeleteObject> unique_hgdiobj;
    typedef unique_any<HPEN, decltype(&::DeleteObject), ::DeleteObject> unique_hpen;
    typedef unique_any<HBRUSH, decltype(&::DeleteObject), ::DeleteObject> unique_hbrush;
    typedef unique_any<HFONT, decltype(&::DeleteObject), ::DeleteObject> unique_hfont;
    typedef unique_any<HBITMAP, decltype(&::DeleteObject), ::DeleteObject> unique_hbitmap;
    typedef unique_any<HRGN, decltype(&::DeleteObject), ::DeleteObject> unique_hrgn;
    typedef unique_any<HPALETTE, decltype(&::DeleteObject), ::DeleteObject> unique_hpalette;
    typedef unique_any<HDC, decltype(&::DeleteDC), ::DeleteDC> unique_hdc;
    typedef unique_any<HICON, decltype(&::DestroyIcon), ::DestroyIcon> unique_hicon;
    typedef unique_any<HMENU, decltype(&::DestroyMenu), ::DestroyMenu> unique_hmenu;
#endif // __WIL_WINGDI_
#if defined(__WIL_WINGDI_) && !defined(__WIL_WINGDI_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_WINGDI_STL
    typedef shared_any<unique_hgdiobj> shared_hgdiobj;
    typedef shared_any<unique_hpen> shared_hpen;
    typedef shared_any<unique_hbrush> shared_hbrush;
    typedef shared_any<unique_hfont> shared_hfont;
    typedef shared_any<unique_hbitmap> shared_hbitmap;
    typedef shared_any<unique_hrgn> shared_hrgn;
    typedef shared_any<unique_hpalette> shared_hpalette;
    typedef shared_any<unique_hdc> shared_hdc;
    typedef shared_any<unique_hicon> shared_hicon;
    typedef shared_any<unique_hmenu> shared_hmenu;

    typedef weak_any<shared_hgdiobj> weak_hgdiobj;
    typedef weak_any<shared_hpen> weak_hpen;
    typedef weak_any<shared_hbrush> weak_hbrush;
    typedef weak_any<shared_hfont> weak_hfont;
    typedef weak_any<shared_hbitmap> weak_hbitmap;
    typedef weak_any<shared_hrgn> weak_hrgn;
    typedef weak_any<shared_hpalette> weak_hpalette;
    typedef weak_any<shared_hdc> weak_hdc;
    typedef weak_any<shared_hicon> weak_hicon;
    typedef weak_any<shared_hmenu> weak_hmenu;
#endif // __WIL_WINGDI_STL


#if defined(_WINSCARD_H_) && !defined(__WIL_WINSCARD_H_) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_WINSCARD_H_
    typedef unique_any<SCARDCONTEXT, decltype(&::SCardReleaseContext), ::SCardReleaseContext> unique_scardctx;
#endif // __WIL_WINSCARD_H_
#if defined(__WIL_WINSCARD_H_) && !defined(__WIL_WINSCARD_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_WINSCARD_H_STL
    typedef shared_any<unique_scardctx> shared_scardctx;
    typedef weak_any<shared_scardctx> weak_scardctx;
#endif // __WIL_WINSCARD_H_STL


#if defined(__WINCRYPT_H__) && !defined(__WIL__WINCRYPT_H__) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL__WINCRYPT_H__
    /// @cond
    namespace details
    {
        inline void CertCloseStoreNoParam(_Pre_opt_valid_ _Frees_ptr_opt_ HCERTSTORE hCertStore) WI_NOEXCEPT
        {
            ::CertCloseStore(hCertStore, 0);
        }

        inline void CryptReleaseContextNoParam(_Pre_opt_valid_ _Frees_ptr_opt_ HCRYPTPROV hCryptCtx) WI_NOEXCEPT
        {
            ::CryptReleaseContext(hCryptCtx, 0);
        }
    }
    /// @endcond

    typedef unique_any<PCCERT_CONTEXT, decltype(&::CertFreeCertificateContext), ::CertFreeCertificateContext> unique_cert_context;
    typedef unique_any<PCCERT_CHAIN_CONTEXT, decltype(&::CertFreeCertificateChain), ::CertFreeCertificateChain> unique_cert_chain_context;
    typedef unique_any<HCERTSTORE, decltype(&details::CertCloseStoreNoParam), details::CertCloseStoreNoParam> unique_hcertstore;
    typedef unique_any<HCRYPTPROV, decltype(&details::CryptReleaseContextNoParam), details::CryptReleaseContextNoParam> unique_hcryptprov;
    typedef unique_any<HCRYPTKEY, decltype(&::CryptDestroyKey), ::CryptDestroyKey> unique_hcryptkey;
    typedef unique_any<HCRYPTHASH, decltype(&::CryptDestroyHash), ::CryptDestroyHash> unique_hcrypthash;
#endif // __WIL__WINCRYPT_H__
#if defined(__WIL__WINCRYPT_H__) && !defined(__WIL__WINCRYPT_H__STL) && defined(WIL_RESOURCE_STL)
#define __WIL__WINCRYPT_H__STL
    typedef shared_any<unique_cert_context> shared_cert_context;
    typedef shared_any<unique_cert_chain_context> shared_cert_chain_context;
    typedef shared_any<unique_hcertstore> shared_hcertstore;
    typedef shared_any<unique_hcryptprov> shared_hcryptprov;
    typedef shared_any<unique_hcryptkey> shared_hcryptkey;
    typedef shared_any<unique_hcrypthash> shared_hcrypthash;

    typedef weak_any<shared_cert_context> weak_cert_context;
    typedef weak_any<shared_cert_chain_context> weak_cert_chain_context;
    typedef weak_any<shared_hcertstore> weak_hcertstore;
    typedef weak_any<shared_hcryptprov> weak_hcryptprov;
    typedef weak_any<shared_hcryptkey> weak_hcryptkey;
    typedef weak_any<shared_hcrypthash> weak_hcrypthash;
#endif // __WIL__WINCRYPT_H__STL


#if defined(__NCRYPT_H__) && !defined(__WIL_NCRYPT_H__) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_NCRYPT_H__
    struct ncrypt_deleter
    {
        template <typename T>
        void operator()(_Pre_opt_valid_ _Frees_ptr_ T* p) const
        {
            ::NCryptFreeBuffer(p);
        }
    };

    template <typename T>
    using unique_ncrypt_ptr = wistd::unique_ptr<T, ncrypt_deleter>;

    typedef unique_any<NCRYPT_PROV_HANDLE, decltype(&::NCryptFreeObject), ::NCryptFreeObject> unique_ncrypt_prov;
    typedef unique_any<NCRYPT_KEY_HANDLE, decltype(&::NCryptFreeObject), ::NCryptFreeObject> unique_ncrypt_key;
    typedef unique_any<NCRYPT_SECRET_HANDLE, decltype(&::NCryptFreeObject), ::NCryptFreeObject> unique_ncrypt_secret;
#endif // __WIL_NCRYPT_H__
#if defined(__WIL_NCRYPT_H__) && !defined(__WIL_NCRYPT_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_NCRYPT_H_STL
    typedef shared_any<unique_ncrypt_prov> shared_ncrypt_prov;
    typedef shared_any<unique_ncrypt_key> shared_ncrypt_key;
    typedef shared_any<unique_ncrypt_secret> shared_ncrypt_secret;

    typedef weak_any<shared_ncrypt_prov> weak_ncrypt_prov;
    typedef weak_any<shared_ncrypt_key> weak_ncrypt_key;
    typedef weak_any<shared_ncrypt_secret> weak_ncrypt_secret;
#endif // __WIL_NCRYPT_H_STL


#if defined(__BCRYPT_H__) && !defined(__WIL_BCRYPT_H__) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_BCRYPT_H__
    /// @cond
    namespace details
    {
        inline void BCryptCloseAlgorithmProviderNoFlags(_Pre_opt_valid_ _Frees_ptr_opt_ BCRYPT_ALG_HANDLE hAlgorithm) WI_NOEXCEPT
        {
            ::BCryptCloseAlgorithmProvider(hAlgorithm, 0);
        }
    }
    /// @endcond

    struct bcrypt_deleter
    {
        template <typename T>
        void operator()(_Pre_opt_valid_ _Frees_ptr_ T* p) const
        {
            ::BCryptFreeBuffer(p);
        }
    };

    template <typename T>
    using unique_bcrypt_ptr = wistd::unique_ptr<T, bcrypt_deleter>;

    typedef unique_any<BCRYPT_ALG_HANDLE, decltype(&details::BCryptCloseAlgorithmProviderNoFlags), details::BCryptCloseAlgorithmProviderNoFlags> unique_bcrypt_algorithm;
    typedef unique_any<BCRYPT_HASH_HANDLE, decltype(&::BCryptDestroyHash), ::BCryptDestroyHash> unique_bcrypt_hash;
    typedef unique_any<BCRYPT_KEY_HANDLE, decltype(&::BCryptDestroyKey), ::BCryptDestroyKey> unique_bcrypt_key;
    typedef unique_any<BCRYPT_SECRET_HANDLE, decltype(&::BCryptDestroySecret), ::BCryptDestroySecret> unique_bcrypt_secret;
#endif // __WIL_BCRYPT_H__
#if defined(__WIL_BCRYPT_H__) && !defined(__WIL_BCRYPT_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_BCRYPT_H_STL
    typedef shared_any<unique_bcrypt_algorithm> shared_bcrypt_algorithm;
    typedef shared_any<unique_bcrypt_hash> shared_bcrypt_hash;
    typedef shared_any<unique_bcrypt_key> shared_bcrypt_key;
    typedef shared_any<unique_bcrypt_secret> shared_bcrypt_secret;

    typedef weak_any<shared_bcrypt_algorithm> weak_bcrypt_algorithm;
    typedef weak_any<shared_bcrypt_hash> weak_bcrypt_hash;
    typedef weak_any<unique_bcrypt_key> weak_bcrypt_key;
    typedef weak_any<shared_bcrypt_secret> weak_bcrypt_secret;
#endif // __WIL_BCRYPT_H_STL


#if defined(_OBJBASE_H_) && !defined(__WIL_OBJBASE_H_)
#define __WIL_OBJBASE_H_
    struct cotaskmem_deleter
    {
        template <typename T>
        void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T* p) const
        {
            ::CoTaskMemFree(p);
        }
    };

    template <typename T>
    using unique_cotaskmem_ptr = wistd::unique_ptr<T, cotaskmem_deleter>;

    template <typename T>
    using unique_cotaskmem_array_ptr = unique_array_ptr<T, cotaskmem_deleter>;

    /** Provides `std::make_unique()` semantics for resources allocated with `CoTaskMemAlloc()` in a context that may not throw upon allocation failure.
    Use `wil::make_unique_cotaskmem_nothrow()` for resources returned from APIs that must satisfy a memory allocation contract that requires the use of `CoTaskMemAlloc()` / `CoTaskMemFree()`.
    Use `wil::make_unique_nothrow()` when `CoTaskMemAlloc()` is not required.

    Allocations are initialized with placement new and will call constructors (if present), but this does not guarantee initialization.

    Note that `wil::make_unique_cotaskmem_nothrow()` is not marked WI_NOEXCEPT as it may be used to create an exception-based class that may throw in its constructor.
    ~~~
    auto foo = wil::make_unique_cotaskmem_nothrow<Foo>();
    if (foo)
    {
        // initialize allocated Foo object as appropriate
    }
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_cotaskmem_ptr<T>>::type make_unique_cotaskmem_nothrow(Args&&... args)
    {
        static_assert(wistd::is_trivially_destructible<T>::value, "T has a destructor that won't be run when used with this function; use make_unique instead");
        unique_cotaskmem_ptr<T> sp(static_cast<T*>(::CoTaskMemAlloc(sizeof(T))));
        if (sp)
        {
            // use placement new to initialize memory from the previous allocation
            new (sp.get()) T(wistd::forward<Args>(args)...);
        }
        return sp;
    }

    /** Provides `std::make_unique()` semantics for array resources allocated with `CoTaskMemAlloc()` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_cotaskmem_nothrow<Foo[]>(size);
    if (foos)
    {
        for (auto& elem : wil::make_range(foos.get(), size))
        {
            // initialize allocated Foo objects as appropriate
        }
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_cotaskmem_ptr<T>>::type make_unique_cotaskmem_nothrow(size_t size)
    {
        typedef typename wistd::remove_extent<T>::type E;
        static_assert(wistd::is_trivially_destructible<E>::value, "E has a destructor that won't be run when used with this function; use make_unique instead");
        FAIL_FAST_IF_FALSE((SIZE_MAX / sizeof(E)) >= size);
        size_t allocSize = sizeof(E) * size;
        unique_cotaskmem_ptr<T> sp(static_cast<E*>(::CoTaskMemAlloc(allocSize)));
        if (sp)
        {
            // use placement new to initialize memory from the previous allocation;
            // note that array placement new cannot be used as the standard allows for operator new[]
            // to consume overhead in the allocation for internal bookkeeping
            for (auto& elem : make_range(static_cast<E*>(sp.get()), size))
            {
                new (&elem) E();
            }
        }
        return sp;
    }

    /** Provides `std::make_unique()` semantics for resources allocated with `CoTaskMemAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_cotaskmem_failfast<Foo>();
    // initialize allocated Foo object as appropriate
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_cotaskmem_ptr<T>>::type make_unique_cotaskmem_failfast(Args&&... args)
    {
        unique_cotaskmem_ptr<T> result(make_unique_cotaskmem_nothrow<T>(wistd::forward<Args>(args)...));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

    /** Provides `std::make_unique()` semantics for array resources allocated with `CoTaskMemAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_cotaskmem_failfast<Foo[]>(size);
    for (auto& elem : wil::make_range(foos.get(), size))
    {
        // initialize allocated Foo objects as appropriate
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_cotaskmem_ptr<T>>::type make_unique_cotaskmem_failfast(size_t size)
    {
        unique_cotaskmem_ptr<T> result(make_unique_cotaskmem_nothrow<T>(size));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    /** Provides `std::make_unique()` semantics for resources allocated with `CoTaskMemAlloc()`.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_cotaskmem<Foo>();
    // initialize allocated Foo object as appropriate
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_cotaskmem_ptr<T>>::type make_unique_cotaskmem(Args&&... args)
    {
        unique_cotaskmem_ptr<T> result(make_unique_cotaskmem_nothrow<T>(wistd::forward<Args>(args)...));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }

    /** Provides `std::make_unique()` semantics for array resources allocated with `CoTaskMemAlloc()`.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_cotaskmem<Foo[]>(size);
    for (auto& elem : wil::make_range(foos.get(), size))
    {
        // initialize allocated Foo objects as appropriate
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_cotaskmem_ptr<T>>::type make_unique_cotaskmem(size_t size)
    {
        unique_cotaskmem_ptr<T> result(make_unique_cotaskmem_nothrow<T>(size));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }
#endif // WIL_ENABLE_EXCEPTIONS

    typedef unique_any<void*, decltype(&::CoTaskMemFree), ::CoTaskMemFree> unique_cotaskmem;
    typedef unique_any<PWSTR, decltype(&::CoTaskMemFree), ::CoTaskMemFree> unique_cotaskmem_string;
    typedef unique_any<PSTR, decltype(&::CoTaskMemFree), ::CoTaskMemFree> unique_cotaskmem_ansistring;

    /** Copies a given string (up to the given length) into memory allocated with `CoTaskMemAlloc()` in a context that may not throw upon allocation failure.
    Use `wil::make_cotaskmem_string_nothrow()` for string resources returned from APIs that must satisfy a memory allocation contract that requires the use of `CoTaskMemAlloc()` / `CoTaskMemFree()`.
    Use `wil::make_unique_nothrow<wchar_t[]>()` and manually copy when `CoTaskMemAlloc()` is not required.
    ~~~
    auto str = wil::make_cotaskmem_string_nothrow(L"a string of words", 8);
    if (str)
    {
        std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    }
    ~~~
    */
    inline unique_cotaskmem_string make_cotaskmem_string_nothrow(_In_reads_opt_(length) PCWSTR source, size_t length) WI_NOEXCEPT
    {
        size_t const cb = (length + 1) * sizeof(*source);
        PWSTR result = static_cast<PWSTR>(::CoTaskMemAlloc(cb));
        if (result)
        {
            if (source)
            {
                memcpy_s(result, cb, source, cb - sizeof(*source));
            }
            else
            {
                *result = L'\0'; // ensure null terminated in the "reserve space" use case.
            }
            result[length] = L'\0'; // ensure zero terminated
        }
        return unique_cotaskmem_string(result);
    }

    /** Copies a given string into memory allocated with `CoTaskMemAlloc()` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_cotaskmem_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_cotaskmem_string_nothrow(L"a string");
    if (str)
    {
        std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    }
    ~~~
    */
    inline unique_cotaskmem_string make_cotaskmem_string_nothrow(_In_ PCWSTR source) WI_NOEXCEPT
    {
        return make_cotaskmem_string_nothrow(source, wcslen(source));
    }

    /** Copies a given string into memory allocated with `CoTaskMemAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_cotaskmem_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_cotaskmem_string_failfast(L"a string");
    std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    ~~~
    */
    template <typename ...Args>
    inline unique_cotaskmem_string make_cotaskmem_string_failfast(Args&& ...args) WI_NOEXCEPT
    {
        unique_cotaskmem_string result(make_cotaskmem_string_nothrow(wistd::forward<Args>(args)...));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    /** Copies a given string into memory allocated with `CoTaskMemAlloc()`.
    See the overload of `wil::make_cotaskmem_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_cotaskmem_string(L"a string");
    std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    ~~~
    */
    template <typename ...Args>
    inline unique_cotaskmem_string make_cotaskmem_string(Args&& ...args)
    {
        unique_cotaskmem_string result(make_cotaskmem_string_nothrow(wistd::forward<Args>(args)...));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }

#endif // WIL_ENABLE_EXCEPTIONS
#endif // __WIL_OBJBASE_H_
#if defined(__WIL_OBJBASE_H_) && !defined(__WIL_OBJBASE_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_OBJBASE_H_STL
    typedef shared_any<unique_cotaskmem> shared_cotaskmem;
    typedef weak_any<shared_cotaskmem> weak_cotaskmem;
    typedef shared_any<unique_cotaskmem_string> shared_cotaskmem_string;
    typedef weak_any<shared_cotaskmem_string> weak_cotaskmem_string;
#endif // __WIL_OBJBASE_H_STL

#if defined(__WIL_OBJBASE_H_) && defined(__WIL_WINBASE_) && !defined(__WIL_OBJBASE_AND_WINBASE_H_) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_OBJBASE_AND_WINBASE_H_

    struct cotaskmem_secure_deleter
    {
        template <typename T>
        void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T* p) const
        {
            IMalloc* malloc;
            if (SUCCEEDED(::CoGetMalloc(1, &malloc)))
            {
                size_t const size = malloc->GetSize(p);
                if (size != static_cast<size_t>(-1))
                {
                    ::SecureZeroMemory(p, size);
                }
                malloc->Release();
            }
            ::CoTaskMemFree(p);
        }
    };

    template <typename T>
    using unique_cotaskmem_secure_ptr = wistd::unique_ptr<T, cotaskmem_secure_deleter>;

    /** Provides `std::make_unique()` semantics for secure resources allocated with `CoTaskMemAlloc()` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_cotaskmem_secure_nothrow<Foo>();
    if (foo)
    {
        // initialize allocated Foo object as appropriate
    }
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_cotaskmem_secure_ptr<T>>::type make_unique_cotaskmem_secure_nothrow(Args&&... args)
    {
        return unique_cotaskmem_secure_ptr<T>(make_unique_cotaskmem_nothrow<T>(wistd::forward<Args>(args)...).release());
    }

    /** Provides `std::make_unique()` semantics for secure array resources allocated with `CoTaskMemAlloc()` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_cotaskmem_secure_nothrow<Foo[]>(size);
    if (foos)
    {
        for (auto& elem : wil::make_range(foos.get(), size))
        {
            // initialize allocated Foo objects as appropriate
        }
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_cotaskmem_secure_ptr<T>>::type make_unique_cotaskmem_secure_nothrow(size_t size)
    {
        return unique_cotaskmem_secure_ptr<T>(make_unique_cotaskmem_nothrow<T>(size).release());
    }

    /** Provides `std::make_unique()` semantics for secure resources allocated with `CoTaskMemAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_cotaskmem_secure_failfast<Foo>();
    // initialize allocated Foo object as appropriate
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_cotaskmem_secure_ptr<T>>::type make_unique_cotaskmem_secure_failfast(Args&&... args)
    {
        unique_cotaskmem_secure_ptr<T> result(make_unique_cotaskmem_secure_nothrow<T>(wistd::forward<Args>(args)...));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

    /** Provides `std::make_unique()` semantics for secure array resources allocated with `CoTaskMemAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_cotaskmem_secure_failfast<Foo[]>(size);
    for (auto& elem : wil::make_range(foos.get(), size))
    {
        // initialize allocated Foo objects as appropriate
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_cotaskmem_secure_ptr<T>>::type make_unique_cotaskmem_secure_failfast(size_t size)
    {
        unique_cotaskmem_secure_ptr<T> result(make_unique_cotaskmem_secure_nothrow<T>(size));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    /** Provides `std::make_unique()` semantics for secure resources allocated with `CoTaskMemAlloc()`.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    auto foo = wil::make_unique_cotaskmem_secure<Foo>();
    // initialize allocated Foo object as appropriate
    ~~~
    */
    template <typename T, typename... Args>
    inline typename wistd::enable_if<!wistd::is_array<T>::value, unique_cotaskmem_secure_ptr<T>>::type make_unique_cotaskmem_secure(Args&&... args)
    {
        unique_cotaskmem_secure_ptr<T> result(make_unique_cotaskmem_secure_nothrow<T>(wistd::forward<Args>(args)...));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }

    /** Provides `std::make_unique()` semantics for secure array resources allocated with `CoTaskMemAlloc()`.
    See the overload of `wil::make_unique_cotaskmem_nothrow()` for non-array types for more details.
    ~~~
    const size_t size = 42;
    auto foos = wil::make_unique_cotaskmem_secure<Foo[]>(size);
    for (auto& elem : wil::make_range(foos.get(), size))
    {
        // initialize allocated Foo objects as appropriate
    }
    ~~~
    */
    template <typename T>
    inline typename wistd::enable_if<wistd::is_array<T>::value && wistd::extent<T>::value == 0, unique_cotaskmem_secure_ptr<T>>::type make_unique_cotaskmem_secure(size_t size)
    {
        unique_cotaskmem_secure_ptr<T> result(make_unique_cotaskmem_secure_nothrow<T>(size));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }
#endif // WIL_ENABLE_EXCEPTIONS

    typedef unique_cotaskmem_secure_ptr<wchar_t []> unique_cotaskmem_string_secure;

    /** Copies a given string into secure memory allocated with `CoTaskMemAlloc()` in a context that may not throw upon allocation failure.
    See the overload of `wil::make_cotaskmem_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_cotaskmem_string_secure_nothrow(L"a string");
    if (str)
    {
        std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    }
    ~~~
    */
    inline unique_cotaskmem_string_secure make_cotaskmem_string_secure_nothrow(_In_ PCWSTR source) WI_NOEXCEPT
    {
        return unique_cotaskmem_string_secure(make_cotaskmem_string_nothrow(source).release());
    }

    /** Copies a given string into secure memory allocated with `CoTaskMemAlloc()` in a context that must fail fast upon allocation failure.
    See the overload of `wil::make_cotaskmem_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_cotaskmem_string_secure_failfast(L"a string");
    std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    ~~~
    */
    inline unique_cotaskmem_string_secure make_cotaskmem_string_secure_failfast(_In_ PCWSTR source) WI_NOEXCEPT
    {
        unique_cotaskmem_string_secure result(make_cotaskmem_string_secure_nothrow(source));
        FAIL_FAST_IF_NULL_ALLOC(result);
        return result;
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    /** Copies a given string into secure memory allocated with `CoTaskMemAlloc()`.
    See the overload of `wil::make_cotaskmem_string_nothrow()` with supplied length for more details.
    ~~~
    auto str = wil::make_cotaskmem_string_secure(L"a string");
    std::wcout << L"This is " << str.get() << std::endl; // prints "This is a string"
    ~~~
    */
    inline unique_cotaskmem_string_secure make_cotaskmem_string_secure(_In_ PCWSTR source)
    {
        unique_cotaskmem_string_secure result(make_cotaskmem_string_secure_nothrow(source));
        THROW_IF_NULL_ALLOC(result);
        return result;
    }
#endif
#endif // __WIL_OBJBASE_AND_WINBASE_H_

#if defined(_OLE2_H_) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP) && !defined(__WIL_OLE2_H_)
#define __WIL_OLE2_H_
    typedef unique_struct<STGMEDIUM, decltype(&::ReleaseStgMedium), ::ReleaseStgMedium> unique_stg_medium;
    struct unique_hglobal_locked : public unique_any<void*, decltype(&::GlobalUnlock), ::GlobalUnlock>
    {
        unique_hglobal_locked() = delete;

        unique_hglobal_locked(_In_ STGMEDIUM& medium) : unique_any<void*, decltype(&::GlobalUnlock), ::GlobalUnlock>(medium.hGlobal)
        {
            // GlobalLock returns a pointer to the associated global memory block and that's what callers care about.
            m_globalMemory = GlobalLock(medium.hGlobal);
        }

        // In the future, we could easily add additional constructor overloads such as unique_hglobal_locked(HGLOBAL) to make consumption easier.
        
        pointer get() const
        {
            return m_globalMemory;
        }
        
        private:
            pointer m_globalMemory;
    };
#endif // __WIL_OLE2_H_

#if (defined(__SECURITY_RUNTIME_H__) || defined(__ACCESSHLPR_H__)) && !defined(__WIL__SECURITY_RUNTIME_H__)
#define __WIL__SECURITY_RUNTIME_H__
    typedef unique_any<PSECURITY_DESCRIPTOR, decltype(&::FreeTransientObjectSecurityDescriptor), ::FreeTransientObjectSecurityDescriptor> unique_transient_security_descriptor;
#endif // __WIL__SECURITY_RUNTIME_H__
#if defined(__WIL__SECURITY_RUNTIME_H__) && !defined(__WIL__SECURITY_RUNTIME_H__STL) && defined(WIL_RESOURCE_STL)
#define __WIL__SECURITY_RUNTIME_H__STL
    typedef shared_any<unique_transient_security_descriptor> shared_transient_security_descriptor;
    typedef weak_any<shared_transient_security_descriptor> weak_transient_security_descriptor;
#endif // __WIL__SECURITY_RUNTIME_H__STL

#if defined(_VAULTCLIEXT_H_) && !defined(__WIL_VAULTCLEXT_H__) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_VAULTCLEXT_H__
    /// @cond
    namespace details
    {
        inline void VaultCloseVault(_Pre_opt_valid_ _Frees_ptr_opt_ VAULT_HANDLE hVault) WI_NOEXCEPT
        {
#pragma prefast(suppress:6031, "annotation issue: VaultCloseVault is annotated __checkreturn, but a failure is not actionable to us");
            ::VaultCloseVault(&hVault);
        }
    }
    /// @endcond

    typedef unique_any<VAULT_HANDLE, decltype(&details::VaultCloseVault), details::VaultCloseVault> unique_vaulthandle;
#endif // __WIL_VAULTCLEXT_H__
#if defined(__WIL_VAULTCLEXT_H__) && !defined(__WIL_VAULTCLEXT_H__STL) && defined(WIL_RESOURCE_STL)
#define __WIL_VAULTCLEXT_H__
    typedef shared_any<unique_vaulthandle> shared_vaulthandle;
    typedef weak_any<unique_vaulthandle> weak_vaulthandle;
#endif // __WIL_VAULTCLEXT_H__

#if defined(_INC_COMMCTRL) && !defined(__WIL_INC_COMMCTRL)
#define __WIL_INC_COMMCTRL
    typedef unique_any<HIMAGELIST, decltype(&::ImageList_Destroy), ::ImageList_Destroy> unique_himagelist;
#endif // __WIL_INC_COMMCTRL
#if defined(__WIL_INC_COMMCTRL) && !defined(__WIL_INC_COMMCTRL_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_INC_COMMCTRL_STL
    typedef shared_any<unique_himagelist> shared_himagelist;
    typedef weak_any<shared_himagelist> weak_himagelist;
#endif // __WIL_INC_COMMCTRL_STL


#if defined(_WINSVC_) && !defined(__WIL_WINSVC) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_HANDLE_H_WINSVC
    typedef unique_any<SC_HANDLE, decltype(&::CloseServiceHandle), ::CloseServiceHandle> unique_schandle;
#endif // __WIL_HANDLE_H_WINSVC
#if defined(__WIL_HANDLE_H_WINSVC) && !defined(__WIL_HANDLE_H_WINSVC_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_HANDLE_H_WINSVC_STL
    typedef shared_any<unique_schandle> shared_schandle;
    typedef weak_any<shared_schandle> weak_schandle;
#endif // __WIL_HANDLE_H_WINSVC_STL

#if defined(_INC_DEVOBJ) && !defined(__WIL_INC_DEVOBJ_) && defined(__WIL_WINBASE_)
#define __WIL_INC_DEVOBJ_
    typedef unique_any_handle_invalid<decltype(&::DevObjDestroyDeviceInfoList), ::DevObjDestroyDeviceInfoList> unique_hdolist;
#endif // __WIL_INC_DEVOBJ_
#if defined(__WIL_INC_DEVOBJ_) && !defined(__WIL_INC_DEVOBJ_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_INC_DEVOBJ_STL
    typedef shared_any<unique_hdolist> shared_hdolist;
    typedef weak_any<shared_hdolist> weak_hdolist;
#endif // __WIL_INC_DEVOBJ_STL

#if defined(_INC_STDIO) && !defined(__WIL_INC_STDIO) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_INC_STDIO
    typedef unique_any<FILE*, decltype(&::_pclose), ::_pclose> unique_pipe;
#endif // __WIL_INC_STDIO
#if defined(__WIL_INC_STDIO) && !defined(__WIL__INC_STDIO_STL) && defined(WIL_RESOURCE_STL)
#define __WIL__INC_STDIO_STL
    typedef shared_any<unique_pipe> shared_pipe;
    typedef weak_any<shared_pipe> weak_pipe;
#endif // __WIL__INC_STDIO_STL

#if defined(_NTLSA_) && !defined(__WIL_NTLSA_)
#define __WIL_NTLSA_
    typedef unique_any<LSA_HANDLE, decltype(&::LsaClose), ::LsaClose> unique_hlsa;
#endif // _NTLSA_
#if defined(_NTLSA_) && !defined(__WIL_NTLSA_SHARED) && defined(__WIL_SHARED)
#define __WIL_NTLSA_SHARED;
    typedef shared_any<unique_hlsa> shared_hlsa;
    typedef weak_any<shared_hlsa> weak_hlsa;
#endif // _NTLSA_

#if defined(_NTLSA_IFS_) && !defined(__WIL_HANDLE_H_NTLSA_IFS_) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_HANDLE_H_NTLSA_IFS_
    struct lsa_deleter
    {
        template <typename T>
        void operator()(_Pre_opt_valid_ _Frees_ptr_opt_ T* p) const
        {
            ::LsaFreeReturnBuffer(p);
        }
    };

    template <typename T>
    using unique_lsa_ptr = wistd::unique_ptr<T, lsa_deleter>;
#endif // __WIL_HANDLE_H_NTLSA_IFS_

#if defined(_NTURTL_) && defined(__NOTHROW_T_DEFINED) && !defined(__WIL_NTURTL_WNF_)
#define __WIL_NTURTL_WNF_
    /** Windows Notification Facility (WNF) helpers that make it easy to produce and consume wnf state blocks.
    Learn more about WNF here: http://windowsarchive/sites/windows8docs/Win8%20Feature%20Docs/Windows%20Core%20(CORE)/Kernel%20Platform%20Group%20(KPG)/Kernel%20Core%20Software%20Services/Communications/Windows%20Notification%20Facility%20(WNF)/WNF%20-%20Functional%20Specification%20BETA.docm
    
    Clients must include <new> or <new.h> to enable use of these helpers as they use new(std::nothrow).
    This is to avoid the dependency on those headers that some clients can't tolerate.
    
    NOTE: The subscription wrappers will wait for outstanding callbacks on destruction
    
    Example use of throwing version:
    ~~~
    auto stateChangeSubscription = wil::make_wnf_subscription<TYPE>([](const TYPE& type)
        {
            // Do stuff with type
        });
    ~~~
    Example use of non-throwing version:
    ~~~
    auto stateChangeSubscription = wil::make_wnf_subscription_nothrow<TYPE>([](const TYPE& type)
        {
            // Do stuff with type
        });
    RETURN_HR_IF_FALSE(E_OUTOFMEMORY, stateChangeSubscription);
    ~~~
    */

    //! WNF_CHANGE_STAMP is an alias for ULONG that causes callers that don't specify changeStamp to select the zero size query with change stamp.
    //! To avoid this use WNF_CHANGE_STAMP_STRUCT instead of WNF_CHANGE_STAMP for this parameter.
    struct WNF_CHANGE_STAMP_STRUCT
    {
        WNF_CHANGE_STAMP value = 0;
        operator WNF_CHANGE_STAMP /* ULONG */ () const { return value; }
        static void AssignToOptParam(_In_opt_ WNF_CHANGE_STAMP_STRUCT* changeStampStruct, WNF_CHANGE_STAMP value)
        {
            if (changeStampStruct)
            {
                changeStampStruct->value = value;
            }
        }
    };

    //! Checks the published state and optionally the change stamp for state blocks of any size.
    inline HRESULT wnf_is_published_nothrow(WNF_STATE_NAME const& stateName, _Out_ bool* isPublished, _Out_opt_ WNF_CHANGE_STAMP_STRUCT* changeStamp = nullptr) WI_NOEXCEPT
    {
        *isPublished = false;
        WNF_CHANGE_STAMP_STRUCT::AssignToOptParam(changeStamp, static_cast<WNF_CHANGE_STAMP>(0));
        ULONG stateDataSize = 0;
        WNF_CHANGE_STAMP tempChangeStamp;
        HRESULT queryResult = HRESULT_FROM_NT(NtQueryWnfStateData(&stateName, nullptr, nullptr, &tempChangeStamp, nullptr, &stateDataSize));
        RETURN_HR_IF_TRUE(queryResult, FAILED(queryResult) && (queryResult != HRESULT_FROM_NT(STATUS_BUFFER_TOO_SMALL)));
        *isPublished = (tempChangeStamp != 0);
        WNF_CHANGE_STAMP_STRUCT::AssignToOptParam(changeStamp, tempChangeStamp);
        return S_OK;
    }

    //! Query the published state and optioanlly change stamp for zero sized state blocks. This will fail fast for not-zero sized state blocks.
    inline HRESULT wnf_query_nothrow(WNF_STATE_NAME const& stateName, _Out_ bool* isPublished, _Out_opt_ WNF_CHANGE_STAMP_STRUCT* changeStamp = nullptr) WI_NOEXCEPT
    {
        *isPublished = false;
        WNF_CHANGE_STAMP_STRUCT::AssignToOptParam(changeStamp, static_cast<WNF_CHANGE_STAMP>(0));
        ULONG stateDataSize = 0;
        WNF_CHANGE_STAMP tempChangeStamp;
        HRESULT queryResult = HRESULT_FROM_NT(NtQueryWnfStateData(&stateName, nullptr, nullptr, &tempChangeStamp, nullptr, &stateDataSize));
        RETURN_HR_IF_TRUE(queryResult, FAILED(queryResult) && (queryResult != HRESULT_FROM_NT(STATUS_BUFFER_TOO_SMALL)));
        __FAIL_FAST_ASSERT__((tempChangeStamp == 0) || (stateDataSize == 0));
        LOG_HR_IF_FALSE_MSG(E_UNEXPECTED, (tempChangeStamp == 0) || (stateDataSize == 0), "Inconsistent state data size in wnf_query");
        *isPublished = (tempChangeStamp != 0) && (stateDataSize == 0);  // if the size is non zero, consider it unpublished
        WNF_CHANGE_STAMP_STRUCT::AssignToOptParam(changeStamp, tempChangeStamp);
        return S_OK;
    }

    //! Query the published state, state data and optionally the change stamp for fixed size state blocks.
    template <typename state_data_t>
    HRESULT wnf_query_nothrow(WNF_STATE_NAME const& stateName, _Out_ bool* isPublished,
        _Pre_ _Notnull_ _Pre_ _Writable_elements_(1) _Post_ _When_((*isPublished == true), _Valid_) state_data_t* stateData, _Out_opt_ WNF_CHANGE_STAMP_STRUCT* changeStamp = nullptr) WI_NOEXCEPT
    {
        *isPublished = false;
        WNF_CHANGE_STAMP_STRUCT::AssignToOptParam(changeStamp, static_cast<WNF_CHANGE_STAMP>(0));
        ULONG stateDataSize = sizeof(*stateData);
        WNF_CHANGE_STAMP tempChangeStamp;
        HRESULT queryResult = HRESULT_FROM_NT(NtQueryWnfStateData(&stateName, nullptr, nullptr, &tempChangeStamp, stateData, &stateDataSize));
        RETURN_HR_IF_TRUE(queryResult, FAILED(queryResult) && (queryResult != HRESULT_FROM_NT(STATUS_BUFFER_TOO_SMALL)));
        __FAIL_FAST_ASSERT__((tempChangeStamp == 0) || (stateDataSize == sizeof(*stateData)));
        LOG_HR_IF_FALSE_MSG(E_UNEXPECTED, (tempChangeStamp == 0) || (stateDataSize == sizeof(*stateData)), "Inconsistent state data size in wnf_query");
        *isPublished = (tempChangeStamp != 0) && (stateDataSize == sizeof(*stateData));  // if the size is mismatched, consider it unpublished
        WNF_CHANGE_STAMP_STRUCT::AssignToOptParam(changeStamp, tempChangeStamp);
        return S_OK;
    }

    //! Query the published state, state data and optionally the change stamp for variable size state blocks.
    inline HRESULT wnf_query_nothrow(WNF_STATE_NAME const& stateName, _Out_ bool* isPublished,
        _Out_writes_bytes_(stateDataByteCount) void* stateData, size_t stateDataByteCount,
        _Out_ size_t* stateDataWrittenByteCount, _Out_opt_ WNF_CHANGE_STAMP_STRUCT* changeStamp = nullptr) WI_NOEXCEPT
    {
        *isPublished = false;
        *stateDataWrittenByteCount = 0;
        WNF_CHANGE_STAMP_STRUCT::AssignToOptParam(changeStamp, static_cast<WNF_CHANGE_STAMP>(0));
        
        WNF_CHANGE_STAMP tempChangeStamp;
        ULONG size = static_cast<ULONG>(stateDataByteCount);
        RETURN_IF_FAILED(HRESULT_FROM_NT(NtQueryWnfStateData(&stateName, nullptr, nullptr, &tempChangeStamp, stateData, &size)));
        *isPublished = (tempChangeStamp != 0);
        *stateDataWrittenByteCount = static_cast<size_t>(size);
        WNF_CHANGE_STAMP_STRUCT::AssignToOptParam(changeStamp, tempChangeStamp);
        return S_OK;
    }

    //! Returns true if the value is published, false if not (or malformed or on error).
    inline _Success_(return) bool try_wnf_query(WNF_STATE_NAME const& stateName, _Out_opt_ WNF_CHANGE_STAMP_STRUCT* changeStamp = nullptr) WI_NOEXCEPT
    {
        bool isPublished;
        return SUCCEEDED(wnf_query_nothrow(stateName, &isPublished, changeStamp)) && isPublished;
    }

    //! Returns true if the value is published, false if not (or malformed or on error)
    //! and returns the fixed size state data.
    template <typename state_data_t>
    _Success_(return) bool try_wnf_query(WNF_STATE_NAME const& stateName, _Out_ state_data_t* stateData, _Out_opt_ WNF_CHANGE_STAMP_STRUCT* changeStamp = nullptr) WI_NOEXCEPT
    {
        bool isPublished;
        return SUCCEEDED(wnf_query_nothrow(stateName, &isPublished, stateData, changeStamp)) && isPublished;
    }

    //! Publish a zero sized state block.
    inline HRESULT wnf_publish_nothrow(WNF_STATE_NAME const& stateName) WI_NOEXCEPT
    {
        return HRESULT_FROM_NT(RtlPublishWnfStateData(stateName, nullptr, nullptr, 0, nullptr));
    }

    //! Publish a variable sized state block.
    inline HRESULT wnf_publish_nothrow(WNF_STATE_NAME const& stateName, _In_reads_bytes_(size) const void* stateData, const size_t size) WI_NOEXCEPT
    {
        return HRESULT_FROM_NT(RtlPublishWnfStateData(stateName, nullptr, stateData, static_cast<const ULONG>(size), nullptr));
    }

    //! Publish a fixed sized state block.
    template <typename state_data_t>
    HRESULT wnf_publish_nothrow(WNF_STATE_NAME const& stateName, state_data_t const& stateData) WI_NOEXCEPT
    {
        return HRESULT_FROM_NT(RtlPublishWnfStateData(stateName, nullptr, &stateData, sizeof(stateData), nullptr));
    }

#ifdef WIL_ENABLE_EXCEPTIONS
    //! Checks the published state and optionally the change stamp for state blocks of any size.
    inline _Success_(return) bool wnf_is_published(WNF_STATE_NAME const& stateName, _Out_opt_ WNF_CHANGE_STAMP_STRUCT* changeStamp = nullptr)
    {
        bool isPublished;
        THROW_IF_FAILED(wnf_is_published_nothrow(stateName, &isPublished, changeStamp));
        return isPublished;
    }

    //! Query the published state, state data and optionally the change stamp for fixed size state blocks.
    template <typename state_data_t>
    _Success_(return) bool wnf_query(WNF_STATE_NAME const& stateName, _Out_ state_data_t* stateData, _Out_opt_ WNF_CHANGE_STAMP_STRUCT* changeStamp = nullptr)
    {
        bool isPublished;
        THROW_IF_FAILED(wnf_query_nothrow(stateName, &isPublished, stateData, changeStamp));
        return isPublished;
    }

    //! Query the published state and optioanlly change stamp for zero sized state blocks. This will fail fast for not-zero sized state blocks.
    inline _Success_(return) bool wnf_query(WNF_STATE_NAME const& stateName, _Out_opt_ WNF_CHANGE_STAMP_STRUCT* changeStamp = nullptr)
    {
        bool isPublished;
        THROW_IF_FAILED(wnf_query_nothrow(stateName, &isPublished, changeStamp));
        return isPublished;
    }

    //! Publish a zero-size state block.
    inline void wnf_publish(WNF_STATE_NAME const& stateName)
    {
        THROW_IF_FAILED(wnf_publish_nothrow(stateName));
    }

    //! Publish a fixed size state block.
    template <typename state_data_t>
    void wnf_publish(WNF_STATE_NAME const& stateName, state_data_t const& stateData)
    {
        THROW_IF_FAILED(wnf_publish_nothrow(stateName, stateData));
    }

#endif // WIL_ENABLE_EXCEPTIONS

    //! Manage the lifetime of a user subscription.
    typedef unique_any<PWNF_USER_SUBSCRIPTION, decltype(&::RtlUnsubscribeWnfNotificationWaitForCompletion), ::RtlUnsubscribeWnfNotificationWaitForCompletion, details::pointer_access_all> unique_pwnf_user_subscription;
    WNF_CHANGE_STAMP const WnfChangeStampLatest = MAXULONG;

    /// @cond
    namespace details
    {
        struct empty_wnf_state
        {
        };

        template <typename state_data_t>
        struct wnf_callback_type
        {
            typedef typename wistd::function<void(state_data_t const&)> type;
        };

        template <>
        struct wnf_callback_type<empty_wnf_state>
        {
            typedef wistd::function<void()> type;
        };

        struct wnf_subscription_state_base
        {
            virtual ~wnf_subscription_state_base() {};
            unique_pwnf_user_subscription m_subscription;
        };

        template <typename state_data_t = empty_wnf_state>
        struct wnf_subscription_state : public wnf_subscription_state_base
        {
            typedef typename details::wnf_callback_type<state_data_t>::type callback_type;

            wnf_subscription_state(callback_type&& callback) : m_callback(wistd::move(callback)) {}

            ~wnf_subscription_state() override
            {
                m_subscription.reset();     // subscription must be released prior to the callback (ordering)
            }

            template <typename state_data_t>
            void InternalCallback(state_data_t const* stateData, ULONG stateDataSize)
            {
                // Must check the size here vs. in the subscription lambda in make_wnf_subscription_state since the default
                // case uses empty_wnf_state which has a size > 0
                __FAIL_FAST_ASSERT__(stateDataSize == sizeof(*stateData));
                if (stateDataSize != sizeof(*stateData))
                {
                    LOG_HR_MSG(E_UNEXPECTED, "Inconsistent state data size in WNF callback");
                    return;
                }
                m_callback(*stateData);
            }

            template <>
            void InternalCallback<empty_wnf_state>(empty_wnf_state const* /*stateData*/, ULONG /*stateDataSize*/)
            {
                m_callback();
            }

            callback_type m_callback;
        };

        template <typename state_data_t = empty_wnf_state>
        HRESULT make_wnf_subscription_state(WNF_STATE_NAME const& stateName, typename details::wnf_callback_type<state_data_t>::type&& callback, _In_ WNF_CHANGE_STAMP subscribeFrom, _Outptr_ wnf_subscription_state<state_data_t>** subscriptionState) WI_NOEXCEPT
        {
            *subscriptionState = nullptr;
            wistd::unique_ptr<wnf_subscription_state<state_data_t>> subscriptionStateT(new(std::nothrow) wnf_subscription_state<state_data_t>(wistd::move(callback)));
            RETURN_IF_NULL_ALLOC(subscriptionStateT.get());

            if (subscribeFrom == WnfChangeStampLatest)
            {
                // Retrieve the latest changestamp and use that as the basis for change notifications
                ULONG bufferSize = 0;
                HRESULT queryResult = HRESULT_FROM_NT(NtQueryWnfStateData(&stateName, 0, nullptr, &subscribeFrom, nullptr, &bufferSize));
                RETURN_HR_IF_TRUE(queryResult, FAILED(queryResult) && (queryResult != HRESULT_FROM_NT(STATUS_BUFFER_TOO_SMALL)));
            }

            RETURN_IF_FAILED(HRESULT_FROM_NT(RtlSubscribeWnfStateChangeNotification(wil::out_param(subscriptionStateT->m_subscription), stateName, subscribeFrom,
                                             [](WNF_STATE_NAME /*stateName*/, WNF_CHANGE_STAMP changeStamp,
                                                WNF_TYPE_ID* /*typeID*/, void* callbackContext, const void* buffer, ULONG length) -> NTSTATUS
                {
                    // Delete notifications always have a change stamp of 0.  Since this class doesn't
                    // provide for a way to handle them differently, it drops them to avoid a fast-fail
                    // due to the payload size of a delete being potentially different than an update.
                    if (changeStamp != 0)
                    {
                        static_cast<details::wnf_subscription_state<state_data_t>*>(callbackContext)->InternalCallback(static_cast<state_data_t const*>(buffer), length);
                    }
                    return STATUS_SUCCESS;
                }, subscriptionStateT.get(), nullptr, 0, 0)));

            *subscriptionState = subscriptionStateT.release();
            return S_OK;
        }

        inline void delete_wnf_subscription_state(_In_opt_ wnf_subscription_state_base* subscriptionState) { delete subscriptionState;  }
    }
    /// @endcond

    //! Manage the lifetime of a subscription.
    typedef unique_any<details::wnf_subscription_state_base*, decltype(&details::delete_wnf_subscription_state), details::delete_wnf_subscription_state, details::pointer_access_none> unique_wnf_subscription;

    //! Subscribe to changes for a zero or fixed size wnf state block.
    //! the callback will only be called if the payload size matches the size of the state block.
    template<typename state_data_t = details::empty_wnf_state>
    unique_wnf_subscription make_wnf_subscription_nothrow(WNF_STATE_NAME const& stateName, typename details::wnf_callback_type<state_data_t>::type&& callback, _In_ WNF_CHANGE_STAMP subscribeFrom = 0) WI_NOEXCEPT
    {
        details::wnf_subscription_state<state_data_t>* subscriptionState;
        if (SUCCEEDED(details::make_wnf_subscription_state<state_data_t>(stateName, wistd::move(callback), subscribeFrom, &subscriptionState)))
        {
            return unique_wnf_subscription(subscriptionState);
        }
        return nullptr;
    }

#ifdef WIL_ENABLE_EXCEPTIONS

    //! Subscribe to changes for a zero or fixed size wnf state block.
    //! the callback will only be called if the payload size matches the size of the state block.
    template<typename state_data_t = details::empty_wnf_state>
    unique_wnf_subscription make_wnf_subscription(WNF_STATE_NAME const& stateName, typename details::wnf_callback_type<state_data_t>::type&& callback, _In_ WNF_CHANGE_STAMP subscribeFrom = 0)
    {
        details::wnf_subscription_state<state_data_t>* subscriptionState;
        THROW_IF_FAILED(details::make_wnf_subscription_state<state_data_t>(stateName, wistd::move(callback), subscribeFrom, &subscriptionState));
        return unique_wnf_subscription(subscriptionState);
    }

#endif // WIL_ENABLE_EXCEPTIONS

#endif // __WIL_NTURTL_WNF_

#if defined(__WERAPI_H__) && !defined(__WIL_WERAPI_H__) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_WERAPI_H__
    typedef unique_any<HREPORT, decltype(&WerReportCloseHandle), WerReportCloseHandle> unique_wer_report;
#endif

#if defined(__MIDLES_H__) && !defined(__WIL_MIDLES_H__) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_MIDLES_H__
    typedef unique_any<handle_t, decltype(&::MesHandleFree), ::MesHandleFree> unique_rpc_pickle;
#endif
#if defined(__WIL_MIDLES_H__) && !defined(__WIL_MIDLES_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_MIDLES_H_STL
    typedef shared_any<unique_rpc_pickle> shared_rpc_pickle;
    typedef weak_any<shared_rpc_pickle> weak_rpc_pickle;
#endif

#if defined(__RPCDCE_H__) && !defined(__WIL_RPCDCE_H__) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_RPCDCE_H__
    /// @cond
    namespace details
    {
        inline void WpRpcBindingFree(_Pre_opt_valid_ _Frees_ptr_opt_ RPC_BINDING_HANDLE binding)
        {
            ::RpcBindingFree(&binding);
        }

        inline void WpRpcBindingVectorFree(_Pre_opt_valid_ _Frees_ptr_opt_ RPC_BINDING_VECTOR* bindingVector)
        {
            ::RpcBindingVectorFree(&bindingVector);
        }
    }
    /// @endcond

    typedef unique_any<RPC_BINDING_HANDLE, decltype(&details::WpRpcBindingFree), details::WpRpcBindingFree> unique_rpc_binding;
    typedef unique_any<RPC_BINDING_VECTOR*, decltype(&details::WpRpcBindingVectorFree), details::WpRpcBindingVectorFree> unique_rpc_binding_vector;
#endif
#if defined(__WIL_RPCDCE_H__) && !defined(__WIL_RPCDCE_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_RPCDCE_H_STL
    typedef shared_any<unique_rpc_binding> shared_rpc_binding;
    typedef weak_any<shared_rpc_binding> weak_rpc_binding;
    typedef shared_any<unique_rpc_binding_vector> shared_rpc_binding_vector;
    typedef weak_any<shared_rpc_binding_vector> weak_rpc_binding_vector;
#endif

#if defined(__SRUMAPI_H__) && !defined(__WIL_SRUMAPI_H__)
#define __WIL_SRUMAPI_H__
    typedef unique_any<PSRU_STATS_RECORD_SET, decltype(&::SruFreeRecordSet), ::SruFreeRecordSet> unique_srum_recordset;
#endif
#if defined(__WIL_SRUMAPI_H__) && !defined(__WIL_SRUMAPI_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_SRUMAPI_H_STL
    typedef shared_any<unique_srum_recordset> shared_srum_recordset;
    typedef weak_any<shared_srum_recordset> weak_srum_recordset;
#endif

#if defined(_WCMAPI_H) && !defined(__WIL_WCMAPI_H_) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_WCMAPI_H_
    struct wcm_deleter
    {
        template<typename T>
        void operator()(_Frees_ptr_ T* p) const
        {
            ::WcmFreeMemory(p);
        }
    };

    template<typename T>
    using unique_wcm_ptr = wistd::unique_ptr<T, wcm_deleter>;
#endif

#if defined(_NETIOAPI_H_) && defined(_WS2IPDEF_) && defined(MIB_INVALID_TEREDO_PORT_NUMBER) && !defined(__WIL_NETIOAPI_H_) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_NETIOAPI_H_
    typedef unique_any<PMIB_IF_TABLE2, decltype(&::FreeMibTable), ::FreeMibTable> unique_mib_iftable;
#endif
#if defined(__WIL_NETIOAPI_H_) && !defined(__WIL_NETIOAPI_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_NETIOAPI_H_STL
    typedef shared_any<unique_mib_iftable> shared_mib_iftable;
    typedef weak_any<shared_mib_iftable> weak_mib_iftable;
#endif

#if defined(__WWAN_API_DECL__) && !defined(__WIL_WWAN_API_DECL__) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_WWAN_API_DECL__
    struct wwan_deleter
    {
        template<typename T>
        void operator()(_Frees_ptr_ T* p) const
        {
            ::WwanFreeMemory(p);
        }
    };

    template<typename T>
    using unique_wwan_ptr = wistd::unique_ptr<T, wwan_deleter>;

    /// @cond
    namespace details
    {
        inline void CloseWwanHandle(_Frees_ptr_ HANDLE hClientHandle)
        {
            ::WwanCloseHandle(hClientHandle, nullptr);
        }
    }
    /// @endcond

    typedef unique_any<HANDLE, decltype(&details::CloseWwanHandle), details::CloseWwanHandle, details::pointer_access_all, HANDLE, INVALID_HANDLE_VALUE> unique_wwan_handle;
#endif
#if defined(__WIL_WWAN_API_DECL__) && !defined(__WIL_WWAN_API_DECL_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_WWAN_API_DECL_STL
    typedef shared_any<unique_wwan_handle> shared_wwan_handle;
    typedef weak_any<shared_wwan_handle> weak_wwan_handle;
#endif

#if defined(_WLAN_WLANAPI_H) && !defined(__WIL_WLAN_WLANAPI_H) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP)
#define __WIL_WLAN_WLANAPI_H
    struct wlan_deleter
    {
        template<typename T>
        void operator()(_Frees_ptr_ T* p) const
        {
            ::WlanFreeMemory(p);
        }
    };

    template<typename T>
    using unique_wlan_ptr = wistd::unique_ptr < T, wlan_deleter > ;

    /// @cond
    namespace details
    {
        inline void CloseWlanHandle(_Frees_ptr_ HANDLE hClientHandle)
        {
            ::WlanCloseHandle(hClientHandle, nullptr);
        }
    }
    /// @endcond

    typedef unique_any<HANDLE, decltype(&details::CloseWlanHandle), details::CloseWlanHandle, details::pointer_access_all, HANDLE, INVALID_HANDLE_VALUE> unique_wlan_handle;
#endif
#if defined(__WIL_WLAN_WLANAPI_H) && !defined(__WIL_WLAN_WLANAPI_H_STL) && defined(WIL_RESOURCE_STL)
#define __WIL_WLAN_WLANAPI_H_STL
    typedef shared_any<unique_wlan_handle> shared_wlan_handle;
    typedef weak_any<shared_wlan_handle> weak_wlan_handle;
#endif

#if defined(_NTEXAPI_) && !defined(__WIL_NTEXAPI_)
#define __WIL_NTEXAPI_
    typedef wil::unique_struct<WNF_STATE_NAME, decltype(&::NtDeleteWnfStateName), ::NtDeleteWnfStateName> unique_wnf_state_name;
#endif

#if defined(STATELOCK_FLAG_TRANSIENT) && !defined(__WIL_STATELOCK_FLAG_TRANSIENT)
#define __WIL_STATELOCK_FLAG_TRANSIENT
    typedef unique_any<HSTATE, decltype(&::CloseState), ::CloseState, details::pointer_access_all, HSTATE, INVALID_STATE_HANDLE> unique_hstate;
#endif

#if defined(_EFSWRTEXT_H_) && !defined(__WIL_EFSWRTEXT_H_)
#define __WIL_EFSWRTEXT_H_
    typedef unique_any<PENCRYPTION_PROTECTOR_LIST, decltype(&::FreeIdentityProtectorList), ::FreeIdentityProtectorList> unique_pencryption_protector_list;
#endif

#if defined(_EDPUTIL_H_) && !defined(__WIL_EDPUTIL_H_)
#define __WIL_EDPUTIL_H_
    typedef unique_any<EDP_CONTEXT*, decltype(&::EdpFreeContext), ::EdpFreeContext> unique_edp_context;
#endif

#if defined(_HPOWERNOTIFY_DEF_) && !defined(__WIL_HPOWERNOTIFY_DEF_H_)
#define __WIL_HPOWERNOTIFY_DEF_H_
    typedef unique_any<HPOWERNOTIFY, decltype(&::UnregisterPowerSettingNotification), ::UnregisterPowerSettingNotification> unique_hpowernotify;
#endif

#if defined(__WIL_WINBASE_DESKTOP) && defined(SID_DEFINED) && !defined(__WIL_PSID_DEF_H_)
#define __WIL_PSID_DEF_H_
    typedef unique_any<PSID, decltype(&::LocalFree), ::LocalFree> unique_any_psid;
#if defined(_OBJBASE_H_)
    typedef unique_any<PSID, decltype(&::CoTaskMemFree), ::CoTaskMemFree> unique_cotaskmem_psid;
#endif
#endif

#if defined(__WINCRYPT_H_) && !defined(__WIL_WINCRYPT_DEF_H_)
#define __WIL_WINCRYPT_DEF_H_
    typedef unique_any<PCCERT_CONTEXT, decltype(&::CertFreeCertificateContext), :: CertFreeCertificateContext> unique_certContext;
#endif

#if defined(_PROCESSTHREADSAPI_H_) && !defined(__WIL_PROCESSTHREADSAPI_H_DESK_SYS) && WINAPI_FAMILY_PARTITION(WINAPI_PARTITION_DESKTOP | WINAPI_PARTITION_SYSTEM)
#define __WIL_PROCESSTHREADSAPI_H_DESK_SYS
    namespace details
    {
        inline void CloseProcessInformation(_In_ PROCESS_INFORMATION* p)
        {
            if (p->hProcess)
            {
                CloseHandle(p->hProcess);
            }

            if (p->hThread)
            {
                CloseHandle(p->hThread);
            }
        }
    }

    /** Manages the outbound parameter containing handles returned by `CreateProcess()` and related methods.
    ~~~
    unique_process_information process;
    CreateProcessW(..., CREATE_SUSPENDED, ..., &process);
    THROW_IF_WIN32_BOOL_FALSE(ResumeThread(process.hThread));
    THROW_LAST_ERROR_IF_FALSE(WaitForSingleObject(process.hProcess, INFINITE) == WAIT_OBJECT_0);
    ~~~
    */
    using unique_process_information = unique_struct<PROCESS_INFORMATION, decltype(&details::CloseProcessInformation), details::CloseProcessInformation>;
#endif

#if defined(POLICYMANAGER_H) && !defined(__WIL_POLICYSTRING_DEF_H_)
#define __WIL_POLICYSTRING_DEF_H_
        typedef unique_any<PWSTR, decltype(&::PolicyManager_FreeStringValue), ::PolicyManager_FreeStringValue> unique_policy_string;
#endif

} // namespace wil

#pragma warning(pop)

