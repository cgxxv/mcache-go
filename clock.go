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

type RealClock struct {
	p unsafe.Pointer
}

func NewRealClock() Clock {
	if runUnitTest {
		return &RealClock{}
	}

	p := time.Now()
	rc := &RealClock{
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

func (rc *RealClock) Now() time.Time {
	if runUnitTest {
		return time.Now()
	}
	return rc.now()
}

func (rc *RealClock) now() time.Time {
	return *(*time.Time)(atomic.LoadPointer(&rc.p))
}

type FakeClock interface {
	Clock

	Advance(d time.Duration)
}

func NewFakeClock() FakeClock {
	return &fakeclock{
		// Taken from github.com/jonboulle/clockwork: use a fixture that does not fulfill Time.IsZero()
		now: time.Date(1984, time.April, 4, 0, 0, 0, 0, time.UTC),
	}
}

type fakeclock struct {
	now time.Time

	mutex sync.RWMutex
}

func (fc *fakeclock) Now() time.Time {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	t := fc.now
	return t
}

func (fc *fakeclock) Advance(d time.Duration) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.now = fc.now.Add(d)
}
