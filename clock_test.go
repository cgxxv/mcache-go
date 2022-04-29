package mcache

import (
	"reflect"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestRealClock(t *testing.T) {
	var (
		p  = time.Now()
		rc = &realClock{
			p: unsafe.Pointer(&p),
		}
		ms = 100 * time.Millisecond
		tc = make(chan time.Time)
	)
	go func() {
		for i := 0; i < 10; i++ {
			p := time.Now()
			atomic.StorePointer(&rc.p, unsafe.Pointer(&p))
			tc <- p
			time.Sleep(ms)
		}
		close(tc)
	}()

	for {
		want, ok := <-tc
		if !ok {
			break
		}
		got := rc.now()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("RealClock.Now() = %v, want %v", got, want)
		}
	}
}

func TestFakeClock(t *testing.T) {
	var (
		fc = NewFakeClock()
	)

	now := fc.Now()
	fc.Advance(time.Second)

	assert.Equal(t, time.Duration(0), now.Add(time.Second).Sub(fc.Now()))
}
