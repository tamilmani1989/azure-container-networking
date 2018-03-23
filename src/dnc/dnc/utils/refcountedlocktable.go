package utils

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
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
	glog.Infof("Acquiring lock:%s, Read:%d", lockName, read)
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
	fmt.Println("Acquired lock", lockName)
}

func (refCountedLockTable *RefCountedLockTable) unlock(lockName string, read bool) {
	glog.Infof("Releasing lock:%s, Read:%d", lockName, read)
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
		fmt.Println("Decremented Count:", lock.refCount)
		if lock.refCount == 0 {
			fmt.Println("Deleting Lock:", lockName)
			delete(refCountedLockTable.table, lockName)
		} else {
			fmt.Println("Unexpected Unlock received.")
		}
	}
}
