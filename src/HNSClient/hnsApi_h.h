

/* this ALWAYS GENERATED file contains the definitions for the interfaces */


 /* File created by MIDL compiler version 8.00.0613 */
/* at Mon Jan 18 19:14:07 2038
 */
/* Compiler settings for hnsApi.idl:
    Oicf, W1, Zp8, env=Win64 (32b run), target_arch=AMD64 8.00.0613 
    protocol : dce , ms_ext, c_ext, robust
    error checks: allocation ref bounds_check enum stub_data 
    VC __declspec() decoration level: 
         __declspec(uuid()), __declspec(selectany), __declspec(novtable)
         DECLSPEC_UUID(), MIDL_INTERFACE()
*/
/* @@MIDL_FILE_HEADING(  ) */

#pragma warning( disable: 4049 )  /* more than 64k source lines */


/* verify that the <rpcndr.h> version is high enough to compile this file*/
#ifndef __REQUIRED_RPCNDR_H_VERSION__
#define __REQUIRED_RPCNDR_H_VERSION__ 475
#endif

#include "rpc.h"
#include "rpcndr.h"

#ifndef __RPCNDR_H_VERSION__
#error this stub requires an updated version of <rpcndr.h>
#endif /* __RPCNDR_H_VERSION__ */

#ifndef COM_NO_WINDOWS_H
#include "windows.h"
#include "ole2.h"
#endif /*COM_NO_WINDOWS_H*/

#ifndef __hnsApi_h_h__
#define __hnsApi_h_h__

#if defined(_MSC_VER) && (_MSC_VER >= 1020)
#pragma once
#endif

/* Forward Declarations */ 

#ifndef __IHNSApi_FWD_DEFINED__
#define __IHNSApi_FWD_DEFINED__
typedef interface IHNSApi IHNSApi;

#endif 	/* __IHNSApi_FWD_DEFINED__ */


#ifndef __HNSApi_FWD_DEFINED__
#define __HNSApi_FWD_DEFINED__

#ifdef __cplusplus
typedef class HNSApi HNSApi;
#else
typedef struct HNSApi HNSApi;
#endif /* __cplusplus */

#endif 	/* __HNSApi_FWD_DEFINED__ */


/* header files for imported files */
#include "oaidl.h"
#include "ocidl.h"

