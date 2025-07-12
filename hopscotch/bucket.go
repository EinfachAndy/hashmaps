package hopscotch

type bucket[K comparable, V any] struct {
	hopInfo uint64 // stores the neighborhood and the state of the reserved bits
	key     K
	val     V
}

const (
	reservedBits        = uintptr(1)        // number of reserved bits within the hop info
	occupyBit           = uint64(1)         // bit mask for the occupy bit
	maxNeighborhoodSize = 64 - reservedBits // max size of H (neighborhood)
)

// flip returns the opposite bit mask
//
//go:inline
func flip(a uint64) uint64 {
	a ^= 0xFFFFFFFFFFFFFFFF
	return a
}

// set the state of v at the i-th position within the neighborhood
//
//go:inline
func (b *bucket[K, V]) set(i uintptr, v bool) {
	mask := uint64(1) << (i + reservedBits)
	if v {
		b.hopInfo |= mask
	} else {
		b.hopInfo &= flip(mask)
	}
}

// getNeighborhood returns the neighborhood bit mask
//
//go:inline
func (b *bucket[K, V]) getNeighborhood() uint64 {
	return b.hopInfo >> uint64(reservedBits)
}

// returns true if the bucket is empty
//
//go:inline
func (b *bucket[K, V]) isEmpty() bool {
	return (b.hopInfo & occupyBit) == 0
}

// release marks the bucket as empty
//
//go:inline
func (b *bucket[K, V]) release() {
	b.hopInfo &= flip(occupyBit)
}

// occupy marks the bucket as not empty
//
//go:inline
func (b *bucket[K, V]) occupy() {
	b.hopInfo |= occupyBit
}
