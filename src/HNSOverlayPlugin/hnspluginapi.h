

/* this ALWAYS GENERATED file contains the definitions for the interfaces */


 /* File created by MIDL compiler version 8.01.0618 */
/* @@MIDL_FILE_HEADING(  ) */



/* verify that the <rpcndr.h> version is high enough to compile this file*/
#ifndef __REQUIRED_RPCNDR_H_VERSION__
#define __REQUIRED_RPCNDR_H_VERSION__ 500
#endif

/* verify that the <rpcsal.h> version is high enough to compile this file*/
#ifndef __REQUIRED_RPCSAL_H_VERSION__
#define __REQUIRED_RPCSAL_H_VERSION__ 100
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

#ifndef __hnspluginapi_h__
#define __hnspluginapi_h__

#if defined(_MSC_VER) && (_MSC_VER >= 1020)
#pragma once
#endif

/* Forward Declarations */ 

#ifndef __IHNSObject_FWD_DEFINED__
#define __IHNSObject_FWD_DEFINED__
typedef interface IHNSObject IHNSObject;

#endif 	/* __IHNSObject_FWD_DEFINED__ */


#ifndef __IHNSObjectList_FWD_DEFINED__
#define __IHNSObjectList_FWD_DEFINED__
typedef interface IHNSObjectList IHNSObjectList;

#endif 	/* __IHNSObjectList_FWD_DEFINED__ */


#ifndef __IHNSProvider_FWD_DEFINED__
#define __IHNSProvider_FWD_DEFINED__
typedef interface IHNSProvider IHNSProvider;

#endif 	/* __IHNSProvider_FWD_DEFINED__ */


#ifndef __IHNSPlugin_FWD_DEFINED__
#define __IHNSPlugin_FWD_DEFINED__
typedef interface IHNSPlugin IHNSPlugin;

#endif 	/* __IHNSPlugin_FWD_DEFINED__ */


/* header files for imported files */
#include "oaidl.h"
#include "ocidl.h"

