#pragma once

#pragma once

#include "NetSetupApiTypes.h"
#include "NetSetupStandardProperties.h"

#if __cplusplus
extern "C" {
#endif

#ifndef NETSETUP_LINKAGE 
#  define NETSETUP_LINKAGE __declspec(dllimport)
#endif

	DECLARE_HANDLE(HNETSETUP);

	typedef HRESULT WINAPI
		NETSETUP_INITIALIZE_API(
			_In_opt_ NETSETUP_ENVIRONMENT *Environment,
			_Out_ HNETSETUP *hNetSetup);

	NETSETUP_LINKAGE NETSETUP_INITIALIZE_API NetSetupInitialize;

	typedef HRESULT WINAPI
		NETSETUP_VALIDATE_TRANSACTION_API(
			_In_ HNETSETUP hNetSetup);

	NETSETUP_LINKAGE NETSETUP_VALIDATE_TRANSACTION_API NetSetupValidateTransaction;

	typedef VOID WINAPI
		NETSETUP_ROLLBACK_API(
			_In_ HNETSETUP hNetSetup);

	NETSETUP_LINKAGE NETSETUP_ROLLBACK_API NetSetupRollback;

	typedef HRESULT WINAPI
		NETSETUP_COMMIT_API(
			_In_ HNETSETUP hNetSetup);

	NETSETUP_LINKAGE NETSETUP_COMMIT_API NetSetupCommit;

	typedef VOID WINAPI
		NETSETUP_CLOSE_API(
			_In_ HNETSETUP hNetSetup);

	NETSETUP_LINKAGE NETSETUP_CLOSE_API NetSetupClose;

	typedef HRESULT WINAPI
		NETSETUP_CREATEOBJECT_API(
			_In_ HNETSETUP hNetSetup,
			_In_ NETSETUP_OBJECT_TYPE ObjectType,
			_In_ ULONG NumInitialProperties,
			_In_reads_opt_(NumInitialProperties) const NETSETUPPROPERTY *InitialProperties,
			_Out_ GUID *ObjectId);

	NETSETUP_LINKAGE NETSETUP_CREATEOBJECT_API NetSetupCreateObject;

	typedef HRESULT WINAPI
		NETSETUP_DELETEOBJECT_API(
			_In_ HNETSETUP hNetSetup,
			_In_ NETSETUP_OBJECT_TYPE ObjectType,
			_In_ GUID ObjectId);

	NETSETUP_LINKAGE NETSETUP_DELETEOBJECT_API NetSetupDeleteObject;

	typedef HRESULT WINAPI
		NETSETUP_GETOBJECTPROPERTYKEYS_API(
			_In_ HNETSETUP hNetSetup,
			_In_ NETSETUP_OBJECT_TYPE ObjectType,
			_In_ GUID ObjectId,
			_Out_writes_to_opt_(ReturnedPropertiesSize, *NumPropertiesReturned) NETSETUPPROPKEY *ReturnedProperties,
			_In_ ULONG ReturnedPropertiesSize,
			_Out_ ULONG *NumPropertiesReturned,
			_Out_ ULONG *NumPropertiesTotal);

	NETSETUP_LINKAGE NETSETUP_GETOBJECTPROPERTYKEYS_API NetSetupGetObjectPropertyKeys;

	typedef HRESULT WINAPI
		NETSETUP_GETOBJECTPROPERTIES_API(
			_In_ HNETSETUP hNetSetup,
			_In_ NETSETUP_OBJECT_TYPE ObjectType,
			_In_ GUID ObjectId,
			_In_ ULONG QueryFlags,
			_In_ ULONG NumRequestedProperties,
			_In_reads_(NumRequestedProperties)
			const NETSETUPPROPCOMPKEY *RequestedProperties,
			_Out_ PULONG NumReturnedProperties,
			_Outptr_result_buffer_(*NumReturnedProperties)
			const NETSETUPPROPERTY **ReturnedProperties);

	NETSETUP_LINKAGE NETSETUP_GETOBJECTPROPERTIES_API NetSetupGetObjectProperties;

	typedef VOID WINAPI
		NETSETUP_FREEOBJECTPROPERTIES_API(
			_In_ ULONG NumReturnedProperties,
			_In_reads_(NumReturnedProperties) const NETSETUPPROPERTY *ReturnedProperties);

	NETSETUP_LINKAGE NETSETUP_FREEOBJECTPROPERTIES_API NetSetupFreeObjectProperties;

	typedef _Check_return_ HRESULT WINAPI
		NETSETUP_SETOBJECTPROPERTIES_API(
			_In_ HNETSETUP hNetSetup,
			_In_ NETSETUP_OBJECT_TYPE ObjectType,
			_In_ GUID ObjectId,
			_In_ ULONG NumProperties,
			_In_reads_(NumProperties) const NETSETUPPROPERTY *Properties);

	NETSETUP_LINKAGE NETSETUP_SETOBJECTPROPERTIES_API NetSetupSetObjectProperties;

	DECLARE_HANDLE(HNETSETUPQUERY);
	typedef HNETSETUPQUERY *PHNETSETUPQUERY;

	typedef VOID WINAPI
		NETSETUP_QUERY_RESULT_CALLBACK(
			_In_ HNETSETUPQUERY hQuery,
			_In_opt_ PVOID CallbackContext,
			_In_ const NETSETUP_QUERY_RESULT_ACTION_DATA *ActionData);

	typedef NETSETUP_QUERY_RESULT_CALLBACK * PNETSETUP_QUERY_RESULT_CALLBACK;

	typedef _Check_return_ HRESULT WINAPI
		NETSETUP_CREATEOBJECTQUERY_API(
			_In_ HNETSETUP hNetSetup,
			_In_ NETSETUP_OBJECT_TYPE ObjectType,
			_In_opt_ const GUID *ObjectId,
			_In_ ULONG QueryFlags,
			_In_ ULONG NumRequestedProperties,
			_In_reads_opt_(NumRequestedProperties) const NETSETUPPROPCOMPKEY *RequestedProperties,
			_In_ ULONG NumFilterExpressions,
			_In_reads_opt_(NumFilterExpressions) const NETSETUPPROP_FILTER_EXPRESSION *FilterExpressions,
			_In_ PNETSETUP_QUERY_RESULT_CALLBACK Callback,
			_In_opt_ PVOID CallbackContext,
			_Out_ PHNETSETUPQUERY hQuery);

	NETSETUP_LINKAGE NETSETUP_CREATEOBJECTQUERY_API NetSetupCreateObjectQuery;

	typedef VOID WINAPI
		NETSETUP_CLOSEOBJECTQUERY_API(
			_In_ HNETSETUPQUERY hQuery);

	NETSETUP_LINKAGE NETSETUP_CLOSEOBJECTQUERY_API NetSetupCloseObjectQuery;

	typedef _Check_return_ HRESULT WINAPI
		NETSETUP_GETOBJECTS_API(
			_In_ HNETSETUP hNetSetup,
			_In_ NETSETUP_OBJECT_TYPE ObjectType,
			_In_opt_ const GUID *ObjectId,
			_In_ ULONG QueryFlags,
			_In_ ULONG NumRequestedProperties,
			_In_reads_opt_(NumRequestedProperties) const NETSETUPPROPCOMPKEY *RequestedProperties,
			_In_ ULONG NumFilterExpressions,
			_In_reads_opt_(NumFilterExpressions) const NETSETUPPROP_FILTER_EXPRESSION *FilterExpressions,
			_Out_ PULONG NumReturnedObjects,
			_Outptr_result_buffer_maybenull_(*NumReturnedObjects) const NETSETUP_OBJECT **ReturnedObjects);

	NETSETUP_LINKAGE NETSETUP_GETOBJECTS_API NetSetupGetObjects;

	typedef VOID WINAPI
		NETSETUP_FREEOBJECTS_API(
			_In_ ULONG NumReturnedObjects,
			_In_reads_(NumReturnedObjects) const NETSETUP_OBJECT *ReturnedObjects);

	NETSETUP_LINKAGE NETSETUP_FREEOBJECTS_API NetSetupFreeObjects;

	typedef _Check_return_ HRESULT WINAPI
		NETSETUP_SERIALIZEFILTER_API(
			_In_ ULONG NumFilterExpressions,
			_In_ NETSETUPPROP_FILTER_EXPRESSION const * FilterExpressions,
			_Out_ PULONG NumBytes,
			_Outptr_result_buffer_maybenull_(*NumBytes) const BYTE **Bytes);

	NETSETUP_LINKAGE NETSETUP_SERIALIZEFILTER_API NetSetupSerializeFilter;

	typedef VOID WINAPI
		NETSETUP_FREESERIALIZEDFILTER_API(
			_In_ const BYTE* Bytes);

	NETSETUP_LINKAGE NETSETUP_FREESERIALIZEDFILTER_API NetSetupFreeSerializedFilter;

	typedef HRESULT WINAPI
		NETSETUP_SYNCHRONIZEDEVICES_API(_Reserved_ PVOID Reserved);

	NETSETUP_LINKAGE NETSETUP_SYNCHRONIZEDEVICES_API NetSetupSynchronizeDevices;

	inline
		NETSETUP_OBJECT_TYPE
		NETSETUP_REFLECTED_PROPERTY_FROM_PID(
			_In_ NETSETUPPROPID pid)
	{
		return ((NETSETUP_OBJECT_TYPE)((ULONG)(pid) << 16 | (ULONG)NetSetupObjectTypeReflectedProperty));
	}

	inline
		NETSETUP_OBJECT_TYPE
		NETSETUP_REFLECTED_PROPERTY_FROM_KEY(
			_In_ NETSETUPPROPKEY const *key)
	{
		return NETSETUP_REFLECTED_PROPERTY_FROM_PID(key->pid);
	}

	inline
		NETSETUPPROPID
		NETSETUP_PROPID_FROM_REFLECTED_PROPERTY(NETSETUP_OBJECT_TYPE type)
	{
		return (ULONG)type >> 16u;
	}

	typedef HRESULT WINAPI NETSETUP_OPENADVANCEDPROPERTYKEY_API(
		_In_ HNETSETUP hNetSetup,
		_In_ NETSETUP_OBJECT_TYPE ObjectType,
		_In_ GUID CONST* ObjectId,
		_In_ REGSAM desiredAccess,
		_Out_ HKEY* hkey);

	NETSETUP_LINKAGE NETSETUP_OPENADVANCEDPROPERTYKEY_API NetSetupOpenAdvancedPropertyKey;

#if __cplusplus
} // extern "C"
#endif