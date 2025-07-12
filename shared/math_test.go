package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/EinfachAndy/hashmaps/shared"
)

func TestNextPowerOfTwo(t *testing.T) {
	assert.Equal(t, uint64(0), shared.NextPowerOf2(0))
	assert.Equal(t, uint64(1), shared.NextPowerOf2(1))
	assert.Equal(t, uint64(2), shared.NextPowerOf2(2))
	assert.Equal(t, uint64(4), shared.NextPowerOf2(3))
	assert.Equal(t, uint64(4), shared.NextPowerOf2(4))
	assert.Equal(t, uint64(8), shared.NextPowerOf2(5))
	assert.Equal(t, uint64(8), shared.NextPowerOf2(7))
	assert.Equal(t, uint64(8), shared.NextPowerOf2(8))
	assert.Equal(t, uint64(16), shared.NextPowerOf2(9))
	assert.Equal(t, uint64(16), shared.NextPowerOf2(10))
	assert.Equal(t, uint64(16), shared.NextPowerOf2(15))
	assert.Equal(t, uint64(16), shared.NextPowerOf2(16))
	assert.Equal(t, uint64(1024), shared.NextPowerOf2(1000))
	assert.Equal(t, uint64(2048), shared.NextPowerOf2(2000))
}