#ifdef __cplusplus
extern "C"{
#endif 


/* interface __MIDL_itf_hnspluginapi_0000_0000 */
/* [local] */ 

typedef /* [public][public][public][public] */ 
enum __MIDL___MIDL_itf_hnspluginapi_0000_0000_0001
    {
        HnsTypeUnknown	= 0,
        HnsTypeNetwork	= ( HnsTypeUnknown + 1 ) ,
        HnsTypeEndpoint	= ( HnsTypeNetwork + 1 ) ,
        HnsTypePolicy	= ( HnsTypeEndpoint + 1 ) ,
        HnsTypeSubnet	= ( HnsTypePolicy + 1 ) ,
        HnsTypeMacPool	= ( HnsTypeSubnet + 1 ) 
    } 	HNSObjectType;

typedef /* [public][public] */ 
enum __MIDL___MIDL_itf_hnspluginapi_0000_0000_0002
    {
        HnsPropertyUndefined	= 0,
        HnsPropertyBoolean	= ( HnsPropertyUndefined + 1 ) ,
        HnsPropertyInteger	= ( HnsPropertyBoolean + 1 ) ,
        HnsPropertyUnsignedInteger	= ( HnsPropertyInteger + 1 ) ,
        HnsPropertyGuid	= ( HnsPropertyUnsignedInteger + 1 ) ,
        HnsPropertyString	= ( HnsPropertyGuid + 1 ) ,
        HnsPropertyStringList	= ( HnsPropertyString + 1 ) ,
        HnsPropertyObject	= ( HnsPropertyStringList + 1 ) ,
        HnsPropertyObjectList	= ( HnsPropertyObject + 1 ) ,
        HnsPropertyPointer	= ( HnsPropertyObjectList + 1 ) ,
        HnsPropertyBlob	= ( HnsPropertyPointer + 1 ) 
    } 	HNSPropertyType;

typedef /* [public] */ 
enum __MIDL___MIDL_itf_hnspluginapi_0000_0000_0003
    {
        RequestCategoryNetwork	= 0,
        RequestCategoryEndpoint	= 1,
        RequestCategoryService	= 2,
        RequestCategoryPolicy	= 3,
        RequestCategoryCount	= ( RequestCategoryPolicy + 1 ) 
    } 	RequestCategory;

typedef /* [public] */ 
enum __MIDL___MIDL_itf_hnspluginapi_0000_0000_0004
    {
        RequestActivityPreCreate	= 0,
        RequestActivityPostCreate	= 1,
        RequestActivityFailCreate	= 2,
        RequestActivityPreDelete	= 3,
        RequestActivityPostDelete	= 4,
        RequestActivityServiceStart	= 0x20,
        RequestActivityServiceStop	= 0x21,
        RequestActivityPluginRequest	= 0x22,
        RequestActivityApplyPortPolicy	= 0x41,
        RequestActivityApplyNicPolicy	= 0x42,
        RequestActivityCleanupPolicy	= 0x43,
        RequestActivityApplyEndpointPolicy	= 0x44,
        RequestActivityCleanupEndpointPolicy	= 0x45
    } 	RequestActivity;




extern RPC_IF_HANDLE __MIDL_itf_hnspluginapi_0000_0000_v0_0_c_ifspec;
extern RPC_IF_HANDLE __MIDL_itf_hnspluginapi_0000_0000_v0_0_s_ifspec;

#ifndef __IHNSObject_INTERFACE_DEFINED__
#define __IHNSObject_INTERFACE_DEFINED__

/* interface IHNSObject */
/* [hidden][unique][helpstring][uuid][object] */ 


EXTERN_C const IID IID_IHNSObject;

#if defined(__cplusplus) && !defined(CINTERFACE)
    
    MIDL_INTERFACE("dbf70581-efeb-4936-b57b-347b74363ba8")
    IHNSObject : public IUnknown
    {
    public:
        virtual HRESULT STDMETHODCALLTYPE IsEqualId( 
            /* [in] */ __RPC__in GUID *CheckGuid) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE IsEqualType( 
            /* [in] */ HNSObjectType ObjectType) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE PropertyExists( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE PropertyIsType( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ HNSPropertyType ExpectedType) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE IsClientProperty( 
            __RPC__in LPCWSTR PropertyName) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE SetObjectId( 
            /* [in] */ __RPC__in GUID *ObjectId) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE SetObjectType( 
            /* [in] */ HNSObjectType ObjectType) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE SetPropertyVisible( 
            /* [in] */ __RPC__in LPCWSTR PropertyName,
            BOOLEAN Visible) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE SetPropertyPersistent( 
            /* [in] */ __RPC__in LPCWSTR PropertyName,
            BOOLEAN Persistent) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE DuplicateObject( 
            /* [out] */ __RPC__deref_out_opt IHNSObject **OutputObject) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE PropertyCount( 
            /* [out] */ __RPC__out DWORD *Count) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE GetPropertyName( 
            DWORD Index,
            /* [string][out] */ __RPC__deref_out_opt_string LPWSTR *PropertyName) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE SetGuid( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ __RPC__in GUID *GuidValue) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE SetInteger( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ DWORD IntegerValue) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE SetBoolean( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ BOOLEAN BoolValue) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE SetString( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [string][in] */ __RPC__in_string LPCWSTR StringValue) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE SetObject( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ __RPC__in_opt IHNSObject *Object) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE SetObjectList( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE AddListObject( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ __RPC__in_opt IHNSObject *Object) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE RemoveListObject( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ __RPC__in_opt IHNSObject *Object) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE GetGuid( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__out GUID *GuidValue) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE GetInteger( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__out DWORD *IntegerValue) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE GetBoolean( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__out BOOLEAN *BooleanValue) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE GetString( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__deref_out_opt LPWSTR *StringValue) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE GetObject( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__deref_out_opt IHNSObject **OutObject) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE GetObjectList( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__deref_out_opt IHNSObjectList **OutObjectList) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE DeleteProperty( 
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName) = 0;
        
    };
    
    
#else 	/* C style interface */

    typedef struct IHNSObjectVtbl
    {
        BEGIN_INTERFACE
        
        HRESULT ( STDMETHODCALLTYPE *QueryInterface )( 
            __RPC__in IHNSObject * This,
            /* [in] */ __RPC__in REFIID riid,
            /* [annotation][iid_is][out] */ 
            _COM_Outptr_  void **ppvObject);
        
        ULONG ( STDMETHODCALLTYPE *AddRef )( 
            __RPC__in IHNSObject * This);
        
        ULONG ( STDMETHODCALLTYPE *Release )( 
            __RPC__in IHNSObject * This);
        
        HRESULT ( STDMETHODCALLTYPE *IsEqualId )( 
            __RPC__in IHNSObject * This,
            /* [in] */ __RPC__in GUID *CheckGuid);
        
        HRESULT ( STDMETHODCALLTYPE *IsEqualType )( 
            __RPC__in IHNSObject * This,
            /* [in] */ HNSObjectType ObjectType);
        
        HRESULT ( STDMETHODCALLTYPE *PropertyExists )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName);
        
        HRESULT ( STDMETHODCALLTYPE *PropertyIsType )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ HNSPropertyType ExpectedType);
        
        HRESULT ( STDMETHODCALLTYPE *IsClientProperty )( 
            __RPC__in IHNSObject * This,
            __RPC__in LPCWSTR PropertyName);
        
        HRESULT ( STDMETHODCALLTYPE *SetObjectId )( 
            __RPC__in IHNSObject * This,
            /* [in] */ __RPC__in GUID *ObjectId);
        
        HRESULT ( STDMETHODCALLTYPE *SetObjectType )( 
            __RPC__in IHNSObject * This,
            /* [in] */ HNSObjectType ObjectType);
        
        HRESULT ( STDMETHODCALLTYPE *SetPropertyVisible )( 
            __RPC__in IHNSObject * This,
            /* [in] */ __RPC__in LPCWSTR PropertyName,
            BOOLEAN Visible);
        
        HRESULT ( STDMETHODCALLTYPE *SetPropertyPersistent )( 
            __RPC__in IHNSObject * This,
            /* [in] */ __RPC__in LPCWSTR PropertyName,
            BOOLEAN Persistent);
        
        HRESULT ( STDMETHODCALLTYPE *DuplicateObject )( 
            __RPC__in IHNSObject * This,
            /* [out] */ __RPC__deref_out_opt IHNSObject **OutputObject);
        
        HRESULT ( STDMETHODCALLTYPE *PropertyCount )( 
            __RPC__in IHNSObject * This,
            /* [out] */ __RPC__out DWORD *Count);
        
        HRESULT ( STDMETHODCALLTYPE *GetPropertyName )( 
            __RPC__in IHNSObject * This,
            DWORD Index,
            /* [string][out] */ __RPC__deref_out_opt_string LPWSTR *PropertyName);
        
        HRESULT ( STDMETHODCALLTYPE *SetGuid )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ __RPC__in GUID *GuidValue);
        
        HRESULT ( STDMETHODCALLTYPE *SetInteger )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ DWORD IntegerValue);
        
        HRESULT ( STDMETHODCALLTYPE *SetBoolean )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ BOOLEAN BoolValue);
        
        HRESULT ( STDMETHODCALLTYPE *SetString )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [string][in] */ __RPC__in_string LPCWSTR StringValue);
        
        HRESULT ( STDMETHODCALLTYPE *SetObject )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ __RPC__in_opt IHNSObject *Object);
        
        HRESULT ( STDMETHODCALLTYPE *SetObjectList )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName);
        
        HRESULT ( STDMETHODCALLTYPE *AddListObject )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ __RPC__in_opt IHNSObject *Object);
        
        HRESULT ( STDMETHODCALLTYPE *RemoveListObject )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [in] */ __RPC__in_opt IHNSObject *Object);
        
        HRESULT ( STDMETHODCALLTYPE *GetGuid )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__out GUID *GuidValue);
        
        HRESULT ( STDMETHODCALLTYPE *GetInteger )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__out DWORD *IntegerValue);
        
        HRESULT ( STDMETHODCALLTYPE *GetBoolean )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__out BOOLEAN *BooleanValue);
        
        HRESULT ( STDMETHODCALLTYPE *GetString )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__deref_out_opt LPWSTR *StringValue);
        
        HRESULT ( STDMETHODCALLTYPE *GetObject )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__deref_out_opt IHNSObject **OutObject);
        
        HRESULT ( STDMETHODCALLTYPE *GetObjectList )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName,
            /* [out] */ __RPC__deref_out_opt IHNSObjectList **OutObjectList);
        
        HRESULT ( STDMETHODCALLTYPE *DeleteProperty )( 
            __RPC__in IHNSObject * This,
            /* [string][in] */ __RPC__in_string LPCWSTR PropertyName);
        
        END_INTERFACE
    } IHNSObjectVtbl;

    interface IHNSObject
    {
        CONST_VTBL struct IHNSObjectVtbl *lpVtbl;
    };

    

