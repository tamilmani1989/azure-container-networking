package benchmark

import (
	"fmt"
	"sync"
)

// MultiError contains multiple errors. It is thread safe to collect errors, but other methods should only be used
// after all writes are done.
type MultiError struct {
	errs []error
	m    sync.Mutex
}

func (e *MultiError) Error() string {
	return fmt.Sprintf("%d errors", len(e.errs))
}

func (e *MultiError) GetErrors() []error {
	return e.errs
}

func (e *MultiError) Collect(err error) {
	e.m.Lock()
	e.errs = append(e.errs, err)
	e.m.Unlock()
}

func (e *MultiError) Any() error {
	if len(e.errs) > 0 {
		return e
	}
	return nil
}
