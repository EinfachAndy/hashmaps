package clrobin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLockUnlock(t *testing.T) {
	s := newStorage[int, int](4, 0.5)

	start := uintptr(1)
	b, idx, psl := s.search(start, 0)
	assert.Nil(t, b)
	assert.Equal(t, start, idx)
	assert.Zero(t, psl)

	s.unlock(start, idx)
}
