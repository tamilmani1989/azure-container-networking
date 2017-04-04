// ***********************************************************************
// <copyright file="PluginMain.cpp" company="Microsoft">
//     Copyright (c) Microsoft Corporation. All rights reserved.
// </copyright>
// <summary></summary>
// ***********************************************************************
#include "PluginMain.h"
#include "hnsPluginApi_i.c" // Get plugin IIDs
#include "ovPluginException.h"

using namespace Microsoft::Azure::Networking::OverlayPlugin;

PCClassFactory* factory = NULL;
ULONG OutstandingObjects = 0;

HRESULT __stdcall
DllGetClassObject(
    REFCLSID rclsid,
    REFIID riid,
    LPVOID *ppv
    )
{
    if (factory == NULL)
    {
        factory = new PCClassFactory();
    }

    if (IsEqualCLSID(rclsid, ClsidPlugin))
    {
        HRESULT hr = factory->QueryInterface(riid, ppv);
        return hr;
    }
    else
    {
        return CLASS_E_CLASSNOTAVAILABLE;
    }
}

HRESULT __stdcall
DllCanUnloadNow(void)
{
    return OutstandingObjects ? S_FALSE : S_OK;
}
////////////////////////////////////////////////////////////////////////////////
// PCClassFactory::QueryInterface
////////////////////////////////////////////////////////////////////////////////
HRESULT 
PCClassFactory::QueryInterface(
    REFIID riid, 
    void ** ppvObject)
{
    if (!ppvObject)
    {
        return E_POINTER;
    }

    if (IsEqualIID(riid, IID_IUnknown) || IsEqualIID(riid, IID_IClassFactory))
    {
        *ppvObject = static_cast<IClassFactory*>(this);
    }
    else
    {
        *ppvObject = NULL;
        return E_NOINTERFACE;
    }

    reinterpret_cast<IUnknown*>(*ppvObject)->AddRef();
    return S_OK;
}
////////////////////////////////////////////////////////////////////////////////
// PCClassFactory::AddRef
////////////////////////////////////////////////////////////////////////////////
ULONG 
PCClassFactory::AddRef()
{
    return 2;
}
////////////////////////////////////////////////////////////////////////////////
// PCClassFactory::Release
////////////////////////////////////////////////////////////////////////////////
ULONG 
PCClassFactory::Release()
{
    return 1;
}
////////////////////////////////////////////////////////////////////////////////
// PCClassFactory::CreateInstance
////////////////////////////////////////////////////////////////////////////////
HRESULT 
PCClassFactory::CreateInstance(
    IUnknown * pUnkOuter, 
    REFIID riid, 
    void ** ppvObject)
{
    if (ppvObject == NULL)
        return E_POINTER;

    *ppvObject = NULL;

    if (pUnkOuter != NULL)
        return E_INVALIDARG;

    PCPlugin* newPlugin = new PCPlugin();
    HRESULT hr = newPlugin->QueryInterface(riid, ppvObject);
    newPlugin->Release();

    return hr;
}
////////////////////////////////////////////////////////////////////////////////
// PCClassFactory::LockServer
////////////////////////////////////////////////////////////////////////////////
HRESULT 
PCClassFactory::LockServer(
    BOOL fLock)
{
    UNREFERENCED_PARAMETER(fLock);
    return S_OK;
}
////////////////////////////////////////////////////////////////////////////////
// PCPlugin::PCPlugin
////////////////////////////////////////////////////////////////////////////////
PCPlugin::PCPlugin() :
    RefCount(1)
{
    InterlockedIncrement(&OutstandingObjects);
}
////////////////////////////////////////////////////////////////////////////////
// PCPlugin::PCPlugin
////////////////////////////////////////////////////////////////////////////////
PCPlugin::~PCPlugin()
{
    InterlockedDecrement(&OutstandingObjects);
}
////////////////////////////////////////////////////////////////////////////////
// PCPlugin::QueryInterface
////////////////////////////////////////////////////////////////////////////////
HRESULT 
PCPlugin::QueryInterface(
    REFIID riid, 
    void ** ppvObject)
{
    if (!ppvObject)
    {
        return E_POINTER;
    }

    if (IsEqualIID(riid, IID_IUnknown) || IsEqualIID(riid, IID_IHNSPlugin))
    {
        *ppvObject = static_cast<IHNSPlugin*>(this);
    }
    else
    {
        *ppvObject = NULL;
        return E_NOINTERFACE;
    }

    reinterpret_cast<IUnknown*>(*ppvObject)->AddRef();
    return S_OK;
}
////////////////////////////////////////////////////////////////////////////////
// PCPlugin::AddRef
////////////////////////////////////////////////////////////////////////////////
ULONG 
PCPlugin::AddRef()
{
    return InterlockedIncrement(&RefCount);
}
////////////////////////////////////////////////////////////////////////////////
// PCPlugin::Release
////////////////////////////////////////////////////////////////////////////////
ULONG 
PCPlugin::Release()
{
    ULONG refCount = InterlockedDecrement(&RefCount);
    if (refCount == 0)
        delete this;

    return refCount;
}
////////////////////////////////////////////////////////////////////////////////
// PCPlugin::Initialize
////////////////////////////////////////////////////////////////////////////////
HRESULT 
PCPlugin::Initialize(
    IHNSProvider * PluginProvider)
{
    UNREFERENCED_PARAMETER(PluginProvider);
    // Identify external network, and convert it to L2 Forwarding

    // Enumerate endpoints and build any needed plugin data structures.

    return S_OK;
}
////////////////////////////////////////////////////////////////////////////////
// PCPlugin::Teardown
////////////////////////////////////////////////////////////////////////////////
HRESULT 
PCPlugin::Teardown(
    IHNSProvider * PluginProvider)
{
    UNREFERENCED_PARAMETER(PluginProvider);
    
    return S_OK;
}
////////////////////////////////////////////////////////////////////////////////
// PCPlugin::HandleRequest
////////////////////////////////////////////////////////////////////////////////
HRESULT 
PCPlugin::HandleRequest(
    DWORD RequestType, 
    IHNSProvider * PluginProvider, 
    IHNSObject * PluginObject)
{
    HRESULT hr = S_OK;
    RequestCategory category;
    RequestActivity activity;
    HNSObjPtr requestObject;
    category = (RequestCategory)(RequestType >> 8);
    activity = (RequestActivity)(RequestType & 0xFF);

    try
    {
        switch (category)
        {
        case RequestCategoryNetwork:
            if (S_OK == PluginObject->PropertyExists(L"RequestObject"))
            {
                PluginObject->GetObject(L"RequestObject", &requestObject);
            }

            switch (activity) {
            case RequestActivityPreCreate:
                // Optionally veto creation
                // Optionally modify configuration
                m_host.HandleNetworkNotification(PluginProvider, requestObject, true);
                break;
            case RequestActivityPostCreate:
                break;
            case RequestActivityFailCreate:
               // m_host.HandleNetworkNotification(PluginProvider, requestObject, false);
                break;
            case RequestActivityPreDelete:
                // Optionally veto deletion
                break;
            case RequestActivityPostDelete:
               // m_host.HandleNetworkNotification(PluginProvider, requestObject, false);
                break;
            }
            break;
        case RequestCategoryEndpoint:
            if (S_OK == PluginObject->PropertyExists(L"RequestObject"))
            {
                PluginObject->GetObject(L"RequestObject", &requestObject);
            }

            switch (activity) 
            {
				case RequestActivityPreCreate:
					// Optionally veto creation
					// Optionally modify configuration e.g. assign an IP address to the endpoint
					m_host.HandleEndpointPreCreate(PluginProvider, requestObject);
					break;
				case RequestActivityPostCreate:
					m_host.HandleEndpointPostCreate(PluginProvider, requestObject);
					// Endpoint was created.
					break;
				case RequestActivityFailCreate:
					// Endpoint wasn't created correctly, deallocate the IP address if one was assigned.
					//m_host.HandleEndpointFailCreate(PluginProvider, requestObject);
					break;
				case RequestActivityPreDelete:
					// Optionally veto deletion
					break;
				case RequestActivityPostDelete:
					m_host.HandleEndpointPostDelete(PluginProvider, requestObject);
					break;
            }
            break;
        case RequestCategoryService:
            switch (activity)
            {
            case RequestActivityServiceStart:
                // Optionally create a network, if this is needed.
                // After this point it is allowed to create networks / endpoints.
               // m_host.Initialize(PluginProvider);
                break;
            case RequestActivityServiceStop:
                // After this point it is no longer allowed to create objects.
               // m_host.UnInitialize();
                break;
            case RequestActivityPluginRequest:
               // m_host.HandlePluginOperations(PluginObject);
                break;
            }
        }
    }
    catch (const std::exception&)
    {
        hr = PluginException::LogException();
    }

    return hr;
}
