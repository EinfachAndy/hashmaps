package clrobin

import (
	"sync"
	"sync/atomic"
)

const (
	emptyBucket = -1
)

type lockableBucket[K comparable, V comparable] struct {
	sync.RWMutex
	b bucket[K, V]
}

type bucket[K comparable, V comparable] struct {
	key K
	// psl is the probe sequence length (PSL), which is the distance value from
	// the optimum insertion. -1 or `emptyBucket` signals a free slot.
	// inspired from:
	//  - https://programming.guide/robin-hood-hashing.html
	//  - https://cs.uwaterloo.ca/research/tr/1986/CS-86-14.pdf
	psl   int8
	value V
}

//go:inline
func (b *bucket[K, V]) isEmpty() bool {
	return b.psl == emptyBucket
}

type storage[K comparable, V comparable] struct {
	length     atomic.Int64
	nextResize int
	capMinus1  int
	buckets    []lockableBucket[K, V]
}

//go:inline
func newStorage[K comparable, V comparable](capacity int, maxLoad float32) *storage[K, V] {
	buckets := make([]lockableBucket[K, V], capacity)

	for i := range buckets {
		buckets[i].b.psl = emptyBucket
	}

	return &storage[K, V]{
		capMinus1:  capacity - 1,
		buckets:    buckets,
		nextResize: int(float32(capacity) * maxLoad),
	}
}

// emplace applies the Robin Hood creed to all following buckets until a empty is found.
// Robin Hood creed: "takes from the rich and gives to the poor".
// rich means, low psl
// poor means, higher psl
//
// The result is a better distribution of the PSL values,
// where the expected length of the longest PSL is O(log(n)).
//
//go:inline
func (s *storage[K, V]) emplace(current *bucket[K, V], idx uintptr) uintptr {
	probing := 0
	for ; ; current.psl++ {
		if probing > 0 {
			s.buckets[idx].Lock()
		}
		probing++

		b := &s.buckets[idx].b

		if b.isEmpty() {
			// emplace the element, a valid bucket was found
			*b = *current
			return idx
		}

		if current.psl > b.psl {
			// swap values, apply the Robin Hood creed
			*current, *b = *b, *current
		}

		// next index
		idx = (idx + 1) & uintptr(s.capMinus1)
	}
}

// search holds locks on buckets.
//
//go:inline
func (s *storage[K, V]) search(start uintptr, key K) (*bucket[K, V], uintptr, int8) {
	psl := int8(0)
	for ; ; psl++ {
		s.buckets[start].Lock()

		if psl > s.buckets[start].b.psl {
			break
		}
		if s.buckets[start].b.key == key {
			return &s.buckets[start].b, start, psl
		}

		// next index
		start = (start + 1) & uintptr(s.capMinus1)
	}

	return nil, start, psl
}

func (s *storage[K, V]) unlock(start, end uintptr) {
	for {
		s.buckets[start].Unlock()
		if start == end {
			return
		}
		start = (start + 1) & uintptr(s.capMinus1)
	}
}

func (s *storage[K, V]) remove(idx uintptr) {
	current := &s.buckets[idx].b
	current.psl = emptyBucket

	idx = (idx + 1) & uintptr(s.capMinus1)
	next := &s.buckets[idx].b

	// now, back shift all buckets until we found a optimum or empty one
	for next.psl > 0 {
		next.psl--
		*current, *next = *next, *current // swap values
		current = next
		idx = (idx + 1) & uintptr(s.capMinus1)
		next = &s.buckets[idx].b
	}

	s.length.Add(-1)
}
