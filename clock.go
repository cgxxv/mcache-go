package mcache

import (
	"sync/atomic"
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
	fc := &fakeclock{}
	fc.now.Store(time.Now())
	return fc
}

type fakeclock struct {
	now atomic.Value
}

func (fc *fakeclock) Now() time.Time {
	return fc.now.Load().(time.Time)
}

func (fc *fakeclock) Advance(d time.Duration) {
	for {
		current := fc.Now()
		newTime := current.Add(d)
		if fc.now.CompareAndSwap(current, newTime) {
			return
		}
	}
}
