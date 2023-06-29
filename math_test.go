package hashmaps_test

import (
	"testing"

	"github.com/EinfachAndy/hashmaps"
	"github.com/stretchr/testify/assert"
)

func TestNextPowerOfTwo(t *testing.T) {
	assert.Equal(t, uint64(0), hashmaps.NextPowerOf2(0))
	assert.Equal(t, uint64(1), hashmaps.NextPowerOf2(1))
	assert.Equal(t, uint64(2), hashmaps.NextPowerOf2(2))
	assert.Equal(t, uint64(4), hashmaps.NextPowerOf2(3))
	assert.Equal(t, uint64(4), hashmaps.NextPowerOf2(4))
	assert.Equal(t, uint64(8), hashmaps.NextPowerOf2(5))
	assert.Equal(t, uint64(8), hashmaps.NextPowerOf2(7))
	assert.Equal(t, uint64(8), hashmaps.NextPowerOf2(8))
	assert.Equal(t, uint64(16), hashmaps.NextPowerOf2(9))
	assert.Equal(t, uint64(16), hashmaps.NextPowerOf2(10))
	assert.Equal(t, uint64(16), hashmaps.NextPowerOf2(15))
	assert.Equal(t, uint64(16), hashmaps.NextPowerOf2(16))
	assert.Equal(t, uint64(1024), hashmaps.NextPowerOf2(1000))
	assert.Equal(t, uint64(2048), hashmaps.NextPowerOf2(2000))
}
