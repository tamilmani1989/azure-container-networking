// +build enablePubsubProxy

package pubsubproxy

import (
	"dnc/logger/loggershim"
	"sync"
	"time"
)

// jobTsLock holds a mutex with a map of locks and latest job timestamp.
// Mutex is used to get exclusive lock on the map while initializing the lock in the map
type jobTsLock struct {
	sync.Mutex
	tsLockMap map[string]*tsLock
}

// tsLock holds the lock and latest job timestamp
type tsLock struct {
	mutex       sync.Mutex
	timestamp   time.Time
	isCompleted bool
}

// NewJobTsLock initializes the named lock struct
func NewJobTsLock() *jobTsLock {
	return &jobTsLock{
		tsLockMap: make(map[string]*tsLock),
	}
}

// LockAcquire acquires the lock for the given job ID
func (lock *jobTsLock) LockAcquire(
	jobId string) {
	lock.Lock()
	_, ok := lock.tsLockMap[jobId]
	if !ok {
		lock.tsLockMap[jobId] = &tsLock{}
	}

	lock.Unlock()
	lock.tsLockMap[jobId].Lock()
}

// LockRelease releases the lock with specified name
func (lock *jobTsLock) LockRelease(
	jobId string) {
	lock.Lock()
	defer lock.Unlock()

	_, ok := lock.tsLockMap[jobId]
	if ok {
		lock.tsLockMap[jobId].Unlock()
	} else {
		loggershim.LS.Debugf("[PubSubProxy] Attempt to unlock: %s without acquiring the lock", jobId)
	}
}

// ReadTimestamp retrives the timestamp value
func (lock *jobTsLock) ReadStatusTimestamp(
	jobId string) (time.Time, bool) {
	var (
		timestamp   time.Time
		isCompleted bool
	)

	if _, ok := lock.tsLockMap[jobId]; ok {
		timestamp = lock.tsLockMap[jobId].timestamp
		isCompleted = lock.tsLockMap[jobId].isCompleted
	}

	return timestamp, isCompleted
}

// UpdateTimestamp updates the timestamp for the jobId
func (lock *jobTsLock) UpdateStatusTimestamp(
	jobId string,
	timestamp time.Time,
	isCompleted bool) {
	lock.Lock()
	lock.tsLockMap[jobId].timestamp = timestamp
	lock.tsLockMap[jobId].isCompleted = isCompleted
	lock.Unlock()
}

// Lock locks tsLock
func (tsLock *tsLock) Lock() {
	tsLock.mutex.Lock()
}

// Unlock unlocks tsLock
func (tsLock *tsLock) Unlock() {
	tsLock.mutex.Unlock()
}
