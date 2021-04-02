package pubsubproxy

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobTsLock(t *testing.T) {
	expectedTS, _ := time.Parse(time.Kitchen, "5:00PM")

	jtsl := NewJobTsLock()
	var wg sync.WaitGroup
	wg.Add(900)
	for i := 0; i < 900; i++ {
		go func(n int) {
			tsl := jtsl.GetTsLock(fmt.Sprintf("job%d", n%3))

			tsl.Lock()
			tsl.UpdateStatusTimestamp(expectedTS, true)
			tsl.Unlock()

			wg.Done()
		}(i)
	}

	wg.Wait()

	require.Len(t, jtsl.tsLockMap, 3)
	for i := 0; i < 3; i++ {
		job := fmt.Sprintf("job%d", i)

		tsl, ok := jtsl.tsLockMap[job]
		require.True(t, ok)

		statusTS, isCompleted := tsl.ReadStatusTimestamp()
		assert.True(t, isCompleted)
		assert.Equal(t, expectedTS, statusTS)
	}
}
