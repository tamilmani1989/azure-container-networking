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
	// _, fileFullPath, line, _ := runtime.Caller(3)
	// fileName := ""
	// if len(fileFullPath) > 0 {
	// 	fileName = filepath.Base(fileFullPath)
	// }

	refCountedLockTable.mutex.Lock()
	// loggershim.LS.Debugf("Acquiring lock:%s, Read:%d", lockName, read)
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

	// loggershim.LS.Debugf("Acquired lock:%s, Read:%v. File:%s, Line:%d", lockName, read, fileName, line)
}

func (refCountedLockTable *RefCountedLockTable) unlock(lockName string, read bool) {
	// _, fileFullPath, line, _ := runtime.Caller(3)
	// fileName := ""
	// if len(fileFullPath) > 0 {
	// 	fileName = filepath.Base(fileFullPath)
	// }

	// loggershim.LS.Debugf("Releasing lock:%s, Read:%v", lockName, read)
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
		// loggershim.LS.Debugf("Decremented Count:%d", lock.refCount)
		if lock.refCount == 0 {
			// loggershim.LS.Debugf("Deleting Lock:%s", lockName)
			delete(refCountedLockTable.table, lockName)
		} else {
			// loggershim.LS.Debugf("Refernece count on lock %s after release %d", lockName, lock.refCount)
		}
	}
}
