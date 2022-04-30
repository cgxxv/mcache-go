package mcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFakeClock(t *testing.T) {
	var (
		fc = NewFakeClock()
	)

	now := fc.Now()
	fc.Advance(time.Second)

	assert.Equal(t, time.Duration(0), now.Add(time.Second).Sub(fc.Now()))
}
