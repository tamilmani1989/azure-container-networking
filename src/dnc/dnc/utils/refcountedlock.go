package utils

import (
	"sync"
)

type RefCountedLock struct {
	mutex    sync.RWMutex
	refCount int
}

// RLock -Acquire read lock
func (refCountedLock *RefCountedLock) RLock() {
	refCountedLock.mutex.RLock()
}

// RUnlock -Release read lock
func (refCountedLock *RefCountedLock) RUnlock() {
	refCountedLock.mutex.RUnlock()
}

// Lock -Acquire write lock
func (refCountedLock *RefCountedLock) Lock() {
	refCountedLock.mutex.Lock()
}

// Unlock -Release write lock
func (refCountedLock *RefCountedLock) Unlock() {
	refCountedLock.mutex.Unlock()
}

// AddRef - Add refcount on the lock
func (refCountedLock *RefCountedLock) AddRef() {
	refCountedLock.refCount++
}

// Release - Rmeove rference on the lock
func (refCountedLock *RefCountedLock) Release() {
	refCountedLock.refCount--
}
