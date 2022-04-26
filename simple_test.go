package mcache

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func randString(l int) string {
	buf := make([]byte, l)
	for i := 0; i < (l+1)/2; i++ {
		buf[i] = byte(rand.Intn(256))
	}
	return fmt.Sprintf("%x", buf)[:l]
}

var (
	ctx = context.TODO()
)

func TestSimpleSetWithExpire(t *testing.T) {
	var (
		mc  = &simpleCache{}
		rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	)

	key := randString(40)
	val := rnd.Intn(math.MaxInt64)

	mc.set(ctx, key, val, nil)
	value, err := mc.get(ctx, key)
	assert.Nil(t, err)
	assert.Equal(t, val, value)

	mc.has(ctx, key)
}
