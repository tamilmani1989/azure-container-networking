// ***********************************************************************
// <copyright file="ovPluginCriticalSection.h" company="Microsoft">
//     Copyright (c) Microsoft Corporation. All rights reserved.
// </copyright>
// <summary></summary>
// ***********************************************************************
#pragma once

#ifndef __CRITICALSECTION_H__
#define __CRITICALSECTION_H__

#include <Windows.h>
#include "ovPluginException.h"

namespace Microsoft { namespace Azure { namespace Networking { namespace OverlayPlugin {

    /// <summary>Provides critical section.</summary>
    class CriticalSection
    {
    public:
        /// <summary>Constructor. Initializes a critical section object.</summary>
        CriticalSection() throw()
        {
            ::InitializeCriticalSection(&m_cs);
        }
        // disable default copy constructor
        CriticalSection(const CriticalSection&) = delete;
        /// <summary>Constructor. Initializes a critical section object
        /// and sets the spin count.</summary>
        /// <remarks>This function may throw exception only on
        /// Windows Server 2003 and Windows XP/2000</remarks>
        CriticalSection(__in DWORD spinCount)
        {
            if (::InitializeCriticalSectionAndSpinCount(&m_cs, spinCount) == 0)
                PluginException::ThrowLastError(L"Unable to initialize critical section");
        }
        /// <summary>Destructor. Destroys critical section.</summary>
        ~CriticalSection() throw()
        {
            ::DeleteCriticalSection(&m_cs);
        }
        // disable default assignment operator
        CriticalSection& operator=(const CriticalSection&) = delete;
        /// <summary>Enters critical section.</summary>
        _Acquires_lock_(this->m_cs)
            void Enter() throw()
        {
            ::EnterCriticalSection(&m_cs);
        }
        /// <summary>Tries to enter critical section.</summary>
        /// <retrun>true if critical section is entered, false otherwise.</retrun>
#pragma prefast (suppress: 26135, "It is annotated with _Acquires_lock_")
        _When_(return != 0, _Acquires_lock_(this->m_cs)) bool TryEnter() throw()
        {
            return (::TryEnterCriticalSection(&m_cs) != 0);
        }
        /// <summary>Leaves critical section.</summary>
        _Releases_lock_(this->m_cs)
            void Leave() throw()
        {
            ::LeaveCriticalSection(&m_cs);
        }
    private:
        /// <summary>Critical section.</summary>
        CRITICAL_SECTION m_cs;
    };

    /// <summary>Scope locker for critical section</summary>
    class CSLocker
    {
    public:
        // disable default copy constructor
        CSLocker(const CSLocker&) = delete;
        /// <summary>Constructor. Enters critical section.</summary>
        _Acquires_lock_(this->m_cs)
            CSLocker(__in CriticalSection& cs) throw() :
            m_cs(cs)
        {
            m_cs.Enter();
        }
        /// <summary>Destructor. Leaves critical section.</summary>
        _Releases_lock_(this->m_cs)
            ~CSLocker() throw()
        {
            m_cs.Leave();
        }
        // disable default assignment operator
        CSLocker& operator=(const CSLocker&) = delete;
    private:
        /// <summary>Reference to critical section</summary>
        CriticalSection& m_cs;
    };

    /// <summary>Slim Reader/Writer (SRW) Lock</summary>
    class SRWLock
    {
    public:
        /// <summary>Constructor. Initializes the lock.</summary>
        SRWLock() throw()
        {
            ::InitializeSRWLock(&m_lock);
        }
        /// <summary>Destructor.</summary>
        ~SRWLock() throw()
        {
        }
        /// <summary>Acquires exclusive lock</summary>
        _Acquires_exclusive_lock_(this->m_lock)
            void AcquireExclusive() throw()
        {
            ::AcquireSRWLockExclusive(&m_lock);
        }
        /// <summary>Acquires shared lock</summary>
        _Acquires_shared_lock_(this->m_lock)
            void AcquireShared() throw()
        {
            ::AcquireSRWLockShared(&m_lock);
        }
        /// <summary>Releases exclusive lock acquired by AcquireExclusive().</summary>
        _Releases_exclusive_lock_(this->m_lock)
            void ReleaseExclusive() throw()
        {
            ::ReleaseSRWLockExclusive(&m_lock);
        }
        /// <summary>Releases shared lock acquired by AcquireShared().</summary>
        _Releases_shared_lock_(this->m_lock)
            void ReleaseShared() throw()
        {
            ::ReleaseSRWLockShared(&m_lock);
        }
        // disable default copy constructor
        SRWLock(const SRWLock&) = delete;
        // disable default assignment operator
        SRWLock& operator=(const SRWLock&) = delete;
    private:
        /// <summary>SRW lock</summary>
        SRWLOCK m_lock;
    };

    /// <summary>Scope locker for SRW lock. Acquires/releases shared lock.</summary>
    class SRWReadLocker
    {
    public:
        /// <summary>Constructor. Acquires shared lock.</summary>
        _Requires_lock_not_held_(lock.m_lock)
            SRWReadLocker(__in SRWLock& lock) throw() :
            m_lock(lock)
        {
            m_lock.AcquireShared();
        }
        /// <summary>Destructor. Releases shared lock.</summary>
        ~SRWReadLocker() throw()
        {
            m_lock.ReleaseShared();
        }
        // disable default copy constructor
        SRWReadLocker(const SRWReadLocker&) = delete;
        // disable default assignment operator
        SRWReadLocker& operator=(const SRWReadLocker&) = delete;
    private:
        // disable default constructor
        SRWReadLocker();
        /// <summary>Reference to critical section</summary>
        SRWLock& m_lock;
    };

    /// <summary>Scope locker for SRW lock. Acquires/releases exclusive lock.</summary>
    class SRWWriteLocker
    {
    public:
        /// <summary>Constructor. Acquires exclusive lock.</summary>
        _Requires_lock_not_held_(lock.m_lock)
            SRWWriteLocker(__in SRWLock& lock) throw() :
            m_lock(lock)
        {
            m_lock.AcquireExclusive();
        }
        /// <summary>Destructor. Releases exclusive lock.</summary>
        ~SRWWriteLocker()
        {
            m_lock.ReleaseExclusive();
        }
        // disable default copy constructor
        SRWWriteLocker(const SRWWriteLocker&) = delete;
        // disable default assignment operator
        SRWWriteLocker& operator=(const SRWWriteLocker&) = delete;
    private:
        // disable default constructor
        SRWWriteLocker();
        /// <summary>Reference to critical section</summary>
        SRWLock& m_lock;
    };

}}}}

#endif /* __CRITICALSECTION_H__ */