#ifdef COBJMACROS


#define IHNSObject_QueryInterface(This,riid,ppvObject)	\
    ( (This)->lpVtbl -> QueryInterface(This,riid,ppvObject) ) 

#define IHNSObject_AddRef(This)	\
    ( (This)->lpVtbl -> AddRef(This) ) 

#define IHNSObject_Release(This)	\
    ( (This)->lpVtbl -> Release(This) ) 


#define IHNSObject_IsEqualId(This,CheckGuid)	\
    ( (This)->lpVtbl -> IsEqualId(This,CheckGuid) ) 

#define IHNSObject_IsEqualType(This,ObjectType)	\
    ( (This)->lpVtbl -> IsEqualType(This,ObjectType) ) 

#define IHNSObject_PropertyExists(This,PropertyName)	\
    ( (This)->lpVtbl -> PropertyExists(This,PropertyName) ) 

#define IHNSObject_PropertyIsType(This,PropertyName,ExpectedType)	\
    ( (This)->lpVtbl -> PropertyIsType(This,PropertyName,ExpectedType) ) 

#define IHNSObject_IsClientProperty(This,PropertyName)	\
    ( (This)->lpVtbl -> IsClientProperty(This,PropertyName) ) 

#define IHNSObject_SetObjectId(This,ObjectId)	\
    ( (This)->lpVtbl -> SetObjectId(This,ObjectId) ) 

