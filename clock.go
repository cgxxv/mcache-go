package mcache

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type Clock interface {
	Now() time.Time
}

type realClock struct {
	p unsafe.Pointer
}

func NewRealClock() Clock {
	p := time.Now()
	rc := &realClock{
		p: unsafe.Pointer(&p),
	}
	go func() {
		for {
			p := time.Now()
			atomic.StorePointer(&rc.p, unsafe.Pointer(&p))
			time.Sleep(100 * time.Millisecond)
		}
	}()
	return rc
}

func (rc *realClock) Now() time.Time {
	return rc.now()
}

func (rc *realClock) now() time.Time {
	return *(*time.Time)(atomic.LoadPointer(&rc.p))
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
