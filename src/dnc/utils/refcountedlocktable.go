package utils

import (
	"sync"
)

type RefCountedLockTable struct {
	mutex sync.RWMutex
	table map[string]*RefCountedLock
}

// Initialize -Initialize RefCounted lock table.
func (refCountedLockTable *RefCountedLockTable) Initialize() {
	refCountedLockTable.table = make(map[string]*RefCountedLock)
}

// RLock -Acquire read lock associated with named lock.
func (refCountedLockTable *RefCountedLockTable) RLock(lockName string) {
	refCountedLockTable.lock(lockName, true)
}

// Lock -Acquire write lock associated with named lock.
func (refCountedLockTable *RefCountedLockTable) Lock(lockName string) {
	refCountedLockTable.lock(lockName, false)
}

// RUnlock -Release read lock associated with named lock.
func (refCountedLockTable *RefCountedLockTable) RUnlock(lockName string) {
	refCountedLockTable.unlock(lockName, true)
}

// Unlock -Release write lock associated with named lock.
func (refCountedLockTable *RefCountedLockTable) Unlock(lockName string) {
	refCountedLockTable.unlock(lockName, false)
}

func (refCountedLockTable *RefCountedLockTable) lock(lockName string, read bool) {
	refCountedLockTable.mutex.Lock()
	lock, ok := refCountedLockTable.table[lockName]
	if !ok {
		lock = &RefCountedLock{refCount: 0}
		refCountedLockTable.table[lockName] = lock
	}

	lock.AddRef()

	refCountedLockTable.mutex.Unlock()

	if read {
		lock.RLock()
	} else {
		lock.Lock()
	}
}

func (refCountedLockTable *RefCountedLockTable) unlock(lockName string, read bool) {
	refCountedLockTable.mutex.Lock()
	defer refCountedLockTable.mutex.Unlock()
	lock, ok := refCountedLockTable.table[lockName]
	if ok {
		if read {
			lock.RUnlock()
		} else {
			lock.Unlock()
		}
		lock.Release()
		if lock.refCount == 0 {
			delete(refCountedLockTable.table, lockName)
		}
	}
}
