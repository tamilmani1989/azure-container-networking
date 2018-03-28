package utils

import (
	"sync"

	"dnc/logger/loggershim"
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
	loggershim.LS.Log("Acquiring lock:%s, Read:%d\n", lockName, read)
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
	loggershim.LS.Log("Acquired lock %s\n", lockName)
}

func (refCountedLockTable *RefCountedLockTable) unlock(lockName string, read bool) {
	loggershim.LS.Log("Releasing lock:%s, Read:%d\n", lockName, read)
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
		loggershim.LS.Log("Decremented Count: %d\n", lock.refCount)
		if lock.refCount == 0 {
			loggershim.LS.Log("Deleting Lock: %s\n", lockName)
			delete(refCountedLockTable.table, lockName)
		} else {
			loggershim.LS.Log("Unexpected Unlock received.\n")
		}
	}
}