#define IHNSObject_SetObjectType(This,ObjectType)	\
    ( (This)->lpVtbl -> SetObjectType(This,ObjectType) ) 

#define IHNSObject_SetPropertyVisible(This,PropertyName,Visible)	\
    ( (This)->lpVtbl -> SetPropertyVisible(This,PropertyName,Visible) ) 

#define IHNSObject_SetPropertyPersistent(This,PropertyName,Persistent)	\
    ( (This)->lpVtbl -> SetPropertyPersistent(This,PropertyName,Persistent) ) 

#define IHNSObject_DuplicateObject(This,OutputObject)	\
    ( (This)->lpVtbl -> DuplicateObject(This,OutputObject) ) 

#define IHNSObject_PropertyCount(This,Count)	\
    ( (This)->lpVtbl -> PropertyCount(This,Count) ) 

#define IHNSObject_GetPropertyName(This,Index,PropertyName)	\
    ( (This)->lpVtbl -> GetPropertyName(This,Index,PropertyName) ) 

#define IHNSObject_SetGuid(This,PropertyName,GuidValue)	\
    ( (This)->lpVtbl -> SetGuid(This,PropertyName,GuidValue) ) 

#define IHNSObject_SetInteger(This,PropertyName,IntegerValue)	\
    ( (This)->lpVtbl -> SetInteger(This,PropertyName,IntegerValue) ) 

#define IHNSObject_SetBoolean(This,PropertyName,BoolValue)	\
    ( (This)->lpVtbl -> SetBoolean(This,PropertyName,BoolValue) ) 

