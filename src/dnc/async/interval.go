package async

import (
	"context"
	"time"
)

// DoCtx executes a function asynchronously on a given interval, until context is cancelled.
func IntervalDoCtx(ctx context.Context, interval time.Duration, f func()) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				f()
			}
		}
	}()
}

type CancelFunc func()

// Do executes a function asynchronously on a given interval, until caller executes the returned CancelFunc.
func IntervalDo(interval time.Duration, f func()) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	IntervalDoCtx(ctx, interval, f)
	return cancel
}