#ifdef __cplusplus
extern "C"{
#endif 


#ifndef __IHNSApi_INTERFACE_DEFINED__
#define __IHNSApi_INTERFACE_DEFINED__

/* interface IHNSApi */
/* [hidden][unique][helpstring][uuid][object] */ 


EXTERN_C const IID IID_IHNSApi;

#if defined(__cplusplus) && !defined(CINTERFACE)
    
    MIDL_INTERFACE("0b43d888-341a-4d8c-8c3a-f9ef5045df01")
    IHNSApi : public IUnknown
    {
    public:
        virtual HRESULT STDMETHODCALLTYPE Request( 
            /* [string][in] */ LPCWSTR Method,
            /* [string][in] */ LPCWSTR Path,
            /* [string][in] */ LPCWSTR Object,
            /* [string][out] */ LPWSTR *Response) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE AttachEndpoint( 
            /* [in] */ const GUID *EndpointId,
            /* [in] */ const UINT32 CompartmentId) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE DetachEndpoint( 
            /* [in] */ const GUID *EndpointId) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE AttachEndpointVNic( 
            /* [in] */ const GUID *EndpointId,
            /* [string][in] */ LPCWSTR NicName) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE GetEndpointConfiguration( 
            /* [in] */ const GUID *EndpointId,
            /* [out] */ DWORD *ConfigurationDataLength,
            /* [out] */ BYTE *EndpointConfigurationData) = 0;
        
    };
    
    
#else 	/* C style interface */

    typedef struct IHNSApiVtbl
    {
        BEGIN_INTERFACE
        
        HRESULT ( STDMETHODCALLTYPE *QueryInterface )( 
            IHNSApi * This,
            /* [in] */ REFIID riid,
            /* [annotation][iid_is][out] */ 
            _COM_Outptr_  void **ppvObject);
        
        ULONG ( STDMETHODCALLTYPE *AddRef )( 
            IHNSApi * This);
        
        ULONG ( STDMETHODCALLTYPE *Release )( 
            IHNSApi * This);
        
        HRESULT ( STDMETHODCALLTYPE *Request )( 
            IHNSApi * This,
            /* [string][in] */ LPCWSTR Method,
            /* [string][in] */ LPCWSTR Path,
            /* [string][in] */ LPCWSTR Object,
            /* [string][out] */ LPWSTR *Response);
        
        HRESULT ( STDMETHODCALLTYPE *AttachEndpoint )( 
            IHNSApi * This,
            /* [in] */ const GUID *EndpointId,
            /* [in] */ const UINT32 CompartmentId);
        
        HRESULT ( STDMETHODCALLTYPE *DetachEndpoint )( 
            IHNSApi * This,
            /* [in] */ const GUID *EndpointId);
        
        HRESULT ( STDMETHODCALLTYPE *AttachEndpointVNic )( 
            IHNSApi * This,
            /* [in] */ const GUID *EndpointId,
            /* [string][in] */ LPCWSTR NicName);
        
        HRESULT ( STDMETHODCALLTYPE *GetEndpointConfiguration )( 
            IHNSApi * This,
            /* [in] */ const GUID *EndpointId,
            /* [out] */ DWORD *ConfigurationDataLength,
            /* [out] */ BYTE *EndpointConfigurationData);
        
        END_INTERFACE
    } IHNSApiVtbl;

    interface IHNSApi
    {
        CONST_VTBL struct IHNSApiVtbl *lpVtbl;
    };

    

#ifdef COBJMACROS


#define IHNSApi_QueryInterface(This,riid,ppvObject)	\
    ( (This)->lpVtbl -> QueryInterface(This,riid,ppvObject) ) 

#define IHNSApi_AddRef(This)	\
    ( (This)->lpVtbl -> AddRef(This) ) 

#define IHNSApi_Release(This)	\
    ( (This)->lpVtbl -> Release(This) ) 


#define IHNSApi_Request(This,Method,Path,Object,Response)	\
    ( (This)->lpVtbl -> Request(This,Method,Path,Object,Response) ) 

#define IHNSApi_AttachEndpoint(This,EndpointId,CompartmentId)	\
    ( (This)->lpVtbl -> AttachEndpoint(This,EndpointId,CompartmentId) ) 

#define IHNSApi_DetachEndpoint(This,EndpointId)	\
    ( (This)->lpVtbl -> DetachEndpoint(This,EndpointId) ) 

#define IHNSApi_AttachEndpointVNic(This,EndpointId,NicName)	\
    ( (This)->lpVtbl -> AttachEndpointVNic(This,EndpointId,NicName) ) 

#define IHNSApi_GetEndpointConfiguration(This,EndpointId,ConfigurationDataLength,EndpointConfigurationData)	\
    ( (This)->lpVtbl -> GetEndpointConfiguration(This,EndpointId,ConfigurationDataLength,EndpointConfigurationData) ) 

#endif /* COBJMACROS */


#endif 	/* C style interface */




#endif 	/* __IHNSApi_INTERFACE_DEFINED__ */



#ifndef __HNSApiLib_LIBRARY_DEFINED__
#define __HNSApiLib_LIBRARY_DEFINED__

/* library HNSApiLib */
/* [helpstring][uuid] */ 


EXTERN_C const IID LIBID_HNSApiLib;

EXTERN_C const CLSID CLSID_HNSApi;

#ifdef __cplusplus

class DECLSPEC_UUID("c8a58877-c594-47fe-9197-bd808576f1da")
HNSApi;
#endif
#endif /* __HNSApiLib_LIBRARY_DEFINED__ */

/* Additional Prototypes for ALL interfaces */

/* end of Additional Prototypes */

#ifdef __cplusplus
}
#endif

#endif