#define IHNSObject_SetString(This,PropertyName,StringValue)	\
    ( (This)->lpVtbl -> SetString(This,PropertyName,StringValue) ) 

#define IHNSObject_SetObject(This,PropertyName,Object)	\
    ( (This)->lpVtbl -> SetObject(This,PropertyName,Object) ) 

#define IHNSObject_SetObjectList(This,PropertyName)	\
    ( (This)->lpVtbl -> SetObjectList(This,PropertyName) ) 

#define IHNSObject_AddListObject(This,PropertyName,Object)	\
    ( (This)->lpVtbl -> AddListObject(This,PropertyName,Object) ) 

#define IHNSObject_RemoveListObject(This,PropertyName,Object)	\
    ( (This)->lpVtbl -> RemoveListObject(This,PropertyName,Object) ) 

#define IHNSObject_GetGuid(This,PropertyName,GuidValue)	\
    ( (This)->lpVtbl -> GetGuid(This,PropertyName,GuidValue) ) 

#define IHNSObject_GetInteger(This,PropertyName,IntegerValue)	\
    ( (This)->lpVtbl -> GetInteger(This,PropertyName,IntegerValue) ) 

#define IHNSObject_GetBoolean(This,PropertyName,BooleanValue)	\
    ( (This)->lpVtbl -> GetBoolean(This,PropertyName,BooleanValue) ) 

#define IHNSObject_GetString(This,PropertyName,StringValue)	\
    ( (This)->lpVtbl -> GetString(This,PropertyName,StringValue) ) 

#define IHNSObject_GetObject(This,PropertyName,OutObject)	\
    ( (This)->lpVtbl -> GetObject(This,PropertyName,OutObject) ) 

#define IHNSObject_GetObjectList(This,PropertyName,OutObjectList)	\
    ( (This)->lpVtbl -> GetObjectList(This,PropertyName,OutObjectList) ) 

#define IHNSObject_DeleteProperty(This,PropertyName)	\
    ( (This)->lpVtbl -> DeleteProperty(This,PropertyName) ) 

#endif /* COBJMACROS */


#endif 	/* C style interface */




#endif 	/* __IHNSObject_INTERFACE_DEFINED__ */


#ifndef __IHNSObjectList_INTERFACE_DEFINED__
#define __IHNSObjectList_INTERFACE_DEFINED__

/* interface IHNSObjectList */
/* [hidden][unique][helpstring][uuid][object] */ 


EXTERN_C const IID IID_IHNSObjectList;

#if defined(__cplusplus) && !defined(CINTERFACE)
    
    MIDL_INTERFACE("dbf70581-efeb-4936-b57b-347b74363ba9")
    IHNSObjectList : public IUnknown
    {
    public:
        virtual HRESULT STDMETHODCALLTYPE ObjectCount( 
            /* [out] */ __RPC__out DWORD *Count) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE GetObject( 
            DWORD ObjectIndex,
            /* [out] */ __RPC__deref_out_opt IHNSObject **OutObject) = 0;
        
    };
    
    
#else 	/* C style interface */

    typedef struct IHNSObjectListVtbl
    {
        BEGIN_INTERFACE
        
        HRESULT ( STDMETHODCALLTYPE *QueryInterface )( 
            __RPC__in IHNSObjectList * This,
            /* [in] */ __RPC__in REFIID riid,
            /* [annotation][iid_is][out] */ 
            _COM_Outptr_  void **ppvObject);
        
        ULONG ( STDMETHODCALLTYPE *AddRef )( 
            __RPC__in IHNSObjectList * This);
        
        ULONG ( STDMETHODCALLTYPE *Release )( 
            __RPC__in IHNSObjectList * This);
        
        HRESULT ( STDMETHODCALLTYPE *ObjectCount )( 
            __RPC__in IHNSObjectList * This,
            /* [out] */ __RPC__out DWORD *Count);
        
        HRESULT ( STDMETHODCALLTYPE *GetObject )( 
            __RPC__in IHNSObjectList * This,
            DWORD ObjectIndex,
            /* [out] */ __RPC__deref_out_opt IHNSObject **OutObject);
        
        END_INTERFACE
    } IHNSObjectListVtbl;

    interface IHNSObjectList
    {
        CONST_VTBL struct IHNSObjectListVtbl *lpVtbl;
    };

    

