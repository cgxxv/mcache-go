package mcache

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func NewRealClock() Clock {
	return realClock{}
}

func (rc realClock) Now() time.Time {
	return time.Now()
}

type FakeClock interface {
	Clock
	Advance(d time.Duration)
}

func NewFakeClock() FakeClock {
	return &fakeclock{
		now: time.Now(),
	}
}

type fakeclock struct {
	now time.Time
	sync.RWMutex
}

func (fc *fakeclock) Now() time.Time {
	fc.RLock()
	defer fc.RUnlock()
	return fc.now
}

func (fc *fakeclock) Advance(d time.Duration) {
	fc.Lock()
	defer fc.Unlock()
	fc.now = fc.now.Add(d)
}
