// ***********************************************************************
// <copyright file="PluginMain.h" company="Microsoft">
//     Copyright (c) Microsoft Corporation. All rights reserved.
// </copyright>
// <summary></summary>
// ***********************************************************************
#pragma once
#include "ContainerHost.h"
#include <Windows.h>

using namespace Microsoft::Azure::Networking::OverlayPlugin;

// {D5AAF7C4-1B3E-4b49-9CF9-9263FAB9DC6D}
static const GUID ClsidPlugin =
{ 0xe5aaf7c4, 0x1b3e, 0x4b49, 0x9c, 0xf9, 0x92, 0x63, 0xfa, 0xb9, 0xdc, 0x6d };

HRESULT __stdcall DllGetClassObject(
    REFCLSID rclsid,
    REFIID riid,
    LPVOID *ppv
    );

HRESULT __stdcall DllCanUnloadNow(void);
/*Implements the private cloud plugin for HNS*/
class PCClassFactory : IClassFactory
{
public:
    virtual HRESULT STDMETHODCALLTYPE QueryInterface(
        __RPC__in REFIID riid,
        _COM_Outptr_  void **ppvObject);

    virtual ULONG STDMETHODCALLTYPE AddRef();

    virtual ULONG STDMETHODCALLTYPE Release();

    virtual HRESULT CreateInstance(
        /*[in]*/  IUnknown *pUnkOuter,
        /*[in]*/  REFIID   riid,
        /*[out]*/ void     **ppvObject
        );

    virtual HRESULT LockServer(
        /*[in]*/ BOOL fLock
        );
};
/*Implements the private cloud plugin for HNS*/
class PCPlugin : IHNSPlugin
{
public:
    PCPlugin();
    ~PCPlugin();

    ULONG RefCount;

    virtual HRESULT STDMETHODCALLTYPE QueryInterface(
        __RPC__in REFIID riid,
        _COM_Outptr_  void **ppvObject);

    virtual ULONG STDMETHODCALLTYPE AddRef();

    virtual ULONG STDMETHODCALLTYPE Release();

    virtual HRESULT Initialize(
        IHNSProvider* PluginProvider
        );

    virtual HRESULT Teardown(
        IHNSProvider* PluginProvider
        );
    
    virtual HRESULT HandleRequest(
        DWORD RequestType,
        IHNSProvider* PluginProvider,
        IHNSObject* PluginObject
        );
private:
    ContainerHost m_host;
};