#ifdef COBJMACROS


#define IHNSObjectList_QueryInterface(This,riid,ppvObject)	\
    ( (This)->lpVtbl -> QueryInterface(This,riid,ppvObject) ) 

#define IHNSObjectList_AddRef(This)	\
    ( (This)->lpVtbl -> AddRef(This) ) 

#define IHNSObjectList_Release(This)	\
    ( (This)->lpVtbl -> Release(This) ) 


#define IHNSObjectList_ObjectCount(This,Count)	\
    ( (This)->lpVtbl -> ObjectCount(This,Count) ) 

#define IHNSObjectList_GetObject(This,ObjectIndex,OutObject)	\
    ( (This)->lpVtbl -> GetObject(This,ObjectIndex,OutObject) ) 

#endif /* COBJMACROS */


#endif 	/* C style interface */




#endif 	/* __IHNSObjectList_INTERFACE_DEFINED__ */


#ifndef __IHNSProvider_INTERFACE_DEFINED__
#define __IHNSProvider_INTERFACE_DEFINED__

/* interface IHNSProvider */
/* [hidden][unique][helpstring][uuid][object] */ 


EXTERN_C const IID IID_IHNSProvider;

#if defined(__cplusplus) && !defined(CINTERFACE)
    
    MIDL_INTERFACE("9292e69b-21ad-400e-9357-9e366d35d2a0")
    IHNSProvider : public IUnknown
    {
    public:
        virtual HRESULT STDMETHODCALLTYPE EnumerateNetworks( 
            /* [out] */ __RPC__deref_out_opt IHNSObjectList **OutNetworks) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE EnumerateEndpoints( 
            /* [out] */ __RPC__deref_out_opt IHNSObjectList **OutEndpoints) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE CreateObject( 
            /* [out] */ __RPC__deref_out_opt IHNSObject **OutNewObject) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE CreateNetwork( 
            /* [in] */ __RPC__in_opt IHNSObject *InputObject,
            /* [out] */ __RPC__deref_out_opt IHNSObject **CreatedNetwork) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE CreateEndpoint( 
            /* [in] */ __RPC__in_opt IHNSObject *InputObject,
            /* [out] */ __RPC__deref_out_opt IHNSObject **CreatedEndpoint) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE FindNetworkByName( 
            /* [string][in] */ __RPC__in_string LPCWSTR NetworkName,
            /* [out] */ __RPC__deref_out_opt IHNSObject **FoundObject) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE FindObject( 
            /* [in] */ __RPC__in GUID *ObjectGuid,
            /* [out] */ __RPC__deref_out_opt IHNSObject **FoundObject) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE DeleteRegisteredObject( 
            /* [in] */ __RPC__in GUID *ObjectGuid,
            HNSObjectType ObjectType) = 0;
        
    };
    
    
