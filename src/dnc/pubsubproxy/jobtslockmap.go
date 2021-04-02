package pubsubproxy

import (
	"sync"
	"time"
)

// jobTsLock holds a mutex with a map of locks and latest job timestamp.
type jobTsLock struct {
	m         sync.Mutex
	tsLockMap map[string]*tsLock
}

// NewJobTsLock initializes the named lock struct
func NewJobTsLock() *jobTsLock {
	return &jobTsLock{
		tsLockMap: make(map[string]*tsLock),
	}
}

// GetTsLock returns the tsLock for the given job ID
func (lock *jobTsLock) GetTsLock(jobId string) *tsLock {
	lock.m.Lock()
	tsl, ok := lock.tsLockMap[jobId]
	if !ok {
		tsl = new(tsLock)
		lock.tsLockMap[jobId] = tsl
	}
	lock.m.Unlock()

	return tsl
}

// tsLock holds a lock specific to a job and latest job timestamp
// It is the caller's responsibility to lock and unlock before read/update operations
type tsLock struct {
	sync.Mutex
	timestamp   time.Time
	isCompleted bool
}

// ReadTimestamp retrieves the timestamp value
func (tsLock *tsLock) ReadStatusTimestamp() (time.Time, bool) {
	return tsLock.timestamp, tsLock.isCompleted
}

// UpdateTimestamp updates the timestamp for the jobId
func (tsLock *tsLock) UpdateStatusTimestamp(timestamp time.Time, isCompleted bool) {
	tsLock.timestamp = timestamp
	tsLock.isCompleted = isCompleted
}
