package mcache

import (
	"reflect"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

func TestRealClock_Now(t *testing.T) {
	var (
		p  = time.Now()
		rc = &RealClock{
			p: unsafe.Pointer(&p),
		}
		ms = 100 * time.Millisecond
		tc = make(chan time.Time)
	)
	go func() {
		for i := 0; i < 100; i++ {
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