#else 	/* C style interface */

    typedef struct IHNSProviderVtbl
    {
        BEGIN_INTERFACE
        
        HRESULT ( STDMETHODCALLTYPE *QueryInterface )( 
            __RPC__in IHNSProvider * This,
            /* [in] */ __RPC__in REFIID riid,
            /* [annotation][iid_is][out] */ 
            _COM_Outptr_  void **ppvObject);
        
        ULONG ( STDMETHODCALLTYPE *AddRef )( 
            __RPC__in IHNSProvider * This);
        
        ULONG ( STDMETHODCALLTYPE *Release )( 
            __RPC__in IHNSProvider * This);
        
        HRESULT ( STDMETHODCALLTYPE *EnumerateNetworks )( 
            __RPC__in IHNSProvider * This,
            /* [out] */ __RPC__deref_out_opt IHNSObjectList **OutNetworks);
        
        HRESULT ( STDMETHODCALLTYPE *EnumerateEndpoints )( 
            __RPC__in IHNSProvider * This,
            /* [out] */ __RPC__deref_out_opt IHNSObjectList **OutEndpoints);
        
        HRESULT ( STDMETHODCALLTYPE *CreateObject )( 
            __RPC__in IHNSProvider * This,
            /* [out] */ __RPC__deref_out_opt IHNSObject **OutNewObject);
        
        HRESULT ( STDMETHODCALLTYPE *CreateNetwork )( 
            __RPC__in IHNSProvider * This,
            /* [in] */ __RPC__in_opt IHNSObject *InputObject,
            /* [out] */ __RPC__deref_out_opt IHNSObject **CreatedNetwork);
        
        HRESULT ( STDMETHODCALLTYPE *CreateEndpoint )( 
            __RPC__in IHNSProvider * This,
            /* [in] */ __RPC__in_opt IHNSObject *InputObject,
            /* [out] */ __RPC__deref_out_opt IHNSObject **CreatedEndpoint);
        
        HRESULT ( STDMETHODCALLTYPE *FindNetworkByName )( 
            __RPC__in IHNSProvider * This,
            /* [string][in] */ __RPC__in_string LPCWSTR NetworkName,
            /* [out] */ __RPC__deref_out_opt IHNSObject **FoundObject);
        
        HRESULT ( STDMETHODCALLTYPE *FindObject )( 
            __RPC__in IHNSProvider * This,
            /* [in] */ __RPC__in GUID *ObjectGuid,
            /* [out] */ __RPC__deref_out_opt IHNSObject **FoundObject);
        
        HRESULT ( STDMETHODCALLTYPE *DeleteRegisteredObject )( 
            __RPC__in IHNSProvider * This,
            /* [in] */ __RPC__in GUID *ObjectGuid,
            HNSObjectType ObjectType);
        
        END_INTERFACE
    } IHNSProviderVtbl;

    interface IHNSProvider
    {
        CONST_VTBL struct IHNSProviderVtbl *lpVtbl;
    };

    

#ifdef COBJMACROS


#define IHNSProvider_QueryInterface(This,riid,ppvObject)	\
    ( (This)->lpVtbl -> QueryInterface(This,riid,ppvObject) ) 

#define IHNSProvider_AddRef(This)	\
    ( (This)->lpVtbl -> AddRef(This) ) 

#define IHNSProvider_Release(This)	\
    ( (This)->lpVtbl -> Release(This) ) 


#define IHNSProvider_EnumerateNetworks(This,OutNetworks)	\
    ( (This)->lpVtbl -> EnumerateNetworks(This,OutNetworks) ) 

#define IHNSProvider_EnumerateEndpoints(This,OutEndpoints)	\
    ( (This)->lpVtbl -> EnumerateEndpoints(This,OutEndpoints) ) 

#define IHNSProvider_CreateObject(This,OutNewObject)	\
    ( (This)->lpVtbl -> CreateObject(This,OutNewObject) ) 

#define IHNSProvider_CreateNetwork(This,InputObject,CreatedNetwork)	\
    ( (This)->lpVtbl -> CreateNetwork(This,InputObject,CreatedNetwork) ) 

#define IHNSProvider_CreateEndpoint(This,InputObject,CreatedEndpoint)	\
    ( (This)->lpVtbl -> CreateEndpoint(This,InputObject,CreatedEndpoint) ) 

#define IHNSProvider_FindNetworkByName(This,NetworkName,FoundObject)	\
    ( (This)->lpVtbl -> FindNetworkByName(This,NetworkName,FoundObject) ) 

#define IHNSProvider_FindObject(This,ObjectGuid,FoundObject)	\
    ( (This)->lpVtbl -> FindObject(This,ObjectGuid,FoundObject) ) 

#define IHNSProvider_DeleteRegisteredObject(This,ObjectGuid,ObjectType)	\
    ( (This)->lpVtbl -> DeleteRegisteredObject(This,ObjectGuid,ObjectType) ) 

#endif /* COBJMACROS */


#endif 	/* C style interface */




