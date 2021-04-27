package benchmark

import (
	"math"
	"sort"
	"sync"
	"time"
)

type durations struct {
	dd []time.Duration
	m  sync.Mutex
}

func (dd *durations) Add(d time.Duration) {
	dd.m.Lock()
	dd.dd = append(dd.dd, d)
	dd.m.Unlock()
}

func (dd *durations) GetAll() []time.Duration {
	dd.m.Lock()
	c := make([]time.Duration, len(dd.dd))
	copy(c, dd.dd)
	dd.m.Unlock()
	return c
}

func (dd *durations) Len() int {
	dd.m.Lock()
	defer dd.m.Unlock()
	return len(dd.dd)
}

func Percentile(samples []time.Duration, th float64) time.Duration {
	sort.Slice(samples, func(i, j int) bool {
		return samples[i] < samples[j]
	})

	idx := int(math.Floor(float64(len(samples)) * th))
	return samples[idx]
}

func PercentileAvg(samples []time.Duration, th float64) time.Duration {
	sort.Slice(samples, func(i, j int) bool {
		return samples[i] < samples[j]
	})

	idx := int(math.Floor(float64(len(samples)) * th))
	return Average(samples[:idx+1])
}

func Average(samples []time.Duration) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	var sum time.Duration
	for _, sample := range samples {
		sum += sample
	}
	a := float64(sum) / float64(len(samples))
	return time.Duration(math.Round(a))
}