#endif 	/* __IHNSProvider_INTERFACE_DEFINED__ */


#ifndef __IHNSPlugin_INTERFACE_DEFINED__
#define __IHNSPlugin_INTERFACE_DEFINED__

/* interface IHNSPlugin */
/* [hidden][unique][helpstring][uuid][object] */ 


EXTERN_C const IID IID_IHNSPlugin;

#if defined(__cplusplus) && !defined(CINTERFACE)
    
    MIDL_INTERFACE("398a46e6-572e-4d19-8476-1caba8462894")
    IHNSPlugin : public IUnknown
    {
    public:
        virtual HRESULT STDMETHODCALLTYPE Initialize( 
            /* [in] */ __RPC__in_opt IHNSProvider *PluginProvider) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE Teardown( 
            /* [in] */ __RPC__in_opt IHNSProvider *PluginProvider) = 0;
        
        virtual HRESULT STDMETHODCALLTYPE HandleRequest( 
            DWORD RequestType,
            /* [in] */ __RPC__in_opt IHNSProvider *PluginProvider,
            /* [optional][in] */ __RPC__in_opt IHNSObject *PluginObject) = 0;
        
    };
    
    
#else 	/* C style interface */

    typedef struct IHNSPluginVtbl
    {
        BEGIN_INTERFACE
        
        HRESULT ( STDMETHODCALLTYPE *QueryInterface )( 
            __RPC__in IHNSPlugin * This,
            /* [in] */ __RPC__in REFIID riid,
            /* [annotation][iid_is][out] */ 
            _COM_Outptr_  void **ppvObject);
        
        ULONG ( STDMETHODCALLTYPE *AddRef )( 
            __RPC__in IHNSPlugin * This);
        
        ULONG ( STDMETHODCALLTYPE *Release )( 
            __RPC__in IHNSPlugin * This);
        
        HRESULT ( STDMETHODCALLTYPE *Initialize )( 
            __RPC__in IHNSPlugin * This,
            /* [in] */ __RPC__in_opt IHNSProvider *PluginProvider);
        
        HRESULT ( STDMETHODCALLTYPE *Teardown )( 
            __RPC__in IHNSPlugin * This,
            /* [in] */ __RPC__in_opt IHNSProvider *PluginProvider);
        
        HRESULT ( STDMETHODCALLTYPE *HandleRequest )( 
            __RPC__in IHNSPlugin * This,
            DWORD RequestType,
            /* [in] */ __RPC__in_opt IHNSProvider *PluginProvider,
            /* [optional][in] */ __RPC__in_opt IHNSObject *PluginObject);
        
        END_INTERFACE
    } IHNSPluginVtbl;

    interface IHNSPlugin
    {
        CONST_VTBL struct IHNSPluginVtbl *lpVtbl;
    };

    

#ifdef COBJMACROS


#define IHNSPlugin_QueryInterface(This,riid,ppvObject)	\
    ( (This)->lpVtbl -> QueryInterface(This,riid,ppvObject) ) 

#define IHNSPlugin_AddRef(This)	\
    ( (This)->lpVtbl -> AddRef(This) ) 

#define IHNSPlugin_Release(This)	\
    ( (This)->lpVtbl -> Release(This) ) 


#define IHNSPlugin_Initialize(This,PluginProvider)	\
    ( (This)->lpVtbl -> Initialize(This,PluginProvider) ) 

#define IHNSPlugin_Teardown(This,PluginProvider)	\
    ( (This)->lpVtbl -> Teardown(This,PluginProvider) ) 

#define IHNSPlugin_HandleRequest(This,RequestType,PluginProvider,PluginObject)	\
    ( (This)->lpVtbl -> HandleRequest(This,RequestType,PluginProvider,PluginObject) ) 

#endif /* COBJMACROS */


#endif 	/* C style interface */




#endif 	/* __IHNSPlugin_INTERFACE_DEFINED__ */


/* Additional Prototypes for ALL interfaces */

/* end of Additional Prototypes */

#ifdef __cplusplus
}
#endif

#endif


